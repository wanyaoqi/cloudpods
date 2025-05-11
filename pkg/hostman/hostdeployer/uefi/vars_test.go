package uefi

import (
    "encoding/json"
    "io/ioutil"
    "os"
    "reflect"
    "testing"
)

func TestParseVarsJson(t *testing.T) {
    // Create a temporary JSON file for testing
    jsonData := `{
        "version": 2,
        "variables": [
            {
                "name": "Boot0000",
                "guid": "8be4df61-93ca-11d2-aa0d-00e098032b8c",
                "attr": 7,
                "data": "010000002c0055006900410070007000000004071400c9bdb87cebf8344faaea3ee4af6516a10406140021aa2c4614760345836e8ab6f46623317fff0400"
            },
            {
                "name": "Boot0001",
                "guid": "8be4df61-93ca-11d2-aa0d-00e098032b8c",
                "attr": 7,
                "data": "010000001e0055004500460049002000510045004d00550020004400560044002d0052004f004d00200051004d00300030003000330033002000000002010c00d041030a0000000003020800010000000000"
            },
            {
                "name": "BootOrder",
                "guid": "8be4df61-93ca-11d2-aa0d-00e098032b8c",
                "attr": 7,
                "data": "000001000200"
            }
        ]
    }`
    
    jsonFile, err := ioutil.TempFile("", "vars-test-*.json")
    if err != nil {
        t.Fatalf("Failed to create temporary file: %v", err)
    }
    defer os.Remove(jsonFile.Name())
    
    if _, err := jsonFile.Write([]byte(jsonData)); err != nil {
        t.Fatalf("Failed to write JSON data: %v", err)
    }
    jsonFile.Close()
    
    // Test ParseVarsJson
    entries, bootOrder, err := ParseVarsJson(jsonFile.Name())
    if err != nil {
        t.Fatalf("ParseVarsJson() error = %v", err)
    }
    
    // Check entries
    if len(entries) != 2 {
        t.Errorf("Expected 2 boot entries, got %d", len(entries))
    } else {
        // Check first entry
        if entries[0].ID != "Boot0000" {
            t.Errorf("Expected first entry ID Boot0000, got %s", entries[0].ID)
        }
        if entries[0].Name != "UiApp" {
            t.Errorf("Expected first entry name UiApp, got %s", entries[0].Name)
        }
        
        // Check second entry
        if entries[1].ID != "Boot0001" {
            t.Errorf("Expected second entry ID Boot0001, got %s", entries[1].ID)
        }
        if entries[1].Name != "UEFI QEMU DVD-ROM QM00033 " {
            t.Errorf("Expected second entry name 'UEFI QEMU DVD-ROM QM00033 ', got %s", entries[1].Name)
        }
        if entries[1].DevType != "CDROM" {
            t.Errorf("Expected second entry device type CDROM, got %s", entries[1].DevType)
        }
    }
    
    // Check boot order
    expectedBootOrder := []string{"0000", "0001", "0002"}
    if !reflect.DeepEqual(bootOrder, expectedBootOrder) {
        t.Errorf("Expected boot order %v, got %v", expectedBootOrder, bootOrder)
    }
}

func TestUpdateBootOrderInJson(t *testing.T) {
    // Create a temporary JSON file for testing
    jsonData := `{
        "version": 2,
        "variables": [
            {
                "name": "Boot0000",
                "guid": "8be4df61-93ca-11d2-aa0d-00e098032b8c",
                "attr": 7,
                "data": "010000002c0055006900410070007000000004071400c9bdb87cebf8344faaea3ee4af6516a10406140021aa2c4614760345836e8ab6f46623317fff0400"
            },
            {
                "name": "Boot0001",
                "guid": "8be4df61-93ca-11d2-aa0d-00e098032b8c",
                "attr": 7,
                "data": "010000001e0055004500460049002000510045004d00550020004400560044002d0052004f004d00200051004d00300030003000330033002000000002010c00d041030a0000000003020800010000000000"
            },
            {
                "name": "BootOrder",
                "guid": "8be4df61-93ca-11d2-aa0d-00e098032b8c",
                "attr": 7,
                "data": "000001000200"
            }
        ]
    }`
    
    jsonFile, err := ioutil.TempFile("", "vars-test-*.json")
    if err != nil {
        t.Fatalf("Failed to create temporary file: %v", err)
    }
    defer os.Remove(jsonFile.Name())
    
    if _, err := jsonFile.Write([]byte(jsonData)); err != nil {
        t.Fatalf("Failed to write JSON data: %v", err)
    }
    jsonFile.Close()
    
    // Test UpdateBootOrderInJson
    newBootOrder := []string{"0001", "0000"}
    err = UpdateBootOrderInJson(jsonFile.Name(), newBootOrder)
    if err != nil {
        t.Fatalf("UpdateBootOrderInJson() error = %v", err)
    }
    
    // Read updated JSON
    updatedData, err := ioutil.ReadFile(jsonFile.Name())
    if err != nil {
        t.Fatalf("Failed to read updated JSON: %v", err)
    }
    
    var varsData VarsData
    err = json.Unmarshal(updatedData, &varsData)
    if err != nil {
        t.Fatalf("Failed to parse updated JSON: %v", err)
    }
    
    // Check if BootOrder was updated
    var bootOrderFound bool
    var bootOrderData string
    for _, v := range varsData.Variables {
        if v.Name == "BootOrder" {
            bootOrderFound = true
            bootOrderData = v.Data
            break
        }
    }
    
    if !bootOrderFound {
        t.Errorf("BootOrder variable not found")
    }
    
    // Check if the boot order was set correctly
    expectedHex := "01000000"
    if bootOrderData != expectedHex {
        t.Errorf("Expected boot order data %s, got %s", expectedHex, bootOrderData)
    }
}

func TestUpdateBootOrderInJson_AddNew(t *testing.T) {
    // Create a temporary JSON file for testing
    jsonData := `{
        "version": 2,
        "variables": [
            {
                "name": "Boot0000",
                "guid": "8be4df61-93ca-11d2-aa0d-00e098032b8c",
                "attr": 7,
                "data": "010000002c0055006900410070007000000004071400c9bdb87cebf8344faaea3ee4af6516a10406140021aa2c4614760345836e8ab6f46623317fff0400"
            },
            {
                "name": "Boot0001",
                "guid": "8be4df61-93ca-11d2-aa0d-00e098032b8c",
                "attr": 7,
                "data": "010000001e0055004500460049002000510045004d00550020004400560044002d0052004f004d00200051004d00300030003000330033002000000002010c00d041030a0000000003020800010000000000"
            }
        ]
    }`
    
    jsonFile, err := ioutil.TempFile("", "vars-test-*.json")
    if err != nil {
        t.Fatalf("Failed to create temporary file: %v", err)
    }
    defer os.Remove(jsonFile.Name())
    
    if _, err := jsonFile.Write([]byte(jsonData)); err != nil {
        t.Fatalf("Failed to write JSON data: %v", err)
    }
    jsonFile.Close()
    
    // Test UpdateBootOrderInJson
    newBootOrder := []string{"0001", "0000"}
    err = UpdateBootOrderInJson(jsonFile.Name(), newBootOrder)
    if err != nil {
        t.Fatalf("UpdateBootOrderInJson() error = %v", err)
    }
    
    // Read updated JSON
    updatedData, err := ioutil.ReadFile(jsonFile.Name())
    if err != nil {
        t.Fatalf("Failed to read updated JSON: %v", err)
    }
    
    var varsData VarsData
    err = json.Unmarshal(updatedData, &varsData)
    if err != nil {
        t.Fatalf("Failed to parse updated JSON: %v", err)
    }
    
    // Check if BootOrder was added
    var bootOrderFound bool
    var bootOrderData string
    for _, v := range varsData.Variables {
        if v.Name == "BootOrder" {
            bootOrderFound = true
            bootOrderData = v.Data
            break
        }
    }
    
    // Check if BootOrder was added
    if !bootOrderFound {
        t.Errorf("BootOrder variable was not added")
    }
    
    // Check if the boot order was set correctly
    expectedHex := "01000000"
    if bootOrderData != expectedHex {
        t.Errorf("Expected boot order data %s, got %s", expectedHex, bootOrderData)
    }
} 