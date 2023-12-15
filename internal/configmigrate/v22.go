package configmigrate

import (
	"fmt"
)

// migrateTo22 performs the following changes:
//
//	# BEFORE:
//	'schema_version': 21
//	'persistent':
//	  - 'name': 'client_name'
//	    'blocked_services':
//	    - 'svc_name'
//	    - # …
//	    # …
//	  # …
//	# …
//
//	# AFTER:
//	'schema_version': 22
//	'persistent':
//	  - 'name': 'client_name'
//	    'blocked_services':
//	      'ids':
//	      - 'svc_name'
//	      - # …
//	      'schedule':
//	        'time_zone': 'Local'
//	    # …
//	  # …
//	# …
func migrateTo22(diskConf yobj) (err error) {
	diskConf["schema_version"] = 22

	const field = "blocked_services"

	clients, ok, err := fieldVal[yobj](diskConf, "clients")
	if !ok {
		return err
	}

	persistent, ok, err := fieldVal[yarr](clients, "persistent")
	if !ok {
		return err
	}

	for i, p := range persistent {
		var c yobj
		c, ok = p.(yobj)
		if !ok {
			return fmt.Errorf("persistent client at index %d: unexpected type %T", i, p)
		}

		var services yarr
		services, ok, err = fieldVal[yarr](c, field)
		if err != nil {
			return fmt.Errorf("persistent client at index %d: %w", i, err)
		} else if !ok {
			continue
		}

		c[field] = yobj{
			"ids": services,
			"schedule": yobj{
				"time_zone": "Local",
			},
		}
	}

	return nil
}
