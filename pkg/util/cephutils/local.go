package cephutils

import (
	"fmt"
	"io/ioutil"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/fileutils2"
	"yunion.io/x/onecloud/pkg/util/procutils"
)

type SLocalDriver struct {
	timeout int
}

func NewLocalDriver(timeout int) *SLocalDriver {
	return &SLocalDriver{
		timeout: timeout,
	}
}

func (cli *SLocalDriver) FilePutConfig(pattern string, content string) (string, error) {
	file, err := ioutil.TempFile("", pattern)
	if err != nil {
		return "", errors.Wrapf(err, "TempFile")
	}
	defer file.Close()
	name := file.Name()
	_, err = file.Write([]byte(content))
	if err != nil {
		return name, errors.Wrapf(err, "write")
	}
	return name, nil
}

func (cli *SLocalDriver) FileGetConfig(filename string) (string, error) {
	return fileutils2.FileGetContents(filename)
}

func (cli *SLocalDriver) Output(name string, opts []string, timeout bool) (jsonutils.JSONObject, error) {
	cmds := []string{name, "--format", "json"}
	cmds = append(cmds, opts...)
	if timeout {
		cmds = append([]string{"timeout", "--signal=KILL", fmt.Sprintf("%ds", cli.timeout)}, cmds...)
	}
	proc := procutils.NewRemoteCommandAsFarAsPossible(cmds[0], cmds[1:]...)
	outb, err := proc.StdoutPipe()
	if err != nil {
		return nil, errors.Wrap(err, "stdout pipe")
	}
	defer outb.Close()

	errb, err := proc.StderrPipe()
	if err != nil {
		return nil, errors.Wrap(err, "stderr pipe")
	}
	defer errb.Close()

	if err := proc.Start(); err != nil {
		return nil, errors.Wrap(err, "start ceph process")
	}

	stdoutPut, err := ioutil.ReadAll(outb)
	if err != nil {
		return nil, err
	}
	stderrPut, err := ioutil.ReadAll(errb)
	if err != nil {
		return nil, err
	}

	if err := proc.Wait(); err != nil {
		return nil, errors.Wrapf(err, "stderr %q", stderrPut)
	}
	return jsonutils.Parse(stdoutPut)
}

func (cli *SLocalDriver) Run(name string, opts []string, timeout bool) error {
	cmds := append([]string{name}, opts...)
	if timeout {
		cmds = append([]string{"timeout", "--signal=KILL", fmt.Sprintf("%ds", cli.timeout)}, cmds...)
	}
	output, err := procutils.NewRemoteCommandAsFarAsPossible(cmds[0], cmds[1:]...).Output()
	if err != nil {
		return errors.Wrapf(err, "%s %s", name, string(output))
	}
	return nil
}
