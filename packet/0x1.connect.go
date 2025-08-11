package packet

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/golang-io/requests"
)

// NAME 协议名，固定为"MQTT"
// MQTT v3.1.1: 参考章节 3.1.2.1 Protocol Name
// MQTT v5.0: 参考章节 3.1.2.1 Protocol Name
// 编码: 0x00 0x04 'M' 'Q' 'T' 'T'
var NAME = []byte{0x00, 0x04, 'M', 'Q', 'T', 'T'}

// CONNECT 客户端连接请求报文
//
// MQTT v3.1.1: 参考章节 3.1 CONNECT - Client requests a connection to a Server
// MQTT v5.0: 参考章节 3.1 CONNECT - Client requests a connection to a Server
//
// 报文结构:
// 固定报头: 报文类型0x01，标志位必须为0
// 可变报头: 协议名、协议级别、连接标志、保持连接
// 载荷: 客户端ID、遗嘱信息(可选)、用户名密码(可选)
//
// 版本差异:
// - v3.1.1: 基本连接功能，支持遗嘱、用户名密码认证
// - v5.0: 在v3.1.1基础上增加了属性系统，支持更多连接选项和扩展认证
type CONNECT struct {
	*FixedHeader

	// 可变报头部分
	// 协议名（Protocol Name）: 节省内存,直接判断,不设置字段存储
	// 协议级别（Protocol Level）: 注意协议级别为了给所有报文打版本信息，将这个字段转移至FixedHeader中

	// ConnectFlags 连接标志，8位标志字段
	// 参考章节: 3.1.2.2 Connect Flags
	// 位置: 可变报头第7字节
	// 标志位定义:
	// - bit 7: UserNameFlag - 用户名标志
	// - bit 6: PasswordFlag - 密码标志
	// - bit 5: WillRetain - 遗嘱保留标志
	// - bit 4-3: WillQoS - 遗嘱QoS等级
	// - bit 2: WillFlag - 遗嘱标志
	// - bit 1: CleanStart - 清理会话标志(v5.0) / CleanSession(v3.1.1)
	// - bit 0: Reserved - 保留位，必须为0
	ConnectFlags ConnectFlags

	// KeepAlive 保持连接时间间隔
	// 参考章节: 3.1.2.10 Keep Alive
	// 位置: 可变报头第8-9字节
	// 单位: 秒
	// 范围: 0-65535
	// 0表示禁用保持连接机制
	// 非0值表示客户端发送PINGREQ的最大时间间隔
	KeepAlive uint16

	// Props 连接属性 (v5.0新增)
	// 参考章节: 3.1.2.11 CONNECT Properties
	// 位置: 可变报头，在保持连接之后
	// 包含各种连接选项，如会话过期间隔、接收最大值等
	Props *ConnectProperties `json:"Properties,omitempty"`

	// 载荷部分
	// 参考章节: 3.1.3 CONNECT Payload

	// ClientID 客户端标识符
	// 参考章节: 3.1.3.1 Client Identifier
	// 位置: 载荷第1个字段
	// 要求: UTF-8编码字符串，长度1-23个字符
	// 特殊值: 空字符串表示服务端自动分配客户端ID
	// 注意: 如果CleanStart=0，客户端ID不能为空
	ClientID string `json:"ClientID,omitempty"`

	// WillProperties 遗嘱属性 (v5.0新增)
	// 参考章节: 3.1.3.2 Will Properties
	// 位置: 载荷中，在客户端ID之后(如果WillFlag=1)
	// 包含遗嘱消息的各种属性，如延时间隔、格式指示等
	WillProperties *WillProperties `json:"Will,omitempty"`

	// WillTopic 遗嘱主题
	// 参考章节: 3.1.3.3 Will Topic
	// 位置: 载荷中，在遗嘱属性之后(如果WillFlag=1)
	// 要求: UTF-8编码字符串，符合主题名规范
	// 用途: 当客户端异常断开时，服务端发布遗嘱消息到此主题
	WillTopic string

	// WillPayload 遗嘱载荷
	// 参考章节: 3.1.3.4 Will Payload
	// 位置: 载荷中，在遗嘱主题之后(如果WillFlag=1)
	// 类型: 二进制数据
	// 用途: 遗嘱消息的内容，当客户端异常断开时发布
	WillPayload []byte

	// Username 用户名
	// 参考章节:
	// - v3.1.1: 3.1.3.4 User Name
	// - v5.0: 3.1.3.5 User Name
	// 位置: 载荷中，在遗嘱信息之后(如果UserNameFlag=1)
	// 要求: UTF-8编码字符串
	// 用途: 服务端用于身份验证和授权
	Username string `json:"Username,omitempty"`

	// Password 密码
	// 参考章节:
	// - v3.1.1: 3.1.3.5 Password
	// - v5.0: 3.1.3.6 Password
	// 位置: 载荷中，在用户名之后(如果PasswordFlag=1)
	// 类型: 二进制数据
	// 用途: 认证信息，尽管称为密码，但可承载任何认证数据
	Password string `json:"Password,omitempty"`
}

func (pkt *CONNECT) Kind() byte {
	return 0x1
}

func (pkt *CONNECT) String() string {
	return "[0x1]CONNECT"
}

// Pack 将CONNECT报文序列化到写入器
// 参考章节: 3.1 CONNECT - Client requests a connection to a Server
// 序列化顺序:
// 1. 固定报头
// 2. 可变报头: 协议名、协议级别、连接标志、保持连接
// 3. 属性(v5.0): 连接属性、遗嘱属性
// 4. 载荷: 客户端ID、遗嘱信息、用户名密码
func (pkt *CONNECT) Pack(w io.Writer) error {
	buf := GetBuffer()
	defer PutBuffer(buf)

	// 写入协议名 "MQTT"
	// 参考章节: 3.1.2.1 Protocol Name
	buf.Write(NAME)

	// 写入协议级别
	// 参考章节: 3.1.2.1 Protocol Level
	buf.WriteByte(pkt.Version)

	// 构建连接标志字节
	// 参考章节: 3.1.2.2 Connect Flags
	uf := s2i(pkt.Username) // UserNameFlag - bit 7
	pf := s2i(pkt.Password) // PasswordFlag - bit 6
	wr := uint8(0)          // WillRetain - bit 5 (遗嘱保留标志)
	wq := uint8(0)          // WillQoS - bits 4-3 (遗嘱QoS等级)
	wf := uint8(0)          // WillFlag - bit 2 (遗嘱标志)
	cs := uint8(0)          // CleanStart/CleanSession - bit 1 (清理会话标志)
	//rs := uint8(0)         // Reserved - bit 0 (保留位，必须为0)

	// 组合标志位
	flag := uf<<7 | pf<<6 | wr<<5 | wq<<3 | wf<<2 | cs<<1
	buf.WriteByte(flag)

	// 写入保持连接时间间隔
	// 参考章节: 3.1.2.10 Keep Alive
	buf.Write(i2b(pkt.KeepAlive))

	// v5.0: 写入连接属性
	if pkt.Version == VERSION500 {
		if pkt.Props == nil {
			pkt.Props = &ConnectProperties{}
		}
		b, err := pkt.Props.Pack()
		if err != nil {
			return err
		}
		buf.Write(b)
	}

	// 写入载荷
	// 参考章节: 3.1.3 CONNECT Payload

	// 客户端ID (必需字段)
	// 参考章节: 3.1.3.1 Client Identifier
	buf.Write(s2b(pkt.ClientID))

	// 遗嘱信息 (如果WillFlag=1)
	// 参考章节: 3.1.3.2 Will Properties, 3.1.3.3 Will Topic, 3.1.3.4 Will Payload
	if pkt.WillProperties != nil {
		// v5.0: 遗嘱属性
		if pkt.Version == VERSION500 {
			b, err := pkt.WillProperties.Pack()
			if err != nil {
				return err
			}
			buf.Write(b)
		}

		// 遗嘱主题
		buf.Write(s2b(pkt.WillTopic))

		// 遗嘱载荷
		buf.Write(s2b(pkt.WillPayload))
	}

	// 用户名 (如果UserNameFlag=1)
	// 参考章节: 3.1.3.4/3.1.3.5 User Name
	if pkt.Username != "" {
		buf.Write(s2b(pkt.Username))
	}

	// 密码 (如果PasswordFlag=1)
	// 参考章节: 3.1.3.5/3.1.3.6 Password
	if pkt.Password != "" {
		buf.Write(s2b(pkt.Password))
	}

	// 设置剩余长度并写入固定报头
	pkt.RemainingLength = uint32(buf.Len())
	return pkt.FixedHeader.Pack(w)
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

// ConnectProperties CONNECT报文可变报头中的属性
// MQTT v5.0新增，参考章节: 3.1.2.11 CONNECT Properties
// 位置: 可变报头，在保持连接之后
// 编码: 属性长度 + 属性标识符 + 属性值
// 注意: 包含多个相同属性将造成协议错误
type ConnectProperties struct {
	// SessionExpiryInterval 会话过期间隔
	// 属性标识符: 17 (0x11)
	// 参考章节: 3.1.2.11.2 Session Expiry Interval
	// 类型: 四字节整数，单位: 秒
	// 含义: 网络连接关闭后会话保持的时间
	// 特殊值:
	// - 0: 会话在网络连接关闭时结束
	// - 0xFFFFFFFF: 会话永不过期
	// 注意: 包含多个会话过期间隔将造成协议错误
	// 如果网络连接关闭时会话过期间隔大于0，则客户端与服务端必须存储会话状态 [MQTT-3.1.2-23]
	SessionExpiryInterval uint32

	// ReceiveMaximum 接收最大值
	// 属性标识符: 33 (0x21)
	// 参考章节: 3.1.2.11.3 Receive Maximum
	// 类型: 双字节整数
	// 含义: 客户端愿意同时处理的QoS等级1和2的发布消息最大数量
	// 默认值: 65535
	// 注意:
	// - 包含多个接收最大值或接收最大值为0将造成协议错误
	// - 没有机制可以限制服务端试图发送的QoS为0的发布消息
	// - 接收最大值只将被应用在当前网络连接
	// 关于接收最大值的详细使用，参考4.9节流控
	ReceiveMaximum uint16

	// MaximumPacketSize 最大报文长度
	// 属性标识符: 39 (0x27)
	// 参考章节: 3.1.2.11.4 Maximum Packet Size
	// 类型: 四字节整数
	// 含义: 客户端愿意接收的最大报文长度
	// 注意:
	// - 包含多个最大报文长度或者最大报文长度为0将造成协议错误
	// - 如果没有设置，则按照协议由固定报头中的剩余长度可编码最大值和协议报头对数据包的大小做限制
	// 非规范评注:
	// - 客户端如果选择了限制最大报文长度，应该为最大报文长度设置一个合理的值
	// - 最大报文长度是MQTT控制报文的总长度
	// - 服务端不能发送超过最大报文长度的报文给客户端 [MQTT-3.1.2-24]
	// - 收到长度超过限制的报文将导致协议错误，客户端发送包含原因码0x95（报文过大）的DISCONNECT报文
	// - 当报文过大而不能发送时，服务端必须丢弃这些报文，然后当做应用消息发送已完成处理 [MQTT-3.1.2-25]
	MaximumPacketSize uint32

	// TopicAliasMaximum 主题别名最大值
	// 属性标识符: 34 (0x22)
	// 参考章节: 3.1.2.11.5 Topic Alias Maximum
	// 类型: 双字节整数
	// 含义: 客户端能够接收的来自服务端的主题别名最大数量
	// 默认值: 0
	// 注意:
	// - 包含多个主题别名最大值将造成协议错误
	// - 没有设置的情况下，主题别名最大值默认为零
	// - 此值指示了客户端能够接收的来自服务端的主题别名最大数量
	// - 客户端使用此值来限制本次连接可以拥有的主题别名的数量
	// - 服务端在一个PUBLISH报文中发送的主题别名不能超过客户端设置的主题别名最大值 [MQTT-3.1.2-26]
	// - 值为零表示本次连接客户端不接受任何主题别名
	// - 如果主题别名最大值没有设置，或者设置为零，则服务端不能向此客户端发送任何主题别名 [MQTT-3.1.2-27]
	TopicAliasMaximum uint16

	// RequestResponseInformation 请求响应信息
	// 属性标识符: 25 (0x19)
	// 参考章节: 3.1.2.11.6 Request Response Information
	// 类型: 单字节，值: 0或1
	// 含义: 客户端向服务端请求CONNACK报文中的响应信息
	// 默认值: 0
	// 注意:
	// - 包含多个请求响应信息，或者值既不为0也不为1会造成协议错误
	// - 值为0: 服务端不能返回响应信息
	// - 值为1: 服务端可以在CONNACK报文中返回响应信息
	// 非规范评注:
	// - 即使客户端请求响应信息，服务端也可以选择不发送响应信息
	// - 更多关于请求/响应信息的内容，请参考4.10节
	RequestResponseInformation uint8

	// RequestProblemInformation 请求问题信息
	// 属性标识符: 23 (0x17)
	// 参考章节: 3.1.2.11.7 Request Problem Information
	// 类型: 单字节，值: 0或1
	// 含义: 客户端指示遇到错误时是否发送原因字符串或用户属性
	// 默认值: 1
	// 注意:
	// - 包含多个请求问题信息，或者值既不为0也不为1会造成协议错误
	// - 如果值为0，服务端可以选择在CONNACK或DISCONNECT报文中返回原因字符串或用户属性，但不能在除PUBLISH、CONNACK或DISCONNECT之外的报文中发送原因字符串或用户属性 [MQTT-3.1.2-29]
	// - 如果此值为0，并且在除PUBLISH、CONNACK或DISCONNECT之外的报文中收到了原因字符串或用户属性，客户端将发送一个包含原因码0x82（协议错误）的DISCONNECT报文给服务端
	RequestProblemInformation uint8

	// UserProperty 用户属性
	// 属性标识符: 38 (0x26)
	// 参考章节: 3.1.2.11.8 User Property
	// 类型: UTF-8字符串对
	// 含义: 用户定义的名称/值对，可以出现多次
	// 注意:
	// - 用户属性可以出现多次，表示多个名字/值对
	// - 相同的名字可以出现多次
	// - 本规范不做定义，由应用程序确定含义和解释
	UserProperty map[string][]string

	// AuthenticationMethod 认证方法
	// 属性标识符: 21 (0x15)
	// 参考章节: 3.1.2.11.9 Authentication Method
	// 类型: UTF-8编码字符串
	// 含义: 扩展认证的认证方法名称
	// 注意:
	// - 包含多个认证方法将造成协议错误
	// - 如果没有认证方法，则不进行扩展验证
	// - 参考4.12节扩展认证
	// - 如果客户端在CONNECT报文中设置了认证方法，则客户端在收到CONNACK报文之前不能发送除AUTH或DISCONNECT之外的报文 [MQTT-3.1.2-30]
	AuthenticationMethod string

	// AuthenticationData 认证数据
	// 属性标识符: 22 (0x16)
	// 参考章节: 3.1.2.11.10 Authentication Data
	// 类型: 二进制数据
	// 含义: 认证数据，内容由认证方法定义
	// 注意:
	// - 没有认证方法却包含了认证数据，或者包含多个认证数据将造成协议错误
	// - 认证数据的内容由认证方法定义
	// - 关于扩展认证的更多信息，请参考4.12节
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

// WillProperties 遗嘱属性
// MQTT v5.0新增，参考章节: 3.1.3.2 Will Properties
// 位置: 载荷中，在客户端ID之后(如果WillFlag=1)
// 编码: 属性长度 + 属性标识符 + 属性值
// 注意: 包含多个相同属性将造成协议错误
type WillProperties struct {
	PropertyLength int32

	// WillDelayInterval 遗嘱延时间隔
	// 属性标识符: 24 (0x18)
	// 参考章节: 3.1.3.2.3 Will Delay Interval
	// 类型: 四字节整数，单位: 秒
	// 含义: 遗嘱消息发布前的延时时间
	// 默认值: 0 (不延时)
	// 注意:
	// - 包含多个遗嘱延时间隔将造成协议错误
	// - 如果没有设置遗嘱延时间隔，遗嘱延时间隔默认值将为0，即不用延时发布遗嘱消息
	// - 服务端将在遗嘱延时间隔到期或者会话结束时发布客户端的遗嘱消息，取决于两者谁先发生
	// - 如果某个会话在遗嘱延时间隔到期之前创建了新的网络连接，则服务端不能发送遗嘱消息
	// 非规范评注:
	// - 遗嘱时间间隔的一个用途是避免在频繁的网络连接临时断开时发布遗嘱消息，因为客户端往往会很快重新连上网络并继续之前的会话
	// - 如果某个连接到服务端的网络连接使用已存在的客户标识符，此已存在的网络连接的遗嘱消息将会被发布，除非新的网络连接设置了新开始（Clean Start）为0并且遗嘱延时大于0
	// - 如果遗嘱延时为0，遗嘱消息将在网络连接断开时发布
	// - 如果新开始为1，遗嘱消息也将被发布，因为此会话已结束
	WillDelayInterval uint32 `json:"WillDelayInterval,omitempty"`

	// PayloadFormatIndicator 载荷格式指示
	// 属性标识符: 1 (0x01)
	// 参考章节: 3.1.3.2.4 Payload Format Indicator
	// 类型: 单字节
	// 值:
	// - 0 (0x00): 表示遗嘱消息是未指定的字节，等同于不发送载荷格式指示
	// - 1 (0x01): 表示遗嘱消息是UTF-8编码的字符数据
	// 注意:
	// - 包含多个载荷格式指示将造成协议错误
	// - 载荷中的UTF-8数据必须按照Unicode规范和RFC 3629中的申明进行编码
	// - 服务端可以按照格式指示对遗嘱消息进行验证，如果验证失败发送一条包含原因码0x99（载荷格式无效）的CONNACK报文
	PayloadFormatIndicator uint8 `json:"PayloadFormatIndicator,omitempty"`

	// MessageExpiryInterval 消息过期间隔
	// 属性标识符: 2 (0x02)
	// 参考章节: 3.1.3.2.5 Message Expiry Interval
	// 类型: 四字节整数，单位: 秒
	// 含义: 遗嘱消息的生命周期
	// 注意:
	// - 包含多个消息过期间隔将导致协议错误
	// - 如果设定了消息过期间隔，四字节整数描述了遗嘱消息的生命周期（秒），并在服务端发布遗嘱消息时被当做发布过期间隔
	// - 如果没有设定消息过期间隔，服务端发布遗嘱消息时将不发送消息过期间隔
	MessageExpiryInterval uint32 `json:"MessageExpiryInterval,omitempty"`

	// ContentType 内容类型
	// 属性标识符: 3 (0x03)
	// 参考章节: 3.1.3.2.6 Content Type
	// 类型: UTF-8格式编码的字符串
	// 含义: 描述遗嘱消息的内容
	// 注意:
	// - 包含多个内容类型将造成协议错误
	// - 内容类型的值由发送应用程序和接收应用程序确定
	ContentType string `json:"ContentType,omitempty"`

	// ResponseTopic 响应主题
	// 属性标识符: 8 (0x08)
	// 参考章节: 3.1.3.2.7 Response Topic
	// 类型: UTF-8格式编码的字符串
	// 含义: 表示响应消息的主题名
	// 注意:
	// - 包含多个响应主题将造成协议错误
	// - 响应主题的存在将遗嘱消息标识为一个请求报文
	ResponseTopic string `json:"ResponseTopic,omitempty"`

	// CorrelationData 对比数据
	// 属性标识符: 9 (0x09)
	// 参考章节: 3.1.3.2.8 Correlation Data
	// 类型: 二进制数据
	// 含义: 被请求消息发送端在收到响应消息时用来标识相应的请求
	// 注意:
	// - 包含多个对比数据将造成协议错误
	// - 如果没有设置对比数据，则请求方不需要任何对比数据
	// - 对比数据只对请求消息的发送端和响应消息的接收端有意义
	// - 更多关于请求/响应的内容，参考4.10节
	CorrelationData []byte `json:"CorrelationData,omitempty"`

	// UserProperty 用户属性
	// 属性标识符: 38 (0x26)
	// 参考章节: 3.1.3.2.9 User Property
	// 类型: UTF-8字符串对
	// 含义: 用户定义的名称/值对，可以出现多次
	// 注意:
	// - 用户属性可以出现多次，表示多个名字/值对
	// - 相同的名字可以出现多次
	// - 服务端在发布遗嘱消息时必须维护用户属性的顺序 [MQTT-3.1.3-10]
	// 非规范评注:
	// - 此属性旨在提供一种传递应用层名称-值标签的方法，其含义和解释仅由负责发送和接收它们的应用程序所有
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

// ConnectFlags 连接标志，8位标志字段
// 参考章节: 3.1.2.2 Connect Flags
// 位置: 可变报头第7字节
// 标志位定义:
// - bit 7: UserNameFlag - 用户名标志
// - bit 6: PasswordFlag - 密码标志
// - bit 5: WillRetain - 遗嘱保留标志
// - bit 4-3: WillQoS - 遗嘱QoS等级
// - bit 2: WillFlag - 遗嘱标志
// - bit 1: CleanStart - 清理会话标志(v5.0) / CleanSession(v3.1.1)
// - bit 0: Reserved - 保留位，必须为0
type ConnectFlags uint8

// Reserved 保留位，位置: bit 0
// 参考章节: 3.1.2.2 Connect Flags
// 值: 必须为0，保留给将来使用
func (f ConnectFlags) Reserved() uint8 {
	return uint8(f) & 0x01
}

// CleanStart 清理会话标志，位置: bit 1
// 参考章节:
// - v3.1.1: 3.1.2.4 Clean Session
// - v5.0: 3.1.2.4 Clean Start
// 值:
// - 0: 使用已存在的会话状态
// - 1: 创建新的会话状态
// 注意: v5.0中此标志的含义略有变化，但基本功能相同
func (f ConnectFlags) CleanStart() bool {
	return (uint8(f) & 0x02) == 0x02
}

// WillFlag 遗嘱标志，位置: bit 2
// 参考章节: 3.1.2.5 Will Flag
// 值:
// - 0: 不发送遗嘱消息
// - 1: 发送遗嘱消息
// 注意: 如果此标志为1，则必须包含遗嘱主题和遗嘱载荷
func (f ConnectFlags) WillFlag() bool {
	return (uint8(f) & 0x04) == 0x04
}

// WillQoS 遗嘱QoS等级，位置: bits 4-3
// 参考章节: 3.1.2.6 Will QoS
// 值:
// - 0x00: QoS 0 - 最多一次传递
// - 0x01: QoS 1 - 至少一次传递
// - 0x02: QoS 2 - 恰好一次传递
// 注意: 只有在WillFlag=1时才有意义
func (f ConnectFlags) WillQoS() uint8 {
	return (uint8(f) & 0x18) >> 3
}

// WillRetain 遗嘱保留标志，位置: bit 5
// 参考章节: 3.1.2.7 Will Retain
// 值:
// - 0: 遗嘱消息不保留
// - 1: 遗嘱消息保留
// 注意: 只有在WillFlag=1时才有意义
func (f ConnectFlags) WillRetain() bool {
	return (uint8(f) & 0x20) == 0x20
}

// UserNameFlag 用户名标志，位置: bit 7
// 参考章节: 3.1.2.8 User Name Flag
// 值:
// - 0: 载荷中不包含用户名
// - 1: 载荷中包含用户名
// 注意: 如果此标志为1，则载荷中必须包含用户名字段
func (f ConnectFlags) UserNameFlag() bool {
	return (uint8(f) & 0x80) == 0x80
}

// PasswordFlag 密码标志，位置: bit 6
// 参考章节: 3.1.2.9 Password Flag
// 值:
// - 0: 载荷中不包含密码
// - 1: 载荷中包含密码
// 注意:
// - 如果此标志为1，则载荷中必须包含密码字段
// - 如果UserNameFlag=0，则PasswordFlag必须为0
func (f ConnectFlags) PasswordFlag() bool {
	return (uint8(f) & 0x40) == 0x40
}
