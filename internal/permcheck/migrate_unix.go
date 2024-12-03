//go:build unix

package permcheck

import (
	"context"
	"log/slog"
	"os"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
)

// needsMigration is a Unix-specific implementation of [NeedsMigration].
//
// TODO(a.garipov):  Consider ways to detect this better.
func needsMigration(ctx context.Context, l *slog.Logger, _, confFilePath string) (ok bool) {
	s, err := os.Stat(confFilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Likely a first run.  Don't check.
			return false
		}

		l.ErrorContext(ctx, "checking a need for permission migration", slogutil.KeyError, err)

		// Unexpected error.  Try to migrate just in case.
		return true
	}

	return s.Mode().Perm() != aghos.DefaultPermFile
}

// migrate is a Unix-specific implementation of [Migrate].
func migrate(
	ctx context.Context,
	l *slog.Logger,
	workDir string,
	dataDir string,
	statsDir string,
	querylogDir string,
	confFilePath string,
) {
	dirLoggger, fileLogger := l.With("type", typeDir), l.With("type", typeFile)

	for _, ent := range entities(workDir, dataDir, statsDir, querylogDir, confFilePath) {
		if ent.Value {
			chmodDir(ctx, dirLoggger, ent.Key)
		} else {
			chmodFile(ctx, fileLogger, ent.Key)
		}
	}
}

// chmodDir changes the permissions of a single directory.  The results are
// logged at the appropriate level.
func chmodDir(ctx context.Context, l *slog.Logger, dirPath string) {
	chmodPath(ctx, l, dirPath, aghos.DefaultPermDir)
}

// chmodFile changes the permissions of a single file.  The results are logged
// at the appropriate level.
func chmodFile(ctx context.Context, l *slog.Logger, filePath string) {
	chmodPath(ctx, l, filePath, aghos.DefaultPermFile)
}
