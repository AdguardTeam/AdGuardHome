package configmigrate

import (
	"context"

	"github.com/AdguardTeam/dnsproxy/proxy"
)

// migrateTo28 performs the following changes:
//
//	# BEFORE:
//	'dns':
//	  'all_servers': true
//	  'fastest_addr': true
//	  # …
//	# …
//
//	# AFTER:
//	'dns':
//	  'upstream_mode': 'parallel'
//	  # …
//	# …
func (m *Migrator) migrateTo28(_ context.Context, diskConf yobj) (err error) {
	diskConf["schema_version"] = 28

	dns, ok, err := fieldVal[yobj](diskConf, "dns")
	if !ok {
		return err
	}

	allServers, _, _ := fieldVal[bool](dns, "all_servers")
	fastestAddr, _, _ := fieldVal[bool](dns, "fastest_addr")

	var upstreamModeType proxy.UpstreamMode
	if allServers {
		upstreamModeType = proxy.UpstreamModeParallel
	} else if fastestAddr {
		upstreamModeType = proxy.UpstreamModeFastestAddr
	} else {
		upstreamModeType = proxy.UpstreamModeLoadBalance
	}

	dns["upstream_mode"] = upstreamModeType

	delete(dns, "all_servers")
	delete(dns, "fastest_addr")

	return nil
}
