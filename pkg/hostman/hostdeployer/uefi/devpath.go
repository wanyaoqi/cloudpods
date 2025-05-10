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
    ACPISubTypeGOP            = 0x03
    
    // Messaging
    MessagingSubTypeSCSI      = 0x02
    MessagingSubTypeUSB       = 0x05
    MessagingSubTypeMAC       = 0x0B
    MessagingSubTypeIPv4      = 0x0C
    MessagingSubTypeIPv6      = 0x0D
    MessagingSubTypeSATA      = 0x12
    MessagingSubTypeISCSI     = 0x13
    MessagingSubTypeURI       = 0x18
    MessagingSubTypeDNS       = 0x1F
    
    // Media
    MediaSubTypeHardDrive     = 0x01
    MediaSubTypeFilePath      = 0x04
    MediaSubTypeFvFileName    = 0x06
    MediaSubTypeFvName        = 0x07
)

// DevPath is the interface for device path elements
type DevPath interface {
    String() string
    Type() byte
    SubType() byte
    Address() interface{}
}

// PCIAddress represents a PCI device address
type PCIAddress struct {
    Function byte
    Device   byte
}

// SCSIAddress represents a SCSI device address
type SCSIAddress struct {
    PUN uint16
    LUN uint16
}

// SATAAddress represents a SATA device address
type SATAAddress struct {
    Port  uint16
    PMult uint16
    LUN   uint16
}

// DevicePathElement implements the DevPath interface
type DevicePathElement struct {
    devType  byte
    subType  byte
    data     []byte
    address  interface{}
}

// String returns a formatted string representation of the device path element
func (d *DevicePathElement) String() string {
    switch d.devType {
    case DevicePathTypeHardware:
        return formatHardwareDevicePath(d.subType, d.data)
    case DevicePathTypeACPI:
        return formatACPIDevicePath(d.subType, d.data)
    case DevicePathTypeMessaging:
        return formatMessagingDevicePath(d.subType, d.data)
    case DevicePathTypeMedia:
        return formatMediaDevicePath(d.subType, d.data)
    case DevicePathTypeEnd:
        return "EndPath"
    default:
        return fmt.Sprintf("Unknown(type=0x%x,subtype=0x%x)", d.devType, d.subType)
    }
}

// Type returns the device path type
func (d *DevicePathElement) Type() byte {
    return d.devType
}

// SubType returns the device path subtype
func (d *DevicePathElement) SubType() byte {
    return d.subType
}

// Address returns the device path address
func (d *DevicePathElement) Address() interface{} {
    return d.address
}

// ParseDevicePathElements parses a device path from binary data
func ParseDevicePathElements(data []byte) []DevPath {
    if len(data) < 4 {
        return nil
    }
    
    var elements []DevPath
    pos := 0
    
    for pos < len(data) {
        if pos+4 > len(data) {
            break
        }
        
        devType := data[pos]
        subType := data[pos+1]
        size := binary.LittleEndian.Uint16(data[pos+2:pos+4])
        
        if size < 4 || pos+int(size) > len(data) {
            break
        }
        
        elemData := data[pos+4:pos+int(size)]
        elem := &DevicePathElement{
            devType: devType,
            subType: subType,
            data:    elemData,
        }
        
        // Extract address information based on type/subtype
        extractAddress(elem)
        
        elements = append(elements, elem)
        
        pos += int(size)
        
        // End of device path
        if devType == DevicePathTypeEnd {
            break
        }
    }
    
    return elements
}

// extractAddress extracts address information from device path element
func extractAddress(elem *DevicePathElement) {
    if elem.devType == DevicePathTypeHardware {
        if elem.subType == HardwareSubTypePCI && len(elem.data) >= 2 {
            elem.address = PCIAddress{
                Function: elem.data[0],
                Device:   elem.data[1],
            }
        }
    } else if elem.devType == DevicePathTypeMessaging {
        if elem.subType == MessagingSubTypeSCSI && len(elem.data) >= 4 {
            elem.address = SCSIAddress{
                PUN: binary.LittleEndian.Uint16(elem.data[0:2]),
                LUN: binary.LittleEndian.Uint16(elem.data[2:4]),
            }
        } else if elem.subType == MessagingSubTypeSATA && len(elem.data) >= 6 {
            elem.address = SATAAddress{
                Port:  binary.LittleEndian.Uint16(elem.data[0:2]),
                PMult: binary.LittleEndian.Uint16(elem.data[2:4]),
                LUN:   binary.LittleEndian.Uint16(elem.data[4:6]),
            }
        }
    }
}

// formatHardwareDevicePath formats a hardware device path element
func formatHardwareDevicePath(subType byte, data []byte) string {
    if subType == HardwareSubTypePCI && len(data) >= 2 {
        function := data[0]
        device := data[1]
        return fmt.Sprintf("PCI(dev=%02x:%x)", device, function)
    }
    return fmt.Sprintf("HW(subtype=0x%x)", subType)
}

// formatACPIDevicePath formats an ACPI device path element
func formatACPIDevicePath(subType byte, data []byte) string {
    if subType == ACPISubTypeBasic && len(data) >= 8 {
        hid := binary.LittleEndian.Uint32(data[0:4])
        uid := binary.LittleEndian.Uint32(data[4:8])
        if hid == 0xa0341d0 {
            return "PciRoot()"
        }
        return fmt.Sprintf("ACPI(hid=0x%x,uid=0x%x)", hid, uid)
    }
    if subType == ACPISubTypeGOP && len(data) >= 4 {
        adr := binary.LittleEndian.Uint32(data[0:4])
        return fmt.Sprintf("GOP(adr=0x%x)", adr)
    }
    return fmt.Sprintf("ACPI(subtype=0x%x)", subType)
}

// formatMessagingDevicePath formats a messaging device path element
func formatMessagingDevicePath(subType byte, data []byte) string {
    switch subType {
    case MessagingSubTypeSCSI:
        if len(data) >= 4 {
            pun := binary.LittleEndian.Uint16(data[0:2])
            lun := binary.LittleEndian.Uint16(data[2:4])
            return fmt.Sprintf("SCSI(pun=%d,lun=%d)", pun, lun)
        }
    case MessagingSubTypeUSB:
        if len(data) >= 2 {
            port := data[0]
            return fmt.Sprintf("USB(port=%d)", port)
        }
    case MessagingSubTypeMAC:
        return "MAC()"
    case MessagingSubTypeIPv4:
        return "IPv4()"
    case MessagingSubTypeIPv6:
        return "IPv6()"
    case MessagingSubTypeSATA:
        if len(data) >= 6 {
            port := binary.LittleEndian.Uint16(data[0:2])
            return fmt.Sprintf("SATA(port=%d)", port)
        }
    case MessagingSubTypeURI:
        return fmt.Sprintf("URI(%s)", string(data))
    case MessagingSubTypeDNS:
        return "DNS()"
    }
    return fmt.Sprintf("Msg(subtype=0x%x)", subType)
}

// formatMediaDevicePath formats a media device path element
func formatMediaDevicePath(subType byte, data []byte) string {
    switch subType {
    case MediaSubTypeHardDrive:
        if len(data) >= 20 {
            pnr := binary.LittleEndian.Uint32(data[0:4])
            return fmt.Sprintf("Partition(nr=%d)", pnr)
        }
    case MediaSubTypeFilePath:
        path := DecodeUTF16LE(data)
        return fmt.Sprintf("FilePath(%s)", path)
    case MediaSubTypeFvFileName:
        return "FvFileName()"
    case MediaSubTypeFvName:
        return "FvName()"
    }
    return fmt.Sprintf("Media(subtype=0x%x)", subType)
}

// FormatDevicePath formats a list of device path elements as a string
func FormatDevicePath(elements []DevPath) string {
    var parts []string
    for _, elem := range elements {
        parts = append(parts, elem.String())
    }
    return strings.Join(parts, "/")
}

// DetermineDeviceType determines the device type from device path elements
func DetermineDeviceType(elements []DevPath) string {
    for _, elem := range elements {
        // Check for CDROM
        if elem.Type() == DevicePathTypeMessaging && elem.SubType() == MessagingSubTypeSCSI {
            if addr, ok := elem.Address().(SCSIAddress); ok && addr.PUN == 1 {
                return "CDROM"
            }
        }
        
        // Check for HD
        if elem.Type() == DevicePathTypeMedia && elem.SubType() == MediaSubTypeHardDrive {
            return "HD"
        }
        
        // Check for Network
        if elem.Type() == DevicePathTypeMessaging && 
           (elem.SubType() == MessagingSubTypeMAC || 
            elem.SubType() == MessagingSubTypeIPv4 || 
            elem.SubType() == MessagingSubTypeIPv6) {
            return "NETWORK"
        }
    }
    
    return "UNKNOWN"
} 