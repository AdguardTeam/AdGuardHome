package configmigrate

import "context"

// migrateTo34 performs the following changes:
//
//	# BEFORE:
//	'http':
//	  # …
//	'tls':
//	  'enabled': true
//	  'allow_unencrypted_doh': false
//
//	# AFTER:
//	'http':
//	  # …
//	  'doh':
//	    'routes':
//	      - 'GET /dns-query'
//	      - 'POST /dns-query'
//	      - 'GET /dns-query/{ClientID}'
//	      - 'POST /dns-query/{ClientID}'
//	  'insecure_enabled': false
//	'tls':
//	  'enabled': true
func (m Migrator) migrateTo34(_ context.Context, diskConf yobj) (err error) {
	diskConf["schema_version"] = 34

	httpConf, ok, err := fieldVal[yobj](diskConf, "http")
	if !ok {
		return err
	}

	tlsConf, ok, err := fieldVal[yobj](diskConf, "tls")
	if !ok {
		return err
	}

	// Create doh section with default routes.
	dohConf := yobj{
		"routes": yarr{
			"GET /dns-query",
			"POST /dns-query",
			"GET /dns-query/{ClientID}",
			"POST /dns-query/{ClientID}",
		},
	}

	httpConf["doh"] = dohConf

	return moveVal[bool](tlsConf, dohConf, "allow_unencrypted_doh", "insecure_enabled")
}
