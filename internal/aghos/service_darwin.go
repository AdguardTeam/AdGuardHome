//go:build darwin

package aghos

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/AdguardTeam/golibs/log"
)

// preCheckActionStart performs the service start action pre-check.  It warns
// user that the service should be installed into Applications directory.
func preCheckActionStart() (err error) {
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("getting executable path: %v", err)
	}

	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return fmt.Errorf("evaluating executable symlinks: %v", err)
	}

	if !strings.HasPrefix(exe, "/Applications/") {
		log.Info("warning: service must be started from within the /Applications directory")
	}

	return err
}
