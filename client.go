package mqtt

import (
	"context"
	"crypto/tls"
	"errors"
	"github.com/golang-io/mqtt/packet"
	"log"
	"net"
	"net/url"
	"time"
)

// A Client is an MQTT client. Its zero value ([DefaultClient]) is a
// usable client that uses [DefaultTransport].
//
// The [Client.Transport] typically has internal state (cached TCP
// connections), so Clients should be reused instead of created as
// needed. Clients are safe for concurrent use by multiple goroutines.
//
// A Client is higher-level than a [RoundTripper] (such as [Transport])
// and additionally handles HTTP details such as cookies and
// redirects.
type Client struct {
	// URL specifies either the URI being requested (for server
	// requests) or the URL to access (for client requests).
	//
	// For server requests, the URL is parsed from the URI
	// supplied on the Request-Line as stored in RequestURI.  For
	// most requests, fields other than Path and RawQuery will be
	// empty. (See RFC 7230, Section 5.3)
	//
	// For client requests, the URL's Host specifies the server to
	// connect to, while the Request's Host field optionally
	// specifies the Host header value to send in the MQTT
	// request.
	URL *url.URL

	conn *conn

	// DialContext specifies the dial function for creating unencrypted TCP connections.
	// If DialContext is nil (and the deprecated Dial below is also nil),
	// then the transport dials using package net.
	//
	// DialContext runs concurrently with calls to RoundTrip.
	// A RoundTrip call that initiates a dial may end up using
	// a connection dialed previously when the earlier connection
	// becomes idle before the later DialContext completes.
	DialContext func(ctx context.Context, network, addr string) (net.Conn, error)

	// DialTLSContext specifies an optional dial function for creating
	// TLS connections for non-proxied HTTPS requests.
	//
	// If DialTLSContext is nil (and the deprecated DialTLS below is also nil),
	// DialContext and TLSClientConfig are used.
	//
	// If DialTLSContext is set, the Dial and DialContext hooks are not used for HTTPS
	// requests and the TLSClientConfig and TLSHandshakeTimeout
	// are ignored. The returned net.Conn is assumed to already be
	// past the TLS handshake.
	DialTLSContext func(ctx context.Context, network, addr string) (net.Conn, error)

	// TLSClientConfig specifies the TLS configuration to use with
	// tls.Client.
	// If nil, the default configuration is used.
	// If non-nil, HTTP/2 support may not be enabled by default.
	TLSClientConfig *tls.Config

	// TLSHandshakeTimeout specifies the maximum amount of time to
	// wait for a TLS handshake. Zero means no timeout.
	TLSHandshakeTimeout time.Duration

	// Timeout specifies a time limit for requests made by this
	// Client. The timeout includes connection time, any
	// redirects, and reading the response body. The timer remains
	// running after Get, Head, Post, or Do return and will
	// interrupt reading of the Response.Body.
	//
	// A Timeout of zero means no timeout.
	//
	// The Client cancels requests to the underlying Transport
	// as if the Request's Context ended.
	//
	// For compatibility, the Client will also use the deprecated
	// CancelRequest method on Transport if found. New
	// RoundTripper implementations should use the Request's Context
	// for cancellation instead of implementing CancelRequest.
	Timeout time.Duration

	opts    []Option
	recv    [0xF + 1]chan packet.Packet
	version byte
	ctx     context.Context
	cancel  context.CancelFunc
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
		c.conn = &conn{rwc: con, remoteAddr: c.conn.rwc.RemoteAddr().String()}
	}
	err := req.Pack(c.conn.rwc)
	if err != nil {
		return nil, err
	}
	log.Printf("todo: t.roundTrip need handle and recv response\n")
	return nil, nil
}

func (c *Client) dial(ctx context.Context, network, addr string) (net.Conn, error) {
	if c.DialContext != nil {
		c, err := c.DialContext(ctx, network, addr)
		if c == nil && err == nil {
			err = errors.New("mqtt: Transport.DialContext hook returned (nil, nil)")
		}
		return c, err
	}
	if c.DialTLSContext != nil {
		c, err := c.DialTLSContext(ctx, network, addr)
		if c == nil && err == nil {
			err = errors.New("mqtt: Transport.DialTLSContext hook returned (nil, nil)")
		}
		return c, err
	}
	return (&net.Dialer{}).DialContext(ctx, network, addr)
}

func New(opts ...Option) *Client {
	options := newOptions(opts...)
	var err error
	client := &Client{
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
	return client
}

func (c *Client) Close() error {
	defer c.cancel()
	for i := 1; i <= 0xF; i++ {
		close(c.recv[i])
	}
	return nil
}

func (c *Client) handle_recv() {
	defer c.Close()
	for {
		pkt, err := packet.Unpack(c.version, c.conn.rwc)
		if err != nil {
			log.Printf("id=%s, mqtt: error unpacking packet: %v", c.conn.ID, err)
			continue
		}
		c.recv[pkt.Kind()] <- pkt
	}
}

func (c *Client) Connect(ctx context.Context, id string) error {
	var err error
	c.ctx, c.cancel = context.WithCancel(ctx)
	if c.conn.rwc, err = c.dial(ctx, "tcp", c.URL.Host); err != nil {
		return err
	}
	go c.handle_recv()
	connect := packet.CONNECT{FixedHeader: &packet.FixedHeader{
		Version: c.version,
		Kind:    CONNECT,
	}, ClientID: id}
	if err := connect.Pack(c.conn.rwc); err != nil {
		return err
	}
	c.conn.ID = connect.ClientID

	select {
	case <-ctx.Done():
	case pkt, ok := <-c.recv[CONNACK]:
		if !ok {
			return ctx.Err()
		}
		connack, ok := pkt.(*packet.CONNACK)
		if !ok || connack.Kind() != CONNACK {
			return errors.New("mqtt: invalid packet received")
		}

		if connack.ConnectReturnCode.Code != 0 {
			return errors.New("mqtt: connect returned non-zero return code")
		}
		log.Printf("connect ok!")
	}
	return nil
}

func (c *Client) Subscribe(ctx context.Context, subs []packet.Subscription, fn func(packet.Message) error) error {

	sub := packet.SUBSCRIBE{
		FixedHeader:   &packet.FixedHeader{Version: c.version, Kind: SUBSCRIBE, QoS: 1},
		PacketID:      1,
		Subscriptions: subs,
	}
	if err := sub.Pack(c.conn.rwc); err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case pkt, ok := <-c.recv[SUBACK]:
		if !ok {
			return ctx.Err()
		}
		suback, ok := pkt.(*packet.SUBACK)
		if !ok || suback.Kind() != SUBACK {
			return errors.New("mqtt: invalid packet received")
		}
		for _, reason := range suback.ReasonCode {
			if reason.Code != 0 {
				return errors.New("mqtt: connect returned non-zero return code")
			}
		}
	}
	for {
		select {
		case <-c.ctx.Done():
			return context.Cause(c.ctx)
		default:
		}
		if err := c.ReceiveAndServe(ctx, fn); err != nil {
			log.Printf("mqtt: error handling packet: %v", err)
			return err
		}
	}
}

func (c *Client) ReceiveAndServe(ctx context.Context, fn func(message packet.Message) error) error {
	var pub *packet.PUBLISH
	select {
	case pkt, ok := <-c.recv[PUBLISH]:
		if !ok {
			return ctx.Err()
		}
		pub, ok = pkt.(*packet.PUBLISH)
		if !ok {
			return errors.New("mqtt: invalid packet received")
		}
		switch pub.QoS {
		case 0:
		case 1:
			puback := packet.PUBACK{
				FixedHeader: &packet.FixedHeader{Version: c.version, Kind: PUBACK},
				PacketID:    pub.PacketID,
			}
			if err := puback.Pack(c.conn.rwc); err != nil {
				return err
			}
		case 2:
			pubrec := packet.PUBREC{
				FixedHeader: &packet.FixedHeader{Version: c.version, Kind: PUBREC},
				PacketID:    pub.PacketID,
			}
			if err := pubrec.Pack(c.conn.rwc); err != nil {
				return err
			}
			c.conn.inFight.Put(pub)
			return nil
		}

	case pkt, ok := <-c.recv[PUBREL]:
		if !ok {
			return ctx.Err()
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
			return err
		}
	}

	return fn(pub.Message)
}

func (c *Client) Publish(ctx context.Context, message packet.Message) error {
	pub := packet.PUBLISH{FixedHeader: &packet.FixedHeader{
		Version: c.version,
		Kind:    PUBLISH,
	}, Message: message}

	if pub.QoS == 0 {

	}
	if pub.QoS == 1 || pub.QoS == 2 {
		pub.PacketID = c.conn.PacketID + 1
		c.conn.PacketID = pub.PacketID
	}

	if err := pub.Pack(c.conn.rwc); err != nil {
		return err
	}
	return nil
}