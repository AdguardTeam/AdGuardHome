//go:build darwin || freebsd || linux || openbsd

package dhcpd

import (
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/insomniacslk/dhcp/dhcpv4"
)

// The aliases for DHCP option types available for explicit declaration.
//
// TODO(e.burkov):  Add an option for classless routes.
const (
	typDel  = "del"
	typBool = "bool"
	typDur  = "dur"
	typHex  = "hex"
	typIP   = "ip"
	typIPs  = "ips"
	typText = "text"
	typU8   = "u8"
	typU16  = "u16"
)

// parseDHCPOptionHex parses a DHCP option as a hex-encoded string.
func parseDHCPOptionHex(s string) (val dhcpv4.OptionValue, err error) {
	var data []byte
	data, err = hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("decoding hex: %w", err)
	}

	return dhcpv4.OptionGeneric{Data: data}, nil
}

// parseDHCPOptionIP parses a DHCP option as a single IP address.
func parseDHCPOptionIP(s string) (val dhcpv4.OptionValue, err error) {
	var ip net.IP
	// All DHCPv4 options require IPv4, so don't put the 16-byte version.
	// Otherwise, the clients will receive weird data that looks like four IPv4
	// addresses.
	//
	// See https://github.com/AdguardTeam/AdGuardHome/issues/2688.
	if ip, err = netutil.ParseIPv4(s); err != nil {
		return nil, err
	}

	return dhcpv4.IP(ip), nil
}

// parseDHCPOptionIPs parses a DHCP option as a comma-separates list of IP
// addresses.
func parseDHCPOptionIPs(s string) (val dhcpv4.OptionValue, err error) {
	var ips dhcpv4.IPs
	var ip dhcpv4.OptionValue
	for i, ipStr := range strings.Split(s, ",") {
		ip, err = parseDHCPOptionIP(ipStr)
		if err != nil {
			return nil, fmt.Errorf("parsing ip at index %d: %w", i, err)
		}

		ips = append(ips, net.IP(ip.(dhcpv4.IP)))
	}

	return ips, nil
}

// parseDHCPOptionDur parses a DHCP option as a duration in a human-readable
// form.
func parseDHCPOptionDur(s string) (val dhcpv4.OptionValue, err error) {
	var v timeutil.Duration
	err = v.UnmarshalText([]byte(s))
	if err != nil {
		return nil, fmt.Errorf("decoding dur: %w", err)
	}

	return dhcpv4.Duration(v), nil
}

// parseDHCPOptionUint parses a DHCP option as an unsigned integer.  bitSize is
// expected to be 8 or 16.
func parseDHCPOptionUint(s string, bitSize int) (val dhcpv4.OptionValue, err error) {
	var v uint64
	v, err = strconv.ParseUint(s, 10, bitSize)
	if err != nil {
		return nil, fmt.Errorf("decoding u%d: %w", bitSize, err)
	}

	switch bitSize {
	case 8:
		return dhcpv4.OptionGeneric{Data: []byte{uint8(v)}}, nil
	case 16:
		return dhcpv4.Uint16(v), nil
	default:
		return nil, fmt.Errorf("unsupported size of integer %d", bitSize)
	}
}

// parseDHCPOptionBool parses a DHCP option as a boolean value.  See
// [strconv.ParseBool] for available values.
func parseDHCPOptionBool(s string) (val dhcpv4.OptionValue, err error) {
	var v bool
	v, err = strconv.ParseBool(s)
	if err != nil {
		return nil, fmt.Errorf("decoding bool: %w", err)
	}

	rawVal := [1]byte{}
	if v {
		rawVal[0] = 1
	}

	return dhcpv4.OptionGeneric{Data: rawVal[:]}, nil
}

// parseDHCPOptionVal parses a DHCP option value considering typ.
func parseDHCPOptionVal(typ, valStr string) (val dhcpv4.OptionValue, err error) {
	switch typ {
	case typBool:
		val, err = parseDHCPOptionBool(valStr)
	case typDel:
		val = dhcpv4.OptionGeneric{Data: nil}
	case typDur:
		val, err = parseDHCPOptionDur(valStr)
	case typHex:
		val, err = parseDHCPOptionHex(valStr)
	case typIP:
		val, err = parseDHCPOptionIP(valStr)
	case typIPs:
		val, err = parseDHCPOptionIPs(valStr)
	case typText:
		val = dhcpv4.String(valStr)
	case typU8:
		val, err = parseDHCPOptionUint(valStr, 8)
	case typU16:
		val, err = parseDHCPOptionUint(valStr, 16)
	default:
		err = fmt.Errorf("unknown option type %q", typ)
	}

	return val, err
}

// parseDHCPOption parses an option.  For the del option value is ignored.  The
// examples of possible option strings:
//
//   - 1  bool true
//   - 2  del
//   - 3  dur  2h5s
//   - 4  hex  736f636b733a2f2f70726f78792e6578616d706c652e6f7267
//   - 5  ip   192.168.1.1
//   - 6  ips  192.168.1.1,192.168.1.2
//   - 7  text http://192.168.1.1/wpad.dat
//   - 8  u8   255
//   - 9  u16  65535
func parseDHCPOption(s string) (code dhcpv4.OptionCode, val dhcpv4.OptionValue, err error) {
	defer func() { err = errors.Annotate(err, "invalid option string %q: %w", s) }()

	s = strings.TrimSpace(s)
	parts := strings.SplitN(s, " ", 3)

	var valStr string
	if pl := len(parts); pl < 3 {
		if pl < 2 || parts[1] != typDel {
			return nil, nil, errors.Error("bad option format")
		}
	} else {
		valStr = parts[2]
	}

	var code64 uint64
	code64, err = strconv.ParseUint(parts[0], 10, 8)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing option code: %w", err)
	}

	val, err = parseDHCPOptionVal(parts[1], valStr)
	if err != nil {
		// Don't wrap an error since it's informative enough as is and there
		// also the deferred annotation.
		return nil, nil, err
	}

	return dhcpv4.GenericOptionCode(code64), val, nil
}

// prepareOptions builds the set of DHCP options according to host requirements
// document and values from conf.
func (s *v4Server) prepareOptions() {
	// Set default values of host configuration parameters listed in Appendix A
	// of RFC-2131.
	s.implicitOpts = dhcpv4.OptionsFromList(
		// IP-Layer Per Host

		// An Internet host that includes embedded gateway code MUST have a
		// configuration switch to disable the gateway function, and this switch
		// MUST default to the non-gateway mode.
		//
		// See https://datatracker.ietf.org/doc/html/rfc1122#section-3.3.5.
		dhcpv4.OptGeneric(dhcpv4.OptionIPForwarding, []byte{0x0}),

		// A host that supports non-local source-routing MUST have a
		// configurable switch to disable forwarding, and this switch MUST
		// default to disabled.
		//
		// See https://datatracker.ietf.org/doc/html/rfc1122#section-3.3.5.
		dhcpv4.OptGeneric(dhcpv4.OptionNonLocalSourceRouting, []byte{0x0}),

		// Do not set the Policy Filter Option since it only makes sense when
		// the non-local source routing is enabled.

		// The minimum legal value is 576.
		//
		// See https://datatracker.ietf.org/doc/html/rfc2132#section-4.4.
		dhcpv4.Option{
			Code:  dhcpv4.OptionMaximumDatagramAssemblySize,
			Value: dhcpv4.Uint16(576),
		},

		// Set the current recommended default time to live for the Internet
		// Protocol which is 64.
		//
		// See https://www.iana.org/assignments/ip-parameters/ip-parameters.xhtml#ip-parameters-2.
		dhcpv4.OptGeneric(dhcpv4.OptionDefaultIPTTL, []byte{0x40}),

		// For example, after the PTMU estimate is decreased, the timeout should
		// be set to 10 minutes; once this timer expires and a larger MTU is
		// attempted, the timeout can be set to a much smaller value.
		//
		// See https://datatracker.ietf.org/doc/html/rfc1191#section-6.6.
		dhcpv4.Option{
			Code:  dhcpv4.OptionPathMTUAgingTimeout,
			Value: dhcpv4.Duration(10 * time.Minute),
		},

		// There is a table describing the MTU values representing all major
		// data-link technologies in use in the Internet so that each set of
		// similar MTUs is associated with a plateau value equal to the lowest
		// MTU in the group.
		//
		// See https://datatracker.ietf.org/doc/html/rfc1191#section-7.
		dhcpv4.OptGeneric(dhcpv4.OptionPathMTUPlateauTable, []byte{
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
		dhcpv4.OptGeneric(dhcpv4.OptionAllSubnetsAreLocal, []byte{0x00}),

		// Set the Perform Mask Discovery Option to false to provide the subnet
		// mask by options only.
		//
		// See https://datatracker.ietf.org/doc/html/rfc1122#section-3.2.2.9.
		dhcpv4.OptGeneric(dhcpv4.OptionPerformMaskDiscovery, []byte{0x00}),

		// A system MUST NOT send an Address Mask Reply unless it is an
		// authoritative agent for address masks.  An authoritative agent may be
		// a host or a gateway, but it MUST be explicitly configured as a
		// address mask agent.
		//
		// See https://datatracker.ietf.org/doc/html/rfc1122#section-3.2.2.9.
		dhcpv4.OptGeneric(dhcpv4.OptionMaskSupplier, []byte{0x00}),

		// Set the Perform Router Discovery Option to true as per Router
		// Discovery Document.
		//
		// See https://datatracker.ietf.org/doc/html/rfc1256#section-5.1.
		dhcpv4.OptGeneric(dhcpv4.OptionPerformRouterDiscovery, []byte{0x01}),

		// The all-routers address is preferred wherever possible.
		//
		// See https://datatracker.ietf.org/doc/html/rfc1256#section-5.1.
		dhcpv4.Option{
			Code:  dhcpv4.OptionRouterSolicitationAddress,
			Value: dhcpv4.IP(netutil.IPv4allrouter()),
		},

		// Don't set the Static Routes Option since it should be set up by
		// system administrator.
		//
		// See https://datatracker.ietf.org/doc/html/rfc1122#section-3.3.1.2.

		// A datagram with the destination address of limited broadcast will be
		// received by every host on the connected physical network but will not
		// be forwarded outside that network.
		//
		// See https://datatracker.ietf.org/doc/html/rfc1122#section-3.2.1.3.
		dhcpv4.OptBroadcastAddress(netutil.IPv4bcast()),

		// Link-Layer Per Interface

		// If the system does not dynamically negotiate use of the trailer
		// protocol on a per-destination basis, the default configuration MUST
		// disable the protocol.
		//
		// See https://datatracker.ietf.org/doc/html/rfc1122#section-2.3.1.
		dhcpv4.OptGeneric(dhcpv4.OptionTrailerEncapsulation, []byte{0x00}),

		// For proxy ARP situations, the timeout needs to be on the order of a
		// minute.
		//
		// See https://datatracker.ietf.org/doc/html/rfc1122#section-2.3.2.1.
		dhcpv4.Option{
			Code:  dhcpv4.OptionArpCacheTimeout,
			Value: dhcpv4.Duration(time.Minute),
		},

		// An Internet host that implements sending both the RFC-894 and the
		// RFC-1042 encapsulations MUST provide a configuration switch to select
		// which is sent, and this switch MUST default to RFC-894.
		//
		// See https://datatracker.ietf.org/doc/html/rfc1122#section-2.3.3.
		dhcpv4.OptGeneric(dhcpv4.OptionEthernetEncapsulation, []byte{0x00}),

		// TCP Per Host

		// A fixed value must be at least big enough for the Internet diameter,
		// i.e., the longest possible path.  A reasonable value is about twice
		// the diameter, to allow for continued Internet growth.
		//
		// See https://datatracker.ietf.org/doc/html/rfc1122#section-3.2.1.7.
		dhcpv4.Option{
			Code:  dhcpv4.OptionDefaulTCPTTL,
			Value: dhcpv4.Duration(60 * time.Second),
		},

		// The interval MUST be configurable and MUST default to no less than
		// two hours.
		//
		// See https://datatracker.ietf.org/doc/html/rfc1122#section-4.2.3.6.
		dhcpv4.Option{
			Code:  dhcpv4.OptionTCPKeepaliveInterval,
			Value: dhcpv4.Duration(2 * time.Hour),
		},

		// Unfortunately, some misbehaved TCP implementations fail to respond to
		// a probe segment unless it contains data.
		//
		// See https://datatracker.ietf.org/doc/html/rfc1122#section-4.2.3.6.
		dhcpv4.OptGeneric(dhcpv4.OptionTCPKeepaliveGarbage, []byte{0x01}),

		// Values From Configuration
		dhcpv4.OptRouter(s.conf.GatewayIP.AsSlice()),

		dhcpv4.OptSubnetMask(s.conf.SubnetMask.AsSlice()),
	)

	// Set values for explicitly configured options.
	s.explicitOpts = dhcpv4.Options{}
	for i, o := range s.conf.Options {
		code, val, err := parseDHCPOption(o)
		if err != nil {
			log.Error("dhcpv4: bad option string at index %d: %s", i, err)

			continue
		}

		s.explicitOpts.Update(dhcpv4.Option{Code: code, Value: val})
		// Remove those from the implicit options.
		delete(s.implicitOpts, code.Code())
	}

	log.Debug("dhcpv4: implicit options:\n%s", s.implicitOpts.Summary(nil))
	log.Debug("dhcpv4: explicit options:\n%s", s.explicitOpts.Summary(nil))

	if len(s.explicitOpts) == 0 {
		s.explicitOpts = nil
	}
}
