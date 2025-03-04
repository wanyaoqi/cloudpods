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

package compute

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/printutils"

	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
	"yunion.io/x/onecloud/pkg/mcclient/options/compute"
)

func init() {
	cmd := shell.NewResourceCmd(&modules.SnapshotPoliciy).WithKeyword("snapshot-policy")
	cmd.List(&compute.SnapshotPolicyListOptions{})
	cmd.Delete(&options.BaseIdOptions{})
	cmd.Create(&compute.SnapshotPolicyCreateOptions{})
	cmd.Perform("bind-disks", &compute.SnapshotPolicyDisksOptions{})
	cmd.Perform("unbind-disks", &compute.SnapshotPolicyDisksOptions{})
	cmd.Perform("bind-guests", &compute.SnapshotPolicyDisksOptions{})
	cmd.Perform("unbind-guests", &compute.SnapshotPolicyDisksOptions{})
	cmd.Perform("syncstatus", &options.BaseIdOptions{})

	type DiskSnapshotPolicyListOptions struct {
		options.BaseListOptions
		SnapshotPolicy string `help:"ID or Name of SnapshotPolicy"`
		Disk           string `help:"ID or name of disk"`
		IsBackupPolicy *bool
	}
	R(&DiskSnapshotPolicyListOptions{}, "disk-snapshot-policy-list", "List disk snapshot policy pairs", func(s *mcclient.ClientSession, args *DiskSnapshotPolicyListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}
		if args.IsBackupPolicy != nil {
			params.Add(jsonutils.NewBool(*args.IsBackupPolicy), "is_backup_policy")
		}
		var result *printutils.ListResult
		var err error
		if len(args.Disk) > 0 {
			result, err = modules.DiskSnapshotPolicies.ListDescendent(s, args.Disk, params)
		} else if len(args.SnapshotPolicy) > 0 {
			result, err = modules.DiskSnapshotPolicies.ListDescendent2(s, args.SnapshotPolicy, params)
		} else {
			result, err = modules.DiskSnapshotPolicies.List(s, params)
		}
		if err != nil {
			return err
		}
		printList(result, modules.DiskSnapshotPolicies.GetColumns(s))
		return nil
	})

	type ServerSnapshotPolicyListOptions struct {
		options.BaseListOptions
		SnapshotPolicy string `help:"ID or Name of SnapshotPolicy"`
		Guest          string `help:"ID or name of disk"`
		IsBackupPolicy *bool
	}
	R(&ServerSnapshotPolicyListOptions{}, "server-snapshot-policy-list", "List guest snapshot policy pairs", func(s *mcclient.ClientSession, args *ServerSnapshotPolicyListOptions) error {
		var params *jsonutils.JSONDict
		{
			var err error
			params, err = args.BaseListOptions.Params()
			if err != nil {
				return err

			}
		}
		if args.IsBackupPolicy != nil {
			params.Add(jsonutils.NewBool(*args.IsBackupPolicy), "is_backup_policy")
		}
		var result *printutils.ListResult
		var err error
		if len(args.Guest) > 0 {
			result, err = modules.GuestSnapshotPolicies.ListDescendent(s, args.Guest, params)
		} else if len(args.SnapshotPolicy) > 0 {
			result, err = modules.GuestSnapshotPolicies.ListDescendent2(s, args.SnapshotPolicy, params)
		} else {
			result, err = modules.GuestSnapshotPolicies.List(s, params)
		}
		if err != nil {
			return err
		}
		printList(result, modules.GuestSnapshotPolicies.GetColumns(s))
		return nil
	})
}
