// Package permcheck contains code for simplifying permissions checks on files
// and directories.
//
// TODO(a.garipov):  Improve the approach on Windows.
package permcheck

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
)

// File type constants for logging.
const (
	typeDir  = "directory"
	typeFile = "file"
)

// Check checks the permissions on important files.  It logs the results at
// appropriate levels.
func Check(workDir, dataDir, statsDir, querylogDir, confFilePath string) {
	checkDir(workDir)

	checkFile(confFilePath)

	// TODO(a.garipov): Put all paths in one place and remove this duplication.
	checkDir(dataDir)
	checkDir(filepath.Join(dataDir, "filters"))
	checkFile(filepath.Join(dataDir, "sessions.db"))
	checkFile(filepath.Join(dataDir, "leases.json"))

	if dataDir != querylogDir {
		checkDir(querylogDir)
	}
	checkFile(filepath.Join(querylogDir, "querylog.json"))
	checkFile(filepath.Join(querylogDir, "querylog.json.1"))

	if dataDir != statsDir {
		checkDir(statsDir)
	}
	checkFile(filepath.Join(statsDir, "stats.db"))
}

// checkDir checks the permissions of a single directory.  The results are
// logged at the appropriate level.
func checkDir(dirPath string) {
	checkPath(dirPath, typeDir, aghos.DefaultPermDir)
}

// checkFile checks the permissions of a single file.  The results are logged at
// the appropriate level.
func checkFile(filePath string) {
	checkPath(filePath, typeFile, aghos.DefaultPermFile)
}

// checkPath checks the permissions of a single filesystem entity.  The results
// are logged at the appropriate level.
func checkPath(entPath, fileType string, want fs.FileMode) {
	s, err := os.Stat(entPath)
	if err != nil {
		logFunc := log.Error
		if errors.Is(err, os.ErrNotExist) {
			logFunc = log.Debug
		}

		logFunc("permcheck: checking %s %q: %s", fileType, entPath, err)

		return
	}

	// TODO(a.garipov): Add a more fine-grained check and result reporting.
	perm := s.Mode().Perm()
	if perm != want {
		log.Info(
			"permcheck: SECURITY WARNING: %s %q has unexpected permissions %#o; want %#o",
			fileType,
			entPath,
			perm,
			want,
		)
	}
}
