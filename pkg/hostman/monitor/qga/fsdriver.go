package qga

import (
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/hostman/diskutils/fsutils/driver"
)

type SQgaDriver struct {
	agent *QemuGuestAgent
}

func NewQgaFsutilDriver(agent *QemuGuestAgent) driver.IFsutilExecDriver {
	ret := new(SQgaDriver)
	ret.agent = agent
	return ret
}

func (q *SQgaDriver) ExecInputWait(name string, args []string, input []string) (int, string, string, error) {
	return q.agent.CommandWithTimeout(name, args, nil, "", true, 60)
}

func (q *SQgaDriver) Exec(name string, args ...string) ([]byte, error) {
	retCode, stdout, stderr, err := q.agent.CommandWithTimeout(name, args, nil, "", true, 60)
	if err != nil {
		return nil, err
	}
	if retCode != 0 {
		return []byte(stdout + "\n" + stderr), errors.Errorf("Exit code %d", retCode)
	}
	var retStr = []byte(stdout)
	if len(stderr) > 0 {
		retStr = []byte(stdout + "\n" + stderr)
	}
	return retStr, nil
}

func (q *SQgaDriver) Run(name string, args ...string) error {
	retCode, stdout, stderr, err := q.agent.CommandWithTimeout(name, args, nil, "", true, 60)
	if err != nil {
		return err
	}
	if retCode != 0 {
		return errors.Errorf("Exit code %d\n%s\n%s", retCode, stdout, stderr)
	}
	return nil
}
