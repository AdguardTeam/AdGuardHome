package permcheck

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
)

// NeedsMigration returns true if AdGuard Home files need permission migration.
//
// TODO(a.garipov):  Consider ways to detect this better.
func NeedsMigration(confFilePath string) (ok bool) {
	s, err := os.Stat(confFilePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Likely a first run.  Don't check.
			return false
		}

		log.Error("permcheck: checking if files need migration: %s", err)

		// Unexpected error.  Try to migrate just in case.
		return true
	}

	return s.Mode().Perm() != aghos.DefaultPermFile
}

// Migrate attempts to change the permissions of AdGuard Home's files.  It logs
// the results at an appropriate level.
func Migrate(workDir, dataDir, statsDir, querylogDir, confFilePath string) {
	chmodDir(workDir)

	chmodFile(confFilePath)

	// TODO(a.garipov): Put all paths in one place and remove this duplication.
	chmodDir(dataDir)
	chmodDir(filepath.Join(dataDir, "filters"))
	chmodFile(filepath.Join(dataDir, "sessions.db"))
	chmodFile(filepath.Join(dataDir, "leases.json"))

	if dataDir != querylogDir {
		chmodDir(querylogDir)
	}
	chmodFile(filepath.Join(querylogDir, "querylog.json"))
	chmodFile(filepath.Join(querylogDir, "querylog.json.1"))

	if dataDir != statsDir {
		chmodDir(statsDir)
	}
	chmodFile(filepath.Join(statsDir, "stats.db"))
}

// chmodDir changes the permissions of a single directory.  The results are
// logged at the appropriate level.
func chmodDir(dirPath string) {
	chmodPath(dirPath, typeDir, aghos.DefaultPermDir)
}

// chmodFile changes the permissions of a single file.  The results are logged
// at the appropriate level.
func chmodFile(filePath string) {
	chmodPath(filePath, typeFile, aghos.DefaultPermFile)
}

// chmodPath changes the permissions of a single filesystem entity.  The results
// are logged at the appropriate level.
func chmodPath(entPath, fileType string, fm fs.FileMode) {
	err := os.Chmod(entPath, fm)
	if err == nil {
		log.Info("permcheck: changed permissions for %s %q", fileType, entPath)

		return
	} else if errors.Is(err, os.ErrNotExist) {
		log.Debug("permcheck: changing permissions for %s %q: %s", fileType, entPath, err)

		return
	}

	log.Error(
		"permcheck: SECURITY WARNING: cannot change permissions for %s %q to %#o: %s; "+
			"this can leave your system vulnerable, see "+
			"https://adguard-dns.io/kb/adguard-home/running-securely/#os-service-concerns",
		fileType,
		entPath,
		fm,
		err,
	)
}
