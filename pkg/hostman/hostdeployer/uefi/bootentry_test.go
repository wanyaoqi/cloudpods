package uefi

import (
	"reflect"
	"testing"
)

func TestParseBootEntryData(t *testing.T) {
	tests := []struct {
		name           string
		hexData        string
		expectedName   string
		expectedDevType string
		expectError    bool
	}{
		{
			name:           "Valid boot entry",
			hexData:        "010000001e0055004500460049002000510045004d00550020004400560044002d0052004f004d00200051004d00300030003000330033002000000002010c00d041030a00000000030208000100000000007fff0400",
			expectedName:   "UEFI QEMU DVD-ROM QM00033 ",
			expectedDevType: "CDROM",
			expectError:    false,
		},
		{
			name:           "Invalid boot entry (too short)",
			hexData:        "0100",
			expectedName:   "",
			expectedDevType: "",
			expectError:    true,
		},
		{
			name:           "Invalid boot entry (invalid path list length)",
			hexData:        "010000001e0055004500460049002000510045004d00550020004400560044002d0052004f004d00200051004d00300030003000330033002000000002010c00",
			expectedName:   "UEFI QEMU DVD-ROM QM00033 ",
			expectedDevType: "",
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, devPaths, err := ParseBootEntryData(tt.hexData)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("ParseBootEntryData() error = nil, expected error")
				}
				return
			}
			
			if err != nil {
				t.Errorf("ParseBootEntryData() error = %v", err)
				return
			}
			
			if name != tt.expectedName {
				t.Errorf("ParseBootEntryData() name = %v, want %v", name, tt.expectedName)
			}
			
			if tt.expectedDevType != "" {
				devType := DetermineDeviceType(devPaths)
				if devType != tt.expectedDevType {
					t.Errorf("DetermineDeviceType() = %v, want %v", devType, tt.expectedDevType)
				}
			}
		})
	}
}

func TestParseBootOrder(t *testing.T) {
	tests := []struct {
		name          string
		hexData       string
		expectedOrder []string
		expectError   bool
	}{
		{
			name:          "Valid boot order",
			hexData:       "000001000200",
			expectedOrder: []string{"0000", "0001", "0002"},
			expectError:   false,
		},
		{
			name:          "Empty boot order",
			hexData:       "",
			expectedOrder: []string{},
			expectError:   false,
		},
		{
			name:          "Invalid boot order (odd length)",
			hexData:       "0000010002",
			expectedOrder: nil,
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order, err := ParseBootOrder(tt.hexData)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("ParseBootOrder() error = nil, expected error")
				}
				return
			}
			
			if err != nil {
				t.Errorf("ParseBootOrder() error = %v", err)
				return
			}
			
			if !reflect.DeepEqual(order, tt.expectedOrder) {
				t.Errorf("ParseBootOrder() = %v, want %v", order, tt.expectedOrder)
			}
		})
	}
}

func TestBuildBootOrderHex(t *testing.T) {
	tests := []struct {
		name          string
		bootOrder     []string
		expectedHex   string
		expectError   bool
	}{
		{
			name:          "Valid boot order",
			bootOrder:     []string{"0000", "0001", "0002"},
			expectedHex:   "000001000200",
			expectError:   false,
		},
		{
			name:          "Empty boot order",
			bootOrder:     []string{},
			expectedHex:   "",
			expectError:   false,
		},
		{
			name:          "Invalid boot order entry",
			bootOrder:     []string{"0000", "invalid", "0002"},
			expectedHex:   "",
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hex, err := BuildBootOrderHex(tt.bootOrder)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("BuildBootOrderHex() error = nil, expected error")
				}
				return
			}
			
			if err != nil {
				t.Errorf("BuildBootOrderHex() error = %v", err)
				return
			}
			
			if hex != tt.expectedHex {
				t.Errorf("BuildBootOrderHex() = %v, want %v", hex, tt.expectedHex)
			}
		})
	}
} 