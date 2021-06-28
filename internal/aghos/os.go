// Package aghos contains utilities for functions requiring system calls and
// other OS-specific APIs.  OS-specific network handling should go to aghnet
// instead.
package aghos

import (
	"fmt"
	"os/exec"
	"runtime"
	"syscall"
)

// UnsupportedError is returned by functions and methods when a particular
// operation Op cannot be performed on the current OS.
type UnsupportedError struct {
	Op string
	OS string
}

// Error implements the error interface for *UnsupportedError.
func (err *UnsupportedError) Error() (msg string) {
	return fmt.Sprintf("%s is unsupported on %s", err.Op, err.OS)
}

// Unsupported is a helper that returns an *UnsupportedError with the Op field
// set to op and the OS field set to the current OS.
func Unsupported(op string) (err error) {
	return &UnsupportedError{
		Op: op,
		OS: runtime.GOOS,
	}
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
