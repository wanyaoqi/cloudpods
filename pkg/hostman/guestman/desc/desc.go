package desc

import (
	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
)

type SGuestPorjectDesc struct {
	Tenant        string
	TenantId      string
	DomainId      string
	ProjectDomain string
}

type SGuestRegionDesc struct {
	Zone     string
	Domain   string
	HostId   string
	Hostname string
}

type SGuestControlDesc struct {
	IsDaemon bool
	IsMaster bool
	IsSlave  bool

	ScalingGroupId string
	EncryptKeyId   string
}

type SGuestHardwareDesc struct {
	Cpu       int64
	Mem       int64
	Machine   string
	Bios      string
	BootOrder string

	Vga       string
	VgaDevice *SGuestVga

	Vdi       string
	VdiDevice *SGuestVdi

	VritioSerial    *SGuestVirtioSerial
	Cdrom           *SGuestCdrom
	Disks           []*SGuestDisk
	Nics            []*SGuestNetwork
	NicsStandby     []*SGuestNetwork
	IsolatedDevices []*api.IsolatedDeviceJsonDesc
}

// %04x:%02x:%02x.%x, domain, bus, device, function
type SPCIDeviceAddr struct {
	Domain   int
	Bus      int
	Device   int
	Function int

	Controller    string
	MultiFunction bool
}

type SGuestDisk struct {
	api.GuestdiskJsonDesc
}

type SGuestCdrom struct {
	api.GuestcdromJsonDesc
}

type SGuestNetwork struct {
	api.GuestnetworkJsonDesc
}

type SGuestVga struct {
	*SPCIDeviceAddr
	Vga string
}

type SGuestVdi struct {
	*SPCIDeviceAddr
	Vdi string
}

// For qga sock
type SGuestVirtioSerial struct {
	*SPCIDeviceAddr
}

type SGuestDesc struct {
	SGuestPorjectDesc
	SGuestRegionDesc
	SGuestControlDesc
	SGuestHardwareDesc

	Name         string
	Uuid         string
	OsName       string
	Pubkey       string
	Keypair      string
	Secgroup     string
	Flavor       string
	UserData     string
	Metadata     map[string]string
	ExtraOptions map[string]jsonutils.JSONObject
}
