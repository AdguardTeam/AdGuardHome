// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

package dhcpd

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/AdguardTeam/golibs/log"
	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/iana"
	"golang.org/x/net/ipv4"
)

// CheckIfOtherDHCPServersPresent sends a DHCP request to the specified network interface,
// and waits for a response for a period defined by defaultDiscoverTime
func CheckIfOtherDHCPServersPresent(ifaceName string) (bool, error) {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return false, wrapErrPrint(err, "Couldn't find interface by name %s", ifaceName)
	}

	// get ipv4 address of an interface
	ifaceIPNet := getIfaceIPv4(*iface)
	if len(ifaceIPNet) == 0 {
		return false, fmt.Errorf("couldn't find IPv4 address of interface %s %+v", ifaceName, iface)
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
	c, err := newBroadcastPacketConn(net.IPv4(0, 0, 0, 0), 68, ifaceName)
	if c != nil {
		defer c.Close()
	}
	if err != nil {
		return false, wrapErrPrint(err, "Couldn't listen on :68")
	}

	// send to 255.255.255.255:67
	cm := ipv4.ControlMessage{}
	_, err = c.WriteTo(req.ToBytes(), &cm, dstAddr)
	if err != nil {
		return false, wrapErrPrint(err, "Couldn't send a packet to %s", dst)
	}

	for {
		// wait for answer
		log.Tracef("Waiting %v for an answer", defaultDiscoverTime)
		// TODO: replicate dhclient's behaviour of retrying several times with progressively bigger timeouts
		b := make([]byte, 1500)
		_ = c.SetReadDeadline(time.Now().Add(defaultDiscoverTime))
		n, _, _, err := c.ReadFrom(b)
		if isTimeout(err) {
			// timed out -- no DHCP servers
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
