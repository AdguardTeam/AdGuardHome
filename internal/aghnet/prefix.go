package aghnet

import (
	"bufio"
	"bytes"
	"fmt"
	"math"
	"net/netip"
	"strconv"
	"strings"
)

// IPv6AddrState describes the current state of one IPv6 interface address.
type IPv6AddrState struct {
	Addr                 netip.Addr
	Prefix               netip.Prefix
	PreferredLifetimeSec uint32
	ValidLifetimeSec     uint32
	Temporary            bool
	Tentative            bool
}

// parseIfconfigIPv6Addrs parses IPv6 interface state lines from ifconfig.
func parseIfconfigIPv6Addrs(out []byte) (states []IPv6AddrState, err error) {
	s := bufio.NewScanner(bytes.NewReader(out))
	for s.Scan() {
		fields := strings.Fields(s.Text())
		if len(fields) < 2 || !strings.EqualFold(fields[0], "inet6") {
			continue
		}

		state, parseErr := parseIfconfigIPv6Addr(fields)
		if parseErr != nil {
			return nil, parseErr
		}

		states = append(states, state)
	}

	return states, s.Err()
}

// parseIfconfigIPv6Addr parses one `ifconfig` IPv6 address line.
func parseIfconfigIPv6Addr(fields []string) (state IPv6AddrState, err error) {
	addr, err := parseIPv6AddrNoZone(fields[1])
	if err != nil {
		return IPv6AddrState{}, fmt.Errorf("parsing addr %q: %w", fields[1], err)
	}

	preferred := uint32(math.MaxUint32)
	valid := uint32(math.MaxUint32)
	prefixBits := -1

	for i := 2; i < len(fields); i++ {
		i, err = parseIfconfigIPv6AddrField(fields, i, &state, &preferred, &valid, &prefixBits)
		if err != nil {
			return IPv6AddrState{}, err
		}
	}

	if prefixBits < 0 {
		return IPv6AddrState{}, fmt.Errorf("missing prefixlen in %q", strings.Join(fields, " "))
	}

	return IPv6AddrState{
		Addr:                 addr,
		Prefix:               netip.PrefixFrom(addr, prefixBits).Masked(),
		PreferredLifetimeSec: preferred,
		ValidLifetimeSec:     valid,
		Temporary:            state.Temporary,
		Tentative:            state.Tentative,
	}, nil
}

// parseIfconfigIPv6AddrField parses one token from an ifconfig IPv6 address
// line.
func parseIfconfigIPv6AddrField(
	fields []string,
	i int,
	state *IPv6AddrState,
	preferred, valid *uint32,
	prefixBits *int,
) (next int, err error) {
	switch strings.ToLower(fields[i]) {
	case "prefixlen":
		return parseIfconfigIPv6AddrInt(fields, i, "prefixlen", func(v int) {
			*prefixBits = v
		})
	case "pltime":
		return parseIfconfigIPv6AddrLifetime(fields, i, "pltime", preferred)
	case "vltime":
		return parseIfconfigIPv6AddrLifetime(fields, i, "vltime", valid)
	case "temporary":
		state.Temporary = true
	case "tentative":
		state.Tentative = true
	}

	return i, nil
}

// parseIfconfigIPv6AddrInt parses one int token from an ifconfig IPv6 address
// line.
func parseIfconfigIPv6AddrInt(
	fields []string,
	i int,
	name string,
	set func(int),
) (next int, err error) {
	i++
	if i >= len(fields) {
		return i, fmt.Errorf("missing %s value in %q", name, strings.Join(fields, " "))
	}

	v, err := strconv.Atoi(fields[i])
	if err != nil {
		return i, fmt.Errorf("parsing %s %q: %w", name, fields[i], err)
	}

	set(v)

	return i, nil
}

// parseIfconfigIPv6AddrLifetime parses one IPv6 lifetime token from an
// ifconfig IPv6 address line.
func parseIfconfigIPv6AddrLifetime(
	fields []string,
	i int,
	name string,
	lifetime *uint32,
) (next int, err error) {
	i++
	if i >= len(fields) {
		return i, fmt.Errorf("missing %s value in %q", name, strings.Join(fields, " "))
	}

	*lifetime, err = parseIPv6Lifetime(fields[i])
	if err != nil {
		return i, fmt.Errorf("parsing %s %q: %w", name, fields[i], err)
	}

	return i, nil
}

// parseIPv6Lifetime parses an IPv6 lifetime token from command output.
func parseIPv6Lifetime(s string) (sec uint32, err error) {
	switch strings.ToLower(s) {
	case "forever", "infinity", "infinite", "infty":
		return math.MaxUint32, nil
	default:
		v, parseErr := strconv.ParseUint(s, 10, 32)
		if parseErr != nil {
			return 0, parseErr
		}

		return uint32(v), nil
	}
}

// parseIPv6AddrNoZone parses an IPv6 address and removes the interface zone.
func parseIPv6AddrNoZone(s string) (addr netip.Addr, err error) {
	addr, err = netip.ParseAddr(s)
	if err != nil {
		return netip.Addr{}, err
	}

	return addr.WithZone(""), nil
}
