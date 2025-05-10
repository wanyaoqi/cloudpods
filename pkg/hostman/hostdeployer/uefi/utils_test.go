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
    // Create test boot entries
    entries := []BootEntry{
        {
            ID:      "Boot0000",
            Name:    "Windows Boot Manager",
            Path:    "PciRoot()/HD(2,GPT)/File(\\EFI\\Microsoft\\Boot\\bootmgfw.efi)",
            DevType: "HD",
        },
        {
            ID:      "Boot0001",
            Name:    "UEFI QEMU DVD-ROM QM00033",
            Path:    "PciRoot()/PCI(dev=01:0)/SCSI(pun=1,lun=0)",
            DevType: "CDROM",
        },
        {
            ID:      "Boot0002",
            Name:    "UEFI QEMU HARDDISK QM00001",
            Path:    "PciRoot()/PCI(dev=01:0)/SCSI(pun=0,lun=0)/HD(1,GPT)",
            DevType: "HD",
        },
        {
            ID:      "Boot0003",
            Name:    "EFI Network",
            Path:    "PciRoot()/PCI(dev=02:0)/MAC()",
            DevType: "NETWORK",
        },
    }

    tests := []struct {
        name           string
        diskPaths      []string
        cdromPaths     []string
        expectedDisks  []string
        expectedCDROMs []string
    }{
        {
            name:           "Match all",
            diskPaths:      []string{},
            cdromPaths:     []string{},
            expectedDisks:  []string{"Boot0000", "Boot0002"},
            expectedCDROMs: []string{"Boot0001"},
        },
        {
            name:           "Match specific disk",
            diskPaths:      []string{"Windows"},
            cdromPaths:     []string{},
            expectedDisks:  []string{"Boot0000"},
            expectedCDROMs: []string{"Boot0001"},
        },
        {
            name:           "Match specific CDROM",
            diskPaths:      []string{},
            cdromPaths:     []string{"QM00033"},
            expectedDisks:  []string{"Boot0000", "Boot0002"},
            expectedCDROMs: []string{"Boot0001"},
        },
        {
            name:           "Match by path",
            diskPaths:      []string{"GPT"},
            cdromPaths:     []string{},
            expectedDisks:  []string{"Boot0000", "Boot0002"},
            expectedCDROMs: []string{"Boot0001"},
        },
        {
            name:           "No matches",
            diskPaths:      []string{"NonExistent"},
            cdromPaths:     []string{"NonExistent"},
            expectedDisks:  []string{},
            expectedCDROMs: []string{},
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            diskEntries, cdromEntries := MatchBootEntries(entries, tt.diskPaths, tt.cdromPaths)
            
            // Extract IDs for comparison
            diskIDs := make([]string, len(diskEntries))
            for i, entry := range diskEntries {
                diskIDs[i] = entry.ID
            }
            
            cdromIDs := make([]string, len(cdromEntries))
            for i, entry := range cdromEntries {
                cdromIDs[i] = entry.ID
            }
            
            if !reflect.DeepEqual(diskIDs, tt.expectedDisks) {
                t.Errorf("MatchBootEntries() disk entries = %v, want %v", diskIDs, tt.expectedDisks)
            }
            
            if !reflect.DeepEqual(cdromIDs, tt.expectedCDROMs) {
                t.Errorf("MatchBootEntries() CDROM entries = %v, want %v", cdromIDs, tt.expectedCDROMs)
            }
        })
    }
}

func TestBuildBootOrder(t *testing.T) {
    // Create test boot entries
    diskEntries := []BootEntry{
        {
            ID:      "Boot0000",
            Name:    "Windows Boot Manager",
            DevType: "HD",
        },
        {
            ID:      "Boot0002",
            Name:    "UEFI QEMU HARDDISK QM00001",
            DevType: "HD",
        },
    }
    
    cdromEntries := []BootEntry{
        {
            ID:      "Boot0001",
            Name:    "UEFI QEMU DVD-ROM QM00033",
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
            name:          "CDROM first",
            diskPriority:  1,
            cdromPriority: 2,
            expectedOrder: []string{"0001", "0000", "0002"},
        },
        {
            name:          "Disk first",
            diskPriority:  2,
            cdromPriority: 1,
            expectedOrder: []string{"0000", "0002", "0001"},
        },
        {
            name:          "Only CDROM",
            diskPriority:  0,
            cdromPriority: 1,
            expectedOrder: []string{"0001"},
        },
        {
            name:          "Only disk",
            diskPriority:  1,
            cdromPriority: 0,
            expectedOrder: []string{"0000", "0002"},
        },
        {
            name:          "No boot devices",
            diskPriority:  0,
            cdromPriority: 0,
            expectedOrder: []string{},
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