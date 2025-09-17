package models

import (
	"context"
	"encoding/base64"
	"io"

	"yunion.io/x/cloudmux/pkg/multicloud/objectstore"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/cloudcommon/db/lockman"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/util/logclient"
)

type SGuestScreenDumpManager struct {
	db.SResourceBaseManager
}

var GuestScreenDumpManager *SGuestScreenDumpManager

func init() {
	db.InitManager(func() {
		GuestScreenDumpManager = &SGuestScreenDumpManager{
			SResourceBaseManager: db.NewResourceBaseManager(
				SGuestScreenDump{},
				"guest_screen_dumps_tbl",
				"guest_screen_dump",
				"guest_screen_dumps",
			),
		}
		GuestScreenDumpManager.SetVirtualObject(GuestScreenDumpManager)
		GuestScreenDumpManager.TableSpec().AddIndex(true, "guest_id")
	})
}

type SGuestScreenDump struct {
	db.SResourceBase

	RowId   int64  `primary:"true" auto_increment:"true" list:"user"`
	GuestId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"`
	Name    string `width:"64" charset:"ascii" nullable:"true" list:"user"`

	// s3 config
	S3AccessKey  string `width:"64" charset:"ascii" nullable:"true"`
	S3SecretKey  string `width:"64" charset:"ascii" nullable:"true"`
	S3Endpoint   string `width:"64" charset:"ascii" nullable:"true" list:"user"`
	S3BucketName string `width:"64" charset:"ascii" nullable:"true" list:"user"`
}

func (manager *SGuestScreenDumpManager) ListItemFilter(
	ctx context.Context,
	q *sqlchemy.SQuery,
	userCred mcclient.TokenCredential,
	query api.GuestScreenDumpListInput,
) (*sqlchemy.SQuery, error) {
	if query.Server != "" {
		iGuest, err := GuestManager.FetchByIdOrName(ctx, userCred, query.Server)
		if err != nil {
			return q, errors.Wrap(err, "fetch guest")
		}
		q = q.Equals("guest_id", iGuest.GetId())
	}
	return q, nil
}

func (self *SGuestScreenDump) Delete(ctx context.Context, userCred mcclient.TokenCredential) error {
	return db.DeleteModel(ctx, userCred, self)
}

func (self *SGuest) SaveGuestScreenDump(ctx context.Context, userCred mcclient.TokenCredential, screenDumpInfo *api.SGuestScreenDump) (*SGuestScreenDump, error) {
	sd := new(SGuestScreenDump)
	sd.SetModelManager(GuestScreenDumpManager, sd)
	sd.GuestId = self.GetId()
	sd.S3SecretKey = screenDumpInfo.S3SecretKey
	sd.S3Endpoint = screenDumpInfo.S3Endpoint
	sd.S3BucketName = screenDumpInfo.S3BucketName
	sd.S3AccessKey = screenDumpInfo.S3AccessKey
	sd.Name = screenDumpInfo.S3ObjectName

	lockman.LockClass(ctx, GuestScreenDumpManager, self.ProjectId)
	defer lockman.ReleaseClass(ctx, GuestScreenDumpManager, self.ProjectId)

	err := GuestScreenDumpManager.TableSpec().Insert(ctx, sd)
	if err != nil {
		return nil, errors.Wrap(err, "save guest screen_dump")
	}
	db.OpsLog.LogEvent(self, db.ACT_GUEST_SCREEN_DUMP, sd.Name, userCred)
	logclient.AddSimpleActionLog(self, logclient.ACT_GUEST_SCREEN_DUMP, sd.Name, userCred, true)
	return sd, nil
}

func (self *SGuest) GetDetailsScreenDump(ctx context.Context, userCred mcclient.TokenCredential, input *api.GetDetailsGuestScreenDumpInput) (*api.GetDetailsGuestScreenDumpOutput, error) {
	if input.Name == "" {
		return nil, httperrors.NewMissingParameterError("name")
	}
	q := GuestScreenDumpManager.Query()
	q = q.Equals("guest_id", self.Id)
	q = q.Equals("name", input.Name)

	screenDump := new(SGuestScreenDump)
	err := q.First(screenDump)
	if err != nil {
		return nil, errors.Wrap(err, "query screenDump")
	}

	cfg := objectstore.NewObjectStoreClientConfig(screenDump.S3Endpoint, screenDump.S3AccessKey, screenDump.S3SecretKey)
	s3Client, err := objectstore.NewObjectStoreClient(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "new minio client")
	}
	bucket, err := s3Client.GetIBucketByName(screenDump.S3BucketName)
	if err != nil {
		return nil, errors.Wrapf(err, "get bucket %s", screenDump.S3BucketName)
	}
	irc, err := bucket.GetObject(ctx, screenDump.Name, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "get object %s", screenDump.Name)
	}
	defer irc.Close()
	obj, err := io.ReadAll(irc)
	if err != nil {
		return nil, errors.Wrapf(err, "read object %s", screenDump.Name)
	}
	ret := new(api.GetDetailsGuestScreenDumpOutput)
	ret.ScreenDump = base64.StdEncoding.EncodeToString(obj)
	ret.GuestId = self.Id
	ret.Name = screenDump.Name
	return ret, nil
}

func (self *SGuest) PerformScreenDump(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject,
	data jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	if self.PowerStates != api.VM_POWER_STATES_ON {
		return nil, httperrors.NewBadRequestError("can't use qga in vm status: %s", self.Status)
	}

	host, _ := self.GetHost()
	res, err := self.GetDriver().RequestGuestScreenDump(ctx, userCred, nil, host, self)
	if err != nil {
		return nil, err
	}
	screenDumpInfo := api.SGuestScreenDump{}
	if err := res.Unmarshal(&screenDumpInfo); err != nil {
		return nil, errors.Wrap(err, "unmarshal screen dump info")
	}
	if _, err := self.SaveGuestScreenDump(ctx, userCred, &screenDumpInfo); err != nil {
		return nil, errors.Wrap(err, "failed save ")
	}
	input := &api.GetDetailsGuestScreenDumpInput{
		Name: screenDumpInfo.S3ObjectName,
	}
	ret, err := self.GetDetailsScreenDump(ctx, userCred, input)
	if err != nil {
		return nil, err
	}
	return jsonutils.Marshal(ret), err
}
