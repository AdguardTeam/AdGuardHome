package confmigrate

// migrateTo15 performs the following changes:
//
//	# BEFORE:
//	'schema_version': 14
//	'dns':
//	  # …
//	  'querylog_enabled': true
//	  'querylog_file_enabled': true
//	  'querylog_interval': '2160h'
//	  'querylog_size_memory': 1000
//	'querylog':
//	  # …
//	# …
//
//	# AFTER:
//	'schema_version': 15
//	'dns':
//	  # …
//	'querylog':
//	  'enabled': true
//	  'file_enabled': true
//	  'interval': '2160h'
//	  'size_memory': 1000
//	  'ignored': []
//	  # …
//	# …
func migrateTo15(diskConf yobj) (err error) {
	diskConf["schema_version"] = 15

	dns, ok, err := fieldVal[yobj](diskConf, "dns")
	if err != nil {
		return err
	} else if !ok {
		return nil
	}

	qlog := map[string]any{
		"ignored":      yarr{},
		"enabled":      true,
		"file_enabled": true,
		"interval":     "2160h",
		"size_memory":  1000,
	}
	err = coalesceError(
		moveVal[bool](dns, qlog, "querylog_enabled", "enabled"),
		moveVal[bool](dns, qlog, "querylog_file_enabled", "file_enabled"),
		moveVal[string](dns, qlog, "querylog_interval", "interval"),
		moveVal[int](dns, qlog, "querylog_size_memory", "size_memory"),
	)
	if err != nil {
		return err
	}

	diskConf["querylog"] = qlog

	return nil
}
