package driver

import (
	"fmt"
	"io"
	"io/ioutil"

	"yunion.io/x/onecloud/pkg/hostman/monitor/qga"
	"yunion.io/x/onecloud/pkg/util/procutils"

	"yunion.io/x/pkg/errors"
)

type IFsutilExecDriver interface {
	Run(name string, args ...string) error
	Exec(name string, args ...string) ([]byte, error)
	ExecInputWait(name string, args []string, input []string) (int, string, string, error)
}

type SQgaDriver struct {
	agent *qga.QemuGuestAgent
}

func NewQgaFsutilDriver(agent *qga.QemuGuestAgent) IFsutilExecDriver {
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

type SProcDriver struct {
}

func NewProcDriver() IFsutilExecDriver {
	return new(SProcDriver)
}

func (*SProcDriver) Exec(name string, args ...string) ([]byte, error) {
	return procutils.NewCommand(name, args...).Output()
}

func (*SProcDriver) Run(name string, args ...string) error {
	return procutils.NewCommand(name, args...).Run()
}

func (*SProcDriver) ExecInputWait(name string, args []string, input []string) (int, string, string, error) {
	proc := procutils.NewCommand(name, args...)
	stdin, err := proc.StdinPipe()
	if err != nil {
		return -1, "", "", err
	}
	defer stdin.Close()

	outb, err := proc.StdoutPipe()
	if err != nil {
		return -1, "", "", err
	}
	defer outb.Close()

	errb, err := proc.StderrPipe()
	if err != nil {
		return -1, "", "", err
	}
	defer errb.Close()
	if err := proc.Start(); err != nil {
		return -1, "", "", err
	}
	for _, s := range input {
		io.WriteString(stdin, fmt.Sprintf("%s\n", s))
	}
	stdoutPut, err := ioutil.ReadAll(outb)
	if err != nil {
		return -1, "", "", err
	}
	stderrOutPut, err := ioutil.ReadAll(errb)
	if err != nil {
		return -1, "", "", err
	}
	if err = proc.Wait(); err != nil {
		if status, succ := proc.GetExitStatus(err); succ {
			return status, string(stdoutPut), string(stderrOutPut), err
		} else {
			return 0, string(stdoutPut), string(stderrOutPut), err
		}
	}
	return 0, string(stdoutPut), string(stderrOutPut), nil
}
