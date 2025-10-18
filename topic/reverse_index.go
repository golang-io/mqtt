package topic

import (
	"strings"
	"sync"
)

// ReverseIndex 反向索引：从topic到订阅filter的映射
type ReverseIndex struct {
	// topic → 匹配的订阅filter列表
	// 例如："sensor/temp" → ["sensor/+", "sensor/#", "#"]
	index map[string][]string
	mu    sync.RWMutex

	// 订阅filter → 订阅者数量（用于优化）
	// 当订阅者数量为0时，可以从index中移除该filter
	filterCount map[string]int
}

func NewReverseIndex() *ReverseIndex {
	return &ReverseIndex{
		index:       make(map[string][]string),
		filterCount: make(map[string]int),
	}
}

// AddFilter 添加订阅filter
func (r *ReverseIndex) AddFilter(filter string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.filterCount[filter]++

	// 清除所有缓存（因为新增了订阅，需要重新匹配）
	// 优化：只清除受影响的topic缓存
	r.index = make(map[string][]string)
}

// RemoveFilter 移除订阅filter
func (r *ReverseIndex) RemoveFilter(filter string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if count, ok := r.filterCount[filter]; ok {
		count--
		if count <= 0 {
			delete(r.filterCount, filter)
			// 清除包含该filter的缓存
			for topic, filters := range r.index {
				newFilters := make([]string, 0, len(filters))
				for _, f := range filters {
					if f != filter {
						newFilters = append(newFilters, f)
					}
				}
				if len(newFilters) == 0 {
					delete(r.index, topic)
				} else {
					r.index[topic] = newFilters
				}
			}
		} else {
			r.filterCount[filter] = count
		}
	}
}

// GetMatches 获取匹配的订阅filter列表（带缓存）
func (r *ReverseIndex) GetMatches(topic string) ([]string, bool) {
	r.mu.RLock()
	filters, cached := r.index[topic]
	r.mu.RUnlock()

	return filters, cached
}

// SetMatches 设置topic的匹配列表（缓存结果）
func (r *ReverseIndex) SetMatches(topic string, filters []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.index[topic] = filters
}

// MatchFilters 找到所有匹配topic的订阅filter
// 这个方法需要遍历所有filter，应该只在缓存未命中时调用
func (r *ReverseIndex) MatchFilters(topic string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var matches []string
	for filter := range r.filterCount {
		if r.filterCount[filter] > 0 && matchTopic(filter, topic) {
			matches = append(matches, filter)
		}
	}
	return matches
}

// matchTopic 检查订阅filter是否匹配topic
// filter: 订阅的topic filter，可能包含 + 和 # 通配符
// topic: 发布的精确topic
func matchTopic(filter, topic string) bool {
	filterParts := strings.Split(filter, "/")
	topicParts := strings.Split(topic, "/")

	fi, ti := 0, 0
	for fi < len(filterParts) && ti < len(topicParts) {
		if filterParts[fi] == "#" {
			// # 匹配所有剩余层级
			return true
		}
		if filterParts[fi] == "+" {
			// + 匹配单层
			fi++
			ti++
			continue
		}
		if filterParts[fi] == topicParts[ti] {
			// 精确匹配
			fi++
			ti++
			continue
		}
		// 不匹配
		return false
	}

	// 检查是否都到达末尾
	if fi == len(filterParts) && ti == len(topicParts) {
		return true
	}

	// filter还有剩余，检查是否是 #
	if fi == len(filterParts)-1 && filterParts[fi] == "#" {
		return true
	}

	return false
}

// GetFilterCount 获取订阅filter的数量
func (r *ReverseIndex) GetFilterCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.filterCount)
}

// GetCacheSize 获取缓存的topic数量
func (r *ReverseIndex) GetCacheSize() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.index)
}

// Clear 清空索引（用于测试或重置）
func (r *ReverseIndex) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.index = make(map[string][]string)
	r.filterCount = make(map[string]int)
}
