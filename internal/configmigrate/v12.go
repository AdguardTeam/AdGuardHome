package configmigrate

import (
	"time"

	"github.com/AdguardTeam/golibs/timeutil"
)

// migrateTo12 performs the following changes:
//
//	# BEFORE:
//	'schema_version': 11
//	'querylog_interval': 90
//	# …
//
//	# AFTER:
//	'schema_version': 12
//	'querylog_interval': '2160h'
//	# …
func migrateTo12(diskConf yobj) (err error) {
	diskConf["schema_version"] = 12

	dns, ok, err := fieldVal[yobj](diskConf, "dns")
	if !ok {
		return err
	}

	const field = "querylog_interval"

	qlogIvl, ok, err := fieldVal[int](dns, field)
	if !ok {
		if err != nil {
			return err
		}

		// Set the initial value from home.initConfig function.
		qlogIvl = 90
	}

	dns[field] = timeutil.Duration(time.Duration(qlogIvl) * timeutil.Day)

	return nil
}
