package packet

import (
	"bytes"
	"io"
)

// PINGRESP 心跳响应报文
//
// MQTT v3.1.1: 参考章节 3.13 PINGRESP - PING response
// MQTT v5.0: 参考章节 3.13 PINGRESP - PING response
//
// 报文结构:
// 固定报头: 报文类型0x0D，标志位必须为0
// 可变报头: 无
// 载荷: 无载荷
//
// 版本差异:
// - v3.1.1: 基本的心跳响应功能，无载荷
// - v5.0: 与v3.1.1完全相同，无变化
//
// 用途:
// - 用于服务端响应客户端的PINGREQ报文
// - 确认网络连接仍然活跃
// - 保持客户端和服务端之间的心跳机制
//
// 标志位规则:
// - DUP: 必须为0
// - QoS: 必须为0
// - RETAIN: 必须为0
//
// 发送时机:
// - 服务端收到PINGREQ报文后必须立即发送PINGRESP
// - 这是服务端对客户端心跳请求的唯一响应方式
// - 如果客户端在合理时间内没有收到PINGRESP，应该关闭网络连接
type PINGRESP struct {
	*FixedHeader `json:"FixedHeader,omitempty"`
}

func (pkt *PINGRESP) Kind() byte {
	return 0xD
}
func (pkt *PINGRESP) Pack(w io.Writer) error {
	return pkt.FixedHeader.Pack(w)
}
func (pkt *PINGRESP) Unpack(_ *bytes.Buffer) error {
	return nil
}
