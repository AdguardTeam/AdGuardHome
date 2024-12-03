//go:build unix

package permcheck

import (
	"context"
	"log/slog"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
)

// check is the Unix-specific implementation of [Check].
func check(
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
			checkDir(ctx, dirLoggger, ent.Key)
		} else {
			checkFile(ctx, fileLogger, ent.Key)
		}
	}
}

// checkDir checks the permissions of a single directory.  The results are
// logged at the appropriate level.
func checkDir(ctx context.Context, l *slog.Logger, dirPath string) {
	checkPath(ctx, l, dirPath, aghos.DefaultPermDir)
}

// checkFile checks the permissions of a single file.  The results are logged at
// the appropriate level.
func checkFile(ctx context.Context, l *slog.Logger, filePath string) {
	checkPath(ctx, l, filePath, aghos.DefaultPermFile)
}
