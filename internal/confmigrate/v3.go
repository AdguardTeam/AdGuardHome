package confmigrate

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
func migrateTo3(diskConf yobj) error {
	diskConf["schema_version"] = 3

	dnsConfig, ok, err := fieldVal[yobj](diskConf, "dns")
	if err != nil {
		return err
	} else if !ok {
		return nil
	}

	bootstrapDNS, ok, err := fieldVal[any](dnsConfig, "bootstrap_dns")
	if err != nil {
		return err
	} else if !ok {
		return nil
	}

	dnsConfig["bootstrap_dns"] = yarr{bootstrapDNS}

	return nil
}
