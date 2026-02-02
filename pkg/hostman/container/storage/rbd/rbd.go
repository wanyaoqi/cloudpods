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

package rbd

import (
	"fmt"
	"path/filepath"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/hostman/container/storage"
	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

func init() {
	storage.RegisterDriver(newRbd())
}

type rbd struct{}

func newRbd() *rbd {
	return &rbd{}
}

func (r rbd) GetType() storage.StorageType {
	return storage.STORAGE_TYPE_RBD
}

// parseRbdPath 解析 diskPath，格式: "rbd:pool/image:conf=/path/to/ceph.conf"
// 返回 pool, image, confPath, keyringPath
func parseRbdPath(diskPath string) (pool, image, confPath, keyringPath string, err error) {
	if !strings.HasPrefix(diskPath, "rbd:") {
		return "", "", "", "", errors.Errorf("invalid rbd path: %s", diskPath)
	}
	diskPath = strings.TrimPrefix(diskPath, "rbd:")
	parts := strings.SplitN(diskPath, ":", 2)
	if len(parts) < 1 || parts[0] == "" {
		return "", "", "", "", errors.Errorf("invalid rbd path: missing pool/image")
	}
	poolImage := parts[0]
	slash := strings.Index(poolImage, "/")
	if slash <= 0 {
		return "", "", "", "", errors.Errorf("invalid rbd path: missing pool/image in %s", poolImage)
	}
	pool = poolImage[:slash]
	image = poolImage[slash+1:]
	if pool == "" || image == "" {
		return "", "", "", "", errors.Errorf("invalid rbd path: empty pool or image")
	}
	confPath = ""
	if len(parts) == 2 && parts[1] != "" {
		confPrefix := "conf="
		if strings.HasPrefix(parts[1], confPrefix) {
			confPath = strings.TrimPrefix(parts[1], confPrefix)
		}
	}
	if confPath == "" {
		return "", "", "", "", errors.Errorf("invalid rbd path: missing conf= in %s", diskPath)
	}
	keyringPath = filepath.Join(filepath.Dir(confPath), "ceph.keyring")
	return pool, image, confPath, keyringPath, nil
}

// imageSpec 返回 "pool/image" 用于 rbd 命令
func imageSpec(pool, image string) string {
	return fmt.Sprintf("%s/%s", pool, image)
}

// rbdDeviceInfo 表示 rbd device list 输出的单个设备信息
type rbdDeviceInfo struct {
	Id        int    `json:"id"`
	Pool      string `json:"pool"`
	Namespace string `json:"namespace"`
	Image     string `json:"image"`
	Snap      string `json:"snap"`
	Device    string `json:"device"`
}

// listMappedDevices 执行 rbd device list 并解析已映射的设备
// 返回 map[pool/image]devicePath，如 map["rbd/disk0"]="/dev/rbd0"
func (r rbd) listMappedDevices(confPath, keyringPath string) (map[string]string, error) {
	args := []string{"device", "list", "--format", "json"}
	if confPath != "" {
		args = append(args, "--conf", confPath)
	}
	if keyringPath != "" && fileutils2.Exists(keyringPath) {
		args = append(args, "--keyring", keyringPath)
	}
	out, err := procutils.NewRemoteCommandAsFarAsPossible("rbd", args...).Output()
	if err != nil {
		return nil, errors.Wrapf(err, "rbd device list: %s", string(out))
	}
	// 解析 JSON 输出
	jsonObj, err := jsonutils.Parse(out)
	if err != nil {
		return nil, errors.Wrapf(err, "parse rbd device list json output: %s", string(out))
	}
	devices, err := jsonObj.GetArray()
	if err != nil {
		return nil, errors.Wrapf(err, "get devices array from json: %s", string(out))
	}
	result := make(map[string]string)
	for _, devObj := range devices {
		devInfo := rbdDeviceInfo{}
		if err := devObj.Unmarshal(&devInfo); err != nil {
			log.Warningf("failed to unmarshal device info: %v, skip", err)
			continue
		}
		if devInfo.Pool == "" || devInfo.Image == "" || devInfo.Device == "" {
			continue
		}
		spec := imageSpec(devInfo.Pool, devInfo.Image)
		result[spec] = devInfo.Device
	}
	return result, nil
}

func (r rbd) CheckConnect(diskPath string) (string, bool, error) {
	pool, image, confPath, keyringPath, err := parseRbdPath(diskPath)
	if err != nil {
		return "", false, err
	}
	spec := imageSpec(pool, image)
	mapped, err := r.listMappedDevices(confPath, keyringPath)
	if err != nil {
		return "", false, err
	}
	dev, ok := mapped[spec]
	if !ok {
		return "", false, nil
	}
	// 若存在分区则返回分区设备（与 local_raw 行为一致）
	devPath := r.checkPartition(dev)
	return devPath, true, nil
}

func (r rbd) checkPartition(devName string) string {
	// /dev/rbd0 -> /dev/rbd0p1, /dev/rbd/pool/image -> /dev/rbd/pool/imagep1
	partPath := devName + "p1"
	if fileutils2.Exists(partPath) {
		return partPath
	}
	return devName
}

func (r rbd) ConnectDisk(diskPath string) (string, error) {
	pool, image, confPath, keyringPath, err := parseRbdPath(diskPath)
	if err != nil {
		return "", err
	}
	spec := imageSpec(pool, image)
	args := []string{"device", "map", spec}
	if confPath != "" {
		args = append(args, "--conf", confPath)
	}
	if keyringPath != "" && fileutils2.Exists(keyringPath) {
		args = append(args, "--keyring", keyringPath)
	}
	out, err := procutils.NewRemoteCommandAsFarAsPossible("rbd", args...).Output()
	if err != nil {
		return "", errors.Wrapf(err, "rbd device map %s: %s", spec, string(out))
	}
	// 输出可能是 "rbd0" 或 "/dev/rbd0"
	devStr := strings.TrimSpace(string(out))
	if devStr == "" {
		// 部分版本不输出设备名，需要 list 再查一次
		devPath, _, err := r.CheckConnect(diskPath)
		if err != nil || devPath == "" {
			return "", errors.Wrapf(err, "rbd map succeeded but device not found for %s", spec)
		}
		return r.checkPartition(devPath), nil
	}
	if !strings.HasPrefix(devStr, "/dev/") {
		devStr = "/dev/" + devStr
	}
	// 去掉可能的后缀如 newline
	devStr = strings.TrimSpace(devStr)
	if idx := strings.Index(devStr, "\n"); idx > 0 {
		devStr = devStr[:idx]
	}
	return r.checkPartition(devStr), nil
}

func (r rbd) DisconnectDisk(diskPath string, mountPoint string) error {
	pool, image, confPath, keyringPath, err := parseRbdPath(diskPath)
	if err != nil {
		return err
	}
	spec := imageSpec(pool, image)
	// 先尝试按设备 unmap（更可靠），否则按 image spec unmap
	mapped, err := r.listMappedDevices(confPath, keyringPath)
	if err != nil {
		log.Warningf("rbd device list before unmap: %v", err)
		// 仍尝试按 spec unmap
		return r.unmapBySpec(spec, confPath, keyringPath)
	}
	dev, ok := mapped[spec]
	if !ok {
		log.Infof("rbd image %s not mapped, skip unmap", spec)
		return nil
	}
	args := []string{"device", "unmap", dev}
	if confPath != "" {
		args = append(args, "--conf", confPath)
	}
	out, err := procutils.NewRemoteCommandAsFarAsPossible("rbd", args...).Output()
	if err != nil {
		if strings.Contains(string(out), "not mapped") || strings.Contains(string(out), "No such device") {
			return nil
		}
		// 回退为按 spec unmap（部分版本支持）
		return r.unmapBySpec(spec, confPath, keyringPath)
	}
	log.Infof("rbd device unmap %s (image %s) ok", dev, spec)
	return nil
}

func (r rbd) unmapBySpec(spec, confPath, keyringPath string) error {
	args := []string{"device", "unmap", spec}
	if confPath != "" {
		args = append(args, "--conf", confPath)
	}
	if keyringPath != "" && fileutils2.Exists(keyringPath) {
		args = append(args, "--keyring", keyringPath)
	}
	out, err := procutils.NewRemoteCommandAsFarAsPossible("rbd", args...).Output()
	if err != nil {
		return errors.Wrapf(err, "rbd device unmap %s: %s", spec, string(out))
	}
	return nil
}
