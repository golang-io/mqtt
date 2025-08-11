package packet

import (
	"bytes"
	"io"
)

// PINGREQ 心跳请求报文
//
// MQTT v3.1.1: 参考章节 3.12 PINGREQ - PING request
// MQTT v5.0: 参考章节 3.12 PINGREQ - PING request
//
// 报文结构:
// 固定报头: 报文类型0x0C，标志位必须为0
// 可变报头: 无
// 载荷: 无载荷
//
// 版本差异:
// - v3.1.1: 基本的心跳功能，无载荷
// - v5.0: 与v3.1.1完全相同，无变化
//
// 用途:
// - 用于客户端向服务端发送心跳请求
// - 保持网络连接活跃
// - 检测网络连接状态
// - 在KeepAlive时间间隔内发送
//
// 标志位规则:
// - DUP: 必须为0
// - QoS: 必须为0
// - RETAIN: 必须为0
//
// 响应:
// - 服务端必须响应PINGRESP报文
// - 如果服务端在合理时间内没有响应，客户端应该关闭网络连接
type PINGREQ struct {
	*FixedHeader `json:"FixedHeader,omitempty"`
}

func (pkt *PINGREQ) Kind() byte {
	return 0xC
}
func (pkt *PINGREQ) Pack(w io.Writer) error {
	return pkt.FixedHeader.Pack(w)
}
func (pkt *PINGREQ) Unpack(_ *bytes.Buffer) error {
	return nil
}
