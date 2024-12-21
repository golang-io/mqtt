package packet

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

type SUBSCRIBE struct {
	*FixedHeader `json:"FixedHeader,omitempty"`
	PacketID     uint16 `json:"PacketID,omitempty"` // 报文标识符

	SubscribeProps *SubscribeProperties

	Subscriptions []Subscription `json:"Subscription,omitempty"`
}

func (pkt *SUBSCRIBE) Kind() byte {
	return 0x8
}

func (pkt *SUBSCRIBE) Pack(w io.Writer) error {
	buf := GetBuffer()
	defer PutBuffer(buf)
	buf.Write(i2b(pkt.PacketID))

	if pkt.Version == VERSION500 {
		if pkt.SubscribeProps == nil {
			pkt.SubscribeProps = &SubscribeProperties{}
		}
		b, err := pkt.SubscribeProps.Pack()
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

	for _, subscription := range pkt.Subscriptions {
		buf.Write(s2b(subscription.TopicFilter))
		buf.WriteByte(subscription.MaximumQoS)
	}
	pkt.FixedHeader.RemainingLength = uint32(buf.Len())
	if err := pkt.FixedHeader.Pack(w); err != nil {
		return err
	}
	_, err := buf.WriteTo(w)
	return err
}
func (pkt *SUBSCRIBE) Unpack(buf *bytes.Buffer) error {
	// SUBSCRIBE 控制报固定报头的第 3,2,1,0 位是保留位，必须分别设置为 0,0,1,0。
	// 服务端必须将其它的任何值都当做是不合法的并关闭网络连接 [MQTT-3.8.1-1]。
	if pkt.Dup != 0x0 || pkt.QoS != 0x1 || pkt.Retain != 0x0 {
		return ErrMalformedFlags
	}

	pkt.PacketID = binary.BigEndian.Uint16(buf.Next(2))

	//
	if pkt.Version == VERSION500 {
		pkt.SubscribeProps = &SubscribeProperties{}
		if err := pkt.SubscribeProps.Unpack(buf); err != nil {
			return err
		}
	}

	for buf.Len() != 0 {
		subscription := Subscription{}
		topicLength := int(binary.BigEndian.Uint16(buf.Next(2))) // topic length
		subscription.TopicFilter = string(buf.Next(topicLength))
		options := buf.Next(1)[0]
		subscription.MaximumQoS = options & 0b00000011
		if subscription.MaximumQoS > 0x02 {
			return ErrProtocolViolationQosOutOfRange
		}
		subscription.NoLocal = options & 0b00000100 >> 2
		subscription.RetainAsPublished = options & 0b00001000 >> 3
		subscription.RetainHandling = options & 0b00110000 >> 4
		if options&0b11000000>>6 != 0 {
			return ErrMalformedFlags
		}
		pkt.Subscriptions = append(pkt.Subscriptions, subscription)
	}
	if len(pkt.Subscriptions) == 0 {
		return ErrProtocolViolationNoTopic
	}
	return nil
}

type Subscription struct {
	TopicFilter       string
	MaximumQoS        uint8
	NoLocal           uint8
	RetainAsPublished uint8
	RetainHandling    uint8
}

func (s *Subscription) String() string {
	return fmt.Sprintf("%s@%d", s.TopicFilter, s.MaximumQoS)
}

type SubscribeProperties struct {
	SubscriptionIdentifier uint32
	UserProperty           map[string][]string
}

func (props *SubscribeProperties) Pack() ([]byte, error) {
	buf := GetBuffer()
	defer PutBuffer(buf)

	if props.SubscriptionIdentifier != 0 {
		buf.WriteByte(0x0B)
		vb, err := encodeLength(props.SubscriptionIdentifier)
		if err != nil {
			return nil, err
		}
		buf.Write(vb)
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
	return buf.Bytes(), nil
}

func (props *SubscribeProperties) Unpack(buf *bytes.Buffer) error {
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
		case 0x0B:
			if props.SubscriptionIdentifier, err = decodeLength(buf); err != nil {
				return err
			}
			vb, _ := encodeLength(props.SubscriptionIdentifier)
			i += uint32(len(vb)) // 用来计算动态数字的占位
		case 0x26:
			if props.UserProperty == nil {
				props.UserProperty = make(map[string][]string)
			}
			key := decodeUTF8[string](buf)
			props.UserProperty[key] = append(props.UserProperty[key], decodeUTF8[string](buf))
		default:
			return ErrProtocolViolation
		}
	}
	return nil
}
