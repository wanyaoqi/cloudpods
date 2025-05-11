package uefi

import (
    "reflect"
    "testing"
)

func TestContains(t *testing.T) {
    tests := []struct {
        name     string
        s        string
        substrs  []string
        expected bool
    }{
        {
            name:     "Contains one substring",
            s:        "UEFI QEMU DVD-ROM",
            substrs:  []string{"DVD"},
            expected: true,
        },
        {
            name:     "Contains multiple substrings",
            s:        "UEFI QEMU DVD-ROM",
            substrs:  []string{"CD", "DVD", "BLU-RAY"},
            expected: true,
        },
        {
            name:     "Contains no substrings",
            s:        "UEFI QEMU DVD-ROM",
            substrs:  []string{"CD", "BLU-RAY"},
            expected: false,
        },
        {
            name:     "Empty string",
            s:        "",
            substrs:  []string{"DVD"},
            expected: false,
        },
        {
            name:     "Empty substrings",
            s:        "UEFI QEMU DVD-ROM",
            substrs:  []string{},
            expected: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := Contains(tt.s, tt.substrs...)
            if result != tt.expected {
                t.Errorf("Contains() = %v, want %v", result, tt.expected)
            }
        })
    }
}

func TestMatchBootEntries(t *testing.T) {
    entries := []BootEntry{
        {
            ID:      "Boot0000",
            Name:    "UEFI QEMU HARDDISK QM00001",
            Path:    "PciRoot()/PCI(0x1,0x1)/SCSI(0,0)/HD(1,GPT)",
            DevType: "HD",
        },
        {
            ID:      "Boot0001",
            Name:    "UEFI QEMU DVD-ROM QM00003",
            Path:    "PciRoot()/PCI(0x1,0x1)/SCSI(1,0)/CDROM(1)",
            DevType: "CDROM",
        },
        {
            ID:      "Boot0002",
            Name:    "UEFI QEMU HARDDISK QM00002",
            Path:    "PciRoot()/PCI(0x1,0x1)/SCSI(0,1)/HD(1,GPT)",
            DevType: "HD",
        },
    }

    tests := []struct {
        name              string
        diskPaths         []string
        cdromPaths        []string
        expectedDiskCount int
        expectedCDCount   int
    }{
        {
            name:              "Match all",
            diskPaths:         []string{},
            cdromPaths:        []string{},
            expectedDiskCount: 2,
            expectedCDCount:   1,
        },
        {
            name:              "Match specific disk",
            diskPaths:         []string{"QM00001"},
            cdromPaths:        []string{},
            expectedDiskCount: 1,
            expectedCDCount:   1,
        },
        {
            name:              "Match specific CDROM",
            diskPaths:         []string{},
            cdromPaths:        []string{"QM00003"},
            expectedDiskCount: 2,
            expectedCDCount:   1,
        },
        {
            name:              "Match by path",
            diskPaths:         []string{"SCSI(0,1)"},
            cdromPaths:        []string{},
            expectedDiskCount: 1,
            expectedCDCount:   1,
        },
        {
            name:              "No matches",
            diskPaths:         []string{"NONEXISTENT"},
            cdromPaths:        []string{"NONEXISTENT"},
            expectedDiskCount: 0,
            expectedCDCount:   0,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            diskEntries, cdromEntries := MatchBootEntries(entries, tt.diskPaths, tt.cdromPaths)
            
            if len(diskEntries) != tt.expectedDiskCount {
                t.Errorf("MatchBootEntries() disk count = %v, want %v", len(diskEntries), tt.expectedDiskCount)
            }
            
            if len(cdromEntries) != tt.expectedCDCount {
                t.Errorf("MatchBootEntries() CDROM count = %v, want %v", len(cdromEntries), tt.expectedCDCount)
            }
        })
    }
}

func TestBuildBootOrder(t *testing.T) {
    diskEntries := []BootEntry{
        {
            ID:      "Boot0000",
            Name:    "UEFI QEMU HARDDISK QM00001",
            Path:    "PciRoot()/PCI(0x1,0x1)/SCSI(0,0)/HD(1,GPT)",
            DevType: "HD",
        },
        {
            ID:      "Boot0002",
            Name:    "UEFI QEMU HARDDISK QM00002",
            Path:    "PciRoot()/PCI(0x1,0x1)/SCSI(0,1)/HD(1,GPT)",
            DevType: "HD",
        },
    }
    
    cdromEntries := []BootEntry{
        {
            ID:      "Boot0001",
            Name:    "UEFI QEMU DVD-ROM QM00003",
            Path:    "PciRoot()/PCI(0x1,0x1)/SCSI(1,0)/CDROM(1)",
            DevType: "CDROM",
        },
    }

    tests := []struct {
        name           string
        diskPriority   int32
        cdromPriority  int32
        expectedOrder  []string
    }{
        {
            name:           "CDROM first",
            diskPriority:   1,
            cdromPriority:  2,
            expectedOrder:  []string{"0001", "0000", "0002"},
        },
        {
            name:           "Disk first",
            diskPriority:   2,
            cdromPriority:  1,
            expectedOrder:  []string{"0000", "0002", "0001"},
        },
        {
            name:           "Only CDROM",
            diskPriority:   0,
            cdromPriority:  1,
            expectedOrder:  []string{"0001"},
        },
        {
            name:           "Only disk",
            diskPriority:   1,
            cdromPriority:  0,
            expectedOrder:  []string{"0000", "0002"},
        },
        {
            name:           "No boot devices",
            diskPriority:   0,
            cdromPriority:  0,
            expectedOrder:  []string{},
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            order := BuildBootOrder(diskEntries, cdromEntries, tt.diskPriority, tt.cdromPriority)
            
            if !reflect.DeepEqual(order, tt.expectedOrder) {
                t.Errorf("BuildBootOrder() = %v, want %v", order, tt.expectedOrder)
            }
        })
    }
}

func TestReorderBootEntries(t *testing.T) {
    entries := []BootEntry{
        {
            ID:      "Boot0000",
            Name:    "UEFI QEMU HARDDISK QM00001",
            Path:    "PciRoot()/PCI(0x1,0x1)/SCSI(0,0)/HD(1,GPT)",
            DevType: "HD",
        },
        {
            ID:      "Boot0001",
            Name:    "UEFI QEMU DVD-ROM QM00003",
            Path:    "PciRoot()/PCI(0x1,0x1)/SCSI(1,0)/CDROM(1)",
            DevType: "CDROM",
        },
        {
            ID:      "Boot0002",
            Name:    "UEFI QEMU HARDDISK QM00002",
            Path:    "PciRoot()/PCI(0x1,0x1)/SCSI(0,1)/HD(1,GPT)",
            DevType: "HD",
        },
    }

    tests := []struct {
        name          string
        devicePaths   []string
        expectedOrder []string
    }{
        {
            name:          "Reorder all",
            devicePaths:   []string{"CDROM(1)", "SCSI(0,0)", "SCSI(0,1)"},
            expectedOrder: []string{"0001", "0000", "0002"},
        },
        {
            name:          "Reorder subset",
            devicePaths:   []string{"SCSI(0,1)", "CDROM(1)"},
            expectedOrder: []string{"0002", "0001"},
        },
        {
            name:          "No matches",
            devicePaths:   []string{"NONEXISTENT"},
            expectedOrder: []string{},
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            order := ReorderBootEntries(entries, tt.devicePaths)
            
            if !reflect.DeepEqual(order, tt.expectedOrder) {
                t.Errorf("ReorderBootEntries() = %v, want %v", order, tt.expectedOrder)
            }
        })
    }
} 