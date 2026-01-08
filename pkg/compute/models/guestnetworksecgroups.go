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
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SGuestnetworksecgroupManager struct {
	SGuestJointsManager
	SSecurityGroupResourceBaseManager
}

var GuestnetworksecgroupManager *SGuestnetworksecgroupManager

func init() {
	db.InitManager(func() {
		GuestnetworksecgroupManager = &SGuestnetworksecgroupManager{
			SGuestJointsManager: NewGuestJointsManager(
				SGuestnetworksecgroup{},
				"guestnetworksecgroups_tbl",
				"guestnetworksecgroup",
				"guestnetworksecgroups",
				SecurityGroupManager,
			),
		}
		GuestnetworksecgroupManager.SetVirtualObject(GuestnetworksecgroupManager)
	})
}

type SGuestnetworksecgroup struct {
	SGuestJointsBase

	SSecurityGroupResourceBase `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"`
	NetworkIndex               int `nullable:"false" list:"user" update:"user"`

	Admin bool `nullable:"false" default:"false" list:"user" create:"optional"`
}

func (manager *SGuestnetworksecgroupManager) GetSlaveFieldName() string {
	return "secgroup_id"
}

func (self *SGuestnetworksecgroup) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, self)
}

func (self *SGuestnetworksecgroup) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, self)
}

func (manager *SGuestnetworksecgroupManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.GuestnetworksecgroupListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SGuestJointsManager.ListItemFilter(ctx, q, userCred, query.GuestJointsListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SGuestJointsManager.ListItemFilter")
	}
	q, err = manager.SSecurityGroupResourceBaseManager.ListItemFilter(ctx, q, userCred, query.SecgroupFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSecurityGroupResourceBaseManager.ListItemFilter")
	}
	if query.NetworkIndex != nil {
		q = q.Equals("network_index", *query.NetworkIndex)
	}
	if query.IsAdmin {
		q = q.IsFalse("admin")
	}
	return q, nil
}

func (manager *SGuestnetworksecgroupManager) OrderByExtraFields(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.GuestnetworksecgroupListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SGuestJointsManager.OrderByExtraFields(ctx, q, userCred, query.GuestJointsListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SGuestJointsManager.OrderByExtraFields")
	}
	q, err = manager.SSecurityGroupResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.SecgroupFilterListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SSecurityGroupResourceBaseManager.OrderByExtraFields")
	}

	return q, nil
}

func (manager *SGuestnetworksecgroupManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SGuestJointsManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SGuestJointsManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SSecurityGroupResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SSecurityGroupResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SSecurityGroupResourceBaseManager.ListItemExportKeys")
		}
	}

	return q, nil
}

func (manager *SGuestnetworksecgroupManager) FetchCustomizeColumns(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	objs []interface{},
	fields stringutils2.SSortedStrings,
	isList bool,
) []api.GuestnetworksecgroupDetails {
	rows := make([]api.GuestnetworksecgroupDetails, len(objs))

	guestRows := manager.SGuestJointsManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	secgroupIds := make([]string, len(rows))
	for i := range rows {
		rows[i].GuestJointResourceDetails = guestRows[i]
		rows[i].NetworkIndex = objs[i].(*SGuestnetworksecgroup).NetworkIndex
		rows[i].Admin = objs[i].(*SGuestnetworksecgroup).Admin
		secgroupIds[i] = objs[i].(*SGuestnetworksecgroup).SecgroupId
	}

	secgroupIdMaps, err := db.FetchIdNameMap2(SecurityGroupManager, secgroupIds)
	if err != nil {
		log.Errorf("FetchIdNameMap2 fail %s", err)
		return rows
	}

	for i := range rows {
		if name, ok := secgroupIdMaps[secgroupIds[i]]; ok {
			rows[i].Secgroup = name
		}
	}

	return rows
}

// guest network attach secgroup
func (self *SGuest) PerformNetworkAddSecgroup(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.GuestNetworkAddSecgroupInput,
) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(self.Status, []string{api.VM_READY, api.VM_RUNNING, api.VM_SUSPEND}) {
		return nil, httperrors.NewInputParameterError("Cannot add security groups in status %s", self.Status)
	}
	if input.NetworkIndex == nil || *input.NetworkIndex < 0 {
		return nil, httperrors.NewBadRequestError("input network index %#v is invalid", input.NetworkIndex)
	}

	maxCount := self.GetDriver().GetMaxSecurityGroupCount()
	if maxCount == 0 {
		return nil, httperrors.NewUnsupportOperationError("Cannot add security groups for hypervisor %s", self.Hypervisor)
	}

	if len(input.SecgroupIds) == 0 {
		return nil, httperrors.NewMissingParameterError("secgroup_ids")
	}

	guestnetwork, err := GuestnetworkManager.FetchByGuestIdIndex(self.Id, int8(*input.NetworkIndex))
	if err != nil {
		return nil, errors.Wrap(err, "failed get guest network")
	}

}
