//go:build linux

package command

import (
	"os/exec"
	"syscall"
)

type Commander interface {
	Command(command string) Command
}

type execCommander struct{}

func (rc *execCommander) Command(command string) Command {
	cmd := exec.Command("/bin/sh", "-c", command)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	return &execCommand{
		cmd: cmd,
	}
}
