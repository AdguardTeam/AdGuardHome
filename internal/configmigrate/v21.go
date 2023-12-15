package configmigrate

// migrateTo21 performs the following changes:
//
//	# BEFORE:
//	'schema_version': 20
//	'dns':
//	  'blocked_services':
//	  - 'svc_name'
//	  - # …
//	  # …
//	# …
//
//	# AFTER:
//	'schema_version': 21
//	'dns':
//	  'blocked_services':
//	    'ids':
//	    - 'svc_name'
//	    - # …
//	    'schedule':
//	      'time_zone': 'Local'
//	  # …
//	# …
func migrateTo21(diskConf yobj) (err error) {
	diskConf["schema_version"] = 21

	const field = "blocked_services"

	dns, ok, err := fieldVal[yobj](diskConf, "dns")
	if !ok {
		return err
	}

	svcs := yobj{
		"schedule": yobj{
			"time_zone": "Local",
		},
	}

	err = moveVal[yarr](dns, svcs, field, "ids")
	if err != nil {
		return err
	}

	dns[field] = svcs

	return nil
}
