package packet

import (
	"bytes"
	"encoding/binary"
	"io"
)

type PUBCOMP struct {
	*FixedHeader `json:"FixedHeader,omitempty"`
	PacketID     uint16 `json:"PacketID,omitempty"` // 报文标识符
	ReasonCode   ReasonCode
	Props        *PubcompProperties
}

func (pkt *PUBCOMP) Kind() byte {
	return 0x7
}

func (pkt *PUBCOMP) Pack(w io.Writer) error {
	var buf bytes.Buffer
	buf.Write(i2b(pkt.PacketID))
	if pkt.Version == VERSION500 {
		buf.WriteByte(pkt.ReasonCode.Code)

		pkt.Props = &PubcompProperties{}
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

func (pkt *PUBCOMP) Unpack(buf *bytes.Buffer) error {

	pkt.PacketID = binary.BigEndian.Uint16(buf.Next(2))

	if pkt.Version == VERSION500 {
		pkt.ReasonCode.Code = buf.Next(1)[0]

		pkt.Props = &PubcompProperties{}
		if err := pkt.Props.Unpack(buf); err != nil {
			return err
		}
	}
	return nil
}

type PubcompProperties struct {
	ReasonString string
	UserProperty map[string][]string
}

func (props *PubcompProperties) Pack() ([]byte, error) {
	buf := GetBuffer()
	defer PutBuffer(buf)

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
	return buf.Bytes(), nil
}

func (props *PubcompProperties) Unpack(buf *bytes.Buffer) error {
	propsLen, err := decodeLength(buf)
	if err != nil {
		return err
	}

	for i := uint32(0); i < propsLen; i++ {
		propsId, err := decodeLength(buf)
		if err != nil {
			return err
		}
		switch propsId {
		case 0x1F: // 会话过期间隔 Session Expiry Interval
			props.ReasonString, i = decodeUTF8[string](buf), i+uint32(len(props.UserProperty))
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
