package configmigrate

import (
	"os"
	"path/filepath"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
)

// migrateTo1 performs the following changes:
//
//	# BEFORE:
//	# …
//
//	# AFTER:
//	'schema_version': 1
//	# …
//
// It also deletes the unused dnsfilter.txt file, since the following versions
// store filters in data/filters/.
func (m *Migrator) migrateTo1(diskConf yobj) (err error) {
	diskConf["schema_version"] = 1

	dnsFilterPath := filepath.Join(m.workingDir, "dnsfilter.txt")
	log.Printf("deleting %s as we don't need it anymore", dnsFilterPath)
	err = os.Remove(dnsFilterPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Info("warning: %s", err)

		// Go on.
	}

	return nil
}
