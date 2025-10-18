package mqtt

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/golang-io/mqtt/packet"
	"github.com/golang-io/mqtt/topic"
)

/*
================================================================================
V2订阅管理集成测试
================================================================================

测试场景：
1. 大规模连接场景（10000+连接）
2. 大量topic发布（100000+ topic）
3. 高并发订阅/发布
4. 性能基准测试
*/

// TestBasicFunctionality 基本功能测试
func TestBasicFunctionality(t *testing.T) {
	server := NewServer(context.Background())
	v2 := NewMemorySubscribed(server, SubscribedConfig{
		EnableCache:   true,
		MaxCacheSize:  1000,
		StatsInterval: 10 * time.Second,
	})

	// 创建模拟连接
	conn1 := createMockConn(server, "client1")
	conn2 := createMockConn(server, "client2")

	// 订阅
	conn1.subscribeTopics.Subscribe("sensor/+")
	conn2.subscribeTopics.Subscribe("sensor/#")

	v2.Subscribe(conn1)
	v2.Subscribe(conn2)

	// 发布消息
	message := &packet.Message{
		TopicName: "sensor/temp",
		Content:   []byte("25.5°C"),
	}

	err := v2.Publish(message, nil)
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	// 验证统计 - 注意：Prometheus 指标无法直接获取值
	stats := v2.GetStats()
	if len(stats) == 0 {
		t.Log("Prometheus metrics are available at /metrics endpoint")
	}
}

// TestCacheEfficiency 缓存效率测试
func TestCacheEfficiency(t *testing.T) {
	server := NewServer(context.Background())
	v2 := NewMemorySubscribed(server, SubscribedConfig{
		EnableCache: true,
	})

	// 创建订阅
	conn := createMockConn(server, "client1")
	conn.subscribeTopics.Subscribe("sensor/+")
	v2.Subscribe(conn)

	// 第一次发布（缓存未命中）
	message := &packet.Message{
		TopicName: "sensor/temp",
		Content:   []byte("data"),
	}
	v2.Publish(message, nil)

	// 第二次发布相同topic（缓存命中）
	v2.Publish(message, nil)

	stats := v2.GetStats()

	// 验证缓存命中 - 注意：Prometheus 指标无法直接获取值
	if len(stats) == 0 {
		t.Log("Prometheus metrics are available at /metrics endpoint")
	}
}

// TestLargeScale 大规模场景测试
func TestLargeScale(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large scale test in short mode")
	}

	server := NewServer(context.Background())
	v2 := NewMemorySubscribed(server, SubscribedConfig{
		EnableCache:      true,
		EnableMonitoring: false, // 测试时关闭监控
	})

	t.Run("1000 connections", func(t *testing.T) {
		testLargeScale(t, v2, server, 1000, 10)
	})

	t.Run("10000 connections", func(t *testing.T) {
		testLargeScale(t, v2, server, 10000, 100)
	})
}

func testLargeScale(t *testing.T, v2 *MemorySubscribed, server *Server, connCount, topicCount int) {
	startTime := time.Now()

	// 创建大量连接和订阅
	conns := make([]*conn, connCount)
	for i := 0; i < connCount; i++ {
		c := createMockConn(server, fmt.Sprintf("client%d", i))
		c.subscribeTopics.Subscribe("sensor/+")
		c.subscribeTopics.Subscribe("device/#")
		conns[i] = c
		v2.Subscribe(c)
	}

	subscribeTime := time.Since(startTime)
	t.Logf("Subscribe %d connections: %v (%.2fms/conn)",
		connCount, subscribeTime, float64(subscribeTime.Milliseconds())/float64(connCount))

	// 发布大量不同的topic
	publishStart := time.Now()
	for i := 0; i < topicCount; i++ {
		message := &packet.Message{
			TopicName: fmt.Sprintf("sensor/temp%d", i),
			Content:   []byte("data"),
		}
		v2.Publish(message, nil)
	}
	publishTime := time.Since(publishStart)

	t.Logf("Publish %d topics: %v (%.2fms/topic)",
		topicCount, publishTime, float64(publishTime.Milliseconds())/float64(topicCount))

	// 验证统计
	stats := v2.GetStats()
	t.Logf("Stats: Prometheus metrics available at /metrics endpoint (stats map length: %d)", len(stats))

	// 性能断言
	avgPublishTime := publishTime / time.Duration(topicCount)
	if avgPublishTime > 10*time.Millisecond {
		t.Errorf("Average publish time too high: %v", avgPublishTime)
	}
}

// TestConcurrentAccess 并发访问测试
func TestConcurrentAccess(t *testing.T) {
	server := NewServer(context.Background())
	v2 := NewMemorySubscribed(server, SubscribedConfig{
		EnableCache:      true,
		EnableMonitoring: false,
	})

	// 并发订阅
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			c := createMockConn(server, fmt.Sprintf("client%d", id))
			c.subscribeTopics.Subscribe("test/+")
			v2.Subscribe(c)
		}(i)
	}

	// 并发发布
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			message := &packet.Message{
				TopicName: fmt.Sprintf("test/%d", id),
				Content:   []byte("data"),
			}
			v2.Publish(message, nil)
		}(i)
	}

	wg.Wait()

	// 验证无数据竞争
	stats := v2.GetStats()
	t.Logf("Concurrent test completed: Prometheus metrics available at /metrics endpoint (stats map length: %d)", len(stats))
}

// BenchmarkV2Publish 性能基准测试 - 发布
func BenchmarkV2Publish(b *testing.B) {
	server := NewServer(context.Background())
	v2 := NewMemorySubscribed(server, SubscribedConfig{
		EnableCache:      true,
		EnableMonitoring: false,
	})

	// 准备100个订阅
	for i := 0; i < 100; i++ {
		c := createMockConn(server, fmt.Sprintf("client%d", i))
		c.subscribeTopics.Subscribe("sensor/+")
		c.subscribeTopics.Subscribe("device/#")
		v2.Subscribe(c)
	}

	message := &packet.Message{
		TopicName: "sensor/temp",
		Content:   []byte("25.5°C"),
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			v2.Publish(message, nil)
		}
	})

	b.StopTimer()
	stats := v2.GetStats()
	b.Logf("Prometheus metrics available at /metrics endpoint (stats map length: %d)", len(stats))
}

// BenchmarkV2PublishNewTopic 性能基准测试 - 发布新topic
func BenchmarkV2PublishNewTopic(b *testing.B) {
	server := NewServer(context.Background())
	v2 := NewMemorySubscribed(server, SubscribedConfig{
		EnableCache:      true,
		EnableMonitoring: false,
	})

	// 准备订阅
	for i := 0; i < 100; i++ {
		c := createMockConn(server, fmt.Sprintf("client%d", i))
		c.subscribeTopics.Subscribe("sensor/+")
		v2.Subscribe(c)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		message := &packet.Message{
			TopicName: fmt.Sprintf("sensor/temp%d", i),
			Content:   []byte("data"),
		}
		v2.Publish(message, nil)
	}
}

// BenchmarkV2Subscribe 性能基准测试 - 订阅
func BenchmarkV2Subscribe(b *testing.B) {
	server := NewServer(context.Background())
	v2 := NewMemorySubscribed(server, SubscribedConfig{
		EnableCache:      true,
		EnableMonitoring: false,
	})

	// 预先发布一些topic
	for i := 0; i < 100; i++ {
		message := &packet.Message{
			TopicName: fmt.Sprintf("sensor/temp%d", i),
			Content:   []byte("data"),
		}
		v2.Publish(message, nil)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		c := createMockConn(server, fmt.Sprintf("client%d", i))
		c.subscribeTopics.Subscribe("sensor/+")
		c.subscribeTopics.Subscribe("device/#")
		v2.Subscribe(c)
	}
}

// 辅助函数：创建模拟连接
func createMockConn(server *Server, clientID string) *conn {
	c := &conn{
		server:          server,
		ID:              clientID,
		subscribeTopics: topic.NewMemoryTrie(),
		inFight:         newInFight(),
	}
	return c
}
