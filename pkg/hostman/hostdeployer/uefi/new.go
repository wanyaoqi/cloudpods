package uefi

import (
	"fmt"
	"io/ioutil"
	"os"
	"sort"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/procutils"
)

func DumpOvmfVarsToJson(ovmfVarsPath string) (string, error) {
	// Create temporary file for JSON output
	jsonFile, err := ioutil.TempFile("", "ovmf-vars-*.json")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %v", err)
	}
	jsonPath := jsonFile.Name()
	jsonFile.Close()

	output, err := procutils.NewCommand("virt-fw-vars", "-i", ovmfVarsPath, "--output-json", jsonPath).Output()
	if err != nil {
		os.Remove(jsonPath)
		return "", errors.Wrapf(err, "virt-fw-vars failed dump to json %s", output)
	}
	return jsonPath, nil
}

func ParseUefiVars(ovmfVarsPath string) ([]*BootEntry, []uint16, string, error) {
	jsonPath, err := DumpOvmfVarsToJson(ovmfVarsPath)
	if err != nil {
		return nil, nil, "", errors.Wrap(err, "DumpOvmfVarsToJson")
	}

	bootEntry, bootOrder, err := ParseVarsJson(jsonPath)
	if err != nil {
		return nil, nil, "", errors.Wrap(err, "ParseVarsJson")
	}
	sort.Slice(bootEntry, func(i, j int) bool {
		return bootEntry[i].ID < bootEntry[j].ID
	})
	return bootEntry, bootOrder, jsonPath, nil
}
