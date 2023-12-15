package configmigrate

// migrateTo25 performs the following changes:
//
//	# BEFORE:
//	'schema_version': 24
//	'debug_pprof': true
//	# …
//
//	# AFTER:
//	'schema_version': 25
//	'http':
//	  'pprof':
//	    'enabled': true
//	    'port': 6060
//	# …
func migrateTo25(diskConf yobj) (err error) {
	diskConf["schema_version"] = 25

	httpObj, ok, err := fieldVal[yobj](diskConf, "http")
	if !ok {
		return err
	}

	pprofObj := yobj{
		"enabled": false,
		"port":    6060,
	}

	err = moveVal[bool](diskConf, pprofObj, "debug_pprof", "enabled")
	if err != nil {
		return err
	}

	httpObj["pprof"] = pprofObj

	return nil
}
