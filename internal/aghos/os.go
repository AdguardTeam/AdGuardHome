// Package aghos contains utilities for functions requiring system calls and
// other OS-specific APIs.  OS-specific network handling should go to aghnet
// instead.
package aghos

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"strings"

	"github.com/AdguardTeam/golibs/log"
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

// PIDByCommand searches for process named command and returns its PID ignoring
// the PIDs from except.  If no processes found, the error returned.
func PIDByCommand(command string, except ...int) (pid int, err error) {
	// Don't use -C flag here since it's a feature of linux's ps
	// implementation.  Use POSIX-compatible flags instead.
	//
	// See https://github.com/AdguardTeam/AdGuardHome/issues/3457.
	cmd := exec.Command("ps", "-A", "-o", "pid=", "-o", "comm=")

	var stdout io.ReadCloser
	if stdout, err = cmd.StdoutPipe(); err != nil {
		return 0, fmt.Errorf("getting the command's stdout pipe: %w", err)
	}

	if err = cmd.Start(); err != nil {
		return 0, fmt.Errorf("start command executing: %w", err)
	}

	var instNum int
	pid, instNum, err = parsePSOutput(stdout, command, except...)
	if err != nil {
		return 0, err
	}

	if err = cmd.Wait(); err != nil {
		return 0, fmt.Errorf("executing the command: %w", err)
	}

	switch instNum {
	case 0:
		// TODO(e.burkov):  Use constant error.
		return 0, fmt.Errorf("no %s instances found", command)
	case 1:
		// Go on.
	default:
		log.Info("warning: %d %s instances found", instNum, command)
	}

	if code := cmd.ProcessState.ExitCode(); code != 0 {
		return 0, fmt.Errorf("ps finished with code %d", code)
	}

	return pid, nil
}

// parsePSOutput scans the output of ps searching the largest PID of the process
// associated with cmdName ignoring PIDs from ignore.  Valid r's line shoud be
// like:
//
//    123 ./example-cmd
//   1230 some/base/path/example-cmd
//   3210 example-cmd
//
func parsePSOutput(r io.Reader, cmdName string, ignore ...int) (largest, instNum int, err error) {
	s := bufio.NewScanner(r)
ScanLoop:
	for s.Scan() {
		fields := strings.Fields(s.Text())
		if len(fields) != 2 || path.Base(fields[1]) != cmdName {
			continue
		}

		cur, aerr := strconv.Atoi(fields[0])
		if aerr != nil || cur < 0 {
			continue
		}

		for _, pid := range ignore {
			if cur == pid {
				continue ScanLoop
			}
		}

		instNum++
		if cur > largest {
			largest = cur
		}
	}
	if err = s.Err(); err != nil {
		return 0, 0, fmt.Errorf("scanning stdout: %w", err)
	}

	return largest, instNum, nil
}

// IsOpenWrt returns true if host OS is OpenWrt.
func IsOpenWrt() (ok bool) {
	return isOpenWrt()
}
