package dhcpsvc

import (
	"fmt"
	"net/netip"
	"time"

	"github.com/AdguardTeam/golibs/log"
	"github.com/google/gopacket/layers"
	"golang.org/x/exp/slices"
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

// iface6 is a DHCP interface for IPv6 address family.
//
// TODO(e.burkov):  Add options.
type iface6 struct {
	// rangeStart is the first IP address in the range.
	rangeStart netip.Addr

	// name is the name of the interface.
	name string

	// implicitOpts are the DHCPv6 options listed in RFC 8415 (and others) and
	// initialized with default values.  It must not have intersections with
	// explicitOpts.
	implicitOpts layers.DHCPv6Options

	// explicitOpts are the user-configured options.  It must not have
	// intersections with implicitOpts.
	explicitOpts layers.DHCPv6Options

	// leaseTTL is the time-to-live of dynamic leases on this interface.
	leaseTTL time.Duration

	// raSLAACOnly defines if DHCP should send ICMPv6.RA packets without MO
	// flags.
	raSLAACOnly bool

	// raAllowSLAAC defines if DHCP should send ICMPv6.RA packets with MO flags.
	raAllowSLAAC bool
}

// newIface6 creates a new DHCP interface for IPv6 address family with the given
// configuration.
//
// TODO(e.burkov):  Validate properly.
func newIface6(name string, conf *IPv6Config) (i *iface6) {
	if !conf.Enabled {
		return nil
	}

	i = &iface6{
		name:         name,
		rangeStart:   conf.RangeStart,
		leaseTTL:     conf.LeaseDuration,
		raSLAACOnly:  conf.RASLAACOnly,
		raAllowSLAAC: conf.RAAllowSLAAC,
	}
	i.implicitOpts, i.explicitOpts = conf.options()

	return i
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
