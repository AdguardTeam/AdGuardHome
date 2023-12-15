package configmigrate

import "fmt"

// migrateTo6 performs the following changes:
//
//	# BEFORE:
//	'schema_version': 5
//	'clients':
//	 - # …
//	   'ip': '127.0.0.1'
//	   'mac': 'AA:AA:AA:AA:AA:AA'
//	  # …
//	# …
//
//	# AFTER:
//	'schema_version': 6
//	'clients':
//	 - # …
//	   'ip': '127.0.0.1'
//	   'mac': 'AA:AA:AA:AA:AA:AA'
//	   'ids':
//	   - '127.0.0.1'
//	   - 'AA:AA:AA:AA:AA:AA'
//	  # …
//	# …
func migrateTo6(diskConf yobj) (err error) {
	diskConf["schema_version"] = 6

	clients, ok, err := fieldVal[yarr](diskConf, "clients")
	if !ok {
		return err
	}

	for i, client := range clients {
		var c yobj
		c, ok = client.(yobj)
		if !ok {
			return fmt.Errorf("unexpected type of client at index %d: %T", i, client)
		}

		ids := yarr{}
		for _, id := range []string{"ip", "mac"} {
			val, _, valErr := fieldVal[string](c, id)
			if valErr != nil {
				return fmt.Errorf("client at index %d: %w", i, valErr)
			} else if val != "" {
				ids = append(ids, val)
			}
		}

		c["ids"] = ids
	}

	return nil
}
