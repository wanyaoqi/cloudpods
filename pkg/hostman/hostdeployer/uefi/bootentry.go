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
    DevType  string    // Device type (HD, CDROM, NETWORK, etc.)
    RawData  string    // Raw hex data
}

// ParseBootEntryData parses a boot entry from hex data
func ParseBootEntryData(hexData string) (string, []DevPath, error) {
    // Decode hex data
    data, err := hex.DecodeString(hexData)
    if err != nil {
        return "", nil, fmt.Errorf("failed to decode hex data: %v", err)
    }
    
    // Check minimum length
    if len(data) < 8 {
        return "", nil, fmt.Errorf("data too short")
    }
    
    // Parse attributes and path list length
    // attributes := binary.LittleEndian.Uint32(data[0:4])
    pathListLen := binary.LittleEndian.Uint16(data[4:6])
    
    // Extract description string
    descData := data[6:]
    descBytes, strLen := ExtractUCS16String(descData)
    name := DecodeUTF16LE(descBytes)
    
    // Calculate path list start
    pathListStart := 6 + uint32(strLen)
    
    // Check if we have enough data for the path list
    if pathListLen == 0 {
        return name, []DevPath{}, nil
    }
    
    if uint32(len(data)) < pathListStart+uint32(pathListLen) {
        return name, nil, fmt.Errorf("invalid path list length")
    }
    
    // Extract path list
    pathListData := data[pathListStart : pathListStart+uint32(pathListLen)]
    
    // Parse device path elements
    devPaths, err := ParseDevicePathElements(pathListData)
    if err != nil {
        return name, nil, fmt.Errorf("failed to parse device path: %v", err)
    }
    
    return name, devPaths, nil
}

// ParseBootOrder parses a boot order from hex data
func ParseBootOrder(hexData string) ([]string, error) {
    // Decode hex data
    data, err := hex.DecodeString(hexData)
    if err != nil {
        return nil, fmt.Errorf("failed to decode hex data: %v", err)
    }
    
    // Check data length
    if len(data) == 0 {
        return []string{}, nil
    }
    
    // Check if data length is valid (must be even)
    if len(data) % 2 != 0 {
        return nil, fmt.Errorf("invalid boot order data length (must be even)")
    }
    
    // Parse boot order (2 bytes per entry)
    var bootOrder []string
    for i := 0; i < len(data); i += 2 {
        entryNum := binary.LittleEndian.Uint16(data[i : i+2])
        bootOrder = append(bootOrder, fmt.Sprintf("%04x", entryNum))
    }
    
    return bootOrder, nil
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