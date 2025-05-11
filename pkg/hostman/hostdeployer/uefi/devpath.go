package uefi

import (
    "encoding/binary"
    "fmt"
    "strings"
)

// DevicePathType constants
const (
    DevicePathTypeHardware    = 0x01
    DevicePathTypeACPI        = 0x02
    DevicePathTypeMessaging   = 0x03
    DevicePathTypeMedia       = 0x04
    DevicePathTypeBBS         = 0x05
    DevicePathTypeEnd         = 0x7F
)

// DevicePathSubType constants
const (
    // Hardware
    HardwareSubTypePCI        = 0x01
    
    // ACPI
    ACPISubTypeBasic          = 0x01
    
    // Messaging
    MessagingSubTypeSCSI      = 0x02
    
    // Media
    MediaSubTypeHardDrive     = 0x01
    MediaSubTypeCDROM         = 0x02
    MediaSubTypeFilePath      = 0x04
    
    // End
    EndSubTypeEndEntire       = 0xFF
)

// SCSIAddress represents a SCSI address (PUN, LUN)
type SCSIAddress struct {
    PUN uint16
    LUN uint16
}

// DevPath interface for device path elements
type DevPath interface {
    Type() byte
    SubType() byte
    String() string
    Address() interface{}
}

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

// Address returns the address information for the device path element
func (e *DevicePathElement) Address() interface{} {
    switch e.devType {
    case DevicePathTypeMessaging:
        switch e.subType {
        case MessagingSubTypeSCSI:
            if len(e.data) >= 4 {
                pun := binary.LittleEndian.Uint16(e.data[0:2])
                lun := binary.LittleEndian.Uint16(e.data[2:4])
                return SCSIAddress{PUN: pun, LUN: lun}
            }
        }
    }
    return nil
}

// String returns a string representation of the device path element
func (e *DevicePathElement) String() string {
    switch e.devType {
    case DevicePathTypeHardware:
        switch e.subType {
        case HardwareSubTypePCI:
            if len(e.data) >= 2 {
                return fmt.Sprintf("PCI(dev=%02x:%x)", e.data[0], e.data[1])
            }
            return "PCI()"
        }
        return fmt.Sprintf("Hw(subtype=0x%x)", e.subType)
        
    case DevicePathTypeACPI:
        switch e.subType {
        case ACPISubTypeBasic:
            return "PciRoot()"
        }
        return fmt.Sprintf("ACPI(subtype=0x%x)", e.subType)
        
    case DevicePathTypeMessaging:
        switch e.subType {
        case MessagingSubTypeSCSI:
            if len(e.data) >= 4 {
                pun := binary.LittleEndian.Uint16(e.data[0:2])
                lun := binary.LittleEndian.Uint16(e.data[2:4])
                return fmt.Sprintf("SCSI(pun=%d,lun=%d)", pun, lun)
            }
            return "SCSI()"
        }
        return fmt.Sprintf("Msg(subtype=0x%x)", e.subType)
        
    case DevicePathTypeMedia:
        switch e.subType {
        case MediaSubTypeHardDrive:
            return "HD()"
        case MediaSubTypeCDROM:
            return "CDROM()"
        case MediaSubTypeFilePath:
            // Parse UTF-16 file path
            path := DecodeUTF16LE(e.data)
            return fmt.Sprintf("FilePath(%s)", path)
        }
        return fmt.Sprintf("Media(subtype=0x%x)", e.subType)
        
    case DevicePathTypeEnd:
        if e.subType == EndSubTypeEndEntire {
            return "EndEntire"
        }
        return fmt.Sprintf("End(subtype=0x%x)", e.subType)
    }
    
    return fmt.Sprintf("DevPath(type=0x%x,subtype=0x%x)", e.devType, e.subType)
}

// ParseDevicePathElements parses a device path from binary data
func ParseDevicePathElements(data []byte) ([]DevPath, error) {
    var elements []DevPath
    
    pos := 0
    
    for pos < len(data) {
        // Check if we have at least 4 bytes (type, subtype, length)
        if pos+4 > len(data) {
            return elements, fmt.Errorf("truncated device path data")
        }
        
        // Parse header
        devType := data[pos]
        subType := data[pos+1]
        length := binary.LittleEndian.Uint16(data[pos+2 : pos+4])
        
        // Validate length
        if length < 4 || pos+int(length) > len(data) {
            return elements, fmt.Errorf("invalid device path element length")
        }
        
        // Extract element data
        elemData := make([]byte, length-4)
        copy(elemData, data[pos+4:pos+int(length)])
        
        // Create device path element
        element := &DevicePathElement{
            devType: devType,
            subType: subType,
            data:    elemData,
        }
        
        elements = append(elements, element)
        
        // Check if this is the end of the device path
        if devType == DevicePathTypeEnd && subType == EndSubTypeEndEntire {
            break
        }
        
        // Move to next element
        pos += int(length)
    }
    
    return elements, nil
}

// FormatDevicePath formats a device path as a string
func FormatDevicePath(devPaths []DevPath) string {
    var parts []string
    for _, elem := range devPaths {
        parts = append(parts, elem.String())
    }
    return strings.Join(parts, "/")
}

// DetermineDeviceType determines the device type from a device path
func DetermineDeviceType(devPaths []DevPath) string {
    for _, elem := range devPaths {
        // Check for CDROM
        if elem.Type() == DevicePathTypeMedia && elem.SubType() == MediaSubTypeCDROM {
            return "CDROM"
        }
        
        // Check for SCSI with PUN=1 (typically CDROM)
        if elem.Type() == DevicePathTypeMessaging && elem.SubType() == MessagingSubTypeSCSI {
            if addr, ok := elem.Address().(SCSIAddress); ok && addr.PUN == 1 {
                return "CDROM"
            }
        }
        
        // Check for HD
        if elem.Type() == DevicePathTypeMedia && elem.SubType() == MediaSubTypeHardDrive {
            return "HD"
        }
    }
    
    return "UNKNOWN"
} 