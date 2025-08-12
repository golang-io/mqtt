package mqtt

// Control packet types. Position: byte 1, bits 7-4
const (
	RESERVED    byte = 0x0
	CONNECT     byte = 0x1
	CONNACK     byte = 0x2
	PUBLISH     byte = 0x3
	PUBACK      byte = 0x4
	PUBREC      byte = 0x5
	PUBREL      byte = 0x6
	PUBCOMP     byte = 0x7
	SUBSCRIBE   byte = 0x8
	SUBACK      byte = 0x9
	UNSUBSCRIBE byte = 0xA
	UNSUBACK    byte = 0xB
	PINGREQ     byte = 0xC
	PINGRESP    byte = 0xD
	DISCONNECT  byte = 0xE
	AUTH        byte = 0xF
)
