//go:build windows

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
	cmd := exec.Command("cmd", "/c", command)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}

	return &execCommand{
		cmd: cmd,
	}
}
