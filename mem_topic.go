package mqtt

import (
	"context"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/golang-io/mqtt/packet"
	"github.com/golang-io/mqtt/topic"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
)

// SubscriptionManager 订阅管理器接口
type SubscriptionManager interface {
	// Subscribe 客户端订阅
	Subscribe(c *conn)

	// Unsubscribe 客户端断开连接时取消所有订阅
	Unsubscribe(c *conn)

	// UnsubscribeTopics 取消订阅指定的topics（处理UNSUBSCRIBE报文时使用）
	UnsubscribeTopics(c *conn, topics []string)

	// Publish 发布消息
	// sourceConn: 消息来源的连接，nil 表示服务端自己发布的消息
	Publish(message *packet.Message, props *packet.PublishProperties, sourceConn *conn) error

	// Print 打印订阅信息（调试用）
	Print()
}

// 确保实现了接口
var _ SubscriptionManager = (*MemorySubscribed)(nil)

// Prometheus 指标定义
var (
	// 发布相关指标
	mqttPublishTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "mqtt_publish_total",
			Help: "Total number of MQTT messages published",
		},
		[]string{"topic"},
	)

	mqttPublishCacheHit = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "mqtt_publish_cache_hit_total",
			Help: "Total number of cache hits during publish operations",
		},
	)

	mqttPublishCacheMiss = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "mqtt_publish_cache_miss_total",
			Help: "Total number of cache misses during publish operations",
		},
	)

	mqttPublishLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "mqtt_publish_duration_seconds",
			Help:    "Time spent on publish operations",
			Buckets: prometheus.ExponentialBuckets(0.0001, 2, 15), // 0.1ms to 3.2s
		},
		[]string{"topic"},
	)

	// 订阅相关指标
	mqttSubscribeTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "mqtt_subscribe_total",
			Help: "Total number of MQTT subscribe operations",
		},
	)

	mqttUnsubscribeTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "mqtt_unsubscribe_total",
			Help: "Total number of MQTT unsubscribe operations",
		},
	)

	// 内存和状态指标
	mqttFilterCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "mqtt_filter_count",
			Help: "Current number of active subscription filters",
		},
	)

	mqttSubscriberCount = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "mqtt_subscriber_count",
			Help: "Current number of active subscribers",
		},
	)

	mqttCacheSize = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "mqtt_cache_size",
			Help: "Current size of the topic matching cache",
		},
	)

	mqttCacheHitRate = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "mqtt_cache_hit_rate",
			Help: "Current cache hit rate (0-1)",
		},
	)
)

// init 函数注册 Prometheus 指标
func init() {
	prometheus.MustRegister(mqttPublishTotal)
	prometheus.MustRegister(mqttPublishCacheHit)
	prometheus.MustRegister(mqttPublishCacheMiss)
	prometheus.MustRegister(mqttPublishLatency)
	prometheus.MustRegister(mqttSubscribeTotal)
	prometheus.MustRegister(mqttUnsubscribeTotal)
	prometheus.MustRegister(mqttFilterCount)
	prometheus.MustRegister(mqttSubscriberCount)
	prometheus.MustRegister(mqttCacheSize)
	prometheus.MustRegister(mqttCacheHitRate)
}

// MemorySubscribed 高性能订阅管理器
type MemorySubscribed struct {
	// ===== 核心数据结构 =====

	// filter → 订阅者列表
	filters   map[string]*FilterSubscription
	filtersMu sync.RWMutex

	// 反向索引：topic → 匹配的filter列表
	reverseIndex *topic.ReverseIndex

	// ===== 配置 =====
	server *Server
	config SubscribedConfig
}

// SubscribedConfig 配置选项
type SubscribedConfig struct {
	// 是否启用缓存（默认true）
	EnableCache bool

	// 缓存大小限制（默认10000）
	MaxCacheSize int

	// 统计信息打印间隔（默认1分钟）
	StatsInterval time.Duration

	// 是否启用性能监控（默认true）
	EnableMonitoring bool
}

// FilterSubscription 单个filter的订阅信息
type FilterSubscription struct {
	Filter string // 订阅filter，如 "sensor/+"

	// 订阅者列表（使用map实现，支持O(1)添加/删除）
	subscribers   map[*conn]struct{}
	subscribersMu sync.RWMutex

	// 统计信息
	subscribeCount   atomic.Int64 // 订阅次数
	unsubscribeCount atomic.Int64 // 取消订阅次数
	// messageCount     atomic.Int64 // 消息推送次数
}

// NewMemorySubscribed 创建订阅管理器
func NewMemorySubscribed(s *Server, cfg SubscribedConfig) *MemorySubscribed {
	// 设置默认配置
	if cfg.MaxCacheSize == 0 {
		cfg.MaxCacheSize = 10000
	}
	if cfg.StatsInterval == 0 {
		cfg.StatsInterval = 1 * time.Minute
	}
	cfg.EnableCache = true // 强制启用缓存
	cfg.EnableMonitoring = true

	m := &MemorySubscribed{
		filters:      make(map[string]*FilterSubscription),
		reverseIndex: topic.NewReverseIndex(),
		server:       s,
		config:       cfg,
	}

	// 启动后台任务
	if cfg.EnableMonitoring {
		go m.monitorStats()
	}

	return m
}

// Subscribe 客户端订阅（优化版）
func (m *MemorySubscribed) Subscribe(c *conn) {
	startTime := time.Now()

	// 获取该客户端的所有订阅filter
	filters := c.subscribeTopics.GetAllFilters()

	for _, filter := range filters {
		m.subscribeFilter(c, filter)
	}

	mqttSubscribeTotal.Add(float64(len(filters)))

	elapsed := time.Since(startTime)
	if elapsed > 10*time.Millisecond {
		log.Printf("[MemSub] Subscribe slow: clientId=%s, filters=%d, elapsed=%v",
			c.ID, len(filters), elapsed)
	}
}

// subscribeFilter 订阅单个filter
func (m *MemorySubscribed) subscribeFilter(c *conn, filter string) {
	// 1. 获取或创建FilterSubscription
	m.filtersMu.Lock()
	fs, exists := m.filters[filter]
	if !exists {
		fs = &FilterSubscription{
			Filter:      filter,
			subscribers: make(map[*conn]struct{}),
		}
		m.filters[filter] = fs
		mqttFilterCount.Inc()

		// 通知反向索引
		m.reverseIndex.AddFilter(filter)
	}
	m.filtersMu.Unlock()

	// 2. 添加订阅者
	fs.subscribersMu.Lock()
	if _, exists := fs.subscribers[c]; !exists {
		fs.subscribers[c] = struct{}{}
		fs.subscribeCount.Add(1)
		mqttSubscriberCount.Inc()
	}
	fs.subscribersMu.Unlock()
}

// Unsubscribe 客户端断开连接时取消所有订阅
func (m *MemorySubscribed) Unsubscribe(c *conn) {
	filters := c.subscribeTopics.GetAllFilters()

	for _, filter := range filters {
		m.unsubscribeFilter(c, filter)
	}

	mqttUnsubscribeTotal.Add(float64(len(filters)))
}

// UnsubscribeTopics 取消订阅指定的topics（处理UNSUBSCRIBE报文时使用）
func (m *MemorySubscribed) UnsubscribeTopics(c *conn, topics []string) {
	// 只取消订阅指定的topics
	for _, topic := range topics {
		m.unsubscribeFilter(c, topic)
	}

	mqttUnsubscribeTotal.Add(float64(len(topics)))

	log.Printf("[MemSub] Client unsubscribed topics: clientId=%s, topics=%v", c.ID, topics)
}

// unsubscribeFilter 取消订阅单个filter
func (m *MemorySubscribed) unsubscribeFilter(c *conn, filter string) {
	m.filtersMu.RLock()
	fs, exists := m.filters[filter]
	m.filtersMu.RUnlock()

	if !exists {
		return
	}

	// 从订阅者列表中移除
	fs.subscribersMu.Lock()
	if _, exists := fs.subscribers[c]; exists {
		delete(fs.subscribers, c)
		fs.unsubscribeCount.Add(1)
		mqttSubscriberCount.Dec()
	}
	isEmpty := len(fs.subscribers) == 0
	fs.subscribersMu.Unlock()

	// 如果没有订阅者了，删除整个FilterSubscription
	if isEmpty {
		m.filtersMu.Lock()
		// 再次检查（双重检查锁）
		fs.subscribersMu.RLock()
		if len(fs.subscribers) == 0 {
			delete(m.filters, filter)
			mqttFilterCount.Dec()
			m.reverseIndex.RemoveFilter(filter)
		}
		fs.subscribersMu.RUnlock()
		m.filtersMu.Unlock()
	}
}

// Publish 发布消息（核心优化方法）
func (m *MemorySubscribed) Publish(message *packet.Message, props *packet.PublishProperties, sourceConn *conn) error {
	startTime := time.Now()
	defer func() {
		elapsed := time.Since(startTime)
		mqttPublishLatency.WithLabelValues(message.TopicName).Observe(elapsed.Seconds())
	}()

	topic := message.TopicName
	mqttPublishTotal.WithLabelValues(topic).Inc()

	// 步骤1: 查询反向索引（带缓存）
	matchedFilters, cached := m.reverseIndex.GetMatches(topic)
	if cached {
		mqttPublishCacheHit.Inc()
	} else {
		mqttPublishCacheMiss.Inc()

		// 缓存未命中，需要匹配所有filter
		matchedFilters = m.reverseIndex.MatchFilters(topic)

		// 缓存结果
		m.reverseIndex.SetMatches(topic, matchedFilters)
		mqttCacheSize.Set(float64(m.reverseIndex.GetCacheSize()))

		if len(matchedFilters) > 0 {
			log.Printf("[MemSub] Cache miss: topic=%s, matched=%d filters", topic, len(matchedFilters))
		}
	}
	if len(matchedFilters) == 0 && !strings.HasPrefix(sourceConn.ID, "MQTT-FEDERATE#") {
		log.Printf("[MemSub] No subscribers found for topic: %s, forward to federated nodes: node_id=%d", topic, len(m.server.Federated))
		for _, client := range m.server.Federated {
			if err := client.SubmitMessage(message); err != nil {
				log.Printf("[MemSub] Forward to federated node failed: node_id=%s, error=%v", client.options.ClientID, err)
			}
		}
		return nil
	}

	// 步骤2: 收集所有订阅者（去重）
	subscribers := m.collectSubscribers(matchedFilters)
	if len(subscribers) == 0 {
		log.Printf("[MemSub] No active subscribers for topic: %s (filters matched but no active connections)", topic)
		return nil
	}

	// 步骤3: 批量推送消息
	return m.publishBatch(subscribers, message, props)
}

// collectSubscribers 收集订阅者（去重）
func (m *MemorySubscribed) collectSubscribers(filters []string) []*conn {
	// 使用map去重
	subscribersMap := make(map[*conn]struct{})

	m.filtersMu.RLock()
	for _, filter := range filters {
		if fs, exists := m.filters[filter]; exists {
			fs.subscribersMu.RLock()
			for sub := range fs.subscribers {
				subscribersMap[sub] = struct{}{}
			}
			fs.subscribersMu.RUnlock()
		}
	}
	m.filtersMu.RUnlock()

	// 转换为slice
	subscribers := make([]*conn, 0, len(subscribersMap))
	for sub := range subscribersMap {
		subscribers = append(subscribers, sub)
	}

	return subscribers
}

// publishBatch 批量推送消息给订阅者
func (m *MemorySubscribed) publishBatch(subscribers []*conn, message *packet.Message, props *packet.PublishProperties) error {
	if len(subscribers) == 0 {
		return nil
	}

	group, _ := errgroup.WithContext(context.Background())
	group.SetLimit(200) // 限制并发数

	for _, c := range subscribers {
		conn := c // 捕获变量
		group.Go(func() error {
			return m.sendToConn(conn, message, props)
		})
	}

	return group.Wait()
}

// sendToConn 发送消息给单个连接
func (m *MemorySubscribed) sendToConn(c *conn, message *packet.Message, props *packet.PublishProperties) error {
	response := &response{conn: c}
	pub := &packet.PUBLISH{
		FixedHeader: &packet.FixedHeader{
			Version: c.version,
			Kind:    PUBLISH,
			QoS:     1,
			Retain:  0,
		},
		Message: message,
		Props:   props,
	}

	if pub.QoS > 0 {
		pub.PacketID = c.PacketID + 1
		c.PacketID = pub.PacketID
	}

	return response.OnSend(pub)
}

// monitorStats 监控统计信息
func (m *MemorySubscribed) monitorStats() {
	ticker := time.NewTicker(m.config.StatsInterval)
	defer ticker.Stop()

	for range ticker.C {
		m.printStats()
	}
}

// printStats 打印统计信息
func (m *MemorySubscribed) printStats() {
	// 注意：Prometheus 指标通过 /metrics 端点暴露，这里只打印基本统计信息
	log.Printf("[subscribe Stats] Prometheus metrics are available at /metrics endpoint")
}

// GetStats 获取统计信息（供外部查询）
// 注意：Prometheus 指标通过 /metrics 端点暴露，这里返回空map
func (m *MemorySubscribed) GetStats() map[string]float64 {
	stats := make(map[string]float64)

	// 注意：Prometheus 指标无法直接获取值，需要通过 /metrics 端点访问
	// 这里返回空map，建议使用 Prometheus 客户端库来查询指标
	log.Printf("[MemSub] Use /metrics endpoint to access Prometheus metrics")

	return stats
}

// Print 打印当前订阅信息（调试用）
func (m *MemorySubscribed) Print() {
	m.filtersMu.RLock()
	defer m.filtersMu.RUnlock()

	log.Printf("[MemSub] Total filters: %d", len(m.filters))
	for filter, fs := range m.filters {
		fs.subscribersMu.RLock()
		count := len(fs.subscribers)
		fs.subscribersMu.RUnlock()
		log.Printf("  Filter: %s, Subscribers: %d", filter, count)
	}
}

// forwardToSingleBridgeClient 转发消息到单个桥接客户端
// 参数:
//   - bridgeConn: 桥接客户端连接
//   - message: 要转发的消息
//   - props: 消息属性
//   - remoteNode: 远程节点名称（用于日志）
//
// 返回:
//   - error: 转发失败的错误
func (m *MemorySubscribed) forwardToSingleBridgeClient(
	bridgeConn *conn,
	message *packet.Message,
	props *packet.PublishProperties,
	remoteNode string,
) error {
	// 检查连接是否有效
	if bridgeConn == nil {
		return nil
	}

	// 使用 sendToConn 方法发送消息
	return m.sendToConn(bridgeConn, message, props)
}
