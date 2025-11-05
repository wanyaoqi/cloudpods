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

package mdadm

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/baremetal/utils/raid"
	"yunion.io/x/onecloud/pkg/compute/baremetal"
)

const (
	MDADM_BIN         = "/usr/sbin/mdadm"
	MDADM_DRIVER_NAME = "mdadm"
)

func init() {
	raid.RegisterDriver(baremetal.DISK_DRIVER_LINUX, NewMdadmRaidLinux)
	raid.RegisterDriver(baremetal.DISK_DRIVER_PCIE, NewMdadmRaidPcie)
}

type MdadmRaid struct {
	term       raid.IExecTerm
	adapters   []*MdadmRaidAdapter
	driverName string
}

func NewMdadmRaidLinux(term raid.IExecTerm) raid.IRaidDriver {
	return &MdadmRaid{
		term:       term,
		adapters:   make([]*MdadmRaidAdapter, 0),
		driverName: baremetal.DISK_DRIVER_LINUX,
	}
}

func NewMdadmRaidPcie(term raid.IExecTerm) raid.IRaidDriver {
	return &MdadmRaid{
		term:       term,
		adapters:   make([]*MdadmRaidAdapter, 0),
		driverName: baremetal.DISK_DRIVER_PCIE,
	}
}

func (r *MdadmRaid) GetName() string {
	return r.driverName
}

func (r *MdadmRaid) ParsePhyDevs() error {
	// 对于软RAID，我们不需要解析物理设备，因为设备列表在构建时提供
	// 创建一个虚拟适配器，索引为0
	if len(r.adapters) == 0 {
		adapter := &MdadmRaidAdapter{
			raid:  r,
			term:  r.term,
			index: 0,
			devs:  make([]*baremetal.BaremetalStorage, 0),
		}
		r.adapters = append(r.adapters, adapter)
	}
	return nil
}

// SetDevicesForAdapter 为指定适配器设置设备列表（用于软RAID）
func (r *MdadmRaid) SetDevicesForAdapter(adapterIdx int, devs []*baremetal.BaremetalStorage) {
	for _, adapter := range r.adapters {
		if adapter.GetIndex() == adapterIdx {
			adapter.setDevices(devs)
			break
		}
	}
}

func (r *MdadmRaid) GetAdapters() []raid.IRaidAdapter {
	ret := make([]raid.IRaidAdapter, len(r.adapters))
	for i := range r.adapters {
		ret[i] = r.adapters[i]
	}
	return ret
}

func (r *MdadmRaid) PreBuildRaid(confs []*api.BaremetalDiskConfig, adapterIdx int) error {
	return nil
}

func (r *MdadmRaid) CleanRaid() error {
	// 停止所有活动的md设备
	cmd := fmt.Sprintf("%s --stop --scan", MDADM_BIN)
	_, err := r.term.Run(cmd)
	if err != nil {
		log.Warningf("Stop md devices: %v", err)
	}

	// 清除所有md设备的superblock
	cmd = fmt.Sprintf("%s --zero-superblock --force /dev/md* 2>/dev/null || true", MDADM_BIN)
	_, err = r.term.Run(cmd)
	if err != nil {
		log.Warningf("Zero superblock: %v", err)
	}

	return nil
}

type MdadmRaidAdapter struct {
	raid  *MdadmRaid
	term  raid.IExecTerm
	index int
	devs  []*baremetal.BaremetalStorage
}

func (a *MdadmRaidAdapter) GetIndex() int {
	return a.index
}

func (a *MdadmRaidAdapter) PreBuildRaid(confs []*api.BaremetalDiskConfig) error {
	return nil
}

func (a *MdadmRaidAdapter) GetLogicVolumes() ([]*raid.RaidLogicalVolume, error) {
	lvs := make([]*raid.RaidLogicalVolume, 0)
	cmd := fmt.Sprintf("%s --detail --scan 2>/dev/null || true", MDADM_BIN)
	output, err := a.term.Run(cmd)
	if err != nil {
		// 如果没有md设备，返回空列表
		return lvs, nil
	}

	// mdadm --detail --scan 输出格式: ARRAY /dev/md0 metadata=1.2 name=hostname:0 UUID=...
	mdRe := regexp.MustCompile(`ARRAY\s+(/dev/md\d+)`)
	usedMdNums := make(map[int]bool)

	for _, line := range output {
		matches := mdRe.FindStringSubmatch(line)
		if len(matches) > 1 {
			mdPath := matches[1]
			// 从路径中提取数字，如 /dev/md0 -> 0
			mdNumRe := regexp.MustCompile(`/dev/md(\d+)`)
			if numMatches := mdNumRe.FindStringSubmatch(mdPath); len(numMatches) > 1 {
				if num, err := strconv.Atoi(numMatches[1]); err == nil {
					if !usedMdNums[num] {
						lv := &raid.RaidLogicalVolume{
							Index:    num,
							Adapter:  a.index,
							BlockDev: mdPath,
						}
						lvs = append(lvs, lv)
						usedMdNums[num] = true
					}
				}
			}
		}
	}
	return lvs, nil
}

func (a *MdadmRaidAdapter) RemoveLogicVolumes() error {
	// 停止所有md设备
	cmd := fmt.Sprintf("%s --stop --scan", MDADM_BIN)
	_, err := a.term.Run(cmd)
	if err != nil {
		log.Warningf("Stop md devices: %v", err)
	}
	return nil
}

func (a *MdadmRaidAdapter) GetDevices() []*baremetal.BaremetalStorage {
	return a.devs
}

func (a *MdadmRaidAdapter) setDevices(devs []*baremetal.BaremetalStorage) {
	a.devs = devs
}

func (a *MdadmRaidAdapter) BuildRaid0(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	return a.buildRaid("0", devs, conf)
}

func (a *MdadmRaidAdapter) BuildRaid1(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	return a.buildRaid("1", devs, conf)
}

func (a *MdadmRaidAdapter) BuildRaid5(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	return a.buildRaid("5", devs, conf)
}

func (a *MdadmRaidAdapter) BuildRaid10(devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	return a.buildRaid("10", devs, conf)
}

func (a *MdadmRaidAdapter) BuildNoneRaid(devs []*baremetal.BaremetalStorage) error {
	return nil
}

func (a *MdadmRaidAdapter) PostBuildRaid() error {
	// 更新mdadm配置
	cmd := fmt.Sprintf("%s --examine --scan > /etc/mdadm/mdadm.conf 2>/dev/null || %s --examine --scan > /etc/mdadm.conf 2>/dev/null || true", MDADM_BIN, MDADM_BIN)
	_, err := a.term.Run(cmd)
	if err != nil {
		log.Warningf("Update mdadm.conf: %v", err)
	}
	return nil
}

func (a *MdadmRaidAdapter) buildRaid(level string, devs []*baremetal.BaremetalStorage, conf *api.BaremetalDiskConfig) error {
	if len(devs) == 0 {
		return fmt.Errorf("no devices provided for RAID %s", level)
	}

	// 获取下一个可用的md设备号
	mdNum, err := a.getNextMdNum()
	if err != nil {
		return errors.Wrap(err, "get next md number")
	}

	// 构建设备列表
	devPaths := make([]string, 0, len(devs))
	for _, dev := range devs {
		if dev.Dev == "" {
			return fmt.Errorf("device path is empty for storage")
		}
		devPaths = append(devPaths, dev.Dev)
	}

	// 确保设备没有被使用
	for _, dev := range devPaths {
		if err := a.ensureDeviceClean(dev); err != nil {
			return errors.Wrapf(err, "clean device %s", dev)
		}
	}

	// 构建mdadm命令
	args := []string{
		"--create",
		fmt.Sprintf("/dev/md%d", mdNum),
		fmt.Sprintf("--level=%s", level),
		fmt.Sprintf("--raid-devices=%d", len(devs)),
	}

	// 添加设备
	for _, dev := range devPaths {
		args = append(args, dev)
	}

	// 对于RAID1和RAID5，可以设置bitmap
	if level == "1" || level == "5" {
		args = append(args, "--bitmap=internal")
	}

	// 自动装配
	args = append(args, "--assume-clean")

	cmd := fmt.Sprintf("%s %s", MDADM_BIN, strings.Join(args, " "))
	log.Infof("Building software RAID %s: %s", level, cmd)

	output, err := a.term.Run(cmd)
	if err != nil {
		return errors.Wrapf(err, "mdadm create raid %s failed, output: %v", level, output)
	}

	log.Infof("Successfully created software RAID %s: /dev/md%d", level, mdNum)
	return nil
}

func (a *MdadmRaidAdapter) getNextMdNum() (int, error) {
	// 查找已存在的md设备
	cmd := "ls -1 /dev/md* 2>/dev/null | grep -E '/dev/md[0-9]+$' || true"
	output, err := a.term.Run(cmd)
	if err != nil {
		return 0, errors.Wrap(err, "list md devices")
	}

	usedNums := make(map[int]bool)
	mdNumRe := regexp.MustCompile(`/dev/md(\d+)`)
	for _, line := range output {
		matches := mdNumRe.FindStringSubmatch(line)
		if len(matches) > 1 {
			if num, err := strconv.Atoi(matches[1]); err == nil {
				usedNums[num] = true
			}
		}
	}

	// 找到第一个未使用的编号（从0开始）
	for i := 0; i < 256; i++ {
		if !usedNums[i] {
			return i, nil
		}
	}

	return 0, fmt.Errorf("no available md device number")
}

func (a *MdadmRaidAdapter) ensureDeviceClean(dev string) error {
	// 检查设备是否在md设备中
	cmd := fmt.Sprintf("%s --examine %s 2>/dev/null || true", MDADM_BIN, dev)
	output, err := a.term.Run(cmd)
	if err != nil {
		return errors.Wrapf(err, "examine device %s", dev)
	}

	// 如果设备有RAID superblock，清除它
	for _, line := range output {
		if strings.Contains(line, "mdadm") || strings.Contains(line, "ARRAY") {
			cmd := fmt.Sprintf("%s --zero-superblock --force %s", MDADM_BIN, dev)
			_, err := a.term.Run(cmd)
			if err != nil {
				return errors.Wrapf(err, "zero superblock on %s", dev)
			}
			break
		}
	}

	return nil
}
