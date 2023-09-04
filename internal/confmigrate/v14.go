package confmigrate

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
	if err != nil {
		return err
	} else if !ok {
		persistent = yarr{}
	}

	var rdnsSrc bool
	dns, ok, err := fieldVal[yobj](diskConf, "dns")
	if err != nil {
		return err
	} else if ok {
		rdnsSrc, ok, err = fieldVal[bool](dns, "resolve_clients")
		if err != nil {
			return err
		} else if ok {
			delete(dns, "resolve_clients")
		}
	}

	diskConf["clients"] = yobj{
		"persistent": persistent,
		"runtime_sources": yobj{
			"whois": true,
			"arp":   true,
			"rdns":  rdnsSrc,
			"dhcp":  true,
			"hosts": true,
		},
	}

	return nil
}
