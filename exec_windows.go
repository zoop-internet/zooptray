//go:build windows
// +build windows

package main

import (
	"os/exec"
	"syscall"
)

func execCmd(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd
}
