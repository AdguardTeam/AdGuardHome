package dhcpd

import (
	"bytes"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"os"
	"time"

	"github.com/AdguardTeam/golibs/log"
	"github.com/krolaw/dhcp4"
	"golang.org/x/net/ipv4"
)

// CheckIfOtherDHCPServersPresent sends a DHCP request to the specified network interface,
// and waits for a response for a period defined by defaultDiscoverTime
// nolint
func CheckIfOtherDHCPServersPresent(ifaceName string) (bool, error) {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return false, wrapErrPrint(err, "Couldn't find interface by name %s", ifaceName)
	}

	// get ipv4 address of an interface
	ifaceIPNet := getIfaceIPv4(*iface)
	if len(ifaceIPNet) == 0 {
		return false, fmt.Errorf("Couldn't find IPv4 address of interface %s %+v", ifaceName, iface)
	}

	srcIP := ifaceIPNet[0]
	src := net.JoinHostPort(srcIP.String(), "68")
	dst := "255.255.255.255:67"

	// form a DHCP request packet, try to emulate existing client as much as possible
	xID := make([]byte, 4)
	n, err := rand.Read(xID)
	if n != 4 && err == nil {
		err = fmt.Errorf("Generated less than 4 bytes")
	}
	if err != nil {
		return false, wrapErrPrint(err, "Couldn't generate random bytes")
	}
	hostname, err := os.Hostname()
	if err != nil {
		return false, wrapErrPrint(err, "Couldn't get hostname")
	}
	requestList := []byte{
		byte(dhcp4.OptionSubnetMask),
		byte(dhcp4.OptionClasslessRouteFormat),
		byte(dhcp4.OptionRouter),
		byte(dhcp4.OptionDomainNameServer),
		byte(dhcp4.OptionDomainName),
		byte(dhcp4.OptionDomainSearch),
		252, // private/proxy autodiscovery
		95,  // LDAP
		byte(dhcp4.OptionNetBIOSOverTCPIPNameServer),
		byte(dhcp4.OptionNetBIOSOverTCPIPNodeType),
	}
	maxUDPsizeRaw := make([]byte, 2)
	binary.BigEndian.PutUint16(maxUDPsizeRaw, 1500)
	leaseTimeRaw := make([]byte, 4)
	leaseTime := uint32(math.RoundToEven((time.Hour * 24 * 90).Seconds()))
	binary.BigEndian.PutUint32(leaseTimeRaw, leaseTime)
	options := []dhcp4.Option{
		{Code: dhcp4.OptionParameterRequestList, Value: requestList},
		{Code: dhcp4.OptionMaximumDHCPMessageSize, Value: maxUDPsizeRaw},
		{Code: dhcp4.OptionClientIdentifier, Value: append([]byte{0x01}, iface.HardwareAddr...)},
		{Code: dhcp4.OptionIPAddressLeaseTime, Value: leaseTimeRaw},
		{Code: dhcp4.OptionHostName, Value: []byte(hostname)},
	}
	packet := dhcp4.RequestPacket(dhcp4.Discover, iface.HardwareAddr, nil, xID, false, options)

	// resolve 0.0.0.0:68
	udpAddr, err := net.ResolveUDPAddr("udp4", src)
	if err != nil {
		return false, wrapErrPrint(err, "Couldn't resolve UDP address %s", src)
	}
	// spew.Dump(udpAddr, err)

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
	_, err = c.WriteTo(packet, &cm, dstAddr)
	if err != nil {
		return false, wrapErrPrint(err, "Couldn't send a packet to %s", dst)
	}

	for {
		// wait for answer
		log.Tracef("Waiting %v for an answer", defaultDiscoverTime)
		// TODO: replicate dhclient's behaviour of retrying several times with progressively bigger timeouts
		b := make([]byte, 1500)
		_ = c.SetReadDeadline(time.Now().Add(defaultDiscoverTime))
		n, _, _, err = c.ReadFrom(b)
		if isTimeout(err) {
			// timed out -- no DHCP servers
			return false, nil
		}
		if err != nil {
			return false, wrapErrPrint(err, "Couldn't receive packet")
		}
		// spew.Dump(n, fromAddr, err, b)

		log.Tracef("Received packet (%v bytes)", n)

		if n < 240 {
			// packet too small for dhcp
			continue
		}

		response := dhcp4.Packet(b[:n])
		if response.OpCode() != dhcp4.BootReply ||
			response.HType() != 1 /*Ethernet*/ ||
			response.HLen() > 16 ||
			!bytes.Equal(response.CHAddr(), iface.HardwareAddr) ||
			!bytes.Equal(response.XId(), xID) {
			continue
		}

		parsedOptions := response.ParseOptions()
		if t := parsedOptions[dhcp4.OptionDHCPMessageType]; len(t) != 1 {
			continue //packet without DHCP message type
		}

		log.Tracef("The packet is from an active DHCP server")
		// that's a DHCP server there
		return true, nil
	}
}
