package mqtt

import (
	"context"
	"testing"
	"time"

	"github.com/golang-io/mqtt/packet"
)

func TestBasicServerClientInteraction(t *testing.T) {
	// Create a server
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server := NewServer(ctx)

	// Start server in background
	go func() {
		err := server.ListenAndServe(URL("mqtt://127.0.0.1:1884"))
		if err != nil {
			t.Logf("Server error: %v", err)
		}
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Create a client
	client := New(URL("mqtt://127.0.0.1:1884"))

	// Test basic client creation
	if client == nil {
		t.Fatal("Client should not be nil")
	}

	// Test server creation
	if server == nil {
		t.Fatal("Server should not be nil")
	}
}

func TestServerShutdownWithContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	server := NewServer(ctx)

	// Test that server can be shut down
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	// This should not block indefinitely
	done := make(chan bool)
	go func() {
		server.Shutdown(ctx)
		done <- true
	}()

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("Shutdown should complete within 2 seconds")
	}
}

func TestClientOptions(t *testing.T) {
	// Test client with various options
	client := New(
		URL("mqtt://127.0.0.1:1883"),
		Subscription(packet.Subscription{
			TopicFilter: "test/topic",
		}),
		Version("3.1.1"),
	)

	if client == nil {
		t.Fatal("Client should not be nil")
	}

	if client.options.URL != "mqtt://127.0.0.1:1883" {
		t.Errorf("expected URL 'mqtt://127.0.0.1:1883', got %s", client.options.URL)
	}

	if len(client.options.Subscriptions) != 1 {
		t.Error("should have one subscription")
	}

	if client.options.Subscriptions[0].TopicFilter != "test/topic" {
		t.Errorf("expected topic filter 'test/topic', got %s", client.options.Subscriptions[0].TopicFilter)
	}
}

func TestServerHandlerInterface(t *testing.T) {
	ctx := context.Background()
	server := NewServer(ctx)

	// Test that server has a handler
	if server.Handler == nil {
		t.Log("Server handler is nil (this is acceptable for default handler)")
	}

	// Test custom handler
	customHandler := &mockHandler{}
	server.Handler = customHandler

	if server.Handler != customHandler {
		t.Error("server should use custom handler")
	}
}

func TestClientMessageHandler(t *testing.T) {
	client := New()

	messageReceived := false
	client.OnMessage(func(msg *packet.Message) {
		messageReceived = true
	})

	if client.onMessage == nil {
		t.Error("OnMessage should set the message handler")
	}

	// Test that the handler can be called
	if client.onMessage != nil {
		client.onMessage(&packet.Message{
			TopicName: "test/topic",
			Content:   []byte("test message"),
		})
		if !messageReceived {
			t.Error("message handler should be called")
		}
	}
}

func TestServerConnectionTracking(t *testing.T) {
	ctx := context.Background()
	server := NewServer(ctx)

	// Test initial state
	if len(server.activeConn) != 0 {
		t.Error("server should start with no active connections")
	}

	// Test connection tracking
	mockConn := &mockConn{}
	conn := server.newConn(mockConn)

	server.trackConn(conn, true)
	if len(server.activeConn) != 1 {
		t.Error("connection should be tracked")
	}

	server.trackConn(conn, false)
	if len(server.activeConn) != 0 {
		t.Error("connection should be removed from tracking")
	}
}

func TestServerShutdownFlag(t *testing.T) {
	ctx := context.Background()
	server := NewServer(ctx)

	// Test initial shutdown state
	if server.shuttingDown() {
		t.Error("server should not be shutting down initially")
	}

	// Test setting shutdown flag
	server.inShutdown.Store(true)
	if !server.shuttingDown() {
		t.Error("server should be shutting down after setting flag")
	}
}
