package packet

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// PUBACK 发布确认报文 (QoS 1)
//
// MQTT v3.1.1: 参考章节 3.4 PUBACK - Publish acknowledgement
// MQTT v5.0: 参考章节 3.4 PUBACK - Publish acknowledgement
//
// 报文结构:
// 固定报头: 报文类型0x04，标志位必须为0
// 可变报头: 报文标识符、原因码(v5.0)、发布确认属性(v5.0)
// 载荷: 无载荷
//
// 版本差异:
// - v3.1.1: 基本的发布确认功能，只包含报文标识符
// - v5.0: 在v3.1.1基础上增加了原因码和属性系统，提供更详细的确认信息
//
// 用途:
// - 用于确认QoS 1的PUBLISH报文
// - 确保消息至少一次传递
// - 提供消息传递状态的反馈
//
// 标志位规则:
// - DUP: 必须为0
// - QoS: 必须为0
// - RETAIN: 必须为0
type PUBACK struct {
	*FixedHeader

	// 可变报头部分
	// 参考章节: 3.4.2 Variable header

	// PacketID 报文标识符
	// 参考章节: 2.3.1 Packet Identifier
	// 位置: 可变报头第1个字段
	// 要求: 必须包含，范围1-65535
	// 用途: 用于标识对应的PUBLISH报文，确保确认的可靠性
	PacketID uint16

	// ReasonCode 原因码 (v5.0新增)
	// 参考章节: 3.4.2.2 PUBACK Reason Code
	// 位置: 可变报头，在报文标识符之后
	// 类型: 单字节
	// 含义: 表示发布确认的结果
	// 常见值:
	// - 0x00: 成功 - 消息已确认
	// - 0x10: 无匹配订阅者 - 没有订阅者接收此消息
	// - 0x80: 未指定错误 - 未指定的错误
	// - 0x83: 实现特定错误 - 实现特定的错误
	// 注意: v3.1.1不支持原因码
	ReasonCode ReasonCode

	// Props 发布确认属性 (v5.0新增)
	// 参考章节: 3.4.2.3 PUBACK Properties
	// 位置: 可变报头，在原因码之后
	// 包含原因字符串、用户属性等
	Props *PubackProperties
}

func (pkt *PUBACK) Kind() byte {
	return 0x4
}

func (pkt *PUBACK) Pack(w io.Writer) error {
	buf := GetBuffer()
	defer PutBuffer(buf)
	pkt.RemainingLength = 2
	buf.Write(i2b(pkt.PacketID))
	if pkt.Version == VERSION500 {
		buf.WriteByte(pkt.ReasonCode.Code)
		pkt.RemainingLength += 1

		pkt.Props = &PubackProperties{}
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

func (pkt *PUBACK) Unpack(buf *bytes.Buffer) error {

	pkt.PacketID = binary.BigEndian.Uint16(buf.Next(2))

	if pkt.Version == VERSION500 {
		pkt.ReasonCode.Code = buf.Next(1)[0]

		pkt.Props = &PubackProperties{}
		if err := pkt.Props.Unpack(buf); err != nil {
			return err
		}
	}
	return nil
}

// PubackProperties 发布确认属性 (v5.0新增)
// 参考章节: 3.4.2.3 PUBACK Properties
// 包含各种发布确认选项，用于扩展确认功能
//
// 版本差异:
// - v3.1.1: 不支持属性系统
// - v5.0: 完整的属性系统，支持原因字符串、用户属性等
type PubackProperties struct {
	// ReasonString 原因字符串
	// 属性标识符: 31 (0x1F)
	// 参考章节: 3.4.2.3.2 Reason String
	// 类型: UTF-8编码字符串
	// 含义: 表示此次确认相关的原因
	// 注意:
	// - 此原因字符串是为诊断而设计的可读字符串，不应该被客户端所解析
	// - 包含多个原因字符串将造成协议错误
	// - 用于提供额外的确认信息
	ReasonString ReasonString

	// UserProperty 用户属性
	// 属性标识符: 38 (0x26)
	// 参考章节: 3.4.2.3.3 User Property
	// 类型: UTF-8字符串对
	// 含义: 用户定义的名称/值对，可以出现多次
	// 注意:
	// - 用户属性可以出现多次，表示多个名字/值对
	// - 相同的名字可以出现多次
	// - 本规范不做定义，由应用程序确定含义和解释
	// - 可用于传递确认相关的额外信息
	UserProperty UserProperty
}

func (props *PubackProperties) Pack() ([]byte, error) {
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

func (props *PubackProperties) Unpack(buf *bytes.Buffer) error {
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
		case 0x1F: // 会话过期间隔 Session Expiry Interval
			if uLen, err = props.ReasonString.Unpack(buf); err != nil {
				return err
			}
		case 0x26:
			if uLen, err = props.UserProperty.Unpack(buf); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown property identifier: 0x%02X", propsId)
		}
		i += uLen
	}
	return nil
}
