package packet

import (
	"bytes"
	"fmt"
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
//
// 协议约束:
// - [MQTT-3.15.1-1] 固定报头的bits 3,2,1,0必须为0
// - [MQTT-3.15.2-1] 发送方必须使用有效的认证原因码
// - [MQTT-3.15.2-2] 原因字符串不能超过接收方指定的最大报文长度
// - [MQTT-3.15.2-3] 用户属性不能超过接收方指定的最大报文长度
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

// NewAUTH 创建新的AUTH包
// 参考章节: 3.15 AUTH - Authentication exchange
func NewAUTH(version byte, reasonCode ReasonCode) *AUTH {
	return &AUTH{
		FixedHeader: &FixedHeader{
			Kind:            0x0F,
			Dup:             0, // 标志位必须为0
			QoS:             0, // 标志位必须为0
			Retain:          0, // 标志位必须为0
			RemainingLength: 0,
			Version:         version,
		},
		ReasonCode: reasonCode,
		Props:      &AuthProperties{},
	}
}

// isValidAuthReasonCode 检查认证原因码是否有效
// 参考章节: 3.15.2.1 Authentication Reason Code
func isValidAuthReasonCode(code uint8) bool {
	switch code {
	case 0x00, 0x18, 0x19: // Success, Continue authentication, Re-authenticate
		return true
	default:
		return false
	}
}

func (pkt *AUTH) Kind() byte {
	return 0xF
}

func (pkt *AUTH) Packet(w io.Writer) error {
	buf := GetBuffer()
	defer PutBuffer(buf)

	// 写入认证原因码
	buf.WriteByte(pkt.ReasonCode.Code)

	// 写入属性 (仅MQTT v5.0)
	if pkt.Version == VERSION500 {
		propsData, err := pkt.Props.Pack()
		if err != nil {
			return fmt.Errorf("failed to pack AUTH properties: %w", err)
		}

		// 写入属性长度
		propsLen, err := encodeLength(len(propsData))
		if err != nil {
			return fmt.Errorf("failed to encode properties length: %w", err)
		}
		buf.Write(propsLen)

		// 写入属性数据
		buf.Write(propsData)
	}

	// 更新剩余长度
	pkt.FixedHeader.RemainingLength = uint32(buf.Len())

	// 写入固定报头
	if err := pkt.FixedHeader.Pack(w); err != nil {
		return fmt.Errorf("failed to pack AUTH fixed header: %w", err)
	}

	// 写入可变报头和载荷
	_, err := buf.WriteTo(w)
	return err
}

func (pkt *AUTH) Unpack(buf *bytes.Buffer) error {

	// 解析认证原因码
	reasonCodeByte := buf.Next(1)[0]
	pkt.ReasonCode = ReasonCode{Code: reasonCodeByte}

	// 验证原因码
	if !isValidAuthReasonCode(reasonCodeByte) {
		return fmt.Errorf("invalid AUTH reason code: 0x%02X", reasonCodeByte)
	}

	// 解析属性 (仅MQTT v5.0)
	if pkt.Version == VERSION500 {
		pkt.Props = &AuthProperties{}
		if err := pkt.Props.Unpack(buf); err != nil {
			return fmt.Errorf("failed to unpack AUTH properties: %w", err)
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
//
// 协议约束:
// - 认证方法必须且只能出现一次
// - 认证数据必须且只能出现一次
// - 原因字符串不能超过最大报文长度
// - 用户属性不能超过最大报文长度
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
	AuthenticationMethod AuthenticationMethod

	// AuthenticationData 认证数据
	// 属性标识符: 22 (0x16)
	// 参考章节: 3.15.2.2.3 Authentication Data
	// 类型: 二进制数据
	// 含义: 认证数据，内容由认证方法定义
	// 注意:
	// - 没有认证方法却包含了认证数据，或者包含多个认证数据将造成协议错误
	// - 认证数据的内容由认证方法定义
	// - 认证数据可以包含任何二进制信息
	AuthenticationData AuthenticationData

	// ReasonString 原因字符串
	// 属性标识符: 31 (0x1F)
	// 参考章节: 3.15.2.2.4 Reason String
	// 类型: UTF-8编码字符串
	// 含义: 表示此次认证相关的原因
	// 注意:
	// - 此原因字符串是为诊断而设计的可读字符串，不应该被客户端所解析
	// - 包含多个原因字符串将造成协议错误
	// - 用于提供额外的认证信息
	// - [MQTT-3.15.2-2] 不能超过接收方指定的最大报文长度
	ReasonString ReasonString

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
	// - [MQTT-3.15.2-3] 不能超过接收方指定的最大报文长度
	UserProperty UserProperty
}

func (props *AuthProperties) Pack() ([]byte, error) {
	buf := GetBuffer()
	defer PutBuffer(buf)
	if err := props.AuthenticationMethod.Pack(buf); err != nil {
		return nil, err
	}
	if err := props.AuthenticationData.Pack(buf); err != nil {
		return nil, err
	}
	if err := props.ReasonString.Pack(buf); err != nil {
		return nil, err
	}
	if err := props.UserProperty.Pack(buf); err != nil {
		return nil, err
	}
	return bytes.Clone(buf.Bytes()), nil
}

func (props *AuthProperties) Unpack(buf *bytes.Buffer) error {
	// 读取属性长度
	propsLen, err := decodeLength(buf)
	if err != nil {
		return fmt.Errorf("failed to decode properties length: %w", err)
	}
	for i := uint32(0); i < propsLen; {
		propId, err := decodeLength(buf)
		if err != nil {
			return fmt.Errorf("failed to decode property ID: %w", err)
		}
		uLen := uint32(0)
		switch propId {
		case 0x15: // Authentication Method
			if uLen, err = props.AuthenticationMethod.Unpack(buf); err != nil {
				return fmt.Errorf("failed to unpack AuthenticationMethod: %w", err)
			}
		case 0x16: // Authentication Data
			if uLen, err = props.AuthenticationData.Unpack(buf); err != nil {
				return fmt.Errorf("failed to unpack AuthenticationData: %w", err)
			}
		case 0x1F: // Reason String
			if uLen, err = props.ReasonString.Unpack(buf); err != nil {
				return fmt.Errorf("failed to unpack ReasonString: %w", err)
			}
		case 0x26: // User Property
			if uLen, err = props.UserProperty.Unpack(buf); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown AUTH property ID: 0x%02X", propId)
		}
		i += uLen
	}
	return nil
}
