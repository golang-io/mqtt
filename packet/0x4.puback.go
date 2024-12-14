package packet

import (
	"bytes"
	"encoding/binary"
	"io"
)

type PUBACK struct {
	*FixedHeader

	// 可变报头
	PacketID    uint16 // 报文标识符
	ReasonCode  ReasonCode
	PubackProps *PubackProperties
}

func (pkt *PUBACK) Kind() byte {
	return 0x4
}

func (pkt *PUBACK) Pack(w io.Writer) error {
	var buf bytes.Buffer
	pkt.RemainingLength = 2
	buf.Write(i2b(pkt.PacketID))
	if pkt.Version == VERSION500 {
		buf.WriteByte(pkt.ReasonCode.Code)
		pkt.RemainingLength += 1

		pkt.PubackProps = &PubackProperties{}
		b, err := pkt.PubackProps.Pack()
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

func (pkt *PUBACK) Unpack(buf *bytes.Buffer) error {

	pkt.PacketID = binary.BigEndian.Uint16(buf.Next(2))

	if pkt.Version == VERSION500 {
		pkt.ReasonCode.Code = buf.Next(1)[0]

		pkt.PubackProps = &PubackProperties{}
		if err := pkt.PubackProps.Unpack(buf); err != nil {
			return err
		}
	}
	return nil
}

type PubackProperties struct {
	ReasonString string
	UserProperty map[string][]string
}

func (props *PubackProperties) Pack() ([]byte, error) {
	//buf := GetBuffer()
	//defer PutBuffer(buf)
	buf := new(bytes.Buffer)
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

func (props *PubackProperties) Unpack(buf *bytes.Buffer) error {
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
