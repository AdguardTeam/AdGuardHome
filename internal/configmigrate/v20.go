package configmigrate

import (
	"time"

	"github.com/AdguardTeam/golibs/timeutil"
)

// migrateTo20 performs the following changes:
//
//	# BEFORE:
//	'schema_version': 19
//	'statistics':
//	  'interval': 1
//	  # …
//	# …
//
//	# AFTER:
//	'schema_version': 20
//	'statistics':
//	  'interval': 24h
//	  # …
//	# …
func migrateTo20(diskConf yobj) (err error) {
	diskConf["schema_version"] = 20

	stats, ok, err := fieldVal[yobj](diskConf, "statistics")
	if !ok {
		return err
	}

	const field = "interval"

	ivl, ok, err := fieldVal[int](stats, field)
	if err != nil {
		return err
	} else if !ok || ivl == 0 {
		ivl = 1
	}

	stats[field] = timeutil.Duration(time.Duration(ivl) * timeutil.Day)

	return nil
}
