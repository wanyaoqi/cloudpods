package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/stringutils2"
)

// +onecloud:swagger-gen-ignore
type SGuestIsolatedDeviceManager struct {
	SGuestJointsManager
	SIsolatedDeviceResourceBaseManager
}

var GuestIsolatedDeviceManager *SGuestIsolatedDeviceManager

func init() {
	db.InitManager(func() {
		GuestIsolatedDeviceManager = &SGuestIsolatedDeviceManager{
			SGuestJointsManager: NewGuestJointsManager(
				SGuestIsolatedDevice{},
				"guestisolateddevices_tbl",
				"guestisolateddevice",
				"guestisolateddevices",
				IsolatedDeviceManager,
			),
		}
		GuestIsolatedDeviceManager.SetVirtualObject(GuestIsolatedDeviceManager)
		GuestIsolatedDeviceManager.TableSpec().AddIndex(false, "isolated_device_id", "guest_id")
	})
}

// +onecloud:model-api-gen
type SGuestIsolatedDevice struct {
	SGuestJointsBase
	SIsolatedDeviceResourceBase

	// guest isolated device memory size limit
	DeviceMemorySize int `nullable:"true" default:"0" list:"user" update:"user" create:"optional"`
	// guest isolated device  Streaming Multiprocessor Utilization limit
	SmUtilLimit int `nullable:"true" default:"0" list:"user" update:"user" create:"optional"`
	// gpu device work type: HPC VGA
	GpuType string `width:"16" charset:"ascii" nullable:"true" default:"" index:"true" list:"user" create:"optional" update:"user"`

	// guest network index
	NetworkIndex int `nullable:"true" default:"-1" list:"user" update:"user"`
	// guest disk index
	DiskIndex int8 `nullable:"true" default:"-1" list:"user" update:"user"`

	Index int8 `nullable:"false" default:"0" list:"user" update:"user"`
}

func (manager *SGuestIsolatedDeviceManager) GetSlaveFieldName() string {
	return "isolated_device_id"
}

func (self *SGuestIsolatedDevice) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, self)
}

func (self *SGuestIsolatedDevice) Detach(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DetachJoint(ctx, userCred, self)
}

func (manager *SGuestIsolatedDeviceManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.GuestIsolatedDeviceListInput,
) (*sqlchemy.SQuery, error) {
	var err error
	q, err = manager.SGuestJointsManager.ListItemFilter(ctx, q, userCred, query.GuestJointsListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SGuestJointsManager.ListItemFilter")
	}
	if len(query.IsolateDeviceIds) > 0 {
		query.IsolatedDeviceListInput.Ids = query.IsolateDeviceIds
	}
	q, err = manager.SIsolatedDeviceResourceBaseManager.ListItemFilter(ctx, q, userCred, query.IsolatedDeviceListInput)
	if err != nil {
		return nil, errors.Wrap(err, "SGuestJointsManager.ListItemFilter")
	}
	return q, nil
}

func (manager *SGuestIsolatedDeviceManager) ListItemExportKeys(ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	keys stringutils2.SSortedStrings,
) (*sqlchemy.SQuery, error) {
	var err error

	q, err = manager.SGuestJointsManager.ListItemExportKeys(ctx, q, userCred, keys)
	if err != nil {
		return nil, errors.Wrap(err, "SGuestJointsManager.ListItemExportKeys")
	}
	if keys.ContainsAny(manager.SIsolatedDeviceResourceBaseManager.GetExportKeys()...) {
		q, err = manager.SIsolatedDeviceResourceBaseManager.ListItemExportKeys(ctx, q, userCred, keys)
		if err != nil {
			return nil, errors.Wrap(err, "SIsolatedDeviceManager.ListItemExportKeys")
		}
	}

	return q, nil
}

func (manager *SGuestIsolatedDeviceManager) CreateByInsertOrUpdate() bool {
	return false
}

func (dev *SIsolatedDevice) getAllocatedMemorySize() (int, error) {
	sq := GuestIsolatedDeviceManager.Query().Equals("isolated_device_id", dev.Id).SubQuery()
	q := sq.Query(sqlchemy.SUM("memory_used", sq.Field("memory_size"))).GroupBy(sq.Field("isolated_device_id"))

	var memoryUsed int
	err := q.All(&memoryUsed)
	if err != nil {
		return -1, err
	}
	return memoryUsed, nil
}

func (dev *SIsolatedDevice) getAllocatedCount() (int, error) {
	sq := GuestIsolatedDeviceManager.Query().Equals("isolated_device_id", dev.Id).SubQuery()
	q := sq.Query(sqlchemy.COUNT("guest_count", sq.Field("guest_id"))).GroupBy(sq.Field("isolated_device_id"))

	var guestCount int
	err := q.All(&guestCount)
	if err != nil {
		return -1, err
	}
	return guestCount, nil
}

func (dev *SIsolatedDevice) getAttachedGuestIds() []string {
	q := GuestIsolatedDeviceManager.Query("guest_id").Equals("isolated_device_id", dev.Id)
	guestIsolatedDevices := make([]string, 0)
	err := q.All(&guestIsolatedDevices)
	if err != nil {
		log.Errorf("failed get guest isolated devices %s", err)
		return nil
	}
	return guestIsolatedDevices
}

func (dev *SIsolatedDevice) getAttachedGuests() []SGuestIsolatedDevice {
	q := GuestIsolatedDeviceManager.Query().Equals("isolated_device_id", dev.Id)
	guestIsolatedDevices := make([]SGuestIsolatedDevice, 0)
	err := db.FetchModelObjects(GuestIsolatedDeviceManager, q, &guestIsolatedDevices)
	if err != nil {
		log.Errorf("failed get guest isolated devices %s", err)
		return nil
	}
	return guestIsolatedDevices
}

func (guest *SGuest) GetIsolatedDevicesQuery() *sqlchemy.SQuery {
	return GuestIsolatedDeviceManager.Query().Equals("guest_id", guest.Id)
}

func (guest *SGuest) GetGuestIsolatedDevices() ([]SGuestIsolatedDevice, error) {
	gdevs := make([]SGuestIsolatedDevice, 0)
	q := guest.GetIsolatedDevicesQuery().Asc("index")
	err := db.FetchModelObjects(GuestIsolatedDeviceManager, q, &gdevs)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return gdevs, nil
}

func (guest *SGuest) GetGuestGpuIsolatedDevices() ([]SGuestIsolatedDevice, error) {
	gdevs := make([]SGuestIsolatedDevice, 0)
	q := guest.GetIsolatedDevicesQuery()
	isq := IsolatedDeviceManager.Query()

	cond := sqlchemy.OR(
		sqlchemy.Startswith(isq.Field("dev_type"), "GPU"),
		sqlchemy.In(isq.Field("dev_type"), api.CONTAINER_GPU_TYPES),
	)
	isq = isq.Filter(cond)
	sidq := isq.SubQuery()
	q = q.Join(sidq, sqlchemy.Equals(q.Field("isolated_device_id"), sidq.Field("id")))
	q = q.Asc("index")
	err := db.FetchModelObjects(GuestIsolatedDeviceManager, q, &gdevs)
	if err != nil {
		return nil, errors.Wrapf(err, "db.FetchModelObjects")
	}
	return gdevs, nil
}

func (self *SGuest) getIsolatedDeviceIndex() int8 {
	guestDevs, _ := self.GetGuestIsolatedDevices()
	var max uint
	for i := 0; i < len(guestDevs); i++ {
		if uint(guestDevs[i].Index) > max {
			max = uint(guestDevs[i].Index)
		}
	}

	idxs := make([]int, max+1)
	for i := 0; i < len(guestDevs); i++ {
		idxs[guestDevs[i].Index] = 1
	}

	// find first idx not set
	for i := 0; i < len(idxs); i++ {
		if idxs[i] != 1 {
			return int8(i)
		}
	}

	return int8(max + 1)
}

func (self *SGuestIsolatedDevice) GetIsolatedDevice() *SIsolatedDevice {
	dev, err := IsolatedDeviceManager.FetchById(self.IsolatedDeviceId)
	if err != nil {
		log.Errorf("IsolatedDeviceManager.FetchById %s", err)
		return nil
	}
	return dev.(*SIsolatedDevice)
}

func (self *SIsolatedDevice) GetGuestIsolatedDevice(guestId string, index int) (*SGuestIsolatedDevice, error) {
	q := GuestIsolatedDeviceManager.Query().Equals("isolated_device_id", self.Id).Equals("guest_id", guestId).Equals("index", index)
	ret := &SGuestIsolatedDevice{}
	err := q.First(ret)
	if err != nil {
		return nil, errors.Wrapf(err, "fetch guest %s isolated device by id %s index %d", guestId, self.Id, index)
	}
	ret.SetModelManager(GuestIsolatedDeviceManager, ret)
	return ret, nil
}

func (self *SIsolatedDevice) GetAllGuestIsolatedDevices() ([]SGuestIsolatedDevice, error) {
	q := GuestIsolatedDeviceManager.Query().Equals("isolated_device_id", self.Id)
	ret := []SGuestIsolatedDevice{}
	err := db.FetchModelObjects(GuestIsolatedDeviceManager, q, &ret)
	if err != nil {
		return nil, errors.Wrapf(err, "fetch guest isolated device by id %s", self.Id)
	}
	return ret, nil
}

func (self *SGuestIsolatedDevice) getDesc() *api.IsolatedDeviceJsonDesc {
	dev := self.GetIsolatedDevice()
	devDesc := dev.getDesc()
	devDesc.DiskIndex = self.DiskIndex
	devDesc.NetworkIndex = self.NetworkIndex
	devDesc.MemoryLimit = self.DeviceMemorySize
	devDesc.SmUtilLimit = self.SmUtilLimit
	devDesc.GpuType = self.GpuType
	return devDesc
}

func (self *SGuestIsolatedDevice) GetShortDesc(ctx context.Context) *jsonutils.JSONDict {
	devDesc := self.getDesc()
	desc := jsonutils.NewDict()
	desc.Update(jsonutils.Marshal(devDesc))
	desc.Add(jsonutils.NewString(GuestIsolatedDeviceManager.Keyword()), "res_name")
	return desc
}
