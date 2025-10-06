//go:build darwin || freebsd || linux || openbsd

package aghnet

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"os"
	"time"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/dhcpv6"
	"github.com/insomniacslk/dhcp/dhcpv6/nclient6"
	"github.com/insomniacslk/dhcp/iana"
)

// defaultDiscoverTime is the default timeout of checking another DHCP server
// response.
const defaultDiscoverTime = 3 * time.Second

func checkOtherDHCP(
	ctx context.Context,
	l *slog.Logger,
	ifaceName string,
) (ok4, ok6 bool, err4, err6 error) {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		err = fmt.Errorf("couldn't find interface by name %s: %w", ifaceName, err)
		err4, err6 = err, err

		return false, false, err4, err6
	}

	ok4, err4 = checkOtherDHCPv4(ctx, l, iface)
	ok6, err6 = checkOtherDHCPv6(ctx, l, iface)

	return ok4, ok6, err4, err6
}

// ifaceIPv4Subnet returns the first suitable IPv4 subnetwork iface has.
// iface must not be nil.
func ifaceIPv4Subnet(iface *net.Interface) (subnet netip.Prefix, err error) {
	var addrs []net.Addr
	if addrs, err = iface.Addrs(); err != nil {
		return netip.Prefix{}, err
	}

	for _, a := range addrs {
		var ip net.IP
		var maskLen int
		switch a := a.(type) {
		case *net.IPAddr:
			ip = a.IP
			maskLen, _ = ip.DefaultMask().Size()
		case *net.IPNet:
			ip = a.IP
			maskLen, _ = a.Mask.Size()
		default:
			continue
		}

		if ip = ip.To4(); ip != nil {
			return netip.PrefixFrom(netip.AddrFrom4([4]byte(ip)), maskLen), nil
		}
	}

	return netip.Prefix{}, fmt.Errorf("interface %s has no ipv4 addresses", iface.Name)
}

// checkOtherDHCPv4 sends a DHCP request to the specified network interface, and
// waits for a response for a period defined by defaultDiscoverTime.  l must not
// be nil.
func checkOtherDHCPv4(
	ctx context.Context,
	l *slog.Logger,
	iface *net.Interface,
) (ok bool, err error) {
	var subnet netip.Prefix
	if subnet, err = ifaceIPv4Subnet(iface); err != nil {
		return false, err
	}

	// Resolve broadcast addr.
	dst := netip.AddrPortFrom(BroadcastFromPref(subnet), 67).String()
	var dstAddr *net.UDPAddr
	if dstAddr, err = net.ResolveUDPAddr("udp4", dst); err != nil {
		return false, fmt.Errorf("couldn't resolve UDP address %s: %w", dst, err)
	}

	var hostname string
	if hostname, err = os.Hostname(); err != nil {
		return false, fmt.Errorf("couldn't get hostname: %w", err)
	}

	return discover4(ctx, l, iface, dstAddr, hostname)
}

// discover4 sends a DHCPv4 discovery to the specified network interface and
// waits for response.  iface and dstAddr must not be nil.
func discover4(
	ctx context.Context,
	l *slog.Logger,
	iface *net.Interface,
	dstAddr *net.UDPAddr,
	hostname string,
) (ok bool, err error) {
	var req *dhcpv4.DHCPv4
	if req, err = dhcpv4.NewDiscovery(iface.HardwareAddr); err != nil {
		return false, fmt.Errorf("dhcpv4.NewDiscovery: %w", err)
	}

	req.Options.Update(dhcpv4.OptClientIdentifier(iface.HardwareAddr))
	req.Options.Update(dhcpv4.OptHostName(hostname))
	req.SetBroadcast()

	// Bind to 0.0.0.0:68.
	//
	// On OpenBSD binding to the port 68 competes with dhclient's binding,
	// so that all incoming packets are ignored and the discovering process
	// is spoiled.
	//
	// It's also known that listening on the specified interface's address
	// ignores broadcast packets when reading.
	var c net.PacketConn
	if c, err = listenPacketReusable(iface.Name, "udp4", ":68"); err != nil {
		return false, fmt.Errorf("couldn't listen on :68: %w", err)
	}
	defer func() { err = errors.WithDeferred(err, c.Close()) }()

	// Send to broadcast.
	if _, err = c.WriteTo(req.ToBytes(), dstAddr); err != nil {
		return false, fmt.Errorf("couldn't send a packet to %s: %w", dstAddr, err)
	}

	for {
		var next bool
		ok, next, err = tryConn4(ctx, l, req, c, iface)
		if next {
			continue
		}

		if err != nil {
			return false, err
		}

		return ok, nil
	}
}

// tryConn4 reads and validates DHCPv4 response packet if it matches
// the original request.  req and c must not be nil.
//
// TODO(a.garipov): Refactor further.  Inspect error handling, remove parameter
// next, address the TODO, merge with tryConn6, etc.
func tryConn4(
	ctx context.Context,
	l *slog.Logger,
	req *dhcpv4.DHCPv4,
	c net.PacketConn,
	iface *net.Interface,
) (ok, next bool, err error) {
	if err = c.SetDeadline(time.Now().Add(defaultDiscoverTime)); err != nil {
		return false, false, fmt.Errorf("dhcpv4: setting deadline: %w", err)
	}

	// TODO: replicate dhclient's behavior of retrying several times with
	// progressively longer timeouts.
	l.Log(ctx, slogutil.LevelTrace, "waiting for an answer", "timeout", defaultDiscoverTime)

	b := make([]byte, 1500)
	n, _, err := c.ReadFrom(b)
	if err != nil {
		if errors.Is(err, os.ErrDeadlineExceeded) {
			l.DebugContext(ctx, "did not receive response")

			return false, false, nil
		}

		return false, false, fmt.Errorf("receiving packet: %w", err)
	}

	l.Log(ctx, slogutil.LevelTrace, "received packet", "size", n)

	response, err := dhcpv4.FromBytes(b[:n])
	if err != nil {
		l.DebugContext(ctx, "encoding", slogutil.KeyError, err)

		return false, true, err
	}

	l.DebugContext(ctx, "received message from server", "summary", response.Summary())

	switch {
	case
		response.OpCode != dhcpv4.OpcodeBootReply,
		response.HWType != iana.HWTypeEthernet,
		!bytes.Equal(response.ClientHWAddr, iface.HardwareAddr),
		response.TransactionID != req.TransactionID,
		!response.Options.Has(dhcpv4.OptionDHCPMessageType):
		l.DebugContext(ctx, "dhcpv4: received response does not match the request")

		return false, true, nil
	default:
		l.Log(ctx, slogutil.LevelTrace, "packet is from an active dhcp server")

		return true, false, nil
	}
}

// checkOtherDHCPv6 sends a DHCP request to the specified network interface, and
// waits for a response for a period defined by defaultDiscoverTime.  l must not
// be nil.
func checkOtherDHCPv6(
	ctx context.Context,
	l *slog.Logger,
	iface *net.Interface,
) (ok bool, err error) {
	ifaceIPNet, err := IfaceIPAddrs(iface, IPVersion6)
	if err != nil {
		return false, fmt.Errorf("getting ipv6 addrs for iface %s: %w", iface.Name, err)
	}
	if len(ifaceIPNet) == 0 {
		return false, fmt.Errorf("interface %s has no ipv6 addresses", iface.Name)
	}

	srcIP := ifaceIPNet[0]
	src := netutil.JoinHostPort(srcIP.String(), 546)
	dst := "[ff02::1:2]:547"

	udpAddr, err := net.ResolveUDPAddr("udp6", src)
	if err != nil {
		return false, fmt.Errorf("dhcpv6: Couldn't resolve UDP address %s: %w", src, err)
	}

	if !udpAddr.IP.To16().Equal(srcIP) {
		return false, fmt.Errorf("dhcpv6: Resolved UDP address is not %s: %w", src, err)
	}

	dstAddr, err := net.ResolveUDPAddr("udp6", dst)
	if err != nil {
		return false, fmt.Errorf("dhcpv6: Couldn't resolve UDP address %s: %w", dst, err)
	}

	return discover6(ctx, l, iface, udpAddr, dstAddr)
}

// discover6 sends a DHCPv6 discovery to the specified network interface and
// waits for response.  iface, updAddr and dstAddr must not be nil.
func discover6(
	ctx context.Context,
	l *slog.Logger,
	iface *net.Interface,
	udpAddr *net.UDPAddr,
	dstAddr *net.UDPAddr,
) (ok bool, err error) {
	req, err := dhcpv6.NewSolicit(iface.HardwareAddr)
	if err != nil {
		return false, fmt.Errorf("dhcpv6: dhcpv6.NewSolicit: %w", err)
	}

	l.DebugContext(ctx, "listening on udp6", "addr", udpAddr)
	c, err := nclient6.NewIPv6UDPConn(iface.Name, dhcpv6.DefaultClientPort)
	if err != nil {
		return false, fmt.Errorf("dhcpv6: Couldn't listen on :546: %w", err)
	}
	defer func() { err = errors.WithDeferred(err, c.Close()) }()

	_, err = c.WriteTo(req.ToBytes(), dstAddr)
	if err != nil {
		return false, fmt.Errorf("dhcpv6: Couldn't send a packet to %s: %w", dstAddr, err)
	}

	for {
		var next bool
		ok, next, err = tryConn6(ctx, l, req, c)
		if next {
			continue
		}

		if err != nil {
			return false, err
		}

		return ok, nil
	}
}

// tryConn6 reads and validates DHCPv6 response packet if it matches
// the original request.  req and c must not be nil.
//
// TODO(a.garipov): See the comment on tryConn4.  Sigh…
func tryConn6(
	ctx context.Context,
	l *slog.Logger,
	req *dhcpv6.Message,
	c net.PacketConn,
) (ok, next bool, err error) {
	// TODO: replicate dhclient's behavior of retrying several times with
	// progressively longer timeouts.
	l.Log(ctx, slogutil.LevelTrace, "waiting for an answer", "timeout", defaultDiscoverTime)

	b := make([]byte, 4096)
	err = c.SetDeadline(time.Now().Add(defaultDiscoverTime))
	if err != nil {
		return false, false, fmt.Errorf("setting deadline: %w", err)
	}

	n, _, err := c.ReadFrom(b)
	if err != nil {
		if errors.Is(err, os.ErrDeadlineExceeded) {
			l.DebugContext(ctx, "did not receive response")

			return false, false, nil
		}

		return false, false, fmt.Errorf("receiving packet: %w", err)
	}

	l.Log(ctx, slogutil.LevelTrace, "dhcpv6: received packet", "size", n)

	response, err := dhcpv6.FromBytes(b[:n])
	if err != nil {
		l.DebugContext(ctx, "encoding", slogutil.KeyError, err)

		return false, true, err
	}

	l.DebugContext(ctx, "received message from server", "summary", response.Summary())

	cid := req.Options.ClientID()
	msg, err := response.GetInnerMessage()
	if err != nil {
		l.DebugContext(ctx, "getting inner message", slogutil.KeyError, err)

		return false, true, err
	}

	rcid := msg.Options.ClientID()
	if !(response.Type() == dhcpv6.MessageTypeAdvertise &&
		msg.TransactionID == req.TransactionID &&
		rcid != nil &&
		cid.Equal(rcid)) {

		l.DebugContext(ctx, "received message from server does not match our request")

		return false, true, nil
	}

	l.Log(ctx, slogutil.LevelTrace, "dhcpv6: the packet is from an active dhcp server")

	return true, false, nil
}
