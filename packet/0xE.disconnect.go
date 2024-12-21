package packet

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

type DISCONNECT struct {
	*FixedHeader `json:"FixedHeader,omitempty"`

	DisconnectReasonCode ReasonCode

	Props *DisconnectProperties
}

func (pkt *DISCONNECT) Kind() byte {
	return 0xE
}

func (pkt *DISCONNECT) Pack(w io.Writer) error {
	buf := GetBuffer()
	defer PutBuffer(buf)

	buf.WriteByte(pkt.DisconnectReasonCode.Code)
	if pkt.Version == VERSION500 {
		if pkt.Props == nil {
			pkt.Props = &DisconnectProperties{}
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
	pkt.FixedHeader.RemainingLength = uint32(buf.Len())

	if err := pkt.FixedHeader.Pack(w); err != nil {
		return err
	}
	_, err := buf.WriteTo(w)
	return err
}

func (pkt *DISCONNECT) Unpack(buf *bytes.Buffer) error {
	// 服务端必须验证所有的保留位都被设置为0，如果它们不为0必须断开连接 [MQTT-3.14.1-1]。
	if pkt.Dup != 0 || pkt.QoS != 0 || pkt.Retain != 0 {
		return fmt.Errorf("DISCONNECT: DUP, QoS, Retain must be 0")
	}
	if pkt.RemainingLength == 0 {
		return nil
	}
	pkt.DisconnectReasonCode = ReasonCode{Code: buf.Next(1)[0]}
	if pkt.Version == VERSION500 {
		pkt.Props = &DisconnectProperties{}
		if err := pkt.Props.Unpack(buf); err != nil {
			return err
		}
	}

	return nil
}

type DisconnectProperties struct {
	SessionExpiryInterval uint32
	ReasonString          string
}

func (props *DisconnectProperties) Pack() ([]byte, error) {
	buf := GetBuffer()
	defer PutBuffer(buf)
	if props.SessionExpiryInterval != 0 {
		buf.WriteByte(0x11)
		buf.Write(i4b(props.SessionExpiryInterval))
	}
	if props.ReasonString != "" {
		buf.WriteByte(0x1F)
		buf.Write(encodeUTF8(props.ReasonString))
	}
	return buf.Bytes(), nil
}
func (props *DisconnectProperties) Unpack(buf *bytes.Buffer) error {
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
		case 0x11:
			if props.SessionExpiryInterval != 0 {
				return ErrProtocolErr
			}
			props.SessionExpiryInterval, i = binary.BigEndian.Uint32(buf.Next(4)), i+4
		case 0x1F:
			props.ReasonString, i = decodeUTF8[string](buf), i+uint32(len(props.ReasonString))
		}
	}
	return nil
}
