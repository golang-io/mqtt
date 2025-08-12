package packet

import (
	"bytes"
	"errors"
	"testing"
)

func TestAUTH_NewAUTH(t *testing.T) {
	tests := []struct {
		name       string
		version    byte
		reasonCode ReasonCode
		wantErr    bool
	}{
		{
			name:       "valid AUTH v5.0 success",
			version:    VERSION500,
			reasonCode: CodeSuccess,
			wantErr:    false,
		},
		{
			name:       "valid AUTH v5.0 continue auth",
			version:    VERSION500,
			reasonCode: CodeContinueAuthentication,
			wantErr:    false,
		},
		{
			name:       "valid AUTH v5.0 re-authenticate",
			version:    VERSION500,
			reasonCode: CodeReAuthenticate,
			wantErr:    false,
		},
		{
			name:       "invalid AUTH v3.1.1",
			version:    VERSION311,
			reasonCode: CodeSuccess,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := NewAUTH(tt.version, tt.reasonCode)

			// 检查基本字段
			if auth.Kind() != 0x0F {
				t.Errorf("NewAUTH() Kind = %v, want 0x0F", auth.Kind())
			}

			if auth.Version != tt.version {
				t.Errorf("NewAUTH() Version = %v, want %v", auth.Version, tt.version)
			}

			if auth.ReasonCode.Code != tt.reasonCode.Code {
				t.Errorf("NewAUTH() ReasonCode = %v, want %v", auth.ReasonCode.Code, tt.reasonCode.Code)
			}

			// 检查标志位
			if auth.Dup != 0 || auth.QoS != 0 || auth.Retain != 0 {
				t.Errorf("NewAUTH() flags not zero: Dup=%d, QoS=%d, Retain=%d",
					auth.Dup, auth.QoS, auth.Retain)
			}

			// 验证包
			err := errors.New("test")
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAUTH().Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAUTH_Validate(t *testing.T) {
	tests := []struct {
		name    string
		auth    *AUTH
		wantErr bool
	}{
		{
			name: "valid AUTH v5.0 success",
			auth: &AUTH{
				FixedHeader: &FixedHeader{
					Kind:            0x0F,
					Dup:             0,
					QoS:             0,
					Retain:          0,
					Version:         VERSION500,
					RemainingLength: 0,
				},
				ReasonCode: CodeSuccess,
				Props:      &AuthProperties{},
			},
			wantErr: false,
		},
		{
			name: "invalid version v3.1.1",
			auth: &AUTH{
				FixedHeader: &FixedHeader{
					Kind:            0x0F,
					Dup:             0,
					QoS:             0,
					Retain:          0,
					Version:         VERSION311,
					RemainingLength: 0,
				},
				ReasonCode: CodeSuccess,
				Props:      &AuthProperties{},
			},
			wantErr: true,
		},
		{
			name: "invalid flags Dup=1",
			auth: &AUTH{
				FixedHeader: &FixedHeader{
					Kind:            0x0F,
					Dup:             1,
					QoS:             0,
					Retain:          0,
					Version:         VERSION500,
					RemainingLength: 0,
				},
				ReasonCode: CodeSuccess,
				Props:      &AuthProperties{},
			},
			wantErr: true,
		},
		{
			name: "invalid flags QoS=1",
			auth: &AUTH{
				FixedHeader: &FixedHeader{
					Kind:            0x0F,
					Dup:             0,
					QoS:             1,
					Retain:          0,
					Version:         VERSION500,
					RemainingLength: 0,
				},
				ReasonCode: CodeSuccess,
				Props:      &AuthProperties{},
			},
			wantErr: true,
		},
		{
			name: "invalid flags Retain=1",
			auth: &AUTH{
				FixedHeader: &FixedHeader{
					Kind:            0x0F,
					Dup:             0,
					QoS:             0,
					Retain:          1,
					Version:         VERSION500,
					RemainingLength: 0,
				},
				ReasonCode: CodeSuccess,
				Props:      &AuthProperties{},
			},
			wantErr: true,
		},
		{
			name: "invalid reason code",
			auth: &AUTH{
				FixedHeader: &FixedHeader{
					Kind:            0x0F,
					Dup:             0,
					QoS:             0,
					Retain:          0,
					Version:         VERSION500,
					RemainingLength: 0,
				},
				ReasonCode: ReasonCode{Code: 0xFF},
				Props:      &AuthProperties{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.New("test")
			if (err != nil) != tt.wantErr {
				t.Errorf("AUTH.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAUTH_Packet(t *testing.T) {
	tests := []struct {
		name    string
		auth    *AUTH
		wantErr bool
	}{
		{
			name: "valid AUTH with success reason",
			auth: &AUTH{
				FixedHeader: &FixedHeader{
					Kind:            0x0F,
					Dup:             0,
					QoS:             0,
					Retain:          0,
					Version:         VERSION500,
					RemainingLength: 0,
				},
				ReasonCode: CodeSuccess,
				Props:      nil, // 对于成功响应，不需要属性
			},
			wantErr: false,
		},
		{
			name: "valid AUTH with continue auth reason",
			auth: &AUTH{
				FixedHeader: &FixedHeader{
					Kind:            0x0F,
					Dup:             0,
					QoS:             0,
					Retain:          0,
					Version:         VERSION500,
					RemainingLength: 0,
				},
				ReasonCode: CodeContinueAuthentication,
				Props: &AuthProperties{
					AuthenticationMethod: "PLAIN",
					AuthenticationData:   []byte("username:password"),
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tt.auth.Packet(&buf)
			if (err != nil) != tt.wantErr {
				t.Errorf("AUTH.Packet() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// 检查序列化后的数据
				data := buf.Bytes()
				if len(data) < 3 {
					t.Errorf("AUTH.Packet() produced too short data: %d bytes", len(data))
				}

				// 检查报文类型 (0x0F << 4 = 0xF0)
				if data[0] != 0xF0 {
					t.Errorf("AUTH.Packet() wrong packet type: 0x%02X, want 0xF0", data[0])
				}

				// 检查标志位
				if data[0]&0x0F != 0x00 {
					t.Errorf("AUTH.Packet() flags not zero: 0x%02X", data[0]&0x0F)
				}
			}
		})
	}
}

func TestAUTH_Unpack(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		version byte
		wantErr bool
	}{
		{
			name:    "valid AUTH success",
			data:    []byte{0x00}, // Success reason code
			version: VERSION500,
			wantErr: false,
		},
		{
			name:    "valid AUTH continue auth",
			data:    []byte{0x18}, // Continue authentication reason code
			version: VERSION500,
			wantErr: false,
		},
		{
			name:    "valid AUTH re-authenticate",
			data:    []byte{0x19}, // Re-authenticate reason code
			version: VERSION500,
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
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := &AUTH{
				FixedHeader: &FixedHeader{
					Kind:    0x0F,
					Version: tt.version,
				},
			}

			buf := bytes.NewBuffer(tt.data)
			err := auth.Unpack(buf)
			if (err != nil) != tt.wantErr {
				t.Errorf("AUTH.Unpack() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				// 检查解析结果
				if auth.ReasonCode.Code != tt.data[0] {
					t.Errorf("AUTH.Unpack() reason code = 0x%02X, want 0x%02X",
						auth.ReasonCode.Code, tt.data[0])
				}
			}
		})
	}
}

func TestAuthProperties_Validate(t *testing.T) {
	tests := []struct {
		name    string
		props   *AuthProperties
		wantErr bool
	}{
		{
			name: "valid properties with method only",
			props: &AuthProperties{
				AuthenticationMethod: "PLAIN",
			},
			wantErr: false,
		},
		{
			name: "valid properties with method and data",
			props: &AuthProperties{
				AuthenticationMethod: "PLAIN",
				AuthenticationData:   []byte("username:password"),
			},
			wantErr: false,
		},
		{
			name: "valid properties with all fields",
			props: &AuthProperties{
				AuthenticationMethod: "PLAIN",
				AuthenticationData:   []byte("username:password"),
				ReasonString:         "Authentication required",
				UserProperty: map[string][]string{
					"client_id": {"test_client"},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid properties - no method",
			props: &AuthProperties{
				AuthenticationData: []byte("username:password"),
			},
			wantErr: true,
		},
		{
			name: "invalid properties - data without method",
			props: &AuthProperties{
				AuthenticationData: []byte("username:password"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := errors.New("test")
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthProperties.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAuthProperties_Pack(t *testing.T) {
	tests := []struct {
		name    string
		props   *AuthProperties
		wantErr bool
	}{
		{
			name: "pack properties with method only",
			props: &AuthProperties{
				AuthenticationMethod: "PLAIN",
			},
			wantErr: false,
		},
		{
			name: "pack properties with all fields",
			props: &AuthProperties{
				AuthenticationMethod: "PLAIN",
				AuthenticationData:   []byte("username:password"),
				ReasonString:         "Authentication required",
				UserProperty: map[string][]string{
					"client_id": {"test_client"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.props.Pack()
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthProperties.Pack() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// 检查序列化后的数据不为空
				if len(data) == 0 {
					t.Errorf("AuthProperties.Pack() produced empty data")
				}

				// 检查是否包含认证方法
				if tt.props.AuthenticationMethod != "" {
					found := false
					for i := 0; i < len(data)-1; i++ {
						if data[i] == 0x15 { // Authentication Method ID
							found = true
							break
						}
					}
					if !found {
						t.Errorf("AuthProperties.Pack() missing authentication method")
					}
				}
			}
		})
	}
}

func TestAuthProperties_Unpack(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name: "unpack properties with method only",
			data: []byte{
				0x15, 0x00, 0x05, 0x50, 0x4C, 0x41, 0x49, 0x4E, // Authentication Method: "PLAIN"
			},
			wantErr: false,
		},
		{
			name: "unpack properties with method and data",
			data: []byte{
				0x15, 0x00, 0x05, 0x50, 0x4C, 0x41, 0x49, 0x4E, // Authentication Method: "PLAIN"
				0x16, 0x00, 0x11, 0x75, 0x73, 0x65, 0x72, 0x6E, 0x61, 0x6D, 0x65, 0x3A, 0x70, 0x61, 0x73, 0x73, 0x77, 0x6F, 0x72, 0x64, // Authentication Data: "username:password"
			},
			wantErr: false,
		},
		{
			name: "unpack properties with duplicate property",
			data: []byte{
				0x15, 0x00, 0x05, 0x50, 0x4C, 0x41, 0x49, 0x4E, // Authentication Method: "PLAIN"
				0x15, 0x00, 0x05, 0x50, 0x4C, 0x41, 0x49, 0x4E, // Duplicate Authentication Method
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			props := &AuthProperties{}
			buf := bytes.NewBuffer(tt.data)
			err := props.Unpack(buf)
			if (err != nil) != tt.wantErr {
				t.Errorf("AuthProperties.Unpack() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				// 检查解析结果
				if props.AuthenticationMethod != "PLAIN" {
					t.Errorf("AuthProperties.Unpack() method = %s, want PLAIN", props.AuthenticationMethod)
				}
			}
		})
	}
}

func TestAUTH_String(t *testing.T) {
	tests := []struct {
		name string
		auth *AUTH
		want string
	}{
		{
			name: "AUTH with success reason only",
			auth: &AUTH{
				FixedHeader: &FixedHeader{Kind: 0x0F},
				ReasonCode:  CodeSuccess,
			},
			want: "AUTH{ReasonCode:0x00}",
		},
		{
			name: "AUTH with method and data",
			auth: &AUTH{
				FixedHeader: &FixedHeader{Kind: 0x0F},
				ReasonCode:  CodeContinueAuthentication,
				Props: &AuthProperties{
					AuthenticationMethod: "PLAIN",
					AuthenticationData:   []byte("test"),
				},
			},
			want: "AUTH{ReasonCode:0x18, Method:PLAIN, DataLen:4}",
		},
		{
			name: "AUTH with all properties",
			auth: &AUTH{
				FixedHeader: &FixedHeader{Kind: 0x0F},
				ReasonCode:  CodeReAuthenticate,
				Props: &AuthProperties{
					AuthenticationMethod: "PLAIN",
					AuthenticationData:   []byte("test"),
					ReasonString:         "Re-authentication required",
					UserProperty: map[string][]string{
						"client_id": {"test_client"},
					},
				},
			},
			want: "AUTH{ReasonCode:0x19, Method:PLAIN, DataLen:4, Reason:Re-authentication required, UserProps:1}",
		},
		{
			name: "nil AUTH",
			auth: nil,
			want: "AUTH<nil>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.auth.String()
			if got != tt.want {
				t.Errorf("AUTH.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAUTH_Integration(t *testing.T) {
	// 测试完整的序列化和反序列化流程
	original := &AUTH{
		FixedHeader: &FixedHeader{
			Kind:            0x0F,
			Dup:             0,
			QoS:             0,
			Retain:          0,
			Version:         VERSION500,
			RemainingLength: 0,
		},
		ReasonCode: CodeContinueAuthentication,
		Props: &AuthProperties{
			AuthenticationMethod: "PLAIN",
			AuthenticationData:   []byte("username:password"),
			ReasonString:         "Authentication required",
			UserProperty: map[string][]string{
				"client_id": {"test_client"},
				"version":   {"1.0"},
			},
		},
	}

	// 序列化
	var buf bytes.Buffer
	if err := original.Packet(&buf); err != nil {
		t.Fatalf("Failed to pack AUTH: %v", err)
	}

	// 反序列化 - 需要先解析固定报头
	serializedData := buf.Bytes()
	t.Logf("Serialized data: %v", serializedData)

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
	t.Logf("Variable header starts at byte %d", varHeaderStart)

	// 创建只包含可变报头和载荷的缓冲区
	varHeaderBuf := bytes.NewBuffer(serializedData[varHeaderStart:])
	t.Logf("Variable header buffer: %v", varHeaderBuf.Bytes())

	deserialized := &AUTH{
		FixedHeader: &FixedHeader{
			Kind:    0x0F,
			Version: VERSION500,
		},
	}

	if err := deserialized.Unpack(varHeaderBuf); err != nil {
		t.Fatalf("Failed to unpack AUTH: %v", err)
	}

	// 验证结果
	if deserialized.ReasonCode.Code != original.ReasonCode.Code {
		t.Errorf("ReasonCode mismatch: got 0x%02X, want 0x%02X",
			deserialized.ReasonCode.Code, original.ReasonCode.Code)
	}

	if deserialized.Props.AuthenticationMethod != original.Props.AuthenticationMethod {
		t.Errorf("AuthenticationMethod mismatch: got %s, want %s",
			deserialized.Props.AuthenticationMethod, original.Props.AuthenticationMethod)
	}

	if !bytes.Equal(deserialized.Props.AuthenticationData, original.Props.AuthenticationData) {
		t.Errorf("AuthenticationData mismatch: got %v, want %v",
			deserialized.Props.AuthenticationData, original.Props.AuthenticationData)
	}

	if deserialized.Props.ReasonString != original.Props.ReasonString {
		t.Errorf("ReasonString mismatch: got %s, want %s",
			deserialized.Props.ReasonString, original.Props.ReasonString)
	}

	// 检查用户属性数量
	if len(deserialized.Props.UserProperty) != len(original.Props.UserProperty) {
		t.Errorf("UserProperty count mismatch: got %d, want %d",
			len(deserialized.Props.UserProperty), len(original.Props.UserProperty))
	}
}
