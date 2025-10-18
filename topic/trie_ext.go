package topic

import "strings"

/*
================================================================================
Trie树扩展方法
================================================================================

为Trie树添加获取所有订阅filter的方法，用于反向索引集成。
*/

// GetAllFilters 获取所有订阅的topic filter列表
// 用于反向索引：需要知道所有的订阅filter才能建立反向映射
func (m *MemoryTrie) GetAllFilters() []string {
	return m.root.getAllPaths()
}

// getAllPaths 递归获取所有路径（即所有订阅的filter）
func (n *node) getAllPaths() []string {
	var results []string
	n.collectPaths("", &results)
	return results
}

// collectPaths 递归收集所有完整路径
func (n *node) collectPaths(currentPath string, results *[]string) {
	n.m.RLock()
	defer n.m.RUnlock()

	// 如果当前节点是叶子节点（没有子节点），则当前路径是一个完整的订阅
	if len(n.next) == 0 && currentPath != "" {
		*results = append(*results, currentPath)
		return
	}

	// 遍历所有子节点
	for path, child := range n.next {
		var newPath string
		if currentPath == "" {
			newPath = path
		} else {
			newPath = currentPath + "/" + path
		}

		// 如果子节点没有后续节点，说明这是一个完整的订阅
		child.m.RLock()
		hasNext := len(child.next) > 0
		child.m.RUnlock()

		if !hasNext {
			*results = append(*results, newPath)
		} else {
			// 继续递归
			child.collectPaths(newPath, results)
		}
	}
}

// HasFilter 检查是否已订阅某个filter
func (m *MemoryTrie) HasFilter(filter string) bool {
	_, ok := m.Find(filter)
	return ok
}

// GetFilterCount 获取订阅的filter数量
func (m *MemoryTrie) GetFilterCount() int {
	return len(m.GetAllFilters())
}

// MatchingFilters 找到所有匹配给定topic的订阅filter
// 这个方法从订阅filter的角度来匹配，而不是从topic的角度
func (m *MemoryTrie) MatchingFilters(topic string) []string {
	filters := m.GetAllFilters()
	var matched []string

	for _, filter := range filters {
		if matchTopicFilter(filter, topic) {
			matched = append(matched, filter)
		}
	}

	return matched
}

// matchTopicFilter 检查订阅filter是否匹配topic
// 这是反向匹配：filter可能包含通配符，topic是精确的
func matchTopicFilter(filter, topic string) bool {
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

// Clear 清空所有订阅（用于测试）
func (m *MemoryTrie) Clear() {
	m.root = newNode("")
}
