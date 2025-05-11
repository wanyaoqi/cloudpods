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
    // 使用完整的设备路径数据，包括结束标记
    hexData := "02010c00d041030a00000000030208000100000000007fff0400"
    data, _ := hex.DecodeString(hexData)
    
    elements, err := ParseDevicePathElements(data)
    if err != nil {
        t.Fatalf("ParseDevicePathElements() error = %v", err)
    }
    
    if len(elements) != 3 {
        t.Fatalf("ParseDevicePathElements() returned %d elements, want 3", len(elements))
    }
    
    // Check first element (ACPI)
    if elements[0].Type() != DevicePathTypeACPI {
        t.Errorf("First element is not ACPI: type=%d", elements[0].Type())
    }
    if elements[0].SubType() != ACPISubTypeBasic {
        t.Errorf("First element is not ACPI Basic: subtype=%d", elements[0].SubType())
    }
    if elements[0].String() != "PciRoot()" {
        t.Errorf("First element string = %s, want PciRoot()", elements[0].String())
    }
    
    // Check second element (Messaging SCSI)
    if elements[1].Type() != DevicePathTypeMessaging {
        t.Errorf("Second element is not Messaging: type=%d", elements[1].Type())
    }
    if elements[1].SubType() != MessagingSubTypeSCSI {
        t.Errorf("Second element is not Messaging SCSI: subtype=%d", elements[1].SubType())
    }
    if elements[1].String() != "SCSI(pun=1,lun=0)" {
        t.Errorf("Second element string = %s, want SCSI(pun=1,lun=0)", elements[1].String())
    }
    
    // Check third element (End)
    if elements[2].Type() != DevicePathTypeEnd {
        t.Errorf("Third element is not End: type=%d", elements[2].Type())
    }
    if elements[2].SubType() != EndSubTypeEndEntire {
        t.Errorf("Third element is not End Entire: subtype=%d", elements[2].SubType())
    }
    if elements[2].String() != "End()" {
        t.Errorf("Third element string = %s, want End()", elements[2].String())
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