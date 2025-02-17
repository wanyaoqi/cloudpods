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

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

// +onecloud:swagger-gen-model-singular=backuppolicy
// +onecloud:swagger-gen-model-plural=backuppolicies
type SBackupPolicyManager struct {
	db.SVirtualResourceBaseManager
	db.SExternalizedResourceBaseManager
	SManagedResourceBaseManager
	SCloudregionResourceBaseManager
}

type SBackupPolicy struct {
	SSnapshotPolicy

	// backup policy type: diskbackup, instance
	BackupType string
}

var BackupPolicyManager *SBackupPolicyManager

func init() {
	BackupPolicyManager = &SBackupPolicyManager{
		SVirtualResourceBaseManager: db.NewVirtualResourceBaseManager(
			SBackupPolicy{},
			"backup_policies_tbl",
			"backuppolicy",
			"backuppolicies",
		),
	}
	BackupPolicyManager.SetVirtualObject(BackupPolicyManager)
}

func (sp *SBackupPolicy) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerId mcclient.IIdentityProvider, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	//sp.StartCreateTask(ctx, userCred)
}

func (sp *SBackupPolicy) CustomizeDelete(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject, data jsonutils.JSONObject) error {
	//return sp.StartDeleteTask(ctx, userCred)
}
