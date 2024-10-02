package configmigrate

import (
	"fmt"
	"path/filepath"
)

// migrateTo29 performs the following changes:
//
//	# BEFORE:
//	'filters':
//	  - 'enabled': true
//	    'url': /path/to/file.txt
//	    'name': My FS Filter
//	    'id': 1234
//
//	# AFTER:
//	'filters':
//	  - 'enabled': true
//	    'url': /path/to/file.txt
//	    'name': My FS Filter
//	    'id': 1234
//	# …
//	'filtering':
//	  'safe_fs_patterns':
//	    - '/opt/AdGuardHome/data/userfilters/*'
//	    - '/path/to/file.txt'
//	  # …
func (m Migrator) migrateTo29(diskConf yobj) (err error) {
	diskConf["schema_version"] = 29

	filterVals, ok, err := fieldVal[[]any](diskConf, "filters")
	if !ok {
		return err
	}

	paths := []string{
		filepath.Join(m.dataDir, "userfilters", "*"),
	}

	for i, v := range filterVals {
		var f yobj
		f, ok = v.(yobj)
		if !ok {
			return fmt.Errorf("filters: at index %d: expected object, got %T", i, v)
		}

		var u string
		u, ok, _ = fieldVal[string](f, "url")
		if ok && filepath.IsAbs(u) {
			paths = append(paths, u)
		}
	}

	fltConf, ok, err := fieldVal[yobj](diskConf, "filtering")
	if !ok {
		return err
	}

	fltConf["safe_fs_patterns"] = paths

	return nil
}
