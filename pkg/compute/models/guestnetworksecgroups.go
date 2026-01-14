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

	"gopkg.in/fatih/set.v0"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/validators"
	"yunion.io/x/onecloud/pkg/compute/options"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

type SGuestnetworksecgroupManager struct {
	db.SResourceBaseManager
	SSecurityGroupResourceBaseManager
	SGuestResourceBaseManager
}

var GuestnetworksecgroupManager *SGuestnetworksecgroupManager

func init() {
	db.InitManager(func() {
		GuestnetworksecgroupManager = &SGuestnetworksecgroupManager{
			SResourceBaseManager: db.NewResourceBaseManager(
				SGuestnetworksecgroup{},
				"guestnetworksecgroups_tbl",
				"guestnetworksecgroup",
				"guestnetworksecgroups",
			),
		}
		GuestnetworksecgroupManager.SetVirtualObject(GuestnetworksecgroupManager)
	})
}

type SGuestnetworksecgroup struct {
	db.SResourceBase

	GuestId                    string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required" index:"true"`
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

func (manager *SGuestnetworksecgroupManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.GuestnetworksecgroupListInput,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SGuestResourceBaseManager.ListItemFilter(ctx, q, userCred, query.ServerFilterListInput)
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

	q, err = manager.SGuestResourceBaseManager.OrderByExtraFields(ctx, q, userCred, query.ServerFilterListInput)
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

	q, err = manager.SGuestResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
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

	guestRows := manager.SGuestResourceBaseManager.FetchCustomizeColumns(ctx, userCred, query, objs, fields, isList)
	secgroupIds := make([]string, len(rows))
	for i := range rows {
		rows[i].GuestResourceInfo = guestRows[i]
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

func (self *SGuestnetworksecgroupManager) QueryDistinctExtraField(q *sqlchemy.SQuery, field string) (*sqlchemy.SQuery, error) {
	q, err := self.SResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = self.SGuestResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	q, err = self.SSecurityGroupResourceBaseManager.QueryDistinctExtraField(q, field)
	if err == nil {
		return q, nil
	}
	return q, httperrors.ErrNotFound
}

func (self *SGuestnetworksecgroupManager) GetGuestnetworksecgroups(guestId string, networkIndex int) ([]SSecurityGroup, error) {
	q := GuestnetworksecgroupManager.Query("secgroup_id").Equals("guest_id", guestId)
	if networkIndex >= 0 {
		q = q.Equals("network_index", networkIndex)
	}
	subQ := q.SubQuery()

	secgrpQuery := SecurityGroupManager.Query()
	secgrpQuery = secgrpQuery.In("id", subQ)

	secgroups := []SSecurityGroup{}
	err := db.FetchModelObjects(SecurityGroupManager, secgrpQuery, &secgroups)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return secgroups, nil
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

	guestnetwork, err := self.getGuestnetworkByIndex(*input.NetworkIndex)
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrap(err, "getGuestnetworkByIndex"))
	}

	secgroups, err := GuestnetworksecgroupManager.GetGuestnetworksecgroups(self.Id, *input.NetworkIndex)
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrap(err, "GetGuestnetworksecgroups"))
	}
	if len(secgroups)+len(input.SecgroupIds) > maxCount {
		return nil, httperrors.NewUnsupportOperationError("guest %s band to up to %d security groups", self.Name, maxCount)
	}

	network, err := guestnetwork.GetNetwork()
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrap(err, "GetNetwork"))
	}

	vpc, err := network.GetVpc()
	if err != nil {
		return nil, errors.Wrap(err, "GetVpc")
	}

	secgroupIds := []string{}
	for _, secgroup := range secgroups {
		secgroupIds = append(secgroupIds, secgroup.Id)
	}

	secgroupNames := []string{}
	for i := range input.SecgroupIds {
		secObj, err := validators.ValidateModel(ctx, userCred, SecurityGroupManager, &input.SecgroupIds[i])
		if err != nil {
			return nil, err
		}
		secgroup := secObj.(*SSecurityGroup)

		if utils.IsInStringArray(secObj.GetId(), secgroupIds) {
			return nil, httperrors.NewInputParameterError(
				"security group %s has already been assigned to guest %s network %d",
				secObj.GetName(), self.GetName(), input.NetworkIndex)
		}

		err = vpc.CheckSecurityGroupConsistent(secgroup)
		if err != nil {
			return nil, err
		}

		secgroupIds = append(secgroupIds, secgroup.GetId())
		secgroupNames = append(secgroupNames, secgroup.Name)
	}

	err = self.SaveNetworkSecgroups(ctx, userCred, secgroupIds, *input.NetworkIndex)
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrap(err, "SaveNetworkSecgroups"))
	}

	notes := map[string][]string{"secgroups": secgroupNames}
	logclient.AddActionLogWithContext(ctx, self, logclient.ACT_VM_ASSIGNSECGROUP, notes, userCred, true)
	return nil, self.StartSyncTask(ctx, userCred, true, "")
}

func (self *SGuest) PerformRevokeNetworkSecgroup(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.GuestRevokeNetworkSecgroupInput,
) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(self.Status, []string{api.VM_READY, api.VM_RUNNING, api.VM_SUSPEND}) {
		return nil, httperrors.NewInputParameterError("Cannot revoke security groups in status %s", self.Status)
	}
	if len(input.SecgroupIds) == 0 {
		return nil, nil
	}

	if input.NetworkIndex == nil || *input.NetworkIndex < 0 {
		return nil, httperrors.NewBadRequestError("input network index %#v is invalid", input.NetworkIndex)
	}
	_, err := self.getGuestnetworkByIndex(*input.NetworkIndex)
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrap(err, "getGuestnetworkByIndex"))
	}

	secgroups, err := GuestnetworksecgroupManager.GetGuestnetworksecgroups(self.Id, *input.NetworkIndex)
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrap(err, "GetGuestnetworksecgroups"))
	}
	secgroupMaps := map[string]string{}
	for _, secgroup := range secgroups {
		secgroupMaps[secgroup.Id] = secgroup.Name
	}

	secgroupNames := []string{}
	for i := range input.SecgroupIds {
		secObj, err := validators.ValidateModel(ctx, userCred, SecurityGroupManager, &input.SecgroupIds[i])
		if err != nil {
			return nil, err
		}
		secgrp := secObj.(*SSecurityGroup)
		_, ok := secgroupMaps[secgrp.GetId()]
		if !ok {
			return nil, httperrors.NewInputParameterError("security group %s network index %d not assigned to guest %s",
				secgrp.GetName(), *input.NetworkIndex, self.GetName())
		}
		delete(secgroupMaps, secgrp.GetId())
		secgroupNames = append(secgroupNames, secgrp.GetName())
	}

	secgrpIds := []string{}
	for secgroupId := range secgroupMaps {
		secgrpIds = append(secgrpIds, secgroupId)
	}

	err = self.SaveNetworkSecgroups(ctx, userCred, secgrpIds, *input.NetworkIndex)
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrap(err, "SaveNetworkSecgroups"))
	}

	notes := map[string][]string{"secgroups": secgroupNames}
	logclient.AddActionLogWithContext(ctx, self, logclient.ACT_VM_REVOKESECGROUP, notes, userCred, true)
	return nil, self.StartSyncTask(ctx, userCred, true, "")
}

func (self *SGuest) PerformSetNetworkSecgroup(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.GuestSetNetworkSecgroupInput,
) (jsonutils.JSONObject, error) {
	if !utils.IsInStringArray(self.Status, []string{api.VM_READY, api.VM_RUNNING, api.VM_SUSPEND}) {
		return nil, httperrors.NewInputParameterError("Cannot set security rules in status %s", self.Status)
	}
	if len(input.SecgroupIds) == 0 {
		return nil, httperrors.NewMissingParameterError("secgroup_ids")
	}

	maxCount := self.GetDriver().GetMaxSecurityGroupCount()
	if maxCount == 0 {
		return nil, httperrors.NewUnsupportOperationError("Cannot set security group for this guest %s", self.Name)
	}

	if len(input.SecgroupIds) > maxCount {
		return nil, httperrors.NewUnsupportOperationError("guest %s band to up to %d security groups", self.Name, maxCount)
	}

	guestnetwork, err := self.getGuestnetworkByIndex(*input.NetworkIndex)
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrap(err, "getGuestnetworkByIndex"))
	}

	network, err := guestnetwork.GetNetwork()
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrap(err, "GetNetwork"))
	}

	vpc, err := network.GetVpc()
	if err != nil {
		return nil, errors.Wrap(err, "GetVpc")
	}

	secgroupIds := []string{}
	secgroupNames := []string{}
	for i := range input.SecgroupIds {
		secObj, err := validators.ValidateModel(ctx, userCred, SecurityGroupManager, &input.SecgroupIds[i])
		if err != nil {
			return nil, err
		}
		secgrp := secObj.(*SSecurityGroup)

		err = vpc.CheckSecurityGroupConsistent(secgrp)
		if err != nil {
			return nil, err
		}

		if !utils.IsInStringArray(secgrp.GetId(), secgroupIds) {
			secgroupIds = append(secgroupIds, secgrp.GetId())
			secgroupNames = append(secgroupNames, secgrp.GetName())
		}
	}

	err = self.SaveNetworkSecgroups(ctx, userCred, secgroupIds, *input.NetworkIndex)
	if err != nil {
		return nil, httperrors.NewGeneralError(errors.Wrapf(err, "SaveNetworkSecgroups"))
	}

	notes := map[string][]string{"secgroups": secgroupNames}
	logclient.AddActionLogWithContext(ctx, self, logclient.ACT_VM_SETSECGROUP, notes, userCred, true)
	return nil, self.StartSyncTask(ctx, userCred, true, "")
}

func (self *SGuest) SaveNetworkSecgroups(
	ctx context.Context, userCred mcclient.TokenCredential, secgroupIds []string, networkIndex int,
) error {
	if len(secgroupIds) == 0 {
		return self.RevokeNetworkAllSecgroups(ctx, userCred, networkIndex)
	}
	oldIds := set.New(set.ThreadSafe)
	newIds := set.New(set.ThreadSafe)
	gnss, err := self.GetGuestNetworkSecgroups(networkIndex)
	if err != nil {
		return errors.Wrap(err, "GetGuestNetworkSecgroups")
	}
	secgroupMaps := map[string]SGuestnetworksecgroup{}
	for i := range gnss {
		oldIds.Add(gnss[i].SecgroupId)
		secgroupMaps[gnss[i].SecgroupId] = gnss[i]
	}
	for i := range secgroupIds {
		newIds.Add(secgroupIds[i])
	}
	for _, removed := range set.Difference(oldIds, newIds).List() {
		id := removed.(string)
		gns, ok := secgroupMaps[id]
		if ok {
			err = gns.Delete(ctx, userCred)
			if err != nil {
				return errors.Wrapf(err,
					"Delete guest network secgroup for guest %s network index %d secgroup %s",
					self.GetName(), networkIndex, id)
			}
		}
	}
	for _, added := range set.Difference(newIds, oldIds).List() {
		id := added.(string)
		err = self.newGuestNetworkSecgroup(ctx, id, networkIndex, false)
		if err != nil {
			return errors.Wrapf(err,
				"New guest network secgroup for guest %s network index %d with secgroup %s",
				self.GetName(), networkIndex, id)
		}
	}
	return nil
}

func (self *SGuest) newGuestNetworkSecgroup(ctx context.Context, secgroupId string, networkIndex int, isAdmin bool) error {
	gns := &SGuestnetworksecgroup{}
	gns.SetModelManager(GuestnetworksecgroupManager, gns)
	gns.GuestId = self.Id
	gns.SecgroupId = secgroupId
	gns.NetworkIndex = networkIndex
	gns.Admin = isAdmin
	return GuestnetworksecgroupManager.TableSpec().Insert(ctx, gns)
}

func (self *SGuest) GetGuestNetworkSecgroups(networkIndex int) ([]SGuestnetworksecgroup, error) {
	gss := []SGuestnetworksecgroup{}
	q := GuestnetworksecgroupManager.Query().Equals("guest_id", self.Id).Equals("network_index", networkIndex)
	err := db.FetchModelObjects(GuestnetworksecgroupManager, q, &gss)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return gss, nil
}

func (self *SGuest) RevokeNetworkAllSecgroups(ctx context.Context, userCred mcclient.TokenCredential, networkIndex int) error {
	gss, err := self.GetGuestNetworkSecgroups(networkIndex)
	if err != nil {
		return errors.Wrapf(err, "GetGuestNetworkSecgroups")
	}
	for i := range gss {
		err = gss[i].Delete(ctx, userCred)
		if err != nil {
			return errors.Wrap(err, "Delete")
		}
	}

	return self.newGuestNetworkSecgroup(ctx, options.Options.DefaultSecurityGroupId, networkIndex, false)
}

func (self *SGuestnetworksecgroupManager) getNetworkSecgroupJson(guestId string, networkIndex int) ([]*api.SecgroupJsonDesc, error) {
	ret := []*api.SecgroupJsonDesc{}
	secgroups, err := GuestnetworksecgroupManager.GetGuestnetworksecgroups(guestId, networkIndex)
	if err != nil {
		return nil, errors.Wrap(err, "GetSecgroups")
	}
	for _, secGrp := range secgroups {
		ret = append(ret, secGrp.getDesc())
	}
	return ret, nil
}
