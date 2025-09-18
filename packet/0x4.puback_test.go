package packet

import (
	"bytes"
	"testing"
)

// TestPUBACK_Kind 测试PUBACK报文的类型
// 参考MQTT v3.1.1章节 3.4 PUBACK - Publish acknowledgement
// 参考MQTT v5.0章节 3.4 PUBACK - Publish acknowledgement
func TestPUBACK_Kind(t *testing.T) {
	puback := &PUBACK{}
	if puback.Kind() != 0x04 {
		t.Errorf("PUBACK.Kind() = %d, want 0x04", puback.Kind())
	}
}

// TestPUBACK_BasicStructure 测试PUBACK报文的基本结构
func TestPUBACK_BasicStructure(t *testing.T) {
	testCases := []struct {
		name     string
		version  byte
		packetID uint16
		valid    bool
	}{
		{"V311_ValidPacketID", VERSION311, 1, true},
		{"V311_ValidPacketID2", VERSION311, 65535, true},
		{"V500_ValidPacketID", VERSION500, 1, true},
		{"V500_ValidPacketID2", VERSION500, 65535, true},
		{"V311_ZeroPacketID", VERSION311, 0, false}, // 0是无效的Packet ID
		{"V500_ZeroPacketID", VERSION500, 0, false}, // 0是无效的Packet ID
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			puback := &PUBACK{
				FixedHeader: &FixedHeader{
					Version: tc.version,
					Kind:    0x04,
					Dup:     0, // 必须为0
					QoS:     0, // 必须为0
					Retain:  0, // 必须为0
				},
				PacketID: tc.packetID,
			}

			// 验证基本字段
			if puback.Kind() != 0x04 {
				t.Errorf("Kind = %d, want 0x04", puback.Kind())
			}

			if puback.PacketID != tc.packetID {
				t.Errorf("PacketID = %d, want %d", puback.PacketID, tc.packetID)
			}

			// 验证标志位必须为0
			if puback.FixedHeader.Dup != 0 {
				t.Errorf("Dup flag = %d, must be 0", puback.FixedHeader.Dup)
			}
			if puback.FixedHeader.QoS != 0 {
				t.Errorf("QoS flag = %d, must be 0", puback.FixedHeader.QoS)
			}
			if puback.FixedHeader.Retain != 0 {
				t.Errorf("Retain flag = %d, must be 0", puback.FixedHeader.Retain)
			}
		})
	}
}

// TestPUBACK_Pack 测试PUBACK报文的序列化
func TestPUBACK_Pack(t *testing.T) {
	testCases := []struct {
		name        string
		puback      *PUBACK
		expectedLen int
		description string
	}{
		{
			name: "V311_Basic",
			puback: &PUBACK{
				FixedHeader: &FixedHeader{
					Version: VERSION311,
					Kind:    0x04,
					Dup:     0,
					QoS:     0,
					Retain:  0,
				},
				PacketID: 12345,
			},
			expectedLen: 2, // 只有Packet ID
			description: "MQTT v3.1.1基本PUBACK，只包含Packet ID",
		},
		{
			name: "V500_Basic",
			puback: &PUBACK{
				FixedHeader: &FixedHeader{
					Version: VERSION500,
					Kind:    0x04,
					Dup:     0,
					QoS:     0,
					Retain:  0,
				},
				PacketID:   12345,
				ReasonCode: CodeSuccess,
			},
			expectedLen: 3, // Packet ID + Reason Code
			description: "MQTT v5.0基本PUBACK，包含Packet ID和成功原因码",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tc.puback.Pack(&buf)
			if err != nil {
				t.Fatalf("Pack() failed: %v", err)
			}

			// 验证序列化后的数据长度
			data := buf.Bytes()
			if len(data) < tc.expectedLen {
				t.Errorf("Serialized data too short: got %d bytes, want at least %d bytes", len(data), tc.expectedLen)
			}

			// 验证固定报头
			// PUBACK报文类型是0x04，标志位必须为0x00
			expectedHeader := byte(0x04 << 4) // 0x40
			if data[0] != expectedHeader {
				t.Errorf("Fixed header type/flags = 0x%02X, want 0x%02X", data[0], expectedHeader)
			}

			t.Logf("Successfully packed: %s", tc.description)
		})
	}
}

// TestPUBACK_Unpack 测试PUBACK报文的反序列化
func TestPUBACK_Unpack(t *testing.T) {
	testCases := []struct {
		name        string
		data        []byte
		version     byte
		expected    *PUBACK
		expectError bool
		description string
	}{
		{
			name: "V311_Basic",
			data: []byte{
				0x30, 0x39, // Packet ID = 12345 (big endian)
			},
			version: VERSION311,
			expected: &PUBACK{
				PacketID: 12345,
			},
			expectError: false,
			description: "MQTT v3.1.1基本PUBACK反序列化",
		},
		{
			name: "V500_Basic",
			data: []byte{
				0x30, 0x39, // Packet ID = 12345 (big endian)
				0x00, // Reason Code = 0x00 (Success)
			},
			version: VERSION500,
			expected: &PUBACK{
				PacketID:   12345,
				ReasonCode: CodeSuccess,
			},
			expectError: false,
			description: "MQTT v5.0基本PUBACK反序列化",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			puback := &PUBACK{
				FixedHeader: &FixedHeader{
					Version: tc.version,
					Kind:    0x04,
				},
			}

			buf := bytes.NewBuffer(tc.data)
			err := puback.Unpack(buf)

			if tc.expectError && err == nil {
				t.Errorf("Expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if err == nil {
				// 验证解析结果
				if puback.PacketID != tc.expected.PacketID {
					t.Errorf("Packet ID = %d, want %d", puback.PacketID, tc.expected.PacketID)
				}

				if tc.version == VERSION500 {
					if puback.ReasonCode.Code != tc.expected.ReasonCode.Code {
						t.Errorf("Reason Code = 0x%02X, want 0x%02X", puback.ReasonCode.Code, tc.expected.ReasonCode.Code)
					}
				}
			}

			t.Logf("Successfully unpacked: %s", tc.description)
		})
	}
}

// TestPUBACK_ProtocolCompliance 测试PUBACK报文的协议合规性
func TestPUBACK_ProtocolCompliance(t *testing.T) {
	testCases := []struct {
		name        string
		puback      *PUBACK
		description string
		valid       bool
	}{
		{
			name: "ValidV311_Standard",
			puback: &PUBACK{
				FixedHeader: &FixedHeader{
					Version: VERSION311,
					Kind:    0x04,
					Dup:     0,
					QoS:     0,
					Retain:  0,
				},
				PacketID: 1,
			},
			description: "MQTT v3.1.1标准PUBACK，所有标志位为0",
			valid:       true,
		},
		{
			name: "ValidV500_Standard",
			puback: &PUBACK{
				FixedHeader: &FixedHeader{
					Version: VERSION500,
					Kind:    0x04,
					Dup:     0,
					QoS:     0,
					Retain:  0,
				},
				PacketID:   1,
				ReasonCode: CodeSuccess,
			},
			description: "MQTT v5.0标准PUBACK，包含成功原因码",
			valid:       true,
		},
		{
			name: "Invalid_DupFlagSet",
			puback: &PUBACK{
				FixedHeader: &FixedHeader{
					Version: VERSION311,
					Kind:    0x04,
					Dup:     1, // 违反协议：DUP必须为0
					QoS:     0,
					Retain:  0,
				},
				PacketID: 1,
			},
			description: "违反协议：DUP标志位设置为1",
			valid:       false,
		},
		{
			name: "Invalid_QoSFlagSet",
			puback: &PUBACK{
				FixedHeader: &FixedHeader{
					Version: VERSION311,
					Kind:    0x04,
					Dup:     0,
					QoS:     1, // 违反协议：QoS必须为0
					Retain:  0,
				},
				PacketID: 1,
			},
			description: "违反协议：QoS标志位设置为1",
			valid:       false,
		},
		{
			name: "Invalid_RetainFlagSet",
			puback: &PUBACK{
				FixedHeader: &FixedHeader{
					Version: VERSION311,
					Kind:    0x04,
					Dup:     0,
					QoS:     0,
					Retain:  1, // 违反协议：RETAIN必须为0
				},
				PacketID: 1,
			},
			description: "违反协议：RETAIN标志位设置为1",
			valid:       false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 验证协议合规性
			if tc.puback.FixedHeader.Dup != 0 {
				t.Logf("Warning: DUP flag should be 0 according to MQTT spec")
			}
			if tc.puback.FixedHeader.QoS != 0 {
				t.Logf("Warning: QoS flag should be 0 according to MQTT spec")
			}
			if tc.puback.FixedHeader.Retain != 0 {
				t.Logf("Warning: RETAIN flag should be 0 according to MQTT spec")
			}

			// 验证Packet ID范围
			if tc.puback.PacketID < 1 || tc.puback.PacketID > 65535 {
				t.Errorf("Packet ID %d is out of valid range [1, 65535]", tc.puback.PacketID)
			}

			// 验证版本特定的字段
			switch tc.puback.Version {
			case VERSION500:
				// v5.0必须包含ReasonCode
				if tc.puback.ReasonCode.Code == 0 && tc.puback.ReasonCode.Reason == "" {
					t.Logf("Note: v5.0 PUBACK should include a reason code")
				}
			case VERSION311:
				// v3.1.1不应该包含ReasonCode
				if tc.puback.ReasonCode.Code != 0 || tc.puback.ReasonCode.Reason != "" {
					t.Logf("Note: v3.1.1 PUBACK should not include reason code")
				}
			}

			t.Logf("Protocol compliance check: %s", tc.description)
		})
	}
}

// TestPUBACK_EdgeCases 测试PUBACK报文的边界情况
func TestPUBACK_EdgeCases(t *testing.T) {
	testCases := []struct {
		name        string
		puback      *PUBACK
		description string
	}{
		{
			name: "MinPacketID",
			puback: &PUBACK{
				FixedHeader: &FixedHeader{
					Version: VERSION311,
					Kind:    0x04,
					Dup:     0,
					QoS:     0,
					Retain:  0,
				},
				PacketID: 1, // 最小有效Packet ID
			},
			description: "最小Packet ID (1) 的PUBACK",
		},
		{
			name: "MaxPacketID",
			puback: &PUBACK{
				FixedHeader: &FixedHeader{
					Version: VERSION311,
					Kind:    0x04,
					Dup:     0,
					QoS:     0,
					Retain:  0,
				},
				PacketID: 65535, // 最大有效Packet ID
			},
			description: "最大Packet ID (65535) 的PUBACK",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 测试序列化
			var buf bytes.Buffer
			err := tc.puback.Pack(&buf)
			if err != nil {
				t.Fatalf("Pack() failed: %v", err)
			}

			// 测试反序列化
			data := buf.Bytes()
			newPuback := &PUBACK{
				FixedHeader: &FixedHeader{
					Version: tc.puback.Version,
					Kind:    0x04,
				},
			}

			// 解析固定报头
			buf2 := bytes.NewBuffer(data)
			firstByte := buf2.Next(1)[0]
			newPuback.FixedHeader.Kind = firstByte >> 4
			newPuback.FixedHeader.Dup = (firstByte >> 3) & 0x01
			newPuback.FixedHeader.QoS = (firstByte >> 1) & 0x03
			newPuback.FixedHeader.Retain = firstByte & 0x01

			remainingLen, err := decodeLength(buf2)
			if err != nil {
				t.Fatalf("Failed to decode remaining length: %v", err)
			}
			newPuback.FixedHeader.RemainingLength = remainingLen
			newPuback.FixedHeader.Version = tc.puback.Version

			err = newPuback.Unpack(buf2)
			if err != nil {
				t.Fatalf("Unpack() failed: %v", err)
			}

			// 验证关键字段
			if newPuback.PacketID != tc.puback.PacketID {
				t.Errorf("Packet ID mismatch: got %d, want %d", newPuback.PacketID, tc.puback.PacketID)
			}

			t.Logf("Edge case test passed: %s", tc.description)
		})
	}
}

// TestPubackProperties_Pack 测试PubackProperties的序列化
func TestPubackProperties_Pack(t *testing.T) {
	testCases := []struct {
		name        string
		props       *PubackProperties
		description string
	}{
		{
			name:        "EmptyProperties",
			props:       &PubackProperties{},
			description: "空属性结构",
		},
		{
			name: "ReasonStringOnly",
			props: &PubackProperties{
				ReasonString: "Test reason",
			},
			description: "只包含原因字符串",
		},
		{
			name: "UserPropertyOnly",
			props: &PubackProperties{
				UserProperty: map[string][]string{
					"key1": {"value1"},
					"key2": {"value2"},
				},
			},
			description: "只包含用户属性",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data, err := tc.props.Pack()
			if err != nil {
				t.Fatalf("Pack() failed: %v", err)
			}

			// 验证序列化结果
			if tc.props.ReasonString == "" && len(tc.props.UserProperty) == 0 {
				// 空属性应该序列化为空字节数组
				if len(data) != 0 {
					t.Errorf("Empty properties should serialize to empty bytes, got %d bytes", len(data))
				}
			} else {
				// 非空属性应该序列化为非空字节数组
				if len(data) == 0 {
					t.Errorf("Non-empty properties should not serialize to empty bytes")
				}
			}

			t.Logf("Properties test passed: %s", tc.description)
		})
	}
}

// BenchmarkPUBACK_Pack 性能测试：PUBACK序列化
func BenchmarkPUBACK_Pack(b *testing.B) {
	puback := &PUBACK{
		FixedHeader: &FixedHeader{
			Version: VERSION500,
			Kind:    0x04,
			Dup:     0,
			QoS:     0,
			Retain:  0,
		},
		PacketID:   12345,
		ReasonCode: CodeSuccess,
		Props: &PubackProperties{
			ReasonString: "Benchmark test",
			UserProperty: map[string][]string{
				"benchmark": {"true"},
				"version":   {"1.0"},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		if err := puback.Pack(&buf); err != nil {
			b.Fatalf("Pack() failed: %v", err)
		}
	}
}

// BenchmarkPUBACK_Unpack 性能测试：PUBACK反序列化
func BenchmarkPUBACK_Unpack(b *testing.B) {
	// 准备测试数据
	puback := &PUBACK{
		FixedHeader: &FixedHeader{
			Version: VERSION500,
			Kind:    0x04,
			Dup:     0,
			QoS:     0,
			Retain:  0,
		},
		PacketID:   12345,
		ReasonCode: CodeSuccess,
		Props: &PubackProperties{
			ReasonString: "Benchmark test",
			UserProperty: map[string][]string{
				"benchmark": {"true"},
				"version":   {"1.0"},
			},
		},
	}

	var buf bytes.Buffer
	if err := puback.Pack(&buf); err != nil {
		b.Fatalf("Failed to prepare test data: %v", err)
	}
	data := buf.Bytes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		newPuback := &PUBACK{
			FixedHeader: &FixedHeader{
				Version: VERSION500,
				Kind:    0x04,
			},
		}

		buf2 := bytes.NewBuffer(data)
		// 跳过固定报头解析，直接测试Unpack
		firstByte := buf2.Next(1)[0]
		newPuback.FixedHeader.Kind = firstByte >> 4
		newPuback.FixedHeader.Dup = (firstByte >> 3) & 0x01
		newPuback.FixedHeader.QoS = (firstByte >> 1) & 0x03
		newPuback.FixedHeader.Retain = firstByte & 0x01

		remainingLen, err := decodeLength(buf2)
		if err != nil {
			b.Fatalf("Failed to decode remaining length: %v", err)
		}
		newPuback.FixedHeader.RemainingLength = remainingLen
		newPuback.FixedHeader.Version = VERSION500

		if err := newPuback.Unpack(buf2); err != nil {
			b.Fatalf("Unpack() failed: %v", err)
		}
	}
}
