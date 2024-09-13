package cephutils

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/util/procutils"
)

type SContainerDriver struct {
	timeout int

	containerCmd string
	containerId  string
}

func NewContainerDriver(timeout int, containerCmd string) (*SContainerDriver, error) {
	cmd := fmt.Sprint("%s ps | grep yunion-ceph-common", containerCmd)
	out, err := procutils.NewRemoteCommandAsFarAsPossible("bash", "-c", cmd).Output()
	if err != nil {
		return nil, errors.Wrapf(err, "failed find ceph common container: %s", out)
	}

	lines := strings.Split(string(out), "\n")
	containerId := ""
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			containerId = strings.Split(line, "")[0]
			break
		}
	}
	if containerId == "" {
		return nil, errors.Errorf("failed find ceph common container")
	}
	log.Infof("found ceph common container id %s", containerId)

	return &SContainerDriver{
		timeout:      timeout,
		containerCmd: containerCmd,
		containerId:  containerId,
	}, nil
}

func (cli *SContainerDriver) FilePutConfig(pattern string, content string) (string, error) {
	var prefix, suffix string
	if pos := strings.LastIndex(pattern, "*"); pos != -1 {
		prefix, suffix = pattern[:pos], pattern[pos+1:]
	} else {
		prefix = pattern
	}
	fname := filepath.Join("/tmp", prefix+strconv.Itoa(int(time.Now().Unix()))+suffix)
	cmd := fmt.Sprint("echo '%s' > %s", content, fname)
	out, err := procutils.NewRemoteCommandAsFarAsPossible(cli.containerCmd, "exec", cli.containerId, "bash", "-c", cmd).Output()
	if err != nil {
		return "", errors.Wrapf(err, "exec failed %s", out)
	}
	return fname, nil
}

func (cli *SContainerDriver) FileGetConfig(filename string) (string, error) {
	out, err := procutils.NewRemoteCommandAsFarAsPossible(cli.containerCmd, "exec", cli.containerId, "cat", filename).Output()
	return string(out), err
}

func (cli *SContainerDriver) Output(name string, opts []string, timeout bool) (jsonutils.JSONObject, error) {
	cmds := []string{name, "--format", "json"}
	cmds = append(cmds, opts...)
	if timeout {
		cmds = append([]string{"timeout", "--signal=KILL", fmt.Sprintf("%ds", cli.timeout)}, cmds...)
	}
	cmds = append(cmds, "2>/error")
	cmd := strings.Join(cmds, " ")
	output, err := procutils.NewRemoteCommandAsFarAsPossible(cli.containerCmd, "exec", cli.containerId, "bash", "-c", cmd).Output()
	if err != nil {
		out, _ := procutils.NewRemoteCommandAsFarAsPossible(cli.containerCmd, "exec", cli.containerId, "cat", "/error").Output()
		return nil, errors.Wrapf(err, "%s %s %s", name, output, out)
	}

	return jsonutils.Parse(output)
}

func (cli *SContainerDriver) Run(name string, opts []string, timeout bool) error {
	cmds := append([]string{name}, opts...)
	if timeout {
		cmds = append([]string{"timeout", "--signal=KILL", fmt.Sprintf("%ds", cli.timeout)}, cmds...)
	}
	cmd := strings.Join(cmds, " ")
	output, err := procutils.NewRemoteCommandAsFarAsPossible(cli.containerCmd, "exec", cli.containerId, "bash", "-c", cmd).Output()
	if err != nil {
		return errors.Wrapf(err, "%s %s", name, string(output))
	}
	return nil
}
