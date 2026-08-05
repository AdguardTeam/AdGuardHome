package dhcpsvc_test

import (
	"encoding/binary"
	"net"
	"net/netip"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/dhcpsvc"
	"github.com/gopacket/gopacket/layers"
)

// newOptStatusCode creates a top-level DHCPv6 Status Code option.
func newOptStatusCode(tb testing.TB, status layers.DHCPv6StatusCode) (opt layers.DHCPv6Option) {
	tb.Helper()

	const (
		statusCodeLen = 2
	)

	data := make([]byte, 0, statusCodeLen)
	data = binary.BigEndian.AppendUint16(data, uint16(status))

	return layers.NewDHCPv6Option(layers.DHCPv6OptStatusCode, data)
}

// newOptIANA creates a DHCPv6 Identity Association for Non-temporary Address
// (3) option containing an IA Address with the specified IAID and requested IP
// address.  The option will have the T1 and T2 values set to the recommended
// values based on ttl, see the RFC reference in the
// [dhcpsvc.DHCPServer.newDHCPInterfaceV6].  reqIP must be a valid IPv6 address.
func newOptIANA(
	tb testing.TB,
	iaid uint32,
	reqIP netip.Addr,
	ttl time.Duration,
) (opt layers.DHCPv6Option) {
	tb.Helper()

	iana := &dhcpsvc.IANAOption{
		ID: iaid,
		Nested: []dhcpsvc.IAAddrOption{{
			PreferredLifetime: ttl,
			ValidLifetime:     ttl,
			Addr:              reqIP,
		}},
		T1: ttl / 2,
		T2: ttl * 4 / 5,
	}

	return iana.Encode()
}

// newOptIANAStatus creates a DHCPv6 IA_NA (3) option carrying only a nested
// Status Code option.  If status is [layers.DHCPv6StatusCodeSuccess], the
// returned option will not contain a nested Status Code option, as per RFC 8415
// section 21.13.
func newOptIANAStatus(
	tb testing.TB,
	iaid uint32,
	status layers.DHCPv6StatusCode,
) (opt layers.DHCPv6Option) {
	tb.Helper()

	const (
		// statusOptLen is the length of the nested status code option:
		//   code (2) + length (2) + status (2) = 6 bytes.
		statusOptLen = 6

		// iaNAMinLen is the minimum length of the IA_NA option:
		//   IAID (4) + T1 (4) + T2 (4) = 12 bytes.
		iaNAMinLen = 12

		// iaNAStatusLen is the length of the IA_NA option with a nested status
		// code option.
		iaNAStatusLen = iaNAMinLen + statusOptLen
	)

	data := make([]byte, 0, iaNAStatusLen)

	data = binary.BigEndian.AppendUint32(data, iaid)
	// T1 and T2 are set to zero.
	data = binary.BigEndian.AppendUint32(data, 0)
	data = binary.BigEndian.AppendUint32(data, 0)

	if status != layers.DHCPv6StatusCodeSuccess {
		// Nested Status Code option.
		data = binary.BigEndian.AppendUint16(data, uint16(layers.DHCPv6OptStatusCode))

		// The length of the Status Code option data is 2 bytes.
		data = binary.BigEndian.AppendUint16(data, 2)
		data = binary.BigEndian.AppendUint16(data, uint16(status))
	}

	return layers.NewDHCPv6Option(layers.DHCPv6OptIANA, data)
}

// newOptPreference creates a DHCPv6 Preference (7) option with the specified
// preference value.
func newOptPreference(tb testing.TB, pref uint8) (opt layers.DHCPv6Option) {
	tb.Helper()

	return layers.NewDHCPv6Option(layers.DHCPv6OptPreference, []byte{pref})
}

// newOptSolMaxRT creates a DHCPv6 Solicit Message Maximum Retransmission Time
// (80) option with the specified maxRT value.
func newOptSolMaxRT(tb testing.TB, maxRT time.Duration) (opt layers.DHCPv6Option) {
	tb.Helper()

	return layers.NewDHCPv6Option(
		layers.DHCPv6OptSolMaxRt,
		binary.BigEndian.AppendUint32(nil, uint32(maxRT.Seconds())),
	)
}

// newOptClientDUID creates a DHCPv6 Client Identifier (1) option containing a
// DUID-LL made of cliHWAddr.
func newOptClientDUID(tb testing.TB, cliHWAddr net.HardwareAddr) (opt layers.DHCPv6Option) {
	tb.Helper()

	return newOptDUIDLL(tb, layers.DHCPv6OptClientID, cliHWAddr)
}

// newOptServerID creates a DHCPv6 Server Identifier (2) option containing a
// DUID-LL made of srvHWAddr.
func newOptServerDUID(tb testing.TB, srvHWAddr net.HardwareAddr) (opt layers.DHCPv6Option) {
	tb.Helper()

	return newOptDUIDLL(tb, layers.DHCPv6OptServerID, srvHWAddr)
}

// newOptDUIDLL creates a DHCPv6 option with the specified code containing a
// DUID-LL made of hwAddr and Ethernet hardware type.
func newOptDUIDLL(
	tb testing.TB,
	code layers.DHCPv6Opt,
	hwAddr net.HardwareAddr,
) (opt layers.DHCPv6Option) {
	tb.Helper()

	duid := &layers.DHCPv6DUID{
		Type:             layers.DHCPv6DUIDTypeLL,
		HardwareType:     binary.BigEndian.AppendUint16(nil, uint16(layers.LinkTypeEthernet)),
		LinkLayerAddress: hwAddr,
	}

	return layers.NewDHCPv6Option(code, duid.Encode())
}
