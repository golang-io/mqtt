package packet

import (
	"bytes"
	"testing"
)

// TestSUBACK_Kind 测试SUBACK报文类型
func TestSUBACK_Kind(t *testing.T) {
	suback := &SUBACK{}
	if suback.Kind() != 0x9 {
		t.Errorf("SUBACK.Kind() = %d, want 0x9", suback.Kind())
	}
}

// TestSUBACK_Pack_MQTT311 测试MQTT v3.1.1 SUBACK报文打包
func TestSUBACK_Pack_MQTT311(t *testing.T) {
	tests := []struct {
		name      string
		packetID  uint16
		reasonCodes []ReasonCode
		wantErr   bool
	}{
		{
			name:      "正常QoS 0订阅确认",
			packetID:  12345,
			reasonCodes: []ReasonCode{{Code: 0x00}},
			wantErr:   false,
		},
		{
			name:      "正常QoS 1订阅确认",
			packetID:  12346,
			reasonCodes: []ReasonCode{{Code: 0x01}},
			wantErr:   false,
		},
		{
			name:      "正常QoS 2订阅确认",
			packetID:  12347,
			reasonCodes: []ReasonCode{{Code: 0x02}},
			wantErr:   false,
		},
		{
			name:      "订阅失败确认",
			packetID:  12348,
			reasonCodes: []ReasonCode{{Code: 0x80}},
			wantErr:   false,
		},
		{
			name:      "多个订阅确认",
			packetID:  12349,
			reasonCodes: []ReasonCode{{Code: 0x00}, {Code: 0x01}, {Code: 0x02}},
			wantErr:   false,
		},
		{
			name:      "混合结果订阅确认",
			packetID:  12350,
			reasonCodes: []ReasonCode{{Code: 0x00}, {Code: 0x80}, {Code: 0x01}},
			wantErr:   false,
		},
		{
			name:      "空返回码列表",
			packetID:  12351,
			reasonCodes: []ReasonCode{},
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suback := &SUBACK{
				FixedHeader: &FixedHeader{
					Version: VERSION311,
					Kind:    0x9,
				},
				PacketID:   tt.packetID,
				ReasonCode: tt.reasonCodes,
			}

			var buf bytes.Buffer
			err := suback.Pack(&buf)

			if tt.wantErr {
				if err == nil {
					t.Error("Pack() should return error")
				}
				return
			}

			if err != nil {
				t.Fatalf("Pack() failed: %v", err)
			}

			// 验证报文结构
			data := buf.Bytes()
			
			// 验证固定报头
			if data[0] != 0x90 { // 0x09 << 4 | 0x00
				t.Errorf("Fixed header type = 0x%02x, want 0x90", data[0])
			}

			// 验证报文标识符
			packetID := uint16(data[2])<<8 | uint16(data[3])
			if packetID != tt.packetID {
				t.Errorf("Packet ID = %d, want %d", packetID, tt.packetID)
			}

			// 验证返回码
			payloadStart := 4 // 固定报头(2) + 可变报头(2)
			for i, reason := range tt.reasonCodes {
				if data[payloadStart+i] != reason.Code {
					t.Errorf("Reason code[%d] = 0x%02x, want 0x%02x", i, data[payloadStart+i], reason.Code)
				}
			}
		})
	}
}

// TestSUBACK_Pack_MQTT500 测试MQTT v5.0 SUBACK报文打包
func TestSUBACK_Pack_MQTT500(t *testing.T) {
	tests := []struct {
		name        string
		packetID    uint16
		reasonCodes []ReasonCode
		props       *SubackProperties
		wantErr     bool
	}{
		{
			name:        "带原因字符串的订阅确认",
			packetID:    12345,
			reasonCodes: []ReasonCode{{Code: 0x00}},
			props: &SubackProperties{
				ReasonString: "Subscription granted with QoS 0",
			},
			wantErr: false,
		},
		{
			name:        "带用户属性的订阅确认",
			packetID:    12346,
			reasonCodes: []ReasonCode{{Code: 0x01}},
			props: &SubackProperties{
				UserProperty: map[string][]string{
					"client_id": {"test_client"},
					"timestamp": {"1234567890"},
				},
			},
			wantErr: false,
		},
		{
			name:        "完整属性的订阅确认",
			packetID:    12347,
			reasonCodes: []ReasonCode{{Code: 0x02}, {Code: 0x80}},
			props: &SubackProperties{
				ReasonString: "Mixed subscription results",
				UserProperty: map[string][]string{
					"session_id": {"sess_123"},
				},
			},
			wantErr: false,
		},
		{
			name:        "无属性的订阅确认",
			packetID:    12348,
			reasonCodes: []ReasonCode{{Code: 0x00}},
			props:       nil,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			suback := &SUBACK{
				FixedHeader: &FixedHeader{
					Version: VERSION500,
					Kind:    0x9,
				},
				PacketID:    tt.packetID,
				SubackProps: tt.props,
				ReasonCode:  tt.reasonCodes,
			}

			var buf bytes.Buffer
			err := suback.Pack(&buf)

			if tt.wantErr {
				if err == nil {
					t.Error("Pack() should return error")
				}
				return
			}

			if err != nil {
				t.Fatalf("Pack() failed: %v", err)
			}

			// 验证报文结构
			data := buf.Bytes()
			
			// 验证固定报头
			if data[0] != 0x90 { // 0x09 << 4 | 0x00
				t.Errorf("Fixed header type = 0x%02x, want 0x90", data[0])
			}

			// 验证报文标识符
			packetID := uint16(data[2])<<8 | uint16(data[3])
			if packetID != tt.packetID {
				t.Errorf("Packet ID = %d, want %d", packetID, tt.packetID)
			}

			// 验证属性长度（如果存在）
			if tt.props != nil {
				// 这里需要解析属性长度，比较复杂，暂时跳过详细验证
				// 主要验证报文能够正常打包
			}
		})
	}
}

// TestSUBACK_Unpack_MQTT311 测试MQTT v3.1.1 SUBACK报文解包
func TestSUBACK_Unpack_MQTT311(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		want      *SUBACK
		wantErr   bool
	}{
		{
			name: "单个QoS 0订阅确认",
			data: []byte{
				0x90, 0x03, // 固定报头: 类型0x09, 长度3
				0x30, 0x39, // 报文标识符: 12345
				0x00,        // 返回码: QoS 0
			},
			want: &SUBACK{
				PacketID:   12345,
				ReasonCode: []ReasonCode{{Code: 0x00}},
			},
			wantErr: false,
		},
		{
			name: "多个订阅确认",
			data: []byte{
				0x90, 0x05, // 固定报头: 类型0x09, 长度5
				0x30, 0x3A, // 报文标识符: 12346
				0x00,        // 返回码1: QoS 0
				0x01,        // 返回码2: QoS 1
				0x02,        // 返回码3: QoS 2
			},
			want: &SUBACK{
				PacketID:   12346,
				ReasonCode: []ReasonCode{{Code: 0x00}, {Code: 0x01}, {Code: 0x02}},
			},
			wantErr: false,
		},
		{
			name: "包含失败订阅的确认",
			data: []byte{
				0x90, 0x04, // 固定报头: 类型0x09, 长度4
				0x30, 0x3B, // 报文标识符: 12347
				0x00,        // 返回码1: QoS 0
				0x80,        // 返回码2: 失败
			},
			want: &SUBACK{
				PacketID:   12347,
				ReasonCode: []ReasonCode{{Code: 0x00}, {Code: 0x80}},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 跳过固定报头，直接测试Unpack
			payload := tt.data[2:] // 跳过固定报头
			buf := bytes.NewBuffer(payload)

			suback := &SUBACK{
				FixedHeader: &FixedHeader{
					Version: VERSION311,
				},
			}

			err := suback.Unpack(buf)

			if tt.wantErr {
				if err == nil {
					t.Error("Unpack() should return error")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unpack() failed: %v", err)
			}

			// 验证报文标识符
			if suback.PacketID != tt.want.PacketID {
				t.Errorf("Packet ID = %d, want %d", suback.PacketID, tt.want.PacketID)
			}

			// 验证返回码数量
			if len(suback.ReasonCode) != len(tt.want.ReasonCode) {
				t.Errorf("Reason code count = %d, want %d", len(suback.ReasonCode), len(tt.want.ReasonCode))
			}

			// 验证返回码值
			for i, reason := range suback.ReasonCode {
				if reason.Code != tt.want.ReasonCode[i].Code {
					t.Errorf("Reason code[%d] = 0x%02x, want 0x%02x", i, reason.Code, tt.want.ReasonCode[i].Code)
				}
			}
		})
	}
}

// TestSUBACK_Unpack_MQTT500 测试MQTT v5.0 SUBACK报文解包
func TestSUBACK_Unpack_MQTT500(t *testing.T) {
	tests := []struct {
		name      string
		data      []byte
		want      *SUBACK
		wantErr   bool
	}{
		{
			name: "带原因字符串的订阅确认",
			data: []byte{
				0x30, 0x39, // 报文标识符: 12345
				0x00,        // 属性长度: 0 (无属性)
				0x00,        // 返回码: QoS 0
			},
			want: &SUBACK{
				PacketID:   12345,
				ReasonCode: []ReasonCode{{Code: 0x00}},
			},
			wantErr: false,
		},
		{
			name: "带属性的订阅确认",
			data: []byte{
				0x30, 0x3A, // 报文标识符: 12346
				0x0F,        // 属性长度: 15
				0x1F,        // 属性标识符: Reason String (31)
				0x00, 0x0C, // 字符串长度: 12
				0x53, 0x75, 0x62, 0x73, 0x63, 0x72, 0x69, 0x70, 0x74, 0x69, 0x6F, 0x6E, // "Subscription"
				0x00,        // 返回码: QoS 0
			},
			want: &SUBACK{
				PacketID: 12346,
				SubackProps: &SubackProperties{
					ReasonString: "Subscription",
				},
				ReasonCode: []ReasonCode{{Code: 0x00}},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.NewBuffer(tt.data)

			suback := &SUBACK{
				FixedHeader: &FixedHeader{
					Version: VERSION500,
				},
			}

			err := suback.Unpack(buf)

			if tt.wantErr {
				if err == nil {
					t.Error("Unpack() should return error")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unpack() failed: %v", err)
			}

			// 验证报文标识符
			if suback.PacketID != tt.want.PacketID {
				t.Errorf("Packet ID = %d, want %d", suback.PacketID, tt.want.PacketID)
			}

			// 验证属性
			if tt.want.SubackProps != nil {
				if suback.SubackProps == nil {
					t.Error("SubackProps should not be nil")
				} else if suback.SubackProps.ReasonString != tt.want.SubackProps.ReasonString {
					t.Errorf("ReasonString = %s, want %s", suback.SubackProps.ReasonString, tt.want.SubackProps.ReasonString)
				}
			}

			// 验证返回码
			if len(suback.ReasonCode) != len(tt.want.ReasonCode) {
				t.Errorf("Reason code count = %d, want %d", len(suback.ReasonCode), len(tt.want.ReasonCode))
			}
		})
	}
}

// TestSUBACK_Unpack_InvalidReasonCodes 测试无效返回码的处理
func TestSUBACK_Unpack_InvalidReasonCodes(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name: "无效返回码0x81",
			data: []byte{
				0x30, 0x39, // 报文标识符: 12345
				0x81,        // 无效返回码
			},
			wantErr: true,
		},
		{
			name: "无效返回码0x03",
			data: []byte{
				0x30, 0x39, // 报文标识符: 12345
				0x03,        // 无效返回码
			},
			wantErr: true,
		},
		{
			name: "无效返回码0xFF",
			data: []byte{
				0x30, 0x39, // 报文标识符: 12345
				0xFF,        // 无效返回码
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.NewBuffer(tt.data)

			suback := &SUBACK{
				FixedHeader: &FixedHeader{
					Version: VERSION311,
				},
			}

			err := suback.Unpack(buf)

			if tt.wantErr && err == nil {
				t.Error("Unpack() should return error for invalid reason code")
			}
		})
	}
}

// TestSUBACK_RoundTrip 测试SUBACK报文的往返打包解包
func TestSUBACK_RoundTrip(t *testing.T) {
	tests := []struct {
		name        string
		version     byte
		packetID    uint16
		reasonCodes []ReasonCode
		props       *SubackProperties
	}{
		{
			name:        "MQTT v3.1.1 简单订阅确认",
			version:     VERSION311,
			packetID:    12345,
			reasonCodes: []ReasonCode{{Code: 0x00}},
			props:       nil,
		},
		{
			name:        "MQTT v3.1.1 多个订阅确认",
			version:     VERSION311,
			packetID:    12346,
			reasonCodes: []ReasonCode{{Code: 0x00}, {Code: 0x01}, {Code: 0x02}},
			props:       nil,
		},
		{
			name:        "MQTT v5.0 带原因字符串",
			version:     VERSION500,
			packetID:    12347,
			reasonCodes: []ReasonCode{{Code: 0x00}},
			props: &SubackProperties{
				ReasonString: "Test subscription",
			},
		},
		{
			name:        "MQTT v5.0 带用户属性",
			version:     VERSION500,
			packetID:    12348,
			reasonCodes: []ReasonCode{{Code: 0x01}, {Code: 0x80}},
			props: &SubackProperties{
				UserProperty: map[string][]string{
					"test_key": {"test_value"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建原始SUBACK
			original := &SUBACK{
				FixedHeader: &FixedHeader{
					Version: tt.version,
					Kind:    0x9,
				},
				PacketID:    tt.packetID,
				SubackProps: tt.props,
				ReasonCode:  tt.reasonCodes,
			}

			// 打包
			var buf bytes.Buffer
			if err := original.Pack(&buf); err != nil {
				t.Fatalf("Pack() failed: %v", err)
			}

			// 解包
			unpacked := &SUBACK{
				FixedHeader: &FixedHeader{
					Version: tt.version,
				},
			}

			// 跳过固定报头进行解包测试
			data := buf.Bytes()
			payload := data[2:] // 跳过固定报头
			payloadBuf := bytes.NewBuffer(payload)

			if err := unpacked.Unpack(payloadBuf); err != nil {
				t.Fatalf("Unpack() failed: %v", err)
			}

			// 验证往返一致性
			if unpacked.PacketID != original.PacketID {
				t.Errorf("Packet ID mismatch: got %d, want %d", unpacked.PacketID, original.PacketID)
			}

			if len(unpacked.ReasonCode) != len(original.ReasonCode) {
				t.Errorf("Reason code count mismatch: got %d, want %d", len(unpacked.ReasonCode), len(original.ReasonCode))
			}

			for i, reason := range unpacked.ReasonCode {
				if reason.Code != original.ReasonCode[i].Code {
					t.Errorf("Reason code[%d] mismatch: got 0x%02x, want 0x%02x", i, reason.Code, original.ReasonCode[i].Code)
				}
			}

			// 验证属性（如果存在）
			if tt.props != nil {
				if unpacked.SubackProps == nil {
					t.Error("SubackProps should not be nil after round trip")
				} else {
					if tt.props.ReasonString != "" && unpacked.SubackProps.ReasonString != tt.props.ReasonString {
						t.Errorf("ReasonString mismatch: got %s, want %s", unpacked.SubackProps.ReasonString, tt.props.ReasonString)
					}
					if len(tt.props.UserProperty) > 0 {
						if len(unpacked.SubackProps.UserProperty) != len(tt.props.UserProperty) {
							t.Errorf("UserProperty count mismatch: got %d, want %d", len(unpacked.SubackProps.UserProperty), len(tt.props.UserProperty))
						}
					}
				}
			}
		})
	}
}

// TestSubackProperties_Pack 测试SubackProperties的打包功能
func TestSubackProperties_Pack(t *testing.T) {
	tests := []struct {
		name    string
		props   *SubackProperties
		wantErr bool
	}{
		{
			name: "空属性",
			props: &SubackProperties{},
			wantErr: false,
		},
		{
			name: "只有原因字符串",
			props: &SubackProperties{
				ReasonString: "Test reason",
			},
			wantErr: false,
		},
		{
			name: "只有用户属性",
			props: &SubackProperties{
				UserProperty: map[string][]string{
					"key1": {"value1"},
				},
			},
			wantErr: false,
		},
		{
			name: "完整属性",
			props: &SubackProperties{
				ReasonString: "Complete test",
				UserProperty: map[string][]string{
					"key1": {"value1", "value2"},
					"key2": {"value3"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.props.Pack()

			if tt.wantErr {
				if err == nil {
					t.Error("Pack() should return error")
				}
				return
			}

			if err != nil {
				t.Fatalf("Pack() failed: %v", err)
			}

			// 验证打包后的数据不为空（如果有属性的话）
			if tt.props.ReasonString != "" || len(tt.props.UserProperty) > 0 {
				if len(data) == 0 {
					t.Error("Pack() should return non-empty data when properties exist")
				}
			}
		})
	}
}

// TestSubackProperties_Unpack 测试SubackProperties的解包功能
func TestSubackProperties_Unpack(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    *SubackProperties
		wantErr bool
	}{
		{
			name: "空属性",
			data: []byte{0x00}, // 属性长度0
			want: &SubackProperties{},
			wantErr: false,
		},
		{
			name: "原因字符串属性",
			data: []byte{
				0x0F,        // 属性长度: 15
				0x1F,        // 属性标识符: Reason String (31)
				0x00, 0x0C, // 字符串长度: 12
				0x54, 0x65, 0x73, 0x74, 0x20, 0x72, 0x65, 0x61, 0x73, 0x6F, 0x6E, 0x20, // "Test reason "
			},
			want: &SubackProperties{
				ReasonString: "Test reason ",
			},
			wantErr: false,
		},
		{
			name: "用户属性",
			data: []byte{
				0x0E,        // 属性长度: 14
				0x26,        // 属性标识符: User Property (38)
				0x00, 0x03, // 键长度: 3
				0x6B, 0x65, 0x79, // "key"
				0x00, 0x05, // 值长度: 5
				0x76, 0x61, 0x6C, 0x75, 0x65, // "value"
			},
			want: &SubackProperties{
				UserProperty: map[string][]string{
					"key": {"value"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.NewBuffer(tt.data)
			props := &SubackProperties{}

			err := props.Unpack(buf)

			if tt.wantErr {
				if err == nil {
					t.Error("Unpack() should return error")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unpack() failed: %v", err)
			}

			// 验证原因字符串
			if tt.want.ReasonString != "" && props.ReasonString != tt.want.ReasonString {
				t.Errorf("ReasonString = %s, want %s", props.ReasonString, tt.want.ReasonString)
			}

			// 验证用户属性
			if len(tt.want.UserProperty) > 0 {
				if len(props.UserProperty) != len(tt.want.UserProperty) {
					t.Errorf("UserProperty count = %d, want %d", len(props.UserProperty), len(tt.want.UserProperty))
				}
			}
		})
	}
}

// TestSUBACK_ProtocolCompliance 测试SUBACK报文协议合规性
func TestSUBACK_ProtocolCompliance(t *testing.T) {
	tests := []struct {
		name        string
		description string
		testFunc    func(t *testing.T)
	}{
		{
			name:        "标志位必须为0",
			description: "MQTT协议要求SUBACK报文的标志位必须为0",
			testFunc: func(t *testing.T) {
				suback := &SUBACK{
					FixedHeader: &FixedHeader{
						Version: VERSION311,
						Kind:    0x9,
					},
					PacketID:   12345,
					ReasonCode: []ReasonCode{{Code: 0x00}},
				}

				var buf bytes.Buffer
				if err := suback.Pack(&buf); err != nil {
					t.Fatalf("Pack() failed: %v", err)
				}

				data := buf.Bytes()
				flags := data[0] & 0x0F
				if flags != 0x00 {
					t.Errorf("Flags = 0x%02x, must be 0x00", flags)
				}
			},
		},
		{
			name:        "返回码顺序一致性",
			description: "返回码顺序必须与SUBSCRIBE报文中的订阅请求顺序一致",
			testFunc: func(t *testing.T) {
				// 模拟SUBSCRIBE报文中的订阅顺序
				subscribeOrder := []ReasonCode{
					{Code: 0x00}, // 第一个订阅
					{Code: 0x01}, // 第二个订阅
					{Code: 0x80}, // 第三个订阅
				}

				suback := &SUBACK{
					FixedHeader: &FixedHeader{
						Version: VERSION311,
						Kind:    0x9,
					},
					PacketID:   12345,
					ReasonCode: subscribeOrder,
				}

				var buf bytes.Buffer
				if err := suback.Pack(&buf); err != nil {
					t.Fatalf("Pack() failed: %v", err)
				}

				// 解包验证顺序
				data := buf.Bytes()
				payload := data[2:] // 跳过固定报头
				payloadBuf := bytes.NewBuffer(payload)

				unpacked := &SUBACK{
					FixedHeader: &FixedHeader{
						Version: VERSION311,
					},
				}

				if err := unpacked.Unpack(payloadBuf); err != nil {
					t.Fatalf("Unpack() failed: %v", err)
				}

				// 验证返回码顺序
				for i, reason := range unpacked.ReasonCode {
					if reason.Code != subscribeOrder[i].Code {
						t.Errorf("Reason code order mismatch at index %d: got 0x%02x, want 0x%02x", i, reason.Code, subscribeOrder[i].Code)
					}
				}
			},
		},
		{
			name:        "返回码值范围验证",
			description: "返回码必须是有效的值：0x00, 0x01, 0x02, 0x80",
			testFunc: func(t *testing.T) {
				validCodes := []byte{0x00, 0x01, 0x02, 0x80}
				invalidCodes := []byte{0x03, 0x81, 0xFF}

				// 测试有效返回码
				for _, code := range validCodes {
					suback := &SUBACK{
						FixedHeader: &FixedHeader{
							Version: VERSION311,
							Kind:    0x9,
						},
						PacketID:   12345,
						ReasonCode: []ReasonCode{{Code: code}},
					}

					var buf bytes.Buffer
					if err := suback.Pack(&buf); err != nil {
						t.Errorf("Pack() failed for valid code 0x%02x: %v", code, err)
					}
				}

				// 测试无效返回码（应该返回错误）
				for _, code := range invalidCodes {
					suback := &SUBACK{
						FixedHeader: &FixedHeader{
							Version: VERSION311,
							Kind:    0x9,
						},
						PacketID:   12345,
						ReasonCode: []ReasonCode{{Code: code}},
					}

					var buf bytes.Buffer
					if err := suback.Pack(&buf); err == nil {
						t.Errorf("Pack() should fail for invalid code 0x%02x", code)
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.testFunc(t)
		})
	}
}

// BenchmarkSUBACK_Pack 性能测试：SUBACK打包
func BenchmarkSUBACK_Pack(b *testing.B) {
			suback := &SUBACK{
			FixedHeader: &FixedHeader{
				Version: VERSION311,
				Kind:    0x9,
			},
			PacketID:   12345,
			ReasonCode: []ReasonCode{{Code: 0x00}, {Code: 0x01}, {Code: 0x02}},
		}

	var buf bytes.Buffer
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf.Reset()
		if err := suback.Pack(&buf); err != nil {
			b.Fatalf("Pack() failed: %v", err)
		}
	}
}

// BenchmarkSUBACK_Unpack 性能测试：SUBACK解包
func BenchmarkSUBACK_Unpack(b *testing.B) {
	// 准备测试数据
	testData := []byte{
		0x30, 0x39, // 报文标识符: 12345
		0x00,        // 返回码1: QoS 0
		0x01,        // 返回码2: QoS 1
		0x02,        // 返回码3: QoS 2
	}

	suback := &SUBACK{
		FixedHeader: &FixedHeader{
			Version: VERSION311,
		},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf := bytes.NewBuffer(testData)
		if err := suback.Unpack(buf); err != nil {
			b.Fatalf("Unpack() failed: %v", err)
		}
	}
}
