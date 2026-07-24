package container_device

import (
	"fmt"
	"strconv"
	"strings"

	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
	"yunion.io/x/log"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/hostman/hostinfo"
	"yunion.io/x/onecloud/pkg/hostman/isolated_device"
	"yunion.io/x/onecloud/pkg/hostman/options"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

func init() {
	isolated_device.RegisterContainerDeviceManager(newAscendNPUHamiManager())
}

type ascendNPUHamiManager struct {
	*ascendNPUManager
}

func newAscendNPUHamiManager() *ascendNPUHamiManager {
	return &ascendNPUHamiManager{}
}

func (m *ascendNPUHamiManager) GetContainerExtraConfigures(devs []*hostapi.ContainerDevice) ([]*runtimeapi.KeyValue, []*runtimeapi.Mount) {
	npus := []string{}
	memoryLimit := ""
	smLimit := ""
	for _, dev := range devs {
		if dev.IsolatedDevice == nil {
			continue
		}
		iDev := hostinfo.Instance().IsolatedDeviceMan.GetDeviceByCloudId(dev.IsolatedDevice.Id)
		devMan := iDev.GetContainerDeviceManager()
		if _, ok := devMan.(*ascendNPUHamiManager); !ok {
			continue
		}
		idx, err := extractPartitionNumber(dev.IsolatedDevice.Path)
		if err != nil {
			npus = append(npus, strconv.Itoa(idx))
		}
		if memoryLimit == "" {
			memoryLimit = fmt.Sprintf("%dM", dev.IsolatedDevice.MemoryLimit)
		}
		if smLimit == "" && dev.IsolatedDevice.SmUtilLimit > 0 {
			smLimit = fmt.Sprintf("%d", dev.IsolatedDevice.SmUtilLimit)
		}
	}
	if len(npus) == 0 {
		return nil, nil
	}
	out, err := procutils.NewRemoteCommandAsFarAsPossible("mkdir", "-p", options.HostOptions.AscendNpuHamiShmPath).Output()
	if err != nil {
		log.Errorf("mkdir -p %s: %s %s", options.HostOptions.AscendNpuHamiShmPath, out, err)
	}
	retEnvs := []*runtimeapi.KeyValue{
		{
			Key:   "ASCEND_VISIBLE_DEVICES",
			Value: strings.Join(npus, ","),
		},
		{
			Key:   "LD_PRELOAD",
			Value: options.HostOptions.HAMICoreLibvgpuPath,
		},
		{
			Key:   "NPU_MEM_QUOTA",
			Value: memoryLimit,
		},
		{
			Key:   "NPU_GLOBAL_SHM_PATH",
			Value: "/hami-shared-region/global_registry",
		},
	}
	if len(smLimit) > 0 {
		retEnvs = append(retEnvs, &runtimeapi.KeyValue{
			Key:   "NPU_PRIORITY",
			Value: smLimit,
		})
	}
	return retEnvs, []*runtimeapi.Mount{
		{
			ContainerPath: "/hami-shared-region",
			HostPath:      options.HostOptions.AscendNpuHamiLibvnpuPath,
			Readonly:      true,
		},
	}
}

func (m *ascendNPUHamiManager) ProbeDevices() ([]isolated_device.IDevice, error) {
	return getAscendNpus(m, computeapi.DEVICE_SHARING_MODE_HAMI)
}
