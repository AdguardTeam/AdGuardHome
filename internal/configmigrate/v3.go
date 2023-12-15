package configmigrate

// migrateTo3 performs the following changes:
//
//	# BEFORE:
//	'schema_version': 2
//	'dns':
//	  'bootstrap_dns': '1.1.1.1'
//	  # …
//
//	# AFTER:
//	'schema_version': 3
//	'dns':
//	  'bootstrap_dns':
//	  - '1.1.1.1'
//	  # …
func migrateTo3(diskConf yobj) (err error) {
	diskConf["schema_version"] = 3

	dnsConfig, ok, err := fieldVal[yobj](diskConf, "dns")
	if !ok {
		return err
	}

	bootstrapDNS, ok, err := fieldVal[any](dnsConfig, "bootstrap_dns")
	if ok {
		dnsConfig["bootstrap_dns"] = yarr{bootstrapDNS}
	}

	return err
}
