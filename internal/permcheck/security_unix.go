//go:build unix

package permcheck

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/AdguardTeam/golibs/container"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
)

// entity is a filesystem entity with a path and a flag indicating whether it is
// a directory.
type entity = container.KeyValue[string, bool]

// entities returns a list of filesystem entities that need to be ranged over.
//
// TODO(a.garipov): Put all paths in one place and remove this duplication.
func entities(workDir, dataDir, statsDir, querylogDir, confFilePath string) (ents []entity) {
	ents = []entity{{
		Key:   workDir,
		Value: true,
	}, {
		Key:   confFilePath,
		Value: false,
	}, {
		Key:   dataDir,
		Value: true,
	}, {
		Key:   filepath.Join(dataDir, "filters"),
		Value: true,
	}, {
		Key:   filepath.Join(dataDir, "sessions.db"),
		Value: false,
	}, {
		Key:   filepath.Join(dataDir, "leases.json"),
		Value: false,
	}}

	if dataDir != querylogDir {
		ents = append(ents, entity{
			Key:   querylogDir,
			Value: true,
		})
	}
	ents = append(ents, entity{
		Key:   filepath.Join(querylogDir, "querylog.json"),
		Value: false,
	}, entity{
		Key:   filepath.Join(querylogDir, "querylog.json.1"),
		Value: false,
	})

	if dataDir != statsDir {
		ents = append(ents, entity{
			Key:   statsDir,
			Value: true,
		})
	}
	ents = append(ents, entity{
		Key: filepath.Join(statsDir, "stats.db"),
	})

	return ents
}

// checkPath checks the permissions of a single filesystem entity.  The results
// are logged at the appropriate level.
func checkPath(ctx context.Context, l *slog.Logger, entPath string, want fs.FileMode) {
	l = l.With("path", entPath)

	s, err := os.Stat(entPath)
	if err != nil {
		lvl := slog.LevelError
		if errors.Is(err, os.ErrNotExist) {
			lvl = slog.LevelDebug
		}

		l.Log(ctx, lvl, "checking permissions", slogutil.KeyError, err)

		return
	}

	// TODO(a.garipov): Add a more fine-grained check and result reporting.
	perm := s.Mode().Perm()
	if perm == want {
		return
	}

	permOct, wantOct := fmt.Sprintf("%#o", perm), fmt.Sprintf("%#o", want)
	l.WarnContext(ctx, "found unexpected permissions", "perm", permOct, "want", wantOct)
}

// chmodPath changes the permissions of a single filesystem entity.  The results
// are logged at the appropriate level.
func chmodPath(ctx context.Context, l *slog.Logger, entPath string, fm fs.FileMode) {
	var lvl slog.Level
	var msg string
	args := []any{"path", entPath}

	switch err := os.Chmod(entPath, fm); {
	case err == nil:
		lvl = slog.LevelInfo
		msg = "changed permissions"
	case errors.Is(err, os.ErrNotExist):
		lvl = slog.LevelDebug
		msg = "checking permissions"
		args = append(args, slogutil.KeyError, err)
	default:
		lvl = slog.LevelError
		msg = "cannot change permissions; this can leave your system vulnerable, see " +
			"https://adguard-dns.io/kb/adguard-home/running-securely/#os-service-concerns"
		args = append(args, "target_perm", fmt.Sprintf("%#o", fm), slogutil.KeyError, err)
	}

	l.Log(ctx, lvl, msg, args...)
}
