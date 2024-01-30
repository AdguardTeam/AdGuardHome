package configmigrate

// migrateTo4 performs the following changes:
//
//	# BEFORE:
//	'schema_version': 3
//	'clients':
//	- # …
//	# …
//
//	# AFTER:
//	'schema_version': 4
//	'clients':
//	- 'use_global_blocked_services': true
//	  # …
//	# …
func migrateTo4(diskConf yobj) (err error) {
	diskConf["schema_version"] = 4

	clients, ok, _ := fieldVal[yarr](diskConf, "clients")
	if ok {
		for i := range clients {
			if c, isYobj := clients[i].(yobj); isYobj {
				c["use_global_blocked_services"] = true
			}
		}
	}

	return nil
}
