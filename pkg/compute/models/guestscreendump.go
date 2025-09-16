package models

import "yunion.io/x/onecloud/pkg/cloudcommon/db"

type SGuestscreendumpManager struct {
	db.SResourceBaseManager
}

var GuestscreendumpManager *SGuestscreendumpManager

func init() {
	db.InitManager(func() {
		GuestscreendumpManager = &SGuestscreendumpManager{
			SResourceBaseManager: db.NewResourceBaseManager(
				SGuestscreendump{},
				"guestscreendumps_tbl",
				"guestscreendump",
				"guestscreendumps",
			),
		}
		GuestscreendumpManager.SetVirtualObject(GuestscreendumpManager)
		GuestscreendumpManager.TableSpec().AddIndex(true, "guest_id")
	})

}

type SGuestscreendump struct {
	db.SResourceBase

	RowId   int64  `primary:"true" auto_increment:"true" list:"user"`
	GuestId string `width:"36" charset:"ascii" nullable:"false" list:"user" create:"required"`
	Name    string `width:"64" charset:"ascii" nullable:"true" list:"user"`

	// s3 config
	S3AccessKey  string
	S3SecretKey  string
	S3Endpoint   string
	S3BucketName string
}
