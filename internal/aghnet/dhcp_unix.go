//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris
// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

package aghnet

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/dhcpv6"
	"github.com/insomniacslk/dhcp/dhcpv6/nclient6"
	"github.com/insomniacslk/dhcp/iana"
)

// defaultDiscoverTime is the
const defaultDiscoverTime = 3 * time.Second

func checkOtherDHCP(ifaceName string) (ok4, ok6 bool, err4, err6 error) {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		err = fmt.Errorf("couldn't find interface by name %s: %w", ifaceName, err)
		err4, err6 = err, err

		return false, false, err4, err6
	}

	ok4, err4 = checkOtherDHCPv4(iface)
	ok6, err6 = checkOtherDHCPv6(iface)

	return ok4, ok6, err4, err6
}

// ifaceIPv4Subnet returns the first suitable IPv4 subnetwork iface has.
func ifaceIPv4Subnet(iface *net.Interface) (subnet *net.IPNet, err error) {
	var addrs []net.Addr
	if addrs, err = iface.Addrs(); err != nil {
		return nil, err
	}

	for _, a := range addrs {
		switch a := a.(type) {
		case *net.IPAddr:
			subnet = &net.IPNet{
				IP:   a.IP,
				Mask: a.IP.DefaultMask(),
			}
		case *net.IPNet:
			subnet = a
		default:
			continue
		}

		if ip4 := subnet.IP.To4(); ip4 != nil {
			subnet.IP = ip4

			return subnet, nil
		}
	}

	return nil, fmt.Errorf("interface %s has no ipv4 addresses", iface.Name)
}

// checkOtherDHCPv4 sends a DHCP request to the specified network interface, and
// waits for a response for a period defined by defaultDiscoverTime.
func checkOtherDHCPv4(iface *net.Interface) (ok bool, err error) {
	var subnet *net.IPNet
	if subnet, err = ifaceIPv4Subnet(iface); err != nil {
		return false, err
	}

	// Resolve broadcast addr.
	dst := netutil.IPPort{
		IP:   BroadcastFromIPNet(subnet),
		Port: 67,
	}.String()
	var dstAddr *net.UDPAddr
	if dstAddr, err = net.ResolveUDPAddr("udp4", dst); err != nil {
		return false, fmt.Errorf("couldn't resolve UDP address %s: %w", dst, err)
	}

	var hostname string
	if hostname, err = os.Hostname(); err != nil {
		return false, fmt.Errorf("couldn't get hostname: %w", err)
	}

	return discover4(iface, dstAddr, hostname)
}

func discover4(iface *net.Interface, dstAddr *net.UDPAddr, hostname string) (ok bool, err error) {
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
	// ignores broadcasted packets when reading.
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
		if err = c.SetDeadline(time.Now().Add(defaultDiscoverTime)); err != nil {
			return false, fmt.Errorf("setting deadline: %w", err)
		}

		var next bool
		ok, next, err = tryConn4(req, c, iface)
		if next {
			if err != nil {
				log.Debug("dhcpv4: trying a connection: %s", err)
			}

			continue
		}

		if err != nil {
			return false, err
		}

		return ok, nil
	}
}

// TODO(a.garipov): Refactor further.  Inspect error handling, remove parameter
// next, address the TODO, merge with tryConn6, etc.
func tryConn4(req *dhcpv4.DHCPv4, c net.PacketConn, iface *net.Interface) (ok, next bool, err error) {
	// TODO: replicate dhclient's behavior of retrying several times with
	// progressively longer timeouts.
	log.Tracef("dhcpv4: waiting %v for an answer", defaultDiscoverTime)

	b := make([]byte, 1500)
	n, _, err := c.ReadFrom(b)
	if err != nil {
		if isTimeout(err) {
			log.Debug("dhcpv4: didn't receive dhcp response")

			return false, false, nil
		}

		return false, false, fmt.Errorf("receiving packet: %w", err)
	}

	log.Tracef("dhcpv4: received packet, %d bytes", n)

	response, err := dhcpv4.FromBytes(b[:n])
	if err != nil {
		log.Debug("dhcpv4: encoding: %s", err)

		return false, true, err
	}

	log.Debug("dhcpv4: received message from server: %s", response.Summary())

	if !(response.OpCode == dhcpv4.OpcodeBootReply &&
		response.HWType == iana.HWTypeEthernet &&
		bytes.Equal(response.ClientHWAddr, iface.HardwareAddr) &&
		bytes.Equal(response.TransactionID[:], req.TransactionID[:]) &&
		response.Options.Has(dhcpv4.OptionDHCPMessageType)) {

		log.Debug("dhcpv4: received message from server doesn't match our request")

		return false, true, nil
	}

	log.Tracef("dhcpv4: the packet is from an active dhcp server")

	return true, false, nil
}

// checkOtherDHCPv6 sends a DHCP request to the specified network interface, and
// waits for a response for a period defined by defaultDiscoverTime.
func checkOtherDHCPv6(iface *net.Interface) (ok bool, err error) {
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

	return discover6(iface, udpAddr, dstAddr)
}

func discover6(iface *net.Interface, udpAddr, dstAddr *net.UDPAddr) (ok bool, err error) {
	req, err := dhcpv6.NewSolicit(iface.HardwareAddr)
	if err != nil {
		return false, fmt.Errorf("dhcpv6: dhcpv6.NewSolicit: %w", err)
	}

	log.Debug("DHCPv6: Listening to udp6 %+v", udpAddr)
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
		ok, next, err = tryConn6(req, c)
		if next {
			if err != nil {
				log.Debug("dhcpv6: trying a connection: %s", err)
			}

			continue
		}

		if err != nil {
			return false, err
		}

		return ok, nil
	}
}

// TODO(a.garipov): See the comment on tryConn4.  Sigh…
func tryConn6(req *dhcpv6.Message, c net.PacketConn) (ok, next bool, err error) {
	// TODO: replicate dhclient's behavior of retrying several times with
	// progressively longer timeouts.
	log.Tracef("dhcpv6: waiting %v for an answer", defaultDiscoverTime)

	b := make([]byte, 4096)
	err = c.SetDeadline(time.Now().Add(defaultDiscoverTime))
	if err != nil {
		return false, false, fmt.Errorf("setting deadline: %w", err)
	}

	n, _, err := c.ReadFrom(b)
	if err != nil {
		if isTimeout(err) {
			log.Debug("dhcpv6: didn't receive dhcp response")

			return false, false, nil
		}

		return false, false, fmt.Errorf("receiving packet: %w", err)
	}

	log.Tracef("dhcpv6: received packet, %d bytes", n)

	response, err := dhcpv6.FromBytes(b[:n])
	if err != nil {
		log.Debug("dhcpv6: encoding: %s", err)

		return false, true, err
	}

	log.Debug("dhcpv6: received message from server: %s", response.Summary())

	cid := req.Options.ClientID()
	msg, err := response.GetInnerMessage()
	if err != nil {
		log.Debug("dhcpv6: resp.GetInnerMessage(): %s", err)

		return false, true, err
	}

	rcid := msg.Options.ClientID()
	if !(response.Type() == dhcpv6.MessageTypeAdvertise &&
		msg.TransactionID == req.TransactionID &&
		rcid != nil &&
		cid.Equal(*rcid)) {

		log.Debug("dhcpv6: received message from server doesn't match our request")

		return false, true, nil
	}

	log.Tracef("dhcpv6: the packet is from an active dhcp server")

	return true, false, nil
}

// isTimeout returns true if err is an operation timeout error from net package.
//
// TODO(e.burkov):  Consider moving into netutil.
func isTimeout(err error) (ok bool) {
	var operr *net.OpError
	if errors.As(err, &operr) {
		return operr.Timeout()
	}

	return false
}
