package mqtt

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net"
	"net/url"
	"time"

	"github.com/golang-io/mqtt/packet"
	"golang.org/x/net/websocket"
	"golang.org/x/sync/errgroup"
)

// A Client is an MQTT client. Its zero value ([DefaultClient]) is a usable client that uses [DefaultTransport].
//
// The [Client.Transport] typically has internal state (cached TCP
// connections), so Clients should be reused instead of created as needed.
// Clients are safe for concurrent use by multiple goroutines.
//
// A Client is higher-level than a [RoundTripper] (such as [Transport])
// and additionally handles HTTP details such as cookies and redirects.
type Client struct {
	// URL specifies either the URI being requested (for server requests) or the URL to access (for client requests).
	//
	// For server requests, the URL is parsed from the URI supplied on the Request-Line as stored in RequestURI.
	// For most requests, fields other than Path and RawQuery will be empty. (See RFC 7230, Section 5.3)
	//
	// For client requests, the URL's Host specifies the server to
	// connect to, while the Request's Host field optionally
	// specifies the Host header value to send in the MQTT request.
	URL *url.URL

	conn *conn

	// DialContext specifies the dial function for creating unencrypted TCP connections.
	// If DialContext is nil (and the deprecated Dial below is also nil), then the transport dials using package net.
	//
	// DialContext runs concurrently with calls to RoundTrip.
	// A RoundTrip call that initiates a dial may end up using
	// a connection dialed previously when the earlier connection
	// becomes idle before the later DialContext completes.
	DialContext func(ctx context.Context, network, addr string) (net.Conn, error)

	// DialTLSContext specifies an optional dial function for creating TLS connections for non-proxied HTTPS requests.
	//
	// If DialTLSContext is nil (and the deprecated DialTLS below is also nil), DialContext and TLSClientConfig are used.
	//
	// If DialTLSContext is set, the Dial and DialContext hooks are not used for HTTPS
	// requests and the TLSClientConfig and TLSHandshakeTimeout are ignored.
	// The returned net.Conn is assumed to already be past the TLS handshake.
	DialTLSContext func(ctx context.Context, network, addr string) (net.Conn, error)

	// TLSClientConfig specifies the TLS configuration to use with tls.Client.
	// If nil, the default configuration is used.
	// If non-nil, HTTP/2 support may not be enabled by default.
	TLSClientConfig *tls.Config

	// TLSHandshakeTimeout specifies the maximum amount of time to wait for a TLS handshake. Zero means no timeout.
	TLSHandshakeTimeout time.Duration

	// Timeout specifies a time limit for requests made by this Client.
	// The timeout includes connection time, any redirects, and reading the response body.
	// The timer remains running after Get, Head, Post, or Do return and will interrupt reading of the Response.Body.
	//
	// A Timeout of zero means no timeout.
	//
	// The Client cancels requests to the underlying Transport as if the Request's Context ended.
	//
	// For compatibility, the Client will also use the deprecated CancelRequest method on Transport if found.
	// New RoundTripper implementations should use the Request's Context
	// for cancellation instead of implementing CancelRequest.
	Timeout time.Duration

	options Options
	recv    [0xF + 1]chan packet.Packet
	version byte
	// cancel  context.CancelFunc

	onMessage func(*packet.Message)
}

func (c *Client) ID() string {
	return c.conn.ID
}

// RoundTrip implements the [RoundTripper] interface.
//
// For higher-level HTTP client support (such as handling of cookies
// and redirects), see [Get], [Post], and the [Client] type.
//
// Like the RoundTripper interface, the error types returned
// by RoundTrip are unspecified.
func (c *Client) RoundTrip(req packet.Packet) (packet.Packet, error) {
	return c.roundTrip(req)
}

// roundTrip implements a RoundTripper over MQTT.
func (c *Client) roundTrip(req packet.Packet) (packet.Packet, error) {
	ctx := context.Background()

	if c.conn == nil {
		con, err := c.dial(ctx, c.URL.Scheme, c.URL.Host)
		if err != nil {
			return nil, err
		}
		c.conn = &conn{rwc: con, remoteAddr: con.RemoteAddr().String()}
	}
	err := req.Pack(c.conn.rwc)
	if err != nil {
		return nil, err
	}
	log.Printf("todo: t.roundTrip need handle and recv response\n")
	return nil, nil
}

func (c *Client) dial(ctx context.Context, scheme, addr string) (net.Conn, error) {
	// 用户自定义拨号优先
	if c.DialContext != nil && (scheme == "tcp" || scheme == "mqtt") {
		con, err := c.DialContext(ctx, "tcp", addr)
		if con == nil && err == nil {
			err = errors.New("mqtt: Transport.DialContext hook returned (nil, nil)")
		}
		return con, err
	}
	if c.DialTLSContext != nil && (scheme == "tls" || scheme == "mqtts") {
		con, err := c.DialTLSContext(ctx, "tcp", addr)
		if con == nil && err == nil {
			err = errors.New("mqtt: Transport.DialTLSContext hook returned (nil, nil)")
		}
		return con, err
	}

	switch scheme {
	case "mqtt", "tcp":
		return (&net.Dialer{}).DialContext(ctx, "tcp", addr)
	case "mqtts", "tls":
		return tls.DialWithDialer(&net.Dialer{}, "tcp", addr, c.TLSClientConfig)
	case "ws", "wss":
		// 构造 WebSocket URL，默认路径 /mqtt
		path := c.URL.Path
		if path == "" {
			path = "/mqtt"
		}
		loc := &url.URL{Scheme: scheme, Host: addr, Path: path}
		// 兼容 Origin 要求
		originScheme := "http"
		if scheme == "wss" {
			originScheme = "https"
		}
		origin := &url.URL{Scheme: originScheme, Host: addr}

		cfg, err := websocket.NewConfig(loc.String(), origin.String())
		if err != nil {
			return nil, err
		}
		// 协商 mqtt 子协议，二进制帧
		cfg.Protocol = []string{"mqtt"}
		if scheme == "wss" {
			cfg.TlsConfig = c.TLSClientConfig
		}
		ws, err := websocket.DialConfig(cfg)
		if err != nil {
			return nil, err
		}
		ws.PayloadType = websocket.BinaryFrame
		return ws, nil
	default:
		// 兜底按 tcp 处理
		return (&net.Dialer{}).DialContext(ctx, "tcp", addr)
	}
}

func New(opts ...Option) *Client {
	options := newOptions(opts...)
	var err error
	client := &Client{
		options: options,
		conn:    &conn{inFight: newInFight()},
		recv:    [0xF + 1]chan packet.Packet{},
		version: packet.VERSION311,
	}

	for i := 1; i <= 0xF; i++ {
		client.recv[i] = make(chan packet.Packet, 1)
	}

	client.recv[PUBLISH] = make(chan packet.Packet, 10000)

	if client.URL, err = url.Parse(options.URL); err != nil {
		panic(err)
	}

	// 记录客户端创建日志
	log.Printf("[CLIENT_CREATED] MQTT client created - ClientID: %s, Server: %s",
		options.ClientID, options.URL)

	return client
}

func (c *Client) Close() error {
	// 记录客户端关闭日志
	log.Printf("[CLIENT_CLOSED] MQTT client closed - ClientID: %s", c.conn.ID)

	for i := 1; i <= 0xF; i++ {
		close(c.recv[i])
	}
	return nil
}

func (c *Client) unpack(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		pkt, err := packet.Unpack(c.version, c.conn.rwc)
		if err != nil {
			log.Printf("[UNPACK_ERROR] Client packet unpack error - ClientID: %s, Error: %v", c.conn.ID, err)
			return err
		}
		c.recv[pkt.Kind()] <- pkt
	}
}

func (c *Client) Connect(ctx context.Context) error {
	// 记录连接尝试日志
	log.Printf("client attempting to connect: client_id=%s, server=%s", c.options.ClientID, c.URL.Host)

	connect := packet.CONNECT{FixedHeader: &packet.FixedHeader{
		Version: c.version,
		Kind:    CONNECT,
	}, ClientID: c.options.ClientID}
	if err := connect.Pack(c.conn.rwc); err != nil {
		log.Printf("client connect packet send failed: client_id=%s, error=%v", c.options.ClientID, err)
		return err
	}
	c.conn.ID = connect.ClientID

	select {
	case <-ctx.Done():
		log.Printf("client connect timeout: client_id=%s", c.options.ClientID)
		return ctx.Err()
	case pkt, ok := <-c.recv[CONNACK]:
		if !ok {
			return ctx.Err()
		}
		connack, ok := pkt.(*packet.CONNACK)
		if !ok || connack.Kind() != CONNACK {
			log.Printf("client received invalid CONNACK packet: client_id=%s", c.options.ClientID)
			return errors.New("mqtt: invalid packet received")
		}

		if connack.ReturnCode.Code != 0 {
			log.Printf("client connect failed: client_id=%s, return_code=%v", c.options.ClientID, connack.ReturnCode)
			return errors.New("mqtt: connect returned non-zero return code")
		}
		log.Printf("client connected successfully: client_id=%s, server=%s", c.options.ClientID, c.URL.Host)
	}
	return nil
}

func (c *Client) Subscribe(ctx context.Context) error {
	// 记录订阅尝试日志
	var topics []string
	for _, sub := range c.options.Subscriptions {
		topics = append(topics, sub.TopicFilter)
	}
	log.Printf("client attempting to subscribe: client_id=%s, topics=%v", c.options.ClientID, topics)

	sub := packet.SUBSCRIBE{
		FixedHeader:   &packet.FixedHeader{Version: c.version, Kind: SUBSCRIBE, QoS: 1},
		PacketID:      1,
		Subscriptions: c.options.Subscriptions,
	}
	if err := sub.Pack(c.conn.rwc); err != nil {
		log.Printf("client subscribe packet send failed: client_id=%s, error=%v", c.options.ClientID, err)
		return err
	}

	select {
	case <-ctx.Done():
		log.Printf("client subscribe timeout: client_id=%s", c.options.ClientID)
		return ctx.Err()
	case pkt, ok := <-c.recv[SUBACK]:
		if !ok {
			return ctx.Err()
		}
		suback, ok := pkt.(*packet.SUBACK)
		if !ok || suback.Kind() != SUBACK {
			log.Printf("client received invalid SUBACK packet: client_id=%s", c.options.ClientID)
			return errors.New("mqtt: invalid packet received")
		}
		for _, reason := range suback.ReasonCode {
			if reason.Code != 0 {
				log.Printf("client subscribe failed: client_id=%s, reason_code=%v", c.options.ClientID, reason)
				return errors.New("mqtt: connect returned non-zero return code")
			}
		}
		log.Printf("client subscribed successfully: client_id=%s, topics=%v", c.options.ClientID, topics)
	}
	return nil
}

func (c *Client) ServeMessageLoop(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		if err := c.ServeMessage(ctx); err != nil {
			return err
		}
	}
}

func (c *Client) OnMessage(fn func(*packet.Message)) {
	c.onMessage = fn
}
func (c *Client) SubmitMessage(message *packet.Message) error {
	if c.conn.rwc == nil {
		log.Printf("client publish: client_id=%s, error=connect is nil", c.options.ClientID)
		return errors.New("mqtt: connect is nil")
	}

	// 记录发布消息日志
	log.Printf("client publish: client_id=%s, topic=%s, size=%d", c.options.ClientID, message.TopicName, len(message.Content))
	pub := packet.PUBLISH{
		FixedHeader: &packet.FixedHeader{Version: c.version, Kind: PUBLISH},
		Message:     message,
	}

	if pub.QoS == 0 {

	}
	if pub.QoS == 1 || pub.QoS == 2 {
		pub.PacketID = c.conn.PacketID + 1
		c.conn.PacketID = pub.PacketID
	}

	if err := pub.Pack(c.conn.rwc); err != nil {
		log.Printf("client publish: client_id=%s, topic=%s, error=%v", c.options.ClientID, message.TopicName, err)
		return err
	}

	log.Printf("client publish: client_id=%s, topic=%s, success", c.options.ClientID, message.TopicName)
	return nil
}

func (c *Client) ServeMessage(ctx context.Context) error {
	var pub *packet.PUBLISH
	select {
	case <-ctx.Done():
		return ctx.Err()
	case pkt, ok := <-c.recv[PUBLISH]:
		if !ok {
			return fmt.Errorf("mqtt: invalid packet received")
		}
		pub, ok = pkt.(*packet.PUBLISH)
		if !ok {
			return errors.New("mqtt: invalid packet received")
		}

		// 记录接收消息日志
		log.Printf("client received: client_id=%s, topic=%s, qos=%d, size=%d", c.options.ClientID, pub.Message.TopicName, pub.QoS, len(pub.Message.Content))

		switch pub.QoS {
		case 0:
		case 1:
			puback := packet.PUBACK{
				FixedHeader: &packet.FixedHeader{Version: c.version, Kind: PUBACK},
				PacketID:    pub.PacketID,
			}
			if err := puback.Pack(c.conn.rwc); err != nil {
				log.Printf("client puback send failed: client_id=%s, packet_id=%d, error=%v", c.options.ClientID, pub.PacketID, err)
				return err
			}
			log.Printf("client puback sent: client_id=%s, packet_id=%d", c.options.ClientID, pub.PacketID)
		case 2:
			pubrec := packet.PUBREC{
				FixedHeader: &packet.FixedHeader{Version: c.version, Kind: PUBREC},
				PacketID:    pub.PacketID,
			}
			if err := pubrec.Pack(c.conn.rwc); err != nil {
				log.Printf("client pubrec send failed: client_id=%s, packet_id=%d, error=%v", c.options.ClientID, pub.PacketID, err)
				return err
			}
			log.Printf("client pubrec sent: client_id=%s, packet_id=%d", c.options.ClientID, pub.PacketID)
			c.conn.inFight.Put(pub)
			return nil
		}

	case pkt, ok := <-c.recv[PUBREL]:
		if !ok {
			return fmt.Errorf("mqtt: invalid packet received")
		}
		pubrel, ok := pkt.(*packet.PUBREL)
		if !ok {
			return errors.New("mqtt: invalid packet received")
		}
		pub, ok = c.conn.inFight.Get(pubrel.PacketID)
		if !ok {
			return errors.New("mqtt: invalid packet received")
		}
		pubcomp := packet.PUBCOMP{
			FixedHeader: &packet.FixedHeader{Version: c.version, Kind: PUBCOMP},
			PacketID:    pubrel.PacketID,
		}
		if err := pubcomp.Pack(c.conn.rwc); err != nil {
			log.Printf("client pubcomp send failed: client_id=%s, packet_id=%d, error=%v", c.options.ClientID, pubrel.PacketID, err)
			return err
		}
		log.Printf("client pubcomp sent: client_id=%s, packet_id=%d", c.options.ClientID, pubrel.PacketID)
	}
	go c.onMessage(pub.Message)
	return nil
}

func (c *Client) ConnectAndSubscribe(ctx context.Context) error {
	timer := time.NewTimer(0)
	defer timer.Stop()
	count := 0
	for {
		select {
		case <-ctx.Done():
			log.Printf("client context done: client_id=%s", c.options.ClientID)
			return ctx.Err()
		case <-timer.C:
			timer.Reset(3 * time.Second)
		}
		if err := c.connectAndSubscribe(ctx); err != nil {
			count++
			if count == 1 || count%10 == 0 {
				log.Printf("client connect and subscribe error[%d]: client_id=%s, error=%v", count, c.options.ClientID, err)
			}
		} else {
			count = 0
		}
	}
}

func (c *Client) connectAndSubscribe(ctx context.Context) error {
	var err error

	// 记录网络连接尝试日志
	log.Printf("client attempting to dial: client_id=%s, server=%s", c.options.ClientID, c.URL.Host)

	if c.conn.rwc, err = c.dial(ctx, c.URL.Scheme, c.URL.Host); err != nil {
		log.Printf("client dial failed: client_id=%s, server=%s, error=%v", c.options.ClientID, c.URL.Host, err)
		return err
	}

	log.Printf("client dialed successfully: client_id=%s, server=%s", c.options.ClientID, c.URL.Host)

	group, ctx := errgroup.WithContext(ctx)
	group.Go(func() error {
		return c.unpack(ctx)
	})
	group.Go(func() error {
		<-ctx.Done()
		return c.Disconnect()
	})

	group.Go(func() error {
		if err := c.Connect(ctx); err != nil {
			return err
		}
		if err := c.Subscribe(ctx); err != nil {
			return err
		}
		return c.ServeMessageLoop(ctx)
	})

	return group.Wait()
}

func (c *Client) Disconnect() error {
	// 记录断开连接日志
	log.Printf("client attempting to disconnect: client_id=%s", c.options.ClientID)

	disconnect := packet.DISCONNECT{
		FixedHeader: &packet.FixedHeader{Version: c.version, Kind: DISCONNECT},
	}
	if err := disconnect.Pack(c.conn.rwc); err != nil {
		log.Printf("client disconnect packet send failed: client_id=%s, error=%v", c.options.ClientID, err)
		return err
	}

	log.Printf("client disconnected successfully: client_id=%s", c.options.ClientID)
	return nil
}
