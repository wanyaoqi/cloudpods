package tasks

import (
	"context"
	"github.com/pkg/errors"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/util/ssh"
)

type SBaremetalIsolatedDevicesProbeTask struct {
	SBaremetalTaskBase
}

func (self *SBaremetalIsolatedDevicesProbeTask) GetName() string {
	return "SBaremetalIsolatedDevicesProbeTask"
}

func NewBaremetalIsolatedDevicesProbeTask(
	userCred mcclient.TokenCredential,
	baremetal IBaremetal,
	taskId string,
	data jsonutils.JSONObject,
) ITask {
	task := &SBaremetalIsolatedDevicesProbeTask{
		SBaremetalTaskBase: newBaremetalTaskBase(userCred, baremetal, taskId, data),
	}
	task.SetVirtualObject(task)

	log.Debugf("SBaremetalIsolatedDevicesProbeTask: %s", data)

	task.SetStage(task.DoProbeIsolatedDevices)
	return task
}

func (self *SBaremetalIsolatedDevicesProbeTask) DoProbeIsolatedDevices(ctx context.Context, args interface{}) error {
	accessIp, _ := self.data.GetString("access_ip")
	sshPort := 22
	if self.data.Contains("ssh_port") {
		sshPort64, _ := self.data.Int("ssh_port")
		sshPort = int(sshPort64)
	}
	username := "cloudroot"
	if self.data.Contains("username") {
		username, _ = self.data.GetString("username")
	}
	passwd, _ := self.data.GetString("password")
	privateKey, _ := self.data.GetString("private_key")
	log.Infof("sshinfo %v %v %v %v %v", accessIp, sshPort, username, passwd, privateKey)
	sshCli, err := ssh.NewClient(accessIp, sshPort, username, passwd, privateKey)
	if err != nil {
		log.Errorf("failed ssh.NewClient with %v %v %v %v %v: %s",
			accessIp, sshPort, username, passwd, privateKey, err)
		return errors.Wrapf(err,
			"ssh.NewClient with %v %v %v %v %v", accessIp, sshPort, username, passwd, privateKey)
	}
	devs, err := getIsolatedDevicesInfo(sshCli, nil)
	if err != nil {
		log.Errorf("failed getIsolatedDevicesInfo %s", err)
		return errors.Wrap(err, "getIsolatedDevicesInfo")
	}
	if len(devs) == 0 {
		return nil
	}
	bmPrepare := newBaremetalPrepareTask(self.Baremetal, self.userCred)
	err = bmPrepare.sendIsolatedDevicesInfo(bmPrepare.baremetal.GetClientSession(), devs)
	if err != nil {
		log.Errorf("send isolated devices info")
		return errors.Wrap(err, "send isolated devices info")
	}
	params := jsonutils.NewDict()
	_, err = modules.Hosts.PerformAction(self.Baremetal.GetClientSession(), self.Baremetal.GetId(), "attach_isolated_devices", params)
	if err != nil {
		log.Errorf("Attach baremetal %s isolated devices error: %v", self.Baremetal.GetId(), err)
		return errors.Wrap(err, "attach_isolated_devices")
	}
	log.Infof("Attach baremetal %s isolated devices success", self.Baremetal.GetId())
	return nil
}
