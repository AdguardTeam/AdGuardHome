package dhcpsvc

import (
	"fmt"
	"sync/atomic"
	"time"

	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

// DHCPServer is a DHCP server for both IPv4 and IPv6 address families.
type DHCPServer struct {
	// enabled indicates whether the DHCP server is enabled and can provide
	// information about its clients.
	enabled *atomic.Bool

	// localTLD is the top-level domain name to use for resolving DHCP
	// clients' hostnames.
	localTLD string

	// interfaces4 is the set of IPv4 interfaces sorted by interface name.
	interfaces4 []*iface4

	// interfaces6 is the set of IPv6 interfaces sorted by interface name.
	interfaces6 []*iface6

	// icmpTimeout is the timeout for checking another DHCP server's presence.
	icmpTimeout time.Duration
}

// New creates a new DHCP server with the given configuration.  It returns an
// error if the given configuration can't be used.
//
// TODO(e.burkov):  Use.
func New(conf *Config) (srv *DHCPServer, err error) {
	if !conf.Enabled {
		// TODO(e.burkov):  Perhaps return [Empty]?
		return nil, nil
	}

	ifaces4 := make([]*iface4, len(conf.Interfaces))
	ifaces6 := make([]*iface6, len(conf.Interfaces))

	ifaceNames := maps.Keys(conf.Interfaces)
	slices.Sort(ifaceNames)

	var i4 *iface4
	var i6 *iface6

	for _, ifaceName := range ifaceNames {
		iface := conf.Interfaces[ifaceName]

		i4, err = newIface4(ifaceName, iface.IPv4)
		if err != nil {
			return nil, fmt.Errorf("interface %q: ipv4: %w", ifaceName, err)
		} else if i4 != nil {
			ifaces4 = append(ifaces4, i4)
		}

		i6 = newIface6(ifaceName, iface.IPv6)
		if i6 != nil {
			ifaces6 = append(ifaces6, i6)
		}
	}

	enabled := &atomic.Bool{}
	enabled.Store(conf.Enabled)

	return &DHCPServer{
		enabled:     enabled,
		interfaces4: ifaces4,
		interfaces6: ifaces6,
		localTLD:    conf.LocalDomainName,
		icmpTimeout: conf.ICMPTimeout,
	}, nil
}
