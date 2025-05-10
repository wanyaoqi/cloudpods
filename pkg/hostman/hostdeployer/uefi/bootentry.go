package uefi

import (
    "encoding/binary"
    "encoding/hex"
    "fmt"
)

// BootEntry constants
const (
    LoadOptionActive         = 0x00000001
    LoadOptionForceReconnect = 0x00000002
    LoadOptionHidden         = 0x00000008
    LoadOptionCategoryMask   = 0x00001F00
    LoadOptionCategoryBoot   = 0x00000000
    LoadOptionCategoryApp    = 0x00000100
)

// BootEntry represents a UEFI boot entry
type BootEntry struct {
    ID       string    // Boot0000, Boot0001, etc.
    Name     string    // Entry title
    Path     string    // Formatted device path
    DevPaths []DevPath // Device path elements
    DevType  string    // "HD", "CDROM", "NETWORK", etc.
    RawData  string    // Original hex data
}

// ParseBootEntryData parses a boot entry from hex data
func ParseBootEntryData(hexData string) (string, []DevPath, error) {
    // Convert hex string to byte array
    data, err := hex.DecodeString(hexData)
    if err != nil {
        return "", nil, fmt.Errorf("failed to parse hex data: %v", err)
    }
    
    // Check if data is long enough
    if len(data) < 8 {
        return "", nil, fmt.Errorf("data too short")
    }
    
    // Parse attributes and device path size
    pathSize := binary.LittleEndian.Uint16(data[4:6])
    
    // Parse title (UCS-16 encoded string)
    titleBytes, titleSize := ExtractUCS16String(data[6:])
    title := DecodeUTF16LE(titleBytes)
    
    // Parse device path
    pathStart := 6 + titleSize
    pathEnd := pathStart + int(pathSize)
    if pathEnd > len(data) {
        return "", nil, fmt.Errorf("device path data out of range")
    }
    
    devicePaths := ParseDevicePathElements(data[pathStart:pathEnd])
    
    return title, devicePaths, nil
}

// ParseBootOrderData parses boot order from hex data
func ParseBootOrderData(hexData string) ([]string, error) {
    // Convert hex string to byte array
    data, err := hex.DecodeString(hexData)
    if err != nil {
        return nil, fmt.Errorf("failed to parse hex data: %v", err)
    }
    
    // Check if data length is even (each boot entry is 2 bytes)
    if len(data)%2 != 0 {
        return nil, fmt.Errorf("data length not multiple of 2")
    }
    
    // Parse boot order
    var bootList []string
    for i := 0; i < len(data); i += 2 {
        nr := binary.LittleEndian.Uint16(data[i : i+2])
        bootList = append(bootList, fmt.Sprintf("%04x", nr))
    }
    
    return bootList, nil
}

// BuildBootOrderHex builds a hex string from boot order list
func BuildBootOrderHex(bootOrder []string) (string, error) {
    // Allocate space for boot order (2 bytes per entry)
    data := make([]byte, len(bootOrder)*2)
    
    for i, entry := range bootOrder {
        // Parse hex string to uint16
        var val uint16
        _, err := fmt.Sscanf(entry, "%04x", &val)
        if err != nil {
            return "", fmt.Errorf("failed to parse boot entry %s: %v", entry, err)
        }
        
        // Write little-endian uint16
        binary.LittleEndian.PutUint16(data[i*2:], val)
    }
    
    // Return hex string
    return hex.EncodeToString(data), nil
} 