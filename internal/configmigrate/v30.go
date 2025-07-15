package configmigrate

// migrateTo30 performs the following changes:
//
//	# BEFORE:
//	'dns':
//	  'cache_size': 123456
//	  # …
//
//	# AFTER:
//	'dns':
//	  'cache_size': 123456
//	  'cache_enabled': true
//	  # …
//
// If cache_size is zero, then cache_enabled should be false.
func (m Migrator) migrateTo30(diskConf yobj) (err error) {
	diskConf["schema_version"] = 30

	dnsConf, ok, err := fieldVal[yobj](diskConf, "dns")
	if !ok {
		return err
	}

	cacheSize, ok, err := fieldVal[int](dnsConf, "cache_size")
	if !ok {
		return err
	}

	if cacheSize > 0 {
		dnsConf["cache_enabled"] = true
	} else {
		dnsConf["cache_enabled"] = false
	}

	return nil
}
