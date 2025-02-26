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

package tasks

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/cloudcommon/notifyclient"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type SnapshotCreateTask struct {
	taskman.STask
}

func init() {
	taskman.RegisterTask(SnapshotCreateTask{})
	taskman.RegisterTask(GuestDisksSnapshotPolicyExecuteTask{})
}

func (self *SnapshotCreateTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	snapshot := obj.(*models.SSnapshot)
	self.DoDiskSnapshot(ctx, snapshot)
}

func (self *SnapshotCreateTask) TaskFailed(ctx context.Context, snapshot *models.SSnapshot, reason jsonutils.JSONObject) {
	snapshot.SetStatus(ctx, self.UserCred, api.SNAPSHOT_FAILED, reason.String())
	db.OpsLog.LogEvent(snapshot, db.ACT_SNAPSHOT_FAIL, reason, self.UserCred)
	logclient.AddActionLogWithStartable(self, snapshot, logclient.ACT_CREATE, reason, self.UserCred, false)
	self.SetStageFailed(ctx, reason)
}

func (self *SnapshotCreateTask) TaskComplete(ctx context.Context, snapshot *models.SSnapshot, data jsonutils.JSONObject) {
	snapshot.SetStatus(ctx, self.UserCred, api.SNAPSHOT_READY, "")
	db.OpsLog.LogEvent(snapshot, db.ACT_SNAPSHOT_DONE, snapshot.GetShortDesc(ctx), self.UserCred)
	logclient.AddActionLogWithStartable(self, snapshot, logclient.ACT_CREATE, snapshot.GetShortDesc(ctx), self.UserCred, true)
	notifyclient.EventNotify(ctx, self.UserCred, notifyclient.SEventNotifyParam{
		Obj:    snapshot,
		Action: notifyclient.ActionCreate,
	})
	self.SetStageComplete(ctx, nil)
}

func (self *SnapshotCreateTask) DoDiskSnapshot(ctx context.Context, snapshot *models.SSnapshot) {
	self.SetStage("OnCreateSnapshot", nil)
	if err := snapshot.GetRegionDriver().RequestCreateSnapshot(ctx, snapshot, self); err != nil {
		self.TaskFailed(ctx, snapshot, jsonutils.NewString(err.Error()))
	}
}

func (self *SnapshotCreateTask) OnCreateSnapshot(ctx context.Context, snapshot *models.SSnapshot, data jsonutils.JSONObject) {
	self.TaskComplete(ctx, snapshot, nil)
}

func (self *SnapshotCreateTask) OnCreateSnapshotFailed(ctx context.Context, snapshot *models.SSnapshot, data jsonutils.JSONObject) {
	self.TaskFailed(ctx, snapshot, data)
}

type GuestDisksSnapshotPolicyExecuteTask struct {
	taskman.STask
}

func (self *GuestDisksSnapshotPolicyExecuteTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	models.SetSnapshotPolicyTasksRunning()
	self.StartExecutePolicy(ctx, obj, data)
}

func (self *GuestDisksSnapshotPolicyExecuteTask) StartExecutePolicy(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	guestPolicies := models.SGuestPolicies{}
	self.Params.Unmarshal(&guestPolicies, "guest_policies")
	if guestPolicies.Length() == 0 {
		models.SetSnapshotPolicyTasksComplete()
		self.SetStageComplete(ctx, nil)
		return
	}
	if len(guestPolicies.SnapshotPolicyGuests) > 0 {
		snapshotPolicyGuest := guestPolicies.SnapshotPolicyGuests[0]
		guestPolicies.SnapshotPolicyGuests = guestPolicies.SnapshotPolicyGuests[1:]
		self.Params.Set("snapshot_policy_disks", jsonutils.Marshal(guestPolicies))
		self.SetStage("StartExecutePolicy", nil)
		if snapshotPolicyGuest.IsBackupPolicy {
			self.DoGuestBackupPolicy(ctx, &snapshotPolicyGuest)
		} else {
			self.DoGuestSnapshotPolicy(ctx, &snapshotPolicyGuest)
		}
	} else {
		snapshotPolicyDisk := guestPolicies.SnapshotPolicyDisks[0]
		guestPolicies.SnapshotPolicyDisks = guestPolicies.SnapshotPolicyDisks[1:]
		self.Params.Set("snapshot_policy_disks", jsonutils.Marshal(guestPolicies))
		self.SetStage("StartExecutePolicy", nil)
		if snapshotPolicyDisk.IsBackupPolicy {
			self.DoDiskBackupPolicy(ctx, &snapshotPolicyDisk)
		} else {
			self.DoDiskSnapshotPolicy(ctx, &snapshotPolicyDisk)
		}
	}
}

func (self *GuestDisksSnapshotPolicyExecuteTask) StartExecutePolicyFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	log.Errorf("Guest create snapshot failed %s: %s", obj.GetId(), data)
	self.StartExecutePolicy(ctx, obj, data)
}

func (self *GuestDisksSnapshotPolicyExecuteTask) DoDiskSnapshotPolicy(ctx context.Context, snapshotPolicyDisk *models.SSnapshotPolicyDisk) {
	disk, err := snapshotPolicyDisk.GetDisk()
	if err != nil {
		log.Errorf("disk snapshot policy %s failed get disk %s", snapshotPolicyDisk.SnapshotpolicyId, err)
		self.StartExecutePolicy(ctx, nil, nil)
		return
	}
	err = models.DiskManager.DoAutoSnapshot(ctx, self.UserCred, snapshotPolicyDisk, disk, self.GetTaskId())
	if err != nil {
		log.Errorf("DoAutoSnapshoto failed %s %s", disk.Id, err)
		db.OpsLog.LogEvent(disk, db.ACT_DISK_AUTO_SNAPSHOT_FAIL, err.Error(), self.UserCred)
		notifyclient.NotifySystemErrorWithCtx(ctx, disk.Id, disk.Name, db.ACT_DISK_AUTO_SNAPSHOT_FAIL, errors.Wrapf(err, "Disk auto create snapshot").Error())
		self.StartExecutePolicy(ctx, nil, nil)
		return
	}
}

func (self *GuestDisksSnapshotPolicyExecuteTask) DoGuestSnapshotPolicy(ctx context.Context, snapshotPolicyGuest *models.SSnapshotPolicyGuest) {
	guest, err := snapshotPolicyGuest.GetGuest()
	if err != nil {
		log.Errorf("guest snapshot policy %s failed get guest %s", snapshotPolicyGuest.SnapshotpolicyId, err)
		self.StartExecutePolicy(ctx, nil, nil)
		return
	}
	policy, err := snapshotPolicyGuest.GetSnapshotPolicy()
	if err != nil {
		log.Errorf("guest snapshot policy %s failed get policy %s", snapshotPolicyGuest.SnapshotpolicyId, err)
		self.StartExecutePolicy(ctx, nil, nil)
		return
	}

	err = models.GuestManager.DoAutoSnapshot(ctx, self.UserCred, policy, guest, self.GetTaskId())
	if err != nil {
		log.Errorf("DoAutoSnapshoto failed %s %s", guest.Id, err)
		db.OpsLog.LogEvent(guest, db.ACR_INSTANCE_AUTO_SNAPSHOT_FAIL, err.Error(), self.UserCred)
		notifyclient.NotifySystemErrorWithCtx(ctx, guest.Id, guest.Name, db.ACR_INSTANCE_AUTO_SNAPSHOT_FAIL, errors.Wrapf(err, "guest auto create snapshot").Error())
		self.StartExecutePolicy(ctx, nil, nil)
		return
	}
}

func (self *GuestDisksSnapshotPolicyExecuteTask) DoDiskBackupPolicy(ctx context.Context, snapshotPolicyDisk *models.SSnapshotPolicyDisk) {
	disk, err := snapshotPolicyDisk.GetDisk()
	if err != nil {
		log.Errorf("disk snapshot policy %s failed get disk %s", snapshotPolicyDisk.SnapshotpolicyId, err)
		self.StartExecutePolicy(ctx, nil, nil)
		return
	}
	err = models.DiskManager.DoAutoBackup(ctx, self.UserCred, snapshotPolicyDisk, disk, self.GetTaskId())
	if err != nil {
		log.Errorf("DoAutoBackup failed %s %s", disk.Id, err)
		db.OpsLog.LogEvent(disk, db.ACT_DISK_AUTO_BACKUP_FAIL, err.Error(), self.UserCred)
		notifyclient.NotifySystemErrorWithCtx(ctx, disk.Id, disk.Name, db.ACT_DISK_AUTO_BACKUP_FAIL, errors.Wrapf(err, "Disk auto create backup").Error())
		self.StartExecutePolicy(ctx, nil, nil)
		return
	}
}

func (self *GuestDisksSnapshotPolicyExecuteTask) DoGuestBackupPolicy(ctx context.Context, snapshotPolicyGuest *models.SSnapshotPolicyGuest) {
	guest, err := snapshotPolicyGuest.GetGuest()
	if err != nil {
		log.Errorf("guest snapshot policy %s failed get guest %s", snapshotPolicyGuest.SnapshotpolicyId, err)
		self.StartExecutePolicy(ctx, nil, nil)
		return
	}
	policy, err := snapshotPolicyGuest.GetSnapshotPolicy()
	if err != nil {
		log.Errorf("guest snapshot policy %s failed get policy %s", snapshotPolicyGuest.SnapshotpolicyId, err)
		self.StartExecutePolicy(ctx, nil, nil)
		return
	}

	err = models.GuestManager.DoAutoBackup(ctx, self.UserCred, policy, snapshotPolicyGuest, guest, self.GetTaskId())
	if err != nil {
		log.Errorf("DoAutoSnapshoto failed %s %s", guest.Id, err)
		db.OpsLog.LogEvent(guest, db.ACR_INSTANCE_AUTO_BACKUP_FAIL, err.Error(), self.UserCred)
		notifyclient.NotifySystemErrorWithCtx(ctx, guest.Id, guest.Name, db.ACR_INSTANCE_AUTO_BACKUP_FAIL, errors.Wrapf(err, "guest auto create backup").Error())
		self.StartExecutePolicy(ctx, nil, nil)
		return
	}
}
