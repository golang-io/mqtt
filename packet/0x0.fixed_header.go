package packet

import (
	"fmt"
	"io"
)

// FixedHeader contains the values of the fixed header portion of the MQTT pkt.
// Each MQTT Control Packet contains a fixed header.
// Bit 		| 7 | 6 |	5	4	3	2	1	0
// byte1    | MQTT Control Packet type | Flags specific to each MQTT Control Packet type|
// byte2...	|    Remaining Length
type FixedHeader struct {
	Version byte // 这是为了兼容多版本定义的字段!

	// Kind MQTT Control Packet type
	// Position: byte 1, bits 7-4.
	Kind byte `json:"Kind,omitempty"` // the type of the packet (PUBLISH, SUBSCRIBE, etc.) from bits 7 - 4 (byte 1).

	// Flags Position: byte 1, bits 3-0.

	// Dup position: byte 1, bytes 3.
	Dup uint8 `json:"Dup,omitempty"` // indicates if the packet was already sent at an earlier time.

	// QoS position: byte1, bytes 2-1.
	QoS uint8 `json:"QoS,omitempty"` // indicates the quality of service expected.

	// Retain position: byte1, bytes 0.
	Retain uint8 `json:"Retain,omitempty"` // whether the message should be retained.

	// RemainingLength position: starts at byte 2.
	RemainingLength uint32 `json:"RemainingLength,omitempty"` // the number of remaining bytes in the payload.
}

func (pkt *FixedHeader) String() string {
	return fmt.Sprintf("%s: Len=%d", Kind[pkt.Kind], pkt.RemainingLength)
}

func (pkt *FixedHeader) Pack(w io.Writer) error {
	b := make([]byte, 1)

	b[0] |= pkt.Kind << 4
	b[0] |= pkt.Dup << 3
	b[0] |= pkt.QoS << 1
	b[0] |= pkt.Retain
	enc, err := encodeLength(pkt.RemainingLength)
	if err != nil {
		return err
	}

	b = append(b, enc...)
	_, err = w.Write(b)
	return err
}

func (pkt *FixedHeader) Unpack(r io.Reader) error {
	b := []uint8{0x00}

	_, err := r.Read(b)
	if err != nil {
		return err
	}

	pkt.Kind = b[0] >> 4
	pkt.Dup = b[0] & 0b00001000 >> 3
	pkt.QoS = b[0] & 0b00000110 >> 1
	pkt.Retain = b[0] & 0b00000001
	// V500: 表格 2.2 中任何标记为“保留”的标志位，都是保留给以后使用的，必须设置为表格中列出的值 [MQTT-2.2.2-1]。
	// 如果收到非法的标志，接收者必须关闭网络连接。有关错误处理的详细信息见 4.8 节 [MQTT-2.2.2-2]。
	switch pkt.Kind {
	case 0x03:
		if pkt.QoS > 2 {
			return ErrProtocolViolationQosOutOfRange
		}
	case 0x06, 0x08, 0x0A:
		if pkt.Dup != 0 || pkt.QoS != 1 || pkt.Retain != 0 {
			return ErrMalformedFlags
		}
	default:
		if pkt.Dup != 0 || pkt.QoS != 0 || pkt.Retain != 0 {
			return ErrMalformedFlags
		}
	}

	pkt.RemainingLength, err = decodeLength(r)
	return err
}
