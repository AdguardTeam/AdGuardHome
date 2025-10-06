package configmigrate

import (
	"context"
	"os"
	"path/filepath"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
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
func (m *Migrator) migrateTo2(ctx context.Context, diskConf yobj) (err error) {
	diskConf["schema_version"] = 2

	coreFilePath := filepath.Join(m.workingDir, "Corefile")
	m.logger.InfoContext(ctx, "deleting file as we do not need it anymore", "path", coreFilePath)
	err = os.Remove(coreFilePath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		m.logger.WarnContext(ctx, "failed to delete", slogutil.KeyError, err)

		// Go on.
	}

	return moveVal[any](diskConf, diskConf, "coredns", "dns")
}
