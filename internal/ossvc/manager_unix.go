//go:build unix

package ossvc

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/golibs/errors"
)

// reload is a UNIX platform implementation for the Reload method of
// [ReloadManager] interface for *manager.
func (m *manager) reload(ctx context.Context, name ServiceName) (err error) {
	nameStr := string(name)
	pidFile := filepath.Join("/var", "run", nameStr+".pid")

	pid, err := getPIDFromFile(ctx, m.logger, pidFile, nameStr)
	if err != nil {
		return fmt.Errorf("getting pid from file: %w", err)
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("finding process with pid %d: %w", pid, err)
	}

	err = proc.Signal(syscall.SIGHUP)

	return errors.Annotate(err, "sending sighup to process with pid %d: %w", pid)
}

// getPIDFromFile reads the PID from the specified pidFile. It returns an error
// if the file does not exist or if the PID cannot be parsed.  logger must not
// be nil.
func getPIDFromFile(
	ctx context.Context,
	logger *slog.Logger,
	pidFile string,
	nameStr string,
) (pid int, err error) {
	// #nosec CWE-22 -- The name of the variable is always predictable, it is a
	// constant.
	data, err := os.ReadFile(pidFile)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return 0, fmt.Errorf("reading service pid file: %w", err)
		}

		pid, err = aghos.PIDByCommand(ctx, logger, nameStr, os.Getpid())
		if err != nil {
			return 0, fmt.Errorf("finding process: %w", err)
		}
	} else {
		parts := bytes.SplitN(data, []byte("\n"), 2)
		if len(parts) == 0 {
			return 0, fmt.Errorf("parsing %q: %w", pidFile, errors.ErrEmptyValue)
		}

		pidStr := string(bytes.TrimSpace(parts[0]))
		pid, err = strconv.Atoi(pidStr)
		if err != nil {
			return 0, fmt.Errorf("parsing pid from %q: %w", pidFile, err)
		}
	}

	return pid, nil
}
