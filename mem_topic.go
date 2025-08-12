package mqtt

import (
	"context"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/golang-io/mqtt/packet"
	"golang.org/x/sync/errgroup"
)

type MemorySubscribed struct {
	maps map[string]*TopicSubscribed
	mu   sync.RWMutex
	s    *Server
}

func (m *MemorySubscribed) Print() {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, sub := range m.maps {
		log.Printf("[%s], conn=%d", sub.TopicName, len(sub.activeConn))
	}
}

func (m *MemorySubscribed) Subscribe(c *conn) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, ts := range m.maps {
		ts.Subscribe(c)
	}
}

func (m *MemorySubscribed) Unsubscribe(c *conn) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, ts := range m.maps {
		ts.Unsubscribe(c)
	}
}

// Publish 发布消息，如果是新topic需要额外处理存量connect订阅列表的构建
func (m *MemorySubscribed) Publish(message *packet.Message, props *packet.PublishProperties) error {
	m.mu.RLock()
	sub, ok := m.maps[message.TopicName]
	m.mu.RUnlock()
	if !ok {
		sub = NewTopicSubscribed(message.TopicName)
		m.s.mu.RLock()
		for c := range m.s.activeConn { // 如果是新的topic, 这里需要构建订阅列表!
			sub.Subscribe(c)
		}
		m.s.mu.RUnlock()
		m.mu.Lock()
		m.maps[message.TopicName] = sub
		m.mu.Unlock()
	}
	return sub.Exchange(message, props)
}

func NewMemorySubscribed(s *Server) *MemorySubscribed {
	m := &MemorySubscribed{
		maps: make(map[string]*TopicSubscribed),
		s:    s,
	}
	go m.CleanEmptyTopic()
	return m
}

func (m *MemorySubscribed) CleanEmptyTopic() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		m.mu.RLock()
		var empty []string
		for key, sub := range m.maps {
			if sub.Len() == 0 {
				empty = append(empty, key)
			}
		}
		m.mu.RUnlock()

		m.mu.Lock()
		for _, key := range empty {
			delete(m.maps, key)
		}
		m.mu.Unlock()
	}
}

// TopicSubscribed 用来存储当前topic有哪些客户端订阅了
type TopicSubscribed struct {
	TopicName  string
	activeConn map[*conn]struct{}
	// share      map[string]map[*conn]struct{} // 共享订阅, group: conn
	mux sync.RWMutex
}

func NewTopicSubscribed(topicName string) *TopicSubscribed {
	if strings.Contains(topicName, "$share/") {

	}
	return &TopicSubscribed{
		TopicName:  topicName,
		activeConn: make(map[*conn]struct{}),
	}
}

func (s *TopicSubscribed) Subscribe(c *conn) {
	if _, ok := c.subscribeTopics.Find(s.TopicName); !ok {
		return
	}
	s.mux.Lock()
	defer s.mux.Unlock()
	s.activeConn[c] = struct{}{}
}

func (s *TopicSubscribed) Len() int {
	s.mux.RLock()
	defer s.mux.RUnlock()
	return len(s.activeConn)
}

func (s *TopicSubscribed) Unsubscribe(c *conn) int {
	s.mux.Lock()
	defer s.mux.Unlock()
	delete(s.activeConn, c)
	return len(s.activeConn)
}

func (s *TopicSubscribed) Exchange(message *packet.Message, props *packet.PublishProperties) error {
	s.mux.RLock()
	defer s.mux.RUnlock()
	group, _ := errgroup.WithContext(context.Background())
	for c := range s.activeConn {
		response := &response{conn: c}
		group.Go(func() error {
			pub := &packet.PUBLISH{FixedHeader: &packet.FixedHeader{Version: c.version, Kind: PUBLISH, Dup: 0, QoS: 1, Retain: 0}, Message: message, Props: props}
			log.Printf("publish: topic=%s, qos=%d, retain=%d, message=%s, props=%v", message.TopicName, pub.QoS, pub.Retain, message.Content, props)
			if pub.QoS > 0 {
				pub.PacketID = c.PacketID + 1
				c.PacketID = pub.PacketID
			}
			return response.OnSend(pub)
		})
	}
	return group.Wait()
}
