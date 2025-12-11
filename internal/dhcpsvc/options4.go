package dhcpsvc

import (
	"context"
	"encoding/binary"
	"log/slog"
	"net"
	"net/netip"
	"slices"
	"time"

	"github.com/AdguardTeam/golibs/netutil"
	"github.com/google/gopacket/layers"
)

// implicitOptions returns the implicit options for the interface, sorted by
// code.
func (c *IPv4Config) implicitOptions() (opts layers.DHCPOptions) {
	// Set default values of host configuration parameters listed in Appendix A
	// of RFC-2131.
	opts = make(layers.DHCPOptions, 0, 20)

	opts = c.appendConfOptions(opts)
	opts = appendIPPerHostOptions(opts)
	opts = appendIPPerInterfaceOptions(opts)
	opts = appendLinkPerInterfaceOptions(opts)
	opts = appendTCPPerHostOptions(opts)

	slices.SortFunc(opts, compareV4OptionCodes)

	return opts
}

// appendConfOptions appends the DHCPv4 options depending on the configuration
// to orig.
func (c *IPv4Config) appendConfOptions(orig layers.DHCPOptions) (res layers.DHCPOptions) {
	return append(
		orig,
		layers.NewDHCPOption(layers.DHCPOptSubnetMask, c.SubnetMask.AsSlice()),
		layers.NewDHCPOption(layers.DHCPOptRouter, c.GatewayIP.AsSlice()),
	)
}

// appendIPPerHostOptions appends the IP-layer per host DHCPv4 options to orig.
func appendIPPerHostOptions(orig layers.DHCPOptions) (res layers.DHCPOptions) {
	return append(
		orig,
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
	)
}

// appendIPPerInterfaceOptions appends the IP-layer per interface DHCPv4 options
// to orig.
func appendIPPerInterfaceOptions(orig layers.DHCPOptions) (res layers.DHCPOptions) {
	return append(
		orig,

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
	)
}

// appendLinkPerInterfaceOptions appends the link-layer per interface DHCPv4
// options to orig.
func appendLinkPerInterfaceOptions(orig layers.DHCPOptions) (res layers.DHCPOptions) {
	return append(
		orig,

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
	)
}

// appendTCPPerHostOptions appends the TCP per host DHCPv4 options to orig.
func appendTCPPerHostOptions(orig layers.DHCPOptions) (res layers.DHCPOptions) {
	return append(
		orig,

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
	)
}

// options returns the implicit and explicit options for the interface.  The two
// lists are disjoint and the implicit options are initialized with default
// values.  All options within exp which have a nil Data field should be treated
// as instruction to remove those from responses.
//
// TODO(e.burkov):  DRY with the IPv6 version.
func (c *IPv4Config) options(ctx context.Context, l *slog.Logger) (imp, exp layers.DHCPOptions) {
	// Set values of implicit options.
	imp = c.implicitOptions()

	// Set values for explicitly configured options.
	for _, o := range c.Options {
		i, found := slices.BinarySearchFunc(imp, o, compareV4OptionCodes)
		if found {
			imp = slices.Delete(imp, i, i+1)
		}

		i, found = slices.BinarySearchFunc(exp, o, compareV4OptionCodes)
		if found {
			exp[i].Data, exp[i].Length = o.Data, o.Length
		} else {
			exp = slices.Insert(exp, i, o)
		}
	}

	l.DebugContext(ctx, "options", "implicit", imp, "explicit", exp)

	return imp, exp
}

// compareV4OptionCodes compares option codes of a and b.
func compareV4OptionCodes(a, b layers.DHCPOption) (res int) {
	return int(a.Type) - int(b.Type)
}

// updateOptions updates the options of the response in accordance with the
// requested parameters.  req and resp must not be nil.
//
// See https://datatracker.ietf.org/doc/html/rfc2131#section-4.3.1.
func (iface *dhcpInterfaceV4) updateOptions(req, resp *layers.DHCPv4) {
	// If the server recognizes the parameter as a parameter defined in the Host
	// Requirements Document, the server MUST include the default value for that
	// parameter.
	optWithCode := layers.DHCPOption{}
	for _, code := range requestedOptions(req) {
		optWithCode.Type = code
		i, has := slices.BinarySearchFunc(iface.implicitOpts, optWithCode, compareV4OptionCodes)
		if has {
			// The client MAY list the options in order of preference. The DHCP
			// server is not required to return the options in the requested
			// order, but MUST try to insert the requested options in the order
			// requested by the client.
			//
			// See https://datatracker.ietf.org/doc/html/rfc2132#section-9.8.
			resp.Options = append(resp.Options, iface.implicitOpts[i])
		}
	}

	// If the server has been explicitly configured with a default value for the
	// parameter or the parameter has a non-default value on the client's
	// subnet, the server MUST include that value in an appropriate option.
	for _, opt := range iface.explicitOpts {
		if opt.Data != nil {
			resp.Options = append(resp.Options, opt)

			continue
		}

		// Remove options explicitly configured to be removed, in case they are
		// already set.
		resp.Options = slices.DeleteFunc(resp.Options, func(o layers.DHCPOption) (ok bool) {
			return o.Type == opt.Type
		})
	}
}

// appendLeaseTime appends the lease time option to the response.  lease must
// not be nil.
func (iface *dhcpInterfaceV4) appendLeaseTime(resp *layers.DHCPv4, lease *Lease) {
	var dur time.Duration
	if lease.IsStatic {
		dur = iface.common.leaseTTL
	} else {
		dur = lease.Expiry.Sub(iface.clock.Now())
	}

	leaseTimeData := binary.BigEndian.AppendUint32(nil, uint32(dur.Seconds()))

	resp.Options = append(
		resp.Options,
		layers.NewDHCPOption(layers.DHCPOptLeaseTime, leaseTimeData),
	)
}

// msg4Type returns the message type of msg, if it's present within the options.
func msg4Type(msg *layers.DHCPv4) (typ layers.DHCPMsgType, ok bool) {
	for _, opt := range msg.Options {
		if opt.Type == layers.DHCPOptMessageType && len(opt.Data) > 0 {
			return layers.DHCPMsgType(opt.Data[0]), true
		}
	}

	return 0, false
}

// requestedIPv4 returns the IPv4 address, requested by client in the DHCP
// message, if any.
//
// TODO(e.burkov):  DRY with other IP-from-option helpers.
func requestedIPv4(msg *layers.DHCPv4) (ip netip.Addr, ok bool) {
	for _, opt := range msg.Options {
		if opt.Type == layers.DHCPOptRequestIP && len(opt.Data) == net.IPv4len {
			return netip.AddrFromSlice(opt.Data)
		}
	}

	return netip.Addr{}, false
}

// serverID4 returns the server ID of the DHCP message, if any.
func serverID4(msg *layers.DHCPv4) (ip netip.Addr, ok bool) {
	for _, opt := range msg.Options {
		if opt.Type == layers.DHCPOptServerID && len(opt.Data) == net.IPv4len {
			return netip.AddrFromSlice(opt.Data)
		}
	}

	return netip.Addr{}, false
}

// hostname4 returns the hostname from the DHCPv4 message, if any.
func hostname4(msg *layers.DHCPv4) (hostname string) {
	for _, opt := range msg.Options {
		if opt.Type == layers.DHCPOptHostname && len(opt.Data) > 0 {
			return string(opt.Data)
		}
	}

	return ""
}

// requestedOptions returns the list of options requested in DHCPv4 message, if
// any.
//
// TODO(e.burkov):  Use [iter.Seq1].
func requestedOptions(msg *layers.DHCPv4) (opts []layers.DHCPOpt) {
	for _, opt := range msg.Options {
		l := len(opt.Data)
		if opt.Type != layers.DHCPOptParamsRequest || l == 0 {
			continue
		}

		opts = make([]layers.DHCPOpt, 0, l)
		for _, code := range opt.Data {
			opts = append(opts, layers.DHCPOpt(code))
		}

		return opts
	}

	return nil
}
