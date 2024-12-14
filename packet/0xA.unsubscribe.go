package packet

import (
	"bytes"
	"encoding/binary"
	"io"
)

type UNSUBSCRIBE struct {
	*FixedHeader
	PacketID      uint16 // 报文标识符
	Subscriptions []Subscription

	UnsubscribeProps *UnsubscribeProperties
}

func (pkt *UNSUBSCRIBE) Kind() byte {
	return 0xA
}

func (pkt *UNSUBSCRIBE) Pack(w io.Writer) error {
	var buf bytes.Buffer
	buf.Write(i2b(pkt.PacketID))
	for _, subscription := range pkt.Subscriptions {
		buf.Write(s2b(subscription.TopicFilter))
	}
	pkt.FixedHeader.RemainingLength = uint32(buf.Len())
	if pkt.Version == VERSION500 {
		b, err := pkt.UnsubscribeProps.Pack()
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

func (pkt *UNSUBSCRIBE) Unpack(buf *bytes.Buffer) error {
	pkt.PacketID = binary.BigEndian.Uint16(buf.Next(2))

	if pkt.Version == VERSION500 {
		pkt.UnsubscribeProps = &UnsubscribeProperties{}
		if err := pkt.UnsubscribeProps.Unpack(buf); err != nil {
			return err
		}
	}

	for buf.Len() != 0 {
		topicLength := int(binary.BigEndian.Uint16(buf.Next(2))) // topic length
		subscription := Subscription{TopicFilter: string(buf.Next(topicLength))}
		pkt.Subscriptions = append(pkt.Subscriptions, subscription)
	}
	return nil
}

type UnsubscribeProperties struct {
	UserProperty map[string][]string
}

func (props *UnsubscribeProperties) Pack() ([]byte, error) {
	buf := GetBuffer()
	defer PutBuffer(buf)

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

func (props *UnsubscribeProperties) Unpack(buf *bytes.Buffer) error {
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
		case 0x26:
			if props.UserProperty == nil {
				props.UserProperty = make(map[string][]string)
			}
			key := decodeUTF8[string](buf)
			props.UserProperty[key] = append(props.UserProperty[key], decodeUTF8[string](buf))
		}
	}
	return nil
}
