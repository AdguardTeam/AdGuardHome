package dhcpd

import (
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"strings"

	"github.com/AdguardTeam/AdGuardHome/internal/agherr"
)

// hexDHCPOptionParserHandler parses a DHCP option as a hex-encoded string.
// For example:
//
//   252 hex 736f636b733a2f2f70726f78792e6578616d706c652e6f7267
//
func hexDHCPOptionParserHandler(s string) (data []byte, err error) {
	data, err = hex.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("decoding hex: %w", err)
	}

	return data, nil
}

// ipDHCPOptionParserHandler parses a DHCP option as a single IP address.
// For example:
//
//   6 ip 192.168.1.1
//
func ipDHCPOptionParserHandler(s string) (data []byte, err error) {
	ip := net.ParseIP(s)
	if ip == nil {
		return nil, agherr.Error("invalid ip")
	}

	// Most DHCP options require IPv4, so do not put the 16-byte
	// version if we can.  Otherwise, the clients will receive weird
	// data that looks like four IPv4 addresses.
	//
	// See https://github.com/AdguardTeam/AdGuardHome/issues/2688.
	if ip4 := ip.To4(); ip4 != nil {
		data = ip4
	} else {
		data = ip
	}

	return data, nil
}

// textDHCPOptionParserHandler parses a DHCP option as a simple UTF-8 encoded
// text.  For example:
//
//   252 text http://192.168.1.1/wpad.dat
//
func ipsDHCPOptionParserHandler(s string) (data []byte, err error) {
	ipStrs := strings.Split(s, ",")
	for i, ipStr := range ipStrs {
		var ipData []byte
		ipData, err = ipDHCPOptionParserHandler(ipStr)
		if err != nil {
			return nil, fmt.Errorf("parsing ip at index %d: %w", i, err)
		}

		data = append(data, ipData...)
	}

	return data, nil
}

// ipsDHCPOptionParserHandler parses a DHCP option as a comma-separates list of
// IP addresses.  For example:
//
//   6 ips 192.168.1.1,192.168.1.2
//
func textDHCPOptionParserHandler(s string) (data []byte, err error) {
	return []byte(s), nil
}

// dhcpOptionParserHandler is a parser for a single dhcp option type.
type dhcpOptionParserHandler func(s string) (data []byte, err error)

// dhcpOptionParser parses DHCP options.
type dhcpOptionParser struct {
	handlers map[string]dhcpOptionParserHandler
}

// newDHCPOptionParser returns a new dhcpOptionParser.
func newDHCPOptionParser() (p *dhcpOptionParser) {
	return &dhcpOptionParser{
		handlers: map[string]dhcpOptionParserHandler{
			"hex":  hexDHCPOptionParserHandler,
			"ip":   ipDHCPOptionParserHandler,
			"ips":  ipsDHCPOptionParserHandler,
			"text": textDHCPOptionParserHandler,
		},
	}
}

// parse parses an option.  See the handlers' documentation for more info.
func (p *dhcpOptionParser) parse(s string) (code uint8, data []byte, err error) {
	defer agherr.Annotate("invalid option string %q: %w", &err, s)

	s = strings.TrimSpace(s)
	parts := strings.SplitN(s, " ", 3)
	if len(parts) < 3 {
		return 0, nil, agherr.Error("need at least three fields")
	}

	codeStr := parts[0]
	typ := parts[1]
	val := parts[2]

	var code64 uint64
	code64, err = strconv.ParseUint(codeStr, 10, 8)
	if err != nil {
		return 0, nil, fmt.Errorf("parsing option code: %w", err)
	}

	code = uint8(code64)

	h, ok := p.handlers[typ]
	if !ok {
		return 0, nil, fmt.Errorf("unknown option type %q", typ)
	}

	data, err = h(val)
	if err != nil {
		return 0, nil, err
	}

	return uint8(code), data, nil
}
