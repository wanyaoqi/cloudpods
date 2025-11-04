package mdadm

import (
	"yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/baremetal/utils/raid"
	"yunion.io/x/onecloud/pkg/compute/baremetal"
)

func init() {
	raid.RegisterDriver(baremetal.DISK_DRIVER_MDADM, NewMdadmRaid)
}

type MdadmRaid struct {
	term raid.IExecTerm
}

func (*MdadmRaid) ParsePhyDevs() error {
	panic("implement me")
}

func (*MdadmRaid) GetName() string {
	return baremetal.DISK_DRIVER_MDADM
}

func (*MdadmRaid) GetAdapters() []raid.IRaidAdapter {
	return []raid.IRaidAdapter{new(MdadmRaidAdaptor)}
}

func (*MdadmRaid) PreBuildRaid(confs []*compute.BaremetalDiskConfig, adapterIdx int) error {
	return nil
}

func (*MdadmRaid) CleanRaid() error {
	return nil
}

func NewMdadmRaid(term raid.IExecTerm) raid.IRaidDriver {
	return &MdadmRaid{
		term: term,
	}
}

type MdadmRaidAdaptor struct {
}

func (adapter *MdadmRaidAdaptor) PreBuildRaid(confs []*compute.BaremetalDiskConfig) error {
	return nil
}

func (adapter *MdadmRaidAdaptor) GetLogicVolumes() ([]*raid.RaidLogicalVolume, error) {
	return nil, nil
}

func (adapter *MdadmRaidAdaptor) RemoveLogicVolumes() error {
	return nil
}

func (adapter *MdadmRaidAdaptor) GetDevices() []*baremetal.BaremetalStorage {
	panic("implement me")
}

func (adapter *MdadmRaidAdaptor) BuildRaid0(devs []*baremetal.BaremetalStorage, conf *compute.BaremetalDiskConfig) error {
	panic("implement me")
}

func (adapter *MdadmRaidAdaptor) BuildRaid1(devs []*baremetal.BaremetalStorage, conf *compute.BaremetalDiskConfig) error {
	panic("implement me")
}

func (adapter *MdadmRaidAdaptor) BuildRaid5(devs []*baremetal.BaremetalStorage, conf *compute.BaremetalDiskConfig) error {
	panic("implement me")
}

func (adapter *MdadmRaidAdaptor) BuildRaid10(devs []*baremetal.BaremetalStorage, conf *compute.BaremetalDiskConfig) error {
	panic("implement me")
}

func (adapter *MdadmRaidAdaptor) BuildNoneRaid(devs []*baremetal.BaremetalStorage) error {
	panic("implement me")
}

func (adapter *MdadmRaidAdaptor) PostBuildRaid() error {
	panic("implement me")
}

func NewMdadmRaidAdaptor() raid.IRaidAdapter {
	return &MdadmRaidAdaptor{}
}

func (adapter *MdadmRaidAdaptor) GetIndex() int {
	return 0
}
