package configmigrate

import (
	"os"
	"path/filepath"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
)

// migrateTo2 performs the following changes:
//
//	# BEFORE:
//	'schema_version': 1
//	'coredns':
//	  # …
//
//	# AFTER:
//	'schema_version': 2
//	'dns':
//	  # …
//
// It also deletes the Corefile file, since it isn't used anymore.
func (m *Migrator) migrateTo2(diskConf yobj) (err error) {
	diskConf["schema_version"] = 2

	coreFilePath := filepath.Join(m.workingDir, "Corefile")
	log.Printf("deleting %s as we don't need it anymore", coreFilePath)
	err = os.Remove(coreFilePath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Info("warning: %s", err)

		// Go on.
	}

	return moveVal[any](diskConf, diskConf, "coredns", "dns")
}
