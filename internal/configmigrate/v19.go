package configmigrate

import (
	"context"

	"github.com/AdguardTeam/golibs/logutil/slogutil"
)

// migrateTo19 performs the following changes:
//
//	# BEFORE:
//	'schema_version': 18
//	'clients':
//	  'persistent':
//	  - 'name': 'client-name'
//	    'safesearch_enabled': true
//	    # …
//	  # …
//	# …
//
//	# AFTER:
//	'schema_version': 19
//	'clients':
//	  'persistent':
//	  - 'name': 'client-name'
//	    'safe_search':
//	      'enabled': true
//		  'bing': true
//		  'duckduckgo': true
//		  'google': true
//		  'pixabay': true
//		  'yandex': true
//		  'youtube': true
//	    # …
//	  # …
//	# …
func (m *Migrator) migrateTo19(ctx context.Context, diskConf yobj) (err error) {
	diskConf["schema_version"] = 19

	clients, ok, err := fieldVal[yobj](diskConf, "clients")
	if !ok {
		return err
	}

	persistent, ok, _ := fieldVal[yarr](clients, "persistent")
	if !ok {
		return nil
	}

	for _, p := range persistent {
		var c yobj
		c, ok = p.(yobj)
		if !ok {
			continue
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

		err = moveVal[bool](c, safeSearch, "safesearch_enabled", "enabled")
		if err != nil {
			m.logger.DebugContext(ctx, "migrating to", "version", 19, slogutil.KeyError, err)
		}

		c["safe_search"] = safeSearch
	}

	return nil
}
