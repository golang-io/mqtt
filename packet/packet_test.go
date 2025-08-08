package packet

import (
	"bytes"
	"testing"
)

func TestVersionConstants(t *testing.T) {
	if VERSION311 == 0 {
		t.Error("VERSION311 should not be 0")
	}
	if VERSION500 == 0 {
		t.Error("VERSION500 should not be 0")
	}
	if VERSION311 == VERSION500 {
		t.Error("VERSION311 and VERSION500 should be different")
	}
}

func TestPacketTypeConstants(t *testing.T) {
	// Test that all packet type constants are defined and unique
	types := []byte{
		0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8, 0x9, 0xA, 0xB, 0xC, 0xD, 0xE, 0xF,
	}

	seen := make(map[byte]bool)
	for _, packetType := range types {
		if packetType == 0 {
			t.Error("packet type constant should not be 0")
		}
		if seen[packetType] {
			t.Errorf("duplicate packet type constant: %d", packetType)
		}
		seen[packetType] = true
	}
}

func TestKindMap(t *testing.T) {
	// Test that Kind map has entries for all packet types
	expectedKinds := []byte{0x0, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x7, 0x8, 0x9, 0xA, 0xB, 0xC, 0xD, 0xE, 0xF}

	for _, kind := range expectedKinds {
		if name, exists := Kind[kind]; !exists {
			t.Errorf("Kind map missing entry for %d", kind)
		} else if name == "" {
			t.Errorf("Kind map has empty name for %d", kind)
		}
	}
}

func TestEncodeDecodeLength(t *testing.T) {
	testCases := []uint32{
		0, 1, 127, 128, 16383, 16384, 2097151, 2097152,
	}

	for _, length := range testCases {
		encoded, err := encodeLength(length)
		if err != nil {
			t.Errorf("encodeLength failed for %d: %v", length, err)
			continue
		}

		// Create a buffer with the encoded length
		buf := bytes.NewBuffer(encoded)
		decoded, err := decodeLength(buf)
		if err != nil {
			t.Errorf("decodeLength failed for %d: %v", length, err)
			continue
		}

		if decoded != length {
			t.Errorf("length mismatch: expected %d, got %d", length, decoded)
		}
	}
}

func TestEncodeLengthTooLarge(t *testing.T) {
	// Test encoding a value that's too large
	_, err := encodeLength(uint32(0xFFFFFFF + 1))
	if err == nil {
		t.Error("encodeLength should return error for value too large")
	}
}

func TestS2BAndI2B(t *testing.T) {
	// Test s2b function
	testString := "test"
	result := s2b(testString)
	if len(result) != len(testString)+2 {
		t.Errorf("s2b result length should be string length + 2, got %d", len(result))
	}

	// Test i2b function
	testInt := uint16(12345)
	resultInt := i2b(testInt)
	if len(resultInt) != 2 {
		t.Error("i2b result should be 2 bytes")
	}
}

func TestEncodeDecodeUTF8(t *testing.T) {
	testStrings := []string{
		"",
		"test",
		"hello world",
		"测试",
	}

	for _, testStr := range testStrings {
		encoded := encodeUTF8(testStr)
		if len(encoded) != len(testStr)+2 {
			t.Errorf("encodeUTF8 result length should be string length + 2, got %d", len(encoded))
		}

		// Create a buffer with the encoded string
		buf := bytes.NewBuffer(encoded)
		decoded := decodeUTF8[string](buf)
		if decoded != testStr {
			t.Errorf("UTF8 encode/decode mismatch: expected %s, got %s", testStr, decoded)
		}
	}
}

func TestS2I(t *testing.T) {
	// Test empty string
	if s2i("") != 0 {
		t.Error("s2i should return 0 for empty string")
	}

	// Test non-empty string
	if s2i("test") != 1 {
		t.Error("s2i should return 1 for non-empty string")
	}
}
