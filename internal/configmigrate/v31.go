package configmigrate

import "context"

// migrateTo31 performs the following changes:
//
//	# BEFORE:
//	'filtering':
//	  'rewrites':
//	    - 'domain': test.example
//	      'answer': 192.0.2.0
//	  # …
//	# …
//
//	# AFTER:
//	'filtering':
//	  'rewrites':
//	    - 'domain': test.example
//	      'answer': 192.0.2.0
//	      'enabled': true
//	  # …
//	# …
func (m *Migrator) migrateTo31(_ context.Context, diskConf yobj) (err error) {
	diskConf["schema_version"] = 31

	fltConf, ok, err := fieldVal[yobj](diskConf, "filtering")
	if !ok {
		return err
	}

	rewrites, ok, err := fieldVal[yarr](fltConf, "rewrites")
	if !ok {
		return err
	}

	for i := range rewrites {
		if r, isYobj := rewrites[i].(yobj); isYobj {
			r["enabled"] = true
		}
	}

	return nil
}
