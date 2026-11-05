//go:build !windows

package pak

import "os/exec"

func hideWindow(*exec.Cmd) {}
