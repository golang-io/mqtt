package packet

import (
	"bytes"
	"testing"
)

// TestPUBREL_Kind 测试PUBREL报文的类型标识符
// 参考MQTT v3.1.1章节 3.6 PUBREL - Publish Release (QoS 2 publish received, part 2)
// 参考MQTT v5.0章节 3.6 PUBREL - Publish Release (QoS 2 publish received, part 2)
func TestPUBREL_Kind(t *testing.T) {
	pubrel := &PUBREL{FixedHeader: &FixedHeader{Kind: 0x06}}
	if pubrel.Kind() != 0x06 {
		t.Errorf("PUBREL.Kind() = %d, want 0x06", pubrel.Kind())
	}
}

// TestPUBREL_Pack 测试PUBREL报文的序列化
// 参考MQTT v3.1.1章节 3.6.2 PUBREL Variable Header
// 参考MQTT v5.0章节 3.6.2 PUBREL Variable Header
func TestPUBREL_Pack(t *testing.T) {
	testCases := []struct {
		name     string
		pubrel   *PUBREL
		version  byte
		expected []byte
	}{
		{
			name: "V311_BasicPubrel",
			pubrel: &PUBREL{
				FixedHeader: &FixedHeader{
					Version:         VERSION311,
					Kind:            0x06,
					Dup:             0,
					QoS:             1, // PUBREL的QoS必须为1
					Retain:          0,
					RemainingLength: 0, // 将在Pack时计算
				},
				PacketID: 12345,
			},
			version: VERSION311,
			expected: []byte{
				0x62, 0x02, // 固定报头: PUBREL, 标志位010, 剩余长度2
				0x30, 0x39, // 报文标识符: 12345
			},
		},
		{
			name: "V500_BasicPubrel",
			pubrel: &PUBREL{
				FixedHeader: &FixedHeader{
					Version:         VERSION500,
					Kind:            0x06,
					Dup:             0,
					QoS:             1, // PUBREL的QoS必须为1
					Retain:          0,
					RemainingLength: 0, // 将在Pack时计算
				},
				PacketID:   12345,
				ReasonCode: ReasonCode{Code: 0x00}, // 成功
				Props:      &PubrelProperties{},
			},
			version: VERSION500,
			expected: []byte{
				0x62, 0x04, // 固定报头: PUBREL, 标志位010, 剩余长度4
				0x30, 0x39, // 报文标识符: 12345
				0x00, // 原因码: 成功
				0x00, // 属性长度: 0
			},
		},
		{
			name: "V500_PubrelWithReasonCode",
			pubrel: &PUBREL{
				FixedHeader: &FixedHeader{
					Version:         VERSION500,
					Kind:            0x06,
					Dup:             0,
					QoS:             1,
					Retain:          0,
					RemainingLength: 0,
				},
				PacketID:   12345,
				ReasonCode: ReasonCode{Code: 0x92}, // 报文标识符未找到
				Props:      &PubrelProperties{},
			},
			version: VERSION500,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tc.pubrel.Pack(&buf)
			if err != nil {
				t.Fatalf("Pack() failed: %v", err)
			}

			result := buf.Bytes()

			// 验证基本结构
			if len(result) < 4 {
				t.Errorf("result too short: %d bytes", len(result))
			}

			// 验证报文类型和标志位
			if result[0] != 0x62 {
				t.Errorf("packet type and flags = %02x, want 0x62", result[0])
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
				if result[4] != tc.pubrel.ReasonCode.Code {
					t.Errorf("reason code = %02x, want %02x", result[4], tc.pubrel.ReasonCode.Code)
				}
			}
		})
	}
}

// TestPUBREL_Unpack 测试PUBREL报文的反序列化
func TestPUBREL_Unpack(t *testing.T) {
	testCases := []struct {
		name     string
		data     []byte
		version  byte
		expected *PUBREL
	}{
		{
			name: "V311_BasicPubrel",
			data: []byte{
				0x62, 0x02, // 固定报头: PUBREL, 标志位010, 剩余长度2
				0x30, 0x39, // 报文标识符: 12345
			},
			version: VERSION311,
			expected: &PUBREL{
				FixedHeader: &FixedHeader{
					Version:         VERSION311,
					Kind:            0x06,
					Dup:             0,
					QoS:             1,
					Retain:          0,
					RemainingLength: 2,
				},
				PacketID: 12345,
			},
		},
		{
			name: "V500_BasicPubrel",
			data: []byte{
				0x62, 0x04, // 固定报头: PUBREL, 标志位010, 剩余长度4
				0x30, 0x39, // 报文标识符: 12345
				0x00, // 原因码: 成功
				0x00, // 属性长度: 0
			},
			version: VERSION500,
			expected: &PUBREL{
				FixedHeader: &FixedHeader{
					Version:         VERSION500,
					Kind:            0x06,
					Dup:             0,
					QoS:             1,
					Retain:          0,
					RemainingLength: 4,
				},
				PacketID:   12345,
				ReasonCode: ReasonCode{Code: 0x00},
				Props:      &PubrelProperties{},
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

			pubrel := &PUBREL{
				FixedHeader: fixedHeader,
			}

			err := pubrel.Unpack(buf)
			if err != nil {
				t.Fatalf("Unpack() failed: %v", err)
			}

			if pubrel.PacketID != tc.expected.PacketID {
				t.Errorf("PacketID = %d, want %d", pubrel.PacketID, tc.expected.PacketID)
			}

			if tc.version == VERSION500 {
				if pubrel.ReasonCode.Code != tc.expected.ReasonCode.Code {
					t.Errorf("ReasonCode = %02x, want %02x", pubrel.ReasonCode.Code, tc.expected.ReasonCode.Code)
				}
			}
		})
	}
}

// TestPUBREL_ProtocolCompliance 测试PUBREL报文的协议合规性
func TestPUBREL_ProtocolCompliance(t *testing.T) {
	t.Run("V311_QoSMustBeOne", func(t *testing.T) {
		// v3.1.1中QoS必须为1
		pubrel := &PUBREL{
			FixedHeader: &FixedHeader{
				Version: VERSION311,
				Kind:    0x06,
				Dup:     0,
				QoS:     0, // 违反协议
				Retain:  0,
			},
			PacketID: 12345,
		}

		var buf bytes.Buffer
		err := pubrel.Pack(&buf)
		if err != nil {
			t.Fatalf("Pack() failed: %v", err)
		}

		// 验证QoS被正确设置
		result := buf.Bytes()
		if result[0] != 0x62 {
			t.Errorf("QoS not properly set: %02x", result[0])
		}
	})

	t.Run("V500_ReasonCodeSupport", func(t *testing.T) {
		// v5.0支持原因码
		pubrel := &PUBREL{
			FixedHeader: &FixedHeader{
				Version: VERSION500,
				Kind:    0x06,
				Dup:     0,
				QoS:     1,
				Retain:  0,
			},
			PacketID:   12345,
			ReasonCode: ReasonCode{Code: 0x92}, // 报文标识符未找到
			Props:      &PubrelProperties{},
		}

		var buf bytes.Buffer
		err := pubrel.Pack(&buf)
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

// TestPUBREL_EdgeCases 测试PUBREL报文的边界情况
func TestPUBREL_EdgeCases(t *testing.T) {
	t.Run("PacketIDZero", func(t *testing.T) {
		pubrel := &PUBREL{
			FixedHeader: &FixedHeader{
				Version: VERSION311,
				Kind:    0x06,
				Dup:     0,
				QoS:     1,
				Retain:  0,
			},
			PacketID: 0, // 边界值
		}

		var buf bytes.Buffer
		err := pubrel.Pack(&buf)
		if err != nil {
			t.Fatalf("Pack() failed: %v", err)
		}

		result := buf.Bytes()
		if result[2] != 0x00 || result[3] != 0x00 {
			t.Errorf("packet ID 0 not properly encoded: %02x%02x", result[2], result[3])
		}
	})

	t.Run("PacketIDMax", func(t *testing.T) {
		pubrel := &PUBREL{
			FixedHeader: &FixedHeader{
				Version: VERSION311,
				Kind:    0x06,
				Dup:     0,
				QoS:     1,
				Retain:  0,
			},
			PacketID: 65535, // 最大值
		}

		var buf bytes.Buffer
		err := pubrel.Pack(&buf)
		if err != nil {
			t.Fatalf("Pack() failed: %v", err)
		}

		result := buf.Bytes()
		if result[2] != 0xFF || result[3] != 0xFF {
			t.Errorf("packet ID 65535 not properly encoded: %02x%02x", result[2], result[3])
		}
	})
}

// TestPubrelProperties_Pack 测试发布释放属性的序列化
func TestPubrelProperties_Pack(t *testing.T) {
	props := &PubrelProperties{
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

// TestPubrelProperties_Unpack 测试发布释放属性的反序列化
func TestPubrelProperties_Unpack(t *testing.T) {
	// 先创建一个属性并序列化
	originalProps := &PubrelProperties{
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
	newProps := &PubrelProperties{}
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

// TestPUBREL_QoS2Flow 测试PUBREL在QoS 2流程中的作用
func TestPUBREL_QoS2Flow(t *testing.T) {
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

		// 验证PUBREL的QoS为1
		if pubrel.QoS != 1 {
			t.Errorf("PUBREL QoS = %d, want 1", pubrel.QoS)
		}
	})
}

// BenchmarkPUBREL_Pack 性能测试
func BenchmarkPUBREL_Pack(b *testing.B) {
	pubrel := &PUBREL{
		FixedHeader: &FixedHeader{
			Version: VERSION500,
			Kind:    0x06,
			Dup:     0,
			QoS:     1,
			Retain:  0,
		},
		PacketID:   12345,
		ReasonCode: ReasonCode{Code: 0x00},
		Props:      &PubrelProperties{},
	}

	var buf bytes.Buffer
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		pubrel.Pack(&buf)
	}
}

// BenchmarkPUBREL_Unpack 性能测试
func BenchmarkPUBREL_Unpack(b *testing.B) {
	data := []byte{
		0x62, 0x04, // 固定报头
		0x30, 0x39, // 报文标识符
		0x00, // 原因码
		0x00, // 属性长度
	}

	pubrel := &PUBREL{
		FixedHeader: &FixedHeader{
			Version:         VERSION500,
			Kind:            0x06,
			Dup:             0,
			QoS:             1,
			Retain:          0,
			RemainingLength: 4,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := bytes.NewBuffer(data)
		pubrel.Unpack(buf)
	}
}
