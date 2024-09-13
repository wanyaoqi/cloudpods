package cephutils

import "yunion.io/x/jsonutils"

type ICephCommonDriver interface {
	// write ceph config to file, generate filename from pattern, return filename
	FilePutConfig(pattern string, content string) (string, error)
	FileGetConfig(filename string) (string, error)

	Output(name string, opts []string, timeout bool) (jsonutils.JSONObject, error)
	Run(name string, opts []string, timeout bool) error
}
