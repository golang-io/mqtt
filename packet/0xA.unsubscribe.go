package packet

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// UNSUBSCRIBE 取消订阅请求报文
//
// MQTT v3.1.1: 参考章节 3.10 UNSUBSCRIBE - Unsubscribe from topics
// MQTT v5.0: 参考章节 3.10 UNSUBSCRIBE - Unsubscribe from topics
//
// 报文结构:
// 固定报头: 报文类型0x0A，标志位必须为DUP=0, QoS=1, RETAIN=0
// 可变报头: 报文标识符、取消订阅属性(v5.0)
// 载荷: 主题过滤器列表，每个主题过滤器对应一个要取消的订阅
//
// 版本差异:
// - v3.1.1: 基本的取消订阅功能，包含报文标识符和主题过滤器列表
// - v5.0: 在v3.1.1基础上增加了属性系统，支持用户属性等
//
// 用途:
// - 用于客户端取消之前建立的订阅
// - 停止接收特定主题的消息
// - 管理客户端的订阅状态
//
// 标志位规则:
// - DUP: 必须为0
// - QoS: 必须为1
// - RETAIN: 必须为0
type UNSUBSCRIBE struct {
	*FixedHeader

	// PacketID 报文标识符
	// 参考章节: 2.3.1 Packet Identifier
	// 位置: 可变报头第1个字段
	// 要求: 必须包含，范围1-65535
	// 用途: 用于标识取消订阅请求，确保确认的可靠性
	PacketID uint16

	// Subscriptions 主题过滤器列表
	// 参考章节: 3.10.3 UNSUBSCRIBE Payload
	// 位置: 载荷部分
	// 要求: 至少包含一个主题过滤器
	// 每个主题过滤器对应一个要取消的订阅
	// 注意: 主题过滤器必须与之前SUBSCRIBE报文中的完全匹配
	Subscriptions []Subscription

	// Props 取消订阅属性 (v5.0新增)
	// 参考章节: 3.10.2.2 UNSUBSCRIBE Properties
	// 位置: 可变报头，在报文标识符之后
	// 包含用户属性等
	Props *UnsubscribeProperties
}

func (pkt *UNSUBSCRIBE) Kind() byte {
	return 0xA
}

func (pkt *UNSUBSCRIBE) Pack(w io.Writer) error {
	// 检查是否至少包含一个主题过滤器
	if len(pkt.Subscriptions) == 0 {
		return ErrMalformedTopic
	}

	buf := GetBuffer()
	defer PutBuffer(buf)
	buf.Write(i2b(pkt.PacketID))

	// 写入主题过滤器
	for _, subscription := range pkt.Subscriptions {
		buf.Write(s2b(subscription.TopicFilter))
	}

	// 计算初始长度
	pkt.FixedHeader.RemainingLength = uint32(buf.Len())
	if pkt.Version == VERSION500 {
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
	pkt.FixedHeader.RemainingLength = uint32(buf.Len())

	if err := pkt.FixedHeader.Pack(w); err != nil {
		return err
	}
	_, err := buf.WriteTo(w)
	return err
}

func (pkt *UNSUBSCRIBE) Unpack(buf *bytes.Buffer) error {
	// 检查是否有足够的数据读取报文标识符
	if buf.Len() < 2 {
		return ErrMalformedPacketID
	}

	pkt.PacketID = binary.BigEndian.Uint16(buf.Next(2))

	// 处理MQTT v5.0属性
	if pkt.Version == VERSION500 {
		pkt.Props = &UnsubscribeProperties{}
		if err := pkt.Props.Unpack(buf); err != nil {
			return err
		}
	}

	for buf.Len() != 0 {
		topicLength := int(binary.BigEndian.Uint16(buf.Next(2))) // topic length
		subscription := Subscription{TopicFilter: string(buf.Next(topicLength))}
		pkt.Subscriptions = append(pkt.Subscriptions, subscription)
	}

	// 检查是否至少有一个主题过滤器
	if len(pkt.Subscriptions) == 0 {
		return ErrMalformedTopic
	}

	return nil
}

// UnsubscribeProperties 取消订阅属性 (v5.0新增)
// 参考章节: 3.10.2.2 UNSUBSCRIBE Properties
// 包含各种取消订阅选项，用于扩展取消订阅功能
//
// 版本差异:
// - v3.1.1: 不支持属性系统
// - v5.0: 完整的属性系统，支持用户属性等
type UnsubscribeProperties struct {
	// UserProperty 用户属性
	// 属性标识符: 38 (0x26)
	// 参考章节: 3.10.2.2.2 User Property
	// 类型: UTF-8字符串对
	// 含义: 用户定义的名称/值对，可以出现多次
	// 注意:
	// - 用户属性可以出现多次，表示多个名字/值对
	// - 相同的名字可以出现多次
	// - 本规范不做定义，由应用程序确定含义和解释
	// - 可用于传递取消订阅相关的额外信息
	UserProperty map[string][]string
}

func (props *UnsubscribeProperties) Pack() ([]byte, error) {
	buf := GetBuffer()
	defer PutBuffer(buf)

	if len(props.UserProperty) != 0 {
		for k, v := range props.UserProperty {
			for i := range v {
				buf.WriteByte(0x26)
				buf.Write(encodeUTF8(k))
				buf.Write(encodeUTF8(v[i]))
			}
		}
	}
	return buf.Bytes(), nil
}

func (props *UnsubscribeProperties) Unpack(buf *bytes.Buffer) error {
	propsLen, err := decodeLength(buf)
	if err != nil {
		return err
	}
	for i := uint32(0); i < propsLen; i++ {
		propsCode, err := decodeLength(buf)
		if err != nil {
			return err
		}
		switch propsCode {
		case 0x26:
			if props.UserProperty == nil {
				props.UserProperty = make(map[string][]string)
			}
			userProperty := &UserProperty{}
			uLen, err := userProperty.Unpack(buf)
			if err != nil {
				return fmt.Errorf("failed to unpack user property: %w", err)
			}
			props.UserProperty[userProperty.Name] = append(props.UserProperty[userProperty.Name], userProperty.Value)
			i += uLen
		default:
			return ErrProtocolViolation
		}
	}
	return nil
}
