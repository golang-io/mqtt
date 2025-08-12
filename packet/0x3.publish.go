package packet

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"strings"
)

/*
================================================================================
MQTT PUBLISH 包 - 发布消息报文
================================================================================

参考文档:
- MQTT v3.1.1: 章节 3.3 PUBLISH - Publish message
- MQTT v5.0: 章节 3.3 PUBLISH - Publish message

================================================================================
协议概述
================================================================================

PUBLISH控制包用于在客户端和服务器之间传输应用消息。客户端使用PUBLISH包向服务器发送
应用消息，服务器使用PUBLISH包向匹配订阅的客户端发送应用消息。

================================================================================
报文结构
================================================================================

1. 固定报头 (Fixed Header)
   - 报文类型: 0x03 (3)
   - 标志位: DUP, QoS, RETAIN
   - 剩余长度: 可变报头 + 载荷的长度

2. 可变报头 (Variable Header)
   - 主题名 (Topic Name): UTF-8编码字符串
   - 报文标识符 (Packet Identifier): 仅当QoS > 0时存在
   - 属性 (Properties): 仅MQTT v5.0支持

3. 载荷 (Payload)
   - 应用消息内容 (Application Message)

================================================================================
标志位详解
================================================================================

1. DUP (Duplicate) - 重复标志
   位置: byte 1, bit 3
   规则:
   - 如果DUP标志设置为0，表示这是客户端或服务器首次尝试发送此MQTT PUBLISH包
   - 如果DUP标志设置为1，表示这可能是重新发送之前尝试发送的包
   - DUP标志必须由客户端或服务器在尝试重新发送PUBLISH包时设置为1 [MQTT-3.3.1-1]
   - 所有QoS 0消息的DUP标志必须设置为0 [MQTT-3.3.1-2]
   - 传入PUBLISH包的DUP标志值在服务器向订阅者发送PUBLISH包时不会被传播 [MQTT-3.3.1-3]

2. QoS (Quality of Service) - 服务质量
   位置: byte 1, bits 2-1
   值:
   - 00 (0): 最多一次传递 (At most once delivery)
   - 01 (1): 至少一次传递 (At least once delivery)
   - 10 (2): 恰好一次传递 (Exactly once delivery)
   - 11 (3): 保留 - 不能使用

   规则:
   - PUBLISH包不能同时将两个QoS位设置为1 [MQTT-3.3.1-4]
   - 如果服务器或客户端收到两个QoS位都设置为1的PUBLISH包，必须关闭网络连接

3. RETAIN - 保留标志
   位置: byte 1, bit 0
   规则:
   - 如果RETAIN标志设置为1，服务器必须存储应用消息及其QoS [MQTT-3.3.1-5]
   - 建立新订阅时，每个匹配主题名的最后保留消息必须发送给订阅者 [MQTT-3.3.1-6]
   - 如果服务器收到QoS 0且RETAIN标志设置为1的消息，必须丢弃该主题之前保留的任何消息 [MQTT-3.3.1-7]
   - 服务器向客户端发送PUBLISH包时，如果消息是由于客户端建立新订阅而发送的，必须将RETAIN标志设置为1 [MQTT-3.3.1-8]
   - 服务器向客户端发送PUBLISH包时，如果是因为匹配已建立的订阅，必须将RETAIN标志设置为0 [MQTT-3.3.1-9]
   - RETAIN标志设置为1且载荷包含零字节的PUBLISH包将被服务器正常处理 [MQTT-3.3.1-10]
   - 零字节保留消息不能作为保留消息存储在服务器上 [MQTT-3.3.1-11]
   - 如果RETAIN标志为0，服务器不能存储消息，不能删除或替换任何现有的保留消息 [MQTT-3.3.1-12]

================================================================================
可变报头详解
================================================================================

1. 主题名 (Topic Name)
   位置: 可变报头第1个字段
   要求:
   - 必须作为PUBLISH包可变报头的第一个字段存在 [MQTT-3.3.2-1]
   - 必须是UTF-8编码字符串
   - 不能包含通配符 [MQTT-3.3.2-2]
   - 服务器发送给订阅客户端的PUBLISH包中的主题名必须根据第4.7节定义的匹配过程匹配订阅的主题过滤器 [MQTT-3.3.2-3]

2. 报文标识符 (Packet Identifier)
   位置: 仅当QoS > 0时存在
   要求:
   - 仅存在于QoS级别为1或2的PUBLISH包中
   - 范围: 1-65535
   - 用于标识QoS > 0的发布消息，确保消息传递的可靠性

3. 属性 (Properties) - 仅MQTT v5.0
   位置: 可变报头，在报文标识符之后(QoS > 0时)
   包含各种发布选项，如主题别名、消息过期、载荷格式等

================================================================================
载荷详解
================================================================================

载荷包含正在发布的应用消息。数据的内容和格式是应用特定的。载荷的长度可以通过从固定报头中的剩余长度字段减去可变报头的长度来计算。
包含零长度载荷的PUBLISH包是有效的。

================================================================================
响应要求
================================================================================

PUBLISH包的接收者必须根据PUBLISH包中的QoS响应相应的包 [MQTT-3.3.4-1]:

QoS级别 | 预期响应
--------|----------
QoS 0   | 无
QoS 1   | PUBACK包
QoS 2   | PUBREC包

================================================================================
版本差异
================================================================================

MQTT v3.1.1:
- 基本的发布功能
- 支持QoS 0/1/2
- 支持保留消息
- 不支持属性系统

MQTT v5.0:
- 在v3.1.1基础上增加了属性系统
- 支持主题别名、消息过期、载荷格式指示等
- 支持订阅标识符
- 更丰富的错误处理机制

================================================================================
代码问题分析
================================================================================

现有代码存在以下问题:

1. DUP标志位处理不完整
   - 缺少对DUP标志位的验证和设置逻辑
   - 没有实现重新发送时的DUP标志位处理

2. QoS验证逻辑有缺陷
   - 当前验证只检查QoS > 2，但应该检查QoS != 3 (0b11)
   - 缺少对QoS 0时不能包含报文标识符的验证

3. 保留消息处理不完整
   - 缺少对RETAIN标志位的处理逻辑
   - 没有实现保留消息的存储和检索机制

4. 属性系统实现不完整
   - 部分属性的编码/解码逻辑有错误
   - 缺少对属性重复性的验证

================================================================================
改进建议
================================================================================

1. 添加完整的标志位验证
2. 实现DUP标志位的正确处理
3. 完善QoS验证逻辑
4. 添加保留消息处理机制
5. 修复属性系统的编码/解码问题
6. 添加更多的协议约束验证

================================================================================
*/

// PUBLISH 发布消息报文
//
// MQTT v3.1.1: 参考章节 3.3 PUBLISH - Publish message
// MQTT v5.0: 参考章节 3.3 PUBLISH - Publish message
//
// 报文结构:
// 固定报头: 报文类型0x03，标志位包含DUP、QoS、RETAIN
// 可变报头: 主题名、报文标识符(QoS>0时)、属性(v5.0)
// 载荷: 应用消息内容
//
// 版本差异:
// - v3.1.1: 基本的发布功能，支持QoS 0/1/2，支持保留消息
// - v5.0: 在v3.1.1基础上增加了属性系统，支持主题别名、消息过期、载荷格式指示等
//
// 标志位规则:
// - DUP: 只有QoS > 0的报文才能设置，表示重复发送
// - QoS: 0(最多一次)、1(至少一次)、2(恰好一次)
// - RETAIN: 表示消息是否应该被服务端保留
type PUBLISH struct {
	*FixedHeader `json:"FixedHeader,omitempty"`

	// PacketID 报文标识符
	// 参考章节: 2.3.1 Packet Identifier
	// 位置: 可变报头，在主题名之后(QoS > 0时)
	// 要求:
	// - QoS = 0: 不能包含报文标识符 [MQTT-2.3.1-5]
	// - QoS > 0: 必须包含报文标识符，范围1-65535
	// 用途: 用于标识QoS > 0的发布消息，确保消息传递的可靠性
	PacketID uint16 `json:"PacketID,omitempty"`

	Message *Message `json:"message,omitempty"`

	// Props 发布属性 (v5.0新增)
	// 参考章节: 3.3.2.3 PUBLISH Properties
	// 位置: 可变报头，在报文标识符之后(QoS > 0时)
	// 包含各种发布选项，如主题别名、消息过期、载荷格式等
	Props *PublishProperties `json:"properties,omitempty"`
}

func (pkt *PUBLISH) Kind() byte {
	return 0x3
}

func (pkt *PUBLISH) Pack(w io.Writer) error {
	buf := GetBuffer()
	defer PutBuffer(buf)

	// FixedHeader 在外层已经初始化
	if pkt.FixedHeader == nil {
		return fmt.Errorf("FixedHeader is nil")
	}

	// 验证QoS值 - 修复: 应该检查QoS != 3 (0b11)，而不是QoS > 2
	// 根据协议 [MQTT-3.3.1-4]: PUBLISH包不能同时将两个QoS位设置为1
	if pkt.FixedHeader.QoS == 3 {
		return fmt.Errorf("invalid QoS value: %d, QoS bits 11 (0b11) are reserved and must not be used [MQTT-3.3.1-4]", pkt.FixedHeader.QoS)
	}

	// 验证主题名不能为空
	if pkt.Message.TopicName == "" {
		return fmt.Errorf("topic name cannot be empty [MQTT-3.3.2-1]")
	}

	// 验证主题名不能包含通配符
	if strings.Contains(pkt.Message.TopicName, "+") || strings.Contains(pkt.Message.TopicName, "#") {
		return fmt.Errorf("topic name cannot contain wildcard characters [MQTT-3.3.2-2]")
	}

	// 验证主题名不能包含空格字符
	if strings.Contains(pkt.Message.TopicName, " ") {
		return fmt.Errorf("topic name cannot contain space characters")
	}

	buf.Write(s2b(pkt.Message.TopicName))
	// QoS 设置为 0 的 Publish 报文不能包含报文标识符 [MQTT-2.3.1-5]
	if pkt.FixedHeader.QoS > 0 {
		// 验证报文标识符范围
		if pkt.PacketID == 0 {
			return fmt.Errorf("packet identifier must be greater than 0 for QoS > 0 [MQTT-2.3.1-1]")
		}
		buf.Write(i2b(pkt.PacketID))
	}
	if pkt.Version == VERSION500 {
		if pkt.Props == nil {
			pkt.Props = &PublishProperties{}
		}
		b, err := pkt.Props.Pack()
		if err != nil {
			return err
		}
		propsLen, err := encodeLength(len(b))
		if err != nil {
			return err
		}

		_, err = buf.Write(propsLen)
		if err != nil {
			return err
		}

		_, err = buf.Write(b)
		if err != nil {
			return err
		}
	}

	if _, err := buf.Write(pkt.Message.Content); err != nil {
		return err
	}
	pkt.FixedHeader.RemainingLength = uint32(buf.Len())

	if err := pkt.FixedHeader.Pack(w); err != nil {
		return err
	}

	_, err := buf.WriteTo(w)
	return err
}

func (pkt *PUBLISH) Unpack(buf *bytes.Buffer) error {
	// 读取主题名长度
	topicLength := int(binary.BigEndian.Uint16(buf.Next(2)))

	// 验证主题名长度
	if topicLength == 0 {
		return fmt.Errorf("topic name cannot be empty [MQTT-3.3.2-1]")
	}

	// 读取主题名
	pkt.Message = &Message{TopicName: string(buf.Next(topicLength))}
	// 验证主题名不能包含通配符 [MQTT-3.3.2-2]
	if strings.Contains(pkt.Message.TopicName, "+") || strings.Contains(pkt.Message.TopicName, "#") {
		return fmt.Errorf("topic name cannot contain wildcard characters [MQTT-3.3.2-2]")
	}

	// 验证主题名不能包含空格字符
	if strings.Contains(pkt.Message.TopicName, " ") {
		return fmt.Errorf("topic name cannot contain space characters")
	}
	// QoS > 0 的 Publish 报文必须包含报文标识符 [MQTT-2.3.1-5]
	if pkt.FixedHeader.QoS > 0 {
		if buf.Len() < 2 {
			return fmt.Errorf("insufficient data for packet identifier")
		}
		pkt.PacketID = binary.BigEndian.Uint16(buf.Next(2))

		// 验证报文标识符范围
		if pkt.PacketID == 0 {
			return fmt.Errorf("packet identifier must be greater than 0 for QoS > 0 [MQTT-2.3.1-1]")
		}
	}

	if pkt.Version == VERSION500 {
		pkt.Props = &PublishProperties{}
		if err := pkt.Props.Unpack(buf); err != nil {
			return fmt.Errorf("pkt.RemainingLength=%v err=%w", pkt.RemainingLength, err)
		}
	}

	// 使用 append([]byte{}, buf.Bytes()...) 创建深度副本，避免内存共享问题
	// 原因：
	// 1. buf.Bytes() 返回指向缓冲区底层数组的切片引用，不是数据副本
	// 2. 如果直接赋值 pkt.Message.Content = buf.Bytes()，两者会共享同一个底层数组
	// 3. 当缓冲区被复用或修改时，pkt.Message.Content 也会被意外修改
	// 4. append([]byte{}, ...) 创建全新的底层数组，确保数据独立性
	pkt.Message.Content = append([]byte{}, buf.Bytes()...)
	return nil
}

// Message 发布消息内容
// 参考章节: 3.3.3 PUBLISH Payload
// 包含主题名和消息内容
//
// 版本差异:
// - v3.1.1: 基本的主题名和消息内容
// - v5.0: 支持更多消息属性，如载荷格式指示、内容类型等
type Message struct {
	// TopicName 主题名
	// 参考章节: 3.3.2.1 Topic Name
	// 位置: 可变报头第1个字段
	// 要求:
	// - UTF-8编码字符串
	// - 不能为空
	// - 不能包含通配符 [MQTT-3.3.2-2]
	// - 不能包含空格字符
	// 用途: 标识消息应该发布到哪个主题
	TopicName string

	// Content 消息内容
	// 参考章节: 3.3.3 PUBLISH Payload
	// 位置: 载荷部分
	// 类型: 二进制数据
	// 注意: 包含零长度有效载荷的Publish报文是合法的
	// 用途: 实际的应用消息内容
	Content []byte
}

func (m *Message) String() string {
	return fmt.Sprintf("%s # %s", m.TopicName, m.Content)
}

// PublishProperties 发布属性 (v5.0新增)
// 参考章节: 3.3.2.3 PUBLISH Properties
// 包含各种发布选项，用于扩展发布功能
//
// 版本差异:
// - v3.1.1: 不支持属性系统
// - v5.0: 完整的属性系统，支持主题别名、消息过期、载荷格式等
type PublishProperties struct {
	// PayloadFormatIndicator 载荷格式指示
	// 属性标识符: 1 (0x01)
	// 参考章节: 3.3.2.3.2 Payload Format Indicator
	// 类型: 单字节
	// 值:
	// - 0 (0x00): 表示载荷是未指定的字节，等同于不发送载荷格式指示
	// - 1 (0x01): 表示载荷是UTF-8编码的字符数据
	// 注意: 包含多个载荷格式指示将造成协议错误
	PayloadFormatIndicator PayloadFormatIndicator

	// MessageExpiryInterval 消息过期间隔
	// 属性标识符: 2 (0x02)
	// 参考章节: 3.3.2.3.3 Message Expiry Interval
	// 类型: 四字节整数，单位: 秒
	// 含义: 消息的生命周期
	// 注意: 包含多个消息过期间隔将造成协议错误
	MessageExpiryInterval MessageExpiryInterval

	// TopicAlias 主题别名
	// 属性标识符: 35 (0x23)
	// 参考章节: 3.3.2.3.4 Topic Alias
	// 类型: 双字节整数
	// 含义: 用于标识主题的数值
	// 注意:
	// - 包含多个主题别名将造成协议错误
	// - 主题别名值必须大于0
	// - 主题别名只在当前网络连接中有效
	TopicAlias TopicAlias

	// ResponseTopic 响应主题
	// 属性标识符: 8 (0x08)
	// 参考章节: 3.3.2.3.5 Response Topic
	// 类型: UTF-8编码字符串
	// 含义: 表示响应消息的主题名
	// 注意: 包含多个响应主题将造成协议错误
	ResponseTopic ReasonString

	// CorrelationData 对比数据
	// 属性标识符: 9 (0x09)
	// 参考章节: 3.3.2.3.6 Correlation Data
	// 类型: 二进制数据
	// 含义: 被请求消息发送端在收到响应消息时用来标识相应的请求
	// 注意: 包含多个对比数据将造成协议错误
	CorrelationData CorrelationData

	// UserProperty 用户属性
	// 属性标识符: 38 (0x26)
	// 参考章节: 3.3.2.3.7 User Property
	// 类型: UTF-8字符串对
	// 含义: 用户定义的名称/值对，可以出现多次
	// 注意: 用户属性可以出现多次，表示多个名字/值对
	UserProperty map[string][]string

	// SubscriptionIdentifier 订阅标识符
	// 属性标识符: 11 (0x0B)
	// 参考章节: 3.3.2.3.8 Subscription Identifier
	// 类型: 变长字节整数
	// 含义: 标识订阅的数值，用于标识消息应该发送给哪个订阅
	// 注意: 可以包含多个订阅标识符
	SubscriptionIdentifier []uint32

	// ContentType 内容类型
	// 属性标识符: 3 (0x03)
	// 参考章节: 3.3.2.3.9 Content Type
	// 类型: UTF-8编码字符串
	// 含义: 描述载荷的内容
	// 注意: 包含多个内容类型将造成协议错误
	ContentType ContentType
}

func (props *PublishProperties) Unpack(buf *bytes.Buffer) error {
	// 读取属性长度
	propsLen, err := decodeLength(buf)
	if err != nil {
		return err
	}

	// 记录已处理的字节数
	uLen := uint32(0)

	// 解析各个属性
	for i := uint32(0); i < propsLen; i++ {
		// 读取属性标识符
		propsId, err := decodeLength(buf)
		if err != nil {
			return err
		}
		switch propsId {
		case 0x01: // Payload Format Indicator

			if uLen, err = props.PayloadFormatIndicator.Unpack(buf); err != nil {
				return fmt.Errorf("failed to unpack PayloadFormatIndicator: %w", err)
			}

		case 0x02: // Message Expiry Interval
			if uLen, err = props.MessageExpiryInterval.Unpack(buf); err != nil {
				return fmt.Errorf("failed to unpack MessageExpiryInterval: %w", err)
			}

		case 0x23: // Topic Alias
			if uLen, err = props.TopicAlias.Unpack(buf); err != nil {
				return fmt.Errorf("failed to unpack TopicAlias: %w", err)
			}

		case 0x08: // Response Topic
			if uLen, err = props.ResponseTopic.Unpack(buf); err != nil {
				return fmt.Errorf("failed to unpack ResponseTopic: %w", err)
			}

		case 0x09: // Correlation Data
			if uLen, err = props.CorrelationData.Unpack(buf); err != nil {
				return fmt.Errorf("failed to unpack CorrelationData: %w", err)
			}

		case 0x26: // User Property
			if props.UserProperty == nil {
				props.UserProperty = make(map[string][]string)
			}

			userProperty := &UserProperty{}
			if uLen, err = userProperty.Unpack(buf); err != nil {
				return fmt.Errorf("failed to unpack UserProperty: %w", err)
			}
			props.UserProperty[userProperty.Name] = append(props.UserProperty[userProperty.Name], userProperty.Value)

		case 0x0B: // Subscription Identifier
			var subscriptionIdentifier SubscriptionIdentifier
			if uLen, err = subscriptionIdentifier.Unpack(buf); err != nil {
				return fmt.Errorf("failed to unpack SubscriptionIdentifier: %w", err)
			}
			props.SubscriptionIdentifier = append(props.SubscriptionIdentifier, subscriptionIdentifier.Uint32())

		case 0x03: // Content Type
			if uLen, err = props.ContentType.Unpack(buf); err != nil {
				return fmt.Errorf("failed to unpack ContentType: %w", err)
			}
		default:
			// 跳过未知属性
			return fmt.Errorf("unknown property identifier: 0x%02X", propsId)
		}
		i += uLen
	}

	return nil
}

func (props *PublishProperties) Pack() ([]byte, error) {
	buf := GetBuffer()
	defer PutBuffer(buf)

	if err := props.PayloadFormatIndicator.Pack(buf); err != nil {
		return nil, err
	}

	if err := props.MessageExpiryInterval.Pack(buf); err != nil {
		return nil, err
	}

	if err := props.TopicAlias.Pack(buf); err != nil {
		return nil, err
	}

	if err := props.ResponseTopic.Pack(buf); err != nil {
		return nil, err
	}

	if err := props.CorrelationData.Pack(buf); err != nil {
		return nil, err
	}

	for k, values := range props.UserProperty {
		for i := range values {
			if err := (&UserProperty{Name: k, Value: values[i]}).Pack(buf); err != nil {
				return nil, err
			}
		}
	}

	if len(props.SubscriptionIdentifier) != 0 {
		for _, subscriptionIdentifier := range props.SubscriptionIdentifier {
			buf.WriteByte(0x0B)
			v, err := encodeLength(subscriptionIdentifier)
			if err != nil {
				return nil, err
			}
			buf.Write(v)
		}
	}

	if err := props.ContentType.Pack(buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil

}
