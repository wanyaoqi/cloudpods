package uefi

import (
	"encoding/binary"
	"fmt"
)

// DevicePathType constants
const (
	DevicePathTypeHardware  = 0x01
	DevicePathTypeACPI      = 0x02
	DevicePathTypeMessaging = 0x03
	DevicePathTypeMedia     = 0x04
	DevicePathTypeEnd       = 0x7F
)

// DevicePathElement represents a UEFI device path element
type DevicePathElement struct {
	devType byte
	subType byte
	data    []byte
}

// Type returns the device path type
func (e *DevicePathElement) Type() byte {
	return e.devType
}

// SubType returns the device path subtype
func (e *DevicePathElement) SubType() byte {
	return e.subType
}

// ParseDevicePathElements parses a device path from binary data
func ParseDevicePathElements(data []byte) ([]*DevicePathElement, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty device path data")
	}

	var elements []*DevicePathElement
	pos := 0

	for pos < len(data) {
		// Check if we have enough data for the header
		if pos+4 > len(data) {
			return nil, fmt.Errorf("truncated device path data")
		}

		// Parse header
		devType := data[pos]
		subType := data[pos+1]
		length := binary.LittleEndian.Uint16(data[pos+2 : pos+4])

		// Validate length
		if length < 4 {
			return nil, fmt.Errorf("invalid device path element length")
		}

		// Check if we have enough data for the element
		if pos+int(length) > len(data) {
			return nil, fmt.Errorf("truncated device path element")
		}

		// Check if this is the end of the device path
		if devType == DevicePathTypeEnd {
			break
		}

		// Create element
		element := &DevicePathElement{
			devType: devType,
			subType: subType,
			data:    data[pos+4 : pos+int(length)],
		}
		elements = append(elements, element)

		// Move to next element
		pos += int(length)
	}

	return elements, nil
}
