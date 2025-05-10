package uefi

import (
	"reflect"
	"testing"
)

func TestParseBootEntryData(t *testing.T) {
	tests := []struct {
		name          string
		hexData       string
		expectedTitle string
		expectedPaths int
		expectError   bool
	}{
		{
			name:          "Boot0000 entry (UiApp)",
			hexData:       "090100002c0055006900410070007000000004071400c9bdb87cebf8344faaea3ee4af6516a10406140021aa2c4614760345836e8ab6f46623317fff0400",
			expectedTitle: "UiApp",
			expectedPaths: 2,
			expectError:   false,
		},
		{
			name:          "Boot0001 entry (UEFI QEMU DVD-ROM)",
			hexData:       "010000001e0055004500460049002000510045004d00550020004400560044002d0052004f004d00200051004d00300030003000330033002000000002010c00d041030a0000000001010600010103010800010000007fff04004eac0881119f594d850ee21a522c59b2",
			expectedTitle: "UEFI QEMU DVD-ROM QM00033 ",
			expectedPaths: 3,
			expectError:   false,
		},
		{
			name:        "Invalid data",
			hexData:     "0102",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			title, paths, err := ParseBootEntryData(tt.hexData)
			if (err != nil) != tt.expectError {
				t.Errorf("ParseBootEntryData() error = %v, expectError %v", err, tt.expectError)
				return
			}
			if !tt.expectError {
				if title != tt.expectedTitle {
					t.Errorf("ParseBootEntryData() title = %v, want %v", title, tt.expectedTitle)
				}
				if len(paths) != tt.expectedPaths {
					t.Errorf("ParseBootEntryData() paths count = %v, want %v", len(paths), tt.expectedPaths)
				}
			}
		})
	}
}

func TestParseBootOrderData(t *testing.T) {
	tests := []struct {
		name           string
		hexData        string
		expectedOrder  []string
		expectError    bool
	}{
		{
			name:          "Valid boot order",
			hexData:       "0300000002000100070004000500",
			expectedOrder: []string{"0003", "0000", "0002", "0001", "0007", "0004", "0005"},
			expectError:   false,
		},
		{
			name:          "Empty boot order",
			hexData:       "",
			expectedOrder: []string{},
			expectError:   false,
		},
		{
			name:        "Invalid data",
			hexData:     "0",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			order, err := ParseBootOrderData(tt.hexData)
			if (err != nil) != tt.expectError {
				t.Errorf("ParseBootOrderData() error = %v, expectError %v", err, tt.expectError)
				return
			}
			if !tt.expectError && !reflect.DeepEqual(order, tt.expectedOrder) {
				t.Errorf("ParseBootOrderData() = %v, want %v", order, tt.expectedOrder)
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
			name:        "Valid boot order",
			bootOrder:   []string{"0003", "0000", "0002", "0001", "0007", "0004", "0005"},
			expectedHex: "0300000002000100070004000500",
			expectError: false,
		},
		{
			name:        "Empty boot order",
			bootOrder:   []string{},
			expectedHex: "",
			expectError: false,
		},
		{
			name:        "Invalid entry",
			bootOrder:   []string{"XXXX"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hex, err := BuildBootOrderHex(tt.bootOrder)
			if (err != nil) != tt.expectError {
				t.Errorf("BuildBootOrderHex() error = %v, expectError %v", err, tt.expectError)
				return
			}
			if !tt.expectError && hex != tt.expectedHex {
				t.Errorf("BuildBootOrderHex() = %v, want %v", hex, tt.expectedHex)
			}
		})
	}
} 