package packet

import (
	"bytes"
	"encoding/binary"
	"io"
)

type SUBACK struct {
	*FixedHeader `json:"FixedHeader,omitempty"`
	PacketID     uint16 `json:"PacketID,omitempty"` // 报文标识符

	SubackProps *SubackProperties

	// payload
	// 0x00 - 最大 QoS 0
	// 0x01 - 成功 – 最大 QoS 1
	// 0x02 - 成功 – 最大 QoS 2
	// 0x80 - Failure 失败
	ReasonCode []ReasonCode `json:"ReasonCode,omitempty"`
}

func (pkt *SUBACK) Kind() byte {
	return 0x9
}

func (pkt *SUBACK) Pack(w io.Writer) error {
	buf := GetBuffer()
	defer PutBuffer(buf)
	if len(pkt.ReasonCode) == 0 {
		return ErrMalformedReasonCode
	}
	buf.Write(i2b(pkt.PacketID))

	if pkt.Version == VERSION500 {
		if pkt.SubackProps == nil {
			pkt.SubackProps = &SubackProperties{}
		}
		b, err := pkt.SubackProps.Pack()
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

	for _, reason := range pkt.ReasonCode {
		buf.WriteByte(reason.Code)
	}
	pkt.FixedHeader.RemainingLength = uint32(buf.Len())
	if err := pkt.FixedHeader.Pack(w); err != nil {
		return err
	}
	_, err := buf.WriteTo(w)
	return err
}

func (pkt *SUBACK) Unpack(buf *bytes.Buffer) error {
	pkt.PacketID = binary.BigEndian.Uint16(buf.Next(2))

	if pkt.Version == VERSION500 {
		pkt.SubackProps = &SubackProperties{}
		if err := pkt.SubackProps.Unpack(buf); err != nil {
			return err
		}
	}

	for buf.Len() != 0 {
		reason := ReasonCode{Code: buf.Next(1)[0]}
		if reason.Code == 0x80 {
			return ErrMalformedReasonCode
		}
		if reason.Code > 0x02 {
			return ErrMalformedReasonCode
		}
		pkt.ReasonCode = append(pkt.ReasonCode, reason)
	}
	return nil
}

type SubackProperties struct {
	ReasonString string
	UserProperty map[string][]string
}

func (props *SubackProperties) Pack() ([]byte, error) {
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

func (props *SubackProperties) Unpack(buf *bytes.Buffer) error {
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
