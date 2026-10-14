//go:build !darwin

package dhcpsvc

import (
	"context"
	"log/slog"
	"net"
	"net/netip"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/gopacket/gopacket"
	"github.com/gopacket/gopacket/layers"
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

	// Unlike on darwin, where [icmp] enables the IP_STRIPHDR socket option,
	// reads from a raw IPv4 socket include the IP header.
	conn = &stripIPHeaderConn{PacketConn: conn}

	return conn, &net.IPAddr{IP: ip.AsSlice()}, nil
}

// stripIPHeaderConn is a [net.PacketConn] wrapper stripping the IPv4 header
// from the packets read from a raw socket.
type stripIPHeaderConn struct {
	// PacketConn is the underlying raw socket.
	net.PacketConn
}

// type check
var _ net.PacketConn = (*stripIPHeaderConn)(nil)

// ReadFrom implements [net.PacketConn] for *stripIPHeaderConn.
func (c *stripIPHeaderConn) ReadFrom(b []byte) (n int, addr net.Addr, err error) {
	n, addr, err = c.PacketConn.ReadFrom(b)
	if err != nil || n < 1 {
		return n, addr, err
	}

	var ip4 layers.IPv4
	if decodeErr := ip4.DecodeFromBytes(b[:n], gopacket.NilDecodeFeedback); decodeErr != nil {
		// The packet is malformed, drop its contents.
		return 0, addr, nil
	}

	// The contents of a decoded IPv4 layer is the header itself.
	hdrLen := len(ip4.Contents)
	if n <= hdrLen {
		// The packet is malformed, drop its contents.
		return 0, addr, nil
	}

	copy(b, b[hdrLen:n])

	return n - hdrLen, addr, nil
}
