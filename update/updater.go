package update

import (
	"os"
	"path/filepath"
)

type Updater struct {
	DisableUpdate bool

	currentBinary string // current binary executable
	workDir       string // updater work dir (where backup/upd dirs will be created)
}

// NewUpdater - creates a new instance of the Updater
func NewUpdater(workDir string) *Updater {
	return &Updater{
		currentBinary: filepath.Base(os.Args[0]),
		workDir:       workDir,
	}
}

// DoUpdate - conducts the auto-update
// 1. Downloads the update file
// 2. Unpacks it and checks the contents
// 3. Backups the current version and configuration
// 4. Replaces the old files
// 5. Restarts the service
func (u *Updater) DoUpdate() error {
	return nil
}
