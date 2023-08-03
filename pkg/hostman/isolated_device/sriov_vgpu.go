package isolated_device

import (
	"fmt"
	"io/ioutil"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/hostman/guestman/desc"
	"yunion.io/x/onecloud/pkg/scheduler/api"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

type sSRIOVGpuDevice struct {
	*sSRIOVBaseDevice
}

func NewSRIOVGpuDevice(dev *PCIDevice, devType string) *sSRIOVGpuDevice {
	return &sSRIOVGpuDevice{
		sSRIOVBaseDevice: newSRIOVBaseDevice(dev, devType),
	}
}

func (dev *sSRIOVGpuDevice) GetHotPlugOptions(isolatedDev *desc.SGuestIsolatedDevice) ([]*HotPlugOption, error) {
	panic("implement me")
}

func (dev *sSRIOVGpuDevice) GetHotUnplugOptions(isolatedDev *desc.SGuestIsolatedDevice) ([]*HotUnplugOption, error) {
	panic("implement me")
}

func (dev *sSRIOVGpuDevice) GetPfName() string {
	return ""
}

func (dev *sSRIOVGpuDevice) GetVirtfn() int {
	return -1
}

func (dev *sSRIOVGpuDevice) GetVGACmd() string {
	return ""
}

func (dev *sSRIOVGpuDevice) GetCPUCmd() string {
	return ""
}

func (dev *sSRIOVGpuDevice) GetQemuId() string {
	return fmt.Sprintf("dev_%s", strings.ReplaceAll(dev.GetAddr(), ":", "_"))
}

func (dev *sSRIOVGpuDevice) GetWireId() string {
	return ""
}

func (dev *sSRIOVGpuDevice) CustomProbe(idx int) error {
	// check environments on first probe
	if idx == 0 {
		for _, driver := range []string{"vfio", "vfio_iommu_type1", "vfio-pci"} {
			if err := procutils.NewRemoteCommandAsFarAsPossible("modprobe", driver).Run(); err != nil {
				return fmt.Errorf("modprobe %s: %v", driver, err)
			}
		}
	}

	driver, err := dev.GetKernelDriver()
	if err != nil {
		return fmt.Errorf("Nic %s is occupied by another driver: %s", dev.GetAddr(), driver)
	}
	return nil
}

func getSRIOVGpus(gpuPF string) ([]*sSRIOVGpuDevice, error) {
	sysDeviceDir := "/sys/bus/pci/devices"
	devicePath := path.Join(sysDeviceDir, gpuPF)
	if !fileutils2.Exists(devicePath) {
		return nil, errors.Errorf("unknown device %s", gpuPF)
	}
	files, err := ioutil.ReadDir(devicePath)
	if err != nil {
		return nil, errors.Wrap(err, "read device path")
	}
	sriovGPUs := make([]*sSRIOVGpuDevice, 0)
	for i := range files {
		if strings.HasPrefix(files[i].Name(), "virtfn") {
			_, err := strconv.Atoi(files[i].Name()[len("virtfn"):])
			if err != nil {
				return nil, err
			}
			vfPath, err := filepath.EvalSymlinks(path.Join(devicePath, files[i].Name()))
			if err != nil {
				return nil, err
			}
			vfBDF := path.Base(vfPath)
			vfDev, err := detectSRIOVDevice(vfBDF)
			if err != nil {
				return nil, err
			}
			sriovGPUs = append(sriovGPUs, NewSRIOVGpuDevice(vfDev, api.GPU_HPC_TYPE))
		}
	}
	return nil, err
}
