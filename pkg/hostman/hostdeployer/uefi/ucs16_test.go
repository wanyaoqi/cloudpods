package uefi

import (
	"bytes"
	"testing"
)

func TestExtractUCS16String(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected []byte
		size     int
	}{
		{
			name:     "Empty string",
			input:    []byte{0, 0},
			expected: []byte{},
			size:     2,
		},
		{
			name:     "Simple string",
			input:    []byte{65, 0, 116, 0, 116, 0, 101, 0, 109, 0, 112, 0, 116, 0, 32, 0, 49, 0, 0, 0},
			expected: []byte{65, 0, 116, 0, 116, 0, 101, 0, 109, 0, 112, 0, 116, 0, 32, 0, 49, 0},
			size:     20,
		},
		{
			name:     "String with data after null",
			input:    []byte{65, 0, 66, 0, 0, 0, 67, 0},
			expected: []byte{65, 0, 66, 0},
			size:     6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, size := ExtractUCS16String(tt.input)
			if !bytes.Equal(result, tt.expected) {
				t.Errorf("ExtractUCS16String() result = %v, want %v", result, tt.expected)
			}
			if size != tt.size {
				t.Errorf("ExtractUCS16String() size = %v, want %v", size, tt.size)
			}
		})
	}
}

func TestDecodeUTF16LE(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected string
	}{
		{
			name:     "Empty string",
			input:    []byte{},
			expected: "",
		},
		{
			name:     "Simple string",
			input:    []byte{65, 0, 116, 0, 116, 0, 101, 0, 109, 0, 112, 0, 116, 0, 32, 0, 49, 0},
			expected: "Attempt 1",
		},
		{
			name:     "UEFI string",
			input:    []byte{85, 0, 69, 0, 70, 0, 73, 0, 32, 0, 81, 0, 69, 0, 77, 0, 85, 0, 32, 0, 68, 0, 86, 0, 68, 0, 45, 0, 82, 0, 79, 0, 77, 0},
			expected: "UEFI QEMU DVD-ROM",
		},
		{
			name:     "Odd length",
			input:    []byte{65, 0, 66, 0, 67},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DecodeUTF16LE(tt.input)
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
