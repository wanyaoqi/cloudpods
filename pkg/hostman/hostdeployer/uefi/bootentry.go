package uefi

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strings"
)

type OvmfDevicePathType int

const (
	DEVICE_TYPE_UNKNOWN OvmfDevicePathType = 0
	DEVICE_TYPE_CDROM
	DEVICE_TYPE_IDE
	DEVICE_TYPE_SCSI
	DEVICE_TYPE_SCSI_CDROM
	DEVICE_TYPE_PCI
	DEVICE_TYPE_SATA
)

type BootEntry struct {
	ID       string               // Boot0000, Boot0001, etc.
	Name     string               // Entry title
	DevPaths []*DevicePathElement // Device path elements
	RawData  string               // Raw hex data
}

func (b *BootEntry) GetType() OvmfDevicePathType {
	lenElements := len(b.DevPaths)
	if lenElements == 0 {
		return DEVICE_TYPE_UNKNOWN
	}
	devElement := b.DevPaths[lenElements-1]
	// fetch last device path element type
	switch devElement.devType {
	case DevicePathTypeHardware:
		if devElement.subType == 0x01 {
			return DEVICE_TYPE_PCI
		}
	case DevicePathTypeMessaging:
		switch devElement.subType {
		case 0x01:
			if strings.HasPrefix(b.Name, "UEFI QEMU DVD-ROM") {
				return DEVICE_TYPE_CDROM
			} else if strings.HasPrefix(b.Name, "UEFI QEMU HARDDISK") {
				return DEVICE_TYPE_IDE
			}
		case 0x02:
			if strings.HasPrefix(b.Name, "UEFI QEMU QEMU CD-ROM") {
				return DEVICE_TYPE_SCSI_CDROM
			} else if strings.HasPrefix(b.Name, "UEFI QEMU QEMU HARDDISK") {
				return DEVICE_TYPE_SCSI
			}
		case 0x12:
			return DEVICE_TYPE_SATA
		}
	}

	return DEVICE_TYPE_UNKNOWN
}

// ParseBootEntryData parses a boot entry from hex data
func ParseBootEntryData(hexData string) (string, []*DevicePathElement, error) {
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
		return name, []*DevicePathElement{}, nil
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
func ParseBootOrder(hexData string) ([]uint16, error) {
	// Decode hex data
	data, err := hex.DecodeString(hexData)
	if err != nil {
		return nil, fmt.Errorf("failed to decode hex data: %v", err)
	}

	// Check data length
	if len(data) == 0 {
		return []uint16{}, nil
	}

	// Check if data length is valid (must be even)
	if len(data)%2 != 0 {
		return nil, fmt.Errorf("invalid boot order data length (must be even)")
	}

	// Parse boot order (2 bytes per entry)
	var bootOrder []uint16
	for i := 0; i < len(data); i += 2 {
		entryNum := binary.LittleEndian.Uint16(data[i : i+2])
		bootOrder = append(bootOrder, entryNum)
	}

	return bootOrder, nil
}

func ParseBootentryToBootorder(entry string) (uint16, error) {
	if !strings.HasPrefix(entry, "Boot") {
		return 0, fmt.Errorf("unknonw boot entry %s", entry)
	}
	hexData := entry[4:]
	// Decode hex data
	data, err := hex.DecodeString(hexData)
	if err != nil {
		return 0, fmt.Errorf("failed to decode hex data %s: %v", hexData, err)
	}
	return binary.BigEndian.Uint16(data), nil
}

// BuildBootOrderHex builds a hex string from boot order list
func BuildBootOrderHex(bootOrder []uint16) string {
	// Allocate space for boot order (2 bytes per entry)
	data := make([]byte, len(bootOrder)*2)
	for i, entry := range bootOrder {
		// Write little-endian uint16
		binary.LittleEndian.PutUint16(data[i*2:], entry)
	}

	// Return hex string
	return hex.EncodeToString(data)
}
