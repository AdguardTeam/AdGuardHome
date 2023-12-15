package configmigrate

// migrateTo13 performs the following changes:
//
//	# BEFORE:
//	'schema_version': 12
//	'dns':
//	  'local_domain_name': 'lan'
//	  # …
//	# …
//
//	# AFTER:
//	'schema_version': 13
//	'dhcp':
//	  'local_domain_name': 'lan'
//	  # …
//	# …
func migrateTo13(diskConf yobj) (err error) {
	diskConf["schema_version"] = 13

	dns, ok, err := fieldVal[yobj](diskConf, "dns")
	if !ok {
		return err
	}

	dhcp, ok, err := fieldVal[yobj](diskConf, "dhcp")
	if !ok {
		return err
	}

	return moveSameVal[string](dns, dhcp, "local_domain_name")
}
