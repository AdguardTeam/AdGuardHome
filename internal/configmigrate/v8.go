package configmigrate

// migrateTo8 performs the following changes:
//
//	# BEFORE:
//	'schema_version': 7
//	'dns':
//	  'bind_host': '127.0.0.1'
//	  # …
//	# …
//
//	# AFTER:
//	'schema_version': 8
//	'dns':
//	  'bind_hosts':
//	  - '127.0.0.1'
//	  # …
//	# …
func migrateTo8(diskConf yobj) (err error) {
	diskConf["schema_version"] = 8

	dns, ok, err := fieldVal[yobj](diskConf, "dns")
	if !ok {
		return err
	}

	bindHost, ok, err := fieldVal[string](dns, "bind_host")
	if !ok {
		return err
	}

	delete(dns, "bind_host")
	dns["bind_hosts"] = yarr{bindHost}

	return nil
}
