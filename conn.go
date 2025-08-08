package mqtt

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang-io/mqtt/packet"
	"github.com/golang-io/mqtt/topic"
)

// conn represents the server side of an HTTP connection.
type conn struct {
	// server is the server on which the connection arrived. Immutable; never nil.
	server *Server

	// cancelCtx cancels the connection-level context.
	cancelCtx context.CancelFunc

	// rwc is the underlying network connection.
	// This is never wrapped by other types and is the value given out to CloseNotifier callers.
	// It is usually of type *net.TCPConn or *tls.Conn.
	rwc net.Conn

	// remoteAddr is rwc.RemoteAddr().String(). It is not populated synchronously
	// inside the Listener's Accept goroutine, as some implementations block.
	// It is populated immediately inside the (*conn).serve goroutine.
	// This is the value of a Handler's (*Request).RemoteAddr.
	remoteAddr string

	//rbuf bufio.Reader
	//wbuf bufio.Writer

	// tlsState is the TLS connection state when using TLS. nil means not TLS.
	tlsState *tls.ConnectionState

	curState atomic.Uint64 // packed (unix time<<8|uint8(ConnState))

	inFight         *InFight // 用这个字典来保存没有处理完QoS1，2的报文
	ID              string
	version         byte // mqtt version
	subscribeTopics *topic.MemoryTrie
	willTopic       string
	willPayload     []byte
	PacketID        uint16
	mu              sync.Mutex
}

func (c *conn) setState(nc net.Conn, state ConnState, runHook bool) {
	srv := c.server
	switch state {
	case StateNew:
		srv.trackConn(c, true)
	case StateHijacked, StateClosed:
		srv.trackConn(c, false)
	default:
	}
	if state > 0xFF || state < 0 {
		panic("invalid conn state")
	}
	packedState := uint64(time.Now().Unix()<<8) | uint64(state)
	c.curState.Store(packedState)
	if !runHook {
		return
	}
	if hook := srv.ConnState; hook != nil {
		hook(nc, state)
	}
}

func (c *conn) Write(w []byte) (int, error) {
	//c.mu.Lock()
	//defer c.mu.Unlock()
	return c.rwc.Write(w)
}

func (c *conn) getState() (state ConnState, unixSec int64) {
	packedState := c.curState.Load()
	return ConnState(packedState & 0xFF), int64(packedState >> 8)
}

// Close the connection.
func (c *conn) close() {
	_ = c.rwc.Close()
}

// Serve a new connection.
func (c *conn) serve(ctx context.Context) {
	if ra := c.rwc.RemoteAddr(); ra != nil {
		c.remoteAddr = ra.String()
	}

	defer func() {
		if err := recover(); err != nil && err != ErrAbortHandler {
			buf := make([]byte, size)
			buf = buf[:runtime.Stack(buf, false)]
			log.Printf("mqtt: panic serving %v: %v", c.remoteAddr, err)
			log.Printf("%s", buf)
		}
		c.server.memorySubscribed.Unsubscribe(c)
		c.close()
		c.setState(c.rwc, StateClosed, true)
		if c.willTopic == "" || c.willPayload == nil {
			return
		}
		_ = c.server.memorySubscribed.Publish(&packet.Message{TopicName: c.willTopic, Content: c.willPayload})
	}()
	// TODO: TLS handle
	if tlsConn, ok := c.rwc.(*tls.Conn); ok {

		tlsTO := 10 * time.Second //c.server.tlsHandshakeTimeout()
		if tlsTO > 0 {
			dl := time.Now().Add(tlsTO)
			_ = c.rwc.SetReadDeadline(dl)
			_ = c.rwc.SetWriteDeadline(dl)
		}
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			// If the handshake failed due to the client not speaking
			// TLS, assume they're speaking plaintext HTTP and write a
			// 400 response on the TLS conn is underlying net.Conn.
			var reason string
			if re, ok := err.(tls.RecordHeaderError); ok && re.Conn != nil {
				_, _ = io.WriteString(re.Conn, "HTTP/1.0 400 Bad Request\r\n\r\nClient sent an HTTP request to an HTTPS server.\n")
				_ = re.Conn.Close()
				reason = "client sent an HTTP request to an HTTPS server"
			} else {
				reason = err.Error()
			}
			log.Printf("mqtt: TLS handshake error from %s: %v", c.rwc.RemoteAddr(), reason)
			return
		}
		// Restore Conn-level deadlines.
		if tlsTO > 0 {
			_ = c.rwc.SetReadDeadline(time.Time{})
			_ = c.rwc.SetWriteDeadline(time.Time{})
		}
		c.tlsState = new(tls.ConnectionState)
		*c.tlsState = tlsConn.ConnectionState()
	}
	ctx, cancel := context.WithCancel(ctx)
	c.cancelCtx = cancel
	defer cancel()

	for {
		rw, err := c.readRequest(ctx)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			return
		}
		serverHandler{c.server}.ServeMQTT(rw, rw.packet)
		c.setState(c.rwc, StateIdle, true)
	}
}

// Read next request from connection.
func (c *conn) readRequest(_ context.Context) (*response, error) {
	w, err := &response{conn: c}, error(nil)
	w.packet, err = packet.Unpack(c.version, c.rwc)
	stat.PacketReceived.Inc()
	if err != nil {
		pktLog.Printf("recv|%s - %s, err=%v", Kind[w.packet.Kind()], c.ID, err)
		return nil, fmt.Errorf("makeRequest: err=%w", err)
	}
	//pktLog.Printf("recv|%s - %s", Kind[w.packet.Kind()], c.ID)

	return w, nil
}

type defaultHandler struct{}

func (defaultHandler) ServeMQTT(w ResponseWriter, req packet.Packet) {
	var spkt packet.Packet
	c := w.(*response).conn
	switch rpkt := req.(type) {
	case *packet.CONNECT:
		c.version, c.ID = rpkt.Version, rpkt.ClientID
		connack := &packet.CONNACK{
			FixedHeader: &packet.FixedHeader{Version: c.version, Kind: CONNACK},
		}

		// 这里没有回CONNACK的话，客户端会重试, 如果CONNACK里面的Code!=0, 客户端直接会字节报错
		// TODO: password rewrite
		password, ok := CONFIG.GetAuth(rpkt.Username)
		if !ok || password != rpkt.Password {
			if rpkt.Version == packet.VERSION500 {
				connack.ConnectReturnCode = packet.ErrMalformedUsernameOrPassword
			} else {
				connack.ConnectReturnCode = packet.ErrBadUsernameOrPassword
			}
		}
		c.ID, c.version, c.willTopic, c.willPayload = rpkt.ClientID, rpkt.Version, rpkt.WillTopic, rpkt.WillPayload
		spkt = connack
	case *packet.PUBLISH:
		switch rpkt.QoS {
		case 0:
			_ = c.server.memorySubscribed.Publish(rpkt.Message)
			return
		case 1:
			_ = c.server.memorySubscribed.Publish(rpkt.Message)
			spkt = &packet.PUBACK{FixedHeader: &packet.FixedHeader{Version: c.version, Kind: PUBACK}, PacketID: rpkt.PacketID}
		case 2:
			// client    -----PUBLISH[QoS2]-----> server
			// client    <----PUBREC------------- server
			// client    -----PUBREL------------> server
			// client    <----PUBCOMP------------ server
			c.inFight.Put(rpkt)
			spkt = &packet.PUBREC{FixedHeader: &packet.FixedHeader{Version: c.version, Kind: PUBREC}, PacketID: rpkt.PacketID}
		}
	case *packet.PUBACK: // TODO:如果服务端作为client转发数据，也需要遵循qos的逻辑
		return
	case *packet.PUBREC:
		spkt = &packet.PUBREL{FixedHeader: &packet.FixedHeader{Version: c.version, Kind: PUBREL, QoS: 1}, PacketID: rpkt.PacketID}
	case *packet.PUBREL:
		pub, ok := c.inFight.Get(rpkt.PacketID)
		if !ok {
			panic("inFight not found packetID")
		}
		_ = c.server.memorySubscribed.Publish(pub.Message)
		spkt = &packet.PUBCOMP{FixedHeader: &packet.FixedHeader{Version: c.version, Kind: PUBCOMP}, PacketID: rpkt.PacketID}
	case *packet.PUBCOMP:
		return
	case *packet.SUBSCRIBE:
		var reasons []packet.ReasonCode
		for _, subscribe := range rpkt.Subscriptions {
			if err := c.subscribeTopics.Subscribe(subscribe.TopicFilter); err != nil {
				log.Printf("subscribeTopics.Subscribe: err=%v", err)
				reasons = append(reasons, packet.ErrTopicNameInvalid)
			} else {
				reasons = append(reasons, packet.ReasonCode{Code: subscribe.MaximumQoS})
			}
		}

		c.server.memorySubscribed.Subscribe(c)
		spkt = &packet.SUBACK{FixedHeader: &packet.FixedHeader{Version: c.version, Kind: SUBACK}, PacketID: rpkt.PacketID, ReasonCode: reasons}
	case *packet.UNSUBSCRIBE:
		for _, subscribe := range rpkt.Subscriptions {
			c.subscribeTopics.Unsubscribe(subscribe.TopicFilter)
		}
		c.server.memorySubscribed.Unsubscribe(c)
		spkt = &packet.UNSUBACK{FixedHeader: &packet.FixedHeader{Version: c.version, Kind: UNSUBACK, QoS: 1}, PacketID: rpkt.PacketID}
	case *packet.PINGREQ:
		// 服务端必须发送 PINGRESP报文响应客户端的PINGREQ报文 [MQTT-3.12.4-1]。
		spkt = &packet.PINGRESP{FixedHeader: &packet.FixedHeader{Version: c.version, Kind: PINGRESP}}
	case *packet.DISCONNECT:
		c.willTopic, c.willPayload = "", nil // 服务端在收到DISCONNECT报文时: 必须丢弃任何与当前连接关联的未发布的遗嘱消息，具体描述见 3.1.2.5节 [MQTT-3.14.4-3]。
		panic(ErrAbortHandler)               // 服务端在收到DISCONNECT报文时: 应该关闭网络连接，如果客户端 还没有这么做。
	case *packet.AUTH:
	default:
		panic("unknown packet type")
	}
	if err := w.OnSend(spkt); err != nil {
		log.Printf("mqtt-onSend: err=%v", err)
	}
}
