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

package models

import (
	"context"
	"fmt"
	"time"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SSnapshotPolicyGuestManager struct {
	SGuestJointsManager
}

func (m *SSnapshotPolicyGuestManager) GetMasterFieldName() string {
	return "guest_id"
}

func (m *SSnapshotPolicyGuestManager) GetSlaveFieldName() string {
	return "snapshotpolicy_id"
}

var SnapshotPolicyGuestManager *SSnapshotPolicyGuestManager

func init() {
	db.InitManager(func() {
		SnapshotPolicyGuestManager = &SSnapshotPolicyGuestManager{
			SGuestJointsManager: NewGuestJointsManager(
				SSnapshotPolicyGuest{},
				"snapshot_policy_guests_tbl",
				"snapshot_policy_guest",
				"snapshot_policy_guests",
				SnapshotPolicyManager,
			),
		}
		SnapshotPolicyGuestManager.SetVirtualObject(SnapshotPolicyGuestManager)
	})
}

type SSnapshotPolicyGuest struct {
	SGuestJointsBase

	SnapshotpolicyId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"`

	// default is snapshot policy
	IsBackupPolicy     bool   `default:"false" list:"user" json:"is_backup_policy"`
	BackupStorageId    string `width:"36" charset:"ascii" nullable:"true" create:"optional" list:"user" index:"true"`
	SaveGuestIpMacAddr bool   `default:"false" list:"user" json:"save_guest_ip_mac_addr"`
}

func (self *SSnapshotPolicyGuest) GetGuest() (*SGuest, error) {
	guest, err := GuestManager.FetchById(self.GuestId)
	if err != nil {
		return nil, errors.Wrapf(err, "FetchById(%s)", self.GuestId)
	}
	return guest.(*SGuest), nil
}

func (self *SSnapshotPolicyGuest) GetSnapshotPolicy() (*SSnapshotPolicy, error) {
	policy, err := SnapshotPolicyManager.FetchById(self.SnapshotpolicyId)
	if err != nil {
		return nil, errors.Wrapf(err, "FetchById(%s)", self.SnapshotpolicyId)
	}
	return policy.(*SSnapshotPolicy), nil
}

func (man *SSnapshotPolicyGuestManager) RemoveByGuest(id string) error {
	_, err := sqlchemy.GetDB().Exec(
		fmt.Sprintf(
			"delete from %s where guest_id = ?",
			man.TableSpec().Name(),
		), id,
	)
	return err
}

func (man *SSnapshotPolicyGuestManager) RemoveBySnapshotpolicy(id string) error {
	_, err := sqlchemy.GetDB().Exec(
		fmt.Sprintf(
			"delete from %s where snapshotpolicy_id = ?",
			man.TableSpec().Name(),
		), id,
	)
	return err
}

func (manager *SSnapshotPolicyGuestManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.GuestSnapshotPolicyListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SGuestJointsManager.ListItemFilter(ctx, q, userCred, query.GuestJointsListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SGuestJointsManager.ListItemFilter")
	}

	if query.IsBackupPolicy != nil {
		q = q.Equals("is_backup_policy", *query.IsBackupPolicy)
	}
	return q, nil
}

func (manager *SSnapshotPolicyGuestManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.GuestSnapshotPolicyListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SGuestJointsManager.OrderByExtraFields(ctx, q, userCred, query.GuestJointsListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SGuestJointsManager.OrderByExtraFields")
	}
	return q, nil
}

func (guest *SGuest) validateDiskAutoCreateSnapshot() error {
	if !utils.IsInStringArray(guest.Status, []string{api.VM_RUNNING, api.VM_READY}) {
		return fmt.Errorf("Guest(%s) in status(%s) cannot do snapshot", guest.Id, guest.Status)
	}
	gds, err := guest.GetDisks()
	if err != nil {
		return errors.Wrap(err, "GetDisks")
	}
	guestDiskSize := 0
	for i := range gds {
		guestDiskSize += gds[i].DiskSize
	}
	storage, err := gds[0].GetStorage()
	if err != nil {
		return errors.Wrap(err, "GetStorage")
	}

	if storageFree := storage.GetFreeCapacity(); storageFree < int64(guestDiskSize) {
		return fmt.Errorf("Storage(%s) space not enough", storage.GetName())
	}
	return nil
}

func (guest *SGuest) validateDiskAutoCreateBackup(backupStorageId string) error {
	if !utils.IsInStringArray(guest.Status, []string{api.VM_RUNNING, api.VM_READY}) {
		return fmt.Errorf("Guest(%s) in status(%s) cannot do snapshot", guest.Id, guest.Status)
	}
	gds, err := guest.GetDisks()
	if err != nil {
		return errors.Wrap(err, "GetDisks")
	}
	guestDiskSize := 0
	for i := range gds {
		guestDiskSize += gds[i].DiskSize
	}
	storage, err := gds[0].GetStorage()
	if err != nil {
		return errors.Wrap(err, "GetStorage")
	}
	backupStorage := StorageManager.FetchStorageById(backupStorageId)
	if backupStorage == nil {
		return errors.Errorf("failed get bakcupstorage %s", backupStorageId)
	}
	if storageFree := storage.GetFreeCapacity(); storageFree < int64(guestDiskSize) {
		return fmt.Errorf("Storage(%s) space not enough", storage.GetName())
	}
	if storageFree := backupStorage.GetFreeCapacity(); storageFree < int64(guestDiskSize) {
		return fmt.Errorf("Storage(%s) space not enough", backupStorage.GetName())
	}
	return nil
}

func (manager *SGuestManager) DoAutoSnapshot(
	ctx context.Context, userCred mcclient.TokenCredential,
	policy *SSnapshotPolicy, guest *SGuest, parentTaskId string,
) error {
	err := guest.validateDiskAutoCreateSnapshot()
	if err != nil {
		return err
	}
	return guest.CreateSnapshotAuto(ctx, userCred, policy, parentTaskId)
}

func (guest *SGuest) CreateSnapshotAuto(
	ctx context.Context, userCred mcclient.TokenCredential, policy *SSnapshotPolicy, parentTaskId string,
) error {
	name := fmt.Sprintf("%s-auto-instance-snapshot-%d", guest.Name, time.Now().Unix())
	var expireAt *time.Time
	if policy.RetentionDays > 0 {
		expireTime := time.Now().AddDate(0, 0, policy.RetentionDays)
		expireAt = &expireTime
	}
	instanceSnapshot, err := InstanceSnapshotManager.CreateInstanceSnapshot(ctx, userCred, guest, name, false, false, expireAt)
	if err != nil {
		return err
	}
	err = guest.InheritTo(ctx, userCred, instanceSnapshot)
	if err != nil {
		return errors.Wrapf(err, "unable to inherit from guest %s to instance snapshot %s", guest.GetId(), instanceSnapshot.GetId())
	}
	err = guest.InstaceCreateSnapshot(ctx, userCred, instanceSnapshot, nil, parentTaskId)
	if err != nil {
		return errors.Wrap(err, "InstaceCreateSnapshot")
	}
	db.OpsLog.LogEvent(guest, db.ACR_INSTANCE_AUTO_SNAPSHOT, instanceSnapshot.Name, userCred)
	policy.ExecuteNotify(ctx, userCred, guest.GetName())
	return nil
}

func (manager *SGuestManager) DoAutoBackup(
	ctx context.Context, userCred mcclient.TokenCredential,
	policy *SSnapshotPolicy, guestSnapshotPolicy *SSnapshotPolicyGuest, guest *SGuest, parentTaskId string,
) error {
	err := guest.validateDiskAutoCreateBackup(guestSnapshotPolicy.BackupStorageId)
	if err != nil {
		return err
	}
	return guest.CreateInstanceBackupAuto(ctx, userCred, policy, guestSnapshotPolicy, parentTaskId)
}

func (guest *SGuest) CreateInstanceBackupAuto(
	ctx context.Context, userCred mcclient.TokenCredential, policy *SSnapshotPolicy, guestSnapshotPolicy *SSnapshotPolicyGuest, parentTaskId string,
) error {

	name := fmt.Sprintf("%s-auto-instance-snapshot-%d", guest.Name, time.Now().Unix())
	var expireAt *time.Time
	if policy.RetentionDays > 0 {
		expireTime := time.Now().AddDate(0, 0, policy.RetentionDays)
		expireAt = &expireTime
	}
	instanceBackup, err := InstanceBackupManager.CreateInstanceBackup(ctx, userCred, guest, name, guestSnapshotPolicy.BackupStorageId, guestSnapshotPolicy.SaveGuestIpMacAddr, expireAt)
	if err != nil {
		return httperrors.NewInternalServerError("create instance backup failed: %s", err)
	}
	err = guest.InheritTo(ctx, userCred, instanceBackup)
	if err != nil {
		return errors.Wrapf(err, "unable to inherit from guest %s to instance backup %s", guest.GetId(), instanceBackup.GetId())
	}
	err = guest.InstanceCreateBackup(ctx, userCred, instanceBackup, parentTaskId)
	if err != nil {
		return errors.Wrap(err, "InstanceCreateBackup")
	}
	db.OpsLog.LogEvent(guest, db.ACR_INSTANCE_AUTO_BACKUP, instanceBackup.Name, userCred)
	policy.ExecuteNotify(ctx, userCred, guest.GetName())
	return nil
}
