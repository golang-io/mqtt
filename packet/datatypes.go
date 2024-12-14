package packet

import (
	"bytes"
	"encoding/binary"
	"io"
)

const (
	VERSION310 byte = 0x3
	VERSION311 byte = 0x4
	VERSION500 byte = 0x5

	max1 = 0x7F      //127
	max2 = 0x3FFF    //16383
	max3 = 0x200000  // 2097152
	max4 = 0xFFFFFFF // 268435455

	KB = 1024 * 1
	MB = 1024 * KB
)

// Kind Control packet types. Position: byte 1, bits 7-4
var Kind = map[byte]string{
	0x0: "[0x0]RESERVED",    // Forbidden 					Reserved
	0x1: "[0x1]CONNECT",     // 客户端到服务端 客户端请求连接服务端
	0x2: "[0x2]CONNACK",     // 服务端到客户端 连接报文确认
	0x3: "[0x3]PUBLISH",     // Client to Server or Server to Client Publish message
	0x4: "[0x4]PUBACK",      // Client to Server or Server to Client Publish acknowledgment
	0x5: "[0x5]PUBREC",      // Client to Server or Server to Client Publish received (assured delivery part 1)
	0x6: "[0x6]PUBREL",      // Client to Server or Server to Client Publish release (assured delivery part 2)
	0x7: "[0x7]PUBCOMP",     // Client to Server or Server to Client Publish complete (assured delivery part 3)
	0x8: "[0x8]SUBSCRIBE",   // Client to Server Client subscribe request
	0x9: "[0x9]SUBACK",      // Server to Client Subscribe acknowledgment
	0xA: "[0xA]UNSUBSCRIBE", // Client to Server Unsubscribe request
	0xB: "[0xB]UNSUBACK",    // Server to Client Unsubscribe acknowledgment
	0xC: "[0xC]PINGREQ",     // Client to Server PING request
	0xD: "[0xD]PINGRESP",    // Server to Client PING response
	0xE: "[0xE]DISCONNECT",  // Client to Server Client is disconnecting
	0xF: "[0xF]AUTH",        // MQTT 3-11-1:Forbidden Reserved, MQTT 5.0:AUTH
}

func encodeLength[T ~uint32 | ~int | ~int64](v T) ([]byte, error) {
	var result []byte
	if v < max1 {
		result = make([]byte, 1)
	} else if v < max2 {
		result = make([]byte, 2)
	} else if v < max3 {
		result = make([]byte, 3)
	} else if v < max4 {
		result = make([]byte, 4)
	} else {
		return nil, ErrPacketTooLarge
	}
	for i := 0; v > 0; i++ {

		enc := v % 128
		v = v / 128
		if v > 0 { // if there are more data to encode, set the top bit of this byte
			enc = enc | 128
		}
		result[i] = byte(enc)
	}
	return result, nil
}
func decodeLength(r io.Reader) (uint32, error) {
	vbi, b := uint32(0), make([]byte, 1)
	for i := 0; i == 0 || b[0]&128 != 0; i += 7 {
		if _, err := r.Read(b); err != nil && err != io.EOF {
			return 0, err
		}
		if vbi |= uint32(b[0]&127) << i; vbi > max4 {
			return 0, ErrPacketTooLarge
		}
	}
	return vbi, nil
}

// s2b insert length into content
func s2b[T string | []byte](s T) []byte {
	b := make([]byte, 2, 2+len(s))
	binary.BigEndian.PutUint16(b, uint16(len(s)))
	return append(b, s...)
}

func i2b(i uint16) []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, i)
	return b
}

func i4b(i uint32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, i)
	return b
}

func s2i(v string) uint8 {
	if len(v) == 0 {
		return 0
	} else {
		return 1
	}
}

func decodeUTF8[T []byte | string](b *bytes.Buffer) T {
	uLength := binary.BigEndian.Uint16(b.Next(2))
	return T(b.Next(int(uLength)))
}

func encodeUTF8[T []byte | string](v T) []byte {
	uLength := len(v)
	b := make([]byte, 2, uLength+2)
	binary.BigEndian.PutUint16(b, uint16(uLength))
	b = append(b, v...)
	return b
}
