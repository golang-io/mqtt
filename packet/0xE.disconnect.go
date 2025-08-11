package packet

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// DISCONNECT 断开连接报文
//
// MQTT v3.1.1: 参考章节 3.14 DISCONNECT - Disconnect notification
// MQTT v5.0: 参考章节 3.14 DISCONNECT - Disconnect notification
//
// 报文结构:
// 固定报头: 报文类型0x0E，标志位必须为0
// 可变报头: 断开原因码(v5.0)、断开属性(v5.0)
// 载荷: 无载荷
//
// 版本差异:
// - v3.1.1: 基本的断开连接功能，无载荷
// - v5.0: 在v3.1.1基础上增加了原因码和属性系统，提供更详细的断开信息
//
// 标志位规则:
// - DUP: 必须为0 [MQTT-3.14.1-1]
// - QoS: 必须为0 [MQTT-3.14.1-1]
// - RETAIN: 必须为0 [MQTT-3.14.1-1]
type DISCONNECT struct {
	*FixedHeader `json:"FixedHeader,omitempty"`

	// ReasonCode 断开原因码 (v5.0新增)
	// 参考章节: 3.14.2.1 Disconnect Reason Code
	// 位置: 可变报头第1个字段
	// 类型: 单字节
	// 含义: 表示断开连接的原因
	// 常见值:
	// - 0x00: 正常断开 - 客户端或服务端正常断开连接
	// - 0x04: 断开但保留会话 - 客户端断开但希望保留会话状态
	// - 0x8C: 临时错误 - 临时错误，客户端可以稍后重连
	// - 0x8D: 恶意行为 - 客户端行为不当，服务端断开连接
	// 注意: 如果剩余长度为0，则不包含原因码
	ReasonCode ReasonCode

	// Props 断开属性 (v5.0新增)
	// 参考章节: 3.14.2.2 DISCONNECT Properties
	// 位置: 可变报头，在断开原因码之后
	// 包含会话过期间隔、原因字符串等
	Props *DisconnectProperties
}

func (pkt *DISCONNECT) Kind() byte {
	return 0xE
}

func (pkt *DISCONNECT) Pack(w io.Writer) error {
	buf := GetBuffer()
	defer PutBuffer(buf)

	buf.WriteByte(pkt.ReasonCode.Code)
	if pkt.Version == VERSION500 {
		if pkt.Props == nil {
			pkt.Props = &DisconnectProperties{}
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

func (pkt *DISCONNECT) Unpack(buf *bytes.Buffer) error {
	// 服务端必须验证所有的保留位都被设置为0，如果它们不为0必须断开连接 [MQTT-3.14.1-1]。
	if pkt.Dup != 0 || pkt.QoS != 0 || pkt.Retain != 0 {
		return fmt.Errorf("DISCONNECT: DUP, QoS, Retain must be 0")
	}
	if pkt.RemainingLength == 0 {
		return nil
	}
	pkt.ReasonCode = ReasonCode{Code: buf.Next(1)[0]}
	if pkt.Version == VERSION500 {
		pkt.Props = &DisconnectProperties{}
		if err := pkt.Props.Unpack(buf); err != nil {
			return err
		}
	}

	return nil
}

// DisconnectProperties 断开属性 (v5.0新增)
// 参考章节: 3.14.2.2 DISCONNECT Properties
// 包含各种断开选项，用于扩展断开功能
//
// 版本差异:
// - v3.1.1: 不支持属性系统
// - v5.0: 完整的属性系统，支持会话过期间隔、原因字符串等
type DisconnectProperties struct {
	// SessionExpiryInterval 会话过期间隔
	// 属性标识符: 17 (0x11)
	// 参考章节: 3.14.2.2.2 Session Expiry Interval
	// 类型: 四字节整数，单位: 秒
	// 含义: 网络连接关闭后会话保持的时间
	// 特殊值:
	// - 0: 会话在网络连接关闭时结束
	// - 0xFFFFFFFF: 会话永不过期
	// 注意: 包含多个会话过期间隔将造成协议错误
	SessionExpiryInterval uint32

	// ReasonString 原因字符串
	// 属性标识符: 31 (0x1F)
	// 参考章节: 3.14.2.2.3 Reason String
	// 类型: UTF-8编码字符串
	// 含义: 表示此次断开相关的原因
	// 注意:
	// - 此原因字符串是为诊断而设计的可读字符串，不应该被客户端所解析
	// - 包含多个原因字符串将造成协议错误
	// - 如果加上原因字符串之后的DISCONNECT报文长度超出了客户端指定的最大报文长度，则服务端不能发送此原因字符串
	ReasonString string
}

func (props *DisconnectProperties) Pack() ([]byte, error) {
	buf := GetBuffer()
	defer PutBuffer(buf)
	if props.SessionExpiryInterval != 0 {
		buf.WriteByte(0x11)
		buf.Write(i4b(props.SessionExpiryInterval))
	}
	if props.ReasonString != "" {
		buf.WriteByte(0x1F)
		buf.Write(encodeUTF8(props.ReasonString))
	}
	return buf.Bytes(), nil
}
func (props *DisconnectProperties) Unpack(buf *bytes.Buffer) error {
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
		case 0x11:
			if props.SessionExpiryInterval != 0 {
				return ErrProtocolErr
			}
			props.SessionExpiryInterval, i = binary.BigEndian.Uint32(buf.Next(4)), i+4
		case 0x1F:
			props.ReasonString, i = decodeUTF8[string](buf), i+uint32(len(props.ReasonString))
		}
	}
	return nil
}
