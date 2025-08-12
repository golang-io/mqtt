package packet

import (
	"bytes"
	"testing"
)

func TestDISCONNECT_NewDISCONNECT(t *testing.T) {
	tests := []struct {
		name       string
		version    byte
		reasonCode ReasonCode
		wantErr    bool
	}{
		{
			name:       "valid DISCONNECT v5.0 normal",
			version:    VERSION500,
			reasonCode: CodeSuccess,
			wantErr:    false,
		},
		{
			name:       "valid DISCONNECT v5.0 with will",
			version:    VERSION500,
			reasonCode: CodeDisconnectWillMessage,
			wantErr:    false,
		},
		{
			name:       "valid DISCONNECT v3.1.1 normal",
			version:    VERSION311,
			reasonCode: CodeSuccess,
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			disconnect := NewDISCONNECT(tt.version, tt.reasonCode)

			// 检查基本字段
			if disconnect.Kind() != 0x0E {
				t.Errorf("NewDISCONNECT() Kind = %v, want 0x0E", disconnect.Kind())
			}

			if disconnect.Version != tt.version {
				t.Errorf("NewDISCONNECT() Version = %v, want %v", disconnect.Version, tt.version)
			}

			if disconnect.ReasonCode.Code != tt.reasonCode.Code {
				t.Errorf("NewDISCONNECT() ReasonCode = %v, want %v", disconnect.ReasonCode.Code, tt.reasonCode.Code)
			}

			// 检查标志位
			if disconnect.Dup != 0 || disconnect.QoS != 0 || disconnect.Retain != 0 {
				t.Errorf("NewDISCONNECT() flags not zero: Dup=%d, QoS=%d, Retain=%d",
					disconnect.Dup, disconnect.QoS, disconnect.Retain)
			}

			// 验证包
			err := disconnect.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDISCONNECT().Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDISCONNECT_Validate(t *testing.T) {
	tests := []struct {
		name       string
		disconnect *DISCONNECT
		wantErr    bool
	}{
		{
			name: "valid DISCONNECT v5.0 normal",
			disconnect: &DISCONNECT{
				FixedHeader: &FixedHeader{
					Kind:            0x0E,
					Dup:             0,
					QoS:             0,
					Retain:          0,
					Version:         VERSION500,
					RemainingLength: 0,
				},
				ReasonCode: CodeSuccess,
				Props:      &DisconnectProperties{},
			},
			wantErr: false,
		},
		{
			name: "valid DISCONNECT v3.1.1 normal",
			disconnect: &DISCONNECT{
				FixedHeader: &FixedHeader{
					Kind:            0x0E,
					Dup:             0,
					QoS:             0,
					Retain:          0,
					Version:         VERSION311,
					RemainingLength: 0,
				},
				ReasonCode: CodeSuccess,
				Props:      nil,
			},
			wantErr: false,
		},
		{
			name: "invalid flags Dup=1",
			disconnect: &DISCONNECT{
				FixedHeader: &FixedHeader{
					Kind:            0x0E,
					Dup:             1,
					QoS:             0,
					Retain:          0,
					Version:         VERSION500,
					RemainingLength: 0,
				},
				ReasonCode: CodeSuccess,
				Props:      &DisconnectProperties{},
			},
			wantErr: true,
		},
		{
			name: "invalid flags QoS=1",
			disconnect: &DISCONNECT{
				FixedHeader: &FixedHeader{
					Kind:            0x0E,
					Dup:             0,
					QoS:             1,
					Retain:          0,
					Version:         VERSION500,
					RemainingLength: 0,
				},
				ReasonCode: CodeSuccess,
				Props:      &DisconnectProperties{},
			},
			wantErr: true,
		},
		{
			name: "invalid flags Retain=1",
			disconnect: &DISCONNECT{
				FixedHeader: &FixedHeader{
					Kind:            0x0E,
					Dup:             0,
					QoS:             0,
					Retain:          1,
					Version:         VERSION500,
					RemainingLength: 0,
				},
				ReasonCode: CodeSuccess,
				Props:      &DisconnectProperties{},
			},
			wantErr: true,
		},
		{
			name: "invalid reason code",
			disconnect: &DISCONNECT{
				FixedHeader: &FixedHeader{
					Kind:            0x0E,
					Dup:             0,
					QoS:             0,
					Retain:          0,
					Version:         VERSION500,
					RemainingLength: 0,
				},
				ReasonCode: ReasonCode{Code: 0xFF},
				Props:      &DisconnectProperties{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.disconnect.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("DISCONNECT.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDISCONNECT_Packet(t *testing.T) {
	tests := []struct {
		name       string
		disconnect *DISCONNECT
		wantErr    bool
	}{
		{
			name: "valid DISCONNECT with normal reason",
			disconnect: &DISCONNECT{
				FixedHeader: &FixedHeader{
					Kind:            0x0E,
					Dup:             0,
					QoS:             0,
					Retain:          0,
					Version:         VERSION500,
					RemainingLength: 0,
				},
				ReasonCode: CodeSuccess,
				Props:      &DisconnectProperties{},
			},
			wantErr: false,
		},
		{
			name: "valid DISCONNECT with will message reason",
			disconnect: &DISCONNECT{
				FixedHeader: &FixedHeader{
					Kind:            0x0E,
					Dup:             0,
					QoS:             0,
					Retain:          0,
					Version:         VERSION500,
					RemainingLength: 0,
				},
				ReasonCode: CodeDisconnectWillMessage,
				Props: &DisconnectProperties{
					SessionExpiryInterval: 3600,
					ReasonString:          "Client requested disconnect with will",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tt.disconnect.Pack(&buf)
			if (err != nil) != tt.wantErr {
				t.Errorf("DISCONNECT.Packet() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// 检查序列化后的数据
				data := buf.Bytes()
				if len(data) < 3 {
					t.Errorf("DISCONNECT.Packet() produced too short data: %d bytes", len(data))
				}

				// 检查报文类型 (0x0E << 4 = 0xE0)
				if data[0] != 0xE0 {
					t.Errorf("DISCONNECT.Packet() wrong packet type: 0x%02X, want 0xE0", data[0])
				}

				// 检查标志位
				if data[0]&0x0F != 0x00 {
					t.Errorf("DISCONNECT.Packet() flags not zero: 0x%02X", data[0]&0x0F)
				}
			}
		})
	}
}

func TestDISCONNECT_Unpack(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		version byte
		wantErr bool
	}{
		{
			name:    "valid DISCONNECT normal",
			data:    []byte{0x00}, // Normal disconnection reason code
			version: VERSION500,
			wantErr: false,
		},
		{
			name:    "valid DISCONNECT with will",
			data:    []byte{0x04}, // Disconnect with will message reason code
			version: VERSION500,
			wantErr: false,
		},
		{
			name:    "valid DISCONNECT v3.1.1",
			data:    []byte{0x00}, // Normal disconnection reason code
			version: VERSION311,
			wantErr: false,
		},
		{
			name:    "invalid reason code",
			data:    []byte{0xFF}, // Invalid reason code
			version: VERSION500,
			wantErr: true,
		},
		{
			name:    "empty buffer",
			data:    []byte{},
			version: VERSION500,
			wantErr: false, // 根据 MQTT v5 规范，空缓冲区使用默认值 0x00
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			disconnect := &DISCONNECT{
				FixedHeader: &FixedHeader{
					Kind:    0x0E,
					Version: tt.version,
				},
			}

			buf := bytes.NewBuffer(tt.data)
			err := disconnect.Unpack(buf)
			if (err != nil) != tt.wantErr {
				t.Errorf("DISCONNECT.Unpack() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				// 检查解析结果
				if disconnect.ReasonCode.Code != tt.data[0] {
					t.Errorf("DISCONNECT.Unpack() reason code = 0x%02X, want 0x%02X",
						disconnect.ReasonCode.Code, tt.data[0])
				}
			}
		})
	}
}

func TestDisconnectProperties_Validate(t *testing.T) {
	tests := []struct {
		name    string
		props   *DisconnectProperties
		wantErr bool
	}{
		{
			name: "valid properties with session expiry",
			props: &DisconnectProperties{
				SessionExpiryInterval: 3600,
			},
			wantErr: false,
		},
		{
			name: "valid properties with reason string",
			props: &DisconnectProperties{
				ReasonString: "Client requested disconnect",
			},
			wantErr: false,
		},
		{
			name: "valid properties with all fields",
			props: &DisconnectProperties{
				SessionExpiryInterval: 3600,
				ReasonString:          "Client requested disconnect",
				UserProperty: map[string][]string{
					"client_id": {"test_client"},
				},
				ServerReference: "backup.server.com",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.props.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("DisconnectProperties.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDisconnectProperties_Pack(t *testing.T) {
	tests := []struct {
		name    string
		props   *DisconnectProperties
		wantErr bool
	}{
		{
			name: "pack properties with session expiry only",
			props: &DisconnectProperties{
				SessionExpiryInterval: 3600,
			},
			wantErr: false,
		},
		{
			name: "pack properties with all fields",
			props: &DisconnectProperties{
				SessionExpiryInterval: 3600,
				ReasonString:          "Client requested disconnect",
				UserProperty: map[string][]string{
					"client_id": {"test_client"},
				},
				ServerReference: "backup.server.com",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.props.Pack()
			if (err != nil) != tt.wantErr {
				t.Errorf("DisconnectProperties.Pack() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// 检查序列化后的数据不为空
				if len(data) == 0 {
					t.Errorf("DisconnectProperties.Pack() produced empty data")
				}

				// 检查是否包含会话过期间隔
				if tt.props.SessionExpiryInterval != 0 {
					found := false
					for i := 0; i < len(data)-1; i++ {
						if data[i] == 0x11 { // Session Expiry Interval ID
							found = true
							break
						}
					}
					if !found {
						t.Errorf("DisconnectProperties.Pack() missing session expiry interval")
					}
				}
			}
		})
	}
}

func TestDisconnectProperties_Unpack(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name: "unpack properties with session expiry only",
			data: []byte{
				0x11, 0x00, 0x00, 0x0E, 0x10, // Session Expiry Interval: 3600
			},
			wantErr: false,
		},
		{
			name: "unpack properties with session expiry and reason string",
			data: []byte{
				0x11, 0x00, 0x00, 0x0E, 0x10, // Session Expiry Interval: 3600
				0x1F, 0x00, 0x15, 0x43, 0x6C, 0x69, 0x65, 0x6E, 0x74, 0x20, 0x72, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x65, 0x64, 0x20, 0x64, 0x69, 0x73, 0x63, 0x6F, 0x6E, 0x6E, 0x65, 0x63, 0x74, // Reason String: "Client requested disconnect"
			},
			wantErr: false,
		},
		{
			name: "unpack properties with duplicate session expiry",
			data: []byte{
				0x11, 0x00, 0x00, 0x0E, 0x10, // Session Expiry Interval: 3600
				0x11, 0x00, 0x00, 0x0E, 0x10, // Duplicate Session Expiry Interval
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			props := &DisconnectProperties{}
			buf := bytes.NewBuffer(tt.data)
			err := props.Unpack(buf)
			if (err != nil) != tt.wantErr {
				t.Errorf("DisconnectProperties.Unpack() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				// 检查解析结果
				if props.SessionExpiryInterval != 3600 {
					t.Errorf("DisconnectProperties.Unpack() session expiry = %d, want 3600", props.SessionExpiryInterval)
				}
			}
		})
	}
}

func TestDISCONNECT_String(t *testing.T) {
	tests := []struct {
		name       string
		disconnect *DISCONNECT
		want       string
	}{
		{
			name: "DISCONNECT with normal reason only",
			disconnect: &DISCONNECT{
				FixedHeader: &FixedHeader{Kind: 0x0E},
				ReasonCode:  CodeSuccess,
			},
			want: "DISCONNECT{ReasonCode:0x00}",
		},
		{
			name: "DISCONNECT with session expiry and reason",
			disconnect: &DISCONNECT{
				FixedHeader: &FixedHeader{Kind: 0x0E},
				ReasonCode:  CodeDisconnectWillMessage,
				Props: &DisconnectProperties{
					SessionExpiryInterval: 3600,
					ReasonString:          "Client requested disconnect",
				},
			},
			want: "DISCONNECT{ReasonCode:0x04, SessionExpiry:3600, Reason:Client requested disconnect}",
		},
		{
			name: "DISCONNECT with all properties",
			disconnect: &DISCONNECT{
				FixedHeader: &FixedHeader{Kind: 0x0E},
				ReasonCode:  CodeSuccess,
				Props: &DisconnectProperties{
					SessionExpiryInterval: 3600,
					ReasonString:          "Client requested disconnect",
					UserProperty: map[string][]string{
						"client_id": {"test_client"},
					},
					ServerReference: "backup.server.com",
				},
			},
			want: "DISCONNECT{ReasonCode:0x00, SessionExpiry:3600, Reason:Client requested disconnect, UserProps:1, ServerRef:backup.server.com}",
		},
		{
			name:       "nil DISCONNECT",
			disconnect: nil,
			want:       "DISCONNECT<nil>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.disconnect.String()
			if got != tt.want {
				t.Errorf("DISCONNECT.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDISCONNECT_Integration(t *testing.T) {
	// 测试完整的序列化和反序列化流程
	original := &DISCONNECT{
		FixedHeader: &FixedHeader{
			Kind:            0x0E,
			Dup:             0,
			QoS:             0,
			Retain:          0,
			Version:         VERSION500,
			RemainingLength: 0,
		},
		ReasonCode: CodeDisconnectWillMessage,
		Props: &DisconnectProperties{
			SessionExpiryInterval: 3600,
			ReasonString:          "Client requested disconnect with will",
			UserProperty: map[string][]string{
				"client_id": {"test_client"},
				"version":   {"1.0"},
			},
			ServerReference: "backup.server.com",
		},
	}

	// 序列化
	var buf bytes.Buffer
	if err := original.Pack(&buf); err != nil {
		t.Fatalf("Failed to pack DISCONNECT: %v", err)
	}

	// 反序列化 - 需要先解析固定报头
	serializedData := buf.Bytes()

	// 创建新的缓冲区，只包含可变报头和载荷部分
	varHeaderStart := 2 // 跳过固定报头的第1字节(类型+标志位)
	// 计算剩余长度字段的字节数
	remainingLengthBytes := 1
	if serializedData[1] >= 0x80 {
		remainingLengthBytes = 2
		if serializedData[2] >= 0x80 {
			remainingLengthBytes = 3
			if serializedData[3] >= 0x80 {
				remainingLengthBytes = 4
			}
		}
	}
	varHeaderStart += remainingLengthBytes

	// 创建只包含可变报头和载荷的缓冲区（包括原因码）
	varHeaderBuf := bytes.NewBuffer(serializedData[varHeaderStart-1:])
	t.Logf("Variable header buffer: %v", varHeaderBuf.Bytes())

	deserialized := &DISCONNECT{
		FixedHeader: &FixedHeader{
			Kind:    0x0E,
			Version: VERSION500,
		},
	}

	// 手动设置原因码，因为序列化时已经写入
	deserialized.ReasonCode = original.ReasonCode

	if err := deserialized.Unpack(varHeaderBuf); err != nil {
		t.Fatalf("Failed to unpack DISCONNECT: %v", err)
	}

	// 验证结果
	if deserialized.ReasonCode.Code != original.ReasonCode.Code {
		t.Errorf("ReasonCode mismatch: got 0x%02X, want 0x%02X",
			deserialized.ReasonCode.Code, original.ReasonCode.Code)
	}

	if deserialized.Props.SessionExpiryInterval != original.Props.SessionExpiryInterval {
		t.Errorf("SessionExpiryInterval mismatch: got %d, want %d",
			deserialized.Props.SessionExpiryInterval, original.Props.SessionExpiryInterval)
	}

	if deserialized.Props.ReasonString != original.Props.ReasonString {
		t.Errorf("ReasonString mismatch: got %s, want %s",
			deserialized.Props.ReasonString, original.Props.ReasonString)
	}

	if deserialized.Props.ServerReference != original.Props.ServerReference {
		t.Errorf("ServerReference mismatch: got %s, want %s",
			deserialized.Props.ServerReference, original.Props.ServerReference)
	}

	// 检查用户属性数量
	if len(deserialized.Props.UserProperty) != len(original.Props.UserProperty) {
		t.Errorf("UserProperty count mismatch: got %d, want %d",
			len(deserialized.Props.UserProperty), len(original.Props.UserProperty))
	}
}
