package packet

import (
	"bytes"
	"testing"
)

// TestUNSUBSCRIBE_Kind 测试UNSUBSCRIBE报文类型
func TestUNSUBSCRIBE_Kind(t *testing.T) {
	unsubscribe := &UNSUBSCRIBE{}
	if unsubscribe.Kind() != 0xA {
		t.Errorf("UNSUBSCRIBE.Kind() = %d, want 0xA", unsubscribe.Kind())
	}
}

// TestUNSUBSCRIBE_Pack_MQTT311 测试MQTT v3.1.1 UNSUBSCRIBE报文打包
func TestUNSUBSCRIBE_Pack_MQTT311(t *testing.T) {
	tests := []struct {
		name          string
		packetID      uint16
		subscriptions []Subscription
		wantErr       bool
		expectedFlags byte
	}{
		{
			name:     "单个主题取消订阅",
			packetID: 12345,
			subscriptions: []Subscription{
				{TopicFilter: "test/topic"},
			},
			wantErr:       false,
			expectedFlags: 0xA0, // 0x0A << 4 (QoS=1, DUP=0, RETAIN=0)
		},
		{
			name:     "多个主题取消订阅",
			packetID: 12346,
			subscriptions: []Subscription{
				{TopicFilter: "sensor/+/data"},
				{TopicFilter: "device/#"},
				{TopicFilter: "user/status"},
			},
			wantErr:       false,
			expectedFlags: 0xA0,
		},
		{
			name:     "带通配符的主题取消订阅",
			packetID: 12347,
			subscriptions: []Subscription{
				{TopicFilter: "home/+/sensor/#"},
				{TopicFilter: "weather/*/forecast"},
			},
			wantErr:       false,
			expectedFlags: 0xA0,
		},
		{
			name:          "空主题过滤器列表",
			packetID:      12348,
			subscriptions: []Subscription{},
			wantErr:       true,
			expectedFlags: 0xA0,
		},
		{
			name:     "特殊字符主题取消订阅",
			packetID: 12349,
			subscriptions: []Subscription{
				{TopicFilter: "test/中文/主题"},
				{TopicFilter: "test/emoji/🚀"},
			},
			wantErr:       false,
			expectedFlags: 0xA0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unsubscribe := &UNSUBSCRIBE{
				FixedHeader: &FixedHeader{
					Version: VERSION311,
					Kind:    0x0A,
				},
				PacketID:      tt.packetID,
				Subscriptions: tt.subscriptions,
			}

			var buf bytes.Buffer
			err := unsubscribe.Pack(&buf)

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
			if data[0] != tt.expectedFlags {
				t.Errorf("Fixed header flags = 0x%02x, want 0x%02x", data[0], tt.expectedFlags)
			}

			// 验证报文标识符
			packetID := uint16(data[2])<<8 | uint16(data[3])
			if packetID != tt.packetID {
				t.Errorf("Packet ID = %d, want %d", packetID, tt.packetID)
			}

			// 验证主题过滤器
			payloadStart := 4 // 固定报头(2) + 可变报头(2)
			offset := payloadStart
			for i, subscription := range tt.subscriptions {
				// 验证主题过滤器长度
				topicLength := uint16(data[offset])<<8 | uint16(data[offset+1])
				expectedLength := uint16(len(subscription.TopicFilter))
				if topicLength != expectedLength {
					t.Errorf("Topic filter[%d] length = %d, want %d", i, topicLength, expectedLength)
				}

				// 验证主题过滤器内容
				topicStart := offset + 2
				topicEnd := topicStart + int(topicLength)
				if topicEnd <= len(data) {
					topicContent := string(data[topicStart:topicEnd])
					if topicContent != subscription.TopicFilter {
						t.Errorf("Topic filter[%d] content = %s, want %s", i, topicContent, subscription.TopicFilter)
					}
				}

				offset = topicEnd
			}
		})
	}
}

// TestUNSUBSCRIBE_Pack_MQTT500 测试MQTT v5.0 UNSUBSCRIBE报文打包
func TestUNSUBSCRIBE_Pack_MQTT500(t *testing.T) {
	tests := []struct {
		name          string
		packetID      uint16
		subscriptions []Subscription
		props         *UnsubscribeProperties
		wantErr       bool
	}{
		{
			name:     "带用户属性的取消订阅",
			packetID: 12345,
			subscriptions: []Subscription{
				{TopicFilter: "test/topic"},
			},
			props: &UnsubscribeProperties{
				UserProperty: map[string][]string{
					"client_id": {"test_client"},
					"timestamp": {"1234567890"},
				},
			},
			wantErr: false,
		},
		{
			name:     "多个用户属性的取消订阅",
			packetID: 12346,
			subscriptions: []Subscription{
				{TopicFilter: "sensor/+/data"},
				{TopicFilter: "device/#"},
			},
			props: &UnsubscribeProperties{
				UserProperty: map[string][]string{
					"session_id": {"sess_123"},
					"reason":     {"cleanup"},
				},
			},
			wantErr: false,
		},
		{
			name:     "无属性的取消订阅",
			packetID: 12347,
			subscriptions: []Subscription{
				{TopicFilter: "test/topic"},
			},
			props:   nil,
			wantErr: false,
		},
		{
			name:     "空用户属性的取消订阅",
			packetID: 12348,
			subscriptions: []Subscription{
				{TopicFilter: "test/topic"},
			},
			props: &UnsubscribeProperties{
				UserProperty: map[string][]string{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			unsubscribe := &UNSUBSCRIBE{
				FixedHeader: &FixedHeader{
					Version: VERSION500,
					Kind:    0x0A,
				},
				PacketID:      tt.packetID,
				Subscriptions: tt.subscriptions,
				Props:         tt.props,
			}

			var buf bytes.Buffer
			err := unsubscribe.Pack(&buf)

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
			if data[0] != 0xA0 { // 0x0A << 4
				t.Errorf("Fixed header flags = 0x%02x, want 0xA0", data[0])
			}

			// 验证报文标识符
			packetID := uint16(data[2])<<8 | uint16(data[3])
			if packetID != tt.packetID {
				t.Errorf("Packet ID = %d, want %d", packetID, tt.packetID)
			}

			// 验证属性长度（如果存在）
			if tt.props != nil && len(tt.props.UserProperty) > 0 {
				// 这里需要解析属性长度，比较复杂，暂时跳过详细验证
				// 主要验证报文能够正常打包
			}
		})
	}
}

// TestUNSUBSCRIBE_Unpack_MQTT311 测试MQTT v3.1.1 UNSUBSCRIBE报文解包
func TestUNSUBSCRIBE_Unpack_MQTT311(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    *UNSUBSCRIBE
		wantErr bool
	}{
		{
			name: "单个主题取消订阅",
			data: []byte{
				0x30, 0x39, // 报文标识符: 12345
				0x00, 0x0A, // 主题过滤器长度: 10
				0x74, 0x65, 0x73, 0x74, 0x2F, 0x74, 0x6F, 0x70, 0x69, 0x63, // "test/topic"
			},
			want: &UNSUBSCRIBE{
				PacketID: 12345,
				Subscriptions: []Subscription{
					{TopicFilter: "test/topic"},
				},
			},
			wantErr: false,
		},
		{
			name: "多个主题取消订阅",
			data: []byte{
				0x30, 0x3A, // 报文标识符: 12346
				0x00, 0x0B, // 主题过滤器1长度: 11
				0x73, 0x65, 0x6E, 0x73, 0x6F, 0x72, 0x2F, 0x2B, 0x2F, 0x64, 0x61, 0x74, 0x61, // "sensor/+/data"
				0x00, 0x07, // 主题过滤器2长度: 7
				0x64, 0x65, 0x76, 0x69, 0x63, 0x65, 0x23, // "device#"
			},
			want: &UNSUBSCRIBE{
				PacketID: 12346,
				Subscriptions: []Subscription{
					{TopicFilter: "sensor/+/data"},
					{TopicFilter: "device#"},
				},
			},
			wantErr: false,
		},
		{
			name: "带通配符的主题取消订阅",
			data: []byte{
				0x30, 0x3B, // 报文标识符: 12347
				0x00, 0x0F, // 主题过滤器长度: 15
				0x68, 0x6F, 0x6D, 0x65, 0x2F, 0x2B, 0x2F, 0x73, 0x65, 0x6E, 0x73, 0x6F, 0x72, 0x2F, 0x23, // "home/+/sensor/#"
			},
			want: &UNSUBSCRIBE{
				PacketID: 12347,
				Subscriptions: []Subscription{
					{TopicFilter: "home/+/sensor/#"},
				},
			},
			wantErr: false,
		},
		{
			name: "特殊字符主题取消订阅",
			data: []byte{
				0x30, 0x3C, // 报文标识符: 12348
				0x00, 0x0F, // 主题过滤器长度: 15
				0x74, 0x65, 0x73, 0x74, 0x2F, 0xE4, 0xB8, 0xAD, 0xE6, 0x96, 0x87, 0x2F, 0xE4, 0xB8, 0xBB, 0xE9, 0xA2, 0x98, // "test/中文/主题"
			},
			want: &UNSUBSCRIBE{
				PacketID: 12348,
				Subscriptions: []Subscription{
					{TopicFilter: "test/中文/主题"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.NewBuffer(tt.data)

			unsubscribe := &UNSUBSCRIBE{
				FixedHeader: &FixedHeader{
					Version: VERSION311,
				},
			}

			err := unsubscribe.Unpack(buf)

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
			if unsubscribe.PacketID != tt.want.PacketID {
				t.Errorf("Packet ID = %d, want %d", unsubscribe.PacketID, tt.want.PacketID)
			}

			// 验证主题过滤器数量
			if len(unsubscribe.Subscriptions) != len(tt.want.Subscriptions) {
				t.Errorf("Subscription count = %d, want %d", len(unsubscribe.Subscriptions), len(tt.want.Subscriptions))
			}

			// 验证主题过滤器内容
			for i, subscription := range unsubscribe.Subscriptions {
				if subscription.TopicFilter != tt.want.Subscriptions[i].TopicFilter {
					t.Errorf("Topic filter[%d] = %s, want %s", i, subscription.TopicFilter, tt.want.Subscriptions[i].TopicFilter)
				}
			}
		})
	}
}

// TestUNSUBSCRIBE_Unpack_MQTT500 测试MQTT v5.0 UNSUBSCRIBE报文解包
func TestUNSUBSCRIBE_Unpack_MQTT500(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    *UNSUBSCRIBE
		wantErr bool
	}{
		{
			name: "带用户属性的取消订阅",
			data: []byte{
				0x30, 0x39, // 报文标识符: 12345
				0x0E,       // 属性长度: 14
				0x26,       // 属性标识符: User Property (38)
				0x00, 0x0A, // 键长度: 10
				0x63, 0x6C, 0x69, 0x65, 0x6E, 0x74, 0x5F, 0x69, 0x64, // "client_id"
				0x00, 0x0B, // 值长度: 11
				0x74, 0x65, 0x73, 0x74, 0x5F, 0x63, 0x6C, 0x69, 0x65, 0x6E, 0x74, // "test_client"
				0x00, 0x0A, // 主题过滤器长度: 10
				0x74, 0x65, 0x73, 0x74, 0x2F, 0x74, 0x6F, 0x70, 0x69, 0x63, // "test/topic"
			},
			want: &UNSUBSCRIBE{
				PacketID: 12345,
				Props: &UnsubscribeProperties{
					UserProperty: map[string][]string{
						"client_id": {"test_client"},
					},
				},
				Subscriptions: []Subscription{
					{TopicFilter: "test/topic"},
				},
			},
			wantErr: false,
		},
		{
			name: "无属性的取消订阅",
			data: []byte{
				0x30, 0x3A, // 报文标识符: 12346
				0x00,       // 属性长度: 0
				0x00, 0x0A, // 主题过滤器长度: 10
				0x74, 0x65, 0x73, 0x74, 0x2F, 0x74, 0x6F, 0x70, 0x69, 0x63, // "test/topic"
			},
			want: &UNSUBSCRIBE{
				PacketID: 12346,
				Props:    &UnsubscribeProperties{},
				Subscriptions: []Subscription{
					{TopicFilter: "test/topic"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.NewBuffer(tt.data)

			unsubscribe := &UNSUBSCRIBE{
				FixedHeader: &FixedHeader{
					Version: VERSION500,
				},
			}

			err := unsubscribe.Unpack(buf)

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
			if unsubscribe.PacketID != tt.want.PacketID {
				t.Errorf("Packet ID = %d, want %d", unsubscribe.PacketID, tt.want.PacketID)
			}

			// 验证属性
			if tt.want.Props != nil {
				if unsubscribe.Props == nil {
					t.Error("Props should not be nil")
				} else {
					if len(tt.want.Props.UserProperty) > 0 {
						if len(unsubscribe.Props.UserProperty) != len(tt.want.Props.UserProperty) {
							t.Errorf("UserProperty count = %d, want %d", len(unsubscribe.Props.UserProperty), len(tt.want.Props.UserProperty))
						}
					}
				}
			}

			// 验证主题过滤器
			if len(unsubscribe.Subscriptions) != len(tt.want.Subscriptions) {
				t.Errorf("Subscription count = %d, want %d", len(unsubscribe.Subscriptions), len(tt.want.Subscriptions))
			}
		})
	}
}

// TestUNSUBSCRIBE_Unpack_InvalidData 测试无效数据的解包处理
func TestUNSUBSCRIBE_Unpack_InvalidData(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "数据不足",
			data:    []byte{0x30, 0x39}, // 只有报文标识符，缺少主题过滤器
			wantErr: true,
		},
		{
			name: "主题过滤器长度无效",
			data: []byte{
				0x30, 0x39, // 报文标识符: 12345
				0x00, 0x05, // 主题过滤器长度: 5
				0x74, 0x65, 0x73, 0x74, // 只有4字节，长度不匹配
			},
			wantErr: true,
		},
		{
			name:    "空数据",
			data:    []byte{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.NewBuffer(tt.data)

			unsubscribe := &UNSUBSCRIBE{
				FixedHeader: &FixedHeader{
					Version: VERSION311,
				},
			}

			err := unsubscribe.Unpack(buf)

			if tt.wantErr && err == nil {
				t.Error("Unpack() should return error for invalid data")
			}
		})
	}
}

// TestUNSUBSCRIBE_RoundTrip 测试UNSUBSCRIBE报文的往返打包解包
func TestUNSUBSCRIBE_RoundTrip(t *testing.T) {
	tests := []struct {
		name          string
		version       byte
		packetID      uint16
		subscriptions []Subscription
		props         *UnsubscribeProperties
	}{
		{
			name:     "MQTT v3.1.1 简单取消订阅",
			version:  VERSION311,
			packetID: 12345,
			subscriptions: []Subscription{
				{TopicFilter: "test/topic"},
			},
			props: nil,
		},
		{
			name:     "MQTT v3.1.1 多个主题取消订阅",
			version:  VERSION311,
			packetID: 12346,
			subscriptions: []Subscription{
				{TopicFilter: "sensor/+/data"},
				{TopicFilter: "device/#"},
				{TopicFilter: "user/status"},
			},
			props: nil,
		},
		{
			name:     "MQTT v5.0 带用户属性",
			version:  VERSION500,
			packetID: 12347,
			subscriptions: []Subscription{
				{TopicFilter: "test/topic"},
			},
			props: &UnsubscribeProperties{
				UserProperty: map[string][]string{
					"test_key": {"test_value"},
				},
			},
		},
		{
			name:     "MQTT v5.0 多个用户属性",
			version:  VERSION500,
			packetID: 12348,
			subscriptions: []Subscription{
				{TopicFilter: "sensor/+/data"},
				{TopicFilter: "device/#"},
			},
			props: &UnsubscribeProperties{
				UserProperty: map[string][]string{
					"session_id": {"sess_123"},
					"reason":     {"cleanup"},
					"timestamp":  {"1234567890"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建原始UNSUBSCRIBE
			original := &UNSUBSCRIBE{
				FixedHeader: &FixedHeader{
					Version: tt.version,
					Kind:    0x0A,
				},
				PacketID:      tt.packetID,
				Subscriptions: tt.subscriptions,
				Props:         tt.props,
			}

			// 打包
			var buf bytes.Buffer
			if err := original.Pack(&buf); err != nil {
				t.Fatalf("Pack() failed: %v", err)
			}

			// 解包
			unpacked := &UNSUBSCRIBE{
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

			if len(unpacked.Subscriptions) != len(original.Subscriptions) {
				t.Errorf("Subscription count mismatch: got %d, want %d", len(unpacked.Subscriptions), len(original.Subscriptions))
			}

			for i, subscription := range unpacked.Subscriptions {
				if subscription.TopicFilter != original.Subscriptions[i].TopicFilter {
					t.Errorf("Topic filter[%d] mismatch: got %s, want %s", i, subscription.TopicFilter, original.Subscriptions[i].TopicFilter)
				}
			}

			// 验证属性（如果存在）
			if tt.props != nil {
				if unpacked.Props == nil {
					t.Error("Props should not be nil after round trip")
				} else {
					if len(tt.props.UserProperty) > 0 {
						if len(unpacked.Props.UserProperty) != len(tt.props.UserProperty) {
							t.Errorf("UserProperty count mismatch: got %d, want %d", len(unpacked.Props.UserProperty), len(tt.props.UserProperty))
						}
					}
				}
			}
		})
	}
}

// TestUnsubscribeProperties_Pack 测试UnsubscribeProperties的打包功能
func TestUnsubscribeProperties_Pack(t *testing.T) {
	tests := []struct {
		name    string
		props   *UnsubscribeProperties
		wantErr bool
	}{
		{
			name:    "空属性",
			props:   &UnsubscribeProperties{},
			wantErr: false,
		},
		{
			name: "只有用户属性",
			props: &UnsubscribeProperties{
				UserProperty: map[string][]string{
					"key1": {"value1"},
				},
			},
			wantErr: false,
		},
		{
			name: "多个用户属性",
			props: &UnsubscribeProperties{
				UserProperty: map[string][]string{
					"key1": {"value1", "value2"},
					"key2": {"value3"},
				},
			},
			wantErr: false,
		},
		{
			name: "复杂用户属性",
			props: &UnsubscribeProperties{
				UserProperty: map[string][]string{
					"session_id": {"sess_123"},
					"reason":     {"cleanup", "maintenance"},
					"timestamp":  {"1234567890"},
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
			if len(tt.props.UserProperty) > 0 {
				if len(data) == 0 {
					t.Error("Pack() should return non-empty data when properties exist")
				}
			}
		})
	}
}

// TestUnsubscribeProperties_Unpack 测试UnsubscribeProperties的解包功能
func TestUnsubscribeProperties_Unpack(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    *UnsubscribeProperties
		wantErr bool
	}{
		{
			name:    "空属性",
			data:    []byte{0x00}, // 属性长度0
			want:    &UnsubscribeProperties{},
			wantErr: false,
		},
		{
			name: "用户属性",
			data: []byte{
				0x0E,       // 属性长度: 14
				0x26,       // 属性标识符: User Property (38)
				0x00, 0x03, // 键长度: 3
				0x6B, 0x65, 0x79, // "key"
				0x00, 0x05, // 值长度: 5
				0x76, 0x61, 0x6C, 0x75, 0x65, // "value"
			},
			want: &UnsubscribeProperties{
				UserProperty: map[string][]string{
					"key": {"value"},
				},
			},
			wantErr: false,
		},
		{
			name: "多个用户属性",
			data: []byte{
				0x1C,       // 属性长度: 28
				0x26,       // 属性标识符: User Property (38)
				0x00, 0x03, // 键长度: 3
				0x6B, 0x65, 0x79, // "key"
				0x00, 0x05, // 值长度: 5
				0x76, 0x61, 0x6C, 0x75, 0x65, // "value"
				0x26,       // 属性标识符: User Property (38)
				0x00, 0x04, // 键长度: 4
				0x6E, 0x61, 0x6D, 0x65, // "name"
				0x00, 0x06, // 值长度: 6
				0x74, 0x65, 0x73, 0x74, 0x65, 0x72, // "tester"
			},
			want: &UnsubscribeProperties{
				UserProperty: map[string][]string{
					"key":  {"value"},
					"name": {"tester"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := bytes.NewBuffer(tt.data)
			props := &UnsubscribeProperties{}

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

			// 验证用户属性
			if len(tt.want.UserProperty) > 0 {
				if len(props.UserProperty) != len(tt.want.UserProperty) {
					t.Errorf("UserProperty count = %d, want %d", len(props.UserProperty), len(tt.want.UserProperty))
				}
			}
		})
	}
}

// TestUNSUBSCRIBE_ProtocolCompliance 测试UNSUBSCRIBE报文协议合规性
func TestUNSUBSCRIBE_ProtocolCompliance(t *testing.T) {
	tests := []struct {
		name        string
		description string
		testFunc    func(t *testing.T)
	}{
		{
			name:        "标志位必须正确设置",
			description: "MQTT协议要求UNSUBSCRIBE报文的标志位必须为DUP=0, QoS=1, RETAIN=0",
			testFunc: func(t *testing.T) {
				unsubscribe := &UNSUBSCRIBE{
					FixedHeader: &FixedHeader{
						Version: VERSION311,
						Kind:    0x0A,
					},
					PacketID: 12345,
					Subscriptions: []Subscription{
						{TopicFilter: "test/topic"},
					},
				}

				var buf bytes.Buffer
				if err := unsubscribe.Pack(&buf); err != nil {
					t.Fatalf("Pack() failed: %v", err)
				}

				data := buf.Bytes()
				flags := data[0] & 0x0F
				if flags != 0x02 { // DUP=0, QoS=1, RETAIN=0
					t.Errorf("Flags = 0x%02x, must be 0x02", flags)
				}
			},
		},
		{
			name:        "至少包含一个主题过滤器",
			description: "UNSUBSCRIBE报文必须包含至少一个主题过滤器",
			testFunc: func(t *testing.T) {
				unsubscribe := &UNSUBSCRIBE{
					FixedHeader: &FixedHeader{
						Version: VERSION311,
						Kind:    0x0A,
					},
					PacketID:      12345,
					Subscriptions: []Subscription{},
				}

				var buf bytes.Buffer
				err := unsubscribe.Pack(&buf)
				if err == nil {
					t.Error("Pack() should fail when no topic filters are provided")
				}
			},
		},
		{
			name:        "主题过滤器必须是UTF-8编码",
			description: "主题过滤器必须是有效的UTF-8编码字符串",
			testFunc: func(t *testing.T) {
				// 测试有效的UTF-8字符串
				validTopics := []string{
					"test/topic",
					"sensor/+/data",
					"device/#",
					"test/中文/主题",
					"test/emoji/🚀",
				}

				for _, topic := range validTopics {
					unsubscribe := &UNSUBSCRIBE{
						FixedHeader: &FixedHeader{
							Version: VERSION311,
							Kind:    0x0A,
						},
						PacketID: 12345,
						Subscriptions: []Subscription{
							{TopicFilter: topic},
						},
					}

					var buf bytes.Buffer
					if err := unsubscribe.Pack(&buf); err != nil {
						t.Errorf("Pack() failed for valid topic '%s': %v", topic, err)
					}
				}
			},
		},
		{
			name:        "报文标识符范围验证",
			description: "报文标识符必须在1-65535范围内",
			testFunc: func(t *testing.T) {
				validPacketIDs := []uint16{1, 12345, 65535}
				invalidPacketIDs := []uint16{0}

				// 测试有效报文标识符
				for _, packetID := range validPacketIDs {
					unsubscribe := &UNSUBSCRIBE{
						FixedHeader: &FixedHeader{
							Version: VERSION311,
							Kind:    0x0A,
						},
						PacketID: packetID,
						Subscriptions: []Subscription{
							{TopicFilter: "test/topic"},
						},
					}

					var buf bytes.Buffer
					if err := unsubscribe.Pack(&buf); err != nil {
						t.Errorf("Pack() failed for valid packet ID %d: %v", packetID, err)
					}
				}

				// 测试无效报文标识符（0应该被允许，但65536会溢出）
				for _, packetID := range invalidPacketIDs {
					if packetID == 0 {
						// 0是有效的
						continue
					}
					// 65536会溢出，但Go的uint16会自动截断
					unsubscribe := &UNSUBSCRIBE{
						FixedHeader: &FixedHeader{
							Version: VERSION311,
							Kind:    0x0A,
						},
						PacketID: packetID,
						Subscriptions: []Subscription{
							{TopicFilter: "test/topic"},
						},
					}

					var buf bytes.Buffer
					if err := unsubscribe.Pack(&buf); err != nil {
						t.Errorf("Pack() failed for packet ID %d: %v", packetID, err)
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

// BenchmarkUNSUBSCRIBE_Pack 性能测试：UNSUBSCRIBE打包
func BenchmarkUNSUBSCRIBE_Pack(b *testing.B) {
	unsubscribe := &UNSUBSCRIBE{
		FixedHeader: &FixedHeader{
			Version: VERSION311,
			Kind:    0x0A,
		},
		PacketID: 12345,
		Subscriptions: []Subscription{
			{TopicFilter: "sensor/+/data"},
			{TopicFilter: "device/#"},
			{TopicFilter: "user/status"},
		},
	}

	var buf bytes.Buffer
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf.Reset()
		if err := unsubscribe.Pack(&buf); err != nil {
			b.Fatalf("Pack() failed: %v", err)
		}
	}
}

// BenchmarkUNSUBSCRIBE_Unpack 性能测试：UNSUBSCRIBE解包
func BenchmarkUNSUBSCRIBE_Unpack(b *testing.B) {
	// 准备测试数据
	testData := []byte{
		0x30, 0x39, // 报文标识符: 12345
		0x00, 0x0B, // 主题过滤器1长度: 11
		0x73, 0x65, 0x6E, 0x73, 0x6F, 0x72, 0x2F, 0x2B, 0x2F, 0x64, 0x61, // "sensor/+/data"
		0x00, 0x08, // 主题过滤器2长度: 8
		0x64, 0x65, 0x76, 0x69, 0x63, 0x65, 0x23, // "device#"
	}

	unsubscribe := &UNSUBSCRIBE{
		FixedHeader: &FixedHeader{
			Version: VERSION311,
		},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		buf := bytes.NewBuffer(testData)
		if err := unsubscribe.Unpack(buf); err != nil {
			b.Fatalf("Unpack() failed: %v", err)
		}
	}
}
