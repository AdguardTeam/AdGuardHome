package configmigrate

import "github.com/AdguardTeam/golibs/errors"

// migrateTo26 performs the following changes:
//
//	# BEFORE:
//	'schema_version': 25
//	'dns':
//	  'filtering_enabled': true
//	  'filters_update_interval': 24
//	  'parental_enabled': false
//	  'safebrowsing_enabled': false
//	  'safebrowsing_cache_size': 1048576
//	  'safesearch_cache_size': 1048576
//	  'parental_cache_size': 1048576
//	  'safe_search':
//	    'enabled': false
//	    'bing': true
//	    'duckduckgo': true
//	    'google': true
//	    'pixabay': true
//	    'yandex': true
//	    'youtube': true
//	  'rewrites': []
//	  'blocked_services':
//	    'schedule':
//	      'time_zone': 'Local'
//	    'ids': []
//	  'protection_enabled':        true,
//	  'blocking_mode':             'custom_ip',
//	  'blocking_ipv4':             '1.2.3.4',
//	  'blocking_ipv6':             '1:2:3::4',
//	  'blocked_response_ttl':      10,
//	  'protection_disabled_until': 'null',
//	  'parental_block_host':       'p.dns.adguard.com',
//	  'safebrowsing_block_host':   's.dns.adguard.com',
//	# …
//
//	# AFTER:
//	'schema_version': 26
//	'filtering':
//	  'filtering_enabled': true
//	  'filters_update_interval': 24
//	  'parental_enabled': false
//	  'safebrowsing_enabled': false
//	  'safebrowsing_cache_size': 1048576
//	  'safesearch_cache_size': 1048576
//	  'parental_cache_size': 1048576
//	  'safe_search':
//	    'enabled': false
//	    'bing': true
//	    'duckduckgo': true
//	    'google': true
//	    'pixabay': true
//	    'yandex': true
//	    'youtube': true
//	  'rewrites': []
//	  'blocked_services':
//	    'schedule':
//	      'time_zone': 'Local'
//	    'ids': []
//	  'protection_enabled':        true,
//	  'blocking_mode':             'custom_ip',
//	  'blocking_ipv4':             '1.2.3.4',
//	  'blocking_ipv6':             '1:2:3::4',
//	  'blocked_response_ttl':      10,
//	  'protection_disabled_until': 'null',
//	  'parental_block_host':       'p.dns.adguard.com',
//	  'safebrowsing_block_host':   's.dns.adguard.com',
//	'dns'
//	  # …
//	# …
func migrateTo26(diskConf yobj) (err error) {
	diskConf["schema_version"] = 26

	dns, ok, err := fieldVal[yobj](diskConf, "dns")
	if !ok {
		return err
	}

	filteringObj := yobj{}
	err = errors.Join(
		moveSameVal[bool](dns, filteringObj, "filtering_enabled"),
		moveSameVal[int](dns, filteringObj, "filters_update_interval"),
		moveSameVal[bool](dns, filteringObj, "parental_enabled"),
		moveSameVal[bool](dns, filteringObj, "safebrowsing_enabled"),
		moveSameVal[int](dns, filteringObj, "safebrowsing_cache_size"),
		moveSameVal[int](dns, filteringObj, "safesearch_cache_size"),
		moveSameVal[int](dns, filteringObj, "parental_cache_size"),
		moveSameVal[yobj](dns, filteringObj, "safe_search"),
		moveSameVal[yarr](dns, filteringObj, "rewrites"),
		moveSameVal[yobj](dns, filteringObj, "blocked_services"),
		moveSameVal[bool](dns, filteringObj, "protection_enabled"),
		moveSameVal[string](dns, filteringObj, "blocking_mode"),
		moveSameVal[string](dns, filteringObj, "blocking_ipv4"),
		moveSameVal[string](dns, filteringObj, "blocking_ipv6"),
		moveSameVal[int](dns, filteringObj, "blocked_response_ttl"),
		moveSameVal[any](dns, filteringObj, "protection_disabled_until"),
		moveSameVal[string](dns, filteringObj, "parental_block_host"),
		moveSameVal[string](dns, filteringObj, "safebrowsing_block_host"),
	)
	if err != nil {
		// Don't wrap the error, because it's informative enough as is.
		return err
	}

	if len(filteringObj) != 0 {
		diskConf["filtering"] = filteringObj
	}

	return nil
}
