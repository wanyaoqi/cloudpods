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
            expected: "PCI(dev=03:1)",
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
            data:     []byte{0x5c, 0x00, 0x45, 0x00, 0x46, 0x00, 0x49, 0x00, 0x5c, 0x00, 0x42, 0x00, 0x4f, 0x00, 0x4f, 0x00, 0x54, 0x00},
            expected: "FilePath(\\EFI\\BOOT)",
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
    // Test data from a real UEFI device path
    testData, _ := hex.DecodeString("02010c00d041030a000000000101060001010301080001000000")
    
    elements := ParseDevicePathElements(testData)
    
    if len(elements) != 3 {
        t.Fatalf("Expected 3 device path elements, got %d", len(elements))
    }
    
    // Check first element (PciRoot)
    if elements[0].Type() != DevicePathTypeACPI || elements[0].SubType() != ACPISubTypeBasic {
        t.Errorf("First element is not ACPI Basic: type=%d, subtype=%d", elements[0].Type(), elements[0].SubType())
    }
    if elements[0].String() != "PciRoot()" {
        t.Errorf("First element string = %s, want PciRoot()", elements[0].String())
    }
    
    // Check second element (PCI)
    if elements[1].Type() != DevicePathTypeHardware || elements[1].SubType() != HardwareSubTypePCI {
        t.Errorf("Second element is not Hardware PCI: type=%d, subtype=%d", elements[1].Type(), elements[1].SubType())
    }
    
    // Check third element (SCSI)
    if elements[2].Type() != DevicePathTypeMessaging || elements[2].SubType() != MessagingSubTypeSCSI {
        t.Errorf("Third element is not Messaging SCSI: type=%d, subtype=%d", elements[2].Type(), elements[2].SubType())
    }
    if elements[2].String() != "SCSI(pun=1,lun=0)" {
        t.Errorf("Third element string = %s, want SCSI(pun=1,lun=0)", elements[2].String())
    }
}

func TestFormatDevicePath(t *testing.T) {
    // Create some test device path elements
    elem1 := &DevicePathElement{
        devType: DevicePathTypeACPI,
        subType: ACPISubTypeBasic,
        data:    []byte{0xd0, 0x41, 0x03, 0x0a, 0x00, 0x00, 0x00, 0x00},
    }
    
    elem2 := &DevicePathElement{
        devType: DevicePathTypeHardware,
        subType: HardwareSubTypePCI,
        data:    []byte{0x01, 0x03},
    }
    
    elem3 := &DevicePathElement{
        devType: DevicePathTypeMessaging,
        subType: MessagingSubTypeSCSI,
        data:    []byte{0x01, 0x00, 0x00, 0x00},
    }
    
    elements := []DevPath{elem1, elem2, elem3}
    
    result := FormatDevicePath(elements)
    expected := "PciRoot()/PCI(dev=03:1)/SCSI(pun=1,lun=0)"
    
    if result != expected {
        t.Errorf("FormatDevicePath() = %v, want %v", result, expected)
    }
}

func TestDetermineDeviceType(t *testing.T) {
    tests := []struct {
        name     string
        elements []DevPath
        expected string
    }{
        {
            name: "CDROM device",
            elements: []DevPath{
                &DevicePathElement{
                    devType: DevicePathTypeACPI,
                    subType: ACPISubTypeBasic,
                    data:    []byte{0xd0, 0x41, 0x03, 0x0a, 0x00, 0x00, 0x00, 0x00},
                },
                &DevicePathElement{
                    devType: DevicePathTypeMessaging,
                    subType: MessagingSubTypeSCSI,
                    data:    []byte{0x01, 0x00, 0x00, 0x00},
                    address: SCSIAddress{PUN: 1, LUN: 0},
                },
            },
            expected: "CDROM",
        },
        {
            name: "Hard drive",
            elements: []DevPath{
                &DevicePathElement{
                    devType: DevicePathTypeACPI,
                    subType: ACPISubTypeBasic,
                    data:    []byte{0xd0, 0x41, 0x03, 0x0a, 0x00, 0x00, 0x00, 0x00},
                },
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
            expected: "NETWORK",
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