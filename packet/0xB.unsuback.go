package packet

import (
	"bytes"
	"encoding/binary"
	"io"
)

// UNSUBACK 取消订阅确认报文
//
// MQTT v3.1.1: 参考章节 3.11 UNSUBACK - Unsubscribe acknowledgement
// MQTT v5.0: 参考章节 3.11 UNSUBACK - Unsubscribe acknowledgement
//
// 报文结构:
// 固定报头: 报文类型0x0B，标志位必须为0
// 可变报头: 报文标识符、取消订阅确认属性(v5.0)
// 载荷: 无载荷
//
// 版本差异:
// - v3.1.1: 基本的取消订阅确认功能，只包含报文标识符
// - v5.0: 在v3.1.1基础上增加了属性系统，支持原因字符串、用户属性等
//
// 用途:
// - 用于确认UNSUBSCRIBE报文的处理结果
// - 通知客户端取消订阅操作是否成功
// - 完成取消订阅的握手过程
//
// 标志位规则:
// - DUP: 必须为0
// - QoS: 必须为0
// - RETAIN: 必须为0
type UNSUBACK struct {
	*FixedHeader `json:"FixedHeader,omitempty"`

	// PacketID 报文标识符
	// 参考章节: 2.3.1 Packet Identifier
	// 位置: 可变报头第1个字段
	// 要求: 必须包含，范围1-65535
	// 用途: 用于标识对应的UNSUBSCRIBE报文，确保确认的可靠性
	PacketID uint16 `json:"PacketID,omitempty"`

	// Props 取消订阅确认属性 (v5.0新增)
	// 参考章节: 3.11.2.2 UNSUBACK Properties
	// 位置: 可变报头，在报文标识符之后
	// 包含原因字符串、用户属性等
	Props *UnsubackProperties
}

func (pkt *UNSUBACK) Kind() byte {
	return 0xB
}

func (pkt *UNSUBACK) Pack(w io.Writer) error {
	buf := GetBuffer()
	defer PutBuffer(buf)

	buf.Write(i2b(pkt.PacketID))

	if pkt.Version == VERSION500 {
		if pkt.Props == nil {
			pkt.Props = &UnsubackProperties{}
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
	pkt.FixedHeader.RemainingLength = uint32(buf.Len())

	if err := pkt.FixedHeader.Pack(w); err != nil {
		return err
	}
	_, err := buf.WriteTo(w)
	return err

}
func (pkt *UNSUBACK) Unpack(buf *bytes.Buffer) error {
	if pkt.FixedHeader.RemainingLength != 2 {
		return ErrMalformedPacket
	}
	pkt.PacketID = binary.BigEndian.Uint16(buf.Bytes())

	switch pkt.Version {
	case VERSION500:
		pkt.Props = &UnsubackProperties{}
		if err := pkt.Props.Unpack(buf); err != nil {
			return err
		}
	case VERSION311:
	case VERSION310:
		return ErrUnsupportedProtocolVersion
	default:
		return ErrMalformedProtocolVersion

	}
	return nil
}

// UnsubackProperties 取消订阅确认属性 (v5.0新增)
// 参考章节: 3.11.2.2 UNSUBACK Properties
// 包含各种取消订阅确认选项，用于扩展确认功能
//
// 版本差异:
// - v3.1.1: 不支持属性系统
// - v5.0: 完整的属性系统，支持原因字符串、用户属性等
type UnsubackProperties struct {
	// ReasonString 原因字符串
	// 属性标识符: 31 (0x1F)
	// 参考章节: 3.11.2.2.2 Reason String
	// 类型: UTF-8编码字符串
	// 含义: 表示此次取消订阅确认相关的原因
	// 注意:
	// - 此原因字符串是为诊断而设计的可读字符串，不应该被客户端所解析
	// - 包含多个原因字符串将造成协议错误
	// - 用于提供额外的取消订阅确认信息
	ReasonString string

	// UserProperty 用户属性
	// 属性标识符: 38 (0x26)
	// 参考章节: 3.11.2.2.3 User Property
	// 类型: UTF-8字符串对
	// 含义: 用户定义的名称/值对，可以出现多次
	// 注意:
	// - 用户属性可以出现多次，表示多个名字/值对
	// - 相同的名字可以出现多次
	// - 本规范不做定义，由应用程序确定含义和解释
	// - 可用于传递取消订阅确认相关的额外信息
	UserProperty map[string][]string
}

func (props *UnsubackProperties) Pack() ([]byte, error) {
	buf := GetBuffer()
	defer PutBuffer(buf)

	if props.ReasonString != "" {
		buf.WriteByte(0x1F)
		buf.Write(encodeUTF8(props.ReasonString))
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
	return buf.Bytes(), nil
}

func (props *UnsubackProperties) Unpack(buf *bytes.Buffer) error {
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
		case 0x1F:
			if props.ReasonString != "" {
				return ErrProtocolErr
			}
			props.ReasonString, i = decodeUTF8[string](buf), i+uint32(len(props.ReasonString))
		case 0x26:
			if props.UserProperty == nil {
				props.UserProperty = make(map[string][]string)
			}
			key := decodeUTF8[string](buf)
			props.UserProperty[key] = append(props.UserProperty[key], decodeUTF8[string](buf))
		}
	}
	return nil
}
