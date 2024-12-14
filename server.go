package mqtt

import (
	"context"
	"crypto/tls"
	"errors"
	"github.com/golang-io/mqtt/packet"
	"github.com/golang-io/mqtt/topic"
	"golang.org/x/net/websocket"
	"log"
	"math/rand"
	"net"
	"net/url"
	"sync"
	"sync/atomic"
	"time"
)

// shutdownPollIntervalMax is the max polling interval when checking
// quiescence during Server.Shutdown. Polling starts with a small
// interval and backs off to the max.
// Ideally we could find a solution that doesn't involve polling,
// but which also doesn't have a high runtime cost (and doesn't
// involve any contentious mutexes), but that is left as an
// exercise for the reader.
const shutdownPollIntervalMax = 500 * time.Millisecond
const size = 64 << 10

// A Handler responds to an MQTT request.
type Handler interface {
	ServeMQTT(ResponseWriter, packet.Packet)
}

type HandlerFunc func(ResponseWriter, packet.Packet)

func (f HandlerFunc) ServeMQTT(rw ResponseWriter, r packet.Packet) {
	f(rw, r)
}

type serverHandler struct {
	s *Server
}

func (s serverHandler) ServeMQTT(rw ResponseWriter, p packet.Packet) {
	handler := s.s.Handler
	if handler == nil {
		handler = defaultHandler{}
	}
	handler.ServeMQTT(rw, p)
}

type ResponseWriter interface {
	OnSend(request packet.Packet) error
}

// response represents the server side of an HTTP response.
type response struct {
	conn   *conn
	packet packet.Packet // request for this response
}

func (w *response) OnSend(pkt packet.Packet) error {
	w.conn.mu.Lock()
	defer w.conn.mu.Unlock()
	stat.PacketSent.Inc()
	return pkt.Pack(w.conn)
}

const (
	// StateNew represents a new connection that is expected to
	// send a request immediately. Connections begin at this
	// state and then transition to either StateActive or
	// StateClosed.
	StateNew ConnState = iota

	// StateActive represents a connection that has read 1 or more
	// bytes of a request. The Server.ConnState hook for
	// StateActive fires before the request has entered a handler
	// and doesn't fire again until the request has been
	// handled. After the request is handled, the state
	// transitions to StateClosed, StateHijacked, or StateIdle.
	// For HTTP/2, StateActive fires on the transition from zero
	// to one active request, and only transitions away once all
	// active requests are complete. That means that ConnState
	// cannot be used to do per-request work; ConnState only notes
	// the overall state of the connection.
	StateActive

	// StateIdle represents a connection that has finished
	// handling a request and is in the keep-alive state, waiting
	// for a new request. Connections transition from StateIdle
	// to either StateActive or StateClosed.
	StateIdle

	// StateHijacked represents a hijacked connection.
	// This is a terminal state. It does not transition to StateClosed.
	StateHijacked

	// StateClosed represents a closed connection.
	// This is a terminal state. Hijacked connections do not
	// transition to StateClosed.
	StateClosed
)

// ErrAbortHandler is a sentinel panic value to abort a handler.
// While any panic from ServeHTTP aborts the response to the client,
// panicking with ErrAbortHandler also suppresses logging of a stack
// trace to the server's error log.
var ErrAbortHandler = errors.New("mqtt: abort Handler")

// A ConnState represents the state of a client connection to a server.
// It's used by the optional [Server.ConnState] hook.
type ConnState int

// A Server defines parameters for running an HTTP server.
// The zero value for Server is a valid configuration.
type Server struct {
	Handler          Handler
	WebsocketHandler websocket.Handler

	// TLSConfig optionally provides a TLS configuration for use
	// by ServeTLS and ListenAndServeTLS. Note that this value is
	// cloned by ServeTLS and ListenAndServeTLS, so it's not
	// possible to modify the configuration with methods like
	// tls.Config.SetSessionTicketKeys. To use
	// SetSessionTicketKeys, use Server.Serve with a TLS Listener
	// instead.
	TLSConfig *tls.Config

	// ConnState specifies an optional callback function that is
	// called when a client connection changes state. See the
	// ConnState type and associated constants for details.
	ConnState func(net.Conn, ConnState)

	// ConnContext optionally specifies a function that modifies
	// the context used for a new connection c. The provided ctx
	// is derived from the base context and has a ServerContextKey
	// value.
	ConnContext func(ctx context.Context, c net.Conn) context.Context

	inShutdown atomic.Bool // true when server is in shutdown

	mu            sync.RWMutex
	listeners     map[*net.Listener]struct{}
	activeConn    map[*conn]struct{} // ClientID:conn
	onShutdown    []func()
	listenerGroup sync.WaitGroup

	memorySubscribed *MemorySubscribed // 订阅列表
}

func NewServer(ctx context.Context) *Server {
	s := &Server{
		activeConn: make(map[*conn]struct{}),
		listeners:  make(map[*net.Listener]struct{}),
	}
	s.memorySubscribed = NewMemorySubscribed(s)

	go func() {
		select {
		case <-ctx.Done():
			if err := s.Shutdown(ctx); err != nil {
				panic(err)
			}
		}
	}()
	return s
}

func (s *Server) Shutdown(ctx context.Context) error {
	s.inShutdown.Store(true)
	s.mu.Lock()
	lnerr := s.closeListenersLocked()
	for _, f := range s.onShutdown {
		go f()
	}
	s.mu.Unlock()
	s.listenerGroup.Wait()

	pollIntervalBase := time.Millisecond
	nextPollInterval := func() time.Duration {
		// Add 10% jitter.
		interval := pollIntervalBase + time.Duration(rand.Intn(int(pollIntervalBase/10)))
		// Double and clamp for next time.
		pollIntervalBase *= 2
		if pollIntervalBase > shutdownPollIntervalMax {
			pollIntervalBase = shutdownPollIntervalMax
		}
		return interval
	}

	timer := time.NewTimer(nextPollInterval())
	defer timer.Stop()
	for {
		if s.closeIdleConns() {
			return lnerr
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer.C:
			timer.Reset(nextPollInterval())
		}
	}
}

// closeIdleConns closes all idle connections and reports whether the
// server is quiescent.
func (s *Server) closeIdleConns() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	quiescent := true
	for c := range s.activeConn {
		st, unixSec := c.getState()
		// Issue 22682: treat StateNew connections as if
		// they're idle if we haven't read the first request's
		// header in over 5 seconds.
		if st == StateNew && unixSec < time.Now().Unix()-5 {
			st = StateIdle
		}
		if st != StateIdle || unixSec == 0 {
			// Assume unixSec == 0 means it's a very new
			// connection, without state set yet.
			quiescent = false
			continue
		}
		_ = c.rwc.Close()
		delete(s.activeConn, c)
	}
	return quiescent
}
func (s *Server) closeListenersLocked() error {
	var err error
	for ln := range s.listeners {
		if cerr := (*ln).Close(); cerr != nil && err == nil {
			err = cerr
		}
	}
	return err
}

// Create new connection from rwc.
func (s *Server) newConn(rwc net.Conn) *conn {
	c := &conn{server: s, rwc: rwc, subscribeTopics: topic.NewMemoryTrie(), inFight: newInFight()}
	return c
}

// Serve accepts incoming connections on the Listener l, creating a
// new service goroutine for each. The service goroutines read requests and
// then call srv.Handler to reply to them.
//
// HTTP/2 support is only enabled if the Listener returns [*tls.Conn]
// connections. and they were configured with "h2" in the TLS
// Config.NextProtos.
//
// Serve always returns a non-nil error and closes l.
// After [Server.Shutdown] or [Server.Close], the returned error is [ErrServerClosed].
func (s *Server) Serve(l net.Listener) error {
	defer l.Close()

	if !s.trackListener(&l, true) {
		return ErrServerClosed
	}
	defer s.trackListener(&l, false)

	ctx := context.Background()

	for {
		rw, err := l.Accept()
		if err != nil {
			if s.shuttingDown() {
				return errors.New("mqtt: Server closed")
			}
			return err
		}
		connCtx := ctx
		if cc := s.ConnContext; cc != nil {
			connCtx = cc(connCtx, rw)
			if connCtx == nil {
				panic("ConnContext returned nil")
			}
		}
		c := s.newConn(rw)
		c.setState(c.rwc, StateNew, true) // before Serve can return
		go c.serve(ctx)
	}
}

func (s *Server) trackConn(c *conn, add bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if add {
		stat.ActiveConnections.Inc()
		s.activeConn[c] = struct{}{}
	} else {
		stat.ActiveConnections.Dec()
		delete(s.activeConn, c)
	}
}

// trackListener adds or removes a net.Listener to the set of tracked
// listeners.
//
// We store a pointer to interface in the map set, in case the
// net.Listener is not comparable. This is safe because we only call
// trackListener via Serve and can track+defer untrack the same
// pointer to local variable there. We never need to compare a
// Listener from another caller.
//
// It reports whether the server is still up (not Shutdown or Closed).
func (s *Server) trackListener(ln *net.Listener, add bool) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listeners == nil {
		s.listeners = make(map[*net.Listener]struct{})
	}
	if add {
		if s.shuttingDown() {
			return false
		}
		s.listeners[ln] = struct{}{}
		s.listenerGroup.Add(1)
	} else {
		delete(s.listeners, ln)
		s.listenerGroup.Done()
	}
	return true
}
func (s *Server) shuttingDown() bool {
	return s.inShutdown.Load()
}

// ErrServerClosed is returned by the [Server.Serve], [ServeTLS], [ListenAndServe],
// and [ListenAndServeTLS] methods after a call to [Server.Shutdown] or [Server.Close].
var ErrServerClosed = errors.New("mqtt: Server closed")

func (s *Server) ListenAndServe(opts ...Option) error {
	options := newOptions(opts...)
	if s.shuttingDown() {
		return ErrServerClosed
	}
	u, err := url.Parse(options.URL)
	if err != nil {
		return err
	}

	ln, err := net.Listen("tcp", u.Host)
	if err != nil {
		return err
	}
	log.Printf("mqtt serve: %s", u.Host)
	return s.Serve(ln)
}

func (s *Server) ServeTLS(l net.Listener, certFile, keyFile string) error {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return err
	}
	config := &tls.Config{Certificates: []tls.Certificate{cert}}
	tlsListener := tls.NewListener(l, config)
	return s.Serve(tlsListener)
}

func (s *Server) ListenAndServeTLS(certFile, keyFile string, opts ...Option) error {
	if s.shuttingDown() {
		return ErrServerClosed
	}
	options := newOptions(opts...)
	u, err := url.Parse(options.URL)
	if err != nil {
		return err
	}

	ln, err := net.Listen("tcp", u.Host)
	if err != nil {
		return err
	}
	log.Printf("mqtt(s) serve: %s", u.Host)
	return s.ServeTLS(ln, certFile, keyFile)
}

// ListenAndServeWebsocket TODO
func (s *Server) ListenAndServeWebsocket(opts ...Option) error {
	if s.shuttingDown() {
		return ErrServerClosed
	}
	options := newOptions(opts...)
	u, err := url.Parse(options.URL)
	if err != nil {
		return err
	}
	s.WebsocketHandler = func(ws *websocket.Conn) {
		ws.PayloadType = websocket.BinaryFrame
		c := s.newConn(ws)
		c.setState(c.rwc, StateNew, true) // before Serve can return
		c.serve(context.Background())
	}

	ln, err := net.Listen("tcp", u.Host)
	if err != nil {
		return err
	}
	log.Printf("websocket serve: %s[todo]", u.Host)
	return s.Serve(ln)
}
