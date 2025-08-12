package packet

import (
	"bytes"
	"fmt"
	"io"
)

/*
MQTT CONNACK 连接确认报文 - 完整协议实现

参考文档:
- MQTT Version 3.1.1: 章节 3.2 CONNACK - Acknowledge connection request
- MQTT Version 5.0: 章节 3.2 CONNACK - Connect acknowledgement

报文结构:
┌─────────────────────────────────────────────────────────────┐
│                    Fixed Header                            │
├─────────────────────────────────────────────────────────────┤
│ MQTT Control Packet Type (2) │ Reserved (0) │ Remaining   │
│                              │              │ Length      │
├─────────────────────────────────────────────────────────────┤
│                   Variable Header                          │
├─────────────────────────────────────────────────────────────┤
│ Connect Acknowledge Flags │ Connect Return Code │ Props    │
│ (1 byte)                 │ (1 byte)           │ (v5.0)   │
├─────────────────────────────────────────────────────────────┤
│                        Payload                             │
│                        (None)                              │
└─────────────────────────────────────────────────────────────┘

版本差异:
- v3.1.1: 基本的连接确认功能，包含连接返回码
- v5.0: 在v3.1.1基础上增加了属性系统，支持更详细的连接状态反馈

协议约束:
[MQTT-3.2.0-1] 服务端发送给客户端的第一个报文必须是CONNACK报文
[MQTT-3.2.0-2] 服务端在网络连接中不能发送超过一个CONNACK报文
[MQTT-3.2.2-1] 连接确认标志的bit 7-1是保留位，必须设置为0
[MQTT-3.2.2-2] 如果服务端接受CleanStart=1的连接，必须设置SessionPresent=0
[MQTT-3.2.2-3] 如果服务端接受CleanStart=0的连接且有会话状态，必须设置SessionPresent=1
[MQTT-3.2.2-4] 如果客户端没有会话状态但收到SessionPresent=1，必须关闭网络连接
[MQTT-3.2.2-5] 如果客户端有会话状态但收到SessionPresent=0，必须丢弃会话状态
[MQTT-3.2.2-6] 如果服务端发送包含非零原因码的CONNACK，必须设置SessionPresent=0
[MQTT-3.2.2-7] 如果服务端发送原因码≥128的CONNACK，必须关闭网络连接
[MQTT-3.2.2-8] 服务端必须使用表3-1中的连接原因码值
*/
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
	// - bits 7-1为保留位，必须为0 [MQTT-3.2.2-1]
	// - 如果服务端发送包含非零原因码的CONNACK，必须设置SessionPresent=0 [MQTT-3.2.2-6]
	SessionPresent uint8

	// ReturnCode 连接返回码 (v3.1.1) / Connect Reason Code (v5.0)
	// 位置: 可变报头第2字节
	// 参考章节: 3.2.2.2 Connect Return code / Connect Reason Code
	// 含义: 表示连接请求的处理结果
	//
	// MQTT v3.1.1 连接返回码:
	// - 0x00: 连接已接受 - 连接已被服务端接受
	// - 0x01: 连接已拒绝，不支持的协议版本 - 服务端不支持客户端请求的MQTT协议级别
	// - 0x02: 连接已拒绝，不合格的客户端标识符 - 客户端标识符是正确的UTF-8编码，但服务端不允许使用
	// - 0x03: 连接已拒绝，服务端不可用 - 网络连接已建立，但MQTT服务不可用
	// - 0x04: 连接已拒绝，无效的用户名或密码 - 用户名或密码的数据格式无效
	// - 0x05: 连接已拒绝，未授权 - 客户端未被授权连接到此服务端
	// - 6-255: 保留供将来使用
	//
	// MQTT v5.0 连接原因码:
	// - 0x00: 成功 - 连接被接受
	// - 0x80: 未指定错误 - 服务端不希望透露失败原因
	// - 0x81: 格式错误报文 - CONNECT报文中的数据无法正确解析
	// - 0x82: 协议错误 - CONNECT报文中的数据不符合本规范
	// - 0x83: 实现特定错误 - CONNECT有效但不被服务端接受
	// - 0x84: 不支持的协议版本 - 服务端不支持客户端请求的MQTT协议版本
	// - 0x85: 客户端标识符无效 - 客户端标识符有效但不被服务端允许
	// - 0x86: 用户名或密码错误 - 服务端不接受客户端指定的用户名或密码
	// - 0x87: 未授权 - 客户端未被授权连接
	// - 0x88: 服务端不可用 - MQTT服务端不可用
	// - 0x89: 服务端忙 - 服务端忙，稍后重试
	// - 0x8A: 被禁止 - 客户端被管理操作禁止
	// - 0x8C: 认证方法错误 - 认证方法不支持或不匹配
	// - 0x90: 主题名无效 - Will主题名格式正确但不被服务端接受
	// - 0x95: 报文过大 - CONNECT报文超过最大允许大小
	// - 0x97: 配额超出 - 超出实现或管理限制
	// - 0x99: 载荷格式无效 - Will载荷与指定的载荷格式指示符不匹配
	// - 0x9A: 不支持保留 - 服务端不支持保留消息
	// - 0x9B: 不支持QoS - 服务端不支持Will QoS
	// - 0x9C: 使用其他服务端 - 客户端应临时使用其他服务端
	// - 0x9D: 服务端已移动 - 客户端应永久使用其他服务端
	// - 0x9F: 连接速率超出 - 连接速率限制已超出
	//
	// 注意:
	// - 如果服务端发送了一个包含非零返回码的CONNACK报文，那么它必须关闭网络连接 [MQTT-3.2.2-5]
	// - 如果认为上表中的所有连接返回码都不太合适，那么服务端必须关闭网络连接，不需要发送CONNACK报文 [MQTT-3.2.2-6]
	// - 如果服务端发送原因码≥128的CONNACK，必须关闭网络连接 [MQTT-3.2.2-7]
	// - 服务端必须使用表3-1中的连接原因码值 [MQTT-3.2.2-8]
	ReturnCode ReasonCode `json:"ReturnCode,omitempty"`

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
	return fmt.Sprintf("[0x2]ConnectReturnCode=%d", pkt.ReturnCode.Code)
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
	// 注意: bits 7-1必须为0 [MQTT-3.2.2-1]
	sessionPresentByte := pkt.SessionPresent & 0x01 // 只保留bit 0，其他位设为0
	buf.WriteByte(sessionPresentByte)

	// 写入连接返回码
	// 参考章节: 3.2.2.2 Connect Return code
	buf.WriteByte(pkt.ReturnCode.Code)

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
	pkt.SessionPresent = buf.Next(1)[0] & 0x01
	pkt.ReturnCode = ReasonCode{Code: buf.Next(1)[0]}

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
//
// 属性编码格式:
// ┌─────────────────────────────────────────────────────────────┐
// │ Property Length (Variable Byte Integer)                    │
// ├─────────────────────────────────────────────────────────────┤
// │ Property Identifier (Variable Byte Integer)                │
// ├─────────────────────────────────────────────────────────────┤
// │ Property Value (根据属性类型编码)                           │
// ├─────────────────────────────────────────────────────────────┤
// │ ... 更多属性 ...                                           │
// └─────────────────────────────────────────────────────────────┘
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
	SessionExpiryInterval SessionExpiryInterval

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
	ReceiveMaximum ReceiveMaximum

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
	MaximumQoS MaximumQoS

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
	RetainAvailable RetainAvailable

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
	MaximumPacketSize MaximumPacketSize

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
	AssignedClientID AssignedClientID

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
	TopicAliasMaximum TopicAliasMaximum

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
	ReasonString ReasonString

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
	UserProperty UserProperty

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
	WildcardSubscriptionAvailable WildcardSubscriptionAvailable

	// SubscriptionIdentifierAvailable 订阅标识符可用
	// 属性标识符: 41 (0x29)
	// 参考章节: 3.2.2.3.12 Subscription Identifier Available
	// 类型: 单字节，值: 0或1
	// 含义: 服务端是否支持订阅标识符
	// 默认值: 1 (支持订阅标识符，如果未设置)
	// 注意:
	// - 包含多个订阅标识符可用属性，或订阅标识符可用属性值不为0也不为1将造成协议错误
	// - 如果服务端在不支持订阅标识符的情况下收到了包含订阅标识符的SUBSCRIBE报文，将造成协议错误
	SubscriptionIdentifierAvailable SubscriptionIdentifierAvailable

	// SharedSubscriptionAvailable 共享订阅可用
	// 属性标识符: 42 (0x2A)
	// 参考章节: 3.2.2.3.13 Shared Subscription Available
	// 类型: 单字节，值: 0或1
	// 含义: 服务端是否支持共享订阅
	// 默认值: 1 (支持共享订阅，如果未设置)
	// 注意:
	// - 包含多个共享订阅可用，或共享订阅可用属性值不为0也不为1将造成协议错误
	// - 如果服务端在不支持共享订阅的情况下收到了包含共享订阅的SUBSCRIBE报文，将造成协议错误
	SharedSubscriptionAvailable SharedSubscriptionAvailable

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
	ServerKeepAlive ServerKeepAlive

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
	ResponseInformation ResponseInformation

	// ServerReference 服务端参考
	// 属性标识符: 28 (0x1C)
	// 参考章节: 3.2.2.3.16 Server Reference
	// 类型: UTF-8编码字符串
	// 含义: 可以被客户端用来标识其他可用的服务端
	// 注意:
	// - 包含多个服务端参考将造成协议错误
	// - 服务端在包含了原因码为0x9C（（临时）使用其他服务端）或0x9D（服务端已（永久）移动）的CONNACK报文或DISCONNECT报文中设置服务端参考
	// - 关于如何使用服务端参考，请参考4.11节服务端重定向信息
	ServerReference ServerReference

	// AuthenticationMethod 认证方法
	// 属性标识符: 21 (0x15)
	// 参考章节: 3.2.2.3.17 Authentication Method
	// 类型: UTF-8编码字符串
	// 含义: 扩展认证的认证方法名称
	// 注意:
	// - 包含多个认证方法将造成协议错误
	// - 更多关于扩展认证的信息，请参考4.12节
	AuthenticationMethod AuthenticationMethod

	// AuthenticationData 认证数据
	// 属性标识符: 22 (0x16)
	// 参考章节: 3.2.2.3.18 Authentication Data
	// 类型: 二进制数据
	// 含义: 包含认证数据的二进制数据
	// 注意:
	// - 此数据的内容由认证方法和已交换的认证数据状态定义
	// - 包含多个认证数据将造成协议错误
	// - 更多关于扩展认证的信息，请参考4.12节
	AuthenticationData AuthenticationData
}

// Pack 将CONNACK属性序列化为字节数组
// 参考章节: 3.2.2.3 CONNACK Properties
// 序列化顺序: 按属性标识符顺序写入属性值
// 注意: 只序列化非零/非空的属性值
func (props *ConnackProps) Pack() ([]byte, error) {
	buf := GetBuffer()
	defer PutBuffer(buf)
	if err := props.SessionExpiryInterval.Pack(buf); err != nil {
		return nil, err
	}
	if err := props.ReceiveMaximum.Pack(buf); err != nil {
		return nil, err
	}
	if err := props.MaximumQoS.Pack(buf); err != nil {
		return nil, err
	}
	if err := props.RetainAvailable.Pack(buf); err != nil {
		return nil, err
	}
	if err := props.MaximumPacketSize.Pack(buf); err != nil {
		return nil, err
	}
	if err := props.TopicAliasMaximum.Pack(buf); err != nil {
		return nil, err
	}
	if err := props.ReasonString.Pack(buf); err != nil {
		return nil, err
	}
	if err := props.UserProperty.Pack(buf); err != nil {
		return nil, err
	}
	if err := props.WildcardSubscriptionAvailable.Pack(buf); err != nil {
		return nil, err
	}
	if err := props.SubscriptionIdentifierAvailable.Pack(buf); err != nil {
		return nil, err
	}
	if err := props.SharedSubscriptionAvailable.Pack(buf); err != nil {
		return nil, err
	}
	if err := props.ServerKeepAlive.Pack(buf); err != nil {
		return nil, err
	}
	if err := props.ResponseInformation.Pack(buf); err != nil {
		return nil, err
	}
	if err := props.ServerReference.Pack(buf); err != nil {
		return nil, err
	}
	if err := props.AuthenticationMethod.Pack(buf); err != nil {
		return nil, err
	}
	if err := props.AuthenticationData.Pack(buf); err != nil {
		return nil, err
	}
	return bytes.Clone(buf.Bytes()), nil
}

// Unpack 从缓冲区解析CONNACK属性
// 参考章节: 3.2.2.3 CONNACK Properties
func (props *ConnackProps) Unpack(buf *bytes.Buffer) error {
	// 解析属性长度
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
		case 0x11: // 会话过期间隔 Session Expiry Interval
			if uLen, err = props.SessionExpiryInterval.Unpack(buf); err != nil {
				return err
			}
		case 0x12: // 分配客户标识符 Assigned Client Identifier
			if uLen, err = props.AssignedClientID.Unpack(buf); err != nil {
				return err
			}
		case 0x13: // 服务端保持连接 Server Keep Alive
			if uLen, err = props.ServerKeepAlive.Unpack(buf); err != nil {
				return err
			}
		case 0x15: // 认证方法 Authentication Method
			if uLen, err = props.AuthenticationMethod.Unpack(buf); err != nil {
				return err
			}
		case 0x16: // 认证数据 Authentication Data
			if uLen, err = props.AuthenticationData.Unpack(buf); err != nil {
				return err
			}
		case 0x1A: // 响应信息 Response Information
			if uLen, err = props.ResponseInformation.Unpack(buf); err != nil {
				return err
			}
		case 0x1C: // 服务端参考 Server Reference
			if uLen, err = props.ServerReference.Unpack(buf); err != nil {
				return err
			}
		case 0x1F: // 原因字符串 Reason String
			if uLen, err = props.ReasonString.Unpack(buf); err != nil {
				return err
			}
		case 0x21: // 接收最大值 Receive Maximum
			if uLen, err = props.ReceiveMaximum.Unpack(buf); err != nil {
				return err
			}
		case 0x22: // 主题别名最大值 Topic Alias Maximum
			if uLen, err = props.TopicAliasMaximum.Unpack(buf); err != nil {
				return err
			}
		case 0x24: // 最大服务质量 Maximum QoS
			if uLen, err = props.MaximumQoS.Unpack(buf); err != nil {
				return err
			}
		case 0x25: // 保留可用 Retain Available
			if uLen, err = props.RetainAvailable.Unpack(buf); err != nil {
				return err
			}
		case 0x26: // 用户属性 User Property
			if uLen, err = props.UserProperty.Unpack(buf); err != nil {
				return err
			}
		case 0x27: // 最大报文长度 Maximum Packet Size
			if uLen, err = props.MaximumPacketSize.Unpack(buf); err != nil {
				return err
			}
		case 0x28: // 通配符订阅可用 Wildcard Subscription Available
			if uLen, err = props.WildcardSubscriptionAvailable.Unpack(buf); err != nil {
				return err
			}
		case 0x29: // 订阅标识符可用 Subscription Identifier Available
			if uLen, err = props.SubscriptionIdentifierAvailable.Unpack(buf); err != nil {
				return err
			}
		case 0x2A: // 共享订阅可用 Shared Subscription Available
			if uLen, err = props.SharedSubscriptionAvailable.Unpack(buf); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown property id: %d", propsId)
		}
		i += uLen
	}
	return nil
}
