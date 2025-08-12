package packet

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// SessionExpiryInterval 会话过期间隔 (0x11)
type SessionExpiryInterval uint32

func (s SessionExpiryInterval) Pack(buf *bytes.Buffer) error {
	if s == 0 {
		return nil
	}
	buf.WriteByte(0x11)
	buf.Write(i4b(uint32(s)))
	return nil
}

func (s *SessionExpiryInterval) Unpack(buf *bytes.Buffer) (uint32, error) {
	interval := binary.BigEndian.Uint32(buf.Next(4))
	*s = SessionExpiryInterval(interval)
	return uint32(4), nil
}

// 添加类型转换方法以保持兼容性
func (s SessionExpiryInterval) Uint32() uint32 {
	return uint32(s)
}

// ReceiveMaximum 接收最大值 (0x21)
type ReceiveMaximum uint16

func (s ReceiveMaximum) Pack(buf *bytes.Buffer) error {
	if s == 0 {
		return nil
	}
	buf.WriteByte(0x21)
	buf.Write(i2b(uint16(s)))
	return nil
}

func (s *ReceiveMaximum) Unpack(buf *bytes.Buffer) (uint32, error) {
	interval := binary.BigEndian.Uint16(buf.Next(2))
	*s = ReceiveMaximum(interval)
	return uint32(2), nil
}

// 添加类型转换方法以保持兼容性
func (s ReceiveMaximum) Uint16() uint16 {
	return uint16(s)
}

// MaximumPacketSize 最大数据包大小 (0x27)
type MaximumPacketSize uint32

func (s MaximumPacketSize) Pack(buf *bytes.Buffer) error {
	if s == 0 {
		return nil
	}
	buf.WriteByte(0x27)
	buf.Write(i4b(uint32(s)))
	return nil
}

func (s *MaximumPacketSize) Unpack(buf *bytes.Buffer) (uint32, error) {
	interval := binary.BigEndian.Uint32(buf.Next(4))
	*s = MaximumPacketSize(interval)
	return uint32(4), nil // 修复：返回值应该是 4 而不是 2
}

// 添加类型转换方法以保持兼容性
func (s MaximumPacketSize) Uint32() uint32 {
	return uint32(s)
}

// TopicAliasMaximum 主题别名最大值 (0x22)
type TopicAliasMaximum uint16

func (s TopicAliasMaximum) Pack(buf *bytes.Buffer) error {
	if s == 0 {
		return nil
	}
	buf.WriteByte(0x22)
	buf.Write(i2b(uint16(s)))
	return nil
}

func (s *TopicAliasMaximum) Unpack(buf *bytes.Buffer) (uint32, error) {
	interval := binary.BigEndian.Uint16(buf.Next(2))
	*s = TopicAliasMaximum(interval)
	return uint32(2), nil
}

// 添加类型转换方法以保持兼容性
func (s TopicAliasMaximum) Uint16() uint16 {
	return uint16(s)
}

// RequestResponseInformation 请求响应信息 (0x19)
type RequestResponseInformation uint8

func (s RequestResponseInformation) Pack(buf *bytes.Buffer) error {
	if s == 0 {
		return nil
	}
	buf.WriteByte(0x19)
	buf.WriteByte(uint8(s))
	return nil
}

func (s *RequestResponseInformation) Unpack(buf *bytes.Buffer) (uint32, error) {
	value := buf.Next(1)
	if len(value) != 1 {
		return 0, fmt.Errorf("%w: invalid request response information", ErrProtocolErr)
	}
	*s = RequestResponseInformation(value[0])
	return uint32(1), nil
}

// 添加类型转换方法以保持兼容性
func (s RequestResponseInformation) Uint8() uint8 {
	return uint8(s)
}

// RequestProblemInformation 请求问题信息 (0x17)
type RequestProblemInformation uint8

func (s RequestProblemInformation) Pack(buf *bytes.Buffer) error {
	if s == 0 {
		return nil
	}

	buf.WriteByte(0x17)
	buf.WriteByte(uint8(s))
	return nil
}

func (s *RequestProblemInformation) Unpack(buf *bytes.Buffer) (uint32, error) {
	value := buf.Next(1)
	if len(value) != 1 {
		return 0, fmt.Errorf("%w: invalid request problem information", ErrProtocolErr)
	}
	*s = RequestProblemInformation(value[0])
	if *s != 0 && *s != 1 {
		return 0, fmt.Errorf("%w: invalid request problem information", ErrProtocolErr)
	}
	return uint32(1), nil
}

// 添加类型转换方法以保持兼容性
func (s RequestProblemInformation) Uint8() uint8 {
	return uint8(s)
}

// UserProperty 用户属性 (0x26)
type UserProperty struct {
	Name  string
	Value string
}

func (s UserProperty) Pack(buf *bytes.Buffer) error {
	if s.Name == "" || s.Value == "" {
		return nil
	}
	buf.WriteByte(0x26)
	// 写入名称长度和名称
	nameBytes := []byte(s.Name)
	buf.Write(i2b(uint16(len(nameBytes))))
	buf.Write(nameBytes)
	// 写入值长度和值
	valueBytes := []byte(s.Value)
	buf.Write(i2b(uint16(len(valueBytes))))
	buf.Write(valueBytes)
	return nil
}

func (s *UserProperty) Unpack(buf *bytes.Buffer) (uint32, error) {
	propsLen, uLen := uint32(4), uint32(0) // 名称长度(2) + 值长度(2)
	s.Name, uLen = decodeUTF8[string](buf)
	propsLen += uLen //  名称
	s.Value, uLen = decodeUTF8[string](buf)
	propsLen += uLen //  值
	return propsLen, nil
}

// AuthenticationMethod 认证方法 (0x15)
type AuthenticationMethod string

func (s *AuthenticationMethod) Pack(buf *bytes.Buffer) error {
	if s == nil || *s == "" {
		return nil
	}
	buf.WriteByte(0x15)
	buf.Write(encodeUTF8(*s))
	return nil
}

func (s *AuthenticationMethod) Unpack(buf *bytes.Buffer) (uint32, error) {
	method, num := decodeUTF8[string](buf)
	*s = AuthenticationMethod(method)
	return num, nil
}

// 添加类型转换方法以保持兼容性
func (s AuthenticationMethod) String() string {
	return string(s)
}

// AuthenticationData 认证数据 (0x16)
type AuthenticationData []byte

func (s *AuthenticationData) Pack(buf *bytes.Buffer) error {
	buf.WriteByte(0x16)
	buf.Write(encodeUTF8(*s))
	return nil
}

func (s *AuthenticationData) Unpack(buf *bytes.Buffer) (uint32, error) {
	authenticationData, num := decodeUTF8[[]byte](buf)
	*s = AuthenticationData(authenticationData)
	return num, nil
}

// 添加类型转换方法以保持兼容性
func (s AuthenticationData) Bytes() []byte {
	return []byte(s)
}

// MaximumQoS 最大服务质量 (0x24)
type MaximumQoS uint8

func (s *MaximumQoS) Pack(buf *bytes.Buffer) error {
	buf.WriteByte(0x24)
	buf.WriteByte(uint8(*s))
	return nil
}

func (s *MaximumQoS) Unpack(buf *bytes.Buffer) (uint32, error) {
	value := buf.Next(1)
	if len(value) != 1 {
		return 0, fmt.Errorf("%w: invalid maximum qos", ErrProtocolErr)
	}
	*s = MaximumQoS(value[0])
	return uint32(1), nil
}

// 添加类型转换方法以保持兼容性
func (s MaximumQoS) Uint8() uint8 {
	return uint8(s)
}

// RetainAvailable 保留消息可用性 (0x25)
type RetainAvailable uint8

func (s *RetainAvailable) Pack(buf *bytes.Buffer) error {
	buf.WriteByte(0x25)
	buf.WriteByte(uint8(*s))
	return nil
}

func (s *RetainAvailable) Unpack(buf *bytes.Buffer) (uint32, error) {
	value := buf.Next(1)
	if len(value) != 1 {
		return 0, fmt.Errorf("%w: invalid retain available", ErrProtocolErr)
	}
	*s = RetainAvailable(value[0])
	return uint32(1), nil
}

// 添加类型转换方法以保持兼容性
func (s RetainAvailable) Uint8() uint8 {
	return uint8(s)
}

// AssignedClientIdentifier 分配的客户端标识符 (0x12)
type AssignedClientIdentifier string

func (s *AssignedClientIdentifier) Pack(buf *bytes.Buffer) error {
	buf.WriteByte(0x12)
	buf.Write(encodeUTF8(*s))
	return nil
}

func (s *AssignedClientIdentifier) Unpack(buf *bytes.Buffer) (uint32, error) {
	identifier, num := decodeUTF8[string](buf)
	*s = AssignedClientIdentifier(identifier)
	return num, nil
}

// 添加类型转换方法以保持兼容性
func (s AssignedClientIdentifier) String() string {
	return string(s)
}

// ReasonString 原因字符串 (0x1F)
type ReasonString string

func (s *ReasonString) Pack(buf *bytes.Buffer) error {
	if s == nil || *s == "" {
		return nil
	}
	buf.WriteByte(0x1F)
	buf.Write(encodeUTF8(*s))
	return nil
}

func (s *ReasonString) Unpack(buf *bytes.Buffer) (uint32, error) {
	reason, num := decodeUTF8[string](buf)
	*s = ReasonString(reason)
	return num, nil
}

// 添加类型转换方法以保持兼容性
func (s ReasonString) String() string {
	return string(s)
}

// WildcardSubscriptionAvailable 通配符订阅可用性 (0x28)
type WildcardSubscriptionAvailable uint8

func (s *WildcardSubscriptionAvailable) Pack(buf *bytes.Buffer) error {
	buf.WriteByte(0x28)
	buf.WriteByte(uint8(*s))
	return nil
}

func (s *WildcardSubscriptionAvailable) Unpack(buf *bytes.Buffer) (uint32, error) {
	value := buf.Next(1)
	if len(value) != 1 {
		return 0, fmt.Errorf("%w: invalid wildcard subscription available", ErrProtocolErr)
	}
	*s = WildcardSubscriptionAvailable(value[0])
	return uint32(1), nil
}

// 添加类型转换方法以保持兼容性
func (s WildcardSubscriptionAvailable) Uint8() uint8 {
	return uint8(s)
}

// SubscriptionIdentifiersAvailable 订阅标识符可用性 (0x29)
type SubscriptionIdentifiersAvailable uint8

func (s *SubscriptionIdentifiersAvailable) Pack(buf *bytes.Buffer) error {
	buf.WriteByte(0x29)
	buf.WriteByte(uint8(*s))
	return nil
}

func (s *SubscriptionIdentifiersAvailable) Unpack(buf *bytes.Buffer) (uint32, error) {
	value := buf.Next(1)
	if len(value) != 1 {
		return 0, fmt.Errorf("%w: invalid subscription identifiers available", ErrProtocolErr)
	}
	*s = SubscriptionIdentifiersAvailable(value[0])
	return uint32(1), nil
}

// 添加类型转换方法以保持兼容性
func (s SubscriptionIdentifiersAvailable) Uint8() uint8 {
	return uint8(s)
}

// SharedSubscriptionAvailable 共享订阅可用性 (0x2A)
type SharedSubscriptionAvailable uint8

func (s *SharedSubscriptionAvailable) Pack(buf *bytes.Buffer) error {
	buf.WriteByte(0x2A)
	buf.WriteByte(uint8(*s))
	return nil
}

func (s *SharedSubscriptionAvailable) Unpack(buf *bytes.Buffer) (uint32, error) {
	value := buf.Next(1)
	if len(value) != 1 {
		return 0, fmt.Errorf("%w: invalid shared subscription available", ErrProtocolErr)
	}
	*s = SharedSubscriptionAvailable(value[0])
	return uint32(1), nil
}

// 添加类型转换方法以保持兼容性
func (s SharedSubscriptionAvailable) Uint8() uint8 {
	return uint8(s)
}

// ServerKeepAlive 服务器保活时间 (0x13)
type ServerKeepAlive uint16

func (s *ServerKeepAlive) Pack(buf *bytes.Buffer) error {
	buf.WriteByte(0x13)
	buf.Write(i2b(uint16(*s)))
	return nil
}

func (s *ServerKeepAlive) Unpack(buf *bytes.Buffer) (uint32, error) {
	keepAlive := binary.BigEndian.Uint16(buf.Next(2))
	*s = ServerKeepAlive(keepAlive)
	return uint32(2), nil
}

// 添加类型转换方法以保持兼容性
func (s ServerKeepAlive) Uint16() uint16 {
	return uint16(s)
}

// ResponseInformation 响应信息 (0x1A)
type ResponseInformation string

func (s *ResponseInformation) Pack(buf *bytes.Buffer) error {
	buf.WriteByte(0x1A)
	buf.Write(encodeUTF8(*s))
	return nil
}

func (s *ResponseInformation) Unpack(buf *bytes.Buffer) (uint32, error) {
	response, num := decodeUTF8[string](buf)
	*s = ResponseInformation(response)
	return num, nil
}

// 添加类型转换方法以保持兼容性
func (s ResponseInformation) String() string {
	return string(s)
}

// ServerReference 服务器引用 (0x1C)
type ServerReference string

func (s *ServerReference) Pack(buf *bytes.Buffer) error {
	buf.WriteByte(0x1C)
	buf.Write(encodeUTF8(*s))
	return nil
}

func (s *ServerReference) Unpack(buf *bytes.Buffer) (uint32, error) {
	reference, num := decodeUTF8[string](buf)
	*s = ServerReference(reference)
	return num, nil
}

// 添加类型转换方法以保持兼容性
func (s ServerReference) String() string {
	return string(s)
}

// PayloadFormatIndicator 载荷格式指示 (0x01)
type PayloadFormatIndicator uint8

func (s *PayloadFormatIndicator) Pack(buf *bytes.Buffer) error {
	if s == nil || *s == 0 {
		return nil
	}
	buf.WriteByte(0x01)
	buf.WriteByte(uint8(*s))
	return nil
}

func (s *PayloadFormatIndicator) Unpack(buf *bytes.Buffer) (uint32, error) {
	value := buf.Next(1)
	if len(value) != 1 {
		return 0, fmt.Errorf("%w: invalid payload format indicator", ErrProtocolErr)
	}
	*s = PayloadFormatIndicator(value[0])
	return uint32(1), nil
}

// 添加类型转换方法以保持兼容性

type MessageExpiryInterval uint32

func (s *MessageExpiryInterval) Pack(buf *bytes.Buffer) error {
	if s == nil || *s == 0 {
		return nil
	}
	buf.WriteByte(0x02)
	buf.Write(i4b(uint32(*s)))
	return nil
}

func (s *MessageExpiryInterval) Unpack(buf *bytes.Buffer) (uint32, error) {
	interval := binary.BigEndian.Uint32(buf.Next(4))
	*s = MessageExpiryInterval(interval)
	return uint32(4), nil
}

// 添加类型转换方法以保持兼容性
func (s MessageExpiryInterval) Uint32() uint32 {
	return uint32(s)
}

// TopicAlias 主题别名 (0x23)
type TopicAlias uint16

func (s *TopicAlias) Pack(buf *bytes.Buffer) error {
	if s == nil || *s == 0 {
		return nil
	}
	buf.WriteByte(0x23)
	buf.Write(i2b(uint16(*s)))
	return nil
}

func (s *TopicAlias) Unpack(buf *bytes.Buffer) (uint32, error) {
	alias := binary.BigEndian.Uint16(buf.Next(2))
	*s = TopicAlias(alias)
	return uint32(2), nil
}

// 添加类型转换方法以保持兼容性
func (s TopicAlias) Uint16() uint16 {
	return uint16(s)
}

type CorrelationData []byte

func (s *CorrelationData) Pack(buf *bytes.Buffer) error {
	if s == nil || len(*s) == 0 {
		return nil
	}
	buf.WriteByte(0x09)
	buf.Write(encodeUTF8(*s))
	return nil
}

func (s *CorrelationData) Unpack(buf *bytes.Buffer) (uint32, error) {
	correlationData, num := decodeUTF8[[]byte](buf)
	*s = CorrelationData(correlationData)
	return num, nil
}

// 添加类型转换方法以保持兼容性
func (s CorrelationData) Bytes() []byte {
	return []byte(s)
}

type ContentType string

func (s ContentType) Pack(buf *bytes.Buffer) error {
	if s == "" {
		return nil
	}
	buf.WriteByte(0x03)
	buf.Write(encodeUTF8(s))
	return nil
}

func (s *ContentType) Unpack(buf *bytes.Buffer) (uint32, error) {
	reason, num := decodeUTF8[string](buf)
	*s = ContentType(reason)
	return num, nil
}

// 添加类型转换方法以保持兼容性
func (s ContentType) String() string {
	return string(s)
}

type SubscriptionIdentifier uint32

func (s SubscriptionIdentifier) Pack(buf *bytes.Buffer) error {
	if s == 0 {
		return nil
	}
	buf.WriteByte(0x0B)
	buf.Write(i4b(uint32(s)))
	return nil
}

func (s *SubscriptionIdentifier) Unpack(buf *bytes.Buffer) (uint32, error) {
	identifier, err := decodeLength(buf)
	if err != nil {
		return 0, err
	}
	*s = SubscriptionIdentifier(identifier)
	return uint32(identifier), nil
}

// 添加类型转换方法以保持兼容性
func (s SubscriptionIdentifier) Uint32() uint32 {
	return uint32(s)
}
