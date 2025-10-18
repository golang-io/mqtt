package topic

import (
	"testing"
)

func TestMatchTopic(t *testing.T) {
	tests := []struct {
		filter   string
		topic    string
		expected bool
	}{
		// 精确匹配
		{"sensor/temp", "sensor/temp", true},
		{"sensor/temp", "sensor/humidity", false},

		// + 通配符（单层）
		{"sensor/+", "sensor/temp", true},
		{"sensor/+", "sensor/humidity", true},
		{"sensor/+", "sensor/temp/room1", false},
		{"sensor/+/room1", "sensor/temp/room1", true},
		{"sensor/+/room1", "sensor/temp/room2", false},

		// # 通配符（多层）
		{"sensor/#", "sensor/temp", true},
		{"sensor/#", "sensor/temp/room1", true},
		{"sensor/#", "sensor/temp/room1/data", true},
		{"sensor/#", "device/temp", false},
		{"#", "any/topic/here", true},
		{"sensor/temp/#", "sensor/temp", true},
		{"sensor/temp/#", "sensor/temp/room1", true},

		// 混合
		{"+/+", "sensor/temp", true},
		{"+/+", "sensor/temp/room1", false},
		{"+/temp/#", "sensor/temp/room1/data", true},
		{"+/temp/#", "device/humidity", false},

		// 边界情况
		{"", "", true},
		{"sensor", "sensor", true},
		{"sensor/", "sensor", false},
	}

	for _, tt := range tests {
		result := matchTopic(tt.filter, tt.topic)
		if result != tt.expected {
			t.Errorf("matchTopic(%q, %q) = %v, want %v",
				tt.filter, tt.topic, result, tt.expected)
		}
	}
}

func TestReverseIndex(t *testing.T) {
	ri := NewReverseIndex()

	// 添加订阅filter
	ri.AddFilter("sensor/+")
	ri.AddFilter("sensor/#")
	ri.AddFilter("device/+/status")
	ri.AddFilter("#")

	// 测试匹配
	matches := ri.MatchFilters("sensor/temp")
	expectedCount := 3 // sensor/+, sensor/#, #
	if len(matches) != expectedCount {
		t.Errorf("Expected %d matches, got %d: %v", expectedCount, len(matches), matches)
	}

	// 测试缓存
	ri.SetMatches("sensor/temp", matches)
	cached, ok := ri.GetMatches("sensor/temp")
	if !ok {
		t.Error("Expected cached result")
	}
	if len(cached) != expectedCount {
		t.Errorf("Expected %d cached matches, got %d", expectedCount, len(cached))
	}

	// 测试移除filter
	ri.RemoveFilter("sensor/+")
	matches2 := ri.MatchFilters("sensor/temp")
	expectedCount2 := 2 // sensor/#, #
	if len(matches2) != expectedCount2 {
		t.Errorf("Expected %d matches after removal, got %d: %v", expectedCount2, len(matches2), matches2)
	}
}

func BenchmarkMatchTopic(b *testing.B) {
	filter := "sensor/+/room/+/data/#"
	topic := "sensor/temp/room/living/data/current/value"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		matchTopic(filter, topic)
	}
}

func BenchmarkReverseIndex_MatchFilters(b *testing.B) {
	ri := NewReverseIndex()

	// 模拟1000个订阅
	for i := 0; i < 1000; i++ {
		ri.AddFilter("sensor/+")
		ri.AddFilter("device/#")
		ri.AddFilter("+/temp")
		ri.AddFilter("#")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ri.MatchFilters("sensor/temp/room1/data")
	}
}

func BenchmarkReverseIndex_GetMatches_Cached(b *testing.B) {
	ri := NewReverseIndex()

	matches := []string{"sensor/+", "sensor/#", "#"}
	ri.SetMatches("sensor/temp", matches)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ri.GetMatches("sensor/temp")
	}
}
