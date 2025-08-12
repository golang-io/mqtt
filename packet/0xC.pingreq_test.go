package packet

import (
	"bytes"
	"testing"
)

// TestPINGREQ_Kind 测试PINGREQ报文的类型标识符
// 参考MQTT v3.1.1章节 3.12 PINGREQ - PING request
// 参考MQTT v5.0章节 3.12 PINGREQ - PING request
func TestPINGREQ_Kind(t *testing.T) {
	pingreq := &PINGREQ{FixedHeader: &FixedHeader{Kind: 0x0C}}
	if pingreq.Kind() != 0x0C {
		t.Errorf("PINGREQ.Kind() = %d, want 0x0C", pingreq.Kind())
	}
}

// TestPINGREQ_Pack 测试PINGREQ报文的序列化
// 参考MQTT v3.1.1章节 3.12.1 PINGREQ Fixed Header
// 参考MQTT v5.0章节 3.12.1 PINGREQ Fixed Header
func TestPINGREQ_Pack(t *testing.T) {
	testCases := []struct {
		name     string
		pingreq  *PINGREQ
		version  byte
		expected []byte
	}{
		{
			name: "V311_BasicPingreq",
			pingreq: &PINGREQ{
				FixedHeader: &FixedHeader{
					Version:         VERSION311,
					Kind:            0x0C,
					Dup:             0,
					QoS:             0,
					Retain:          0,
					RemainingLength: 0, // PINGREQ没有载荷
				},
			},
			version: VERSION311,
			expected: []byte{
				0xC0, 0x00, // 固定报头: PINGREQ, 标志位0, 剩余长度0
			},
		},
		{
			name: "V500_BasicPingreq",
			pingreq: &PINGREQ{
				FixedHeader: &FixedHeader{
					Version:         VERSION500,
					Kind:            0x0C,
					Dup:             0,
					QoS:             0,
					Retain:          0,
					RemainingLength: 0, // PINGREQ没有载荷
				},
			},
			version: VERSION500,
			expected: []byte{
				0xC0, 0x00, // 固定报头: PINGREQ, 标志位0, 剩余长度0
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tc.pingreq.Pack(&buf)
			if err != nil {
				t.Fatalf("Pack() failed: %v", err)
			}

			result := buf.Bytes()

			// 验证基本结构
			if len(result) != 2 {
				t.Errorf("result length = %d, want 2", len(result))
			}

			// 验证报文类型和标志位
			if result[0] != 0xC0 {
				t.Errorf("packet type and flags = %02x, want 0xC0", result[0])
			}

			// 验证剩余长度
			if result[1] != 0x00 {
				t.Errorf("remaining length = %02x, want 0x00", result[1])
			}
		})
	}
}

// TestPINGREQ_Unpack 测试PINGREQ报文的反序列化
func TestPINGREQ_Unpack(t *testing.T) {
	testCases := []struct {
		name     string
		data     []byte
		version  byte
		expected *PINGREQ
	}{
		{
			name: "V311_BasicPingreq",
			data: []byte{
				0xC0, 0x00, // 固定报头: PINGREQ, 标志位0, 剩余长度0
			},
			version: VERSION311,
			expected: &PINGREQ{
				FixedHeader: &FixedHeader{
					Version:         VERSION311,
					Kind:            0x0C,
					Dup:             0,
					QoS:             0,
					Retain:          0,
					RemainingLength: 0,
				},
			},
		},
		{
			name: "V500_BasicPingreq",
			data: []byte{
				0xC0, 0x00, // 固定报头: PINGREQ, 标志位0, 剩余长度0
			},
			version: VERSION500,
			expected: &PINGREQ{
				FixedHeader: &FixedHeader{
					Version:         VERSION500,
					Kind:            0x0C,
					Dup:             0,
					QoS:             0,
					Retain:          0,
					RemainingLength: 0,
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

			pingreq := &PINGREQ{
				FixedHeader: fixedHeader,
			}

			err := pingreq.Unpack(buf)
			if err != nil {
				t.Fatalf("Unpack() failed: %v", err)
			}

			// PINGREQ没有载荷，所以Unpack应该成功且不修改任何字段
			if pingreq.FixedHeader.Kind != tc.expected.FixedHeader.Kind {
				t.Errorf("Kind = %d, want %d", pingreq.FixedHeader.Kind, tc.expected.FixedHeader.Kind)
			}
		})
	}
}

// TestPINGREQ_ProtocolCompliance 测试PINGREQ报文的协议合规性
func TestPINGREQ_ProtocolCompliance(t *testing.T) {
	t.Run("V311_FlagsMustBeZero", func(t *testing.T) {
		// v3.1.1中标志位必须为0
		pingreq := &PINGREQ{
			FixedHeader: &FixedHeader{
				Version: VERSION311,
				Kind:    0x0C,
				Dup:     1, // 违反协议
				QoS:     0,
				Retain:  0,
			},
		}

		var buf bytes.Buffer
		err := pingreq.Pack(&buf)
		if err != nil {
			t.Fatalf("Pack() failed: %v", err)
		}

		// 验证标志位被正确设置
		result := buf.Bytes()
		if result[0] != 0xC0 {
			t.Errorf("flags not properly set: %02x", result[0])
		}
	})

	t.Run("V500_FlagsMustBeZero", func(t *testing.T) {
		// v5.0中标志位也必须为0
		pingreq := &PINGREQ{
			FixedHeader: &FixedHeader{
				Version: VERSION500,
				Kind:    0x0C,
				Dup:     0,
				QoS:     1, // 违反协议
				Retain:  0,
			},
		}

		var buf bytes.Buffer
		err := pingreq.Pack(&buf)
		if err != nil {
			t.Fatalf("Pack() failed: %v", err)
		}

		// 验证标志位被正确设置
		result := buf.Bytes()
		if result[0] != 0xC0 {
			t.Errorf("flags not properly set: %02x", result[0])
		}
	})
}

// TestPINGREQ_EdgeCases 测试PINGREQ报文的边界情况
func TestPINGREQ_EdgeCases(t *testing.T) {
	t.Run("MinimalPacket", func(t *testing.T) {
		// 测试最小的PINGREQ报文
		pingreq := &PINGREQ{
			FixedHeader: &FixedHeader{
				Version: VERSION311,
				Kind:    0x0C,
				Dup:     0,
				QoS:     0,
				Retain:  0,
			},
		}

		var buf bytes.Buffer
		err := pingreq.Pack(&buf)
		if err != nil {
			t.Fatalf("Pack() failed: %v", err)
		}

		result := buf.Bytes()
		if len(result) != 2 {
			t.Errorf("result length = %d, want 2", len(result))
		}
	})

	t.Run("VersionCompatibility", func(t *testing.T) {
		// 测试v3.1.1和v5.0的兼容性
		versions := []byte{VERSION311, VERSION500}

		for _, version := range versions {
			pingreq := &PINGREQ{
				FixedHeader: &FixedHeader{
					Version: version,
					Kind:    0x0C,
					Dup:     0,
					QoS:     0,
					Retain:  0,
				},
			}

			var buf bytes.Buffer
			err := pingreq.Pack(&buf)
			if err != nil {
				t.Errorf("Pack() failed for version %d: %v", version, err)
				continue
			}

			result := buf.Bytes()
			if len(result) != 2 {
				t.Errorf("result length = %d for version %d, want 2", len(result), version)
			}
		}
	})
}

// TestPINGREQ_KeepAlive 测试PINGREQ在保持连接中的作用
func TestPINGREQ_KeepAlive(t *testing.T) {
	t.Run("KeepAliveFlow", func(t *testing.T) {
		// 模拟保持连接流程
		keepAliveInterval := uint16(60) // 60秒

		// 1. CONNECT报文设置保持连接时间
		_ = &CONNECT{
			FixedHeader: &FixedHeader{
				Version: VERSION311,
				Kind:    0x01,
				Dup:     0,
				QoS:     0,
				Retain:  0,
			},
			ClientID:  "testclient",
			KeepAlive: keepAliveInterval,
		}

		// 2. PINGREQ报文用于保持连接
		pingreq := &PINGREQ{
			FixedHeader: &FixedHeader{
				Version: VERSION311,
				Kind:    0x0C,
				Dup:     0,
				QoS:     0,
				Retain:  0,
			},
		}

		// 3. PINGRESP报文作为响应
		pingresp := &PINGRESP{
			FixedHeader: &FixedHeader{
				Version: VERSION311,
				Kind:    0x0D,
				Dup:     0,
				QoS:     0,
				Retain:  0,
			},
		}

		// 验证PINGREQ的简单性
		if pingreq.FixedHeader.RemainingLength != 0 {
			t.Error("PINGREQ should have no payload")
		}

		// 验证PINGREQ和PINGRESP的对应关系
		if pingreq.Kind() != 0x0C {
			t.Errorf("PINGREQ Kind = %d, want 0x0C", pingreq.Kind())
		}
		if pingresp.Kind() != 0x0D {
			t.Errorf("PINGRESP Kind = %d, want 0x0D", pingresp.Kind())
		}
	})
}

// TestPINGREQ_NetworkBehavior 测试PINGREQ的网络行为
func TestPINGREQ_NetworkBehavior(t *testing.T) {
	t.Run("NoResponseTimeout", func(t *testing.T) {
		// 测试没有响应时的超时处理
		pingreq := &PINGREQ{
			FixedHeader: &FixedHeader{
				Version: VERSION311,
				Kind:    0x0C,
				Dup:     0,
				QoS:     0,
				Retain:  0,
			},
		}

		// PINGREQ本身不包含超时信息，超时由应用层处理
		var buf bytes.Buffer
		err := pingreq.Pack(&buf)
		if err != nil {
			t.Fatalf("Pack() failed: %v", err)
		}

		result := buf.Bytes()
		if len(result) != 2 {
			t.Errorf("PINGREQ should be minimal: %d bytes", len(result))
		}
	})

	t.Run("DuplicateHandling", func(t *testing.T) {
		// 测试重复PINGREQ的处理
		pingreq1 := &PINGREQ{
			FixedHeader: &FixedHeader{
				Version: VERSION311,
				Kind:    0x0C,
				Dup:     0,
				QoS:     0,
				Retain:  0,
			},
		}

		pingreq2 := &PINGREQ{
			FixedHeader: &FixedHeader{
				Version: VERSION311,
				Kind:    0x0C,
				Dup:     0,
				QoS:     0,
				Retain:  0,
			},
		}

		// 两个PINGREQ应该完全相同
		var buf1, buf2 bytes.Buffer
		err1 := pingreq1.Pack(&buf1)
		err2 := pingreq2.Pack(&buf2)

		if err1 != nil || err2 != nil {
			t.Fatalf("Pack() failed: %v, %v", err1, err2)
		}

		result1 := buf1.Bytes()
		result2 := buf2.Bytes()

		if !bytes.Equal(result1, result2) {
			t.Error("Identical PINGREQ packets should produce identical results")
		}
	})
}

// TestPINGREQ_ErrorHandling 测试PINGREQ的错误处理
func TestPINGREQ_ErrorHandling(t *testing.T) {
	t.Run("InvalidFlags", func(t *testing.T) {
		// 测试无效标志位的处理
		pingreq := &PINGREQ{
			FixedHeader: &FixedHeader{
				Version: VERSION311,
				Kind:    0x0C,
				Dup:     1, // 无效标志位
				QoS:     1, // 无效标志位
				Retain:  1, // 无效标志位
			},
		}

		var buf bytes.Buffer
		err := pingreq.Pack(&buf)
		if err != nil {
			t.Fatalf("Pack() should handle invalid flags gracefully: %v", err)
		}

		// 验证标志位被正确设置
		result := buf.Bytes()
		if result[0] != 0xC0 {
			t.Errorf("flags should be corrected to 0xC0, got %02x", result[0])
		}
	})

	t.Run("NilFixedHeader", func(t *testing.T) {
		// 测试空固定报头的处理
		pingreq := &PINGREQ{
			FixedHeader: nil,
		}

		var buf bytes.Buffer
		err := pingreq.Pack(&buf)
		if err == nil {
			t.Error("Pack() should fail with nil FixedHeader")
		}
	})
}

// BenchmarkPINGREQ_Pack 性能测试
func BenchmarkPINGREQ_Pack(b *testing.B) {
	pingreq := &PINGREQ{
		FixedHeader: &FixedHeader{
			Version: VERSION500,
			Kind:    0x0C,
			Dup:     0,
			QoS:     0,
			Retain:  0,
		},
	}

	var buf bytes.Buffer
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		pingreq.Pack(&buf)
	}
}

// BenchmarkPINGREQ_Unpack 性能测试
func BenchmarkPINGREQ_Unpack(b *testing.B) {
	data := []byte{
		0xC0, 0x00, // 固定报头
	}

	pingreq := &PINGREQ{
		FixedHeader: &FixedHeader{
			Version:         VERSION500,
			Kind:            0x0C,
			Dup:             0,
			QoS:             0,
			Retain:          0,
			RemainingLength: 0,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf := bytes.NewBuffer(data)
		pingreq.Unpack(buf)
	}
}
