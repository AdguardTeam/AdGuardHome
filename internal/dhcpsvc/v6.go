package dhcpsvc

import (
	"context"
	"fmt"
	"log/slog"
	"net/netip"
	"slices"
	"time"

	"github.com/AdguardTeam/golibs/errors"
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
func (c *IPv6Config) validate() (err error) {
	if c == nil {
		return errNilConfig
	} else if !c.Enabled {
		return nil
	}

	var errs []error

	if !c.RangeStart.Is6() {
		err = fmt.Errorf("range start %s should be a valid ipv6", c.RangeStart)
		errs = append(errs, err)
	}

	if c.LeaseDuration <= 0 {
		err = fmt.Errorf("lease duration %s must be positive", c.LeaseDuration)
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

// dhcpInterfaceV6 is a DHCP interface for IPv6 address family.
type dhcpInterfaceV6 struct {
	// common is the common part of any network interface within the DHCP
	// server.
	common *netInterface

	// rangeStart is the first IP address in the range.
	rangeStart netip.Addr

	// implicitOpts are the DHCPv6 options listed in RFC 8415 (and others) and
	// initialized with default values.  It must not have intersections with
	// explicitOpts.
	implicitOpts layers.DHCPv6Options

	// explicitOpts are the user-configured options.  It must not have
	// intersections with implicitOpts.
	explicitOpts layers.DHCPv6Options

	// raSLAACOnly defines if DHCP should send ICMPv6.RA packets without MO
	// flags.
	raSLAACOnly bool

	// raAllowSLAAC defines if DHCP should send ICMPv6.RA packets with MO flags.
	raAllowSLAAC bool
}

// newDHCPInterfaceV6 creates a new DHCP interface for IPv6 address family with
// the given configuration.
//
// TODO(e.burkov):  Validate properly.
func newDHCPInterfaceV6(
	ctx context.Context,
	l *slog.Logger,
	name string,
	conf *IPv6Config,
) (i *dhcpInterfaceV6) {
	l = l.With(keyInterface, name, keyFamily, netutil.AddrFamilyIPv6)
	if !conf.Enabled {
		l.DebugContext(ctx, "disabled")

		return nil
	}

	i = &dhcpInterfaceV6{
		rangeStart:   conf.RangeStart,
		common:       newNetInterface(name, l, conf.LeaseDuration),
		raSLAACOnly:  conf.RASLAACOnly,
		raAllowSLAAC: conf.RAAllowSLAAC,
	}
	i.implicitOpts, i.explicitOpts = conf.options(ctx, l)

	return i
}

// dhcpInterfacesV6 is a slice of network interfaces of IPv6 address family.
type dhcpInterfacesV6 []*dhcpInterfaceV6

// find returns the first network interface within ifaces containing ip.  It
// returns false if there is no such interface.
func (ifaces dhcpInterfacesV6) find(ip netip.Addr) (iface6 *netInterface, ok bool) {
	// prefLen is the length of prefix to match ip against.
	//
	// TODO(e.burkov):  DHCPv6 inherits the weird behavior of legacy
	// implementation where the allocated range constrained by the first address
	// and the first address with last byte set to 0xff.  Proper prefixes should
	// be used instead.
	const prefLen = netutil.IPv6BitLen - 8

	i := slices.IndexFunc(ifaces, func(iface *dhcpInterfaceV6) (contains bool) {
		return !ip.Less(iface.rangeStart) &&
			netip.PrefixFrom(iface.rangeStart, prefLen).Contains(ip)
	})
	if i < 0 {
		return nil, false
	}

	return ifaces[i].common, true
}

// options returns the implicit and explicit options for the interface.  The two
// lists are disjoint and the implicit options are initialized with default
// values.
//
// TODO(e.burkov):  Add implicit options according to RFC.
func (c *IPv6Config) options(ctx context.Context, l *slog.Logger) (imp, exp layers.DHCPv6Options) {
	// Set default values of host configuration parameters listed in RFC 8415.
	imp = layers.DHCPv6Options{}
	slices.SortFunc(imp, compareV6OptionCodes)

	// Set values for explicitly configured options.
	for _, e := range c.Options {
		i, found := slices.BinarySearchFunc(imp, e, compareV6OptionCodes)
		if found {
			imp = slices.Delete(imp, i, i+1)
		}

		exp = append(exp, e)
	}

	l.DebugContext(ctx, "options", "implicit", imp, "explicit", exp)

	return imp, exp
}

// compareV6OptionCodes compares option codes of a and b.
func compareV6OptionCodes(a, b layers.DHCPv6Option) (res int) {
	return int(a.Code) - int(b.Code)
}
