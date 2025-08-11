package packet

import (
	"fmt"
	"io"
)

// FixedHeader 包含MQTT控制报文固定报头部分的值
// 每个MQTT控制报文都包含一个固定报头
//
// MQTT v3.1.1: 参考章节 2.2 Fixed header
// MQTT v5.0: 参考章节 2.2 Fixed header
//
// 固定报头结构 (参考章节 2.2.1 MQTT Control Packet type):
// Bit 		| 7 | 6 |	5	4	3	2	1	0
// byte1    | MQTT Control Packet type | Flags specific to each MQTT Control Packet type|
// byte2...	|    Remaining Length
//
// 版本差异:
// - v3.1.1: 固定报头结构完全按照上述格式
// - v5.0: 固定报头结构与v3.1.1相同，但标志位的使用规则更加严格
type FixedHeader struct {
	Version byte // 这是为了兼容多版本定义的字段!
	// 注意: 此字段不在标准协议中，是为了代码实现方便而添加的

	// Kind MQTT控制报文类型
	// 位置: 第1字节的bits 7-4
	// 参考章节: 2.2.1 MQTT Control Packet type
	// 范围: 0x01-0x0F
	// - 0x01: CONNECT - 客户端连接请求
	// - 0x02: CONNACK - 连接确认
	// - 0x03: PUBLISH - 发布消息
	// - 0x04: PUBACK - 发布确认(QoS 1)
	// - 0x05: PUBREC - 发布收到(QoS 2第一步)
	// - 0x06: PUBREL - 发布释放(QoS 2第二步)
	// - 0x07: PUBCOMP - 发布完成(QoS 2第三步)
	// - 0x08: SUBSCRIBE - 订阅请求
	// - 0x09: SUBACK - 订阅确认
	// - 0x0A: UNSUBSCRIBE - 取消订阅
	// - 0x0B: UNSUBACK - 取消订阅确认
	// - 0x0C: PINGREQ - 心跳请求
	// - 0x0D: PINGRESP - 心跳响应
	// - 0x0E: DISCONNECT - 断开连接
	// - 0x0F: AUTH - 认证交换(v5.0新增)
	Kind byte `json:"Kind,omitempty"`

	// Flags 标志位，位置: 第1字节的bits 3-0
	// 参考章节: 2.2.2 Flags

	// Dup 重复标志，位置: 第1字节的bit 3
	// 参考章节: 2.2.2.1 DUP flag
	// 表示此报文是否是重复发送的
	// - 0: 首次发送
	// - 1: 重复发送
	// 注意: 只有QoS > 0的PUBLISH报文才能设置此标志
	Dup uint8 `json:"Dup,omitempty"`

	// QoS 服务质量等级，位置: 第1字节的bits 2-1
	// 参考章节: 2.2.2.2 QoS flag
	// 表示消息传递的服务质量等级
	// - 0x00: QoS 0 - 最多一次传递
	// - 0x01: QoS 1 - 至少一次传递
	// - 0x02: QoS 2 - 恰好一次传递
	// - 0x03: 保留值，不允许使用
	// 注意: 只有PUBLISH报文使用QoS标志，其他报文类型必须为0
	QoS uint8 `json:"QoS,omitempty"`

	// Retain 保留标志，位置: 第1字节的bit 0
	// 参考章节: 2.2.2.3 RETAIN flag
	// 表示消息是否应该被服务端保留
	// - 0: 不保留
	// - 1: 保留
	// 注意: 只有PUBLISH报文使用RETAIN标志，其他报文类型必须为0
	Retain uint8 `json:"Retain,omitempty"`

	// RemainingLength 剩余长度，位置: 从第2字节开始
	// 参考章节: 2.2.3 Remaining Length
	// 表示可变报头和载荷中剩余的字节数
	// 编码方式: 变长字节整数，最多4字节
	// 最大值: 268,435,455 (0xFFFFFF7F)
	// 注意: 此长度不包括固定报头本身的长度
	RemainingLength uint32 `json:"RemainingLength,omitempty"`
}

func (pkt *FixedHeader) String() string {
	return fmt.Sprintf("%s: Len=%d", Kind[pkt.Kind], pkt.RemainingLength)
}

// Pack 将固定报头序列化到写入器
// 参考章节: 2.2 Fixed header
// 序列化顺序:
// 1. 第1字节: 报文类型(4位) + 标志位(4位)
// 2. 剩余长度: 变长字节整数编码
func (pkt *FixedHeader) Pack(w io.Writer) error {
	b := make([]byte, 1)

	// 构建第1字节: 报文类型(4位) + 标志位(4位)
	// 参考章节: 2.2.1 MQTT Control Packet type
	b[0] |= pkt.Kind << 4 // bits 7-4: 报文类型
	b[0] |= pkt.Dup << 3  // bit 3: 重复标志
	b[0] |= pkt.QoS << 1  // bits 2-1: QoS等级
	b[0] |= pkt.Retain    // bit 0: 保留标志

	// 编码剩余长度
	// 参考章节: 2.2.3 Remaining Length
	enc, err := encodeLength(pkt.RemainingLength)
	if err != nil {
		return err
	}

	b = append(b, enc...)
	_, err = w.Write(b)
	return err
}

// Unpack 从读取器解析固定报头
// 参考章节: 2.2 Fixed header
// 解析顺序:
// 1. 第1字节: 报文类型和标志位
// 2. 剩余长度: 变长字节整数解码
func (pkt *FixedHeader) Unpack(r io.Reader) error {
	b := []uint8{0x00}

	// 读取第1字节
	_, err := r.Read(b)
	if err != nil {
		return err
	}

	// 解析第1字节的各个字段
	// 参考章节: 2.2.1 MQTT Control Packet type
	pkt.Kind = b[0] >> 4             // bits 7-4: 报文类型
	pkt.Dup = b[0] & 0b00001000 >> 3 // bit 3: 重复标志
	pkt.QoS = b[0] & 0b00000110 >> 1 // bits 2-1: QoS等级
	pkt.Retain = b[0] & 0b00000001   // bit 0: 保留标志

	// 标志位验证
	// MQTT v5.0: 参考章节 2.2.2 Flags
	// 表格 2.2 中任何标记为"保留"的标志位，都是保留给以后使用的，必须设置为表格中列出的值 [MQTT-2.2.2-1]
	// 如果收到非法的标志，接收者必须关闭网络连接。有关错误处理的详细信息见 4.8 节 [MQTT-2.2.2-2]
	switch pkt.Kind {
	case 0x03: // PUBLISH
		// PUBLISH报文的QoS必须为0、1或2
		if pkt.QoS > 2 {
			return ErrProtocolViolationQosOutOfRange
		}
	case 0x06, 0x08, 0x0A: // PUBREL, SUBSCRIBE, UNSUBSCRIBE
		// 这些报文类型的标志位必须严格符合规范
		// PUBREL: DUP=0, QoS=1, RETAIN=0
		// SUBSCRIBE: DUP=0, QoS=1, RETAIN=0
		// UNSUBSCRIBE: DUP=0, QoS=1, RETAIN=0
		if pkt.Dup != 0 || pkt.QoS != 1 || pkt.Retain != 0 {
			return ErrMalformedFlags
		}
	default:
		// 其他报文类型的标志位必须为0
		if pkt.Dup != 0 || pkt.QoS != 0 || pkt.Retain != 0 {
			return ErrMalformedFlags
		}
	}

	// 解析剩余长度
	// 参考章节: 2.2.3 Remaining Length
	pkt.RemainingLength, err = decodeLength(r)
	return err
}
