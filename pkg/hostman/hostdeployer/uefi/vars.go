package uefi

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io/ioutil"
    "os"
    "os/exec"
    "path/filepath"
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
        return "", output, fmt.Errorf("failed to execute virt-fw-vars command: %v", err)
    }
    
    return jsonPath, output, nil
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
        return nil, nil, fmt.Errorf("failed to parse JSON data: %v", err)
    }
    
    // Extract boot entries and boot order
    var bootEntries []BootEntry
    var bootOrder []string
    
    for _, variable := range varsData.Variables {
        if strings.HasPrefix(variable.Name, "Boot") && len(variable.Name) == 8 {
            // Parse boot entry
            title, devicePaths, err := ParseBootEntryData(variable.Data)
            if err != nil {
                continue // Skip invalid entries
            }
            
            // Format device path
            formattedPath := FormatDevicePath(devicePaths)
            
            // Determine device type
            devType := DetermineDeviceType(devicePaths)
            
            bootEntries = append(bootEntries, BootEntry{
                ID:       variable.Name,
                Name:     title,
                Path:     formattedPath,
                DevPaths: devicePaths,
                DevType:  devType,
                RawData:  variable.Data,
            })
        } else if variable.Name == "BootOrder" {
            // Parse boot order
            order, err := ParseBootOrderData(variable.Data)
            if err == nil {
                bootOrder = order
            }
        }
    }
    
    return bootEntries, bootOrder, nil
}

// UpdateBootOrderInJson updates the boot order in a JSON file
func UpdateBootOrderInJson(jsonPath string, newBootOrder []string) error {
    // Read JSON file
    jsonData, err := ioutil.ReadFile(jsonPath)
    if err != nil {
        return fmt.Errorf("failed to read JSON file: %v", err)
    }
    
    // Parse JSON
    var varsData VarsData
    err = json.Unmarshal(jsonData, &varsData)
    if err != nil {
        return fmt.Errorf("failed to parse JSON data: %v", err)
    }
    
    // Convert new boot order to hex format
    bootOrderHex, err := BuildBootOrderHex(newBootOrder)
    if err != nil {
        return fmt.Errorf("failed to convert boot order: %v", err)
    }
    
    // Update BootOrder variable
    bootOrderUpdated := false
    for i, variable := range varsData.Variables {
        if variable.Name == "BootOrder" {
            varsData.Variables[i].Data = bootOrderHex
            bootOrderUpdated = true
            break
        }
    }
    
    // If BootOrder variable not found, add it
    if !bootOrderUpdated {
        varsData.Variables = append(varsData.Variables, Variable{
            Name: "BootOrder",
            GUID: "8be4df61-93ca-11d2-aa0d-00e098032b8c",
            Attr: 7,
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