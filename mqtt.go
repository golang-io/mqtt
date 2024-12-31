package mqtt

import (
	"log"
	"os"
)

const KB = 1024 * 1
const MB = 1024 * KB

// maxRemainLength Digits From - To
// 1 0 (0x00) - 127 (0x7F)
// 2 128 (0x80, 0x01) - 16383 (0xFF, 0x7F)
// 3 16384 (0x80, 0x80, 0x01) - 2097151 (0xFF, 0xFF, 0x7F)
// 4 2097152 (0x80, 0x80, 0x80, 0x01) - 268435455 (0xFF, 0xFF, 0xFF, 0x7F)
// const maxRemainLength = 256 * MB // 256MB

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

var pktLog *log.Logger

func init() {
	pktLog = log.New(os.Stdout, "[PKT]", log.Lmsgprefix|log.Lmicroseconds|log.Lshortfile)
}
