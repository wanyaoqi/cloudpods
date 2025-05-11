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
    ACPISubTypeExtended       = 0x02
    ACPISubTypeADR            = 0x03
    ACPISubTypeGOP            = 0x04
    
    // Messaging
    MessagingSubTypeSCSI      = 0x02
    MessagingSubTypeUSB       = 0x05
    MessagingSubTypeMAC       = 0x0B
    MessagingSubTypeIPv4      = 0x0C
    MessagingSubTypeIPv6      = 0x0D
    MessagingSubTypeSATA      = 0x12
    
    // Media
    MediaSubTypeHardDrive     = 0x01
    MediaSubTypeCDROM         = 0x02
    MediaSubTypeFilePath      = 0x04
    
    // End
    EndSubTypeEndEntire       = 0xFF
    EndSubTypeEndThis         = 0x01
)

// DevPath is the interface for device path elements
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

// Address returns the device path address
func (e *DevicePathElement) Address() interface{} {
    switch e.devType {
    case DevicePathTypeHardware:
        if e.subType == HardwareSubTypePCI {
            if len(e.data) >= 2 {
                return PCIAddress{
                    Device:   e.data[0],
                    Function: e.data[1],
                }
            }
        }
    case DevicePathTypeMessaging:
        if e.subType == MessagingSubTypeSCSI {
            if len(e.data) >= 4 {
                return SCSIAddress{
                    PUN: binary.LittleEndian.Uint16(e.data[0:2]),
                    LUN: binary.LittleEndian.Uint16(e.data[2:4]),
                }
            }
        }
    }
    return nil
}

// String returns a string representation of the device path element
func (e *DevicePathElement) String() string {
    switch e.devType {
    case DevicePathTypeHardware:
        if e.subType == HardwareSubTypePCI {
            if len(e.data) >= 2 {
                return fmt.Sprintf("PCI(dev=%02x:%x)", e.data[0], e.data[1])
            }
        }
        return fmt.Sprintf("Hw(subtype=0x%x)", e.subType)
    case DevicePathTypeACPI:
        if e.subType == ACPISubTypeBasic {
            return "PciRoot()"
        }
        return fmt.Sprintf("ACPI(subtype=0x%x)", e.subType)
    case DevicePathTypeMessaging:
        if e.subType == MessagingSubTypeSCSI {
            if len(e.data) >= 4 {
                pun := binary.LittleEndian.Uint16(e.data[0:2])
                lun := binary.LittleEndian.Uint16(e.data[2:4])
                return fmt.Sprintf("SCSI(pun=%d,lun=%d)", pun, lun)
            }
        }
        return fmt.Sprintf("Msg(subtype=0x%x)", e.subType)
    case DevicePathTypeMedia:
        if e.subType == MediaSubTypeFilePath {
            return "FilePath()"
        } else if e.subType == MediaSubTypeHardDrive {
            return "HD()"
        } else if e.subType == MediaSubTypeCDROM {
            return "CDROM()"
        }
        return fmt.Sprintf("Media(subtype=0x%x)", e.subType)
    case DevicePathTypeEnd:
        return "End()"
    default:
        return fmt.Sprintf("Unknown(type=0x%x,subtype=0x%x)", e.devType, e.subType)
    }
}

// PCIAddress represents a PCI device address
type PCIAddress struct {
    Device   byte
    Function byte
}

// SCSIAddress represents a SCSI device address
type SCSIAddress struct {
    PUN uint16
    LUN uint16
}

// ParseDevicePathElements parses a device path from binary data
func ParseDevicePathElements(data []byte) ([]DevPath, error) {
    var elements []DevPath
    
    // Check minimum length
    if len(data) < 4 {
        return nil, fmt.Errorf("device path data too short")
    }
    
    // Parse device path elements
    pos := 0
    for pos < len(data) {
        // Check if we have enough data for the header
        if pos+4 > len(data) {
            return nil, fmt.Errorf("truncated device path data")
        }
        
        // Parse header
        devType := data[pos]
        subType := data[pos+1]
        length := binary.LittleEndian.Uint16(data[pos+2:pos+4])
        
        // Validate length
        if length < 4 {
            return nil, fmt.Errorf("invalid device path element length")
        }
        
        // Check if we have enough data for the element
        if pos+int(length) > len(data) {
            return nil, fmt.Errorf("truncated device path element")
        }
        
        // Create element
        element := &DevicePathElement{
            devType: devType,
            subType: subType,
            data:    data[pos+4:pos+int(length)],
        }
        
        // Add element to list
        elements = append(elements, element)
        
        // Check if this is the end of the device path
        if devType == DevicePathTypeEnd {
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