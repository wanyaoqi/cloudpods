// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package container_device

import (
	"fmt"
	"strings"

	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
	"yunion.io/x/onecloud/pkg/hostman/options"

	hostapi "yunion.io/x/onecloud/pkg/apis/host"
	"yunion.io/x/onecloud/pkg/hostman/isolated_device"
)

func init() {
	isolated_device.RegisterContainerDeviceManager(newNvidiaHAMIManager())
}

type nvidiaHAMIManager struct {
	nvidiaGPUManager
}

func newNvidiaHAMIManager() *nvidiaHAMIManager {
	return &nvidiaHAMIManager{}
}

func (m *nvidiaHAMIManager) GetType() isolated_device.ContainerDeviceType {
	return isolated_device.ContainerDeviceTypeNvidiaHAMI
}

func (m *nvidiaHAMIManager) ProbeDevices() ([]isolated_device.IDevice, error) {
	return probeNvidiaGpus(isolated_device.ContainerDeviceTypeNvidiaHAMI)
}

func (m *nvidiaHAMIManager) GetContainerExtraConfigures(devs []*hostapi.ContainerDevice) ([]*runtimeapi.KeyValue, []*runtimeapi.Mount) {
	gpuIds := []string{}
	memoryLimit := ""
	smLimit := ""
	for _, dev := range devs {
		if dev.IsolatedDevice == nil {
			continue
		}
		if dev.IsolatedDevice.DeviceType == string(isolated_device.ContainerDeviceTypeNvidiaHAMI) {
			continue
		}
		gpuIds = append(gpuIds, dev.IsolatedDevice.Path)
		if memoryLimit == "" {
			memoryLimit = fmt.Sprintf("%dM", dev.IsolatedDevice.MemoryLimit)
		}
		if smLimit == "" && dev.IsolatedDevice.SmUtilLimit > 0 {
			smLimit = fmt.Sprintf("%d", dev.IsolatedDevice.SmUtilLimit)
		}
	}
	if len(gpuIds) == 0 {
		return nil, nil
	}
	retEnvs := []*runtimeapi.KeyValue{}
	if len(gpuIds) > 0 {
		retEnvs = append(retEnvs, []*runtimeapi.KeyValue{
			{
				Key:   "NVIDIA_VISIBLE_DEVICES",
				Value: strings.Join(gpuIds, ","),
			},
			{
				Key:   "NVIDIA_DRIVER_CAPABILITIES",
				Value: "all",
			},
			{
				Key:   "LD_PRELOAD",
				Value: options.HostOptions.HAMICoreLibvgpuPath,
			},
			{
				Key:   "CUDA_DEVICE_MEMORY_LIMIT",
				Value: memoryLimit,
			},
		}...)
		if len(smLimit) > 0 {
			retEnvs = append(retEnvs, &runtimeapi.KeyValue{
				Key:   "CUDA_DEVICE_SM_LIMIT",
				Value: smLimit,
			})
		}
	}
	return retEnvs, []*runtimeapi.Mount{
		{
			ContainerPath: options.HostOptions.HAMICoreLibvgpuPath,
			HostPath:      options.HostOptions.HAMICoreLibvgpuPath,
			Readonly:      true,
		},
	}
}
