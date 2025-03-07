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
	"database/sql"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/taskman"
	"yunion.io/x/onecloud/pkg/compute/models"
)

func init() {
	taskman.RegisterTask(SnapshotPolicyCleanupTask{})
}

type SnapshotPolicyCleanupTask struct {
	taskman.STask
}

func (self *SnapshotPolicyCleanupTask) taskCompleted(ctx context.Context, data jsonutils.JSONObject) {
	log.Infof("SnapshotPolicyCleanupTask completed %s", data)
	models.SetSnapshotPolicyCleanupTasksComplete()
	self.SetStageComplete(ctx, nil)
}

func (self *SnapshotPolicyCleanupTask) OnInit(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	models.SetSnapshotPolicyCleanupTasksRunning()
	self.StartCleanSnapshots(ctx, obj, data)
}

func (self *SnapshotPolicyCleanupTask) StartCleanSnapshots(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	now, err := self.Params.GetTime("tick")
	if err != nil {
		log.Errorf("StartCleanSnapshots failed get tick time")
		self.StartCleanInstanceSnapshots(ctx, obj, data)
		return
	}
	var snapshots = make([]models.SSnapshot, 0)
	q := models.SnapshotManager.Query().
		Equals("fake_deleted", false).
		Equals("created_by", compute.SNAPSHOT_AUTO).
		LE("expired_at", now)
	log.Errorf("StartCleanSnapshots %s", q.DebugString())
	err = q.All(&snapshots)
	if err == sql.ErrNoRows {
		self.StartCleanInstanceSnapshots(ctx, obj, data)
		return
	} else if err != nil {
		log.Errorf("failed get snapshot %s", err)
		self.StartCleanInstanceSnapshots(ctx, obj, data)
		return
	}
	snapshotIds := make([]string, len(snapshots))
	for i := range snapshots {
		snapshotIds[i] = snapshots[i].Id
	}
	self.SetStage("OnDeleteSnapshot", nil)
	self.StartSnapshotsDelete(ctx, snapshotIds)
}

func (self *SnapshotPolicyCleanupTask) StartSnapshotsDelete(ctx context.Context, snapshotIds []string) {
	snapshotId := snapshotIds[0]
	snapshotIds = snapshotIds[1:]
	data := jsonutils.Marshal(map[string]interface{}{"snapshots": snapshotIds})
	self.SaveParams(data.(*jsonutils.JSONDict))

	iSnapshot, err := models.SnapshotManager.FetchById(snapshotId)
	if err != nil {
		log.Errorf("failed get snapshot %s: %s", snapshotId, err)
		self.OnDeleteSnapshot(ctx, nil, nil)
		return
	}
	snapshot := iSnapshot.(*models.SSnapshot)
	if snapshot.Status == compute.SNAPSHOT_DELETING {
		self.OnDeleteSnapshot(ctx, snapshot, nil)
		return
	}
	disk, err := snapshot.GetDisk()
	if err != nil && err != sql.ErrNoRows {
		log.Errorf("SnapshotPolicyCleanupTask snapshot %s get disk failed", snapshot.Id)
		self.OnDeleteSnapshot(ctx, snapshot, nil)
		return
	}
	if disk != nil {
		if disk.Status != api.DISK_READY {
			log.Errorf("SnapshotPolicyCleanupTask skip snapshot %s disk status %s", snapshot.Id, disk.Status)
			self.OnDeleteSnapshot(ctx, snapshot, nil)
			return
		}
		if guest := disk.GetGuest(); guest != nil {
			if !utils.IsInStringArray(guest.Status, []string{api.VM_RUNNING, api.VM_READY}) {
				log.Errorf("SnapshotPolicyCleanupTask skip snapshot %s guest status %s ", snapshot.Id, guest.Status)
				self.OnDeleteSnapshot(ctx, snapshot, nil)
				return
			}
		}
	}

	err = snapshot.StartSnapshotDeleteTask(ctx, self.UserCred, false, self.GetId(), 0, 0)
	if err != nil {
		log.Errorf("failed OnDeleteInstanceBackupFailed %s", err)
		self.OnDeleteSnapshot(ctx, nil, nil)
	}
}

func (self *SnapshotPolicyCleanupTask) OnDeleteSnapshot(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	var snapshotIds = make([]string, 0)
	err := self.Params.Unmarshal(&snapshotIds, "snapshots")
	if err != nil {
		log.Errorf("failed get snapshots %s", err)
		self.StartCleanInstanceSnapshots(ctx, obj, data)
		return
	}
	if len(snapshotIds) > 0 {
		self.StartSnapshotsDelete(ctx, snapshotIds)
	} else {
		self.StartCleanInstanceSnapshots(ctx, obj, data)
	}
}

func (self *SnapshotPolicyCleanupTask) OnDeleteSnapshotFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	log.Errorf("snapshot delete faield %s", data)
	self.OnDeleteSnapshot(ctx, obj, data)
}

// StartCleanInstanceSnapshots
func (self *SnapshotPolicyCleanupTask) StartCleanInstanceSnapshots(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	now, err := self.Params.GetTime("tick")
	if err != nil {
		log.Errorf("StartCleanInstanceSnapshots failed get tick time")
		self.StartCleanDiskBackups(ctx, obj, data)
		return
	}

	var ips = make([]models.SInstanceSnapshot, 0)
	err = models.InstanceSnapshotManager.Query().
		LE("expired_at", now).All(&ips)
	if err == sql.ErrNoRows {
		self.StartCleanDiskBackups(ctx, obj, data)
		return
	} else if err != nil {
		log.Errorf("failed get snapshot %s", err)
		self.StartCleanDiskBackups(ctx, obj, data)
		return
	}
	ipsIds := make([]string, len(ips))
	for i := range ips {
		ipsIds[i] = ips[i].Id
	}
	self.SetStage("OnDeleteInstanceSnapshot", nil)
	self.StartInstanceSnapshotsDelete(ctx, ipsIds)
}

func (self *SnapshotPolicyCleanupTask) StartInstanceSnapshotsDelete(ctx context.Context, instanceSnapshotIds []string) {
	instanceSnapshotId := instanceSnapshotIds[0]
	instanceSnapshotIds = instanceSnapshotIds[1:]
	self.Params.Set("instance_snapshots", jsonutils.Marshal(instanceSnapshotIds))
	self.SaveParams(nil)

	iSnapshot, err := models.InstanceSnapshotManager.FetchById(instanceSnapshotId)
	if err != nil {
		log.Errorf("failed get instance snapshot %s: %s", instanceSnapshotId, err)
		self.OnDeleteInstanceSnapshot(ctx, nil, nil)
		return
	}
	instanceSnapshot := iSnapshot.(*models.SInstanceSnapshot)
	if instanceSnapshot.Status == compute.INSTANCE_SNAPSHOT_START_DELETE {
		self.OnDeleteInstanceSnapshot(ctx, nil, nil)
		return
	}
	guest, _ := instanceSnapshot.GetGuest()
	if guest != nil && !utils.IsInStringArray(guest.Status, []string{api.VM_READY, api.VM_RUNNING}) {
		log.Errorf("can't delete instance snapshot %s in guest status %s", instanceSnapshot.Id, guest.Status)
		self.OnDeleteInstanceSnapshot(ctx, nil, nil)
	}
	err = instanceSnapshot.StartInstanceSnapshotDeleteTask(ctx, self.UserCred, self.GetTaskId())
	if err != nil {
		log.Errorf("failed StartInstanceSnapshotDeleteTask for instance snapshot %s", instanceSnapshotId)
		self.OnDeleteInstanceSnapshot(ctx, nil, nil)
	}
}

func (self *SnapshotPolicyCleanupTask) OnDeleteInstanceSnapshot(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	var instanceSnapshotIds = make([]string, 0)
	err := self.Params.Unmarshal(&instanceSnapshotIds, "instance_snapshots")
	if err != nil {
		log.Errorf("failed get instance_snapshots %s", err)
		self.StartCleanDiskBackups(ctx, obj, data)
		return
	}
	if len(instanceSnapshotIds) > 0 {
		self.StartInstanceSnapshotsDelete(ctx, instanceSnapshotIds)
	} else {
		self.StartCleanDiskBackups(ctx, obj, data)
	}
}

func (self *SnapshotPolicyCleanupTask) OnDeleteInstanceSnapshotFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	log.Errorf("instance snapshot delete faield %s", data)
	self.OnDeleteInstanceSnapshot(ctx, obj, data)
}

// StartCleanDiskBackups
func (self *SnapshotPolicyCleanupTask) StartCleanDiskBackups(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	now, err := self.Params.GetTime("tick")
	if err != nil {
		log.Errorf("StartCleanDiskBackups failed get tick time")
		self.StartCleanInstanceBackups(ctx, obj, data)
		return
	}

	var diskBackups = make([]models.SDiskBackup, 0)
	err = models.DiskBackupManager.Query().
		LE("expired_at", now).All(&diskBackups)
	if err == sql.ErrNoRows {
		self.StartCleanInstanceBackups(ctx, obj, data)
		return
	} else if err != nil {
		log.Errorf("failed get snapshot %s", err)
		self.StartCleanInstanceBackups(ctx, obj, data)
		return
	}
	backupIds := make([]string, len(diskBackups))
	for i := range diskBackups {
		backupIds[i] = diskBackups[i].Id
	}
	self.SetStage("OnDeleteDiskBackup", nil)
	self.StartDiskBackupDelete(ctx, backupIds)
}

func (self *SnapshotPolicyCleanupTask) StartDiskBackupDelete(ctx context.Context, backupIds []string) {
	diskBackupId := backupIds[0]
	backupIds = backupIds[1:]
	self.Params.Set("disk_backups", jsonutils.Marshal(backupIds))
	self.SaveParams(nil)

	iBackup, err := models.DiskBackupManager.FetchById(diskBackupId)
	if err != nil {
		log.Errorf("failed get disk backup %s: %s", diskBackupId, err)
		self.OnDeleteDiskBackup(ctx, nil, nil)
		return
	}
	diskBackup := iBackup.(*models.SDiskBackup)
	if diskBackup.Status == compute.BACKUP_STATUS_DELETING {
		self.OnDeleteDiskBackup(ctx, nil, nil)
		return
	}
	err = diskBackup.StartBackupDeleteTask(ctx, self.UserCred, self.GetTaskId(), false)
	if err != nil {
		log.Errorf("failed StartBackupDeleteTask for disk backup %s", diskBackupId)
		self.OnDeleteDiskBackup(ctx, nil, nil)
	}
}

func (self *SnapshotPolicyCleanupTask) OnDeleteDiskBackup(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	var diskBackupIds = make([]string, 0)
	err := self.Params.Unmarshal(&diskBackupIds, "disk_backups")
	if err != nil {
		log.Errorf("failed get disk_backups %s", err)
		self.StartCleanInstanceBackups(ctx, obj, data)
		return
	}
	if len(diskBackupIds) > 0 {
		self.StartDiskBackupDelete(ctx, diskBackupIds)
	} else {
		self.StartCleanInstanceBackups(ctx, obj, data)
	}
}

func (self *SnapshotPolicyCleanupTask) OnDeleteDiskBackupFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	log.Errorf("disk backup delete faield %s", data)
	self.OnDeleteDiskBackup(ctx, obj, data)
}

// StartCleanInstanceBackups
func (self *SnapshotPolicyCleanupTask) StartCleanInstanceBackups(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	now, err := self.Params.GetTime("tick")
	if err != nil {
		log.Errorf("StartCleanInstanceBackups failed get tick time")
		self.taskCompleted(ctx, data)
		return
	}

	var instanceBackups = make([]models.SInstanceBackup, 0)
	err = models.DiskBackupManager.Query().
		LE("expired_at", now).All(&instanceBackups)
	if err == sql.ErrNoRows {
		self.taskCompleted(ctx, nil)
		return
	} else if err != nil {
		log.Errorf("failed get snapshot %s", err)
		self.taskCompleted(ctx, nil)
		return
	}
	ipsIds := make([]string, len(instanceBackups))
	for i := range instanceBackups {
		ipsIds[i] = instanceBackups[i].Id
	}
	self.SetStage("OnDeleteInstanceBackup", nil)
	self.StartInstanceBackupDelete(ctx, ipsIds)
}

func (self *SnapshotPolicyCleanupTask) StartInstanceBackupDelete(ctx context.Context, backupIds []string) {
	instanceBackupId := backupIds[0]
	backupIds = backupIds[1:]
	self.Params.Set("instance_backups", jsonutils.Marshal(backupIds))
	self.SaveParams(nil)

	iBackup, err := models.InstanceBackupManager.FetchById(instanceBackupId)
	if err != nil {
		log.Errorf("failed get instance backup %s: %s", instanceBackupId, err)
		self.OnDeleteInstanceBackup(ctx, nil, nil)
		return
	}
	instanceBackup := iBackup.(*models.SInstanceBackup)
	if instanceBackup.Status == compute.INSTANCE_BACKUP_STATUS_DELETING {
		self.OnDeleteInstanceBackup(ctx, nil, nil)
		return
	}
	err = instanceBackup.StartInstanceBackupDeleteTask(ctx, self.UserCred, self.GetTaskId(), false)
	if err != nil {
		log.Errorf("failed StartBackupDeleteTask for instance backup %s", instanceBackupId)
		self.OnDeleteInstanceBackup(ctx, nil, nil)
	}
}

func (self *SnapshotPolicyCleanupTask) OnDeleteInstanceBackup(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	var instanceBackupIds = make([]string, 0)
	err := self.Params.Unmarshal(&instanceBackupIds, "instance_backups")
	if err != nil {
		log.Errorf("failed get instance_backups %s", err)
		self.taskCompleted(ctx, data)
		return
	}
	if len(instanceBackupIds) > 0 {
		self.StartInstanceBackupDelete(ctx, instanceBackupIds)
	} else {
		self.taskCompleted(ctx, data)
	}
}

func (self *SnapshotPolicyCleanupTask) OnDeleteInstanceBackupFailed(ctx context.Context, obj db.IStandaloneModel, data jsonutils.JSONObject) {
	log.Errorf("instance backup delete faield %s", data)
	self.OnDeleteInstanceBackup(ctx, obj, data)
}
