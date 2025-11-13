package configmigrate

import (
	"context"
	"os"
	"path/filepath"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
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
func (m *Migrator) migrateTo1(ctx context.Context, diskConf yobj) (err error) {
	diskConf["schema_version"] = 1

	dnsFilterPath := filepath.Join(m.workingDir, "dnsfilter.txt")
	m.logger.InfoContext(ctx, "deleting file as we do not need it anymore", "path", dnsFilterPath)
	err = os.Remove(dnsFilterPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		m.logger.InfoContext(ctx, "failed to delete", slogutil.KeyError, err)

		// Go on.
	}

	return nil
}
