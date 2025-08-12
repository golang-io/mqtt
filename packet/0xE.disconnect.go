package packet

import (
	"bytes"
	"encoding/binary"
	"errors"
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
//
// 协议约束:
// - [MQTT-3.14.0-1] 服务端在发送CONNACK且原因码小于0x80之前，不能发送DISCONNECT
// - [MQTT-3.14.1-1] 客户端或服务端必须验证保留位为0，否则发送原因码0x81的DISCONNECT
// - [MQTT-3.14.2-2] 服务端不能在DISCONNECT中发送会话过期间隔
// - [MQTT-3.14.2-3] 原因字符串不能超过接收方指定的最大报文长度
// - [MQTT-3.14.2-4] 用户属性不能超过接收方指定的最大报文长度
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

// NewDISCONNECT 创建新的DISCONNECT包
// 参考章节: 3.14 DISCONNECT - Disconnect notification
func NewDISCONNECT(version byte, reasonCode ReasonCode) *DISCONNECT {
	return &DISCONNECT{
		FixedHeader: &FixedHeader{
			Kind:            0x0E,
			Dup:             0, // 标志位必须为0
			QoS:             0, // 标志位必须为0
			Retain:          0, // 标志位必须为0
			RemainingLength: 0,
			Version:         version,
		},
		ReasonCode: reasonCode,
		Props:      &DisconnectProperties{},
	}
}

// Validate 验证DISCONNECT包的协议合规性
// 参考章节: 3.14.1 DISCONNECT Fixed Header, 3.14.2 DISCONNECT Variable Header
func (pkt *DISCONNECT) Validate() error {
	// 检查固定报头标志位 [MQTT-3.14.1-1]
	if pkt.Dup != 0 || pkt.QoS != 0 || pkt.Retain != 0 {
		return fmt.Errorf("DISCONNECT packet flags must be 0, got Dup:%d QoS:%d Retain:%d", pkt.Dup, pkt.QoS, pkt.Retain)
	}

	// 检查认证原因码
	if !isValidDisconnectReasonCode(pkt.ReasonCode.Code) {
		return fmt.Errorf("invalid DISCONNECT reason code: 0x%02X", pkt.ReasonCode.Code)
	}

	// 验证属性
	if pkt.Props != nil {
		if err := pkt.Props.Validate(); err != nil {
			return fmt.Errorf("DISCONNECT properties validation failed: %w", err)
		}
	}

	return nil
}

// isValidDisconnectReasonCode 检查断开原因码是否有效
// 参考章节: 3.14.2.1 Disconnect Reason Code
func isValidDisconnectReasonCode(code uint8) bool {
	switch code {
	case 0x00, 0x04, 0x80, 0x81, 0x82, 0x8C, 0x8D, 0x9C, 0x9D:
		return true
	default:
		return false
	}
}

func (pkt *DISCONNECT) Kind() byte {
	return 0xE
}

func (pkt *DISCONNECT) Pack(w io.Writer) error {
	// 验证包的有效性
	if err := pkt.Validate(); err != nil {
		return fmt.Errorf("DISCONNECT packet validation failed: %w", err)
	}

	buf := GetBuffer()
	defer PutBuffer(buf)

	// 写入断开原因码
	buf.WriteByte(pkt.ReasonCode.Code)

	// 写入属性 (仅MQTT v5.0)
	if pkt.Version == VERSION500 && pkt.Props != nil {
		propsData, err := pkt.Props.Pack()
		if err != nil {
			return fmt.Errorf("failed to pack DISCONNECT properties: %w", err)
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
		return fmt.Errorf("failed to pack DISCONNECT fixed header: %w", err)
	}

	// 写入可变报头和载荷
	_, err := buf.WriteTo(w)
	return err
}

func (pkt *DISCONNECT) Unpack(buf *bytes.Buffer) error {
	// 根据 MQTT v5 规范 3.14.2.1:
	// "If the Remaining Length is less than 1 the value of 0x00 (Normal disconnection) is used."

	// 检查缓冲区是否有足够的数据来读取原因码
	if buf.Len() >= 1 {
		// 解析断开原因码
		reasonCodeByte := buf.Next(1)[0]
		pkt.ReasonCode = ReasonCode{Code: reasonCodeByte}

		// 验证原因码 (仅对 MQTT v5.0)
		if pkt.Version == VERSION500 && !isValidDisconnectReasonCode(reasonCodeByte) {
			return fmt.Errorf("invalid DISCONNECT reason code: 0x%02X", reasonCodeByte)
		}
	} else {
		// 缓冲区数据不足，使用默认值 0x00 (Normal disconnection)
		// 这符合 MQTT v5 规范 3.14.2.1 的要求
		pkt.ReasonCode = ReasonCode{Code: 0x00}
	}

	// 解析属性 (仅MQTT v5.0)
	if pkt.Version == VERSION500 {
		// 确保Props字段被初始化
		pkt.Props = &DisconnectProperties{}

		// 如果还有数据，尝试解析属性
		if buf.Len() > 0 {
			if err := pkt.Props.Unpack(buf); err != nil {
				return fmt.Errorf("failed to unpack DISCONNECT properties: %w", err)
			}
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
//
// 协议约束:
// - 会话过期间隔必须且只能出现一次
// - 原因字符串不能超过最大报文长度
// - 用户属性不能超过最大报文长度
// - 服务端引用必须且只能出现一次
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
	// [MQTT-3.14.2-2] 服务端不能在DISCONNECT中发送此属性
	SessionExpiryInterval uint32

	// ReasonString 原因字符串
	// 属性标识符: 31 (0x1F)
	// 参考章节: 3.14.2.2.3 Reason String
	// 类型: UTF-8编码字符串
	// 含义: 表示此次断开相关的原因
	// 注意:
	// - 此原因字符串是为诊断而设计的可读字符串，不应该被客户端所解析
	// - 包含多个原因字符串将造成协议错误
	// - [MQTT-3.14.2-3] 不能超过接收方指定的最大报文长度
	ReasonString string

	// UserProperty 用户属性
	// 属性标识符: 38 (0x26)
	// 参考章节: 3.14.2.2.4 User Property
	// 类型: UTF-8字符串对
	// 含义: 用户定义的名称/值对，可以出现多次
	// 注意:
	// - 用户属性可以出现多次，表示多个名字/值对
	// - 相同的名字可以出现多次
	// - 本规范不做定义，由应用程序确定含义和解释
	// - [MQTT-3.14.2-4] 不能超过接收方指定的最大报文长度
	UserProperty map[string][]string

	// ServerReference 服务端引用
	// 属性标识符: 28 (0x1C)
	// 参考章节: 3.14.2.2.5 Server Reference
	// 类型: UTF-8编码字符串
	// 含义: 客户端可用于识别另一个服务端的字符串
	// 注意:
	// - 包含多个服务端引用将造成协议错误
	// - 服务端发送DISCONNECT时包含服务端引用和原因码0x9C或0x9D
	// - 参考章节4.11 Server Redirection了解服务端引用的使用
	ServerReference string
}

// Validate 验证断开属性的协议合规性
// 参考章节: 3.14.2.2 DISCONNECT Properties
func (props *DisconnectProperties) Validate() error {
	// 验证UTF-8字符串
	if props.ReasonString != "" {
		if !isValidUTF8String(props.ReasonString) {
			return errors.New("reason string contains invalid UTF-8")
		}
	}

	if props.ServerReference != "" {
		if !isValidUTF8String(props.ServerReference) {
			return errors.New("server reference contains invalid UTF-8")
		}
	}

	// 验证用户属性
	if len(props.UserProperty) > 0 {
		for key, values := range props.UserProperty {
			if !isValidUTF8String(key) {
				return fmt.Errorf("user property key contains invalid UTF-8: %s", key)
			}
			for _, value := range values {
				if !isValidUTF8String(value) {
					return fmt.Errorf("user property value contains invalid UTF-8: %s", value)
				}
			}
		}
	}

	return nil
}

func (props *DisconnectProperties) Pack() ([]byte, error) {
	// 验证属性
	if err := props.Validate(); err != nil {
		return nil, fmt.Errorf("properties validation failed: %w", err)
	}

	buf := GetBuffer()
	defer PutBuffer(buf)

	// 写入会话过期间隔 (可选)
	if props.SessionExpiryInterval != 0 {
		buf.WriteByte(0x11)
		buf.Write(i4b(props.SessionExpiryInterval))
	}

	// 写入原因字符串 (可选)
	if props.ReasonString != "" {
		buf.WriteByte(0x1F)
		buf.Write(encodeUTF8(props.ReasonString))
	}

	// 写入用户属性 (可选，可多次)
	if len(props.UserProperty) > 0 {
		for key, values := range props.UserProperty {
			for _, value := range values {
				buf.WriteByte(0x26)
				buf.Write(encodeUTF8(key))
				buf.Write(encodeUTF8(value))
			}
		}
	}

	// 写入服务端引用 (可选)
	if props.ServerReference != "" {
		buf.WriteByte(0x1C)
		buf.Write(encodeUTF8(props.ServerReference))
	}

	return buf.Bytes(), nil
}

func (props *DisconnectProperties) Unpack(buf *bytes.Buffer) error {
	// 读取属性长度
	propsLen, err := decodeLength(buf)
	if err != nil {
		return fmt.Errorf("failed to decode properties length: %w", err)
	}

	// 记录已处理的属性，用于重复性检查（只检查不允许重复的属性）
	processedProps := make(map[uint8]bool)

	// 解析属性
	for i := uint32(0); i < propsLen; {
		// 读取属性标识符 (1字节)
		if buf.Len() < 1 {
			return fmt.Errorf("insufficient data for property ID")
		}
		propID := buf.Next(1)[0]

		// 检查属性是否重复（只检查不允许重复的属性）
		if propID == 0x11 || propID == 0x1F || propID == 0x1C { // Session Expiry Interval, Reason String, Server Reference
			if processedProps[uint8(propID)] {
				return fmt.Errorf("duplicate property ID: 0x%02X", propID)
			}
			processedProps[uint8(propID)] = true
		}

		// 根据属性标识符解析属性值
		switch propID {
		case 0x11: // Session Expiry Interval
			if props.SessionExpiryInterval != 0 {
				return fmt.Errorf("duplicate session expiry interval")
			}
			props.SessionExpiryInterval = binary.BigEndian.Uint32(buf.Next(4))
			i += 4

		case 0x1F: // Reason String
			props.ReasonString, _ = decodeUTF8[string](buf)
			i += uint32(len(props.ReasonString)) + 2 // +2 for property ID and length

		case 0x26: // User Property
			if props.UserProperty == nil {
				props.UserProperty = make(map[string][]string)
			}
			key, _ := decodeUTF8[string](buf)
			value, _ := decodeUTF8[string](buf)
			props.UserProperty[key] = append(props.UserProperty[key], value)
			i += uint32(len(key)+len(value)) + 3 // +3 for property ID and two lengths

		case 0x1C: // Server Reference
			if props.ServerReference != "" {
				return fmt.Errorf("duplicate server reference")
			}
			props.ServerReference, _ = decodeUTF8[string](buf)
			i += uint32(len(props.ServerReference)) + 2 // +2 for property ID and length

		default:
			return fmt.Errorf("unknown DISCONNECT property ID: 0x%02X", propID)
		}
	}

	// 最终验证
	return props.Validate()
}

// String 返回DISCONNECT包的字符串表示
func (pkt *DISCONNECT) String() string {
	if pkt == nil {
		return "DISCONNECT<nil>"
	}

	result := fmt.Sprintf("DISCONNECT{ReasonCode:0x%02X", pkt.ReasonCode.Code)

	if pkt.Props != nil {
		if pkt.Props.SessionExpiryInterval != 0 {
			result += fmt.Sprintf(", SessionExpiry:%d", pkt.Props.SessionExpiryInterval)
		}
		if pkt.Props.ReasonString != "" {
			result += fmt.Sprintf(", Reason:%s", pkt.Props.ReasonString)
		}
		if len(pkt.Props.UserProperty) > 0 {
			result += fmt.Sprintf(", UserProps:%d", len(pkt.Props.UserProperty))
		}
		if pkt.Props.ServerReference != "" {
			result += fmt.Sprintf(", ServerRef:%s", pkt.Props.ServerReference)
		}
	}

	result += "}"
	return result
}
