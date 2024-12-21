package packet

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"strings"
)

type PUBLISH struct {
	*FixedHeader `json:"FixedHeader,omitempty"`
	PacketID     uint16             `json:"PacketID,omitempty"` // 报文标识符
	Message      *Message           `json:"message,omitempty"`
	Props        *PublishProperties `json:"properties,omitempty"`
}

func (pkt *PUBLISH) Kind() byte {
	return 0x3
}

func (pkt *PUBLISH) Pack(w io.Writer) error {
	buf := GetBuffer()
	defer PutBuffer(buf)
	buf.Write(s2b(pkt.Message.TopicName))
	// QoS 设置为 0 的 Publish 报文不能包含报文标识符 [MQTT-2.3.1-5]。
	if pkt.QoS != 0 {
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
		buf.Write(propsLen)
		buf.Write(b)
	}

	buf.Write(pkt.Message.Content)
	pkt.FixedHeader.RemainingLength = uint32(buf.Len())
	//fmt.Printf("buf.Size=%d, topic=%s, content=%s, packet=%v\n", buf.Len(), pkt.Message.TopicName, pkt.Message.Content, buf.Bytes())
	if err := pkt.FixedHeader.Pack(w); err != nil {
		return err
	}

	_, err := buf.WriteTo(w)
	return err
}

func (pkt *PUBLISH) Unpack(buf *bytes.Buffer) error {
	topicLength := int(binary.BigEndian.Uint16(buf.Next(2))) // topic length
	if pkt.Message == nil {
		pkt.Message = &Message{}
	}
	pkt.Message.TopicName = string(buf.Next(topicLength))

	// Publish 报文中的主题名不能包含通配符 [MQTT-3.3.2-2]。
	if pkt.Message.TopicName == "" || strings.Contains(pkt.Message.TopicName, " ") {

		return fmt.Errorf("pkt.RemainingLength=%v, err=%w", pkt.RemainingLength, ErrTopicNameInvalid)
	}
	// QoS 设置为 0 的 Publish 报文不能包含报文标识符 [MQTT-2.3.1-5]。
	if pkt.QoS != 0 {
		pkt.PacketID = binary.BigEndian.Uint16(buf.Next(2))
	}

	if pkt.Version == VERSION500 {
		pkt.Props = &PublishProperties{}
		if err := pkt.Props.Unpack(buf); err != nil {
			return fmt.Errorf("pkt.RemainingLength=%v err=%w", pkt.RemainingLength, err)
		}
	}

	pkt.Message.Content = buf.Bytes()
	return nil
}

// Message Publish message
type Message struct {
	TopicName string
	// 包含零长度有效载荷的 Publish 报文是合法的。
	Content []byte
}

func (m *Message) String() string {
	return fmt.Sprintf("%s # %s", m.TopicName, m.Content)
}

type PublishProperties struct {
	PayloadFormatIndicator uint8
	MessageExpiryInterval  uint32
	TopicAlias             uint16
	ResponseTopic          string
	CorrelationData        []byte
	UserProperty           map[string][]string
	SubscriptionIdentifier []uint32
	ContentType            string
}

func (props *PublishProperties) Unpack(b *bytes.Buffer) error {
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
		case 0x01:
			props.PayloadFormatIndicator, i = b.Next(1)[0], i+1
		case 0x02:
			props.MessageExpiryInterval, i = binary.BigEndian.Uint32(b.Next(4)), i+4
		case 0x23:
			props.TopicAlias, i = binary.BigEndian.Uint16(b.Next(2)), i+2
		case 0x08:
			props.ResponseTopic, i = decodeUTF8[string](b), i+uint32(len(props.ResponseTopic))
		case 0x09:
			props.CorrelationData, i = decodeUTF8[[]byte](b), i+uint32(len(props.CorrelationData))
		case 0x26:
			if props.UserProperty == nil {
				props.UserProperty = make(map[string][]string)
			}
			key := decodeUTF8[string](b)
			props.UserProperty[key] = append(props.UserProperty[key], decodeUTF8[string](b))
		case 0x0B:
			subscriptionIdentifier, err := decodeLength(b)
			if err != nil {
				return err
			}
			props.SubscriptionIdentifier = append(props.SubscriptionIdentifier, subscriptionIdentifier)
			vb, _ := encodeLength(subscriptionIdentifier)
			i += uint32(len(vb)) // 用来计算动态
		case 0x03:
			props.ContentType, i = decodeUTF8[string](b), i+uint32(len(props.ContentType))

		}
	}
	return nil
}

func (props *PublishProperties) Pack() ([]byte, error) {
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
	if props.TopicAlias != 0 {
		buf.WriteByte(0x23)
		buf.Write(i2b(props.TopicAlias))
	}
	if props.ResponseTopic != "" {
		buf.WriteByte(0x08)
		buf.Write(encodeUTF8(props.ResponseTopic))
	}
	if props.CorrelationData != nil {
		buf.WriteByte(0x09)
		buf.Write(encodeUTF8(props.CorrelationData))
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
	if props.ContentType != "" {
		buf.WriteByte(0x03)
		buf.Write(encodeUTF8(props.ContentType))
	}
	return buf.Bytes(), nil

}
