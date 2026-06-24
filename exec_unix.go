//go:build !windows
// +build !windows

package main

import "os/exec"

func execCmd(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}
