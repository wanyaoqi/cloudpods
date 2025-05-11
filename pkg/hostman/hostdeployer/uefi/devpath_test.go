package uefi

import (
    "encoding/hex"
    "testing"
)

func TestDevicePathElement_String(t *testing.T) {
    tests := []struct {
        name     string
        devType  byte
        subType  byte
        data     []byte
        expected string
    }{
        {
            name:     "PCI device",
            devType:  DevicePathTypeHardware,
            subType:  HardwareSubTypePCI,
            data:     []byte{0x01, 0x03},
            expected: "PCI(dev=01:3)",
        },
        {
            name:     "PciRoot",
            devType:  DevicePathTypeACPI,
            subType:  ACPISubTypeBasic,
            data:     []byte{0xd0, 0x41, 0x03, 0x0a, 0x00, 0x00, 0x00, 0x00},
            expected: "PciRoot()",
        },
        {
            name:     "SCSI device",
            devType:  DevicePathTypeMessaging,
            subType:  MessagingSubTypeSCSI,
            data:     []byte{0x01, 0x00, 0x00, 0x00},
            expected: "SCSI(pun=1,lun=0)",
        },
        {
            name:     "FilePath",
            devType:  DevicePathTypeMedia,
            subType:  MediaSubTypeFilePath,
            data:     []byte{0x5c, 0x00, 0x45, 0x00, 0x46, 0x00, 0x49, 0x00},
            expected: "FilePath()",
        },
        {
            name:     "End",
            devType:  DevicePathTypeEnd,
            subType:  EndSubTypeEndEntire,
            data:     []byte{},
            expected: "End()",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            elem := &DevicePathElement{
                devType: tt.devType,
                subType: tt.subType,
                data:    tt.data,
            }
            
            result := elem.String()
            if result != tt.expected {
                t.Errorf("DevicePathElement.String() = %v, want %v", result, tt.expected)
            }
        })
    }
}

func TestParseDevicePathElements(t *testing.T) {
    // Valid device path
    validPath := []byte{
        // PciRoot
        0x02, 0x01, 0x0c, 0x00, 0xd0, 0x41, 0x03, 0x0a, 0x00, 0x00, 0x00, 0x00,
        // PCI device
        0x01, 0x01, 0x06, 0x00, 0x01, 0x01,
        // End
        0x7f, 0xff, 0x04, 0x00
    }
    
    // Test valid path
    elements, err := ParseDevicePathElements(validPath)
    if err != nil {
        t.Errorf("ParseDevicePathElements() error = %v", err)
        return
    }
    
    if len(elements) != 3 {
        t.Errorf("Expected 3 elements, got %d", len(elements))
        return
    }
    
    // Test empty path
    _, err = ParseDevicePathElements([]byte{})
    if err == nil {
        t.Errorf("ParseDevicePathElements() error = nil, expected error for empty path")
    }
    
    // Test truncated path
    _, err = ParseDevicePathElements([]byte{0x01, 0x01})
    if err == nil {
        t.Errorf("ParseDevicePathElements() error = nil, expected error for truncated path")
    }
    
    // Test invalid element length
    invalidLength := []byte{0x01, 0x01, 0x02, 0x00}
    _, err = ParseDevicePathElements(invalidLength)
    if err == nil {
        t.Errorf("ParseDevicePathElements() error = nil, expected error for invalid element length")
    }
}

func TestDetermineDeviceType(t *testing.T) {
    tests := []struct {
        name     string
        elements []DevPath
        expected string
    }{
        {
            name: "CDROM device (Media)",
            elements: []DevPath{
                &DevicePathElement{
                    devType: DevicePathTypeMedia,
                    subType: MediaSubTypeCDROM,
                    data:    []byte{},
                },
            },
            expected: "CDROM",
        },
        {
            name: "CDROM device (SCSI)",
            elements: []DevPath{
                &DevicePathElement{
                    devType: DevicePathTypeMessaging,
                    subType: MessagingSubTypeSCSI,
                    data:    []byte{0x01, 0x00, 0x00, 0x00},
                },
            },
            expected: "CDROM",
        },
        {
            name: "HD device",
            elements: []DevPath{
                &DevicePathElement{
                    devType: DevicePathTypeMedia,
                    subType: MediaSubTypeHardDrive,
                    data:    []byte{0x01, 0x00, 0x00, 0x00},
                },
            },
            expected: "HD",
        },
        {
            name: "Network device",
            elements: []DevPath{
                &DevicePathElement{
                    devType: DevicePathTypeMessaging,
                    subType: MessagingSubTypeMAC,
                    data:    []byte{},
                },
            },
            expected: "UNKNOWN",
        },
        {
            name: "Unknown device",
            elements: []DevPath{
                &DevicePathElement{
                    devType: DevicePathTypeACPI,
                    subType: ACPISubTypeGOP,
                    data:    []byte{},
                },
            },
            expected: "UNKNOWN",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := DetermineDeviceType(tt.elements)
            if result != tt.expected {
                t.Errorf("DetermineDeviceType() = %v, want %v", result, tt.expected)
            }
        })
    }
}