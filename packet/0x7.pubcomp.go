package packet

import (
	"bytes"
	"encoding/binary"
	"io"
)

// PUBCOMP 发布完成报文 (QoS 2第三步)
//
// MQTT v3.1.1: 参考章节 3.7 PUBCOMP - Publish complete (QoS 2 publish received, part 3)
// MQTT v5.0: 参考章节 3.7 PUBCOMP - Publish complete (QoS 2 publish received, part 3)
//
// 报文结构:
// 固定报头: 报文类型0x07，标志位必须为0
// 可变报头: 报文标识符、原因码(v5.0)、发布完成属性(v5.0)
// 载荷: 无载荷
//
// 版本差异:
// - v3.1.1: 基本的发布完成功能，只包含报文标识符
// - v5.0: 在v3.1.1基础上增加了原因码和属性系统，提供更详细的完成信息
//
// 用途:
// - 用于QoS 2消息传递流程的最后一步
// - 服务端确认收到PUBREL后，发送PUBCOMP完成消息
// - 完成QoS 2的可靠消息传递机制
//
// QoS 2流程:
// 1. 客户端发送PUBLISH (QoS=2)
// 2. 服务端响应PUBREC
// 3. 客户端发送PUBREL
// 4. 服务端响应PUBCOMP ← 当前报文
//
// 标志位规则:
// - DUP: 必须为0
// - QoS: 必须为0
// - RETAIN: 必须为0
type PUBCOMP struct {
	*FixedHeader `json:"FixedHeader,omitempty"`

	// PacketID 报文标识符
	// 参考章节: 2.3.1 Packet Identifier
	// 位置: 可变报头第1个字段
	// 要求: 必须包含，范围1-65535
	// 用途: 用于标识对应的PUBLISH报文，确保QoS 2流程的可靠性
	PacketID uint16 `json:"PacketID,omitempty"`

	// ReasonCode 原因码 (v5.0新增)
	// 参考章节: 3.7.2.2 PUBCOMP Reason Code
	// 位置: 可变报头，在报文标识符之后
	// 类型: 单字节
	// 含义: 表示发布完成的结果
	// 常见值:
	// - 0x00: 成功 - 消息已完成
	// - 0x92: 报文标识符未找到 - 找不到对应的PUBLISH报文
	// 注意: v3.1.1不支持原因码
	ReasonCode ReasonCode

	// Props 发布完成属性 (v5.0新增)
	// 参考章节: 3.7.2.3 PUBCOMP Properties
	// 位置: 可变报头，在原因码之后
	// 包含原因字符串、用户属性等
	Props *PubcompProperties
}

func (pkt *PUBCOMP) Kind() byte {
	return 0x7
}

func (pkt *PUBCOMP) Pack(w io.Writer) error {
	// 确保标志位正确设置
	pkt.Dup = 0
	pkt.QoS = 0
	pkt.Retain = 0

	buf := GetBuffer()
	defer PutBuffer(buf)
	buf.Write(i2b(pkt.PacketID))
	if pkt.Version == VERSION500 {
		buf.WriteByte(pkt.ReasonCode.Code)

		pkt.Props = &PubcompProperties{}
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

func (pkt *PUBCOMP) Unpack(buf *bytes.Buffer) error {

	pkt.PacketID = binary.BigEndian.Uint16(buf.Next(2))

	if pkt.Version == VERSION500 {
		pkt.ReasonCode.Code = buf.Next(1)[0]

		pkt.Props = &PubcompProperties{}
		if err := pkt.Props.Unpack(buf); err != nil {
			return err
		}
	}
	return nil
}

// PubcompProperties 发布完成属性 (v5.0新增)
// 参考章节: 3.7.2.3 PUBCOMP Properties
// 包含各种发布完成选项，用于扩展完成功能
//
// 版本差异:
// - v3.1.1: 不支持属性系统
// - v5.0: 完整的属性系统，支持原因字符串、用户属性等
type PubcompProperties struct {
	// ReasonString 原因字符串
	// 属性标识符: 31 (0x1F)
	// 参考章节: 3.7.2.3.2 Reason String
	// 类型: UTF-8编码字符串
	// 含义: 表示此次发布完成相关的原因
	// 注意:
	// - 此原因字符串是为诊断而设计的可读字符串，不应该被客户端所解析
	// - 包含多个原因字符串将造成协议错误
	// - 用于提供额外的发布完成信息
	ReasonString ReasonString

	// UserProperty 用户属性
	// 属性标识符: 38 (0x26)
	// 参考章节: 3.7.2.3.3 User Property
	// 类型: UTF-8字符串对
	// 含义: 用户定义的名称/值对，可以出现多次
	// 注意:
	// - 用户属性可以出现多次，表示多个名字/值对
	// - 相同的名字可以出现多次
	// - 本规范不做定义，由应用程序确定含义和解释
	// - 可用于传递发布完成相关的额外信息
	UserProperty UserProperty
}

func (props *PubcompProperties) Pack() ([]byte, error) {
	buf := &bytes.Buffer{}
	if err := props.ReasonString.Pack(buf); err != nil {
		return nil, err
	}
	if err := props.UserProperty.Pack(buf); err != nil {
		return nil, err
	}
	return bytes.Clone(buf.Bytes()), nil
}

func (props *PubcompProperties) Unpack(buf *bytes.Buffer) error {
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
		}
		i += uLen
	}
	return nil
}
