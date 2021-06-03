// Package aghos contains utilities for functions requiring system calls.
package aghos

import (
	"fmt"
	"os/exec"
	"syscall"

	"github.com/AdguardTeam/golibs/errors"
)

// ErrUnsupported is returned when the functionality is unsupported on the
// current operating system.
//
// TODO(a.garipov): Make a structured error and use it everywhere instead of
// a bunch of fmt.Errorf and all that.
const ErrUnsupported errors.Error = "unsupported"

// CanBindPrivilegedPorts checks if current process can bind to privileged
// ports.
func CanBindPrivilegedPorts() (can bool, err error) {
	return canBindPrivilegedPorts()
}

// SetRlimit sets user-specified limit of how many fd's we can use.
//
// See https://github.com/AdguardTeam/AdGuardHome/internal/issues/659.
func SetRlimit(val uint64) (err error) {
	return setRlimit(val)
}

// HaveAdminRights checks if the current user has root (administrator) rights.
func HaveAdminRights() (bool, error) {
	return haveAdminRights()
}

// SendProcessSignal sends signal to a process.
func SendProcessSignal(pid int, sig syscall.Signal) error {
	return sendProcessSignal(pid, sig)
}

// MaxCmdOutputSize is the maximum length of performed shell command output.
const MaxCmdOutputSize = 2 * 1024

// RunCommand runs shell command.
func RunCommand(command string, arguments ...string) (int, string, error) {
	cmd := exec.Command(command, arguments...)
	out, err := cmd.Output()
	if len(out) > MaxCmdOutputSize {
		out = out[:MaxCmdOutputSize]
	}
	if err != nil {
		return 1, "", fmt.Errorf("exec.Command(%s) failed: %v: %s", command, err, string(out))
	}

	return cmd.ProcessState.ExitCode(), string(out), nil
}

// IsOpenWrt returns true if host OS is OpenWrt.
func IsOpenWrt() (ok bool) {
	return isOpenWrt()
}
