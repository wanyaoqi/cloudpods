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

package fsdriver

import (
	"fmt"

	"yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/apis"
	computeapi "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	deployapi "yunion.io/x/onecloud/pkg/hostman/hostdeployer/apis"
)

func unmarshalNicConfigs(jsonNics []jsonutils.JSONObject) []types.SServerNic {
	nics := make([]types.SServerNic, 0)
	for i := range jsonNics {
		nic := types.SServerNic{}
		if err := jsonNics[i].Unmarshal(&nic); err == nil {
			nics = append(nics, nic)
		}
	}
	return nics
}

func findTeamingNic(nics []*types.SServerNic, mac string) *types.SServerNic {
	for i := range nics {
		if nics[i].TeamWith == mac {
			return nics[i]
		}
	}
	return nil
}

func ToServerNics(nics []*deployapi.Nic, hypervisor string) []*types.SServerNic {
	ret := make([]*types.SServerNic, len(nics))
	for i := 0; i < len(nics); i++ {
		domain := nics[i].Domain
		if apis.IsIllegalSearchDomain(domain) {
			domain = ""
		}
		nic := nics[i]
		snic := &types.SServerNic{
			Name:      nic.Name,
			Index:     int(nic.Index),
			Bridge:    nic.Bridge,
			Domain:    domain,
			Ip:        nic.Ip,
			Vlan:      int(nic.Vlan),
			Driver:    nic.Driver,
			Masklen:   int(nic.Masklen),
			Virtual:   nic.Virtual,
			Manual:    nic.Manual,
			WireId:    nic.WireId,
			NetId:     nic.NetId,
			Mac:       nic.Mac,
			BandWidth: int(nic.Bw),
			Dns:       nic.Dns,
			Net:       nic.Net,
			Interface: nic.Interface,
			Gateway:   nic.Gateway,
			Ifname:    nic.Ifname,
			Routes:    deployapi.ConvertRoutes(nic.Routes),
			NicType:   compute.TNicType(nic.NicType),
			LinkUp:    nic.LinkUp,
			Mtu:       int16(nic.Mtu),
			TeamWith:  nic.TeamWith,
			IsDefault: nic.IsDefault,

			Ip6:      nic.Ip6,
			Masklen6: int(nic.Masklen6),
			Gateway6: nic.Gateway6,

			VlanInterface: shouldConfigureVlanInterface(nic, hypervisor),
		}
		ret[i] = snic
	}
	return ret
}

func shouldConfigureVlanInterface(nic *deployapi.Nic, hypervisor string) bool {
	return hypervisor == api.HYPERVISOR_BAREMETAL && 
		   nic.Manual && 
		   nic.Vlan > 1
}

func convertNicConfigs(nics []*types.SServerNic) ([]*types.SServerNic, []*types.SServerNic) {
	allNics := make([]*types.SServerNic, 0)
	bondNics := make([]*types.SServerNic, 0)

	var netDevPrefix = GetNetDevPrefix(nics)
	for i := range nics {
		// skip nics without mac
		if len(nics[i].Mac) == 0 {
			continue
		}
		// skip team slave nics
		if len(nics[i].TeamWith) > 0 {
			continue
		}
		teamNic := findTeamingNic(nics, nics[i].Mac)
		if teamNic == nil {
			// no teaming nic
			nnic := nics[i]
			if nnic.NicType == computeapi.NIC_TYPE_INFINIBAND {
				nnic.Name = fmt.Sprintf("%s%d", GetIBNetDevPrefix(), nnic.Index)
			} else {
				nnic.Name = fmt.Sprintf("%s%d", netDevPrefix, nnic.Index)
			}
			allNics = append(allNics, nnic)
			continue
		}
		// copy nic into master and nnic
		master := nics[i]
		nnic := *nics[i]
		tnic := *teamNic
		nnic.Name = fmt.Sprintf("%s%d", netDevPrefix, nnic.Index)
		nnic.TeamingMaster = master
		nnic.Ip = ""
		nnic.Masklen = 0
		nnic.Gateway = ""
		nnic.Ip6 = ""
		nnic.Masklen6 = 0
		nnic.Gateway6 = ""
		nnic.IsDefault = false
		tnic.Name = fmt.Sprintf("%s%d", netDevPrefix, tnic.Index)
		tnic.TeamingMaster = master
		tnic.Ip = ""
		tnic.Masklen = 0
		tnic.Gateway = ""
		tnic.Ip6 = ""
		tnic.Masklen6 = 0
		tnic.Gateway6 = ""
		tnic.IsDefault = false
		master.Name = fmt.Sprintf("bond%d", len(bondNics))
		master.TeamingSlaves = []*types.SServerNic{&nnic, &tnic}
		// why reset master.Mac?
		// master.Mac = ""
		allNics = append(allNics, &nnic, &tnic, master)
		bondNics = append(bondNics, master)
	}
	return allNics, bondNics
}
