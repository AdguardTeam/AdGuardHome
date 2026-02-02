package configmigrate

import "context"

// migrateTo33 performs the following changes:
//
//	# BEFORE:
//	'querylog':
//	  # …
//	  'ignored':
//	  - '|.^'
//	'statistics':
//	  # …
//	  'ignored':
//	  - '|.^'
//
//	# AFTER:
//	'querylog':
//	  # …
//	  'ignored':
//	  - '|.^'
//	  'ignored_enabled': true
//	'statistics':
//	  # …
//	  'ignored':
//	  - '|.^'
//	  'ignored_enabled': true
//
// If ignored is empty, then ignored_enabled should be false.
func (m Migrator) migrateTo33(_ context.Context, diskConf yobj) (err error) {
	diskConf["schema_version"] = 33

	queryLogConf, ok, err := fieldVal[yobj](diskConf, "querylog")
	if !ok {
		return err
	}

	ignored, ok, err := fieldVal[yarr](queryLogConf, "ignored")
	if !ok {
		return err
	}

	queryLogConf["ignored_enabled"] = len(ignored) > 0

	statisticsConf, ok, err := fieldVal[yobj](diskConf, "statistics")
	if !ok {
		return err
	}

	ignored, ok, err = fieldVal[yarr](statisticsConf, "ignored")
	if !ok {
		return err
	}

	statisticsConf["ignored_enabled"] = len(ignored) > 0

	return nil
}
