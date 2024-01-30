package configmigrate

// migrateTo27 performs the following changes:
//
//	# BEFORE:
//	'querylog':
//	  'ignored':
//	  - '.'
//	  - # …
//	  # …
//	'statistics':
//	  'ignored':
//	  - '.'
//	  - # …
//	  # …
//	# …
//
//	# AFTER:
//	'querylog':
//	  'ignored':
//	  - '|.^'
//	  - # …
//	  # …
//	'statistics':
//	  'ignored':
//	  - '|.^'
//	  - # …
//	  # …
//	# …
func migrateTo27(diskConf yobj) (err error) {
	diskConf["schema_version"] = 27

	keys := []string{"querylog", "statistics"}
	for _, k := range keys {
		err = replaceDot(diskConf, k)
		if err != nil {
			return err
		}
	}

	return nil
}

// replaceDot replaces rules blocking root domain "." with AdBlock style syntax
// "|.^".
func replaceDot(diskConf yobj, key string) (err error) {
	var obj yobj
	var ok bool
	obj, ok, err = fieldVal[yobj](diskConf, key)
	if err != nil {
		return err
	} else if !ok {
		return nil
	}

	var ignored yarr
	ignored, ok, err = fieldVal[yarr](obj, "ignored")
	if err != nil {
		return err
	} else if !ok {
		return nil
	}

	for i, hostVal := range ignored {
		var host string
		host, ok = hostVal.(string)
		if !ok {
			continue
		}

		if host == "." {
			ignored[i] = "|.^"
		}
	}

	return nil
}
