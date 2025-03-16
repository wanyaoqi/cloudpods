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

package cgrouputils

import (
	"strings"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

const (
	CGROUP_PATH_SYSFS = "/sys/fs/cgroup"
	CGROUP_PATH_ROOT  = "/cgroup"
)

type ICgroupManager interface {
}

var cgroupManager ICgroupManager

func Init() error {
	cgroupPath := ""
	if fileutils2.Exists(CGROUP_PATH_SYSFS) {
		cgroupPath = CGROUP_PATH_SYSFS
	} else if fileutils2.Exists("CGROUP_PATH_ROOT") {
		cgroupPath = CGROUP_PATH_ROOT
	}
	if cgroupPath == "" {
		return errors.Errorf("Can't detect cgroup path")
	}
	output, err := procutils.NewCommand("stat", "-fc", "%T", cgroupPath).Output()
	if err != nil {
		return errors.Wrapf(err, "stat cgroup path %s", cgroupPath)
	}
	cgroupfs := strings.TrimSpace(string(output))
	if cgroupfs == "cgroup2fs" {
		// cgroup v2
		cgroupManager = cgroup2.NewManager()
	} else {
		// cgroup v1
		cgroupManager = cgroup1.NewManager()
	}
	return nil
}
