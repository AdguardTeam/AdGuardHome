package dhcpsvc

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"slices"
	"time"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/google/gopacket/layers"
)

// IPv4Config is the interface-specific configuration for DHCPv4.
type IPv4Config struct {
	// GatewayIP is the IPv4 address of the network's gateway.  It is used as
	// the default gateway for DHCP clients and also used in calculating the
	// network-specific broadcast address.
	GatewayIP netip.Addr

	// SubnetMask is the IPv4 subnet mask of the network.  It should be a valid
	// IPv4 CIDR (i.e. all 1s followed by all 0s).
	SubnetMask netip.Addr

	// RangeStart is the first address in the range to assign to DHCP clients.
	RangeStart netip.Addr

	// RangeEnd is the last address in the range to assign to DHCP clients.
	RangeEnd netip.Addr

	// Options is the list of DHCP options to send to DHCP clients.  The options
	// having a zero value within the Length field are treated as deletions of
	// the corresponding options, either implicit or explicit.
	Options layers.DHCPOptions

	// LeaseDuration is the TTL of a DHCP lease.
	LeaseDuration time.Duration

	// Enabled is the state of the DHCPv4 service, whether it is enabled or not
	// on the specific interface.
	Enabled bool
}

// validate returns an error in conf if any.
func (c *IPv4Config) validate() (err error) {
	if c == nil {
		return errNilConfig
	} else if !c.Enabled {
		return nil
	}

	var errs []error

	if !c.GatewayIP.Is4() {
		err = newMustErr("gateway ip", "be a valid ipv4", c.GatewayIP)
		errs = append(errs, err)
	}

	if !c.SubnetMask.Is4() {
		err = newMustErr("subnet mask", "be a valid ipv4 cidr mask", c.SubnetMask)
		errs = append(errs, err)
	}

	if !c.RangeStart.Is4() {
		err = newMustErr("range start", "be a valid ipv4", c.RangeStart)
		errs = append(errs, err)
	}

	if !c.RangeEnd.Is4() {
		err = newMustErr("range end", "be a valid ipv4", c.RangeEnd)
		errs = append(errs, err)
	}

	if c.LeaseDuration <= 0 {
		err = newMustErr("icmp timeout", "be positive", c.LeaseDuration)
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

// dhcpInterfaceV4 is a DHCP interface for IPv4 address family.
type dhcpInterfaceV4 struct {
	// common is the common part of any network interface within the DHCP
	// server.
	common *netInterface

	// gateway is the IP address of the network gateway.
	gateway netip.Addr

	// subnet is the network subnet.
	subnet netip.Prefix

	// addrSpace is the IPv4 address space allocated for leasing.
	addrSpace ipRange

	// implicitOpts are the options listed in Appendix A of RFC 2131 and
	// initialized with default values.  It must not have intersections with
	// explicitOpts.
	implicitOpts layers.DHCPOptions

	// explicitOpts are the user-configured options.  It must not have
	// intersections with implicitOpts.
	explicitOpts layers.DHCPOptions
}

// newDHCPInterfaceV4 creates a new DHCP interface for IPv4 address family with
// the given configuration.  It returns an error if the given configuration
// can't be used.
func newDHCPInterfaceV4(
	ctx context.Context,
	l *slog.Logger,
	name string,
	conf *IPv4Config,
) (i *dhcpInterfaceV4, err error) {
	l = l.With(
		keyInterface, name,
		keyFamily, netutil.AddrFamilyIPv4,
	)

	if !conf.Enabled {
		l.DebugContext(ctx, "disabled")

		return nil, nil
	}

	maskLen, _ := net.IPMask(conf.SubnetMask.AsSlice()).Size()
	subnet := netip.PrefixFrom(conf.GatewayIP, maskLen)

	switch {
	case !subnet.Contains(conf.RangeStart):
		return nil, fmt.Errorf("range start %s is not within %s", conf.RangeStart, subnet)
	case !subnet.Contains(conf.RangeEnd):
		return nil, fmt.Errorf("range end %s is not within %s", conf.RangeEnd, subnet)
	}

	addrSpace, err := newIPRange(conf.RangeStart, conf.RangeEnd)
	if err != nil {
		return nil, err
	} else if addrSpace.contains(conf.GatewayIP) {
		return nil, fmt.Errorf("gateway ip %s in the ip range %s", conf.GatewayIP, addrSpace)
	}

	i = &dhcpInterfaceV4{
		gateway:   conf.GatewayIP,
		subnet:    subnet,
		addrSpace: addrSpace,
		common:    newNetInterface(name, l, conf.LeaseDuration),
	}
	i.implicitOpts, i.explicitOpts = conf.options(ctx, l)

	return i, nil
}

// dhcpInterfacesV4 is a slice of network interfaces of IPv4 address family.
type dhcpInterfacesV4 []*dhcpInterfaceV4

// find returns the first network interface within ifaces containing ip.  It
// returns false if there is no such interface.
func (ifaces dhcpInterfacesV4) find(ip netip.Addr) (iface4 *netInterface, ok bool) {
	i := slices.IndexFunc(ifaces, func(iface *dhcpInterfaceV4) (contains bool) {
		return iface.subnet.Contains(ip)
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
// TODO(e.burkov):  DRY with the IPv6 version.
func (c *IPv4Config) options(ctx context.Context, l *slog.Logger) (imp, exp layers.DHCPOptions) {
	// Set default values of host configuration parameters listed in Appendix A
	// of RFC-2131.
	imp = layers.DHCPOptions{
		// Values From Configuration

		layers.NewDHCPOption(layers.DHCPOptSubnetMask, c.SubnetMask.AsSlice()),
		layers.NewDHCPOption(layers.DHCPOptRouter, c.GatewayIP.AsSlice()),

		// IP-Layer Per Host

		// An Internet host that includes embedded gateway code MUST have a
		// configuration switch to disable the gateway function, and this switch
		// MUST default to the non-gateway mode.
		//
		// See https://datatracker.ietf.org/doc/html/rfc1122#section-3.3.5.
		layers.NewDHCPOption(layers.DHCPOptIPForwarding, []byte{0x0}),

		// A host that supports non-local source-routing MUST have a
		// configurable switch to disable forwarding, and this switch MUST
		// default to disabled.
		//
		// See https://datatracker.ietf.org/doc/html/rfc1122#section-3.3.5.
		layers.NewDHCPOption(layers.DHCPOptSourceRouting, []byte{0x0}),

		// Do not set the Policy Filter Option since it only makes sense when
		// the non-local source routing is enabled.

		// The minimum legal value is 576.
		//
		// See https://datatracker.ietf.org/doc/html/rfc2132#section-4.4.
		layers.NewDHCPOption(layers.DHCPOptDatagramMTU, []byte{0x2, 0x40}),

		// Set the current recommended default time to live for the Internet
		// Protocol which is 64.
		//
		// See https://www.iana.org/assignments/ip-parameters/ip-parameters.xhtml#ip-parameters-2.
		layers.NewDHCPOption(layers.DHCPOptDefaultTTL, []byte{0x40}),

		// For example, after the PTMU estimate is decreased, the timeout should
		// be set to 10 minutes; once this timer expires and a larger MTU is
		// attempted, the timeout can be set to a much smaller value.
		//
		// See https://datatracker.ietf.org/doc/html/rfc1191#section-6.6.
		layers.NewDHCPOption(layers.DHCPOptPathMTUAgingTimeout, []byte{0x0, 0x0, 0x2, 0x58}),

		// There is a table describing the MTU values representing all major
		// data-link technologies in use in the Internet so that each set of
		// similar MTUs is associated with a plateau value equal to the lowest
		// MTU in the group.
		//
		// See https://datatracker.ietf.org/doc/html/rfc1191#section-7.
		layers.NewDHCPOption(layers.DHCPOptPathPlateuTableOption, []byte{
			0x0, 0x44,
			0x1, 0x28,
			0x1, 0xFC,
			0x3, 0xEE,
			0x5, 0xD4,
			0x7, 0xD2,
			0x11, 0x0,
			0x1F, 0xE6,
			0x45, 0xFA,
		}),

		// IP-Layer Per Interface

		// Don't set the Interface MTU because client may choose the value on
		// their own since it's listed in the [Host Requirements RFC].  It also
		// seems the values listed there sometimes appear obsolete, see
		// https://github.com/AdguardTeam/AdGuardHome/issues/5281.
		//
		// [Host Requirements RFC]: https://datatracker.ietf.org/doc/html/rfc1122#section-3.3.3.

		// Set the All Subnets Are Local Option to false since commonly the
		// connected hosts aren't expected to be multihomed.
		//
		// See https://datatracker.ietf.org/doc/html/rfc1122#section-3.3.3.
		layers.NewDHCPOption(layers.DHCPOptAllSubsLocal, []byte{0x0}),

		// Set the Perform Mask Discovery Option to false to provide the subnet
		// mask by options only.
		//
		// See https://datatracker.ietf.org/doc/html/rfc1122#section-3.2.2.9.
		layers.NewDHCPOption(layers.DHCPOptMaskDiscovery, []byte{0x0}),

		// A system MUST NOT send an Address Mask Reply unless it is an
		// authoritative agent for address masks.  An authoritative agent may be
		// a host or a gateway, but it MUST be explicitly configured as a
		// address mask agent.
		//
		// See https://datatracker.ietf.org/doc/html/rfc1122#section-3.2.2.9.
		layers.NewDHCPOption(layers.DHCPOptMaskSupplier, []byte{0x0}),

		// Set the Perform Router Discovery Option to true as per Router
		// Discovery Document.
		//
		// See https://datatracker.ietf.org/doc/html/rfc1256#section-5.1.
		layers.NewDHCPOption(layers.DHCPOptRouterDiscovery, []byte{0x1}),

		// The all-routers address is preferred wherever possible.
		//
		// See https://datatracker.ietf.org/doc/html/rfc1256#section-5.1.
		layers.NewDHCPOption(layers.DHCPOptSolicitAddr, netutil.IPv4allrouter()),

		// Don't set the Static Routes Option since it should be set up by
		// system administrator.
		//
		// See https://datatracker.ietf.org/doc/html/rfc1122#section-3.3.1.2.

		// A datagram with the destination address of limited broadcast will be
		// received by every host on the connected physical network but will not
		// be forwarded outside that network.
		//
		// See https://datatracker.ietf.org/doc/html/rfc1122#section-3.2.1.3.
		layers.NewDHCPOption(layers.DHCPOptBroadcastAddr, netutil.IPv4bcast()),

		// Link-Layer Per Interface

		// If the system does not dynamically negotiate use of the trailer
		// protocol on a per-destination basis, the default configuration MUST
		// disable the protocol.
		//
		// See https://datatracker.ietf.org/doc/html/rfc1122#section-2.3.1.
		layers.NewDHCPOption(layers.DHCPOptARPTrailers, []byte{0x0}),

		// For proxy ARP situations, the timeout needs to be on the order of a
		// minute.
		//
		// See https://datatracker.ietf.org/doc/html/rfc1122#section-2.3.2.1.
		layers.NewDHCPOption(layers.DHCPOptARPTimeout, []byte{0x0, 0x0, 0x0, 0x3C}),

		// An Internet host that implements sending both the RFC-894 and the
		// RFC-1042 encapsulations MUST provide a configuration switch to select
		// which is sent, and this switch MUST default to RFC-894.
		//
		// See https://datatracker.ietf.org/doc/html/rfc1122#section-2.3.3.
		layers.NewDHCPOption(layers.DHCPOptEthernetEncap, []byte{0x0}),

		// TCP Per Host

		// A fixed value must be at least big enough for the Internet diameter,
		// i.e., the longest possible path.  A reasonable value is about twice
		// the diameter, to allow for continued Internet growth.
		//
		// See https://datatracker.ietf.org/doc/html/rfc1122#section-3.2.1.7.
		layers.NewDHCPOption(layers.DHCPOptTCPTTL, []byte{0x0, 0x0, 0x0, 0x3C}),

		// The interval MUST be configurable and MUST default to no less than
		// two hours.
		//
		// See https://datatracker.ietf.org/doc/html/rfc1122#section-4.2.3.6.
		layers.NewDHCPOption(layers.DHCPOptTCPKeepAliveInt, []byte{0x0, 0x0, 0x1C, 0x20}),

		// Unfortunately, some misbehaved TCP implementations fail to respond to
		// a probe segment unless it contains data.
		//
		// See https://datatracker.ietf.org/doc/html/rfc1122#section-4.2.3.6.
		layers.NewDHCPOption(layers.DHCPOptTCPKeepAliveGarbage, []byte{0x1}),
	}
	slices.SortFunc(imp, compareV4OptionCodes)

	// Set values for explicitly configured options.
	for _, o := range c.Options {
		i, found := slices.BinarySearchFunc(imp, o, compareV4OptionCodes)
		if found {
			imp = slices.Delete(imp, i, i+1)
		}

		i, found = slices.BinarySearchFunc(exp, o, compareV4OptionCodes)
		if o.Length > 0 {
			exp = slices.Insert(exp, i, o)
		} else if found {
			exp = slices.Delete(exp, i, i+1)
		}
	}

	l.DebugContext(ctx, "options", "implicit", imp, "explicit", exp)

	return imp, exp
}

// compareV4OptionCodes compares option codes of a and b.
func compareV4OptionCodes(a, b layers.DHCPOption) (res int) {
	return int(a.Type) - int(b.Type)
}
