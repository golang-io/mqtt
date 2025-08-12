package packet

import (
	"bytes"
	"encoding/binary"
	"io"
)

// SUBACK 订阅确认报文
//
// MQTT v3.1.1: 参考章节 3.9 SUBACK - Subscribe acknowledgement
// MQTT v5.0: 参考章节 3.9 SUBACK - Subscribe acknowledgement
//
// 报文结构:
// 固定报头: 报文类型0x09，标志位必须为0
// 可变报头: 报文标识符、订阅确认属性(v5.0)
// 载荷: 订阅返回码列表，每个返回码对应一个订阅请求
//
// 版本差异:
// - v3.1.1: 基本的订阅确认功能，包含报文标识符和返回码列表
// - v5.0: 在v3.1.1基础上增加了属性系统，支持原因字符串、用户属性等
//
// 用途:
// - 用于确认SUBSCRIBE报文的处理结果
// - 为每个订阅请求提供QoS等级反馈
// - 通知客户端订阅是否成功
//
// 标志位规则:
// - DUP: 必须为0
// - QoS: 必须为0
// - RETAIN: 必须为0
type SUBACK struct {
	*FixedHeader `json:"FixedHeader,omitempty"`

	// PacketID 报文标识符
	// 参考章节: 2.3.1 Packet Identifier
	// 位置: 可变报头第1个字段
	// 要求: 必须包含，范围1-65535
	// 用途: 用于标识对应的SUBSCRIBE报文，确保确认的可靠性
	PacketID uint16 `json:"PacketID,omitempty"`

	// SubackProps 订阅确认属性 (v5.0新增)
	// 参考章节: 3.9.2.2 SUBACK Properties
	// 位置: 可变报头，在报文标识符之后
	// 包含原因字符串、用户属性等
	SubackProps *SubackProperties

	// 载荷部分
	// 参考章节: 3.9.3 SUBACK Payload
	// 位置: 载荷部分
	// 要求: 至少包含一个返回码
	// 每个返回码对应SUBSCRIBE报文中的一个订阅请求
	// 返回码值:
	// - 0x00: 最大 QoS 0 - 订阅成功，最大QoS为0
	// - 0x01: 最大 QoS 1 - 订阅成功，最大QoS为1
	// - 0x02: 最大 QoS 2 - 订阅成功，最大QoS为2
	// - 0x80: 失败 - 订阅失败
	// 注意: 返回码列表的顺序必须与SUBSCRIBE报文中的订阅请求顺序一致
	ReasonCode []ReasonCode `json:"ReasonCode,omitempty"`
}

func (pkt *SUBACK) Kind() byte {
	return 0x9
}

func (pkt *SUBACK) Pack(w io.Writer) error {
	buf := GetBuffer()
	defer PutBuffer(buf)
	if len(pkt.ReasonCode) == 0 {
		return ErrMalformedReasonCode
	}
	buf.Write(i2b(pkt.PacketID))

	if pkt.Version == VERSION500 {
		if pkt.SubackProps == nil {
			pkt.SubackProps = &SubackProperties{}
		}
		b, err := pkt.SubackProps.Pack()
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

	for _, reason := range pkt.ReasonCode {
		buf.WriteByte(reason.Code)
	}
	pkt.FixedHeader.RemainingLength = uint32(buf.Len())
	if err := pkt.FixedHeader.Pack(w); err != nil {
		return err
	}
	_, err := buf.WriteTo(w)
	return err
}

func (pkt *SUBACK) Unpack(buf *bytes.Buffer) error {
	pkt.PacketID = binary.BigEndian.Uint16(buf.Next(2))

	if pkt.Version == VERSION500 {
		pkt.SubackProps = &SubackProperties{}
		if err := pkt.SubackProps.Unpack(buf); err != nil {
			return err
		}
	}

	for buf.Len() != 0 {
		reason := ReasonCode{Code: buf.Next(1)[0]}
		if reason.Code == 0x80 {
			return ErrMalformedReasonCode
		}
		if reason.Code > 0x02 {
			return ErrMalformedReasonCode
		}
		pkt.ReasonCode = append(pkt.ReasonCode, reason)
	}
	return nil
}

// SubackProperties 订阅确认属性 (v5.0新增)
// 参考章节: 3.9.2.2 SUBACK Properties
// 包含各种订阅确认选项，用于扩展确认功能
//
// 版本差异:
// - v3.1.1: 不支持属性系统
// - v5.0: 完整的属性系统，支持原因字符串、用户属性等
type SubackProperties struct {
	// ReasonString 原因字符串
	// 属性标识符: 31 (0x1F)
	// 参考章节: 3.9.2.2.2 Reason String
	// 类型: UTF-8编码字符串
	// 含义: 表示此次订阅确认相关的原因
	// 注意:
	// - 此原因字符串是为诊断而设计的可读字符串，不应该被客户端所解析
	// - 包含多个原因字符串将造成协议错误
	// - 用于提供额外的订阅确认信息
	ReasonString ReasonString

	// UserProperty 用户属性
	// 属性标识符: 38 (0x26)
	// 参考章节: 3.9.2.2.3 User Property
	// 类型: UTF-8字符串对
	// 含义: 用户定义的名称/值对，可以出现多次
	// 注意:
	// - 用户属性可以出现多次，表示多个名字/值对
	// - 相同的名字可以出现多次
	// - 本规范不做定义，由应用程序确定含义和解释
	// - 可用于传递订阅确认相关的额外信息
	UserProperty UserProperty
}

func (props *SubackProperties) Pack() ([]byte, error) {
	buf := GetBuffer()
	defer PutBuffer(buf)

	if err := props.ReasonString.Pack(buf); err != nil {
		return nil, err
	}

	if err := props.UserProperty.Pack(buf); err != nil {
		return nil, err
	}

	return bytes.Clone(buf.Bytes()), nil
}

func (props *SubackProperties) Unpack(buf *bytes.Buffer) error {
	propsLen, err := decodeLength(buf)
	if err != nil {
		return err
	}

	for i := uint32(0); i < propsLen; i++ {
		propsId, err := decodeLength(buf)
		if err != nil {
			return err
		}
		uLen := uint32(0)
		switch propsId {
		case 0x1F: // ReasonString
			if uLen, err = props.ReasonString.Unpack(buf); err != nil {
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
