package packet

import (
	"bytes"
	"testing"
)

// TestPUBREC_Kind 测试PUBREC报文的类型标识符
// 参考MQTT v3.1.1章节 3.5 PUBREC - Publish received (QoS 2 publish received, part 1)
// 参考MQTT v5.0章节 3.5 PUBREC - Publish received (QoS 2 publish received, part 1)
func TestPUBREC_Kind(t *testing.T) {
	pubrec := &PUBREC{FixedHeader: &FixedHeader{Kind: 0x05}}
	if pubrec.Kind() != 0x05 {
		t.Errorf("PUBREC.Kind() = %d, want 0x05", pubrec.Kind())
	}
}

// TestPUBREC_Pack 测试PUBREC报文的序列化
// 参考MQTT v3.1.1章节 3.5.2 PUBREC Variable Header
// 参考MQTT v5.0章节 3.5.2 PUBREC Variable Header
func TestPUBREC_Pack(t *testing.T) {
	testCases := []struct {
		name     string
		pubrec   *PUBREC
		version  byte
		expected []byte
	}{
		{
			name: "V311_BasicPubrec",
			pubrec: &PUBREC{
				FixedHeader: &FixedHeader{
					Version:         VERSION311,
					Kind:            0x05,
					Dup:             0,
					QoS:             0,
					Retain:          0,
					RemainingLength: 0, // 将在Pack时计算
				},
				PacketID: 12345,
			},
			version: VERSION311,
			expected: []byte{
				0x50, 0x02, // 固定报头: PUBREC, 标志位0, 剩余长度2
				0x30, 0x39, // 报文标识符: 12345
			},
		},
		{
			name: "V500_BasicPubrec",
			pubrec: &PUBREC{
				FixedHeader: &FixedHeader{
					Version:         VERSION500,
					Kind:            0x05,
					Dup:             0,
					QoS:             0,
					Retain:          0,
					RemainingLength: 0, // 将在Pack时计算
				},
				PacketID:   12345,
				ReasonCode: ReasonCode{Code: 0x00}, // 成功
				Props:      &PubrecProperties{},
			},
			version: VERSION500,
			expected: []byte{
				0x50, 0x04, // 固定报头: PUBREC, 标志位0, 剩余长度4
				0x30, 0x39, // 报文标识符: 12345
				0x00, // 原因码: 成功
				0x00, // 属性长度: 0
			},
		},
		{
			name: "V500_PubrecWithReasonString",
			pubrec: &PUBREC{
				FixedHeader: &FixedHeader{
					Version:         VERSION500,
					Kind:            0x05,
					Dup:             0,
					QoS:             0,
					Retain:          0,
					RemainingLength: 0, // 将在Pack时计算
				},
				PacketID:   12345,
				ReasonCode: ReasonCode{Code: 0x10}, // 无匹配订阅者
				Props: &PubrecProperties{
					ReasonString: "No subscribers found",
				},
			},
			version: VERSION500,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tc.pubrec.Pack(&buf)
			if err != nil {
				t.Fatalf("Pack() failed: %v", err)
			}

			result := buf.Bytes()

			// 对于v3.1.1，验证基本结构
			if tc.version == VERSION311 {
				if len(result) < 4 {
					t.Errorf("result too short: %d bytes", len(result))
				}
				if result[0] != 0x50 {
					t.Errorf("packet type = %02x, want 0x50", result[0])
				}
				if result[2] != 0x30 || result[3] != 0x39 {
					t.Errorf("packet ID = %02x%02x, want 0x3039", result[2], result[3])
				}
			}

			// 对于v5.0，验证扩展结构
			if tc.version == VERSION500 {
				if len(result) < 6 {
					t.Errorf("result too short: %d bytes", len(result))
				}
				if result[0] != 0x50 {
					t.Errorf("packet type = %02x, want 0x50", result[0])
				}
				if result[2] != 0x30 || result[3] != 0x39 {
					t.Errorf("packet ID = %02x%02x, want 0x3039", result[2], result[3])
				}
				if result[4] != tc.pubrec.ReasonCode.Code {
					t.Errorf("reason code = %02x, want %02x", result[4], tc.pubrec.ReasonCode.Code)
				}
			}
		})
	}
}

// TestPUBREC_Unpack 测试PUBREC报文的反序列化
func TestPUBREC_Unpack(t *testing.T) {
	testCases := []struct {
		name     string
		data     []byte
		version  byte
		expected *PUBREC
	}{
		{
			name: "V311_BasicPubrec",
			data: []byte{
				0x50, 0x02, // 固定报头: PUBREC, 标志位0, 剩余长度2
				0x30, 0x39, // 报文标识符: 12345
			},
			version: VERSION311,
			expected: &PUBREC{
				FixedHeader: &FixedHeader{
					Version:         VERSION311,
					Kind:            0x05,
					Dup:             0,
					QoS:             0,
					Retain:          0,
					RemainingLength: 2,
				},
				PacketID: 12345,
			},
		},
		{
			name: "V500_BasicPubrec",
			data: []byte{
				0x50, 0x04, // 固定报头: PUBREC, 标志位0, 剩余长度4
				0x30, 0x39, // 报文标识符: 12345
				0x00, // 原因码: 成功
				0x00, // 属性长度: 0
			},
			version: VERSION500,
			expected: &PUBREC{
				FixedHeader: &FixedHeader{
					Version:         VERSION500,
					Kind:            0x05,
					Dup:             0,
					QoS:             0,
					Retain:          0,
					RemainingLength: 4,
				},
				PacketID:   12345,
				ReasonCode: ReasonCode{Code: 0x00},
				Props:      &PubrecProperties{},
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

			pubrec := &PUBREC{
				FixedHeader: fixedHeader,
			}

			err := pubrec.Unpack(buf)
			if err != nil {
				t.Fatalf("Unpack() failed: %v", err)
			}

			if pubrec.PacketID != tc.expected.PacketID {
				t.Errorf("PacketID = %d, want %d", pubrec.PacketID, tc.expected.PacketID)
			}

			if tc.version == VERSION500 {
				if pubrec.ReasonCode.Code != tc.expected.ReasonCode.Code {
					t.Errorf("ReasonCode = %02x, want %02x", pubrec.ReasonCode.Code, tc.expected.ReasonCode.Code)
				}
			}
		})
	}
}

// TestPUBREC_ProtocolCompliance 测试PUBREC报文的协议合规性
func TestPUBREC_ProtocolCompliance(t *testing.T) {
	t.Run("V311_FlagsMustBeZero", func(t *testing.T) {
		// v3.1.1中标志位必须为0
		pubrec := &PUBREC{
			FixedHeader: &FixedHeader{
				Version: VERSION311,
				Kind:    0x05,
				Dup:     1, // 违反协议
				QoS:     0,
				Retain:  0,
			},
			PacketID: 12345,
		}

		var buf bytes.Buffer
		err := pubrec.Pack(&buf)
		if err != nil {
			t.Fatalf("Pack() failed: %v", err)
		}

		// 验证标志位被正确设置
		result := buf.Bytes()
		if result[0] != 0x50 {
			t.Errorf("flags not properly set: %02x", result[0])
		}
	})

	t.Run("V500_ReasonCodeSupport", func(t *testing.T) {
		// v5.0支持原因码
		pubrec := &PUBREC{
			FixedHeader: &FixedHeader{
				Version: VERSION500,
				Kind:    0x05,
				Dup:     0,
				QoS:     0,
				Retain:  0,
			},
			PacketID:   12345,
			ReasonCode: ReasonCode{Code: 0x10}, // 无匹配订阅者
			Props: &PubrecProperties{
				ReasonString: "No subscribers found",
			},
		}

		var buf bytes.Buffer
		err := pubrec.Pack(&buf)
		if err != nil {
			t.Fatalf("Pack() failed: %v", err)
		}

		result := buf.Bytes()
		if len(result) < 6 {
			t.Errorf("result too short: %d bytes", len(result))
		}
		if result[4] != 0x10 {
			t.Errorf("reason code not preserved: %02x", result[4])
		}
	})
}

// TestPUBREC_EdgeCases 测试PUBREC报文的边界情况
func TestPUBREC_EdgeCases(t *testing.T) {
	t.Run("PacketIDZero", func(t *testing.T) {
		pubrec := &PUBREC{
			FixedHeader: &FixedHeader{
				Version: VERSION311,
				Kind:    0x05,
				Dup:     0,
				QoS:     0,
				Retain:  0,
			},
			PacketID: 0, // 边界值
		}

		var buf bytes.Buffer
		err := pubrec.Pack(&buf)
		if err != nil {
			t.Fatalf("Pack() failed: %v", err)
		}

		result := buf.Bytes()
		if result[2] != 0x00 || result[3] != 0x00 {
			t.Errorf("packet ID 0 not properly encoded: %02x%02x", result[2], result[3])
		}
	})

	t.Run("PacketIDMax", func(t *testing.T) {
		pubrec := &PUBREC{
			FixedHeader: &FixedHeader{
				Version: VERSION311,
				Kind:    0x05,
				Dup:     0,
				QoS:     0,
				Retain:  0,
			},
			PacketID: 65535, // 最大值
		}

		var buf bytes.Buffer
		err := pubrec.Pack(&buf)
		if err != nil {
			t.Fatalf("Pack() failed: %v", err)
		}

		result := buf.Bytes()
		if result[2] != 0xFF || result[3] != 0xFF {
			t.Errorf("packet ID 65535 not properly encoded: %02x%02x", result[2], result[3])
		}
	})
}

// TestPubrecProperties_Pack 测试发布收到属性的序列化
func TestPubrecProperties_Pack(t *testing.T) {
	props := &PubrecProperties{
		ReasonString: "Test reason",
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

	// 验证包含原因字符串
	if !bytes.Contains(result, []byte("Test reason")) {
		t.Error("reason string not found in packed result")
	}
}

// TestPubrecProperties_Unpack 测试发布收到属性的反序列化
func TestPubrecProperties_Unpack(t *testing.T) {
	// 先创建一个属性并序列化
	originalProps := &PubrecProperties{
		ReasonString: "Test reason",
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
	newProps := &PubrecProperties{}
	err = newProps.Unpack(buf)
	if err != nil {
		t.Fatalf("Unpack() failed: %v", err)
	}

	if newProps.ReasonString != originalProps.ReasonString {
		t.Errorf("ReasonString = %s, want %s", newProps.ReasonString, originalProps.ReasonString)
	}

	if len(newProps.UserProperty) != len(originalProps.UserProperty) {
		t.Errorf("UserProperty count = %d, want %d", len(newProps.UserProperty), len(originalProps.UserProperty))
	}
}

// BenchmarkPUBREC_Pack 性能测试
func BenchmarkPUBREC_Pack(b *testing.B) {
	pubrec := &PUBREC{
		FixedHeader: &FixedHeader{
			Version: VERSION500,
			Kind:    0x05,
			Dup:     0,
			QoS:     0,
			Retain:  0,
		},
		PacketID:   12345,
		ReasonCode: ReasonCode{Code: 0x00},
		Props:      &PubrecProperties{},
	}

	var buf bytes.Buffer
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		pubrec.Pack(&buf)
	}
}

// BenchmarkPUBREC_Unpack 性能测试
func BenchmarkPUBREC_Unpack(b *testing.B) {
	data := []byte{
		0x50, 0x04, // 固定报头
		0x30, 0x39, // 报文标识符
		0x00, // 原因码
		0x00, // 属性长度
	}

	pubrec := &PUBREC{
		FixedHeader: &FixedHeader{
			Version:         VERSION500,
			Kind:            0x05,
			Dup:             0,
			QoS:             0,
			Retain:          0,
			RemainingLength: 4,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := bytes.NewBuffer(data)
		pubrec.Unpack(buf)
	}
}
