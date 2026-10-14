//go:build darwin

package dhcpsvc

import (
	"context"
	"log/slog"
	"net"
	"net/netip"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"golang.org/x/net/icmp"
)

// dialICMP4 establishes an IPv4 connection to ip.  It prefers the unprivileged
// datagram socket and falls back to the raw one.  The raw socket requires root
// privileges, which is the same expectation the DHCP server itself has.
func dialICMP4(
	ctx context.Context,
	l *slog.Logger,
	ip netip.Addr,
) (conn net.PacketConn, addr net.Addr, err error) {
	pc, udpErr := icmp.ListenPacket("udp4", "0.0.0.0")
	if udpErr == nil {
		l.DebugContext(ctx, "listening udp4")

		return pc, net.UDPAddrFromAddrPort(netip.AddrPortFrom(ip, 0)), nil
	}

	l.DebugContext(ctx, "failed to listen udp4", slogutil.KeyError, udpErr)

	conn, rawErr := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if rawErr != nil {
		return nil, nil, errors.Join(udpErr, rawErr)
	}

	l.DebugContext(ctx, "listening raw ip4:icmp")

	return conn, &net.IPAddr{IP: ip.AsSlice()}, nil
}
