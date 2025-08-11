package packet

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// CONNACK 连接确认报文
//
// MQTT v3.1.1: 参考章节 3.2 CONNACK - Acknowledge connection request
// MQTT v5.0: 参考章节 3.2 CONNACK - Acknowledge connection request
//
// 报文结构:
// 固定报头: 报文类型0x02，标志位必须为0
// 可变报头: 连接确认标志、连接返回码
// 载荷: 无载荷
//
// 版本差异:
// - v3.1.1: 基本的连接确认功能，包含连接返回码
// - v5.0: 在v3.1.1基础上增加了属性系统，支持更详细的连接状态反馈
type CONNACK struct {
	*FixedHeader

	// 可变报头部分
	// 参考章节: 3.2.2 Variable header

	// SessionPresent 会话存在标志
	// 位置: 可变报头第1字节的bit 0
	// 参考章节: 3.2.2.1 Session Present
	// 值:
	// - 0: 服务端没有客户端的会话状态
	// - 1: 服务端有客户端的会话状态
	// 注意:
	// - 只有在CleanSession=0时才有意义
	// - bits 7-6为保留位，必须为0
	SessionPresent uint8

	// ConnectReturnCode 连接返回码
	// 位置: 可变报头第2字节
	// 参考章节: 3.2.2.2 Connect Return code
	// 含义: 表示连接请求的处理结果
	// 值:
	// - 0x00: 连接已接受 - 连接已被服务端接受
	// - 0x01: 连接已拒绝，不支持的协议版本 - 服务端不支持客户端请求的MQTT协议级别
	// - 0x02: 连接已拒绝，不合格的客户端标识符 - 客户端标识符是正确的UTF-8编码，但服务端不允许使用
	// - 0x03: 连接已拒绝，服务端不可用 - 网络连接已建立，但MQTT服务不可用
	// - 0x04: 连接已拒绝，无效的用户名或密码 - 用户名或密码的数据格式无效
	// - 0x05: 连接已拒绝，未授权 - 客户端未被授权连接到此服务端
	// 注意:
	// - 如果服务端发送了一个包含非零返回码的CONNACK报文，那么它必须关闭网络连接 [MQTT-3.2.2-5]
	// - 如果认为上表中的所有连接返回码都不太合适，那么服务端必须关闭网络连接，不需要发送CONNACK报文 [MQTT-3.2.2-6]
	ConnectReturnCode ReasonCode `json:"ConnectReturnCode,omitempty"`

	// Props 连接确认属性 (v5.0新增)
	// 位置: 可变报头，在连接返回码之后
	// 参考章节: 3.2.2.3 CONNACK Properties
	// 包含各种连接确认信息，如会话过期间隔、接收最大值等
	Props *ConnackProps
}

func (pkt *CONNACK) Kind() byte {
	return 0x2
}

func (pkt *CONNACK) String() string {
	return fmt.Sprintf("[0x2]ConnectReturnCode=%d", pkt.ConnectReturnCode.Code)
}

// Pack 将CONNACK报文序列化到写入器
// 参考章节: 3.2 CONNACK - Acknowledge connection request
// 序列化顺序:
// 1. 固定报头
// 2. 可变报头: 会话存在标志、连接返回码
// 3. 属性(v5.0): 连接确认属性
func (pkt *CONNACK) Pack(w io.Writer) error {
	buf := GetBuffer()
	defer PutBuffer(buf)

	// 写入会话存在标志
	// 参考章节: 3.2.2.1 Session Present
	buf.WriteByte(pkt.SessionPresent)

	// 写入连接返回码
	// 参考章节: 3.2.2.2 Connect Return code
	buf.WriteByte(pkt.ConnectReturnCode.Code)

	// v5.0: 写入连接确认属性
	if pkt.Version == VERSION500 {
		if pkt.Props == nil {
			pkt.Props = &ConnackProps{}
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

// Unpack 从缓冲区解析CONNACK报文
// 参考章节: 3.2 CONNACK - Acknowledge connection request
// 解析顺序:
// 1. 会话存在标志
// 2. 连接返回码
// 3. 属性(v5.0): 连接确认属性
func (pkt *CONNACK) Unpack(buf *bytes.Buffer) error {
	// 解析会话存在标志
	// 参考章节: 3.2.2.1 Session Present
	pkt.SessionPresent = buf.Next(1)[0]

	// 解析连接返回码
	// 参考章节: 3.2.2.2 Connect Return code
	pkt.ConnectReturnCode = ReasonCode{Code: buf.Next(1)[0]}

	// v5.0: 解析连接确认属性
	if pkt.Version == VERSION500 {
		pkt.Props = &ConnackProps{}
		if err := pkt.Props.Unpack(buf); err != nil {
			return err
		}
	}
	return nil
}

// ConnackProps CONNACK报文可变报头中的属性
// MQTT v5.0新增，参考章节: 3.2.2.3 CONNACK Properties
// 位置: 可变报头，在连接返回码之后
// 编码: 属性长度 + 属性标识符 + 属性值
// 注意: 包含多个相同属性将造成协议错误
type ConnackProps struct {
	// SessionExpiryInterval 会话过期间隔
	// 属性标识符: 17 (0x11)
	// 参考章节: 3.2.2.3.2 Session Expiry Interval
	// 类型: 四字节整数，单位: 秒
	// 含义: 服务端使用的会话过期时间间隔
	// 注意:
	// - 包含多个会话过期间隔将造成协议错误
	// - 如果会话过期间隔值未指定，则使用CONNECT报文中指定的会话过期时间间隔
	// - 服务端使用此属性通知客户端它使用的会话过期时间间隔与客户端在CONNECT中发送的值不同
	// - 更详细的关于会话过期时间的描述，请参考3.1.2.11.2节
	SessionExpiryInterval uint32

	// ReceiveMaximum 接收最大值
	// 属性标识符: 33 (0x21)
	// 参考章节: 3.2.2.3.3 Receive Maximum
	// 类型: 双字节整数
	// 含义: 服务端愿意同时处理的QoS等级1和2的发布消息最大数量
	// 默认值: 65535
	// 注意:
	// - 包含多个接收最大值或接收最大值为0将造成协议错误
	// - 没有机制可以限制客户端试图发送的QoS为0的发布消息
	// - 如果没有设置最大接收值，将使用默认值65535
	// - 关于接收最大值的详细使用，参考4.9节流控部分
	ReceiveMaximum uint16

	// MaximumQoS 最大服务质量
	// 属性标识符: 36 (0x24)
	// 参考章节: 3.2.2.3.4 Maximum QoS
	// 类型: 单字节，值: 0或1
	// 含义: 服务端支持的最大QoS等级
	// 默认值: 2 (如果未设置)
	// 注意:
	// - 包含多个最大服务质量或最大服务质量既不为0也不为1将造成协议错误
	// - 如果服务端不支持QoS为1或2的PUBLISH报文，服务端必须在CONNACK报文中发送最大服务质量以指定其支持的最大QoS值 [MQTT-3.2.2-9]
	// - 即使不支持QoS为1或2的PUBLISH报文，服务端也必须接受请求QoS为0、1或2的SUBSCRIBE报文 [MQTT-3.2.2-10]
	// - 如果从服务端接收到了最大QoS等级，则客户端不能发送超过最大QoS等级所指定的QoS等级的PUBLISH报文 [MQTT-3.2.2-11]
	// - 服务端接收到超过其指定的最大服务质量的PUBLISH报文将造成协议错误
	// - 如果服务端收到包含遗嘱的QoS超过服务端处理能力的CONNECT报文，服务端必须拒绝此连接
	MaximumQoS uint8

	// RetainAvailable 保留可用
	// 属性标识符: 37 (0x25)
	// 参考章节: 3.2.2.3.5 Retain Available
	// 类型: 单字节，值: 0或1
	// 含义: 服务端是否支持保留消息
	// 默认值: 1 (支持保留消息，如果未设置)
	// 注意:
	// - 包含多个保留可用字段或保留可用字段值不为0也不为1将造成协议错误
	// - 如果服务端收到一个包含保留标志位1的遗嘱消息的CONNECT报文且服务端不支持保留消息，服务端必须拒绝此连接请求
	// - 从服务端接收到的保留可用标志为0时，客户端不能发送保留标志设置为1的PUBLISH报文 [MQTT-3.2.2-14]
	RetainAvailable uint8

	// MaximumPacketSize 最大报文长度
	// 属性标识符: 39 (0x27)
	// 参考章节: 3.2.2.3.6 Maximum Packet Size
	// 类型: 四字节整数
	// 含义: 服务端愿意接收的最大报文长度
	// 注意:
	// - 包含多个最大报文长度，或最大报文长度为0将造成协议错误
	// - 如果没有设置，则按照协议由固定报头中的剩余长度可编码最大值和协议报头对数据包的大小做限制
	// - 最大报文长度是MQTT控制报文的总长度
	// - 客户端不能发送超过最大报文长度的报文给服务端 [MQTT-3.2.2-15]
	// - 收到长度超过限制的报文将导致协议错误
	MaximumPacketSize uint32

	// AssignedClientID 分配客户标识符
	// 属性标识符: 18 (0x12)
	// 参考章节: 3.2.2.3.7 Assigned Client Identifier
	// 类型: UTF-8编码字符串
	// 含义: 服务端为客户端分配的客户标识符
	// 注意:
	// - 包含多个分配客户标识符将造成协议错误
	// - 服务端分配客户标识符的原因是CONNECT报文中的客户标识符长度为0
	// - 如果客户端使用长度为0的客户标识符，服务端必须回复包含分配客户标识符的CONNACK报文
	// - 分配客户标识符必须是没有被服务端的其他会话所使用的新客户标识符 [MQTT-3.2.2-16]
	AssignedClientID string

	// TopicAliasMaximum 主题别名最大值
	// 属性标识符: 34 (0x22)
	// 参考章节: 3.2.2.3.8 Topic Alias Maximum
	// 类型: 双字节整数
	// 含义: 服务端能够接收的来自客户端的主题别名最大值
	// 默认值: 0 (如果未设置)
	// 注意:
	// - 包含多个主题别名最大值将造成协议错误
	// - 没有设置的情况下，主题别名最大值默认为零
	// - 此值指示了服务端能够接收的来自客户端的主题别名最大值
	// - 客户端在一个PUBLISH报文中发送的主题别名值不能超过服务端设置的主题别名最大值 [MQTT-3.2.2-17]
	// - 值为0表示本次连接服务端不接受任何主题别名
	// - 如果主题别名最大值没有设置，或者设置为0，则客户端不能向此服务端发送任何主题别名 [MQTT-3.2.2-18]
	TopicAliasMaximum uint16

	// ReasonString 原因字符串
	// 属性标识符: 31 (0x1F)
	// 参考章节: 3.2.2.3.9 Reason String
	// 类型: UTF-8编码字符串
	// 含义: 表示此次响应相关的原因
	// 注意:
	// - 此原因字符串是为诊断而设计的可读字符串，不应该被客户端所解析
	// - 服务端使用此值向客户端提供附加信息
	// - 如果加上原因字符串之后的CONNACK报文长度超出了客户端指定的最大报文长度，则服务端不能发送此原因字符串 [MQTT-3.2.2-19]
	// - 包含多个原因字符串将造成协议错误
	// 非规范评注:
	// - 客户端对原因字符串的恰当使用包括：抛出异常时使用此字符串，或者将此字符串写入日志
	ReasonString string

	// UserProperty 用户属性
	// 属性标识符: 38 (0x26)
	// 参考章节: 3.2.2.3.10 User Property
	// 类型: UTF-8字符串对
	// 含义: 用户定义的名称/值对，可以出现多次
	// 注意:
	// - 此属性可用于向客户端提供包括诊断信息在内的附加信息
	// - 如果加上用户属性之后的CONNACK报文长度超出了客户端指定的最大报文长度，则服务端不能发送此属性 [MQTT-3.2.2-20]
	// - 用户属性允许出现多次，以表示多个名字/值对，且相同的名字可以多次出现
	// - 用户属性的内容和意义本规范不做定义
	// - CONNACK报文的接收端可以选择忽略此属性
	UserProperty map[string][]string

	// WildcardSubscriptionAvailable 通配符订阅可用
	// 属性标识符: 40 (0x28)
	// 参考章节: 3.2.2.3.11 Wildcard Subscription Available
	// 类型: 单字节，值: 0或1
	// 含义: 服务端是否支持通配符订阅
	// 默认值: 1 (支持通配符订阅，如果未设置)
	// 注意:
	// - 包含多个通配符订阅可用属性，或通配符订阅可用属性值不为0也不为1将造成协议错误
	// - 如果服务端在不支持通配符订阅的情况下收到了包含通配符订阅的SUBSCRIBE报文，将造成协议错误
	// - 服务端在支持通配符订阅的情况下仍然可以拒绝特定的包含通配符订阅的订阅请求
	WildcardSubscriptionAvailable uint8

	// SubscriptionIdentifierAvailable 订阅标识符可用
	// 属性标识符: 41 (0x29)
	// 参考章节: 3.2.2.3.12 Subscription Identifier Available
	// 类型: 单字节，值: 0或1
	// 含义: 服务端是否支持订阅标识符
	// 默认值: 1 (支持订阅标识符，如果未设置)
	// 注意:
	// - 包含多个订阅标识符可用属性，或订阅标识符可用属性值不为0也不为1将造成协议错误
	// - 如果服务端在不支持订阅标识符的情况下收到了包含订阅标识符的SUBSCRIBE报文，将造成协议错误
	SubscriptionIdentifierAvailable uint8

	// SharedSubscriptionAvailable 共享订阅可用
	// 属性标识符: 42 (0x2A)
	// 参考章节: 3.2.2.3.13 Shared Subscription Available
	// 类型: 单字节，值: 0或1
	// 含义: 服务端是否支持共享订阅
	// 默认值: 1 (支持共享订阅，如果未设置)
	// 注意:
	// - 包含多个共享订阅可用，或共享订阅可用属性值不为0也不为1将造成协议错误
	// - 如果服务端在不支持共享订阅的情况下收到了包含共享订阅的SUBSCRIBE报文，将造成协议错误
	SharedSubscriptionAvailable uint8

	// ServerKeepAlive 服务端保持连接
	// 属性标识符: 19 (0x13)
	// 参考章节: 3.2.2.3.14 Server Keep Alive
	// 类型: 双字节整数，单位: 秒
	// 含义: 服务端分配的保持连接时间
	// 注意:
	// - 如果服务端发送了服务端保持连接属性，客户端必须使用此值代替其在CONNECT报文中发送的保持连接时间值 [MQTT-3.2.2-21]
	// - 如果服务端没有发送服务端保持连接属性，服务端必须使用客户端在CONNECT报文中设置的保持连接时间值 [MQTT-3.2.2-22]
	// - 包含多个服务端保持连接属性将造成协议错误
	// 非规范评注:
	// - 服务端保持连接属性的主要作用是通知客户端它将会比客户端指定的保持连接更快的断开非活动的客户端
	ServerKeepAlive uint16

	// ResponseInformation 响应信息
	// 属性标识符: 26 (0x1A)
	// 参考章节: 3.2.2.3.15 Response Information
	// 类型: UTF-8编码字符串
	// 含义: 作为创建响应主题的基本信息
	// 注意:
	// - 关于客户端如何根据响应信息创建响应主题不在本规范的定义范围内
	// - 包含多个响应信息将造成协议错误
	// - 如果客户端发送的请求响应信息值为1，则服务端在CONNACK报文中发送响应信息为可选项
	// 非规范评注:
	// - 响应信息通常被用来传递主题订阅树的一个全局唯一分支，此分支至少在该客户端的会话生命周期内为该客户端所保留
	// - 请求客户端和响应客户端的授权需要使用它，所以它通常不能仅仅是一个随机字符串
	// - 一般把此分支作为特定客户端的订阅树根节点
	ResponseInformation string

	// ServerReference 服务端参考
	// 属性标识符: 28 (0x1C)
	// 参考章节: 3.2.2.3.16 Server Reference
	// 类型: UTF-8编码字符串
	// 含义: 可以被客户端用来标识其他可用的服务端
	// 注意:
	// - 包含多个服务端参考将造成协议错误
	// - 服务端在包含了原因码为0x9C（（临时）使用其他服务端）或0x9D（服务端已（永久）移动）的CONNACK报文或DISCONNECT报文中设置服务端参考
	// - 关于如何使用服务端参考，请参考4.11节服务端重定向信息
	ServerReference string

	// AuthenticationMethod 认证方法
	// 属性标识符: 21 (0x15)
	// 参考章节: 3.2.2.3.17 Authentication Method
	// 类型: UTF-8编码字符串
	// 含义: 扩展认证的认证方法名称
	// 注意:
	// - 包含多个认证方法将造成协议错误
	// - 更多关于扩展认证的信息，请参考4.12节
	AuthenticationMethod string

	// AuthenticationData 认证数据
	// 属性标识符: 22 (0x16)
	// 参考章节: 3.2.2.3.18 Authentication Data
	// 类型: 二进制数据
	// 含义: 包含认证数据的二进制数据
	// 注意:
	// - 此数据的内容由认证方法和已交换的认证数据状态定义
	// - 包含多个认证数据将造成协议错误
	// - 更多关于扩展认证的信息，请参考4.12节
	AuthenticationData []byte
}

func (props *ConnackProps) Pack() ([]byte, error) {
	buf := GetBuffer()
	defer PutBuffer(buf)
	if props.SessionExpiryInterval != 0 {
		buf.WriteByte(0x11)
		buf.Write(i4b(props.SessionExpiryInterval))
	}

	if props.ReceiveMaximum != 0 {
		buf.WriteByte(0x21)
		buf.Write(i2b(props.ReceiveMaximum))
	}

	if props.MaximumQoS != 0 {
		buf.WriteByte(0x24)
		buf.WriteByte(props.MaximumQoS)
	}

	if props.RetainAvailable != 0 {
		buf.WriteByte(0x25)
		buf.WriteByte(props.RetainAvailable)
	}

	if props.MaximumPacketSize != 0 {
		buf.WriteByte(0x27)
		buf.Write(i4b(props.MaximumPacketSize))
	}
	if props.AssignedClientID != "" {
		buf.WriteByte(0x12)
		buf.Write(encodeUTF8(props.AssignedClientID))
	}

	if props.TopicAliasMaximum != 0 {
		buf.WriteByte(0x22)
		buf.Write(i2b(props.TopicAliasMaximum))
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

	if props.WildcardSubscriptionAvailable != 0 {
		buf.WriteByte(0x28)
		buf.WriteByte(props.WildcardSubscriptionAvailable)
	}

	if props.SubscriptionIdentifierAvailable != 0 {
		buf.WriteByte(0x29)
		buf.WriteByte(props.SubscriptionIdentifierAvailable)
	}

	if props.SharedSubscriptionAvailable != 0 {
		buf.WriteByte(0x2A)
		buf.WriteByte(props.SharedSubscriptionAvailable)
	}

	if props.ServerKeepAlive != 0 {
		buf.WriteByte(0x13)
		buf.Write(i2b(props.ServerKeepAlive))
	}

	if len(props.ResponseInformation) != 0 {
		buf.WriteByte(0x1A)
		buf.Write(encodeUTF8(props.ResponseInformation))
	}

	if len(props.ServerReference) != 0 {
		buf.WriteByte(0x1C)
		buf.Write(encodeUTF8(props.ServerReference))
	}

	if len(props.AuthenticationMethod) != 0 {
		buf.WriteByte(0x15)
		buf.Write(encodeUTF8(props.AuthenticationMethod))
	}

	if len(props.AuthenticationData) != 0 {
		buf.WriteByte(0x16)
		buf.Write(encodeUTF8(props.AuthenticationData))
	}

	return buf.Bytes(), nil

}

func (props *ConnackProps) Unpack(b *bytes.Buffer) error {
	propsLen, err := decodeLength(b)
	if err != nil {
		return err
	}

	for i := uint32(0); i < propsLen; i++ {
		propsId, err := decodeLength(b)
		if err != nil {
			return err
		}
		switch propsId {
		case 0x11: // 会话过期间隔 Session Expiry Interval
			props.SessionExpiryInterval, i = binary.BigEndian.Uint32(b.Next(4)), i+4
		case 0x21:
			props.ReceiveMaximum, i = binary.BigEndian.Uint16(b.Next(2)), i+2
		case 0x24:
			props.MaximumQoS, i = b.Next(1)[0], i+1
		case 0x25:
			props.RetainAvailable, i = b.Next(1)[0], i+1
		case 0x27:
			props.MaximumPacketSize, i = binary.BigEndian.Uint32(b.Next(4)), i+4
		case 0x12:
			props.AssignedClientID, i = decodeUTF8[string](b), i+uint32(len(props.AssignedClientID))
		case 0x22:
			props.TopicAliasMaximum, i = binary.BigEndian.Uint16(b.Next(2)), i+2
		case 0x1F:
			props.ReasonString, i = decodeUTF8[string](b), i+uint32(len(props.ReasonString))
		case 0x26:
			if props.UserProperty == nil {
				props.UserProperty = make(map[string][]string)
			}
			key := decodeUTF8[string](b)
			props.UserProperty[key] = append(props.UserProperty[key], decodeUTF8[string](b))
		case 0x28:
			props.WildcardSubscriptionAvailable, i = b.Next(1)[0], i+1
		case 0x29:
			props.SubscriptionIdentifierAvailable, i = b.Next(1)[0], i+1
		case 0x2A:
			props.SharedSubscriptionAvailable, i = b.Next(1)[0], i+1
		case 0x13:
			props.ServerKeepAlive, i = binary.BigEndian.Uint16(b.Next(2)), i+2
		case 0x1A:
			props.ResponseInformation, i = decodeUTF8[string](b), i+uint32(len(props.ResponseInformation))
		case 0x1C:
			props.ServerReference, i = decodeUTF8[string](b), i+uint32(len(props.ServerReference))
		case 0x15:
			props.AuthenticationMethod, i = decodeUTF8[string](b), i+uint32(len(props.AuthenticationMethod))
		case 0x16:
			props.AuthenticationData, i = decodeUTF8[[]byte](b), i+uint32(len(props.AuthenticationMethod))
		}
	}
	return nil
}
