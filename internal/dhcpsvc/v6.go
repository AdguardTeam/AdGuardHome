package dhcpsvc

import (
	"context"
	"log/slog"
	"net/netip"
	"slices"
	"time"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/validate"
	"github.com/google/gopacket/layers"
)

// Port numbers for DHCPv6.
//
// See RFC 9915 Section 7.2.
const (
	// ServerPortV6 is the standard DHCPv6 server port.
	ServerPortV6 layers.UDPPort = 547

	// ClientPortV6 is the standard DHCPv6 client port.
	ClientPortV6 layers.UDPPort = 546
)

// HardwareTypeEthernet is the IANA hardware type number for Ethernet, used in
// DUID-LL and DUID-LLT construction.  Its value is 1, encoded as a big-endian
// uint16.
//
// See https://www.iana.org/assignments/arp-parameters/arp-parameters.xhtml#arp-parameters-2.
//
// TODO(e.burkov):  Use.
var HardwareTypeEthernet = []byte{0x00, 0x01}

// DHCPv6 multicast addresses.
//
// See RFC 9915 Section 7.1.
var (
	// AllDHCPRelayAgentsAndServers is the well-known IPv6 multicast address
	// All_DHCP_Relay_Agents_and_Servers.  Clients send messages to this address
	// to reach all servers on the local link.
	AllDHCPRelayAgentsAndServers = netip.MustParseAddr("ff02::1:2")

	// AllDHCPServers is the well-known IPv6 multicast address All_DHCP_Servers.
	// Relay agents use this to reach all servers.
	AllDHCPServers = netip.MustParseAddr("ff05::1:3")
)

// v6PrefLen is the length of prefix to match ip against.
//
// TODO(e.burkov):  DHCPv6 inherits the weird behavior of legacy implementation
// where the allocated range constrained by the first address and the first
// address with last byte set to 0xff.  Proper prefixes should be used instead.
const v6PrefLen = netutil.IPv6BitLen - 8

// IPv6Config is the interface-specific configuration for DHCPv6.
//
// TODO(e.burkov):  Add RangeEnd and SubnetPrefix fields, and validate them.
type IPv6Config struct {
	// RangeStart is the first address in the range to assign to DHCP clients.
	// It should be a valid IPv6 address.
	RangeStart netip.Addr

	// Options is the list of explicit DHCP options to send to clients.  The
	// options with zero length are treated as deletions of the corresponding
	// options, either implicit or explicit.
	Options layers.DHCPv6Options

	// LeaseDuration is the TTL of a DHCP lease.  It should be positive.
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

// type check
var _ validate.Interface = (*IPv6Config)(nil)

// Validate implements the [validate.Interface] interface for *IPv6Config.
func (c *IPv6Config) Validate() (err error) {
	if c == nil {
		return errors.ErrNoValue
	} else if !c.Enabled {
		return nil
	}

	errs := []error{
		validate.Positive("lease duration", c.LeaseDuration),
	}

	errs = c.validateSubnet(errs)

	return errors.Join(errs...)
}

// validateSubnet validates the subnet configuration.
//
// TODO(e.burkov):  Use [validate].
func (c *IPv6Config) validateSubnet(orig []error) (errs []error) {
	errs = orig

	if !c.RangeStart.Is6() {
		err := newMustErr("range start", "be a valid ipv6", c.RangeStart)
		errs = append(errs, err)
	}

	return errs
}

// dhcpInterfaceV6 is a DHCP interface for IPv6 address family.
type dhcpInterfaceV6 struct {
	// common is the common part of any network interface within the DHCP
	// server.
	common *netInterface

	// subnetPrefix is the network prefix of the interface's IPv6 subnet.  It is
	// used for on-link address determination.
	subnetPrefix netip.Prefix

	// implicitOpts are the DHCPv6 options listed in RFC 8415 (and others) and
	// initialized with default values.  It must not have intersections with
	// explicitOpts.
	implicitOpts layers.DHCPv6Options

	// explicitOpts are the user-configured options.  It must not have
	// intersections with implicitOpts.
	explicitOpts layers.DHCPv6Options

	// t1 is the pre-computed T1 value (0.5 × LeaseDuration) per RFC 9915 §21.4.
	// It is the time after which the client should contact the same server to
	// extend the lease.
	t1 time.Duration

	// t2 is the pre-computed T2 value (0.8 × LeaseDuration) per RFC 9915 §21.4.
	// It is the time after which the client may contact any server to extend
	// the lease.
	t2 time.Duration

	// raSLAACOnly defines if DHCP should send ICMPv6.RA packets without MO
	// flags.
	raSLAACOnly bool

	// raAllowSLAAC defines if DHCP should send ICMPv6.RA packets with MO flags.
	raAllowSLAAC bool
}

// newDHCPInterfaceV6 creates a new DHCP interface for IPv6 address family with
// the given configuration.  If the interface is disabled, it returns nil.  conf
// must be valid.
func (srv *DHCPServer) newDHCPInterfaceV6(
	ctx context.Context,
	l *slog.Logger,
	name string,
	conf *IPv6Config,
) (iface *dhcpInterfaceV6) {
	if !conf.Enabled {
		l.DebugContext(ctx, "disabled")

		return nil
	}

	// TODO(e.burkov):  Migrate the configuration to use proper range start,
	// end, and subnet prefix.
	rangeEndData := conf.RangeStart.As16()
	rangeEndData[15] = 0xff

	// TODO(e.burkov):  Validate the range end and subnet prefix against the
	// range start during configuration validation.
	addrSpace, _ := newIPRange(conf.RangeStart, netip.AddrFrom16(rangeEndData))

	iface = &dhcpInterfaceV6{
		common: &netInterface{
			logger:        l,
			leases:        map[macKey]*Lease{},
			indexMu:       srv.leasesMu,
			index:         srv.leases,
			name:          name,
			addrSpace:     addrSpace,
			leasedOffsets: newBitSet(),
			leaseTTL:      conf.LeaseDuration,
		},
		subnetPrefix: netip.PrefixFrom(conf.RangeStart, v6PrefLen),
		t1:           conf.LeaseDuration / 2,
		t2:           conf.LeaseDuration * 4 / 5,
		raSLAACOnly:  conf.RASLAACOnly,
		raAllowSLAAC: conf.RAAllowSLAAC,
	}
	iface.implicitOpts, iface.explicitOpts = conf.options(ctx, l)

	return iface
}

// dhcpInterfacesV6 is a slice of network interfaces of IPv6 address family.
type dhcpInterfacesV6 []*dhcpInterfaceV6

// find returns the first network interface within ifaces whose subnet prefix
// contains ip.  It returns false if there is no such interface.
func (ifaces dhcpInterfacesV6) find(ip netip.Addr) (iface6 *netInterface, ok bool) {
	i := slices.IndexFunc(ifaces, func(iface *dhcpInterfaceV6) (contains bool) {
		return iface.subnetPrefix.Contains(ip)
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
