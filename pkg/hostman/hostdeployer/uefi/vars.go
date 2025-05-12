package uefi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"

	"yunion.io/x/log"
)

// VarsData represents the UEFI variables data
type VarsData struct {
	Version   int        `json:"version"`
	Variables []Variable `json:"variables"`
}

// Variable represents a UEFI variable
type Variable struct {
	Name string `json:"name"`
	GUID string `json:"guid"`
	Attr int    `json:"attr"`
	Data string `json:"data"`
}

// EFI_GLOBAL_VARIABLE_GUID is the GUID for EFI global variables
const EFI_GLOBAL_VARIABLE_GUID = "8be4df61-93ca-11d2-aa0d-00e098032b8c"

// DumpVarsToJson dumps UEFI variables to a JSON file
func DumpVarsToJson(ovmfVarsPath string) (string, string, error) {
	// Create temporary file for JSON output
	jsonFile, err := ioutil.TempFile("", "ovmf-vars-*.json")
	if err != nil {
		return "", "", fmt.Errorf("failed to create temporary file: %v", err)
	}
	jsonPath := jsonFile.Name()
	jsonFile.Close()

	// Execute virt-fw-vars with JSON output
	cmd := exec.Command("virt-fw-vars", "-i", ovmfVarsPath, "--output-json", jsonPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	output := stdout.String() + stderr.String()

	if err != nil {
		os.Remove(jsonPath)
		return output, "", fmt.Errorf("failed to execute virt-fw-vars: %v", err)
	}

	return output, jsonPath, nil
}

// ParseVarsJson parses UEFI variables from a JSON file
func ParseVarsJson(jsonPath string) ([]BootEntry, []uint16, error) {
	// Read JSON file
	data, err := ioutil.ReadFile(jsonPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read JSON file: %v", err)
	}

	// Parse JSON
	var varsData VarsData
	err = json.Unmarshal(data, &varsData)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse JSON: %v", err)
	}

	// Parse boot entries and boot order
	var bootEntries []BootEntry
	var bootOrder []uint16

	for _, v := range varsData.Variables {
		// Check if this is a boot entry
		if len(v.Name) >= 8 && v.Name[:4] == "Boot" && v.GUID == EFI_GLOBAL_VARIABLE_GUID {
			// Check if this is the boot order
			if v.Name == "BootOrder" {
				var err error
				bootOrder, err = ParseBootOrder(v.Data)
				if err != nil {
					return nil, nil, fmt.Errorf("failed to parse boot order: %v", err)
				}
				continue
			}

			// Parse boot entry
			name, devPaths, err := ParseBootEntryData(v.Data)
			if err != nil {
				log.Errorf("failed to parse boot entry %s: %s", v.Name, err)
				continue
			}

			// Create boot entry
			entry := BootEntry{
				ID:       v.Name,
				Name:     name,
				DevPaths: devPaths,
				RawData:  v.Data,
			}

			// Format device path
			if len(devPaths) > 0 {
				entry.Path = FormatDevicePath(devPaths)
				entry.DevType = DetermineDeviceType(devPaths)
			} else {
				entry.DevType = "UNKNOWN"
			}

			// Add entry to list
			bootEntries = append(bootEntries, entry)
		}
	}

	return bootEntries, bootOrder, nil
}

// UpdateBootOrderInJson updates the boot order in a UEFI variables JSON file
func UpdateBootOrderInJson(jsonPath string, bootOrder []string) error {
	// Read JSON file
	data, err := ioutil.ReadFile(jsonPath)
	if err != nil {
		return fmt.Errorf("failed to read JSON file: %v", err)
	}

	// Parse JSON
	var varsData VarsData
	err = json.Unmarshal(data, &varsData)
	if err != nil {
		return fmt.Errorf("failed to parse JSON: %v", err)
	}

	// Build boot order hex data
	bootOrderHex, err := BuildBootOrderHex(bootOrder)
	if err != nil {
		return fmt.Errorf("failed to build boot order hex: %v", err)
	}

	// Update or add boot order
	bootOrderFound := false
	for i, v := range varsData.Variables {
		if v.Name == "BootOrder" && v.GUID == EFI_GLOBAL_VARIABLE_GUID {
			varsData.Variables[i].Data = bootOrderHex
			bootOrderFound = true
			break
		}
	}

	// Add boot order if not found
	if !bootOrderFound {
		varsData.Variables = append(varsData.Variables, Variable{
			Name: "BootOrder",
			GUID: EFI_GLOBAL_VARIABLE_GUID,
			Attr: 7, // NV+BS+RT
			Data: bootOrderHex,
		})
	}

	// Write updated JSON
	updatedData, err := json.MarshalIndent(varsData, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %v", err)
	}

	err = ioutil.WriteFile(jsonPath, updatedData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write JSON file: %v", err)
	}

	return nil
}

// ApplyJsonToVars applies a JSON file to OVMF_VARS.fd
func ApplyJsonToVars(jsonPath, inputVarsPath, outputVarsPath string) (string, error) {
	// Execute virt-fw-vars to apply JSON
	cmd := exec.Command("virt-fw-vars", "-i", inputVarsPath, "-o", outputVarsPath, "--set-json", jsonPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	output := stdout.String() + stderr.String()

	if err != nil {
		return output, fmt.Errorf("failed to execute virt-fw-vars --set-json command: %v", err)
	}

	return output, nil
}

// CopyFile copies a file from src to dst
func CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	return destFile.Sync()
}

// SaveBootOrder saves the boot order to a UEFI variables JSON file
func SaveBootOrder(jsonPath string, diskPaths, cdromPaths []string, diskPriority, cdromPriority int) error {
	// Parse UEFI variables
	entries, _, err := ParseVarsJson(jsonPath)
	if err != nil {
		return fmt.Errorf("failed to parse UEFI variables: %v", err)
	}

	// Match boot entries
	diskEntries, cdromEntries := MatchBootEntries(entries, diskPaths, cdromPaths)

	// Build boot order
	bootOrder := BuildBootOrder(diskEntries, cdromEntries, int32(diskPriority), int32(cdromPriority))

	// Update boot order in JSON
	err = UpdateBootOrderInJson(jsonPath, bootOrder)
	if err != nil {
		return fmt.Errorf("failed to update boot order: %v", err)
	}

	return nil
}
