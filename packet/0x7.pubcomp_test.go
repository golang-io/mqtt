package packet

import (
	"bytes"
	"testing"
)

// TestPUBCOMP_Kind 测试PUBCOMP报文的类型标识符
// 参考MQTT v3.1.1章节 3.7 PUBCOMP - Publish Complete (QoS 2 publish received, part 3)
// 参考MQTT v5.0章节 3.7 PUBCOMP - Publish Complete (QoS 2 publish received, part 3)
func TestPUBCOMP_Kind(t *testing.T) {
	pubcomp := &PUBCOMP{FixedHeader: &FixedHeader{Kind: 0x07}}
	if pubcomp.Kind() != 0x07 {
		t.Errorf("PUBCOMP.Kind() = %d, want 0x07", pubcomp.Kind())
	}
}

// TestPUBCOMP_Pack 测试PUBCOMP报文的序列化
// 参考MQTT v3.1.1章节 3.7.2 PUBCOMP Variable Header
// 参考MQTT v5.0章节 3.7.2 PUBCOMP Variable Header
func TestPUBCOMP_Pack(t *testing.T) {
	testCases := []struct {
		name     string
		pubcomp  *PUBCOMP
		version  byte
		expected []byte
	}{
		{
			name: "V311_BasicPubcomp",
			pubcomp: &PUBCOMP{
				FixedHeader: &FixedHeader{
					Version:         VERSION311,
					Kind:            0x07,
					Dup:             0,
					QoS:             0,
					Retain:          0,
					RemainingLength: 0, // 将在Pack时计算
				},
				PacketID: 12345,
			},
			version: VERSION311,
			expected: []byte{
				0x70, 0x02, // 固定报头: PUBCOMP, 标志位0, 剩余长度2
				0x30, 0x39, // 报文标识符: 12345
			},
		},
		{
			name: "V500_BasicPubcomp",
			pubcomp: &PUBCOMP{
				FixedHeader: &FixedHeader{
					Version:         VERSION500,
					Kind:            0x07,
					Dup:             0,
					QoS:             0,
					Retain:          0,
					RemainingLength: 0, // 将在Pack时计算
				},
				PacketID:   12345,
				ReasonCode: ReasonCode{Code: 0x00}, // 成功
				Props:      &PubcompProperties{},
			},
			version: VERSION500,
			expected: []byte{
				0x70, 0x04, // 固定报头: PUBCOMP, 标志位0, 剩余长度4
				0x30, 0x39, // 报文标识符: 12345
				0x00, // 原因码: 成功
				0x00, // 属性长度: 0
			},
		},
		{
			name: "V500_PubcompWithReasonCode",
			pubcomp: &PUBCOMP{
				FixedHeader: &FixedHeader{
					Version:         VERSION500,
					Kind:            0x07,
					Dup:             0,
					QoS:             0,
					Retain:          0,
					RemainingLength: 0,
				},
				PacketID:   12345,
				ReasonCode: ReasonCode{Code: 0x92}, // 报文标识符未找到
				Props:      &PubcompProperties{},
			},
			version: VERSION500,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tc.pubcomp.Pack(&buf)
			if err != nil {
				t.Fatalf("Pack() failed: %v", err)
			}

			result := buf.Bytes()

			// 验证基本结构
			if len(result) < 4 {
				t.Errorf("result too short: %d bytes", len(result))
			}

			// 验证报文类型和标志位
			if result[0] != 0x70 {
				t.Errorf("packet type and flags = %02x, want 0x70", result[0])
			}

			// 验证报文标识符
			if result[2] != 0x30 || result[3] != 0x39 {
				t.Errorf("packet ID = %02x%02x, want 0x3039", result[2], result[3])
			}

			// 对于v5.0，验证扩展结构
			if tc.version == VERSION500 {
				if len(result) < 6 {
					t.Errorf("v5.0 result too short: %d bytes", len(result))
				}
				if result[4] != tc.pubcomp.ReasonCode.Code {
					t.Errorf("reason code = %02x, want %02x", result[4], tc.pubcomp.ReasonCode.Code)
				}
			}
		})
	}
}

// TestPUBCOMP_Unpack 测试PUBCOMP报文的反序列化
func TestPUBCOMP_Unpack(t *testing.T) {
	testCases := []struct {
		name     string
		data     []byte
		version  byte
		expected *PUBCOMP
	}{
		{
			name: "V311_BasicPubcomp",
			data: []byte{
				0x70, 0x02, // 固定报头: PUBCOMP, 标志位0, 剩余长度2
				0x30, 0x39, // 报文标识符: 12345
			},
			version: VERSION311,
			expected: &PUBCOMP{
				FixedHeader: &FixedHeader{
					Version:         VERSION311,
					Kind:            0x07,
					Dup:             0,
					QoS:             0,
					Retain:          0,
					RemainingLength: 2,
				},
				PacketID: 12345,
			},
		},
		{
			name: "V500_BasicPubcomp",
			data: []byte{
				0x70, 0x04, // 固定报头: PUBCOMP, 标志位0, 剩余长度4
				0x30, 0x39, // 报文标识符: 12345
				0x00, // 原因码: 成功
				0x00, // 属性长度: 0
			},
			version: VERSION500,
			expected: &PUBCOMP{
				FixedHeader: &FixedHeader{
					Version:         VERSION500,
					Kind:            0x07,
					Dup:             0,
					QoS:             0,
					Retain:          0,
					RemainingLength: 4,
				},
				PacketID:   12345,
				ReasonCode: ReasonCode{Code: 0x00},
				Props:      &PubcompProperties{},
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

			pubcomp := &PUBCOMP{
				FixedHeader: fixedHeader,
			}

			err := pubcomp.Unpack(buf)
			if err != nil {
				t.Fatalf("Unpack() failed: %v", err)
			}

			if pubcomp.PacketID != tc.expected.PacketID {
				t.Errorf("PacketID = %d, want %d", pubcomp.PacketID, tc.expected.PacketID)
			}

			if tc.version == VERSION500 {
				if pubcomp.ReasonCode.Code != tc.expected.ReasonCode.Code {
					t.Errorf("ReasonCode = %02x, want %02x", pubcomp.ReasonCode.Code, tc.expected.ReasonCode.Code)
				}
			}
		})
	}
}

// TestPUBCOMP_ProtocolCompliance 测试PUBCOMP报文的协议合规性
func TestPUBCOMP_ProtocolCompliance(t *testing.T) {
	t.Run("V311_FlagsMustBeZero", func(t *testing.T) {
		// v3.1.1中标志位必须为0
		pubcomp := &PUBCOMP{
			FixedHeader: &FixedHeader{
				Version: VERSION311,
				Kind:    0x07,
				Dup:     1, // 违反协议
				QoS:     0,
				Retain:  0,
			},
			PacketID: 12345,
		}

		var buf bytes.Buffer
		err := pubcomp.Pack(&buf)
		if err != nil {
			t.Fatalf("Pack() failed: %v", err)
		}

		// 验证标志位被正确设置
		result := buf.Bytes()
		if result[0] != 0x70 {
			t.Errorf("flags not properly set: %02x", result[0])
		}
	})

	t.Run("V500_ReasonCodeSupport", func(t *testing.T) {
		// v5.0支持原因码
		pubcomp := &PUBCOMP{
			FixedHeader: &FixedHeader{
				Version: VERSION500,
				Kind:    0x07,
				Dup:     0,
				QoS:     0,
				Retain:  0,
			},
			PacketID:   12345,
			ReasonCode: ReasonCode{Code: 0x92}, // 报文标识符未找到
			Props:      &PubcompProperties{},
		}

		var buf bytes.Buffer
		err := pubcomp.Pack(&buf)
		if err != nil {
			t.Fatalf("Pack() failed: %v", err)
		}

		result := buf.Bytes()
		if len(result) < 6 {
			t.Errorf("result too short: %d bytes", len(result))
		}
		if result[4] != 0x92 {
			t.Errorf("reason code not preserved: %02x", result[4])
		}
	})
}

// TestPUBCOMP_EdgeCases 测试PUBCOMP报文的边界情况
func TestPUBCOMP_EdgeCases(t *testing.T) {
	t.Run("PacketIDZero", func(t *testing.T) {
		pubcomp := &PUBCOMP{
			FixedHeader: &FixedHeader{
				Version: VERSION311,
				Kind:    0x07,
				Dup:     0,
				QoS:     0,
				Retain:  0,
			},
			PacketID: 0, // 边界值
		}

		var buf bytes.Buffer
		err := pubcomp.Pack(&buf)
		if err != nil {
			t.Fatalf("Pack() failed: %v", err)
		}

		result := buf.Bytes()
		if result[2] != 0x00 || result[3] != 0x00 {
			t.Errorf("packet ID 0 not properly encoded: %02x%02x", result[2], result[3])
		}
	})

	t.Run("PacketIDMax", func(t *testing.T) {
		pubcomp := &PUBCOMP{
			FixedHeader: &FixedHeader{
				Version: VERSION311,
				Kind:    0x07,
				Dup:     0,
				QoS:     0,
				Retain:  0,
			},
			PacketID: 65535, // 最大值
		}

		var buf bytes.Buffer
		err := pubcomp.Pack(&buf)
		if err != nil {
			t.Fatalf("Pack() failed: %v", err)
		}

		result := buf.Bytes()
		if result[2] != 0xFF || result[3] != 0xFF {
			t.Errorf("packet ID 65535 not properly encoded: %02x%02x", result[2], result[3])
		}
	})
}

// TestPubcompProperties_Pack 测试发布完成属性的序列化
func TestPubcompProperties_Pack(t *testing.T) {
	props := &PubcompProperties{
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

// TestPubcompProperties_Unpack 测试发布完成属性的反序列化
func TestPubcompProperties_Unpack(t *testing.T) {
	// 先创建一个属性并序列化
	originalProps := &PubcompProperties{
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
	newProps := &PubcompProperties{}
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

// TestPUBCOMP_QoS2Flow 测试PUBCOMP在QoS 2流程中的作用
func TestPUBCOMP_QoS2Flow(t *testing.T) {
	t.Run("QoS2FlowSequence", func(t *testing.T) {
		// 模拟QoS 2的完整流程
		packetID := uint16(12345)

		// 1. PUBLISH (QoS=2)
		publish := &PUBLISH{
			FixedHeader: &FixedHeader{
				Version: VERSION311,
				Kind:    0x03,
				Dup:     0,
				QoS:     2,
				Retain:  0,
			},
			PacketID: packetID,
			Message: &Message{
				TopicName: "test/topic",
				Content:   []byte("test message"),
			},
		}

		// 2. PUBREC (QoS 2第一步)
		pubrec := &PUBREC{
			FixedHeader: &FixedHeader{
				Version: VERSION311,
				Kind:    0x05,
				Dup:     0,
				QoS:     0,
				Retain:  0,
			},
			PacketID: packetID,
		}

		// 3. PUBREL (QoS 2第二步)
		pubrel := &PUBREL{
			FixedHeader: &FixedHeader{
				Version: VERSION311,
				Kind:    0x06,
				Dup:     0,
				QoS:     1,
				Retain:  0,
			},
			PacketID: packetID,
		}

		// 4. PUBCOMP (QoS 2第三步)
		pubcomp := &PUBCOMP{
			FixedHeader: &FixedHeader{
				Version: VERSION311,
				Kind:    0x07,
				Dup:     0,
				QoS:     0,
				Retain:  0,
			},
			PacketID: packetID,
		}

		// 验证所有报文使用相同的PacketID
		if publish.PacketID != pubrec.PacketID ||
			pubrec.PacketID != pubrel.PacketID ||
			pubrel.PacketID != pubcomp.PacketID {
			t.Error("All QoS 2 packets must use the same PacketID")
		}

		// 验证PUBCOMP的QoS为0
		if pubcomp.QoS != 0 {
			t.Errorf("PUBCOMP QoS = %d, want 0", pubcomp.QoS)
		}

		// 验证QoS 2流程的标志位设置
		if publish.QoS != 2 {
			t.Errorf("PUBLISH QoS = %d, want 2", publish.QoS)
		}
		if pubrec.QoS != 0 {
			t.Errorf("PUBREC QoS = %d, want 0", pubrec.QoS)
		}
		if pubrel.QoS != 1 {
			t.Errorf("PUBREL QoS = %d, want 1", pubrel.QoS)
		}
		if pubcomp.QoS != 0 {
			t.Errorf("PUBCOMP QoS = %d, want 0", pubcomp.QoS)
		}
	})
}

// TestPUBCOMP_ReasonCodes 测试PUBCOMP的原因码
func TestPUBCOMP_ReasonCodes(t *testing.T) {
	t.Run("V500_ValidReasonCodes", func(t *testing.T) {
		validReasonCodes := []byte{
			0x00, // 成功
			0x10, // 无匹配订阅者
			0x80, // 未指定错误
			0x83, // 实现特定错误
			0x87, // 未授权
			0x92, // 报文标识符未找到
		}

		for _, reasonCode := range validReasonCodes {
			pubcomp := &PUBCOMP{
				FixedHeader: &FixedHeader{
					Version: VERSION500,
					Kind:    0x07,
					Dup:     0,
					QoS:     0,
					Retain:  0,
				},
				PacketID:   12345,
				ReasonCode: ReasonCode{Code: reasonCode},
				Props:      &PubcompProperties{},
			}

			var buf bytes.Buffer
			err := pubcomp.Pack(&buf)
			if err != nil {
				t.Fatalf("Pack() failed for reason code %02x: %v", reasonCode, err)
			}

			result := buf.Bytes()
			if result[4] != reasonCode {
				t.Errorf("reason code not preserved for %02x: got %02x", reasonCode, result[4])
			}
		}
	})
}

// BenchmarkPUBCOMP_Pack 性能测试
func BenchmarkPUBCOMP_Pack(b *testing.B) {
	pubcomp := &PUBCOMP{
		FixedHeader: &FixedHeader{
			Version: VERSION500,
			Kind:    0x07,
			Dup:     0,
			QoS:     0,
			Retain:  0,
		},
		PacketID:   12345,
		ReasonCode: ReasonCode{Code: 0x00},
		Props:      &PubcompProperties{},
	}

	var buf bytes.Buffer
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		pubcomp.Pack(&buf)
	}
}

// BenchmarkPUBCOMP_Unpack 性能测试
func BenchmarkPUBCOMP_Unpack(b *testing.B) {
	data := []byte{
		0x70, 0x04, // 固定报头
		0x30, 0x39, // 报文标识符
		0x00, // 原因码
		0x00, // 属性长度
	}

	pubcomp := &PUBCOMP{
		FixedHeader: &FixedHeader{
			Version:         VERSION500,
			Kind:            0x07,
			Dup:             0,
			QoS:             0,
			Retain:          0,
			RemainingLength: 4,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := bytes.NewBuffer(data)
		pubcomp.Unpack(buf)
	}
}
