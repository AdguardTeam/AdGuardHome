//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris
// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

package dhcpd

import (
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/insomniacslk/dhcp/dhcpv4"
)

// The aliases for DHCP option types available for explicit declaration.
const (
	typHex  = "hex"
	typIP   = "ip"
	typIPs  = "ips"
	typText = "text"
	typDel  = "del"
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
	// Otherwise, the clients will receive weird data that looks like four
	// IPv4 addresses.
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
	var ip net.IP
	for i, ipStr := range strings.Split(s, ",") {
		// See notes in the ipDHCPOptionParserHandler.
		if ip, err = netutil.ParseIPv4(ipStr); err != nil {
			return nil, fmt.Errorf("parsing ip at index %d: %w", i, err)
		}

		ips = append(ips, ip)
	}

	return ips, nil
}

// parseDHCPOptionText parses a DHCP option as a simple UTF-8 encoded
// text.
func parseDHCPOptionText(s string) (val dhcpv4.OptionValue) {
	return dhcpv4.OptionGeneric{Data: []byte(s)}
}

// parseDHCPOptionVal parses a DHCP option value considering typ.  For the del
// option the value string is ignored.  The examples of possible value pairs:
//
//   - hex  736f636b733a2f2f70726f78792e6578616d706c652e6f7267
//   - ip   192.168.1.1
//   - ips  192.168.1.1,192.168.1.2
//   - text http://192.168.1.1/wpad.dat
//   - del
func parseDHCPOptionVal(typ, valStr string) (val dhcpv4.OptionValue, err error) {
	switch typ {
	case typHex:
		val, err = parseDHCPOptionHex(valStr)
	case typIP:
		val, err = parseDHCPOptionIP(valStr)
	case typIPs:
		val, err = parseDHCPOptionIPs(valStr)
	case typText:
		val = parseDHCPOptionText(valStr)
	case typDel:
		val = dhcpv4.OptionGeneric{Data: nil}
	default:
		err = fmt.Errorf("unknown option type %q", typ)
	}

	return val, err
}

// parseDHCPOption parses an option.  See the documentation of
// parseDHCPOptionVal for more info.
func parseDHCPOption(s string) (opt dhcpv4.Option, err error) {
	defer func() { err = errors.Annotate(err, "invalid option string %q: %w", s) }()

	s = strings.TrimSpace(s)
	parts := strings.SplitN(s, " ", 3)

	var valStr string
	if pl := len(parts); pl < 3 {
		if pl < 2 || parts[1] != typDel {
			return opt, errors.Error("bad option format")
		}
	} else {
		valStr = parts[2]
	}

	var code64 uint64
	code64, err = strconv.ParseUint(parts[0], 10, 8)
	if err != nil {
		return opt, fmt.Errorf("parsing option code: %w", err)
	}

	val, err := parseDHCPOptionVal(parts[1], valStr)
	if err != nil {
		// Don't wrap an error since it's informative enough as is and there
		// also the deferred annotation.
		return opt, err
	}

	return dhcpv4.Option{
		Code:  dhcpv4.GenericOptionCode(code64),
		Value: val,
	}, nil
}

// prepareOptions builds the set of DHCP options according to host requirements
// document and values from conf.
func prepareOptions(conf V4ServerConf) (opts dhcpv4.Options) {
	// Set default values for host configuration parameters listed in Appendix
	// A of RFC-2131.  Those parameters, if requested by client, should be
	// returned with values defined by Host Requirements Document.
	//
	// See https://datatracker.ietf.org/doc/html/rfc2131#appendix-A.
	//
	// See also https://datatracker.ietf.org/doc/html/rfc1122,
	// https://datatracker.ietf.org/doc/html/rfc1123, and
	// https://datatracker.ietf.org/doc/html/rfc2132.
	opts = dhcpv4.OptionsFromList(
		// IP-Layer Per Host

		dhcpv4.OptGeneric(dhcpv4.OptionNonLocalSourceRouting, []byte{0}),

		// Set the current recommended default time to live for the
		// Internet Protocol which is 64, see
		// https://datatracker.ietf.org/doc/html/rfc1700.
		dhcpv4.OptGeneric(dhcpv4.OptionDefaultIPTTL, []byte{0x40}),

		dhcpv4.OptGeneric(dhcpv4.OptionPerformMaskDiscovery, []byte{0}),
		dhcpv4.OptGeneric(dhcpv4.OptionMaskSupplier, []byte{0}),
		dhcpv4.OptGeneric(dhcpv4.OptionPerformRouterDiscovery, []byte{1}),
		// The all-routers address is preferred wherever possible, see
		// https://datatracker.ietf.org/doc/html/rfc1256#section-5.1.
		dhcpv4.Option{
			Code:  dhcpv4.OptionRouterSolicitationAddress,
			Value: dhcpv4.IP(netutil.IPv4allrouter()),
		},
		dhcpv4.OptBroadcastAddress(netutil.IPv4bcast()),

		// Link-Layer Per Interface

		dhcpv4.OptGeneric(dhcpv4.OptionTrailerEncapsulation, []byte{0}),
		dhcpv4.OptGeneric(dhcpv4.OptionEthernetEncapsulation, []byte{0}),

		// TCP Per Host

		dhcpv4.Option{
			Code:  dhcpv4.OptionTCPKeepaliveInterval,
			Value: dhcpv4.Duration(0),
		},
		dhcpv4.OptGeneric(dhcpv4.OptionTCPKeepaliveGarbage, []byte{0}),

		// Values From Configuration

		dhcpv4.OptRouter(conf.subnet.IP),
		dhcpv4.OptSubnetMask(conf.subnet.Mask),
	)

	// Set values for explicitly configured options.
	for i, o := range conf.Options {
		opt, err := parseDHCPOption(o)
		if err != nil {
			log.Error("dhcpv4: bad option string at index %d: %s", i, err)

			continue
		}

		opts.Update(opt)
	}

	return opts
}
