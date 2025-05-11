package uefi

import (
	"bytes"
	"reflect"
	"testing"
)

func TestExtractUCS16String(t *testing.T) {
	tests := []struct {
		name           string
		data           []byte
		expectedString []byte
		expectedLen    uint32
	}{
		{
			name:           "Simple string",
			data:           []byte{0x48, 0x00, 0x65, 0x00, 0x6C, 0x00, 0x6C, 0x00, 0x6F, 0x00, 0x00, 0x00, 0xFF, 0xFF},
			expectedString: []byte{0x48, 0x00, 0x65, 0x00, 0x6C, 0x00, 0x6C, 0x00, 0x6F, 0x00},
			expectedLen:    12,
		},
		{
			name:           "Empty string",
			data:           []byte{0x00, 0x00, 0xFF, 0xFF},
			expectedString: []byte{},
			expectedLen:    2,
		},
		{
			name:           "No null terminator",
			data:           []byte{0x48, 0x00, 0x65, 0x00},
			expectedString: []byte{0x48, 0x00, 0x65, 0x00},
			expectedLen:    4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			str, length := ExtractUCS16String(tt.data)
			if !reflect.DeepEqual(str, tt.expectedString) {
				t.Errorf("ExtractUCS16String() string = %v, want %v", str, tt.expectedString)
			}
			if length != tt.expectedLen {
				t.Errorf("ExtractUCS16String() length = %v, want %v", length, tt.expectedLen)
			}
		})
	}
}

func TestDecodeUTF16LE(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected string
	}{
		{
			name:     "Hello",
			data:     []byte{0x48, 0x00, 0x65, 0x00, 0x6C, 0x00, 0x6C, 0x00, 0x6F, 0x00},
			expected: "Hello",
		},
		{
			name:     "Empty string",
			data:     []byte{},
			expected: "",
		},
		{
			name:     "Unicode characters",
			data:     []byte{0x48, 0x00, 0x65, 0x00, 0x6C, 0x00, 0x6C, 0x00, 0x6F, 0x00, 0x20, 0x00, 0x1C, 0x04, 0x38, 0x04, 0x40, 0x04},
			expected: "Hello Мир",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DecodeUTF16LE(tt.data)
			if result != tt.expected {
				t.Errorf("DecodeUTF16LE() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestEncodeUTF16LE(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []byte
	}{
		{
			name:     "Empty string",
			input:    "",
			expected: []byte{},
		},
		{
			name:     "Simple string",
			input:    "Attempt 1",
			expected: []byte{65, 0, 116, 0, 116, 0, 101, 0, 109, 0, 112, 0, 116, 0, 32, 0, 49, 0},
		},
		{
			name:     "UEFI string",
			input:    "UEFI QEMU DVD-ROM",
			expected: []byte{85, 0, 69, 0, 70, 0, 73, 0, 32, 0, 81, 0, 69, 0, 77, 0, 85, 0, 32, 0, 68, 0, 86, 0, 68, 0, 45, 0, 82, 0, 79, 0, 77, 0},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EncodeUTF16LE(tt.input)
			if !bytes.Equal(result, tt.expected) {
				t.Errorf("EncodeUTF16LE() = %v, want %v", result, tt.expected)
			}
		})
	}
}
