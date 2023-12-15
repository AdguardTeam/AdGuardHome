package configmigrate

// migrateTo11 performs the following changes:
//
//	# BEFORE:
//	'schema_version': 10
//	'rlimit_nofile': 42
//	# …
//
//	# AFTER:
//	'schema_version': 11
//	'os':
//	  'group': ''
//	  'rlimit_nofile': 42
//	  'user': ''
//	# …
func migrateTo11(diskConf yobj) (err error) {
	diskConf["schema_version"] = 11

	rlimit, _, err := fieldVal[int](diskConf, "rlimit_nofile")
	if err != nil {
		return err
	}

	delete(diskConf, "rlimit_nofile")
	diskConf["os"] = yobj{
		"group":         "",
		"rlimit_nofile": rlimit,
		"user":          "",
	}

	return nil
}
