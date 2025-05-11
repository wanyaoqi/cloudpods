package uefi

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "io/ioutil"
    "os"
    "os/exec"
    "strings"
)

// VarsData represents the structure of the JSON output from virt-fw-vars
type VarsData struct {
    Version   int        `json:"version"`
    Variables []Variable `json:"variables"`
}

// Variable represents a UEFI variable in the JSON output
type Variable struct {
    Name string `json:"name"`
    GUID string `json:"guid"`
    Attr int    `json:"attr"`
    Data string `json:"data"`
}

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
func ParseVarsJson(jsonPath string) ([]BootEntry, []string, error) {
    // Read JSON file
    jsonData, err := ioutil.ReadFile(jsonPath)
    if err != nil {
        return nil, nil, fmt.Errorf("failed to read JSON file: %v", err)
    }
    
    // Parse JSON
    var varsData VarsData
    err = json.Unmarshal(jsonData, &varsData)
    if err != nil {
        return nil, nil, fmt.Errorf("failed to parse JSON: %v", err)
    }
    
    // Extract boot entries and boot order
    var bootEntries []BootEntry
    var bootOrder []string
    
    for _, v := range varsData.Variables {
        if strings.HasPrefix(v.Name, "Boot") && len(v.Name) == 8 {
            // Parse boot entry
            entryID := v.Name
            entryName, devPaths, err := ParseBootEntryData(v.Data)
            if err != nil {
                fmt.Printf("Warning: failed to parse boot entry %s: %v\n", entryID, err)
                continue
            }
            
            // Determine device type
            devType := DetermineDeviceType(devPaths)
            
            // Format device path
            devPathStr := FormatDevicePath(devPaths)
            
            // Create boot entry
            entry := BootEntry{
                ID:       entryID,
                Name:     entryName,
                Path:     devPathStr,
                DevPaths: devPaths,
                DevType:  devType,
                RawData:  v.Data,
            }
            
            bootEntries = append(bootEntries, entry)
        } else if v.Name == "BootOrder" {
            // Parse boot order
            order, err := ParseBootOrderData(v.Data)
            if err != nil {
                fmt.Printf("Warning: failed to parse boot order: %v\n", err)
            } else {
                bootOrder = order
            }
        }
    }
    
    return bootEntries, bootOrder, nil
}

// UpdateBootOrderInJson updates the boot order in a JSON file
func UpdateBootOrderInJson(jsonPath string, bootOrder []string) error {
    // Read JSON file
    jsonData, err := ioutil.ReadFile(jsonPath)
    if err != nil {
        return fmt.Errorf("failed to read JSON file: %v", err)
    }
    
    // Parse JSON
    var varsData VarsData
    err = json.Unmarshal(jsonData, &varsData)
    if err != nil {
        return fmt.Errorf("failed to parse JSON: %v", err)
    }
    
    // Build boot order hex data
    bootOrderHex, err := BuildBootOrderHex(bootOrder)
    if err != nil {
        return fmt.Errorf("failed to build boot order hex: %v", err)
    }
    
    // Update or add BootOrder variable
    bootOrderFound := false
    for i, v := range varsData.Variables {
        if v.Name == "BootOrder" {
            varsData.Variables[i].Data = bootOrderHex
            bootOrderFound = true
            break
        }
    }
    
    if !bootOrderFound {
        // Add new BootOrder variable
        varsData.Variables = append(varsData.Variables, Variable{
            Name: "BootOrder",
            GUID: "8be4df61-93ca-11d2-aa0d-00e098032b8c", // EFI Global Variable GUID
            Attr: 7, // NV+BS+RT
            Data: bootOrderHex,
        })
    }
    
    // Write updated data back to JSON file
    updatedJsonData, err := json.MarshalIndent(varsData, "", "    ")
    if err != nil {
        return fmt.Errorf("failed to serialize JSON data: %v", err)
    }
    
    err = ioutil.WriteFile(jsonPath, updatedJsonData, 0644)
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