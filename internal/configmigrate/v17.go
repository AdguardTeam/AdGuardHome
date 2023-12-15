package configmigrate

// migrateTo17 performs the following changes:
//
//	# BEFORE:
//	'schema_version': 16
//	'dns':
//	  'edns_client_subnet': false
//	  # …
//	# …
//
//	# AFTER:
//	'schema_version': 17
//	'dns':
//	  'edns_client_subnet':
//	    'enabled': false
//	    'use_custom': false
//	    'custom_ip': ""
//	  # …
//	# …
func migrateTo17(diskConf yobj) (err error) {
	diskConf["schema_version"] = 17

	dns, ok, err := fieldVal[yobj](diskConf, "dns")
	if !ok {
		return err
	}

	const field = "edns_client_subnet"

	enabled, _, _ := fieldVal[bool](dns, field)
	dns[field] = yobj{
		"enabled":    enabled,
		"use_custom": false,
		"custom_ip":  "",
	}

	return nil
}
