package configmigrate

// migrateTo14 performs the following changes:
//
//	# BEFORE:
//	'schema_version': 13
//	'dns':
//	  'resolve_clients': true
//	  # …
//	'clients':
//	- 'name': 'client-name'
//	  # …
//	# …
//
//	# AFTER:
//	'schema_version': 14
//	'dns':
//	  # …
//	'clients':
//	  'persistent':
//	  - 'name': 'client-name'
//	    # …
//	  'runtime_sources':
//	    'whois': true
//	    'arp': true
//	    'rdns': true
//	    'dhcp': true
//	    'hosts': true
//	# …
func migrateTo14(diskConf yobj) (err error) {
	diskConf["schema_version"] = 14

	persistent, ok, err := fieldVal[yarr](diskConf, "clients")
	if !ok {
		if err != nil {
			return err
		}

		persistent = yarr{}
	}

	runtimeClients := yobj{
		"whois": true,
		"arp":   true,
		"rdns":  false,
		"dhcp":  true,
		"hosts": true,
	}
	diskConf["clients"] = yobj{
		"persistent":      persistent,
		"runtime_sources": runtimeClients,
	}

	dns, ok, err := fieldVal[yobj](diskConf, "dns")
	if err != nil {
		return err
	} else if !ok {
		return nil
	}

	return moveVal[bool](dns, runtimeClients, "resolve_clients", "rdns")
}
