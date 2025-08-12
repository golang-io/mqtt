package packet

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// SUBSCRIBE 订阅请求报文
//
// MQTT v3.1.1: 参考章节 3.8 SUBSCRIBE - Subscribe to topics
// MQTT v5.0: 参考章节 3.8 SUBSCRIBE - Subscribe to topics
//
// 报文结构:
// 固定报头: 报文类型0x08，标志位必须为DUP=0, QoS=1, RETAIN=0
// 可变报头: 报文标识符、订阅属性(v5.0)
// 载荷: 订阅列表，每个订阅包含主题过滤器和订阅选项
//
// 版本差异:
// - v3.1.1: 基本的订阅功能，支持主题过滤器和QoS设置
// - v5.0: 在v3.1.1基础上增加了属性系统，支持订阅标识符、用户属性等
//
// 标志位规则:
// - DUP: 必须为0 [MQTT-3.8.1-1]
// - QoS: 必须为1 [MQTT-3.8.1-1]
// - RETAIN: 必须为0 [MQTT-3.8.1-1]
type SUBSCRIBE struct {
	*FixedHeader `json:"FixedHeader,omitempty"`

	// PacketID 报文标识符
	// 参考章节: 2.3.1 Packet Identifier
	// 位置: 可变报头第1个字段
	// 要求: 必须包含，范围1-65535
	// 用途: 用于标识订阅请求，确保订阅确认的可靠性
	PacketID uint16 `json:"PacketID,omitempty"`

	// Props 订阅属性 (v5.0新增)
	// 参考章节: 3.8.2.2 SUBSCRIBE Properties
	// 位置: 可变报头，在报文标识符之后
	// 包含订阅标识符、用户属性等
	Props *SubscribeProperties

	// Subscriptions 订阅列表
	// 参考章节: 3.8.3 SUBSCRIBE Payload
	// 位置: 载荷部分
	// 要求: 至少包含一个订阅 [MQTT-3.8.3-1]
	// 每个订阅包含主题过滤器和订阅选项
	Subscriptions []Subscription `json:"Subscription,omitempty"`
}

func (pkt *SUBSCRIBE) Kind() byte {
	return 0x8
}

func (pkt *SUBSCRIBE) Pack(w io.Writer) error {
	buf := GetBuffer()
	defer PutBuffer(buf)
	buf.Write(i2b(pkt.PacketID))

	if pkt.Version == VERSION500 {
		if pkt.Props == nil {
			pkt.Props = &SubscribeProperties{}
		}
		b, err := pkt.Props.Pack()
		if err != nil {
			return err
		}
		propsLen, err := encodeLength(len(b))
		if err != nil {
			return err
		}
		buf.Write(propsLen)
		buf.Write(b)
	}

	for _, subscription := range pkt.Subscriptions {
		if subscription.TopicFilter == "" {
			return ErrProtocolViolationNoTopic
		}
		buf.Write(s2b(subscription.TopicFilter))
		buf.WriteByte(subscription.MaximumQoS)
	}
	pkt.FixedHeader.RemainingLength = uint32(buf.Len())
	if err := pkt.FixedHeader.Pack(w); err != nil {
		return err
	}
	_, err := buf.WriteTo(w)
	return err
}
func (pkt *SUBSCRIBE) Unpack(buf *bytes.Buffer) error {
	// SUBSCRIBE 控制报固定报头的第 3,2,1,0 位是保留位，必须分别设置为 0,0,1,0。
	// 服务端必须将其它的任何值都当做是不合法的并关闭网络连接 [MQTT-3.8.1-1]。
	if pkt.Dup != 0x0 || pkt.QoS != 0x1 || pkt.Retain != 0x0 {
		return ErrMalformedFlags
	}
	pkt.PacketID = binary.BigEndian.Uint16(buf.Next(2))
	if pkt.Version == VERSION500 {
		if err := pkt.Props.Unpack(buf); err != nil {
			return fmt.Errorf("pkt.RemainingLength=%v err=%w", pkt.RemainingLength, err)
		}
	}
	for buf.Len() != 0 {
		subscription := Subscription{}
		subscription.TopicFilter, _ = decodeUTF8[string](buf)
		options := buf.Next(1)[0]
		subscription.MaximumQoS = options & 0b00000011
		if subscription.MaximumQoS > 0x02 {
			return ErrProtocolViolationQosOutOfRange
		}
		subscription.NoLocal = options & 0b00000100 >> 2
		subscription.RetainAsPublished = options & 0b00001000 >> 3
		subscription.RetainHandling = options & 0b00110000 >> 4
		if options&0b11000000>>6 != 0 {
			return ErrMalformedFlags
		}
		pkt.Subscriptions = append(pkt.Subscriptions, subscription)
	}
	if len(pkt.Subscriptions) == 0 {
		return ErrProtocolViolationNoTopic
	}
	return nil
}

// Subscription 订阅项
// 参考章节: 3.8.3 SUBSCRIBE Payload
// 包含主题过滤器和订阅选项
//
// 版本差异:
// - v3.1.1: 基本的主题过滤器和QoS设置
// - v5.0: 增加了NoLocal、RetainAsPublished、RetainHandling等选项
type Subscription struct {
	// TopicFilter 主题过滤器
	// 参考章节: 3.8.3.1 Topic Filter
	// 位置: 载荷中，每个订阅的第1个字段
	// 要求: UTF-8编码字符串，支持通配符
	// 通配符:
	// - +: 单层通配符，匹配任意一个层级
	// - #: 多层通配符，匹配任意数量的层级
	// 注意: 多层通配符必须是主题过滤器的最后一个字符
	TopicFilter string

	// MaximumQoS 最大QoS等级
	// 参考章节: 3.8.3.2 Subscription Options
	// 位置: 订阅选项字节的bits 1-0
	// 值:
	// - 0x00: QoS 0 - 最多一次传递
	// - 0x01: QoS 1 - 至少一次传递
	// - 0x02: QoS 2 - 恰好一次传递
	// 注意: 0x03是保留值，不允许使用
	MaximumQoS uint8

	// NoLocal 本地标志 (v5.0新增)
	// 参考章节: 3.8.3.2.1 No Local
	// 位置: 订阅选项字节的bit 2
	// 值:
	// - 0: 应用消息可以被发送给发布者自己
	// - 1: 应用消息不能被发送给发布者自己
	// 用途: 防止客户端收到自己发布的消息
	NoLocal uint8

	// RetainAsPublished 保留为已发布标志 (v5.0新增)
	// 参考章节: 3.8.3.2.2 Retain as Published
	// 位置: 订阅选项字节的bit 3
	// 值:
	// - 0: 当服务端向客户端转发应用消息时，必须设置RETAIN标志为0
	// - 1: 当服务端向客户端转发应用消息时，必须保持RETAIN标志不变
	// 用途: 控制保留消息的转发行为
	RetainAsPublished uint8

	// RetainHandling 保留处理选项 (v5.0新增)
	// 参考章节: 3.8.3.2.3 Retain Handling
	// 位置: 订阅选项字节的bits 5-4
	// 值:
	// - 0x00: 发送保留消息，即使订阅是新的
	// - 0x01: 只在订阅是新的时发送保留消息
	// - 0x02: 不发送保留消息
	// 注意: 0x03是保留值，不允许使用
	RetainHandling uint8
}

func (s *Subscription) String() string {
	return fmt.Sprintf("%s@%d", s.TopicFilter, s.MaximumQoS)
}

// SubscribeProperties 订阅属性 (v5.0新增)
// 参考章节: 3.8.2.2 SUBSCRIBE Properties
// 包含各种订阅选项，用于扩展订阅功能
//
// 版本差异:
// - v3.1.1: 不支持属性系统
// - v5.0: 完整的属性系统，支持订阅标识符、用户属性等
type SubscribeProperties struct {
	// SubscriptionIdentifier 订阅标识符
	// 属性标识符: 11 (0x0B)
	// 参考章节: 3.8.2.2.2 Subscription Identifier
	// 类型: 变长字节整数
	// 含义: 标识订阅的数值，用于标识消息应该发送给哪个订阅
	// 注意: 包含多个订阅标识符将造成协议错误
	SubscriptionIdentifier SubscriptionIdentifier

	// UserProperty 用户属性
	// 属性标识符: 38 (0x26)
	// 参考章节: 3.8.2.2.3 User Property
	// 类型: UTF-8字符串对
	// 含义: 用户定义的名称/值对，可以出现多次
	// 注意: 用户属性可以出现多次，表示多个名字/值对
	UserProperty UserProperty
}

func (props *SubscribeProperties) Pack() ([]byte, error) {
	buf := GetBuffer()
	defer PutBuffer(buf)

	if props.SubscriptionIdentifier != 0 {
		buf.WriteByte(0x0B)
		vb, err := encodeLength(props.SubscriptionIdentifier)
		if err != nil {
			return nil, err
		}
		buf.Write(vb)
	}
	if err := props.UserProperty.Pack(buf); err != nil {
		return nil, err
	}

	return bytes.Clone(buf.Bytes()), nil
}

func (props *SubscribeProperties) Unpack(buf *bytes.Buffer) error {
	propsLen, err := decodeLength(buf)
	if err != nil {
		return err
	}
	for i := uint32(0); i < propsLen; i++ {
		propsCode, err := decodeLength(buf)
		if err != nil {
			return err
		}
		uLen := uint32(0)
		switch propsCode {
		case 0x0B:
			if uLen, err = props.SubscriptionIdentifier.Unpack(buf); err != nil {
				return err
			}
		case 0x26:
			if uLen, err = props.UserProperty.Unpack(buf); err != nil {
				return err
			}
		default:
			return ErrProtocolViolation
		}
		i += uLen
	}
	return nil
}
