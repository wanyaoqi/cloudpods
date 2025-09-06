package fsutils

import "yunion.io/x/onecloud/pkg/hostman/diskutils/fsutils/driver"

type SFsutilDriver struct {
	execDriver driver.IFsutilExecDriver
}

func NewFsutilDriver(execDriver driver.IFsutilExecDriver) *SFsutilDriver {
	return &SFsutilDriver{execDriver}
}

func (d *SFsutilDriver) Exec(name string, args ...string) ([]byte, error) {
	return d.execDriver.Exec(name, args...)
}

func (d *SFsutilDriver) Run(name string, args ...string) error {
	return d.execDriver.Run(name, args...)
}

func (d *SFsutilDriver) ExecInputWait(name string, args []string, input []string) (int, string, string, error) {
	return d.execDriver.ExecInputWait(name, args, input)
}
