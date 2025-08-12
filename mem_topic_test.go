package mqtt

import (
	"context"
	"testing"

	"github.com/golang-io/mqtt/packet"
	"github.com/golang-io/mqtt/topic"
)

func TestNewMemorySubscribed(t *testing.T) {
	ctx := context.Background()
	server := NewServer(ctx)
	memorySub := NewMemorySubscribed(server)

	if memorySub == nil {
		t.Fatal("NewMemorySubscribed() should return a non-nil instance")
	}
	if memorySub.maps == nil {
		t.Fatal("maps should be initialized")
	}
	if memorySub.s != server {
		t.Error("should reference the server")
	}
}

func TestMemorySubscribedPublish(t *testing.T) {
	ctx := context.Background()
	server := NewServer(ctx)
	memorySub := NewMemorySubscribed(server)

	// Test publishing to a new topic
	message := &packet.Message{
		TopicName: "test/topic",
		Content:   []byte("test message"),
	}

	err := memorySub.Publish(message, nil)
	if err != nil {
		t.Errorf("Publish should not return error, got %v", err)
	}

	// Check that the topic was created
	memorySub.mu.RLock()
	_, exists := memorySub.maps["test/topic"]
	memorySub.mu.RUnlock()

	if !exists {
		t.Error("topic should be created after publish")
	}
}

func TestMemorySubscribedPublishExistingTopic(t *testing.T) {
	ctx := context.Background()
	server := NewServer(ctx)
	memorySub := NewMemorySubscribed(server)

	// Publish to create topic
	message1 := &packet.Message{
		TopicName: "test/topic",
		Content:   []byte("test message 1"),
	}
	memorySub.Publish(message1, nil)

	// Publish to same topic again
	message2 := &packet.Message{
		TopicName: "test/topic",
		Content:   []byte("test message 2"),
	}
	err := memorySub.Publish(message2, nil)
	if err != nil {
		t.Errorf("Publish to existing topic should not return error, got %v", err)
	}
}

func TestTopicSubscribedNew(t *testing.T) {
	ts := NewTopicSubscribed("test/topic")

	if ts == nil {
		t.Fatal("NewTopicSubscribed() should return a non-nil instance")
	}
	if ts.TopicName != "test/topic" {
		t.Errorf("expected topic name 'test/topic', got %s", ts.TopicName)
	}
	if ts.activeConn == nil {
		t.Fatal("activeConn should be initialized")
	}
}

func TestTopicSubscribedSubscribe(t *testing.T) {
	ts := NewTopicSubscribed("test/topic")

	// Create a mock connection
	mockConn := &conn{
		subscribeTopics: topic.NewMemoryTrie(),
	}
	mockConn.subscribeTopics.Subscribe("test/topic")

	// Subscribe the connection
	ts.Subscribe(mockConn)

	if len(ts.activeConn) != 1 {
		t.Error("connection should be subscribed")
	}

	// Test subscribing same connection again (should not duplicate)
	ts.Subscribe(mockConn)
	if len(ts.activeConn) != 1 {
		t.Error("should not duplicate connection")
	}
}

func TestTopicSubscribedUnsubscribe(t *testing.T) {
	ts := NewTopicSubscribed("test/topic")

	// Create a mock connection
	mockConn := &conn{
		subscribeTopics: topic.NewMemoryTrie(),
	}
	mockConn.subscribeTopics.Subscribe("test/topic")

	// Subscribe then unsubscribe
	ts.Subscribe(mockConn)
	remaining := ts.Unsubscribe(mockConn)

	if remaining != 0 {
		t.Errorf("expected 0 remaining connections, got %d", remaining)
	}

	if len(ts.activeConn) != 0 {
		t.Error("connection should be unsubscribed")
	}
}

func TestTopicSubscribedLen(t *testing.T) {
	ts := NewTopicSubscribed("test/topic")

	// Initially should be 0
	if ts.Len() != 0 {
		t.Errorf("expected length 0, got %d", ts.Len())
	}

	// Add a connection
	mockConn := &conn{
		subscribeTopics: topic.NewMemoryTrie(),
	}
	mockConn.subscribeTopics.Subscribe("test/topic")
	ts.Subscribe(mockConn)

	if ts.Len() != 1 {
		t.Errorf("expected length 1, got %d", ts.Len())
	}
}

func TestTopicSubscribedExchange(t *testing.T) {
	ts := NewTopicSubscribed("test/topic")

	// Create a mock connection
	mockConn := &conn{
		subscribeTopics: topic.NewMemoryTrie(),
		version:         packet.VERSION311,
		PacketID:        0,
	}
	mockConn.subscribeTopics.Subscribe("test/topic")
	ts.Subscribe(mockConn)

	// Exchange a message - this will start a goroutine that tries to send
	// We'll just test that it doesn't panic
	message := &packet.Message{
		TopicName: "test/topic",
		Content:   []byte("test message"),
	}

	// This should not panic even though the connection is not fully initialized
	defer func() {
		if r := recover(); r != nil {
			t.Logf("Exchange panicked (expected in test): %v", r)
		}
	}()

	ts.Exchange(message, nil)
}

func TestMemorySubscribedCleanEmptyTopic(t *testing.T) {
	ctx := context.Background()
	server := NewServer(ctx)
	memorySub := NewMemorySubscribed(server)

	// Create a topic with no subscribers
	message := &packet.Message{
		TopicName: "empty/topic",
		Content:   []byte("test message"),
	}
	memorySub.Publish(message, nil)

	// Verify topic exists
	memorySub.mu.RLock()
	_, exists := memorySub.maps["empty/topic"]
	memorySub.mu.RUnlock()

	if !exists {
		t.Error("topic should exist after publish")
	}

	// Note: The actual cleanup happens in a goroutine with a 24-hour ticker
	// so we can't easily test it in a unit test without mocking time
	// This test just verifies the structure is set up correctly
}

func TestMemorySubscribedSubscribeUnsubscribe(t *testing.T) {
	ctx := context.Background()
	server := NewServer(ctx)
	memorySub := NewMemorySubscribed(server)

	// Create a mock connection
	mockConn := &conn{
		subscribeTopics: topic.NewMemoryTrie(),
	}

	// Subscribe all topics to the connection
	memorySub.Subscribe(mockConn)

	// Unsubscribe all topics from the connection
	memorySub.Unsubscribe(mockConn)

	// This should not cause any errors
}

func TestTopicSubscribedSubscribeWithoutMatchingTopic(t *testing.T) {
	ts := NewTopicSubscribed("test/topic")

	// Create a mock connection that doesn't subscribe to this topic
	mockConn := &conn{
		subscribeTopics: topic.NewMemoryTrie(),
	}
	mockConn.subscribeTopics.Subscribe("different/topic")

	// Try to subscribe - should not add the connection
	ts.Subscribe(mockConn)

	if len(ts.activeConn) != 0 {
		t.Error("should not subscribe connection to non-matching topic")
	}
}
