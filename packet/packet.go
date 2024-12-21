package packet

import (
	"bytes"
	"io"
)

type Packet interface {
	Kind() byte
	Unpack(*bytes.Buffer) error
	Pack(io.Writer) error
}

func Unpack(version byte, r io.Reader) (Packet, error) {
	pkt, fixed := Packet(nil), &FixedHeader{Version: version}
	if err := fixed.Unpack(r); err != nil {
		return &RESERVED{FixedHeader: fixed}, err
	}

	buf := GetBuffer()
	defer PutBuffer(buf)

	lr := io.LimitReader(r, int64(fixed.RemainingLength))

	if _, err := buf.ReadFrom(lr); err != nil {
		return pkt, err
	}

	switch fixed.Kind {
	case 0x1:
		pkt = &CONNECT{FixedHeader: fixed}
	case 0x2:
		pkt = &CONNACK{FixedHeader: fixed}
	case 0x3:
		pkt = &PUBLISH{FixedHeader: fixed}
	case 0x4:
		pkt = &PUBACK{FixedHeader: fixed}
	case 0x5:
		pkt = &PUBREC{FixedHeader: fixed}
	case 0x6:
		pkt = &PUBREL{FixedHeader: fixed}
	case 0x7:
		pkt = &PUBCOMP{FixedHeader: fixed}
	case 0x8:
		pkt = &SUBSCRIBE{FixedHeader: fixed}
	case 0x9:
		pkt = &SUBACK{FixedHeader: fixed}
	case 0xA:
		pkt = &UNSUBSCRIBE{FixedHeader: fixed}
	case 0xB:
		pkt = &UNSUBACK{FixedHeader: fixed}
	case 0xC:
		pkt = &PINGREQ{FixedHeader: fixed}
	case 0xD:
		pkt = &PINGRESP{FixedHeader: fixed}
	case 0xE:
		pkt = &DISCONNECT{FixedHeader: fixed}
	case 0xF:
		pkt = &AUTH{FixedHeader: fixed}
	default:
		return pkt, ErrMalformedPacket
	}
	return pkt, pkt.Unpack(buf)
}
