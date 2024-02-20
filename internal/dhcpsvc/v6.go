package dhcpsvc

import (
	"fmt"
	"net/netip"
	"slices"
	"time"

	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/google/gopacket/layers"
)

// IPv6Config is the interface-specific configuration for DHCPv6.
type IPv6Config struct {
	// RangeStart is the first address in the range to assign to DHCP clients.
	RangeStart netip.Addr

	// Options is the list of DHCP options to send to DHCP clients.  The options
	// with zero length are treated as deletions of the corresponding options,
	// either implicit or explicit.
	Options layers.DHCPv6Options

	// LeaseDuration is the TTL of a DHCP lease.
	LeaseDuration time.Duration

	// RASlaacOnly defines whether the DHCP clients should only use SLAAC for
	// address assignment.
	RASLAACOnly bool

	// RAAllowSlaac defines whether the DHCP clients may use SLAAC for address
	// assignment.
	RAAllowSLAAC bool

	// Enabled is the state of the DHCPv6 service, whether it is enabled or not
	// on the specific interface.
	Enabled bool
}

// validate returns an error in conf if any.
func (conf *IPv6Config) validate() (err error) {
	switch {
	case conf == nil:
		return errNilConfig
	case !conf.Enabled:
		return nil
	case !conf.RangeStart.Is6():
		return fmt.Errorf("range start %s should be a valid ipv6", conf.RangeStart)
	case conf.LeaseDuration <= 0:
		return fmt.Errorf("lease duration %s must be positive", conf.LeaseDuration)
	default:
		return nil
	}
}

// options returns the implicit and explicit options for the interface.  The two
// lists are disjoint and the implicit options are initialized with default
// values.
//
// TODO(e.burkov):  Add implicit options according to RFC.
func (conf *IPv6Config) options() (implicit, explicit layers.DHCPv6Options) {
	// Set default values of host configuration parameters listed in RFC 8415.
	implicit = layers.DHCPv6Options{}
	slices.SortFunc(implicit, compareV6OptionCodes)

	// Set values for explicitly configured options.
	for _, exp := range conf.Options {
		i, found := slices.BinarySearchFunc(implicit, exp, compareV6OptionCodes)
		if found {
			implicit = slices.Delete(implicit, i, i+1)
		}

		explicit = append(explicit, exp)
	}

	log.Debug("dhcpsvc: v6: implicit options: %s", implicit)
	log.Debug("dhcpsvc: v6: explicit options: %s", explicit)

	return implicit, explicit
}

// compareV6OptionCodes compares option codes of a and b.
func compareV6OptionCodes(a, b layers.DHCPv6Option) (res int) {
	return int(a.Code) - int(b.Code)
}

// netInterfaceV6 is a DHCP interface for IPv6 address family.
//
// TODO(e.burkov):  Add options.
type netInterfaceV6 struct {
	// rangeStart is the first IP address in the range.
	rangeStart netip.Addr

	// implicitOpts are the DHCPv6 options listed in RFC 8415 (and others) and
	// initialized with default values.  It must not have intersections with
	// explicitOpts.
	implicitOpts layers.DHCPv6Options

	// explicitOpts are the user-configured options.  It must not have
	// intersections with implicitOpts.
	explicitOpts layers.DHCPv6Options

	// netInterface is embedded here to provide some common network interface
	// logic.
	netInterface

	// raSLAACOnly defines if DHCP should send ICMPv6.RA packets without MO
	// flags.
	raSLAACOnly bool

	// raAllowSLAAC defines if DHCP should send ICMPv6.RA packets with MO flags.
	raAllowSLAAC bool
}

// newNetInterfaceV6 creates a new DHCP interface for IPv6 address family with
// the given configuration.
//
// TODO(e.burkov):  Validate properly.
func newNetInterfaceV6(name string, conf *IPv6Config) (i *netInterfaceV6) {
	if !conf.Enabled {
		return nil
	}

	i = &netInterfaceV6{
		rangeStart: conf.RangeStart,
		netInterface: netInterface{
			name:     name,
			leaseTTL: conf.LeaseDuration,
		},
		raSLAACOnly:  conf.RASLAACOnly,
		raAllowSLAAC: conf.RAAllowSLAAC,
	}
	i.implicitOpts, i.explicitOpts = conf.options()

	return i
}

// netInterfacesV4 is a slice of network interfaces of IPv4 address family.
type netInterfacesV6 []*netInterfaceV6

// find returns the first network interface within ifaces containing ip.  It
// returns false if there is no such interface.
func (ifaces netInterfacesV6) find(ip netip.Addr) (iface6 *netInterface, ok bool) {
	// prefLen is the length of prefix to match ip against.
	//
	// TODO(e.burkov):  DHCPv6 inherits the weird behavior of legacy
	// implementation where the allocated range constrained by the first address
	// and the first address with last byte set to 0xff.  Proper prefixes should
	// be used instead.
	const prefLen = netutil.IPv6BitLen - 8

	i := slices.IndexFunc(ifaces, func(iface *netInterfaceV6) (contains bool) {
		return !ip.Less(iface.rangeStart) &&
			netip.PrefixFrom(iface.rangeStart, prefLen).Contains(ip)
	})
	if i < 0 {
		return nil, false
	}

	return &ifaces[i].netInterface, true
}
