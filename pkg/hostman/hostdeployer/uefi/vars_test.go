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
                "data": "090100002c0055006900410070007000000004071400c9bdb87cebf8344faaea3ee4af6516a10406140021aa2c4614760345836e8ab6f46623317fff0400"
            },
            {
                "name": "Boot0001",
                "guid": "8be4df61-93ca-11d2-aa0d-00e098032b8c",
                "attr": 7,
                "data": "010000001e0055004500460049002000510045004d00550020004400560044002d0052004f004d00200051004d00300030003000330033002000000002010c00d041030a0000000001010600010103010800010000007fff04004eac0881119f594d850ee21a522c59b2"
            },
            {
                "name": "BootOrder",
                "guid": "8be4df61-93ca-11d2-aa0d-00e098032b8c",
                "attr": 7,
                "data": "0000000100"
            }
        ]
    }`

    tmpfile, err := ioutil.TempFile("", "test-vars-*.json")
    if err != nil {
        t.Fatalf("Failed to create temp file: %v", err)
    }
    defer os.Remove(tmpfile.Name())

    if _, err := tmpfile.Write([]byte(jsonData)); err != nil {
        t.Fatalf("Failed to write to temp file: %v", err)
    }
    if err := tmpfile.Close(); err != nil {
        t.Fatalf("Failed to close temp file: %v", err)
    }

    // Test ParseVarsJson
    entries, order, err := ParseVarsJson(tmpfile.Name())
    if err != nil {
        t.Fatalf("ParseVarsJson() error = %v", err)
    }

    // Check boot entries
    if len(entries) != 2 {
        t.Errorf("Expected 2 boot entries, got %d", len(entries))
    }
    if entries[0].ID != "Boot0000" || entries[0].Name != "UiApp" {
        t.Errorf("Unexpected boot entry: %+v", entries[0])
    }
    if entries[1].ID != "Boot0001" || entries[1].Name != "UEFI QEMU DVD-ROM QM00033 " {
        t.Errorf("Unexpected boot entry: %+v", entries[1])
    }

    // Check boot order
    expectedOrder := []string{"0000", "0001", "00"}
    if !reflect.DeepEqual(order, expectedOrder) {
        t.Errorf("Expected boot order %v, got %v", expectedOrder, order)
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
                "data": "090100002c0055006900410070007000000004071400c9bdb87cebf8344faaea3ee4af6516a10406140021aa2c4614760345836e8ab6f46623317fff0400"
            },
            {
                "name": "Boot0001",
                "guid": "8be4df61-93ca-11d2-aa0d-00e098032b8c",
                "attr": 7,
                "data": "010000001e0055004500460049002000510045004d00550020004400560044002d0052004f004d00200051004d00300030003000330033002000000002010c00d041030a0000000001010600010103010800010000007fff04004eac0881119f594d850ee21a522c59b2"
            },
            {
                "name": "BootOrder",
                "guid": "8be4df61-93ca-11d2-aa0d-00e098032b8c",
                "attr": 7,
                "data": "0000000100"
            }
        ]
    }`

    tmpfile, err := ioutil.TempFile("", "test-vars-*.json")
    if err != nil {
        t.Fatalf("Failed to create temp file: %v", err)
    }
    defer os.Remove(tmpfile.Name())

    if _, err := tmpfile.Write([]byte(jsonData)); err != nil {
        t.Fatalf("Failed to write to temp file: %v", err)
    }
    if err := tmpfile.Close(); err != nil {
        t.Fatalf("Failed to close temp file: %v", err)
    }

    // Test UpdateBootOrderInJson
    newOrder := []string{"0001", "0000"}
    err = UpdateBootOrderInJson(tmpfile.Name(), newOrder)
    if err != nil {
        t.Fatalf("UpdateBootOrderInJson() error = %v", err)
    }

    // Read the updated file
    updatedData, err := ioutil.ReadFile(tmpfile.Name())
    if err != nil {
        t.Fatalf("Failed to read updated file: %v", err)
    }

    // Parse the updated JSON
    var varsData VarsData
    err = json.Unmarshal(updatedData, &varsData)
    if err != nil {
        t.Fatalf("Failed to parse updated JSON: %v", err)
    }

    // Find the BootOrder variable
    var bootOrderData string
    for _, v := range varsData.Variables {
        if v.Name == "BootOrder" {
            bootOrderData = v.Data
            break
        }
    }

    // Check if the boot order was updated correctly
    expectedHex := "0100000000"
    if bootOrderData != expectedHex {
        t.Errorf("Expected boot order data %s, got %s", expectedHex, bootOrderData)
    }
}

func TestUpdateBootOrderInJson_AddNew(t *testing.T) {
    // Create a temporary JSON file without BootOrder
    jsonData := `{
        "version": 2,
        "variables": [
            {
                "name": "Boot0000",
                "guid": "8be4df61-93ca-11d2-aa0d-00e098032b8c",
                "attr": 7,
                "data": "090100002c0055006900410070007000000004071400c9bdb87cebf8344faaea3ee4af6516a10406140021aa2c4614760345836e8ab6f46623317fff0400"
            },
            {
                "name": "Boot0001",
                "guid": "8be4df61-93ca-11d2-aa0d-00e098032b8c",
                "attr": 7,
                "data": "010000001e0055004500460049002000510045004d00550020004400560044002d0052004f004d00200051004d00300030003000330033002000000002010c00d041030a0000000001010600010103010800010000007fff04004eac0881119f594d850ee21a522c59b2"
            }
        ]
    }`

    tmpfile, err := ioutil.TempFile("", "test-vars-*.json")
    if err != nil {
        t.Fatalf("Failed to create temp file: %v", err)
    }
    defer os.Remove(tmpfile.Name())

    if _, err := tmpfile.Write([]byte(jsonData)); err != nil {
        t.Fatalf("Failed to write to temp file: %v", err)
    }
    if err := tmpfile.Close(); err != nil {
        t.Fatalf("Failed to close temp file: %v", err)
    }

    // Test UpdateBootOrderInJson
    newOrder := []string{"0001", "0000"}
    err = UpdateBootOrderInJson(tmpfile.Name(), newOrder)
    if err != nil {
        t.Fatalf("UpdateBootOrderInJson() error = %v", err)
    }

    // Read the updated file
    updatedData, err := ioutil.ReadFile(tmpfile.Name())
    if err != nil {
        t.Fatalf("Failed to read updated file: %v", err)
    }

    // Parse the updated JSON
    var varsData VarsData
    err = json.Unmarshal(updatedData, &varsData)
    if err != nil {
        t.Fatalf("Failed to parse updated JSON: %v", err)
    }

    // Find the BootOrder variable
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
    expectedHex := "0100000000"
    if bootOrderData != expectedHex {
        t.Errorf("Expected boot order data %s, got %s", expectedHex, bootOrderData)
    }
} 