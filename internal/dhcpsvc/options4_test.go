package dhcpsvc_test

import (
	"encoding/binary"
	"net/netip"
	"testing"
	"time"

	"github.com/google/gopacket/layers"
)

// newOptHostname creates a DHCP hostname (12) option.
func newOptHostname(tb testing.TB, hostname string) (opt layers.DHCPOption) {
	tb.Helper()

	return layers.NewDHCPOption(layers.DHCPOptHostname, []byte(hostname))
}

// newOptLeaseTime creates a DHCP lease time (51) option.
func newOptLeaseTime(tb testing.TB, dur time.Duration) (opt layers.DHCPOption) {
	tb.Helper()

	secs := uint32(dur.Seconds())
	var buf [4]byte
	binary.BigEndian.PutUint32(buf[:], secs)

	return layers.NewDHCPOption(layers.DHCPOptLeaseTime, buf[:])
}

// newOptMessageType creates a DHCP message type (53) option.
func newOptMessageType(tb testing.TB, msgType layers.DHCPMsgType) (opt layers.DHCPOption) {
	tb.Helper()

	return layers.NewDHCPOption(layers.DHCPOptMessageType, []byte{byte(msgType)})
}

// newOptServerID creates a DHCP server identifier (54) option.
func newOptServerID(tb testing.TB, serverIP netip.Addr) (opt layers.DHCPOption) {
	tb.Helper()

	return layers.NewDHCPOption(layers.DHCPOptServerID, serverIP.AsSlice())
}
