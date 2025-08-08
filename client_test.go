package mqtt

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/golang-io/mqtt/packet"
)

func TestNewClient(t *testing.T) {
	client := New(URL("mqtt://localhost:1883"))
	if client == nil {
		t.Fatal("New() should return a non-nil client")
	}
	if client.URL == nil {
		t.Fatal("client.URL should not be nil")
	}
	if client.URL.Host != "localhost:1883" {
		t.Errorf("expected host localhost:1883, got %s", client.URL.Host)
	}
}

func TestClientID(t *testing.T) {
	client := New()
	// Note: ClientID is set automatically in newOptions, so we test the default behavior
	if client.options.ClientID == "" {
		t.Error("ClientID should not be empty")
	}
}

func TestClientClose(t *testing.T) {
	client := New()
	err := client.Close()
	if err != nil {
		t.Errorf("Close() should not return error, got %v", err)
	}
}

func TestClientDial(t *testing.T) {
	client := New()

	// Test with nil DialContext
	conn, err := client.dial(context.Background(), "tcp", "localhost:1883")
	if err == nil {
		// If connection succeeds, close it
		if conn != nil {
			conn.Close()
		}
	}
	// We expect an error since localhost:1883 is not listening
	if err == nil {
		t.Log("Note: localhost:1883 might be listening, this is unexpected")
	}
}

func TestClientWithCustomDialer(t *testing.T) {
	dialCalled := false
	client := New()
	client.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		dialCalled = true
		return nil, nil
	}

	_, err := client.dial(context.Background(), "tcp", "localhost:1883")
	if !dialCalled {
		t.Error("custom dialer should be called")
	}
	// We expect an error since our custom dialer returns (nil, nil)
	if err == nil {
		t.Error("expected error from custom dialer returning (nil, nil)")
	}
}

func TestClientOnMessage(t *testing.T) {
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

func TestClientIDMethod(t *testing.T) {
	client := New()
	client.conn = &conn{ID: "test-client-123"}

	id := client.ID()
	if id != "test-client-123" {
		t.Errorf("expected ID 'test-client-123', got %s", id)
	}
}

func TestClientWithTimeout(t *testing.T) {
	timeout := 30 * time.Second
	client := New()
	client.Timeout = timeout

	if client.Timeout != timeout {
		t.Errorf("expected timeout %v, got %v", timeout, client.Timeout)
	}
}

func TestClientWithTLSConfig(t *testing.T) {
	client := New()

	if client.TLSClientConfig != nil {
		t.Error("TLSClientConfig should be nil when not configured")
	}
}

func TestClientRecvChannels(t *testing.T) {
	client := New()

	// Test that recv channels are properly initialized
	for i := 1; i <= 0xF; i++ {
		if client.recv[i] == nil {
			t.Errorf("recv[%d] should not be nil", i)
		}
	}

	// Test that PUBLISH channel has larger buffer
	if cap(client.recv[PUBLISH]) != 10000 {
		t.Errorf("PUBLISH channel should have capacity 10000, got %d", cap(client.recv[PUBLISH]))
	}
}
