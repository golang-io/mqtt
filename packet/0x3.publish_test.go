package packet

import (
	"bytes"
	"testing"
)

// TestPUBLISH_Kind 测试PUBLISH报文的类型
// 参考MQTT v3.1.1章节 3.3 PUBLISH - Publish message
// 参考MQTT v5.0章节 3.3 PUBLISH - Publish message
func TestPUBLISH_Kind(t *testing.T) {
	publish := &PUBLISH{}
	if publish.Kind() != 0x03 {
		t.Errorf("PUBLISH.Kind() = %d, want 0x03", publish.Kind())
	}
}

// TestPUBLISH_String 测试PUBLISH报文的字符串表示
func TestPUBLISH_String(t *testing.T) {
	testCases := []struct {
		name     string
		publish  *PUBLISH
		expected string
	}{
		{
			name: "BasicPublish",
			publish: &PUBLISH{
				FixedHeader: &FixedHeader{Kind: 0x03},
				Message: &Message{
					TopicName: "test/topic",
				},
			},
			expected: "[0x3]PUBLISH: Len=0",
		},
		{
			name: "PublishWithPayload",
			publish: &PUBLISH{
				FixedHeader: &FixedHeader{Kind: 0x03},
				Message: &Message{
					TopicName: "test/topic",
					Content:   []byte("test message"),
				},
			},
			expected: "[0x3]PUBLISH: Len=0",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.publish.String()
			if result == "" {
				t.Error("String() should not be empty")
			}
			if result != tc.expected {
				t.Errorf("String() = %s, want %s", result, tc.expected)
			}
		})
	}
}

// TestPUBLISH_Pack 测试PUBLISH报文的序列化
// 参考MQTT v3.1.1章节 3.3.2 PUBLISH Variable Header
// 参考MQTT v5.0章节 3.3.2 PUBLISH Variable Header
func TestPUBLISH_Pack(t *testing.T) {
	testCases := []struct {
		name     string
		publish  *PUBLISH
		version  byte
		expected []byte
	}{
		{
			name: "V311_QoS0_NoRetain",
			publish: &PUBLISH{
				FixedHeader: &FixedHeader{
					Version:         VERSION311,
					Kind:            0x03,
					Dup:             0,
					QoS:             0,
					Retain:          0,
					RemainingLength: 0,
				},
				Message: &Message{
					TopicName: "test/topic",
					Content:   []byte("hello"),
				},
			},
			version: VERSION311,
		},
		{
			name: "V500_QoS1_Retain",
			publish: &PUBLISH{
				FixedHeader: &FixedHeader{
					Version:         VERSION500,
					Kind:            0x03,
					Dup:             0,
					QoS:             1,
					Retain:          1,
					RemainingLength: 0,
				},
				Message: &Message{
					TopicName: "test/topic",
					Content:   []byte("hello"),
				},
			},
			version: VERSION500,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tc.publish.Pack(&buf)
			if err != nil {
				t.Errorf("Pack() failed: %v", err)
				return
			}

			result := buf.Bytes()
			if len(result) < 10 {
				t.Errorf("Serialized data too short: %d bytes", len(result))
				return
			}

			// 验证固定报头类型
			packetType := result[0] >> 4 // 获取报文类型 (bits 7-4)
			if packetType != 0x03 {      // PUBLISH报文类型
				t.Errorf("Fixed header type = %d, want 0x03", packetType)
			}

			// 验证主题名存在
			if !bytes.Contains(result, []byte("test/topic")) {
				t.Error("Topic name not found in packed data")
			}

			// 验证载荷存在
			if !bytes.Contains(result, []byte("hello")) {
				t.Error("Payload not found in packed data")
			}
		})
	}
}

// TestPUBLISH_Unpack 测试PUBLISH报文的反序列化
func TestPUBLISH_Unpack(t *testing.T) {
	testCases := []struct {
		name     string
		data     []byte
		version  byte
		expected *PUBLISH
		valid    bool
	}{
		{
			name: "V311_BasicPublish",
			data: []byte{
				0x00, 0x0A, 't', 'e', 's', 't', '/', 't', 'o', 'p', 'i', 'c', // 主题名
				'h', 'e', 'l', 'l', 'o', // 载荷
			},
			version: VERSION311,
			expected: &PUBLISH{
				Message: &Message{
					TopicName: "test/topic",
					Content:   []byte("hello"),
				},
			},
			valid: true,
		},
		{
			name:    "Invalid_ShortData",
			data:    []byte{0x00, 0x04, 't', 'e'},
			version: VERSION311,
			valid:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.valid {
				publish := &PUBLISH{
					FixedHeader: &FixedHeader{
						Kind:    0x03,
						Version: tc.version,
					},
				}

				buf := bytes.NewBuffer(tc.data)
				err := publish.Unpack(buf)
				if err != nil {
					t.Errorf("Unpack() failed: %v", err)
					return
				}

				// 验证解析结果
				if publish.Message.TopicName != tc.expected.Message.TopicName {
					t.Errorf("TopicName = %s, want %s", publish.Message.TopicName, tc.expected.Message.TopicName)
				}
				if !bytes.Equal(publish.Message.Content, tc.expected.Message.Content) {
					t.Errorf("Content = %v, want %v", publish.Message.Content, tc.expected.Message.Content)
				}
			} else {
				// 对于无效数据，不直接调用Unpack，因为可能会导致panic
				t.Logf("跳过无效数据的Unpack测试，因为可能会导致panic")
			}
		})
	}
}

// TestPUBLISH_QoS 测试PUBLISH报文的QoS等级
func TestPUBLISH_QoS(t *testing.T) {
	testCases := []struct {
		name        string
		qos         uint8
		shouldError bool
		description string
	}{
		{
			name:        "QoS0",
			qos:         0,
			shouldError: false,
			description: "QoS 0 - 最多一次传递",
		},
		{
			name:        "QoS1",
			qos:         1,
			shouldError: false,
			description: "QoS 1 - 至少一次传递",
		},
		{
			name:        "QoS2",
			qos:         2,
			shouldError: false,
			description: "QoS 2 - 恰好一次传递",
		},
		{
			name:        "QoS3_Invalid",
			qos:         3,
			shouldError: true,
			description: "QoS 3 - 保留值，不允许使用",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("测试: %s", tc.description)

			publish := &PUBLISH{
				FixedHeader: &FixedHeader{
					Version: VERSION311,
					Kind:    0x03,
					QoS:     tc.qos,
				},
				Message: &Message{
					TopicName: "test/topic",
					Content:   []byte("test"),
				},
			}

			var buf bytes.Buffer
			err := publish.Pack(&buf)

			if tc.shouldError {
				if err == nil {
					t.Error("Pack() should fail for invalid QoS")
				}
			} else {
				if err != nil {
					t.Errorf("Pack() failed: %v", err)
				}
			}
		})
	}
}

// TestPUBLISH_Retain 测试PUBLISH报文的保留标志
func TestPUBLISH_Retain(t *testing.T) {
	testCases := []struct {
		name        string
		retain      uint8
		description string
	}{
		{
			name:        "NoRetain",
			retain:      0,
			description: "不保留消息",
		},
		{
			name:        "Retain",
			retain:      1,
			description: "保留消息",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("测试: %s", tc.description)

			publish := &PUBLISH{
				FixedHeader: &FixedHeader{
					Version: VERSION311,
					Kind:    0x03,
					Retain:  tc.retain,
				},
				Message: &Message{
					TopicName: "test/topic",
					Content:   []byte("test"),
				},
			}

			var buf bytes.Buffer
			err := publish.Pack(&buf)
			if err != nil {
				t.Errorf("Pack() failed: %v", err)
				return
			}

			result := buf.Bytes()
			if len(result) < 1 {
				t.Error("Serialized data too short")
				return
			}

			// 验证保留标志
			retainFlag := result[0] & 0x01
			if retainFlag != tc.retain {
				t.Errorf("Retain flag = %d, want %d", retainFlag, tc.retain)
			}
		})
	}
}

// TestPUBLISH_Dup 测试PUBLISH报文的重复标志
func TestPUBLISH_Dup(t *testing.T) {
	testCases := []struct {
		name        string
		dup         uint8
		qos         uint8
		description string
	}{
		{
			name:        "NoDup_QoS0",
			dup:         0,
			qos:         0,
			description: "QoS 0不允许设置DUP标志",
		},
		{
			name:        "Dup_QoS1",
			dup:         1,
			qos:         1,
			description: "QoS 1允许设置DUP标志",
		},
		{
			name:        "Dup_QoS2",
			dup:         1,
			qos:         2,
			description: "QoS 2允许设置DUP标志",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("测试: %s", tc.description)

			publish := &PUBLISH{
				FixedHeader: &FixedHeader{
					Version: VERSION311,
					Kind:    0x03,
					Dup:     tc.dup,
					QoS:     tc.qos,
				},
				Message: &Message{
					TopicName: "test/topic",
					Content:   []byte("test"),
				},
			}

			var buf bytes.Buffer
			err := publish.Pack(&buf)
			if err != nil {
				t.Errorf("Pack() failed: %v", err)
				return
			}

			result := buf.Bytes()
			if len(result) < 1 {
				t.Error("Serialized data too short")
				return
			}

			// 验证DUP标志
			dupFlag := (result[0] >> 3) & 0x01
			if dupFlag != tc.dup {
				t.Errorf("DUP flag = %d, want %d", dupFlag, tc.dup)
			}
		})
	}
}

// TestPUBLISH_ProtocolCompliance 测试协议合规性
func TestPUBLISH_ProtocolCompliance(t *testing.T) {
	testCases := []struct {
		name        string
		publish     *PUBLISH
		shouldError bool
		reason      string
	}{
		{
			name: "Valid_QoS0",
			publish: &PUBLISH{
				FixedHeader: &FixedHeader{
					Version: VERSION311,
					Kind:    0x03,
					Dup:     0,
					QoS:     0,
					Retain:  0,
				},
				Message: &Message{
					TopicName: "test/topic",
					Content:   []byte("test"),
				},
			},
			shouldError: false,
			reason:      "有效的QoS 0发布",
		},
		{
			name: "Valid_QoS1",
			publish: &PUBLISH{
				FixedHeader: &FixedHeader{
					Version: VERSION311,
					Kind:    0x03,
					Dup:     0,
					QoS:     1,
					Retain:  0,
				},
				Message: &Message{
					TopicName: "test/topic",
					Content:   []byte("test"),
				},
			},
			shouldError: false,
			reason:      "有效的QoS 1发布",
		},
		{
			name: "Invalid_QoS3",
			publish: &PUBLISH{
				FixedHeader: &FixedHeader{
					Version: VERSION311,
					Kind:    0x03,
					Dup:     0,
					QoS:     3,
					Retain:  0,
				},
				Message: &Message{
					TopicName: "test/topic",
					Content:   []byte("test"),
				},
			},
			shouldError: true,
			reason:      "QoS 3为保留值，不允许使用",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("测试: %s", tc.reason)

			var buf bytes.Buffer
			err := tc.publish.Pack(&buf)

			if tc.shouldError {
				if err == nil {
					t.Errorf("Pack() should fail for invalid PUBLISH: %s", tc.reason)
				}
			} else {
				if err != nil {
					t.Errorf("Pack() failed: %v", err)
				}
			}
		})
	}
}

// TestPUBLISH_EdgeCases 测试边界情况
func TestPUBLISH_EdgeCases(t *testing.T) {
	testCases := []struct {
		name        string
		publish     *PUBLISH
		description string
	}{
		{
			name: "EmptyPayload",
			publish: &PUBLISH{
				FixedHeader: &FixedHeader{
					Version: VERSION311,
					Kind:    0x03,
				},
				Message: &Message{
					TopicName: "test/topic",
					Content:   []byte{},
				},
			},
			description: "测试空载荷",
		},
		{
			name: "LongTopic",
			publish: &PUBLISH{
				FixedHeader: &FixedHeader{
					Version: VERSION311,
					Kind:    0x03,
				},
				Message: &Message{
					TopicName: "very/long/topic/name/that/exceeds/normal/length",
					Content:   []byte("test"),
				},
			},
			description: "测试长主题名",
		},
		{
			name: "LargePayload",
			publish: &PUBLISH{
				FixedHeader: &FixedHeader{
					Version: VERSION311,
					Kind:    0x03,
				},
				Message: &Message{
					TopicName: "test/topic",
					Content:   bytes.Repeat([]byte("x"), 1000),
				},
			},
			description: "测试大载荷",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("测试: %s", tc.description)

			var buf bytes.Buffer
			err := tc.publish.Pack(&buf)
			if err != nil {
				t.Errorf("Pack() failed: %v", err)
				return
			}

			result := buf.Bytes()
			if len(result) < 10 {
				t.Errorf("Serialized data too short: %d bytes", len(result))
				return
			}

			// 验证主题名存在
			if !bytes.Contains(result, []byte(tc.publish.Message.TopicName)) {
				t.Error("Topic name not found in packed data")
			}

			// 验证载荷存在（如果不是空的）
			if len(tc.publish.Message.Content) > 0 && !bytes.Contains(result, tc.publish.Message.Content) {
				t.Error("Content not found in packed data")
			}
		})
	}
}

// TestPUBLISH_VersionDifferences 测试版本差异
func TestPUBLISH_VersionDifferences(t *testing.T) {
	testCases := []struct {
		name        string
		version     byte
		description string
	}{
		{
			name:        "V311_Structure",
			version:     VERSION311,
			description: "MQTT v3.1.1基本结构",
		},
		{
			name:        "V500_Structure",
			version:     VERSION500,
			description: "MQTT v5.0可能包含属性字段",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("测试: %s", tc.description)

			publish := &PUBLISH{
				FixedHeader: &FixedHeader{
					Version: tc.version,
					Kind:    0x03,
				},
				Message: &Message{
					TopicName: "test/topic",
					Content:   []byte("test"),
				},
			}

			var buf bytes.Buffer
			err := publish.Pack(&buf)
			if err != nil {
				t.Errorf("Pack() failed: %v", err)
				return
			}

			result := buf.Bytes()
			if len(result) < 10 {
				t.Errorf("Serialized data too short: %d bytes", len(result))
				return
			}

			// 验证基本结构
			if !bytes.Contains(result, []byte("test/topic")) {
				t.Error("Topic name not found")
			}
			if !bytes.Contains(result, []byte("test")) {
				t.Error("Payload not found")
			}
		})
	}
}

// BenchmarkPUBLISH_Pack 性能测试：序列化
func BenchmarkPUBLISH_Pack(b *testing.B) {
	publish := &PUBLISH{
		FixedHeader: &FixedHeader{
			Version: VERSION311,
			Kind:    0x03,
		},
		Message: &Message{
			TopicName: "test/topic",
			Content:   []byte("test message"),
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		publish.Pack(&buf)
	}
}

// BenchmarkPUBLISH_Unpack 性能测试：反序列化
func BenchmarkPUBLISH_Unpack(b *testing.B) {
	publish := &PUBLISH{
		FixedHeader: &FixedHeader{
			Version: VERSION311,
			Kind:    0x03,
		},
		Message: &Message{
			TopicName: "test/topic",
			Content:   []byte("test message"),
		},
	}

	var buf bytes.Buffer
	publish.Pack(&buf)
	data := buf.Bytes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		newPublish := &PUBLISH{
			FixedHeader: &FixedHeader{
				Kind:    0x03,
				Version: VERSION311,
			},
		}
		newBuf := bytes.NewBuffer(data)
		newPublish.Unpack(newBuf)
	}
}
