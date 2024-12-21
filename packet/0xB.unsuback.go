package packet

import (
	"bytes"
	"encoding/binary"
	"io"
)

type UNSUBACK struct {
	*FixedHeader `json:"FixedHeader,omitempty"`
	PacketID     uint16 `json:"PacketID,omitempty"` // 报文标识符

	UnsubackProps *UnsubackProperties
}

func (pkt *UNSUBACK) Kind() byte {
	return 0xB
}

func (pkt *UNSUBACK) Pack(w io.Writer) error {
	buf := GetBuffer()
	defer PutBuffer(buf)

	buf.Write(i2b(pkt.PacketID))

	if pkt.Version == VERSION500 {
		if pkt.UnsubackProps == nil {
			pkt.UnsubackProps = &UnsubackProperties{}
		}
		b, err := pkt.UnsubackProps.Pack()
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
func (pkt *UNSUBACK) Unpack(buf *bytes.Buffer) error {
	if pkt.FixedHeader.RemainingLength != 2 {
		return ErrMalformedPacket
	}
	pkt.PacketID = binary.BigEndian.Uint16(buf.Bytes())

	switch pkt.Version {
	case VERSION500:
		pkt.UnsubackProps = &UnsubackProperties{}
		if err := pkt.UnsubackProps.Unpack(buf); err != nil {
			return err
		}
	case VERSION311:
	case VERSION310:
		return ErrUnsupportedProtocolVersion
	default:
		return ErrMalformedProtocolVersion

	}
	return nil
}

type UnsubackProperties struct {
	ReasonString string
	UserProperty map[string][]string
}

func (props *UnsubackProperties) Pack() ([]byte, error) {
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

func (props *UnsubackProperties) Unpack(buf *bytes.Buffer) error {
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
		case 0x1F:
			if props.ReasonString != "" {
				return ErrProtocolErr
			}
			props.ReasonString, i = decodeUTF8[string](buf), i+uint32(len(props.ReasonString))
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
