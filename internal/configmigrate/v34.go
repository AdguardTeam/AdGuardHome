package configmigrate

import "context"

// migrateTo34 performs the following changes:
//
//	# BEFORE:
//	'http':
//	  # …
//
//	# AFTER:
//	'http':
//	  # …
//	  'http':
//	  'doh':
//	    'routes':
//	      - 'GET /dns-query'
//	      - 'POST /dns-query'
//	      - 'GET /dns-query/{ClientID}'
//	      - 'POST /dns-query/{ClientID}'
//	  'insecure_enabled': false
func (m *Migrator) migrateTo34(_ context.Context, diskConf yobj) (err error) {
	diskConf["schema_version"] = 34

	httpObj, ok, err := fieldVal[yobj](diskConf, "http")
	if !ok {
		return err
	}

	// Migrate allow_unencrypted_doh from tls to http.doh.insecure_enabled.
	tlsConf, ok, err := fieldVal[yobj](diskConf, "tls")
	if !ok {
		return err
	}

	allowUnencrypted, ok, err := fieldVal[bool](tlsConf, "allow_unencrypted_doh")
	if !ok {
		return err
	}

	// Create doh section with default routes.
	dohObj := yobj{
		"routes": yarr{
			"GET /dns-query",
			"POST /dns-query",
			"GET /dns-query/{ClientID}",
			"POST /dns-query/{ClientID}",
		},
		"insecure_enabled": allowUnencrypted,
	}

	httpObj["doh"] = dohObj

	return nil
}
