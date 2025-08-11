package packet

import (
	"bytes"
	"io"
)

// Packet 定义了MQTT控制报文的通用接口
//
// MQTT v3.1.1 (OASIS Standard, 29 October 2014):
// - 参考章节: 2.1 Structure of an MQTT Control Packet
// - 每个MQTT控制报文都包含固定报头和可变报头，某些报文还包含载荷
//
// MQTT v5.0 (OASIS Standard, 7 March 2019):
// - 参考章节: 2.1 Structure of an MQTT Control Packet
// - 在v3.1.1基础上增加了属性(Properties)系统，提供更丰富的元数据支持
// - 属性系统允许在报文中携带额外的控制信息，如用户属性、原因码等
type Packet interface {
	// Kind 返回报文的类型标识符
	//
	// MQTT v3.1.1: 参考章节 2.2.1 MQTT Control Packet type
	// - 位置: 固定报头第1字节的bits 7-4
	// - 范围: 0x01-0x0E (CONNECT到DISCONNECT)
	// - 0x0F为保留值，v3.1.1中不使用
	//
	// MQTT v5.0: 参考章节 2.2.1 MQTT Control Packet type
	// - 位置: 固定报头第1字节的bits 7-4
	// - 范围: 0x01-0x0F (CONNECT到AUTH)
	// - 新增AUTH报文(0x0F)用于扩展认证流程
	Kind() byte

	// Unpack 从缓冲区解析报文内容
	//
	// MQTT v3.1.1: 参考章节 2.1 Structure of an MQTT Control Packet
	// - 解析顺序: 固定报头 -> 可变报头 -> 载荷(如果有)
	// - 固定报头: 报文类型、标志位、剩余长度
	// - 可变报头: 报文标识符(某些报文类型)
	// - 载荷: 应用数据或控制信息
	//
	// MQTT v5.0: 参考章节 2.1 Structure of an MQTT Control Packet
	// - 解析顺序: 固定报头 -> 可变报头 -> 属性 -> 载荷(如果有)
	// - 新增属性解析: 属性长度 + 属性标识符 + 属性值
	// - 属性系统提供更丰富的元数据和控制能力
	Unpack(*bytes.Buffer) error

	// Pack 将报文序列化到写入器
	//
	// MQTT v3.1.1: 参考章节 2.1 Structure of an MQTT Control Packet
	// - 序列化顺序: 固定报头 -> 可变报头 -> 载荷(如果有)
	// - 固定报头: 报文类型、标志位、剩余长度
	// - 可变报头: 报文标识符(某些报文类型)
	// - 载荷: 应用数据或控制信息
	//
	// MQTT v5.0: 参考章节 2.1 Structure of an MQTT Control Packet
	// - 序列化顺序: 固定报头 -> 可变报头 -> 属性 -> 载荷(如果有)
	// - 新增属性序列化: 属性长度 + 属性标识符 + 属性值
	// - 属性系统支持更复杂的控制流程
	Pack(io.Writer) error
}

// Unpack 从读取器解析MQTT控制报文
//
// version: MQTT协议版本，用于确定报文格式和字段
// - v3.1.1: 协议级别4，参考章节 3.1.2.1 Protocol Level
// - v5.0: 协议级别5，参考章节 3.1.2.1 Protocol Level
//
// 解析流程参考章节 2.1 Structure of an MQTT Control Packet:
// 1. 解析固定报头获取报文类型和剩余长度
// 2. 根据报文类型创建对应的报文结构
// 3. 解析可变报头和载荷内容
func Unpack(version byte, r io.Reader) (Packet, error) {
	pkt, fixed := Packet(nil), &FixedHeader{Version: version}
	if err := fixed.Unpack(r); err != nil {
		return &RESERVED{FixedHeader: fixed}, err
	}

	buf := GetBuffer()
	defer PutBuffer(buf)

	lr := io.LimitReader(r, int64(fixed.RemainingLength))

	if _, err := buf.ReadFrom(lr); err != nil {
		return pkt, err
	}

	// 根据报文类型创建对应的报文结构
	// MQTT v3.1.1和v5.0的报文类型定义相同，但v5.0增加了属性支持
	switch fixed.Kind {
	case 0x1: // CONNECT - 客户端连接请求
		// MQTT v3.1.1: 参考章节 3.1 CONNECT - Client requests a connection to a Server
		// - 固定报头: 报文类型0x01，标志位必须为0
		// - 可变报头: 协议名、协议级别、连接标志、保持连接
		// - 载荷: 客户端ID、遗嘱信息(可选)、用户名密码(可选)
		//
		// MQTT v5.0: 参考章节 3.1 CONNECT - Client requests a connection to a Server
		// - 在v3.1.1基础上增加了属性(Properties)字段
		// - 新增属性: 会话过期间隔、接收最大值、最大报文长度、主题别名最大值等
		// - 支持扩展认证方法和认证数据
		pkt = &CONNECT{FixedHeader: fixed}
	case 0x2: // CONNACK - 连接确认
		// MQTT v3.1.1: 参考章节 3.2 CONNACK - Acknowledge connection request
		// - 固定报头: 报文类型0x02，标志位必须为0
		// - 可变报头: 连接确认标志、连接返回码
		// - 无载荷
		//
		// MQTT v5.0: 参考章节 3.2 CONNACK - Acknowledge connection request
		// - 在v3.1.1基础上增加了属性(Properties)和原因码
		// - 新增属性: 会话过期间隔、接收最大值、最大QoS、保留可用等
		// - 支持更详细的连接状态反馈
		pkt = &CONNACK{FixedHeader: fixed}
	case 0x3: // PUBLISH - 发布消息
		// MQTT v3.1.1: 参考章节 3.3 PUBLISH - Publish message
		// - 固定报头: 报文类型0x03，标志位包含DUP、QoS、RETAIN
		// - 可变报头: 主题名、报文标识符(QoS>0时)
		// - 载荷: 应用消息
		//
		// MQTT v5.0: 参考章节 3.3 PUBLISH - Publish message
		// - 在v3.1.1基础上增加了属性(Properties)
		// - 新增属性: 载荷格式指示、消息过期间隔、主题别名、订阅标识符等
		// - 支持主题别名优化和消息生命周期管理
		pkt = &PUBLISH{FixedHeader: fixed}
	case 0x4: // PUBACK - 发布确认(QoS 1)
		// MQTT v3.1.1: 参考章节 3.4 PUBACK - Publish acknowledgement
		// - 固定报头: 报文类型0x04，标志位必须为0
		// - 可变报头: 报文标识符
		// - 无载荷
		//
		// MQTT v5.0: 参考章节 3.4 PUBACK - Publish acknowledgement
		// - 在v3.1.1基础上增加了属性(Properties)和原因码
		// - 支持更详细的发布状态反馈
		pkt = &PUBACK{FixedHeader: fixed}
	case 0x5: // PUBREC - 发布收到(QoS 2第一步)
		// MQTT v3.1.1: 参考章节 3.5 PUBREC - Publish received (QoS 2 publish received, part 1)
		// - 固定报头: 报文类型0x05，标志位必须为0
		// - 可变报头: 报文标识符
		// - 无载荷
		//
		// MQTT v5.0: 参考章节 3.5 PUBREC - Publish received (QoS 2 publish received, part 1)
		// - 在v3.1.1基础上增加了属性(Properties)和原因码
		pkt = &PUBREC{FixedHeader: fixed}
	case 0x6: // PUBREL - 发布释放(QoS 2第二步)
		// MQTT v3.1.1: 参考章节 3.6 PUBREL - Publish release (QoS 2 publish received, part 2)
		// - 固定报头: 报文类型0x06，DUP=0, QoS=1, RETAIN=0
		// - 可变报头: 报文标识符
		// - 无载荷
		//
		// MQTT v5.0: 参考章节 3.6 PUBREL - Publish release (QoS 2 publish received, part 2)
		// - 在v3.1.1基础上增加了属性(Properties)和原因码
		pkt = &PUBREL{FixedHeader: fixed}
	case 0x7: // PUBCOMP - 发布完成(QoS 2第三步)
		// MQTT v3.1.1: 参考章节 3.7 PUBCOMP - Publish complete (QoS 2 publish received, part 3)
		// - 固定报头: 报文类型0x07，标志位必须为0
		// - 可变报头: 报文标识符
		// - 无载荷
		//
		// MQTT v5.0: 参考章节 3.7 PUBCOMP - Publish complete (QoS 2 publish received, part 3)
		// - 在v3.1.1基础上增加了属性(Properties)和原因码
		pkt = &PUBCOMP{FixedHeader: fixed}
	case 0x8: // SUBSCRIBE - 订阅请求
		// MQTT v3.1.1: 参考章节 3.8 SUBSCRIBE - Subscribe to topics
		// - 固定报头: 报文类型0x08，DUP=0, QoS=1, RETAIN=0
		// - 可变报头: 报文标识符
		// - 载荷: 主题过滤器列表，每个包含主题过滤器和QoS
		//
		// MQTT v5.0: 参考章节 3.8 SUBSCRIBE - Subscribe to topics
		// - 在v3.1.1基础上增加了属性(Properties)
		// - 新增属性: 订阅标识符，用于关联订阅和消息
		pkt = &SUBSCRIBE{FixedHeader: fixed}
	case 0x9: // SUBACK - 订阅确认
		// MQTT v3.1.1: 参考章节 3.9 SUBACK - Subscribe acknowledgement
		// - 固定报头: 报文类型0x09，标志位必须为0
		// - 可变报头: 报文标识符
		// - 载荷: 返回码列表，对应每个主题过滤器的订阅结果
		//
		// MQTT v5.0: 参考章节 3.9 SUBACK - Subscribe acknowledgement
		// - 在v3.1.1基础上增加了属性(Properties)和原因码列表
		// - 支持更详细的订阅状态反馈
		pkt = &SUBACK{FixedHeader: fixed}
	case 0xA: // UNSUBSCRIBE - 取消订阅
		// MQTT v3.1.1: 参考章节 3.10 UNSUBSCRIBE - Unsubscribe from topics
		// - 固定报头: 报文类型0x0A，DUP=0, QoS=1, RETAIN=0
		// - 可变报头: 报文标识符
		// - 载荷: 主题过滤器列表
		//
		// MQTT v5.0: 参考章节 3.10 UNSUBSCRIBE - Unsubscribe from topics
		// - 在v3.1.1基础上增加了属性(Properties)
		pkt = &UNSUBSCRIBE{FixedHeader: fixed}
	case 0xB: // UNSUBACK - 取消订阅确认
		// MQTT v3.1.1: 参考章节 3.11 UNSUBACK - Unsubscribe acknowledgement
		// - 固定报头: 报文类型0x0B，标志位必须为0
		// - 可变报头: 报文标识符
		// - 无载荷
		//
		// MQTT v5.0: 参考章节 3.11 UNSUBACK - Unsubscribe acknowledgement
		// - 在v3.1.1基础上增加了属性(Properties)和原因码列表
		pkt = &UNSUBACK{FixedHeader: fixed}
	case 0xC: // PINGREQ - 心跳请求
		// MQTT v3.1.1: 参考章节 3.12 PINGREQ - PING request
		// - 固定报头: 报文类型0x0C，标志位必须为0
		// - 无可变报头和载荷
		//
		// MQTT v5.0: 参考章节 3.12 PINGREQ - PING request
		// - 与v3.1.1完全相同
		pkt = &PINGREQ{FixedHeader: fixed}
	case 0xD: // PINGRESP - 心跳响应
		// MQTT v3.1.1: 参考章节 3.13 PINGRESP - PING response
		// - 固定报头: 报文类型0x0D，标志位必须为0
		// - 无可变报头和载荷
		//
		// MQTT v5.0: 参考章节 3.13 PINGRESP - PING response
		// - 与v3.1.1完全相同
		pkt = &PINGRESP{FixedHeader: fixed}
	case 0xE: // DISCONNECT - 断开连接
		// MQTT v3.1.1: 参考章节 3.14 DISCONNECT - Disconnect notification
		// - 固定报头: 报文类型0x0E，标志位必须为0
		// - 无可变报头和载荷
		//
		// MQTT v5.0: 参考章节 3.14 DISCONNECT - Disconnect notification
		// - 在v3.1.1基础上增加了属性(Properties)和原因码
		// - 支持优雅断开和状态反馈
		pkt = &DISCONNECT{FixedHeader: fixed}
	case 0xF: // AUTH - 认证交换
		// MQTT v3.1.1: 不支持此报文类型
		// - 0x0F为保留值，收到此报文类型将导致协议错误
		//
		// MQTT v5.0: 参考章节 3.15 AUTH - Authentication exchange
		// - 固定报头: 报文类型0x0F，标志位必须为0
		// - 可变报头: 认证原因码
		// - 载荷: 认证方法和认证数据(可选)
		// - 用于客户端和服务端之间的扩展认证流程
		pkt = &AUTH{FixedHeader: fixed}
	default:
		return pkt, ErrMalformedPacket
	}
	return pkt, pkt.Unpack(buf)
}
