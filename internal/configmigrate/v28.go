package configmigrate

import (
	"github.com/AdguardTeam/AdGuardHome/internal/dnsforward"
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
func migrateTo28(diskConf yobj) (err error) {
	diskConf["schema_version"] = 28

	dns, ok, err := fieldVal[yobj](diskConf, "dns")
	if !ok {
		return err
	}

	allServers, _, _ := fieldVal[bool](dns, "all_servers")
	fastestAddr, _, _ := fieldVal[bool](dns, "fastest_addr")

	var upstreamModeType dnsforward.UpstreamMode
	if allServers {
		upstreamModeType = dnsforward.UpstreamModeParallel
	} else if fastestAddr {
		upstreamModeType = dnsforward.UpstreamModeFastestAddr
	} else {
		upstreamModeType = dnsforward.UpstreamModeLoadBalance
	}

	dns["upstream_mode"] = upstreamModeType

	delete(dns, "all_servers")
	delete(dns, "fastest_addr")

	return nil
}
