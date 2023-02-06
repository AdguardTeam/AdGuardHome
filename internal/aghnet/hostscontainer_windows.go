//go:build windows

package aghnet

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/AdguardTeam/golibs/log"
	"golang.org/x/sys/windows"
)

func defaultHostsPaths() (paths []string) {
	sysDir, err := windows.GetSystemDirectory()
	if err != nil {
		log.Error("aghnet: getting system directory: %s", err)

		return []string{}
	}

	// Split all the elements of the path to join them afterwards.  This is
	// needed to make the Windows-specific path string returned by
	// windows.GetSystemDirectory to be compatible with fs.FS.
	pathElems := strings.Split(sysDir, string(os.PathSeparator))
	if len(pathElems) > 0 && pathElems[0] == filepath.VolumeName(sysDir) {
		pathElems = pathElems[1:]
	}

	return []string{path.Join(append(pathElems, "drivers/etc/hosts")...)}
}
