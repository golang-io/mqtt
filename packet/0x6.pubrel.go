package packet

import (
	"bytes"
	"encoding/binary"
	"io"
)

// PUBREL 发布释放报文 (QoS 2第二步)
//
// MQTT v3.1.1: 参考章节 3.6 PUBREL - Publish release (QoS 2 publish received, part 2)
// MQTT v5.0: 参考章节 3.6 PUBREL - Publish release (QoS 2 publish received, part 2)
//
// 报文结构:
// 固定报头: 报文类型0x06，标志位必须为DUP=0, QoS=1, RETAIN=0
// 可变报头: 报文标识符、原因码(v5.0)、发布释放属性(v5.0)
// 载荷: 无载荷
//
// 版本差异:
// - v3.1.1: 基本的发布释放功能，只包含报文标识符
// - v5.0: 在v3.1.1基础上增加了原因码和属性系统，提供更详细的释放信息
//
// 用途:
// - 用于QoS 2消息传递流程的第二步
// - 客户端确认收到PUBREC后，发送PUBREL释放消息
// - 继续QoS 2的可靠消息传递机制
//
// QoS 2流程:
// 1. 客户端发送PUBLISH (QoS=2)
// 2. 服务端响应PUBREC
// 3. 客户端发送PUBREL ← 当前报文
// 4. 服务端响应PUBCOMP
//
// 标志位规则:
// - DUP: 必须为0 [MQTT-3.6.1-1]
// - QoS: 必须为1 [MQTT-3.6.1-1]
// - RETAIN: 必须为0 [MQTT-3.6.1-1]
type PUBREL struct {
	*FixedHeader `json:"FixedHeader,omitempty"`

	// PacketID 报文标识符
	// 参考章节: 2.3.1 Packet Identifier
	// 位置: 可变报头第1个字段
	// 要求: 必须包含，范围1-65535
	// 用途: 用于标识对应的PUBLISH报文，确保QoS 2流程的可靠性
	PacketID uint16 `json:"PacketID,omitempty"`

	// ReasonCode 原因码 (v5.0新增)
	// 参考章节: 3.6.2.2 PUBREL Reason Code
	// 位置: 可变报头，在报文标识符之后
	// 类型: 单字节
	// 含义: 表示发布释放的结果
	// 常见值:
	// - 0x00: 成功 - 消息已释放
	// - 0x92: 报文标识符未找到 - 找不到对应的PUBLISH报文
	// 注意: v3.1.1不支持原因码
	ReasonCode ReasonCode

	// Props 发布释放属性 (v5.0新增)
	// 参考章节: 3.6.2.3 PUBREL Properties
	// 位置: 可变报头，在原因码之后
	// 包含原因字符串、用户属性等
	Props *PubrelProperties
}

func (pkt *PUBREL) Kind() byte {
	return 0x6
}

func (pkt *PUBREL) Pack(w io.Writer) error {
	buf := GetBuffer()
	defer PutBuffer(buf)
	buf.Write(i2b(pkt.PacketID))
	if pkt.Version == VERSION500 {
		buf.WriteByte(pkt.ReasonCode.Code)

		pkt.Props = &PubrelProperties{}
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

func (pkt *PUBREL) Unpack(buf *bytes.Buffer) error {
	pkt.PacketID = binary.BigEndian.Uint16(buf.Next(2))
	if pkt.RemainingLength == 2 {
		return nil
	}
	if pkt.Version == VERSION500 {
		pkt.ReasonCode.Code = buf.Next(1)[0]
		pkt.Props = &PubrelProperties{}
		if err := pkt.Props.Unpack(buf); err != nil {
			return err
		}
	}
	return nil
}

// PubrelProperties 发布释放属性 (v5.0新增)
// 参考章节: 3.6.2.3 PUBREL Properties
// 包含各种发布释放选项，用于扩展释放功能
//
// 版本差异:
// - v3.1.1: 不支持属性系统
// - v5.0: 完整的属性系统，支持原因字符串、用户属性等
type PubrelProperties struct {
	// ReasonString 原因字符串
	// 属性标识符: 31 (0x1F)
	// 参考章节: 3.6.2.3.2 Reason String
	// 类型: UTF-8编码字符串
	// 含义: 表示此次发布释放相关的原因
	// 注意:
	// - 此原因字符串是为诊断而设计的可读字符串，不应该被客户端所解析
	// - 包含多个原因字符串将造成协议错误
	// - 用于提供额外的发布释放信息
	ReasonString string

	// UserProperty 用户属性
	// 属性标识符: 38 (0x26)
	// 参考章节: 3.6.2.3.3 User Property
	// 类型: UTF-8字符串对
	// 含义: 用户定义的名称/值对，可以出现多次
	// 注意:
	// - 用户属性可以出现多次，表示多个名字/值对
	// - 相同的名字可以出现多次
	// - 本规范不做定义，由应用程序确定含义和解释
	// - 可用于传递发布释放相关的额外信息
	UserProperty map[string][]string
}

func (props *PubrelProperties) Pack() ([]byte, error) {
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

func (props *PubrelProperties) Unpack(buf *bytes.Buffer) error {
	propsLen, err := decodeLength(buf)
	if err != nil {
		return err
	}

	for i := uint32(0); i < propsLen; i++ {
		propsId, err := decodeLength(buf)
		if err != nil {
			return err
		}
		switch propsId {
		case 0x1F: // 会话过期间隔 Session Expiry Interval
			props.ReasonString, i = decodeUTF8[string](buf), i+uint32(len(props.UserProperty))
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
