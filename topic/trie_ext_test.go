package topic

import (
	"testing"
)

func TestGetAllFilters(t *testing.T) {
	trie := NewMemoryTrie()

	// 添加一些订阅
	filters := []string{
		"sensor/temp",
		"sensor/+",
		"sensor/#",
		"device/+/status",
		"#",
	}

	for _, filter := range filters {
		if err := trie.Subscribe(filter); err != nil {
			t.Fatalf("Failed to subscribe %s: %v", filter, err)
		}
	}

	// 获取所有filter
	allFilters := trie.GetAllFilters()

	// 验证数量
	if len(allFilters) != len(filters) {
		t.Errorf("Expected %d filters, got %d", len(filters), len(allFilters))
	}

	// 验证内容（转换为map便于查找）
	filterMap := make(map[string]bool)
	for _, f := range allFilters {
		filterMap[f] = true
	}

	for _, expected := range filters {
		if !filterMap[expected] {
			t.Errorf("Expected filter %s not found in results", expected)
		}
	}
}

func TestMatchingFilters(t *testing.T) {
	trie := NewMemoryTrie()

	// 添加订阅
	filters := []string{
		"sensor/temp",
		"sensor/+",
		"sensor/#",
		"device/+/status",
	}

	for _, filter := range filters {
		trie.Subscribe(filter)
	}

	tests := []struct {
		topic    string
		expected []string
	}{
		{
			topic:    "sensor/temp",
			expected: []string{"sensor/temp", "sensor/+", "sensor/#"},
		},
		{
			topic:    "sensor/humidity",
			expected: []string{"sensor/+", "sensor/#"},
		},
		{
			topic:    "sensor/temp/room1",
			expected: []string{"sensor/#"},
		},
		{
			topic:    "device/001/status",
			expected: []string{"device/+/status"},
		},
		{
			topic:    "other/topic",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		matched := trie.MatchingFilters(tt.topic)

		if len(matched) != len(tt.expected) {
			t.Errorf("Topic %s: expected %d matches, got %d: %v",
				tt.topic, len(tt.expected), len(matched), matched)
			continue
		}

		// 转换为map便于比较
		matchedMap := make(map[string]bool)
		for _, m := range matched {
			matchedMap[m] = true
		}

		for _, exp := range tt.expected {
			if !matchedMap[exp] {
				t.Errorf("Topic %s: expected filter %s not found in matches",
					tt.topic, exp)
			}
		}
	}
}

func TestHasFilter(t *testing.T) {
	trie := NewMemoryTrie()

	trie.Subscribe("sensor/+")
	trie.Subscribe("device/#")

	if !trie.HasFilter("sensor/+") {
		t.Error("Expected sensor/+ to be subscribed")
	}

	if trie.HasFilter("sensor/temp") {
		t.Error("Expected sensor/temp NOT to be subscribed")
	}
}

func TestGetFilterCount(t *testing.T) {
	trie := NewMemoryTrie()

	if trie.GetFilterCount() != 0 {
		t.Error("Expected 0 filters initially")
	}

	trie.Subscribe("sensor/+")
	trie.Subscribe("device/#")

	if trie.GetFilterCount() != 2 {
		t.Errorf("Expected 2 filters, got %d", trie.GetFilterCount())
	}
}

func BenchmarkGetAllFilters(b *testing.B) {
	trie := NewMemoryTrie()

	// 添加100个订阅
	for i := 0; i < 100; i++ {
		trie.Subscribe("sensor/+")
		trie.Subscribe("device/#")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trie.GetAllFilters()
	}
}

func BenchmarkMatchingFilters(b *testing.B) {
	trie := NewMemoryTrie()

	// 添加100个订阅
	for i := 0; i < 100; i++ {
		trie.Subscribe("sensor/+")
		trie.Subscribe("device/#")
		trie.Subscribe("+/temp")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		trie.MatchingFilters("sensor/temp/room1/data")
	}
}
