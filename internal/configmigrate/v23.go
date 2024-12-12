package configmigrate

import (
	"fmt"
	"net/netip"
	"time"

	"github.com/AdguardTeam/golibs/timeutil"
)

// migrateTo23 performs the following changes:
//
//	# BEFORE:
//	'schema_version': 22
//	'bind_host': '1.2.3.4'
//	'bind_port': 8080
//	'web_session_ttl': 720
//	# …
//
//	# AFTER:
//	'schema_version': 23
//	'http':
//	  'address': '1.2.3.4:8080'
//	  'session_ttl': '720h'
//	# …
func migrateTo23(diskConf yobj) (err error) {
	diskConf["schema_version"] = 23

	bindHost, ok, err := fieldVal[string](diskConf, "bind_host")
	if !ok {
		return err
	}

	bindHostAddr, err := netip.ParseAddr(bindHost)
	if err != nil {
		return fmt.Errorf("invalid bind_host value: %s", bindHost)
	}

	bindPort, _, err := fieldVal[int](diskConf, "bind_port")
	if err != nil {
		return err
	}

	sessionTTL, _, err := fieldVal[int](diskConf, "web_session_ttl")
	if err != nil {
		return err
	}

	diskConf["http"] = yobj{
		"address":     netip.AddrPortFrom(bindHostAddr, uint16(bindPort)).String(),
		"session_ttl": timeutil.Duration(time.Duration(sessionTTL) * time.Hour).String(),
	}

	delete(diskConf, "bind_host")
	delete(diskConf, "bind_port")
	delete(diskConf, "web_session_ttl")

	return nil
}
