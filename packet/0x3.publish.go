package packet

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"strings"
)

// PUBLISH 发布消息报文
//
// MQTT v3.1.1: 参考章节 3.3 PUBLISH - Publish message
// MQTT v5.0: 参考章节 3.3 PUBLISH - Publish message
//
// 报文结构:
// 固定报头: 报文类型0x03，标志位包含DUP、QoS、RETAIN
// 可变报头: 主题名、报文标识符(QoS>0时)、属性(v5.0)
// 载荷: 应用消息内容
//
// 版本差异:
// - v3.1.1: 基本的发布功能，支持QoS 0/1/2，支持保留消息
// - v5.0: 在v3.1.1基础上增加了属性系统，支持主题别名、消息过期、载荷格式指示等
//
// 标志位规则:
// - DUP: 只有QoS > 0的报文才能设置，表示重复发送
// - QoS: 0(最多一次)、1(至少一次)、2(恰好一次)
// - RETAIN: 表示消息是否应该被服务端保留
type PUBLISH struct {
	*FixedHeader `json:"FixedHeader,omitempty"`

	// PacketID 报文标识符
	// 参考章节: 2.3.1 Packet Identifier
	// 位置: 可变报头，在主题名之后(QoS > 0时)
	// 要求:
	// - QoS = 0: 不能包含报文标识符 [MQTT-2.3.1-5]
	// - QoS > 0: 必须包含报文标识符，范围1-65535
	// 用途: 用于标识QoS > 0的发布消息，确保消息传递的可靠性
	PacketID uint16 `json:"PacketID,omitempty"`

	Message *Message `json:"message,omitempty"`

	// Props 发布属性 (v5.0新增)
	// 参考章节: 3.3.2.3 PUBLISH Properties
	// 位置: 可变报头，在报文标识符之后(QoS > 0时)
	// 包含各种发布选项，如主题别名、消息过期、载荷格式等
	Props *PublishProperties `json:"properties,omitempty"`
}

func (pkt *PUBLISH) Kind() byte {
	return 0x3
}

func (pkt *PUBLISH) Pack(w io.Writer) error {
	buf := GetBuffer()
	defer PutBuffer(buf)
	buf.Write(s2b(pkt.Message.TopicName))
	// QoS 设置为 0 的 Publish 报文不能包含报文标识符 [MQTT-2.3.1-5]。
	if pkt.QoS != 0 {
		buf.Write(i2b(pkt.PacketID))
	}
	if pkt.Version == VERSION500 {
		if pkt.Props == nil {
			pkt.Props = &PublishProperties{}
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

	buf.Write(pkt.Message.Content)
	pkt.FixedHeader.RemainingLength = uint32(buf.Len())
	//fmt.Printf("buf.Size=%d, topic=%s, content=%s, packet=%v\n", buf.Len(), pkt.Message.TopicName, pkt.Message.Content, buf.Bytes())
	if err := pkt.FixedHeader.Pack(w); err != nil {
		return err
	}

	_, err := buf.WriteTo(w)
	return err
}

func (pkt *PUBLISH) Unpack(buf *bytes.Buffer) error {
	topicLength := int(binary.BigEndian.Uint16(buf.Next(2))) // topic length
	if pkt.Message == nil {
		pkt.Message = &Message{}
	}
	pkt.Message.TopicName = string(buf.Next(topicLength))

	// Publish 报文中的主题名不能包含通配符 [MQTT-3.3.2-2]。
	if pkt.Message.TopicName == "" || strings.Contains(pkt.Message.TopicName, " ") {

		return fmt.Errorf("pkt.RemainingLength=%v, err=%w", pkt.RemainingLength, ErrTopicNameInvalid)
	}
	// QoS 设置为 0 的 Publish 报文不能包含报文标识符 [MQTT-2.3.1-5]。
	if pkt.QoS != 0 {
		pkt.PacketID = binary.BigEndian.Uint16(buf.Next(2))
	}

	if pkt.Version == VERSION500 {
		pkt.Props = &PublishProperties{}
		if err := pkt.Props.Unpack(buf); err != nil {
			return fmt.Errorf("pkt.RemainingLength=%v err=%w", pkt.RemainingLength, err)
		}
	}

	pkt.Message.Content = buf.Bytes()
	return nil
}

// Message 发布消息内容
// 参考章节: 3.3.3 PUBLISH Payload
// 包含主题名和消息内容
//
// 版本差异:
// - v3.1.1: 基本的主题名和消息内容
// - v5.0: 支持更多消息属性，如载荷格式指示、内容类型等
type Message struct {
	// TopicName 主题名
	// 参考章节: 3.3.2.1 Topic Name
	// 位置: 可变报头第1个字段
	// 要求:
	// - UTF-8编码字符串
	// - 不能为空
	// - 不能包含通配符 [MQTT-3.3.2-2]
	// - 不能包含空格字符
	// 用途: 标识消息应该发布到哪个主题
	TopicName string

	// Content 消息内容
	// 参考章节: 3.3.3 PUBLISH Payload
	// 位置: 载荷部分
	// 类型: 二进制数据
	// 注意: 包含零长度有效载荷的Publish报文是合法的
	// 用途: 实际的应用消息内容
	Content []byte
}

func (m *Message) String() string {
	return fmt.Sprintf("%s # %s", m.TopicName, m.Content)
}

// PublishProperties 发布属性 (v5.0新增)
// 参考章节: 3.3.2.3 PUBLISH Properties
// 包含各种发布选项，用于扩展发布功能
//
// 版本差异:
// - v3.1.1: 不支持属性系统
// - v5.0: 完整的属性系统，支持主题别名、消息过期、载荷格式等
type PublishProperties struct {
	// PayloadFormatIndicator 载荷格式指示
	// 属性标识符: 1 (0x01)
	// 参考章节: 3.3.2.3.2 Payload Format Indicator
	// 类型: 单字节
	// 值:
	// - 0 (0x00): 表示载荷是未指定的字节，等同于不发送载荷格式指示
	// - 1 (0x01): 表示载荷是UTF-8编码的字符数据
	// 注意: 包含多个载荷格式指示将造成协议错误
	PayloadFormatIndicator uint8

	// MessageExpiryInterval 消息过期间隔
	// 属性标识符: 2 (0x02)
	// 参考章节: 3.3.2.3.3 Message Expiry Interval
	// 类型: 四字节整数，单位: 秒
	// 含义: 消息的生命周期
	// 注意: 包含多个消息过期间隔将造成协议错误
	MessageExpiryInterval uint32

	// TopicAlias 主题别名
	// 属性标识符: 35 (0x23)
	// 参考章节: 3.3.2.3.4 Topic Alias
	// 类型: 双字节整数
	// 含义: 用于标识主题的数值
	// 注意:
	// - 包含多个主题别名将造成协议错误
	// - 主题别名值必须大于0
	// - 主题别名只在当前网络连接中有效
	TopicAlias uint16

	// ResponseTopic 响应主题
	// 属性标识符: 8 (0x08)
	// 参考章节: 3.3.2.3.5 Response Topic
	// 类型: UTF-8编码字符串
	// 含义: 表示响应消息的主题名
	// 注意: 包含多个响应主题将造成协议错误
	ResponseTopic string

	// CorrelationData 对比数据
	// 属性标识符: 9 (0x09)
	// 参考章节: 3.3.2.3.6 Correlation Data
	// 类型: 二进制数据
	// 含义: 被请求消息发送端在收到响应消息时用来标识相应的请求
	// 注意: 包含多个对比数据将造成协议错误
	CorrelationData []byte

	// UserProperty 用户属性
	// 属性标识符: 38 (0x26)
	// 参考章节: 3.3.2.3.7 User Property
	// 类型: UTF-8字符串对
	// 含义: 用户定义的名称/值对，可以出现多次
	// 注意: 用户属性可以出现多次，表示多个名字/值对
	UserProperty map[string][]string

	// SubscriptionIdentifier 订阅标识符
	// 属性标识符: 11 (0x0B)
	// 参考章节: 3.3.2.3.8 Subscription Identifier
	// 类型: 变长字节整数
	// 含义: 标识订阅的数值，用于标识消息应该发送给哪个订阅
	// 注意: 可以包含多个订阅标识符
	SubscriptionIdentifier []uint32

	// ContentType 内容类型
	// 属性标识符: 3 (0x03)
	// 参考章节: 3.3.2.3.9 Content Type
	// 类型: UTF-8编码字符串
	// 含义: 描述载荷的内容
	// 注意: 包含多个内容类型将造成协议错误
	ContentType string
}

func (props *PublishProperties) Unpack(b *bytes.Buffer) error {
	propsLen, err := decodeLength(b)
	if err != nil {
		return err
	}
	for i := uint32(0); i < propsLen; i++ {
		propsId, err := decodeLength(b)
		if err != nil {
			return err
		}
		switch propsId {
		case 0x01:
			props.PayloadFormatIndicator, i = b.Next(1)[0], i+1
		case 0x02:
			props.MessageExpiryInterval, i = binary.BigEndian.Uint32(b.Next(4)), i+4
		case 0x23:
			props.TopicAlias, i = binary.BigEndian.Uint16(b.Next(2)), i+2
		case 0x08:
			props.ResponseTopic, i = decodeUTF8[string](b), i+uint32(len(props.ResponseTopic))
		case 0x09:
			props.CorrelationData, i = decodeUTF8[[]byte](b), i+uint32(len(props.CorrelationData))
		case 0x26:
			if props.UserProperty == nil {
				props.UserProperty = make(map[string][]string)
			}
			key := decodeUTF8[string](b)
			props.UserProperty[key] = append(props.UserProperty[key], decodeUTF8[string](b))
		case 0x0B:
			subscriptionIdentifier, err := decodeLength(b)
			if err != nil {
				return err
			}
			props.SubscriptionIdentifier = append(props.SubscriptionIdentifier, subscriptionIdentifier)
			vb, _ := encodeLength(subscriptionIdentifier)
			i += uint32(len(vb)) // 用来计算动态
		case 0x03:
			props.ContentType, i = decodeUTF8[string](b), i+uint32(len(props.ContentType))

		}
	}
	return nil
}

func (props *PublishProperties) Pack() ([]byte, error) {
	buf := GetBuffer()
	defer PutBuffer(buf)
	if props.PayloadFormatIndicator != 0 {
		buf.WriteByte(0x01)
		buf.WriteByte(props.PayloadFormatIndicator)
	}
	if props.MessageExpiryInterval != 0 {
		buf.WriteByte(0x02)
		buf.Write(i4b(props.MessageExpiryInterval))
	}
	if props.TopicAlias != 0 {
		buf.WriteByte(0x23)
		buf.Write(i2b(props.TopicAlias))
	}
	if props.ResponseTopic != "" {
		buf.WriteByte(0x08)
		buf.Write(encodeUTF8(props.ResponseTopic))
	}
	if props.CorrelationData != nil {
		buf.WriteByte(0x09)
		buf.Write(encodeUTF8(props.CorrelationData))
	}
	if len(props.UserProperty) != 0 {
		for k, v := range props.UserProperty {
			for i := range v {
				buf.WriteByte(0x26)
				buf.Write(encodeUTF8(k))
				buf.Write(encodeUTF8(v[i]))
			}
		}
	}
	if len(props.SubscriptionIdentifier) != 0 {
		for _, subscriptionIdentifier := range props.SubscriptionIdentifier {
			buf.WriteByte(0x0B)
			v, err := encodeLength(subscriptionIdentifier)
			if err != nil {
				return nil, err
			}
			buf.Write(v)
		}
	}
	if props.ContentType != "" {
		buf.WriteByte(0x03)
		buf.Write(encodeUTF8(props.ContentType))
	}
	return buf.Bytes(), nil

}
