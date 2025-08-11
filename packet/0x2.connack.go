package packet

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

type CONNACK struct {
	*FixedHeader

	// 可变报头
	SessionPresent uint8 // byte1.bits0, bits 7-6. Reserved,

	// ConnectReturnCode ConnectReturnCode
	// 如果服务端收到一个合法的CONNECT报文，但出于某些原因无法处理它，服务端应该尝试发送一个包含非零返回码（表格中的某一个）的 CONNACK 报文。
	// 如果服务端发送了一个包含非零返回码的 CONNACK 报文，那么它必须关闭网络连接 [MQTT-3.2.2-5].
	// 0x00 连接已接受 					连接已被服务端接受
	// 0x01 连接已拒绝，不支持的协议版本 	服务端不支持客户端请求的 MQTT 协议级别
	// 0x02 连接已拒绝，不合格的客户端标识符	客户端标识符是正确的 UTF-8 编码，但服务端不允许使用
	// 0x03 连接已拒绝，服务端不可用		网络连接已建立，但 MQTT 服务不可用
	// 0x04 连接已拒绝，无效的用户名或密码	用户名或密码的数据格式无效
	// 0x05 连接已拒绝，未授权
	// --
	// 如果认为上表中的所有连接返回码都不太合适，那么服务端必须关闭网络连接，不需要发送 CONNACK 报文 [MQTT-3.2.2-6]。
	ConnectReturnCode ReasonCode `json:"ConnectReturnCode,omitempty"` // ConnectReturnCode
	ConnackProps      *ConnackProps
}

func (pkt *CONNACK) Kind() byte {
	return 0x2
}

func (pkt *CONNACK) String() string {
	return fmt.Sprintf("[0x2]ConnectReturnCode=%d", pkt.ConnectReturnCode.Code)
}

func (pkt *CONNACK) Pack(w io.Writer) error {
	buf := GetBuffer()
	defer PutBuffer(buf)
	buf.WriteByte(pkt.SessionPresent)
	buf.WriteByte(pkt.ConnectReturnCode.Code)
	if pkt.Version == VERSION500 {
		if pkt.ConnackProps == nil {
			pkt.ConnackProps = &ConnackProps{}
		}
		b, err := pkt.ConnackProps.Pack()
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
func (pkt *CONNACK) Unpack(buf *bytes.Buffer) error {
	pkt.SessionPresent = buf.Next(1)[0]
	pkt.ConnectReturnCode = ReasonCode{Code: buf.Next(1)[0]}

	if pkt.Version == VERSION500 {
		pkt.ConnackProps = &ConnackProps{}
		if err := pkt.ConnackProps.Unpack(buf); err != nil {
			return err
		}
	}
	return nil
}

type ConnackProps struct {
	// 3.2.2.3.1
	//属性长度
	//CONNACK 报文可变报头中的属性长度，编码为变长字节整数。
	//PropertyLength int

	//3.2.2.3.2
	//会话过期间隔
	//17 (0x11)，会话过期间隔（Session Expiry Interval）标识符。
	//跟随其后的是用四字节整数表示的以秒为单位的会话过期间隔（Session Expiry Interval）。包含多个会话
	//过期间隔（Session Expiry Interval）将造成协议错误（Protocol Error）。
	//如果会话过期间隔（Session Expiry Interval）值未指定，则使用 CONNECT 报文中指定的会话过期时间间
	//隔。服务端使用此属性通知客户端它使用的会话过期时间间隔与客户端在 CONNECT 中发送的值不同。更
	//详细的关于会话过期时间的描述，请参考 3.1.2.11.2 节 。
	SessionExpiryInterval uint32

	// 3.2.2.3.3
	//接收最大值
	//33 (0x21)，接收最大值（Receive Maximum）描述符。
	//跟随其后的是由双字节整数表示的最大接收值。包含多个接收最大值或接收最大值为 0 将造成协议错误
	//（Protocol Error）。
	//服务端使用此值限制服务端愿意为该客户端同时处理的 QoS 为 1 和 QoS 为 2 的发布消息最大数量。没有
	//机制可以限制客户端试图发送的 QoS 为 0 的发布消息。
	// 如果没有设置最大接收值，将使用默认值 65535。
	//关于接收最大值的详细使用，参考 4.9 节 流控部分。
	ReceiveMaximum uint16

	// 3.2.2.3.4
	//最大服务质量
	//36 (0x24)，最大服务质量（Maximum QoS）标识符。
	//跟随其后的是用一个字节表示的 0 或 1。包含多个最大服务质量（Maximum QoS）或最大服务质量既不为
	//0 也不为 1 将造成协议错误。如果没有设置最大服务质量，客户端可使用最大 QoS 为 2。
	//如果服务端不支持 Qos 为 1 或 2 的 PUBLISH 报文，服务端必须在 CONNACK 报文中发送最大服务质量以
	//指定其支持的最大 QoS 值 [MQTT-3.2.2-9]。即使不支持 QoS 为 1 或 2 的 PUBLISH 报文，服务端也必须
	//接受请求 QoS 为 0、1 或 2 的 SUBSCRIBE 报文 [MQTT-3.2.2-10]。
	//如果从服务端接收到了最大 QoS 等级，则客户端不能发送超过最大 QoS 等级所指定的 QoS 等级的
	//PUBLISH 报文 [MQTT-3.2.2-11]。服务端接收到超过其指定的最大服务质量的 PUBLISH 报文将造成协议
	//错误（Protocol Error）。这种情况下应使用包含原因码为 0x9B（不支持的 QoS 等级）的 DISCONNECT
	//报文进行处理，如 4.13 节 所述。
	//如果服务端收到包含遗嘱的 QoS 超过服务端处理能力的 CONNECT 报文，服务端必须拒绝此连接。服务
	//端应该使用包含原因码为 0x9B（不支持的 QoS 等级）的 CONNACK 报文进行错误处理，随后必须关闭网
	//络连接。4.13 节 所述 [MQTT-3.2.2-12]。
	//非规范评注
	//客户端不必支持 QoS 为 1 和 2 的 PUBLISH 报文。客户端只需将其发送的任何 SUBSCRIBE 报文
	//中的 QoS 字段限制在其支持的最大服务质量以内即可。
	MaximumQoS uint8
	//3.2.2.3.5
	//保留可用
	//37 (0x25)，保留可用（Retain Available）标识符。
	//跟随其后的是一个单字节字段，用来声明服务端是否支持保留消息。值为 0 表示不支持保留消息，为 1 表
	//示支持保留消息。如果没有设置保留可用字段，表示支持保留消息。包含多个保留可用字段或保留可用字
	//段值不为 0 也不为 1 将造成协议错误（Protocol Error）。
	//如果服务端收到一个包含保留标志位 1 的遗嘱消息的 CONNECT 报文且服务端不支持保留消息，服务端必
	//须拒绝此连接请求，且应该发送包含原因码为 0x9A（不支持保留）的 CONNACK 报文，随后必须关闭网
	//络连接 [MQTT-3.2.2-13]。
	//从服务端接收到的保留可用标志为 0 时，客户端不能发送保留标志设置为 1 的 PUBLISH 报文 [MQTT-
	//3.2.2-14]。如果服务端收到这种 PUBLISH 报文，将造成协议错误（Protocol Error），此时服务端应该发
	//送包含原因码为 0x9A（不支持保留）的 DISCONNECT 报文，如 4.13 节 所述。

	RetainAvailable uint8
	// 3.2.2.3.6
	//最大报文长度
	//39 (0x27)，最大报文长度（Maximum Packet Size）标识符。
	//跟随其后的是由四字节整数表示的服务端愿意接收的最大报文长度（Maximum Packet Size）。如果没有
	//设置最大报文长度，则按照协议由固定报头中的剩余长度可编码最大值和协议报头对数据包的大小做限制。
	//包含多个最大报文长度（Maximum Packet Size），或最大报文长度为 0 将造成协议错误（Protocol
	//Error）。
	//如 2.1.4 节 所述，最大报文长度是 MQTT 控制报文的总长度。服务端使用最大报文长度通知客户端其所能
	//处理的单个报文长度限制。
	//客户端不能发送超过最大报文长度（Maximum Packet Size）的报文给服务端 [MQTT-3.2.2-15]。收到长度
	//超过限制的报文将导致协议错误，此时服务端应该发送包含原因码 0x95（报文过长）的 DISCONNECT 报
	//文给客户端，详见 4.13 节。
	MaximumPacketSize uint32

	//3.2.2.3.7
	//分配客户标识符
	//18 (0x12)，分配客户标识符（Assigned Client Identifier）标识符。
	//跟随其后的是 UTF-8 编码的分配客户标识符（Assigned Client Identifier）字符串。包含多个分配客户标识
	//符将造成协议错误（Protocol Error）。
	//服务端分配客户标识符的原因是 CONNECT 报文中的客户标识符长度为 0。
	//如果客户端使用长度为 0 的客户标识符（ClientID），服务端必须回复包含分配客户标识符（Assigned
	//Client Identifier）的 CONNACK 报文。分配客户标识符必须是没有被服务端的其他会话所使用的新客户标
	//识符 [MQTT-3.2.2-16]。
	AssignedClientID string

	//3.2.2.3.8
	//主题别名最大值
	//34 (0x22)，主题别名最大值（Topic Alias Maximum）标识符。
	//跟随其后的是用双字节整数表示的主题别名最大值（Topic Alias Maximum）。包含多个主题别名最大值
	//（Topic Alias Maximum）将造成协议错误（Protocol Error）。没有设置主题别名最大值属性的情况下，主
	//题别名最大值默认为零。
	//此值指示了服务端能够接收的来自客户端的主题别名（Topic Alias）最大值。服务端使用此值来限制本次
	//连接可以拥有的主题别名的值。客户端在一个 PUBLISH 报文中发送的主题别名值不能超过服务端设置的主
	//题别名最大值（Topic Alias Maximum） [MQTT-3.2.2-17]。值为 0 表示本次连接服务端不接受任何主题别
	//名（Topic Alias）。如果主题别名最大值（Topic Alias）没有设置，或者设置为 0，则客户端不能向此服务
	//端发送任何主题别名（Topic Alias） [MQTT-3.2.2-18]。
	TopicAliasMaximum uint16

	//3.2.2.3.9
	//原因字符串
	//31 (0x1F)，原因字符串（Reason String）标识符。
	// 跟随其后的是 UTF-8 编码的字符串，表示此次响应相关的原因。此原因字符串（Reason String）是为诊断
	//而设计的可读字符串，不应该被客户端所解析。
	//服务端使用此值向客户端提供附加信息。如果加上原因字符串之后的 CONNACK 报文长度超出了客户端指
	//定的最大报文长度，则服务端不能发送此原因字符串 [MQTT-3.2.2-19]。包含多个原因字符串将造成协议错
	//误（Protocol Error）。
	//非规范评注
	//客户端对原因字符串的恰当使用包括：抛出异常时使用此字符串，或者将此字符串写入日志。
	ReasonString string

	//3.2.2.3.10
	//用户属性
	//38 (0x26)，用户属性（User Property）标识符。
	//跟随其后的是 UTF-8 字符串对。此属性可用于向客户端提供包括诊断信息在内的附加信息。如果加上用户
	//属性之后的 CONNACK 报文长度超出了客户端指定的最大报文长度，则服务端不能发送此属性 [MQTT-
	//3.2.2-20]。用户属性（User Property）允许出现多次，以表示多个名字/值对，且相同的名字可以多次出现。
	//用户属性的内容和意义本规范不做定义。CONNACK 报文的接收端可以选择忽略此属性。
	UserProperty map[string][]string
	//3.2.2.3.11
	//通配符订阅可用
	//40 (0x28)，通配符订阅可用（Wildcard Subscription Available）标识符。
	//跟随其后的是一个单字节字段，用来声明服务器是否支持通配符订阅（Wildcard Subscriptions）。值为 0
	//表示不支持通配符订阅，值为 1 表示支持通配符订阅。如果没有设置此值，则表示支持通配符订阅。包含
	//多个通配符订阅可用属性，或通配符订阅可用属性值不为 0 也不为 1 将造成协议错误（Protocol Error）。
	//如果服务端在不支持通配符订阅（Wildcard Subscription）的情况下收到了包含通配符订阅的 SUBSCRIBE
	//报文，将造成协议错误（Protocol Error）。此时服务端将发送包含原因码为 0xA2（通配符订阅不支持）的
	//DISCONNECT 报文，如 4.13 节 所述。
	//服务端在支持通配符订阅的情况下仍然可以拒绝特定的包含通配符订阅的订阅请求。这种情况下，服务端
	//可以发送一个包含原因码为 0xA2（通配符订阅不支持）的 SUBACK 报文。
	WildcardSubscriptionAvailable uint8

	//3.2.2.3.12
	//订阅标识符可用
	//41 (0x29)，订阅标识符可用（Subscription Identifier Available）标识符。
	//跟随其后的是一个单字节字段，用来声明服务端是否支持订阅标识符（Subscription Identifiers）。值为 0
	//表示不支持订阅标识符，值为 1 表示支持订阅标识符。如果没有设置此值，则表示支持订阅标识符。包含
	//多个订阅标识符可用属性，或订阅标识符可用属性值不为 0 也不为 1 将造成协议错误（Protocol Error）。
	//如果服务端在不支持订阅标识符（Subscription Identifier）的情况下收到了包含订阅标识符的 SUBSCRIBE
	//报文，将造成协议错误（Protocol Error）。此时服务端将发送包含原因码为 0xA1（订阅标识符不支持）的
	//DISCONNECT 报文，如 4.13 节 所述。
	SubscriptionIdentifierAvailable uint8

	// 3.2.2.3.13
	//共享订阅可用
	//42 (0x2A)，共享订阅可用（Shared Subscription Available）标识符。
	//跟随其后的是一个单字节字段，用来声明服务端是否支持共享订阅（Shared Subscription）。值为 0 表示
	//不支持共享订阅，值为 1 表示支持共享订阅。如果没有设置此值，则表示支持共享订阅。包含多个共享订
	//阅可用（Shared Subscription Available），或共享订阅可用属性值不为 0 也不为 1 将造成协议错误
	//（Protocol Error）。
	//如果服务端在不支持共享订阅（Shared Subscription）的情况下收到了包含共享订阅的 SUBSCRIBE 报文，
	//将造成协议错误（Protocol Error）。此时服务端将发送包含原因码为 0x9E（共享订阅不支持）的
	//DISCONNECT 报文，如 4.13 节 所述。
	SharedSubscriptionAvailable uint8

	//3.2.2.3.14
	//服务端保持连接
	//19 (0x13)，服务端保持连接（Server Keep Alive）标识符。
	//跟随其后的是由服务端分配的双字节整数表示的保持连接（Keep Alive）时间。如果服务端发送了服务端保
	//持连接（Server Keep Alive）属性，客户端必须使用此值代替其在 CONNECT 报文中发送的保持连接时间
	//值 [MQTT-3.2.2-21]。如果服务端没有发送服务端保持连接属性，服务端必须使用客户端在 CONNECT 报
	//文中设置的保持连接时间值 [MQTT-3.2.2-22]。包含多个服务端保持连接属性将造成协议错误（Protocol
	//Error）。
	//非规范评注
	//客户端。
	//服务端保持连接属性的主要作用是通知客户端它将会比客户端指定的保持连接更快的断开非活动的
	ServerKeepAlive uint16

	//3.2.2.3.15
	//响应信息
	//26 (0x1A)，响应信息（Response Information）标识符。
	//跟随其后的是一个以 UTF-8 编码的字符串，作为创建响应主题（Response Topic）的基本信息。关于客户
	//端如何根据响应信息（Response Information）创建响应主题不在本规范的定义范围内。包含多个响应信息
	//将造成协议错误（Protocol Error）。
	//如果客户端发送的请求响应信息（Request Response Information）值为 1，则服务端在 CONNACK 报文
	//中发送响应信息（Response Information）为可选项。
	//非规范评注
	//响应信息通常被用来传递主题订阅树的一个全局唯一分支，此分支至少在该客户端的会话生命周期
	//内为该客户端所保留。请求客户端和响应客户端的授权需要使用它，所以它通常不能仅仅是一个随
	//机字符串。一般把此分支作为特定客户端的订阅树根节点。通常此信息需要正确配置，以使得服务
	//器能返回信息。使用此机制时，具体的信息一般由服务端来进行统一配置，而非由各个客户端自己
	//配置。
	ResponseInformation string

	// ServerReference 3.2.2.3.16 28 (0x1C)，服务端参考（Server Reference）标识符。
	//跟随其后的是一个以 UTF-8 编码的字符串，可以被客户端用来标识其他可用的服务端。包含多个服务端参
	//考（Server Reference）将造成协议错误（Protocol Error）。
	//服务端在包含了原因码为 0x9C（（临时）使用其他服务端）或 0x9D（服务端已（永久）移动）的
	//CONNACK 报文或 DISCONNECT 报文中设置服务端参考，如 4.13 节 所述。
	//关于如何使用服务端参考，请参考 4.11 节 服务端重定向信息。
	ServerReference string

	// AuthenticationMethod 3.2.2.3.17 认证方法 21 (0x15)，认证方法（Authentication Method）标识符。
	// 跟随其后的是一个以 UTF-8 编码的字符串，包含了认证方法（Authentication Method）名。包含多个认证
	// 方法将造成协议错误（Protocol Error）。更多关于扩展认证的信息，请参考 4.12 节 。
	AuthenticationMethod string

	// AuthenticationData 22 (0x16) 认证数据 3.2.2.3.18
	// 跟随其后的是包含认证数据（Authentication Data）的二进制数据。
	// 此数据的内容由认证方法和已交换的认证数据状态定义。
	// 包含多个认证数据将造成协议错误（Protocol Error）。
	// 更多关于扩展认证的信息，请参考 4.12 节 。
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
