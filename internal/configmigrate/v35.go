package configmigrate

import "context"

// migrateTo35 performs the following changes:
//
//	# BEFORE cmdline: --no-etc-hosts
//	# BEFORE config:
//	'dns':
//	  # …
//	  'hostsfile_enabled': true
//	'clients':
//	  # …
//	  'runtime_sources':
//	    # …
//	    hosts: true
//
//	# AFTER config:
//	'dns':
//	  # …
//	  'hostsfile_enabled': false
//	'clients':
//	  # …
//	  'runtime_sources':
//	    # …
//	    hosts: false
func (m Migrator) migrateTo35(_ context.Context, diskConf yobj) (err error) {
	diskConf["schema_version"] = 35

	if !m.legacyNoEtcHosts {
		return nil
	}

	dnsConf, ok, err := fieldVal[yobj](diskConf, "dns")
	if !ok {
		// Don't wrap the error, since it's informative enough as is.
		return err
	}

	dnsConf["hostsfile_enabled"] = false

	clientsConf, ok, err := fieldVal[yobj](diskConf, "clients")
	if !ok {
		// Don't wrap the error, since it's informative enough as is.
		return err
	}

	rtSrcConf, ok, err := fieldVal[yobj](clientsConf, "runtime_sources")
	if !ok {
		// Don't wrap the error, since it's informative enough as is.
		return err
	}

	rtSrcConf["hosts"] = false

	return nil
}
