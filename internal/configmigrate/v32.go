package configmigrate

import "context"

// migrateTo32 performs the following changes:
//
//	# BEFORE:
//	'dns':
//	  'cache_enabled': true
//	  'cache_optimistic': true
//	  # …
//
//	# AFTER:
//	'dns':
//	  'cache_enabled': true
//	  'cache_optimistic': true
//	  'cache_optimistic_answer_ttl': '30s'
//	  'cache_optimistic_max_age': '12h'
//	  # …
//
// If cache_size is zero, then cache_enabled should be false.
func (m Migrator) migrateTo32(_ context.Context, diskConf yobj) (err error) {
	diskConf["schema_version"] = 32

	dnsConf, ok, err := fieldVal[yobj](diskConf, "dns")
	if !ok {
		return err
	}

	dnsConf["cache_optimistic_answer_ttl"] = "30s"
	dnsConf["cache_optimistic_max_age"] = "12h"

	return nil
}
