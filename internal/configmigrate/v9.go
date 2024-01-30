package configmigrate

// migrateTo9 performs the following changes:
//
//	# BEFORE:
//	'schema_version': 8
//	'dns':
//	  'autohost_tld': 'lan'
//	  # …
//	# …
//
//	# AFTER:
//	'schema_version': 9
//	'dns':
//	  'local_domain_name': 'lan'
//	  # …
//	# …
func migrateTo9(diskConf yobj) (err error) {
	diskConf["schema_version"] = 9

	dns, ok, err := fieldVal[yobj](diskConf, "dns")
	if !ok {
		return err
	}

	return moveVal[string](dns, dns, "autohost_tld", "local_domain_name")
}
