//go:build darwin || freebsd || linux || openbsd

package dhcpd

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/dhcpv4/server4"
	"github.com/mdlayher/ethernet"
	"github.com/mdlayher/packet"
)

// dhcpUnicastAddr is the combination of MAC and IP addresses for responding to
// the unconfigured host.
type dhcpUnicastAddr struct {
	// packet.Addr is embedded here to make *dhcpUcastAddr a net.Addr without
	// actually implementing all methods.  It also contains the client's
	// hardware address.
	packet.Addr

	// yiaddr is an IP address just allocated by server for the host.
	yiaddr net.IP
}

// dhcpConn is the net.PacketConn capable of handling both net.UDPAddr and
// net.HardwareAddr.
type dhcpConn struct {
	// udpConn is the connection for UDP addresses.
	udpConn net.PacketConn
	// bcastIP is the broadcast address specific for the configured
	// interface's subnet.
	bcastIP net.IP

	// rawConn is the connection for MAC addresses.
	rawConn net.PacketConn
	// srcMAC is the hardware address of the configured network interface.
	srcMAC net.HardwareAddr
	// srcIP is the IP address  of the configured network interface.
	srcIP net.IP
}

// newDHCPConn creates the special connection for DHCP server.
func (s *v4Server) newDHCPConn(iface *net.Interface) (c net.PacketConn, err error) {
	var ucast net.PacketConn
	if ucast, err = packet.Listen(iface, packet.Raw, int(ethernet.EtherTypeIPv4), nil); err != nil {
		return nil, fmt.Errorf("creating raw udp connection: %w", err)
	}

	// Create the UDP connection.
	var bcast net.PacketConn
	bcast, err = server4.NewIPv4UDPConn(iface.Name, &net.UDPAddr{
		// TODO(e.burkov):  Listening on zeroes makes the server handle
		// requests from all the interfaces.  Inspect the ways to
		// specify the interface-specific listening addresses.
		//
		// See https://github.com/AdguardTeam/AdGuardHome/issues/3539.
		IP:   net.IP{0, 0, 0, 0},
		Port: dhcpv4.ServerPort,
	})
	if err != nil {
		return nil, fmt.Errorf("creating ipv4 udp connection: %w", err)
	}

	return &dhcpConn{
		udpConn: bcast,
		bcastIP: s.conf.broadcastIP.AsSlice(),
		rawConn: ucast,
		srcMAC:  iface.HardwareAddr,
		srcIP:   s.conf.dnsIPAddrs[0].AsSlice(),
	}, nil
}

// wrapErrs is a helper to wrap the errors from two independent underlying
// connections.
func (*dhcpConn) wrapErrs(action string, udpConnErr, rawConnErr error) (err error) {
	switch {
	case udpConnErr != nil && rawConnErr != nil:
		return errors.List(fmt.Sprintf("%s both connections", action), udpConnErr, rawConnErr)
	case udpConnErr != nil:
		return fmt.Errorf("%s udp connection: %w", action, udpConnErr)
	case rawConnErr != nil:
		return fmt.Errorf("%s raw connection: %w", action, rawConnErr)
	default:
		return nil
	}
}

// WriteTo implements net.PacketConn for *dhcpConn.  It selects the underlying
// connection to write to based on the type of addr.
func (c *dhcpConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	switch addr := addr.(type) {
	case *dhcpUnicastAddr:
		// Unicast the message to the client's MAC address.  Use the raw
		// connection.
		//
		// Note: unicasting is performed on the only network interface
		// that is configured.  For now it may be not what users expect
		// so additionally broadcast the message via UDP connection.
		//
		// See https://github.com/AdguardTeam/AdGuardHome/issues/3539.
		var rerr error
		n, rerr = c.unicast(p, addr)

		_, uerr := c.broadcast(p, &net.UDPAddr{
			IP:   netutil.IPv4bcast(),
			Port: dhcpv4.ClientPort,
		})

		return n, c.wrapErrs("writing to", uerr, rerr)
	case *net.UDPAddr:
		if addr.IP.Equal(net.IPv4bcast) {
			// Broadcast the message for the client which supports
			// it.  Use the UDP connection.
			return c.broadcast(p, addr)
		}

		// Unicast the message to the client's IP address.  Use the UDP
		// connection.
		return c.udpConn.WriteTo(p, addr)
	default:
		return 0, fmt.Errorf("addr has an unexpected type %T", addr)
	}
}

// ReadFrom implements net.PacketConn for *dhcpConn.
func (c *dhcpConn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	return c.udpConn.ReadFrom(p)
}

// unicast wraps respData with required frames and writes it to the peer.
func (c *dhcpConn) unicast(respData []byte, peer *dhcpUnicastAddr) (n int, err error) {
	var data []byte
	data, err = c.buildEtherPkt(respData, peer)
	if err != nil {
		return 0, err
	}

	return c.rawConn.WriteTo(data, &peer.Addr)
}

// Close implements net.PacketConn for *dhcpConn.
func (c *dhcpConn) Close() (err error) {
	rerr := c.rawConn.Close()
	if errors.Is(rerr, os.ErrClosed) {
		// Ignore the error since the actual file is closed already.
		rerr = nil
	}

	return c.wrapErrs("closing", c.udpConn.Close(), rerr)
}

// LocalAddr implements net.PacketConn for *dhcpConn.
func (c *dhcpConn) LocalAddr() (a net.Addr) {
	return c.udpConn.LocalAddr()
}

// SetDeadline implements net.PacketConn for *dhcpConn.
func (c *dhcpConn) SetDeadline(t time.Time) (err error) {
	return c.wrapErrs("setting deadline on", c.udpConn.SetDeadline(t), c.rawConn.SetDeadline(t))
}

// SetReadDeadline implements net.PacketConn for *dhcpConn.
func (c *dhcpConn) SetReadDeadline(t time.Time) error {
	return c.wrapErrs(
		"setting reading deadline on",
		c.udpConn.SetReadDeadline(t),
		c.rawConn.SetReadDeadline(t),
	)
}

// SetWriteDeadline implements net.PacketConn for *dhcpConn.
func (c *dhcpConn) SetWriteDeadline(t time.Time) error {
	return c.wrapErrs(
		"setting writing deadline on",
		c.udpConn.SetWriteDeadline(t),
		c.rawConn.SetWriteDeadline(t),
	)
}

// ipv4DefaultTTL is the default Time to Live value in seconds as recommended by
// RFC-1700.
//
// See https://datatracker.ietf.org/doc/html/rfc1700.
const ipv4DefaultTTL = 64

// buildEtherPkt wraps the payload with IPv4, UDP and Ethernet frames.
// Validation of the payload is a caller's responsibility.
func (c *dhcpConn) buildEtherPkt(payload []byte, peer *dhcpUnicastAddr) (pkt []byte, err error) {
	udpLayer := &layers.UDP{
		SrcPort: dhcpv4.ServerPort,
		DstPort: dhcpv4.ClientPort,
	}

	ipv4Layer := &layers.IPv4{
		Version:  uint8(layers.IPProtocolIPv4),
		Flags:    layers.IPv4DontFragment,
		TTL:      ipv4DefaultTTL,
		Protocol: layers.IPProtocolUDP,
		SrcIP:    c.srcIP,
		DstIP:    peer.yiaddr,
	}

	// Ignore the error since it's only returned for invalid network layer's
	// type.
	_ = udpLayer.SetNetworkLayerForChecksum(ipv4Layer)

	ethLayer := &layers.Ethernet{
		SrcMAC:       c.srcMAC,
		DstMAC:       peer.HardwareAddr,
		EthernetType: layers.EthernetTypeIPv4,
	}

	buf := gopacket.NewSerializeBuffer()
	setts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	err = gopacket.SerializeLayers(
		buf,
		setts,
		ethLayer,
		ipv4Layer,
		udpLayer,
		gopacket.Payload(payload),
	)
	if err != nil {
		return nil, fmt.Errorf("serializing layers: %w", err)
	}

	return buf.Bytes(), nil
}
