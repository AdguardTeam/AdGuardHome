package configmigrate

// migrateTo18 performs the following changes:
//
//	# BEFORE:
//	'schema_version': 17
//	'dns':
//	  'safesearch_enabled': true
//	  # …
//	# …
//
//	# AFTER:
//	'schema_version': 18
//	'dns':
//	  'safe_search':
//	    'enabled': true
//	    'bing': true
//	    'duckduckgo': true
//	    'google': true
//	    'pixabay': true
//	    'yandex': true
//	    'youtube': true
//	  # …
//	# …
func migrateTo18(diskConf yobj) (err error) {
	diskConf["schema_version"] = 18

	dns, ok, err := fieldVal[yobj](diskConf, "dns")
	if !ok {
		return err
	}

	safeSearch := yobj{
		"enabled":    true,
		"bing":       true,
		"duckduckgo": true,
		"google":     true,
		"pixabay":    true,
		"yandex":     true,
		"youtube":    true,
	}
	dns["safe_search"] = safeSearch

	return moveVal[bool](dns, safeSearch, "safesearch_enabled", "enabled")
}
