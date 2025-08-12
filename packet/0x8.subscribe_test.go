package packet

import (
	"bytes"
	"testing"
)

// TestSUBSCRIBE_Kind 测试SUBSCRIBE报文的类型标识符
// 参考MQTT v3.1.1章节 3.8 SUBSCRIBE - Subscribe to topics
// 参考MQTT v5.0章节 3.8 SUBSCRIBE - Subscribe to topics
func TestSUBSCRIBE_Kind(t *testing.T) {
	subscribe := &SUBSCRIBE{FixedHeader: &FixedHeader{Kind: 0x08}}
	if subscribe.Kind() != 0x08 {
		t.Errorf("SUBSCRIBE.Kind() = %d, want 0x08", subscribe.Kind())
	}
}

// TestSUBSCRIBE_Pack 测试SUBSCRIBE报文的序列化
// 参考MQTT v3.1.1章节 3.8.2 SUBSCRIBE Variable Header
// 参考MQTT v5.0章节 3.8.2 SUBSCRIBE Variable Header
func TestSUBSCRIBE_Pack(t *testing.T) {
	testCases := []struct {
		name      string
		subscribe *SUBSCRIBE
		version   byte
		expected  []byte
	}{
		{
			name: "V311_BasicSubscribe",
			subscribe: &SUBSCRIBE{
				FixedHeader: &FixedHeader{
					Version:         VERSION311,
					Kind:            0x08,
					Dup:             0,
					QoS:             1, // SUBSCRIBE的QoS必须为1
					Retain:          0,
					RemainingLength: 0, // 将在Pack时计算
				},
				PacketID: 12345,
				Subscriptions: []Subscription{
					{
						TopicFilter: "test/topic",
						MaximumQoS:  1,
					},
				},
			},
			version: VERSION311,
			expected: []byte{
				0x82, 0x0F, // 固定报头: SUBSCRIBE, 标志位010, 剩余长度15
				0x30, 0x39, // 报文标识符: 12345
				0x00, 0x0B, 't', 'e', 's', 't', '/', 't', 'o', 'p', 'i', 'c', // 主题过滤器
				0x01, // 订阅选项: QoS 1
			},
		},
		{
			name: "V500_BasicSubscribe",
			subscribe: &SUBSCRIBE{
				FixedHeader: &FixedHeader{
					Version:         VERSION500,
					Kind:            0x08,
					Dup:             0,
					QoS:             1, // SUBSCRIBE的QoS必须为1
					Retain:          0,
					RemainingLength: 0, // 将在Pack时计算
				},
				PacketID: 12345,
				Props:    &SubscribeProperties{},
				Subscriptions: []Subscription{
					{
						TopicFilter: "test/topic",
						MaximumQoS:  1,
					},
				},
			},
			version: VERSION500,
		},
		{
			name: "V500_SubscribeWithProperties",
			subscribe: &SUBSCRIBE{
				FixedHeader: &FixedHeader{
					Version:         VERSION500,
					Kind:            0x08,
					Dup:             0,
					QoS:             1,
					Retain:          0,
					RemainingLength: 0,
				},
				PacketID: 12345,
				Props: &SubscribeProperties{
					SubscriptionIdentifier: 123,
					UserProperty: map[string][]string{
						"key1": {"value1"},
					},
				},
				Subscriptions: []Subscription{
					{
						TopicFilter: "test/topic",
						MaximumQoS:  1,
					},
				},
			},
			version: VERSION500,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tc.subscribe.Pack(&buf)
			if err != nil {
				t.Fatalf("Pack() failed: %v", err)
			}

			result := buf.Bytes()

			// 验证基本结构
			if len(result) < 6 {
				t.Errorf("result too short: %d bytes", len(result))
			}

			// 验证报文类型和标志位
			if result[0] != 0x82 {
				t.Errorf("packet type and flags = %02x, want 0x82", result[0])
			}

			// 验证报文标识符
			if result[2] != 0x30 || result[3] != 0x39 {
				t.Errorf("packet ID = %02x%02x, want 0x3039", result[2], result[3])
			}

			// 对于v5.0，验证扩展结构
			if tc.version == VERSION500 {
				// 验证属性部分存在
				if len(result) < 8 {
					t.Errorf("v5.0 result too short: %d bytes", len(result))
				}
			}

			// 验证订阅列表
			if len(tc.subscribe.Subscriptions) > 0 {
				// 检查主题过滤器是否被正确编码
				if !bytes.Contains(result, []byte("test/topic")) {
					t.Error("topic filter not found in packed result")
				}
			}
		})
	}
}

// TestSUBSCRIBE_Unpack 测试SUBSCRIBE报文的反序列化
func TestSUBSCRIBE_Unpack(t *testing.T) {
	testCases := []struct {
		name        string
		data        []byte
		payloadData []byte
		version     byte
		expected    *SUBSCRIBE
	}{
		{
			name: "V311_BasicSubscribe",
			data: []byte{
				0x82, 0x10, // 固定报头: SUBSCRIBE, 标志位010, 剩余长度16
				0x30, 0x39, // 报文标识符: 12345
				0x00, 0x0B, 't', 'e', 's', 't', '/', 't', 'o', 'p', 'i', 'c', // 主题过滤器
				0x01, // 订阅选项: QoS 1
			},
			// 只包含可变报头和载荷的测试数据
			payloadData: []byte{
				0x30, 0x39, // 报文标识符: 12345
				0x00, 0x0B, // 主题过滤器长度: 11
				116, 101, 115, 116, 47, 116, 111, 112, 105, 99, // "test/topic" 的ASCII码
				0x01, // 订阅选项: QoS 1
			},
			version: VERSION311,
			expected: &SUBSCRIBE{
				FixedHeader: &FixedHeader{
					Version:         VERSION311,
					Kind:            0x08,
					Dup:             0,
					QoS:             1,
					Retain:          0,
					RemainingLength: 16,
				},
				PacketID: 12345,
				Subscriptions: []Subscription{
					{
						TopicFilter: "test/topic",
						MaximumQoS:  1,
					},
				},
			},
		},
		{
			name: "V500_BasicSubscribe",
			data: []byte{
				0x82, 0x12, // 固定报头: SUBSCRIBE, 标志位010, 剩余长度18
				0x30, 0x39, // 报文标识符: 12345
				0x00,                                                         // 属性长度: 0
				0x00, 0x0B, 't', 'e', 's', 't', '/', 't', 'o', 'p', 'i', 'c', // 主题过滤器
				0x01, // 订阅选项: QoS 1
			},
			// 只包含可变报头和载荷的测试数据
			payloadData: []byte{
				0x30, 0x39, // 报文标识符: 12345
				0x00,       // 属性长度: 0 (表示没有属性)
				0x00, 0x0B, // 主题过滤器长度: 11
				116, 101, 115, 116, 47, 116, 111, 112, 105, 99, // "test/topic" 的ASCII码
				0x01, // 订阅选项: QoS 1
			},
			version: VERSION500,
			expected: &SUBSCRIBE{
				FixedHeader: &FixedHeader{
					Version:         VERSION500,
					Kind:            0x08,
					Dup:             0,
					QoS:             1,
					Retain:          0,
					RemainingLength: 18,
				},
				PacketID: 12345,
				Props:    &SubscribeProperties{},
				Subscriptions: []Subscription{
					{
						TopicFilter: "test/topic",
						MaximumQoS:  1,
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 先解析固定报头
			fixedHeader := &FixedHeader{}
			buf := bytes.NewBuffer(tc.data)
			if err := fixedHeader.Unpack(buf); err != nil {
				t.Fatalf("FixedHeader.Unpack() failed: %v", err)
			}

			subscribe := &SUBSCRIBE{
				FixedHeader: fixedHeader,
			}

			// 设置版本信息，这样Unpack方法知道如何处理属性
			subscribe.Version = tc.version

			// 使用payloadData来测试Unpack，这样只包含可变报头和载荷部分
			payloadBuf := bytes.NewBuffer(tc.payloadData)
			err := subscribe.Unpack(payloadBuf)
			if err != nil {
				t.Fatalf("Unpack() failed: %v", err)
			}

			if subscribe.PacketID != tc.expected.PacketID {
				t.Errorf("PacketID = %d, want %d", subscribe.PacketID, tc.expected.PacketID)
			}

			if len(subscribe.Subscriptions) != len(tc.expected.Subscriptions) {
				t.Errorf("Subscriptions count = %d, want %d", len(subscribe.Subscriptions), len(tc.expected.Subscriptions))
			}

			if len(subscribe.Subscriptions) > 0 {
				if subscribe.Subscriptions[0].TopicFilter != tc.expected.Subscriptions[0].TopicFilter {
					t.Errorf("TopicFilter = %s, want %s", subscribe.Subscriptions[0].TopicFilter, tc.expected.Subscriptions[0].TopicFilter)
				}
				if subscribe.Subscriptions[0].MaximumQoS != tc.expected.Subscriptions[0].MaximumQoS {
					t.Errorf("MaximumQoS = %d, want %d", subscribe.Subscriptions[0].MaximumQoS, tc.expected.Subscriptions[0].MaximumQoS)
				}
			}
		})
	}
}

// TestSUBSCRIBE_ProtocolCompliance 测试SUBSCRIBE报文的协议合规性
func TestSUBSCRIBE_ProtocolCompliance(t *testing.T) {
	t.Run("V311_QoSMustBeOne", func(t *testing.T) {
		// v3.1.1中QoS必须为1
		subscribe := &SUBSCRIBE{
			FixedHeader: &FixedHeader{
				Version: VERSION311,
				Kind:    0x08,
				Dup:     0,
				QoS:     0, // 违反协议
				Retain:  0,
			},
			PacketID: 12345,
			Subscriptions: []Subscription{
				{
					TopicFilter: "test/topic",
					MaximumQoS:  1,
				},
			},
		}

		var buf bytes.Buffer
		err := subscribe.Pack(&buf)
		if err != nil {
			t.Fatalf("Pack() failed: %v", err)
		}

		// 验证QoS被正确设置
		result := buf.Bytes()
		if result[0] != 0x82 {
			t.Errorf("QoS not properly set: %02x", result[0])
		}
	})

	t.Run("V500_PropertiesSupport", func(t *testing.T) {
		// v5.0支持属性
		subscribe := &SUBSCRIBE{
			FixedHeader: &FixedHeader{
				Version: VERSION500,
				Kind:    0x08,
				Dup:     0,
				QoS:     1,
				Retain:  0,
			},
			PacketID: 12345,
			Props: &SubscribeProperties{
				SubscriptionIdentifier: 123,
			},
			Subscriptions: []Subscription{
				{
					TopicFilter: "test/topic",
					MaximumQoS:  1,
				},
			},
		}

		var buf bytes.Buffer
		err := subscribe.Pack(&buf)
		if err != nil {
			t.Fatalf("Pack() failed: %v", err)
		}

		result := buf.Bytes()
		if len(result) < 8 {
			t.Errorf("result too short: %d bytes", len(result))
		}
	})
}

// TestSUBSCRIBE_EdgeCases 测试SUBSCRIBE报文的边界情况
func TestSUBSCRIBE_EdgeCases(t *testing.T) {
	t.Run("EmptySubscriptions", func(t *testing.T) {
		subscribe := &SUBSCRIBE{
			FixedHeader: &FixedHeader{
				Version: VERSION311,
				Kind:    0x08,
				Dup:     0,
				QoS:     1,
				Retain:  0,
			},
			PacketID:      12345,
			Subscriptions: []Subscription{}, // 空订阅列表
		}

		var buf bytes.Buffer
		err := subscribe.Pack(&buf)
		if err != nil {
			t.Fatalf("Pack() failed: %v", err)
		}

		result := buf.Bytes()
		if len(result) < 4 {
			t.Errorf("result too short: %d bytes", len(result))
		}
	})

	t.Run("MultipleSubscriptions", func(t *testing.T) {
		subscribe := &SUBSCRIBE{
			FixedHeader: &FixedHeader{
				Version: VERSION311,
				Kind:    0x08,
				Dup:     0,
				QoS:     1,
				Retain:  0,
			},
			PacketID: 12345,
			Subscriptions: []Subscription{
				{
					TopicFilter: "test/topic1",
					MaximumQoS:  0,
				},
				{
					TopicFilter: "test/topic2",
					MaximumQoS:  1,
				},
				{
					TopicFilter: "test/topic3",
					MaximumQoS:  2,
				},
			},
		}

		var buf bytes.Buffer
		err := subscribe.Pack(&buf)
		if err != nil {
			t.Fatalf("Pack() failed: %v", err)
		}

		result := buf.Bytes()
		if len(result) < 20 {
			t.Errorf("result too short: %d bytes", len(result))
		}

		// 验证所有主题都被编码
		if !bytes.Contains(result, []byte("test/topic1")) {
			t.Error("topic1 not found in packed result")
		}
		if !bytes.Contains(result, []byte("test/topic2")) {
			t.Error("topic2 not found in packed result")
		}
		if !bytes.Contains(result, []byte("test/topic3")) {
			t.Error("topic3 not found in packed result")
		}
	})
}

// TestSubscription_String 测试订阅的字符串表示
func TestSubscription_String(t *testing.T) {
	sub := &Subscription{
		TopicFilter: "test/topic",
		MaximumQoS:  1,
	}

	result := sub.String()
	if result == "" {
		t.Error("String() should not be empty")
	}

	if !bytes.Contains([]byte(result), []byte("test/topic")) {
		t.Error("String() should contain topic filter")
	}
}

// TestSubscribeProperties_Pack 测试订阅属性的序列化
func TestSubscribeProperties_Pack(t *testing.T) {
	props := &SubscribeProperties{
		SubscriptionIdentifier: 123,
		UserProperty: map[string][]string{
			"key1": {"value1", "value2"},
			"key2": {"value3"},
		},
	}

	result, err := props.Pack()
	if err != nil {
		t.Fatalf("Pack() failed: %v", err)
	}

	if len(result) == 0 {
		t.Error("Pack() returned empty result")
	}

	// 验证包含订阅标识符
	if !bytes.Contains(result, []byte{0x0B}) {
		t.Error("subscription identifier not found in packed result")
	}
}

// TestSubscribeProperties_Unpack 测试订阅属性的反序列化
func TestSubscribeProperties_Unpack(t *testing.T) {
	// 先创建一个属性并序列化
	originalProps := &SubscribeProperties{
		SubscriptionIdentifier: 123,
		UserProperty: map[string][]string{
			"key1": {"value1"},
		},
	}

	packed, err := originalProps.Pack()
	if err != nil {
		t.Fatalf("Pack() failed: %v", err)
	}

	// 反序列化
	propsLen, err := encodeLength(len(packed))
	if err != nil {
		t.Fatalf("encodeLength() failed: %v", err)
	}

	buf := bytes.NewBuffer(append(propsLen, packed...))
	newProps := &SubscribeProperties{}
	err = newProps.Unpack(buf)
	if err != nil {
		t.Fatalf("Unpack() failed: %v", err)
	}

	if newProps.SubscriptionIdentifier != originalProps.SubscriptionIdentifier {
		t.Errorf("SubscriptionIdentifier = %d, want %d", newProps.SubscriptionIdentifier, originalProps.SubscriptionIdentifier)
	}

	if len(newProps.UserProperty) != len(originalProps.UserProperty) {
		t.Errorf("UserProperty count = %d, want %d", len(newProps.UserProperty), len(originalProps.UserProperty))
	}
}

// TestSUBSCRIBE_TopicFilters 测试各种主题过滤器
func TestSUBSCRIBE_TopicFilters(t *testing.T) {
	testCases := []struct {
		name        string
		topicFilter string
		maxQoS      uint8
		valid       bool
	}{
		{"SimpleTopic", "test/topic", 1, true},
		{"SingleLevelWildcard", "test/+/topic", 1, true},
		{"MultiLevelWildcard", "test/#", 1, true},
		{"EmptyTopic", "", 1, false},
		{"WildcardInMiddle", "test/+/+/topic", 1, true},
		{"MultipleWildcards", "test/+/+/#", 1, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			subscribe := &SUBSCRIBE{
				FixedHeader: &FixedHeader{
					Version: VERSION311,
					Kind:    0x08,
					Dup:     0,
					QoS:     1,
					Retain:  0,
				},
				PacketID: 12345,
				Subscriptions: []Subscription{
					{
						TopicFilter: tc.topicFilter,
						MaximumQoS:  tc.maxQoS,
					},
				},
			}

			var buf bytes.Buffer
			err := subscribe.Pack(&buf)

			if tc.valid {
				if err != nil {
					t.Errorf("Pack() failed for valid topic filter '%s': %v", tc.topicFilter, err)
				}
			} else {
				if err == nil {
					t.Errorf("Pack() should fail for invalid topic filter '%s'", tc.topicFilter)
				}
			}
		})
	}
}

// TestSUBSCRIBE_QoSLevels 测试QoS等级设置
func TestSUBSCRIBE_QoSLevels(t *testing.T) {
	testCases := []struct {
		name   string
		maxQoS uint8
		valid  bool
	}{
		{"QoS0", 0, true},
		{"QoS1", 1, true},
		{"QoS2", 2, true},
		{"QoS3", 3, false}, // 保留值，不允许使用
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			subscribe := &SUBSCRIBE{
				FixedHeader: &FixedHeader{
					Version: VERSION311,
					Kind:    0x08,
					Dup:     0,
					QoS:     1,
					Retain:  0,
				},
				PacketID: 12345,
				Subscriptions: []Subscription{
					{
						TopicFilter: "test/topic",
						MaximumQoS:  tc.maxQoS,
					},
				},
			}

			var buf bytes.Buffer
			err := subscribe.Pack(&buf)

			if tc.valid {
				if err != nil {
					t.Errorf("Pack() failed for valid QoS %d: %v", tc.maxQoS, err)
				}
			} else {
				if err == nil {
					t.Errorf("Pack() should fail for invalid QoS %d", tc.maxQoS)
				}
			}
		})
	}
}

// BenchmarkSUBSCRIBE_Pack 性能测试
func BenchmarkSUBSCRIBE_Pack(b *testing.B) {
	subscribe := &SUBSCRIBE{
		FixedHeader: &FixedHeader{
			Version: VERSION500,
			Kind:    0x08,
			Dup:     0,
			QoS:     1,
			Retain:  0,
		},
		PacketID: 12345,
		Props:    &SubscribeProperties{},
		Subscriptions: []Subscription{
			{
				TopicFilter: "test/topic",
				MaximumQoS:  1,
			},
		},
	}

	var buf bytes.Buffer
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		subscribe.Pack(&buf)
	}
}

// BenchmarkSUBSCRIBE_Unpack 性能测试
func BenchmarkSUBSCRIBE_Unpack(b *testing.B) {
	data := []byte{
		0x82, 0x11, // 固定报头
		0x30, 0x39, // 报文标识符
		0x00,                                                         // 属性长度
		0x00, 0x0B, 't', 'e', 's', 't', '/', 't', 'o', 'p', 'i', 'c', // 主题过滤器
		0x01, // 订阅选项
	}

	subscribe := &SUBSCRIBE{
		FixedHeader: &FixedHeader{
			Version:         VERSION500,
			Kind:            0x08,
			Dup:             0,
			QoS:             1,
			Retain:          0,
			RemainingLength: 17,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := bytes.NewBuffer(data)
		subscribe.Unpack(buf)
	}
}
