// Package aghos contains utilities for functions requiring system calls and
// other OS-specific APIs.  OS-specific network handling should go to aghnet
// instead.
package aghos

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
)

// Default file, binary, and directory permissions.
const (
	DefaultPermDir  fs.FileMode = 0o700
	DefaultPermExe  fs.FileMode = 0o700
	DefaultPermFile fs.FileMode = 0o600
)

// Unsupported is a helper that returns a wrapped [errors.ErrUnsupported].
func Unsupported(op string) (err error) {
	return fmt.Errorf("%s: not supported on %s: %w", op, runtime.GOOS, errors.ErrUnsupported)
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

// MaxCmdOutputSize is the maximum length of performed shell command output in
// bytes.
const MaxCmdOutputSize = 64 * 1024

// RunCommand runs shell command.
func RunCommand(command string, arguments ...string) (code int, output []byte, err error) {
	cmd := exec.Command(command, arguments...)
	out, err := cmd.Output()

	out = out[:min(len(out), MaxCmdOutputSize)]

	if err != nil {
		if eerr := new(exec.ExitError); errors.As(err, &eerr) {
			return eerr.ExitCode(), eerr.Stderr, nil
		}

		return 1, nil, fmt.Errorf("command %q failed: %w: %s", command, err, out)
	}

	return cmd.ProcessState.ExitCode(), out, nil
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
	pid, instNum, err = parsePSOutput(stdout, command, except)
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
// associated with cmdName ignoring PIDs from ignore.  A valid line from r
// should look like these:
//
//	 123 ./example-cmd
//	1230 some/base/path/example-cmd
//	3210 example-cmd
func parsePSOutput(r io.Reader, cmdName string, ignore []int) (largest, instNum int, err error) {
	s := bufio.NewScanner(r)
	for s.Scan() {
		fields := strings.Fields(s.Text())
		if len(fields) != 2 || path.Base(fields[1]) != cmdName {
			continue
		}

		cur, aerr := strconv.Atoi(fields[0])
		if aerr != nil || cur < 0 || slices.Contains(ignore, cur) {
			continue
		}

		instNum++
		largest = max(largest, cur)
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

// SendShutdownSignal sends the shutdown signal to the channel.
func SendShutdownSignal(c chan<- os.Signal) {
	sendShutdownSignal(c)
}
