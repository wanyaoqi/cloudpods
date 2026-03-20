// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package guestman

import (
	"yunion.io/x/onecloud/pkg/hostman/guestman/desc"
	"yunion.io/x/onecloud/pkg/hostman/metadata"
)

// sMetadataDescGetter implements metadata.DescGetter and metadata.DescGetterByGuestId
// for the host metadata /monitor telegraf influx path.
type sMetadataDescGetter struct {
	m *SGuestManager
}

// NewMetadataDescGetter returns a DescGetter suitable for metadata.Service.DescGetter.
// The dynamic type also implements metadata.DescGetterByGuestId for vm_id-based lookups.
func NewMetadataDescGetter(m *SGuestManager) metadata.DescGetter {
	return &sMetadataDescGetter{m: m}
}

func (g *sMetadataDescGetter) Get(ip string) *desc.SGuestDesc {
	var out *desc.SGuestDesc
	g.m.Servers.Range(func(_, v interface{}) bool {
		inst := v.(GuestRuntimeInstance)
		d := inst.GetDesc()
		if guestDescHasNicIP(d, ip) {
			out = d
			return false
		}
		return true
	})
	return out
}

func (g *sMetadataDescGetter) GetByGuestId(guestId string) *desc.SGuestDesc {
	inst, ok := g.m.GetServer(guestId)
	if !ok {
		return nil
	}
	return inst.GetDesc()
}

func guestDescHasNicIP(d *desc.SGuestDesc, ip string) bool {
	if d == nil || ip == "" {
		return false
	}
	for _, nic := range d.Nics {
		if nic.Ip == ip {
			return true
		}
		if nic.Vpc.MappedIpAddr != "" && nic.Vpc.MappedIpAddr == ip {
			return true
		}
		for _, vip := range nic.VirtualIps {
			if vip == ip {
				return true
			}
		}
		if nic.Ip6 != "" && nic.Ip6 == ip {
			return true
		}
	}
	return false
}
