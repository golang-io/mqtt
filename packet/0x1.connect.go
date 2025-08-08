package packet

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/golang-io/requests"
)

var NAME = []byte{0x00, 0x04, 'M', 'Q', 'T', 'T'}

type CONNECT struct {
	*FixedHeader

	// 可变报头
	// 协议名（Protocol Name）: 节省内存,直接判断,不设置字段存储

	// 协议级别（Protocol Level）: 注意协议级别为了给所有报文打版本信息，将这个字段转移至FixedHeader中

	// uint8 连接标志（Connect Flags）: 也是在报文处理的时候使用，不设置存储字段
	ConnectFlags ConnectFlags
	//  UserNameFlag bit7
	//  PasswordFlag byt6
	//	WillRetain bit5
	//	WillQoS bit4-3
	//	WillFlag bit2
	//	CleanStart bit1
	//	Reserved bit0

	// 保持连接（Keep Alive）
	KeepAlive uint16

	// 属性（Properties）
	Props *ConnectProperties `json:"Properties,omitempty"`

	// Connect 载荷
	// CONNECT Payload
	ClientID string `json:"ClientID,omitempty"` // Client 客户端ID

	// 这个是v5独有的
	WillProperties *WillProperties `json:"Will,omitempty"`

	// 3.1.3.3 遗嘱主题
	//如果遗嘱标志（Will Flag）被设置为 1，遗嘱主题（Will Topic）为载荷中下一个字段。遗嘱主题（Will
	//1.5.4 节 所定义 [MQTT-3.1.3-11]。
	WillTopic string

	// 3.1.3.4 遗嘱载荷
	//如果遗嘱标志（Will Flag）被设置为 1，遗嘱载荷（Will Payload）为载荷中下一个字段。遗嘱载荷定义了
	//将要发布到遗嘱主题（Will Topic）的应用消息载荷，如 3.1.2.5 节 所定义。此字段为二进制数据。
	WillPayload []byte
	// 3.1.3.5 用户名
	//如果用户名标志（User Name Flag）被设置为 1，用户名（User Name）为载荷中下一个字段。用户名必
	//须是 1.5.4 节 定义的 UTF-8 编码字符串 [MQTT-3.1.3-12]。服务端可以将它用于身份验证和授权。
	Username string `json:"Username,omitempty"` // v311: 3.1.3.4 	v500: 3.1.3.5

	// 3.1.3.6 密码
	//如果密码标志（Password Flag）被设置为 1，密码（Password）为载荷中下一个字段。密码字段是二进
	//制数据，尽管被称为密码，但可以被用来承载任何认证信息。
	Password string `json:"Password,omitempty"` // v311: 3.1.3.5	v500: 3.1.3.6
}

func (pkt *CONNECT) Kind() byte {
	return 0x1
}

func (pkt *CONNECT) String() string {
	return "[0x1]CONNECT"
}

func (pkt *CONNECT) Pack(w io.Writer) error {
	buf := GetBuffer()
	defer PutBuffer(buf)

	buf.Write(NAME)
	buf.WriteByte(pkt.Version)

	uf := s2i(pkt.Username) // UserNameFlag
	pf := s2i(pkt.Password) // PasswordFlag
	wr := uint8(0)          //pkt.Will.Retain   //  WillRetain
	wq := uint8(0)          //pkt.Will.QoS      // WillQoS
	wf := uint8(0)          //pkt.Will.Retain   // WillFlag
	cs := uint8(0)          //pkt.CleanStart    // CleanSession
	//rs := uint8(0)             // Reserved

	flag := uf<<7 | pf<<6 | wr<<5 | wq<<3 | wf<<2 | cs<<1

	buf.WriteByte(flag)
	buf.Write(i2b(pkt.KeepAlive)) // 9-10

	if pkt.Version == VERSION500 {
		if pkt.Props == nil {
			pkt.Props = &ConnectProperties{}
		}
		b, err := pkt.Props.Pack()
		if err != nil {
			return err
		}
		propsLen := uint32(len(b))
		propsPktLen, err := encodeLength(propsLen)
		if err != nil {
			return err
		}

		// remainLength|propsLen|propsContent|payload[connack is null]
		pkt.RemainingLength += propsLen + uint32(len(propsPktLen)) // 注意! 属性长度也是需要计算在也会占用可变报头的空间, 也需要计算!

		buf.Write(propsPktLen)
		buf.Write(b)
	}

	buf.Write(s2b(pkt.ClientID))

	if pkt.ConnectFlags.WillFlag() {
		if pkt.Version == VERSION500 {
			if pkt.Props == nil {
				pkt.WillProperties = &WillProperties{}
			}
			b, err := pkt.WillProperties.Pack()
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
		buf.Write(encodeUTF8(pkt.WillTopic))
		buf.Write(encodeUTF8(pkt.WillPayload))

	}

	if wf == 1 {
		buf.Write(s2b(pkt.WillTopic))
		buf.Write(s2b(pkt.WillPayload))
	}
	if uf == 1 {
		buf.Write(s2b(pkt.Username))
	}
	if pf == 1 {
		buf.Write(s2b(pkt.Password))
	}
	pkt.FixedHeader.RemainingLength = uint32(buf.Len())
	if err := pkt.FixedHeader.Pack(w); err != nil {
		return err
	}
	_, err := buf.WriteTo(w)
	return err
}

func (pkt *CONNECT) Unpack(buf *bytes.Buffer) error {

	name := buf.Next(6)
	if !bytes.Equal(name, NAME) {
		return fmt.Errorf("%w: Len=%d, %v", ErrMalformedProtocolName, pkt.RemainingLength, name)
	}

	pkt.Version, pkt.ConnectFlags = buf.Next(1)[0], ConnectFlags(buf.Next(1)[0])

	// The Server MUST validate that the reserved flag in the CONNECT Control Packet is set to zero and
	// disconnect the Client if it is not zero [MQTT-3.1.2-3].
	if pkt.ConnectFlags.Reserved() != 0 {
		return ErrMalformedPacket
	}

	if pkt.ConnectFlags.WillQoS() > 2 {
		// 如果遗嘱标志被设置为 1，遗嘱 QoS 的值可以等于 0(0x00)，1(0x01)，2(0x02)。它的值不能等于 3 [MQTT-3.1.2-14]。
		return ErrProtocolViolationQosOutOfRange
	}

	pkt.KeepAlive = binary.BigEndian.Uint16(buf.Next(2))

	switch pkt.Version {
	case VERSION500:
		pkt.Props = &ConnectProperties{}
		if err := pkt.Props.Unpack(buf); err != nil {
			return err
		}
	case VERSION311:
	case VERSION310:
		return ErrUnsupportedProtocolVersion
	default:
		return ErrMalformedProtocolVersion

	}

	pkt.ClientID = decodeUTF8[string](buf)
	if pkt.ClientID == "" {
		pkt.ClientID = requests.GenId()
	}

	// 如果遗嘱标志被设置为 0，遗嘱保留（Will Retain）标志也必须设置为 0 [MQTT-3.1.2-15]。
	// TODO: 如果遗嘱标志被设置为 0，连接标志中的Will QoS 和 Will Retain 字段必须设置为 0，并且有效载荷中不能包含Will Topic 和Will Message 字段 [MQTT-3.1.2-11]。
	// TODO: 如果遗嘱标志被设置为 1，连接标志中的Will QoS 和 Will Retain 字段会被服务端用到，同时有效载荷中必须包含Will Topic 和Will Message 字段 [MQTT-3.1.2-9]。
	if pkt.ConnectFlags.WillFlag() {
		if pkt.Version == VERSION500 {
			pkt.WillProperties = &WillProperties{}
			if err := pkt.WillProperties.Unpack(buf); err != nil {
				return err
			}
		}
		pkt.WillTopic = decodeUTF8[string](buf)
		pkt.WillPayload = decodeUTF8[[]byte](buf)
		//pkt.Will.Retain, pkt.Will.QoS = wr, wq
	}

	if pkt.ConnectFlags.UserNameFlag() {
		// 如果用户名（User Name）标志被设置为 1，有效载荷中必须包含用户名字段 [MQTT-3.1.2-19]。
		pkt.Username = decodeUTF8[string](buf)
	} else {
		// 如果用户名（User Name）标志被设置为 0，有效载荷中不能包含用户名字段 [MQTT-3.1.2-18]。
		// 如果用户名标志被设置为 0，密码标志也必须设置为 0 [MQTT-3.1.2-22]。
		if pkt.ConnectFlags.PasswordFlag() {
			return ErrMalformedPassword
		}
	}
	if pkt.ConnectFlags.PasswordFlag() {
		//如果密码（Password）标志被设置为 0，有效载荷中不能包含密码字段 [MQTT-3.1.2-20]。
		pkt.Password = decodeUTF8[string](buf)
	} else {
		//如果密码（Password）标志被设置为 0，有效载荷中不能包含密码字段 [MQTT-3.1.2-20]。
	}
	return nil
}

type Will struct {
	TopicName string
	Message   []byte
	Retain    uint8 // 保留标志
	QoS       uint8 // 服务质量
}

type ConnectProperties struct {
	// CONNECT 报文可变报头中的属性（Properties）长度被编码为变长字节整数。
	//PropertyLength int
	// 17 (0x11)，会话过期间隔（Session Expiry Interval）标识符。
	// 跟随其后的是用四字节整数表示的以秒为单位的会话过期间隔（Session Expiry Interval）。
	//包含多个会话过期间隔（Session Expiry Interval）将造成协议错误（Protocol Error）。
	//如果会话过期间隔（Session Expiry Interval）值未指定，则使用 0。
	//如果设置为 0 或者未指定，会话将在网络连接（Network Connection）关闭时结束。
	//如果会话过期间隔（Session Expiry Interval）为 0xFFFFFFFF (UINT_MAX)，则会话永不过期。
	//如果网络连接关闭时会话过期间隔（Session Expiry Interval）大于 0，则客户端与服务端必须存储会话状态 [MQTT-3.1.2-23]。
	SessionExpiryInterval uint32
	// 33 (0x21)，接收最大值（Receive Maximum）标识符。
	//跟随其后的是由双字节整数表示的最大接收值。包含多个接收最大值或接收最大值为 0 将造成协议错误
	//（Protocol Error）。
	//客户端使用此值限制客户端愿意同时处理的 QoS 等级 1 和 QoS 等级 2 的发布消息最大数量。没有机制可
	//以限制服务端试图发送的 QoS 为 0 的发布消息。
	//接收最大值只将被应用在当前网络连接。如果没有设置最大接收值，将使用默认值 65535。
	//关于接收最大值的详细使用，参考 4.9 节 流控。
	ReceiveMaximum uint16
	// 39 (0x27)，最大报文长度（Maximum Packet Size）标识符。
	//跟随其后的是由四字节整数表示的客户端愿意接收的最大报文长度（Maximum Packet Size），如果没有
	//设置最大报文长度（Maximum Packet Size），则按照协议由固定报头中的剩余长度可编码最大值和协议
	//报头对数据包的大小做限制。
	//包含多个最大报文长度（Maximum Packet Size）或者最大报文长度（Maximum Packet Size）值为 0 将
	//造成协议错误。
	//非规范评注
	//客户端如果选择了限制最大报文长度，应该为最大报文长度设置一个合理的值。
	//如 2.1.4 节 所述，最大报文长度是 MQTT 控制报文的总长度。客户端使用最大报文长度通知服务端其所能
	//处理的单个报文长度限制。
	// 服务端不能发送超过最大报文长度（Maximum Packet Size）的报文给客户端 [MQTT-3.1.2-24]。收到长度
	//超过限制的报文将导致协议错误，客户端发送包含原因码 0x95（报文过大）的 DISCONNECT 报文给服务
	//端，详见 4.13 节。
	//当报文过大而不能发送时，服务端必须丢弃这些报文，然后当做应用消息发送已完成处理 [MQTT-3.1.2-25]。
	//共享订阅的情况下，如果一条消息对于部分客户端来说太长而不能发送，服务端可以选择丢弃此消息或者
	//把消息发送给剩余能够接收此消息的客户端。
	//非规范评注
	//服务端可以把那些没有发送就被丢弃的报文放在死信队列 上，或者执行其他诊断操作。具体的操
	//作超出了本规范的范围。
	MaximumPacketSize uint32

	// 34 (0x22)，主题别名最大值（Topic Alias Maximum）标识符。
	//跟随其后的是用双字节整数表示的主题别名最大值（Topic Alias Maximum）。包含多个主题别名最大值
	//（Topic Alias Maximum）将造成协议错误（Protocol Error）。没有设置主题别名最大值属性的情况下，主
	//题别名最大值默认为零。
	//此值指示了客户端能够接收的来自服务端的主题别名（Topic Alias）最大数量。客户端使用此值来限制本
	//次连接可以拥有的主题别名的数量。服务端在一个 PUBLISH 报文中发送的主题别名不能超过客户端设置的
	//主题别名最大值（Topic Alias Maximum） [MQTT-3.1.2-26]。值为零表示本次连接客户端不接受任何主题
	//别名（Topic Alias）。如果主题别名最大值（Topic Alias）没有设置，或者设置为零，则服务端不能向此客
	//户端发送任何主题别名（Topic Alias） [MQTT-3.1.2-27]。
	TopicAliasMaximum uint16

	// 25 (0x19)，请求响应信息（Request Response Information）标识符。
	//跟随其后的是用一个字节表示的 0 或 1。包含多个请求响应信息（Request Response Information），或者
	//请求响应信息（Request Response Information）的值既不为 0 也不为 1 会造成协议错误（Protocol
	//Error）。如果没有请求响应信息（Request Response Information），则请求响应默认值为 0。
	//客户端使用此值向服务端请求 CONNACK 报文中的响应信息（Response Information）。值为 0，表示服
	//务端不能返回响应信息 [MQTT-3.1.2-28]。值为 1，表示服务端可以在 CONNACK 报文中返回响应信息。
	//非规范评注
	//即使客户端请求响应信息（Response Information），服务端也可以选择不发送响应信息
	//（Response Information）。
	//更多关于请求/响应信息的内容，请参考 4.10 节。
	RequestProblemInformation uint8
	// 23 (0x17)，请求问题信息（Request Problem Information）标识符。
	//跟随其后的是用一个字节表示的 0 或 1。包含多个请求问题信息（Request Problem Information），或者
	//请求问题信息（Request Problem Information）的值既不为 0 也不为 1 会造成协议错误（Protocol Error）。
	//如果没有请求问题信息（Request Problem Information），则请求问题默认值为 1。
	//客户端使用此值指示遇到错误时是否发送原因字符串（Reason String）或用户属性（User Properties）。
	//如果请求问题信息的值为 0，服务端可以选择在 CONNACK 或 DISCONNECT 报文中返回原因字符串
	//（Reason String）或用户属性（User Properties），但不能在除 PUBLISH，CONNACK 或
	//DISCONNECT 之外的报文中发送原因字符串（Reason String）或用户属性（User Properties） [MQTT-
	//3.1.2-29]。如果此值为 0，并且在除 PUBLISH，CONNACK 或 DISCONNECT 之外的报文中收到了原因字
	//符串（Reason String）或用户属性（User Properties），客户端将发送一个包含原因码 0x82（协议错误）
	//的 DISCONNECT 报文给服务端，如 4.13 节 所述。
	//如果此值为 1，服
	RequestResponseInformation uint8
	// 38 (0x26)，用户属性（User Property）标识符。
	//跟随其后的是 UTF-8 字符串对。
	//用户属性（User Property）可以出现多次，表示多个名字/值对。相同的名字可以出现多次。
	//非规范评注
	//本规范不做定义。
	//CONNECT 报文中的
	UserProperty map[string][]string

	// 3.1.2.11.9
	//认证方法
	//21 (0x15)，认证方法（Authentication Method）标识符。
	//跟随其后的是一个 UTF-8 编码的字符串，包含了扩展认证的认证方法（Authentication Method）名称。包
	//含多个认证方法将造成协议错误（协议错误）。
	//如果没有认证方法，则不进行扩展验证。参考 4.12 节。
	//如果客户端在 CONNECT 报文中设置了认证方法，则客户端在收到 CONNACK 报文之前不能发送除
	//AUTH 或 DISCONNECT 之外的报文 [MQTT-3.1.2-30]。
	AuthenticationMethod string

	// 3.1.2.11.10
	//认证数据
	//22 (0x16)，认证数据（Authentication Data）标识符。
	//跟随其后的是二进制的认证数据。没有认证方法却包含了认证数据（Authentication Data），或者包含多个
	//认证数据（Authentication Data）将造成协议错误（Protocol Error）。
	//认证数据的内容由认证方法定义，关于扩展认证的更多信息，请参考 4.12 节。
	AuthenticationData []byte
}

func (props *ConnectProperties) Pack() ([]byte, error) {
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
	if props.MaximumPacketSize != 0 {
		buf.WriteByte(0x27)
		buf.Write(i4b(props.MaximumPacketSize))
	}

	if props.TopicAliasMaximum != 0 {
		buf.WriteByte(0x22)
		buf.Write(i2b(props.TopicAliasMaximum))
	}
	if props.RequestResponseInformation != 0 {
		buf.WriteByte(0x19)
		buf.WriteByte(props.RequestResponseInformation)
	}
	if props.RequestProblemInformation != 0 {
		buf.WriteByte(0x17)
		buf.WriteByte(props.RequestProblemInformation)
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
	if props.AuthenticationMethod != "" {
		buf.WriteByte(0x15)
		buf.Write(encodeUTF8(props.AuthenticationMethod))
	}
	if props.AuthenticationData != nil {
		buf.WriteByte(0x16)
		buf.Write(encodeUTF8(props.AuthenticationData))
	}
	return buf.Bytes(), nil

}

func (props *ConnectProperties) Unpack(buf *bytes.Buffer) error {
	propsLen, err := decodeLength(buf)
	if err != nil {
		return err
	}
	for i := uint32(0); i < propsLen; i++ {
		propsCode, err := decodeLength(buf)
		if err != nil {
			return err
		}
		switch propsCode {
		case 0x11:
			if props.SessionExpiryInterval != 0 {
				return ErrProtocolErr
			}
			props.SessionExpiryInterval, i = binary.BigEndian.Uint32(buf.Next(4)), i+4
		case 0x21:
			// 包含多个接收最大值将造成协议错误
			if props.ReceiveMaximum != 0 {
				return ErrProtocolErr
			}
			props.ReceiveMaximum, i = binary.BigEndian.Uint16(buf.Next(2)), i+2
			// 接收最大值为 0 将造成协议错误
			if props.ReceiveMaximum == 0 {
				return ErrProtocolErr
			}
		case MaximumPacketSize:
			if props.MaximumPacketSize != 0 {
				return ErrProtocolErr
			}
			props.MaximumPacketSize, i = binary.BigEndian.Uint32(buf.Next(4)), i+4
			if props.MaximumPacketSize == 0 {
				return ErrProtocolErr
			}
		case TopicAliasMaximum:
			if props.TopicAliasMaximum != 0 {
				return ErrProtocolErr
			}
			props.TopicAliasMaximum, i = binary.BigEndian.Uint16(buf.Next(2)), i+2
			if props.TopicAliasMaximum == 0 {
				return ErrProtocolErr
			}
		case 0x19:
			// TODO:
			props.RequestResponseInformation, i = buf.Next(1)[0], i+1
			if props.RequestResponseInformation != 0 && props.RequestResponseInformation != 1 {
				return ErrProtocolErr
			}
		case 0x17:
			props.RequestProblemInformation, i = buf.Next(1)[0], i+1
		case 0x26:
			if props.UserProperty == nil {
				props.UserProperty = make(map[string][]string)
			}
			key := decodeUTF8[string](buf)
			props.UserProperty[key] = append(props.UserProperty[key], decodeUTF8[string](buf))
		case 0x15:
			authMethod := decodeUTF8[string](buf)
			props.AuthenticationMethod, i = authMethod, i+uint32(len(authMethod))
		case 0x16:
			authData := decodeUTF8[[]byte](buf)
			props.AuthenticationData, i = authData, i+uint32(len(authData))
		default:
			return ErrMalformedProperties
		}
	}
	return nil
}

const TopicAliasMaximum = 0x22
const MaximumPacketSize = 0x27

type WillProperties struct {
	PropertyLength int32

	// 24 (0x18)，遗嘱延时间隔（Will Delay Interval）标识符。
	// 24 (0x18)，遗嘱延时间隔（Will Delay Interval）标识符。
	//跟随其后的是由四字节整数表示的以秒为单位的遗嘱延时间隔（Will Delay Interval）。包含多个遗嘱延时
	//间隔将造成协议错误（Protocol Error）。如果没有设置遗嘱延时间隔，遗嘱延时间隔默认值将为 0，即不
	//用延时发布遗嘱消息（Will Message）。
	//服务端将在遗嘱延时间隔（Will Delay Interval）到期或者会话（Session）结束时发布客户端的遗嘱消息
	//（Will Message），取决于两者谁先发生。如果某个会话在遗嘱延时间隔到期之前创建了新的网络连接，
	//则服务端不能发送遗嘱消息 [MQTT-3.1.3-9]。
	//非规范评注
	//遗嘱时间间隔的一个用途是避免在频繁的网络连接临时断开时发布遗嘱消息，因为客户端往往会很
	//快重新连上网络并继续之前的会话。
	//非规范评注
	//如果某个连接到服务端的网络连接使用已存在的客户标识符，此已存在的网络连接的遗嘱消息将会
	//被发布，除非新的网络连接设置了新开始（Clean Start）为 0 并且遗嘱延时大于 0。如果遗嘱延时
	//为 0，遗嘱消息将在网络连接断开时发布。如果新开始为 1，遗嘱消息也将被发布，因为此会话已
	//结束。
	WillDelayInterval uint32 `json:"WillDelayInterval,omitempty"`
	// 3.1.3.2.3
	//载荷格式指示
	//1 (0x01)，载荷格式指示（Payload Format Indicator）标识符。
	//跟随载荷格式指示（Payload Format Indicator ）之后的可能是：
	// 0 (0x00)，表示遗嘱消息（Will Message）是未指定的字节，等同于不发送载荷格式指示。
	// 1 (0x01)，表示遗嘱消息（Will Message）是 UTF-8 编码的字符数据。载荷中的 UTF-8 数据必须
	//按照 Unicode 规范[Unicode] 和 RFC 3629 [RFC3629]中的申明进行编码。
	//包含多个载荷格式指示（Payload Format Indicator）将造成协议错误（Protocol Error）。服务端可以按照
	//格式指示对遗嘱消息（Will Message）进行验证，如果验证失败发送一条包含原因码 0x99（载荷格式无效）
	//的 CONNACK 报文。如 4.13 节 所述。
	PayloadFormatIndicator uint8 `json:"PayloadFormatIndicator,omitempty"`
	// 3.1.3.2.4
	//消息过期间隔
	//2 (0x02)，消息过期间隔（Message Expiry Interval）标识符。
	//跟随其后的是表示消息过期间隔（Message Expiry Interval）的四字节整数。包含多个消息过期间隔将导致
	//协议错误（Protocol Error）。
	//如果设定了消息过期间隔（Message Expiry Interval），四字节整数描述了遗嘱消息的生命周期（秒），并
	//在服务端发布遗嘱消息时被当做发布过期间隔（Publication Expiry Interval）。
	//如果没有设定消息过期间隔，服务端发布遗嘱消息时将不发送消息过期间隔（Message Expiry Interval）。
	//3.1.3.
	MessageExpiryInterval uint32 `json:"MessageExpiryInterval,omitempty"`
	// 3.1.3.2.5
	//内容类型
	//3 (0x03)，内容类型（Content Type）标识符。
	//跟随其后的是一个以 UTF-8 格式编码的字符串，用来描述遗嘱消息（Will Message）的内容。包含多个内
	//容类型（Content Type）将造成协议错误（Protocol Error）。内容类型的值由发送应用程序和接收应用程
	//序确定。
	ContentType string `json:"ContentType,omitempty"`
	// 3.1.3.2.6
	//响应主题
	//8 (0x08)，响应主题（Response Topic）标识符。
	//跟随其后的是一个以 UTF-8 格式编码的字符串，用来表示响应消息的主题名（Topic Name）。包含多个响
	//应主题（Response Topic）将造成协议错误。响应主题的存在将遗嘱消息（Will Message）标识为一个请
	//求报文。
	ResponseTopic string `json:"ResponseTopic,omitempty"`
	// 9 (0x09)，对比数据（Correlation Data）标识符。
	//跟随其后的是二进制数据。对比数据被请求消息发送端在收到响应消息时用来标识相应的请求。包含多个
	//对比数据将造成协议错误（Protocol Error）。如果没有设置对比数据，则请求方（Requester）不需要任何
	//对比数据。
	//对比数据只对请求消息（Request Message）的发送端和响应消息（Response Message）的接收端有意
	//义。
	//更多关于请求/响应的内容，参考 4.10 节。
	CorrelationData []byte `json:"CorrelationData,omitempty"`
	//38 (0x26)，用户属性（User Property）标识符。
	//跟随其后的是一个 UTF-8 字符串对。用户属性（User Property）可以出现多次，表示多个名字/值对。相同
	//的名字可以出现多次。
	//服务端在发布遗嘱消息（Will Message）时必须维护用户属性（User Properties）的顺序 [MQTT-3.1.3-10]。
	//非规范评注
	// 此属性旨在提供一种传递应用层名称-值标签的方法，其含义和解释仅由负责发送和接收它们的应
	//用程序所有。
	UserProperty []byte
}

func (props *WillProperties) Pack() ([]byte, error) {
	buf := GetBuffer()
	defer PutBuffer(buf)

	if props.PayloadFormatIndicator != 0 {
		buf.WriteByte(0x01)
		buf.WriteByte(props.PayloadFormatIndicator)
	}
	if props.MessageExpiryInterval != 0 {
		buf.WriteByte(0x02)
		buf.Write(i4b(props.MessageExpiryInterval))
	}
	if props.ContentType != "" {
		buf.WriteByte(0x03)
		buf.Write(encodeUTF8(props.ContentType))
	}
	if props.ResponseTopic != "" {
		buf.WriteByte(0x08)
		buf.Write(encodeUTF8(props.ResponseTopic))
	}
	if props.CorrelationData != nil {
		buf.WriteByte(0x09)
		buf.Write(encodeUTF8(props.CorrelationData))
	}
	if props.WillDelayInterval != 0 {
		buf.WriteByte(0x18)
		buf.Write(i4b(props.WillDelayInterval))
	}
	return buf.Bytes(), nil
}

func (props *WillProperties) Unpack(b *bytes.Buffer) error {
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
		case 0x01: // 会话过期间隔 Session Expiry Interval
			props.PayloadFormatIndicator, i = b.Next(1)[0], i+1
		case 0x02:
			props.MessageExpiryInterval, i = binary.BigEndian.Uint32(b.Next(4)), i+4
		case 0x03: // 最大报文长度 Maximum Packet Size
			props.ContentType, i = decodeUTF8[string](b), i+uint32(len(props.ContentType))

		case 0x08: // 主题别名最大值 Topic Alias Maximum
			props.ResponseTopic, i = decodeUTF8[string](b), i+uint32(len(props.ResponseTopic))

		case 0x09:
			props.CorrelationData, i = decodeUTF8[[]byte](b), i+uint32(len(props.CorrelationData))

		case 0x18:
			props.WillDelayInterval, i = binary.BigEndian.Uint32(b.Next(4)), i+4
		default:
			return ErrMalformedWillProperties
		}
	}
	return nil
}

type ConnectFlags uint8

func (f ConnectFlags) Reserved() uint8 {
	return uint8(f) & 0b00000001 >> 0
}
func (f ConnectFlags) CleanStart() bool {
	return uint8(f)&0b00000010>>1 == 1
}
func (f ConnectFlags) WillFlag() bool {
	return uint8(f)&0b00000100>>2 == 1
}

func (f ConnectFlags) WillQoS() uint8 {
	return uint8(f) & 0b00011000 >> 3
}
func (f ConnectFlags) WillRetain() bool {
	return uint8(f)&0b00100000>>5 == 1
}
func (f ConnectFlags) UserNameFlag() bool {
	return uint8(f)&0b01000000>>6 == 1
}
func (f ConnectFlags) PasswordFlag() bool {
	return uint8(f)&0b10000000>>7 == 1
}
