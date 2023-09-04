package confmigrate

import "fmt"

// migrateTo6 performs the following changes:
//
//	# BEFORE:
//	'schema_version': 5
//	'clients':
//	 - # …
//	   'ip': '127.0.0.1'
//	   'mac': 'AA:AA:AA:AA:AA:AA'
//	# …
//
//	# AFTER:
//	'schema_version': 6
//	'clients':
//	 - # …
//	   'ids':
//	   - '127.0.0.1'
//	   - 'AA:AA:AA:AA:AA:AA'
//	# …
func migrateTo6(diskConf yobj) (err error) {
	diskConf["schema_version"] = 6

	clients, ok, err := fieldVal[yarr](diskConf, "clients")
	if err != nil {
		return err
	} else if !ok {
		return nil
	}

	for i, client := range clients {
		var c yobj
		c, ok = client.(yobj)
		if !ok {
			return fmt.Errorf("unexpected type of client at index %d: %T", i, client)
		}

		var ids []string

		var ip string
		ip, _, err = fieldVal[string](c, "ip")
		if err != nil {
			return fmt.Errorf("client at index %d: %w", i, err)
		} else if ip != "" {
			ids = append(ids, ip)
		}

		var mac string
		mac, _, err = fieldVal[string](c, "mac")
		if err != nil {
			return fmt.Errorf("client at index %d: %w", i, err)
		} else if mac != "" {
			ids = append(ids, mac)
		}

		c["ids"] = ids
	}

	return nil
}
