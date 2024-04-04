package configmigrate

import "github.com/AdguardTeam/golibs/errors"

// migrateTo24 performs the following changes:
//
//	# BEFORE:
//	'schema_version': 23
//	'log_file': ""
//	'log_max_backups': 0
//	'log_max_size': 100
//	'log_max_age': 3
//	'log_compress': false
//	'log_localtime': false
//	'verbose': false
//	# …
//
//	# AFTER:
//	'schema_version': 24
//	'log':
//	  'file': ""
//	  'max_backups': 0
//	  'max_size': 100
//	  'max_age': 3
//	  'compress': false
//	  'local_time': false
//	  'verbose': false
//	# …
func migrateTo24(diskConf yobj) (err error) {
	diskConf["schema_version"] = 24

	logObj := yobj{}
	err = errors.Join(
		moveVal[string](diskConf, logObj, "log_file", "file"),
		moveVal[int](diskConf, logObj, "log_max_backups", "max_backups"),
		moveVal[int](diskConf, logObj, "log_max_size", "max_size"),
		moveVal[int](diskConf, logObj, "log_max_age", "max_age"),
		moveVal[bool](diskConf, logObj, "log_compress", "compress"),
		moveVal[bool](diskConf, logObj, "log_localtime", "local_time"),
		moveVal[bool](diskConf, logObj, "verbose", "verbose"),
	)
	if err != nil {
		// Don't wrap the error, because it's informative enough as is.
		return err
	}

	if len(logObj) != 0 {
		diskConf["log"] = logObj
	}

	return nil
}
