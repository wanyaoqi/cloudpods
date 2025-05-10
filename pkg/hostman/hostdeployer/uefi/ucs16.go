package uefi

import (
	"unicode/utf16"
)

// ExtractUCS16String extracts a UCS-16 encoded string from data until null terminator
func ExtractUCS16String(data []byte) ([]byte, int) {
	var result []byte
	pos := 0
	
	for pos+1 < len(data) {
		// Check for null terminator
		if data[pos] == 0 && data[pos+1] == 0 {
			break
		}
		result = append(result, data[pos], data[pos+1])
		pos += 2
	}
	
	// Include null terminator in size
	return result, len(result) + 2
}

// DecodeUTF16LE decodes UTF-16LE bytes to a string
func DecodeUTF16LE(b []byte) string {
	if len(b)%2 != 0 {
		return ""
	}
	
	u16s := make([]uint16, 0, len(b)/2)
	for i := 0; i < len(b); i += 2 {
		u16s = append(u16s, uint16(b[i]) | (uint16(b[i+1])<<8))
	}
	return string(utf16.Decode(u16s))
}

// EncodeUTF16LE encodes a string to UTF-16LE bytes
func EncodeUTF16LE(s string) []byte {
	u16s := utf16.Encode([]rune(s))
	bytes := make([]byte, len(u16s)*2)
	for i, u16 := range u16s {
		bytes[i*2] = byte(u16)
		bytes[i*2+1] = byte(u16 >> 8)
	}
	return bytes
} 