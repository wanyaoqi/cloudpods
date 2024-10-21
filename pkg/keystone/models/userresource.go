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
	"database/sql"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/identity"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SUserResourceBaseManager struct{}

func (manager *SUserResourceBaseManager) FetchUser(userId, userDomainId string, userCred mcclient.TokenCredential) (db.IModel, error) {
	var ownerId mcclient.IIdentityProvider
	if len(userDomainId) > 0 {
		domain, err := DomainManager.FetchDomainByIdOrName(userDomainId)
		if err != nil {
			if errors.Cause(err) == sql.ErrNoRows {
				return nil, httperrors.NewResourceNotFoundError2(DomainManager.Keyword(), userDomainId)
			} else {
				return nil, errors.Wrap(err, "DomainManager.FetchDomainByIdOrName")
			}
		}
		ownerId = &db.SOwnerId{
			Domain:   domain.Name,
			DomainId: domain.Id,
		}

	} else {
		ownerId = userCred
	}
	userObj, err := UserManager.FetchByIdOrName(ownerId, userId)
	if err != nil {
		if errors.Cause(err) == sql.ErrNoRows {
			return nil, httperrors.NewResourceNotFoundError2(UserManager.Keyword(), userId)
		} else {
			return nil, errors.Wrap(err, "UserManager.FetchByIdOrName")
		}
	}
	return userObj, nil
}

func (manager *SUserResourceBaseManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.UserFilterListInput,
) (*sqlchemy.SQuery, error) {
	if len(query.UserId) > 0 {
		userObj, err := manager.FetchUser(query.UserId, query.UserDomainId, userCred)
		if err != nil {
			return nil, err
		}
		q = q.Equals("user_id", userObj.GetId())
	}
	return q, nil
}
