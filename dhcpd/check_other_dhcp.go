// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

package dhcpd

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"runtime"
	"time"

	"github.com/AdguardTeam/AdGuardHome/dhcpd/nclient4"
	"github.com/AdguardTeam/golibs/log"
	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/dhcpv6"
	"github.com/insomniacslk/dhcp/dhcpv6/nclient6"
	"github.com/insomniacslk/dhcp/iana"
)

// CheckIfOtherDHCPServersPresentV4 sends a DHCP request to the specified network interface,
// and waits for a response for a period defined by defaultDiscoverTime
func CheckIfOtherDHCPServersPresentV4(ifaceName string) (bool, error) {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return false, wrapErrPrint(err, "Couldn't find interface by name %s", ifaceName)
	}

	// get ipv4 address of an interface
	ifaceIPNet := getIfaceIPv4(*iface)
	if len(ifaceIPNet) == 0 {
		return false, fmt.Errorf("couldn't find IPv4 address of interface %s %+v", ifaceName, iface)
	}

	if runtime.GOOS == "darwin" {
		return false, fmt.Errorf("can't find DHCP server: not supported on macOS")
	}

	srcIP := ifaceIPNet[0]
	src := net.JoinHostPort(srcIP.String(), "68")
	dst := "255.255.255.255:67"

	hostname, _ := os.Hostname()

	req, err := dhcpv4.NewDiscovery(iface.HardwareAddr)
	if err != nil {
		return false, fmt.Errorf("dhcpv4.NewDiscovery: %s", err)
	}
	req.Options.Update(dhcpv4.OptClientIdentifier(iface.HardwareAddr))
	req.Options.Update(dhcpv4.OptHostName(hostname))

	// resolve 0.0.0.0:68
	udpAddr, err := net.ResolveUDPAddr("udp4", src)
	if err != nil {
		return false, wrapErrPrint(err, "Couldn't resolve UDP address %s", src)
	}

	if !udpAddr.IP.To4().Equal(srcIP) {
		return false, wrapErrPrint(err, "Resolved UDP address is not %s", src)
	}

	// resolve 255.255.255.255:67
	dstAddr, err := net.ResolveUDPAddr("udp4", dst)
	if err != nil {
		return false, wrapErrPrint(err, "Couldn't resolve UDP address %s", dst)
	}

	// bind to 0.0.0.0:68
	log.Tracef("Listening to udp4 %+v", udpAddr)
	c, err := nclient4.NewRawUDPConn(ifaceName, 68)
	if err != nil {
		return false, wrapErrPrint(err, "Couldn't listen on :68")
	}
	if c != nil {
		defer c.Close()
	}

	// send to 255.255.255.255:67
	_, err = c.WriteTo(req.ToBytes(), dstAddr)
	if err != nil {
		return false, wrapErrPrint(err, "Couldn't send a packet to %s", dst)
	}

	for {
		// wait for answer
		log.Tracef("Waiting %v for an answer", defaultDiscoverTime)
		// TODO: replicate dhclient's behaviour of retrying several times with progressively bigger timeouts
		b := make([]byte, 1500)
		_ = c.SetReadDeadline(time.Now().Add(defaultDiscoverTime))
		n, _, err := c.ReadFrom(b)
		if isTimeout(err) {
			// timed out -- no DHCP servers
			log.Debug("DHCPv4: didn't receive DHCP response")
			return false, nil
		}
		if err != nil {
			return false, wrapErrPrint(err, "Couldn't receive packet")
		}

		log.Tracef("Received packet (%v bytes)", n)

		response, err := dhcpv4.FromBytes(b[:n])
		if err != nil {
			log.Debug("DHCPv4: dhcpv4.FromBytes: %s", err)
			continue
		}

		log.Debug("DHCPv4: received message from server: %s", response.Summary())

		if !(response.OpCode == dhcpv4.OpcodeBootReply &&
			response.HWType == iana.HWTypeEthernet &&
			bytes.Equal(response.ClientHWAddr, iface.HardwareAddr) &&
			bytes.Equal(response.TransactionID[:], req.TransactionID[:]) &&
			response.Options.Has(dhcpv4.OptionDHCPMessageType)) {
			log.Debug("DHCPv4: received message from server doesn't match our request")
			continue
		}

		log.Tracef("The packet is from an active DHCP server")
		// that's a DHCP server there
		return true, nil
	}
}

// CheckIfOtherDHCPServersPresentV6 sends a DHCP request to the specified network interface,
// and waits for a response for a period defined by defaultDiscoverTime
func CheckIfOtherDHCPServersPresentV6(ifaceName string) (bool, error) {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return false, fmt.Errorf("DHCPv6: net.InterfaceByName: %s: %s", ifaceName, err)
	}

	ifaceIPNet := getIfaceIPv6(*iface)
	if len(ifaceIPNet) == 0 {
		return false, fmt.Errorf("DHCPv6: couldn't find IPv6 address of interface %s %+v", ifaceName, iface)
	}

	srcIP := ifaceIPNet[0]
	src := net.JoinHostPort(srcIP.String(), "546")
	dst := "[ff02::1:2]:547"

	req, err := dhcpv6.NewSolicit(iface.HardwareAddr)
	if err != nil {
		return false, fmt.Errorf("DHCPv6: dhcpv6.NewSolicit: %s", err)
	}

	udpAddr, err := net.ResolveUDPAddr("udp6", src)
	if err != nil {
		return false, wrapErrPrint(err, "DHCPv6: Couldn't resolve UDP address %s", src)
	}

	if !udpAddr.IP.To16().Equal(srcIP) {
		return false, wrapErrPrint(err, "DHCPv6: Resolved UDP address is not %s", src)
	}

	dstAddr, err := net.ResolveUDPAddr("udp6", dst)
	if err != nil {
		return false, fmt.Errorf("DHCPv6: Couldn't resolve UDP address %s: %s", dst, err)
	}

	log.Debug("DHCPv6: Listening to udp6 %+v", udpAddr)
	c, err := nclient6.NewIPv6UDPConn(ifaceName, dhcpv6.DefaultClientPort)
	if err != nil {
		return false, fmt.Errorf("DHCPv6: Couldn't listen on :546: %s", err)
	}
	if c != nil {
		defer c.Close()
	}

	_, err = c.WriteTo(req.ToBytes(), dstAddr)
	if err != nil {
		return false, fmt.Errorf("DHCPv6: Couldn't send a packet to %s: %s", dst, err)
	}

	for {
		log.Debug("DHCPv6: Waiting %v for an answer", defaultDiscoverTime)
		b := make([]byte, 4096)
		_ = c.SetReadDeadline(time.Now().Add(defaultDiscoverTime))
		n, _, err := c.ReadFrom(b)
		if isTimeout(err) {
			log.Debug("DHCPv6: didn't receive DHCP response")
			return false, nil
		}
		if err != nil {
			return false, wrapErrPrint(err, "Couldn't receive packet")
		}

		log.Debug("DHCPv6: Received packet (%v bytes)", n)

		resp, err := dhcpv6.FromBytes(b[:n])
		if err != nil {
			log.Debug("DHCPv6: dhcpv6.FromBytes: %s", err)
			continue
		}

		log.Debug("DHCPv6: received message from server: %s", resp.Summary())

		cid := req.Options.ClientID()
		msg, err := resp.GetInnerMessage()
		if err != nil {
			log.Debug("DHCPv6: resp.GetInnerMessage: %s", err)
			continue
		}
		rcid := msg.Options.ClientID()
		if resp.Type() == dhcpv6.MessageTypeAdvertise &&
			msg.TransactionID == req.TransactionID &&
			rcid != nil &&
			cid.Equal(*rcid) {
			log.Debug("DHCPv6: The packet is from an active DHCP server")
			return true, nil
		}

		log.Debug("DHCPv6: received message from server doesn't match our request")
	}
}
