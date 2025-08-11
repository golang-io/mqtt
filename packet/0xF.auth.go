package packet

import (
	"bytes"
	"io"
)

// AUTH 认证交换报文 (MQTT v5.0新增)
//
// MQTT v3.1.1: 不支持此报文类型
// MQTT v5.0: 参考章节 3.15 AUTH - Authentication exchange
//
// 报文结构:
// 固定报头: 报文类型0x0F，标志位必须为0
// 可变报头: 认证原因码、认证属性
// 载荷: 无载荷
//
// 版本差异:
// - v3.1.1: 不支持认证交换报文
// - v5.0: 新增的报文类型，用于扩展认证流程
//
// 用途:
// - 用于客户端和服务端之间的认证交换
// - 支持多轮认证过程
// - 可以在连接建立后继续认证
//
// 标志位规则:
// - DUP: 必须为0
// - QoS: 必须为0
// - RETAIN: 必须为0
type AUTH struct {
	*FixedHeader

	// ReasonCode 认证原因码
	// 参考章节: 3.15.2.1 Authentication Reason Code
	// 位置: 可变报头第1个字段
	// 类型: 单字节
	// 含义: 表示认证交换的原因
	// 常见值:
	// - 0x00: 成功 - 认证成功
	// - 0x18: 继续认证 - 需要继续认证过程
	// - 0x19: 重新认证 - 需要重新认证
	// 注意: 认证原因码是必需的字段
	ReasonCode ReasonCode

	// Props 认证属性
	// 参考章节: 3.15.2.2 AUTH Properties
	// 位置: 可变报头，在认证原因码之后
	// 包含认证方法、认证数据、原因字符串等
	Props *AuthProperties
}

func (pkt *AUTH) Kind() byte {
	return 0xF
}

func (pkt *AUTH) Packet(w io.Writer) error {
	buf := GetBuffer()
	defer PutBuffer(buf)

	buf.WriteByte(pkt.ReasonCode.Code)
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

func (pkt *AUTH) Unpack(buf *bytes.Buffer) error {
	pkt.ReasonCode = ReasonCode{Code: buf.Next(1)[0]}
	if pkt.Version == VERSION500 {
		pkt.Props = &AuthProperties{}
		if err := pkt.Props.Unpack(buf); err != nil {
			return err
		}
	}
	return nil
}

// AuthProperties 认证属性
// 参考章节: 3.15.2.2 AUTH Properties
// 包含各种认证选项，用于扩展认证功能
//
// 版本差异:
// - v3.1.1: 不支持属性系统
// - v5.0: 完整的属性系统，支持认证方法、认证数据等
type AuthProperties struct {
	// AuthenticationMethod 认证方法
	// 属性标识符: 21 (0x15)
	// 参考章节: 3.15.2.2.2 Authentication Method
	// 类型: UTF-8编码字符串
	// 含义: 扩展认证的认证方法名称
	// 注意:
	// - 包含多个认证方法将造成协议错误
	// - 如果没有认证方法，则不进行扩展验证
	// - 认证方法名称由应用程序定义
	AuthenticationMethod string

	// AuthenticationData 认证数据
	// 属性标识符: 22 (0x16)
	// 参考章节: 3.15.2.2.3 Authentication Data
	// 类型: 二进制数据
	// 含义: 认证数据，内容由认证方法定义
	// 注意:
	// - 没有认证方法却包含了认证数据，或者包含多个认证数据将造成协议错误
	// - 认证数据的内容由认证方法定义
	// - 认证数据可以包含任何二进制信息
	AuthenticationData []byte

	// ReasonString 原因字符串
	// 属性标识符: 31 (0x1F)
	// 参考章节: 3.15.2.2.4 Reason String
	// 类型: UTF-8编码字符串
	// 含义: 表示此次认证相关的原因
	// 注意:
	// - 此原因字符串是为诊断而设计的可读字符串，不应该被客户端所解析
	// - 包含多个原因字符串将造成协议错误
	// - 用于提供额外的认证信息
	ReasonString string

	// UserProperty 用户属性
	// 属性标识符: 38 (0x26)
	// 参考章节: 3.15.2.2.5 User Property
	// 类型: UTF-8字符串对
	// 含义: 用户定义的名称/值对，可以出现多次
	// 注意:
	// - 用户属性可以出现多次，表示多个名字/值对
	// - 相同的名字可以出现多次
	// - 本规范不做定义，由应用程序确定含义和解释
	// - 可用于传递认证相关的额外信息
	UserProperty map[string][]string
}

func (props *AuthProperties) Pack() ([]byte, error) {
	buf := GetBuffer()
	defer PutBuffer(buf)
	buf.WriteByte(0x15)
	buf.Write(encodeUTF8(props.AuthenticationMethod))

	if props.AuthenticationData != nil {
		buf.WriteByte(0x16)
		buf.Write(encodeUTF8(props.AuthenticationData))
	}

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

func (props *AuthProperties) Unpack(buf *bytes.Buffer) error {
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
		case 0x15:
			props.AuthenticationMethod, i = decodeUTF8[string](buf), i+uint32(len(props.AuthenticationMethod))
		case 0x16:
			props.AuthenticationData, i = decodeUTF8[[]byte](buf), i+uint32(len(props.AuthenticationData))
		case 0x1F:
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
