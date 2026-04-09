//go:build unix

package ossvc

import (
	"bytes"
	"context"
	"fmt"
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

	var pid int
	pidFile := filepath.Join("/var", "run", nameStr+".pid")
	// #nosec CWE-22 -- The name of the variable is always predictable, it is a
	// constant.
	data, err := os.ReadFile(pidFile)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("reading service pid file: %w", err)
		}

		pid, err = aghos.PIDByCommand(ctx, m.logger, nameStr, os.Getpid())
		if err != nil {
			return fmt.Errorf("finding process: %w", err)
		}
	} else {
		parts := bytes.SplitN(data, []byte("\n"), 2)
		if len(parts) == 0 {
			return fmt.Errorf("parsing %q: %w", pidFile, errors.ErrEmptyValue)
		}

		pidStr := string(bytes.TrimSpace(parts[0]))
		pid, err = strconv.Atoi(pidStr)
		if err != nil {
			return fmt.Errorf("parsing pid from %q: %w", pidFile, err)
		}
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("finding process with pid %d: %w", pid, err)
	}

	err = proc.Signal(syscall.SIGHUP)
	if err != nil {
		return fmt.Errorf("sending sighup to process with pid %d: %w", pid, err)
	}

	return nil
}
