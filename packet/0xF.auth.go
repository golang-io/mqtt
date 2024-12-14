package packet

import (
	"bytes"
	"io"
)

type AUTH struct {
	*FixedHeader
	AuthenticationReasonCode ReasonCode
	AuthProps                *AuthProperties
}

func (pkt *AUTH) Kind() byte {
	return 0xF
}

func (pkt *AUTH) Packet(w io.Writer) error {
	//buf := GetBuffer()
	//defer PutBuffer(buf)
	buf := new(bytes.Buffer)

	buf.WriteByte(pkt.AuthenticationReasonCode.Code)
	if pkt.Version == VERSION500 {
		b, err := pkt.AuthProps.Pack()
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

func (pkt *AUTH) Unpack(buf *bytes.Buffer) error {
	pkt.AuthenticationReasonCode = ReasonCode{Code: buf.Next(1)[0]}
	if pkt.Version == VERSION500 {
		pkt.AuthProps = &AuthProperties{}
		if err := pkt.AuthProps.Unpack(buf); err != nil {
			return err
		}
	}
	return nil
}

type AuthProperties struct {
	AuthenticationMethod string
	AuthenticationData   []byte
	ReasonString         string

	UserProperty map[string][]string
}

func (props *AuthProperties) Pack() ([]byte, error) {
	buf := GetBuffer()
	defer PutBuffer(buf)
	buf.WriteByte(0x15)
	buf.Write(encodeUTF8(props.AuthenticationMethod))

	if props.AuthenticationData != nil {
		buf.WriteByte(0x16)
		buf.Write(encodeUTF8(props.AuthenticationData))
	}

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

func (props *AuthProperties) Unpack(buf *bytes.Buffer) error {
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
		case 0x15:
			props.AuthenticationMethod, i = decodeUTF8[string](buf), i+uint32(len(props.AuthenticationMethod))
		case 0x16:
			props.AuthenticationData, i = decodeUTF8[[]byte](buf), i+uint32(len(props.AuthenticationData))
		case 0x1F:
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
