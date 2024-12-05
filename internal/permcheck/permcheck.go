// Package permcheck contains code for simplifying permissions checks on files
// and directories.
package permcheck

import (
	"context"
	"log/slog"
)

// File type constants for logging.
const (
	typeDir  = "directory"
	typeFile = "file"
)

// Check checks the permissions on important files.  It logs the results at
// appropriate levels.
func Check(
	ctx context.Context,
	l *slog.Logger,
	workDir string,
	dataDir string,
	statsDir string,
	querylogDir string,
	confFilePath string,
) {
	check(ctx, l, workDir, dataDir, statsDir, querylogDir, confFilePath)
}

// NeedsMigration returns true if AdGuard Home files need permission migration.
func NeedsMigration(ctx context.Context, l *slog.Logger, workDir, confFilePath string) (ok bool) {
	return needsMigration(ctx, l, workDir, confFilePath)
}

// Migrate attempts to change the permissions of AdGuard Home's files.  It logs
// the results at an appropriate level.
func Migrate(
	ctx context.Context,
	l *slog.Logger,
	workDir string,
	dataDir string,
	statsDir string,
	querylogDir string,
	confFilePath string,
) {
	migrate(ctx, l, workDir, dataDir, statsDir, querylogDir, confFilePath)
}
