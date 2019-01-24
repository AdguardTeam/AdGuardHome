package dhcpd

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"os"
	"time"

	"github.com/hmage/golibs/log"
	"github.com/krolaw/dhcp4"
)

// CheckIfOtherDHCPServersPresent sends a DHCP request to the specified network interface,
// and waits for a response for a period defined by defaultDiscoverTime
func CheckIfOtherDHCPServersPresent(ifaceName string) (bool, error) {
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return false, wrapErrPrint(err, "Couldn't find interface by name %s", ifaceName)
	}

	// get ipv4 address of an interface
	ifaceIPNet := getIfaceIPv4(iface)
	if ifaceIPNet == nil {
		return false, fmt.Errorf("Couldn't find IPv4 address of interface %s %+v", ifaceName, iface)
	}

	srcIP := ifaceIPNet.IP
	src := net.JoinHostPort(srcIP.String(), "68")
	dst := "255.255.255.255:67"

	// form a DHCP request packet, try to emulate existing client as much as possible
	xID := make([]byte, 8)
	n, err := rand.Read(xID)
	if n != 8 && err == nil {
		err = fmt.Errorf("Generated less than 8 bytes")
	}
	if err != nil {
		return false, wrapErrPrint(err, "Couldn't generate 8 random bytes")
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
	leaseTime := uint32(math.RoundToEven(time.Duration(time.Hour * 24 * 90).Seconds()))
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
	c, err := net.ListenPacket("udp4", src)
	if c != nil {
		defer c.Close()
	}
	// spew.Dump(c, err)
	// spew.Printf("net.ListenUDP returned %v, %v\n", c, err)
	if err != nil {
		return false, wrapErrPrint(err, "Couldn't listen to %s", src)
	}

	// send to 255.255.255.255:67
	_, err = c.WriteTo(packet, dstAddr)
	// spew.Dump(n, err)
	if err != nil {
		return false, wrapErrPrint(err, "Couldn't send a packet to %s", dst)
	}

	// wait for answer
	log.Tracef("Waiting %v for an answer", defaultDiscoverTime)
	// TODO: replicate dhclient's behaviour of retrying several times with progressively bigger timeouts
	b := make([]byte, 1500)
	c.SetReadDeadline(time.Now().Add(defaultDiscoverTime))
	n, _, err = c.ReadFrom(b)
	if isTimeout(err) {
		// timed out -- no DHCP servers
		return false, nil
	}
	if err != nil {
		return false, wrapErrPrint(err, "Couldn't receive packet")
	}
	if n > 0 {
		b = b[:n]
	}
	// spew.Dump(n, fromAddr, err, b)

	if n < 240 {
		// packet too small for dhcp
		return false, wrapErrPrint(err, "got packet that's too small for DHCP")
	}

	response := dhcp4.Packet(b[:n])
	if response.HLen() > 16 {
		// invalid size
		return false, wrapErrPrint(err, "got malformed packet with HLen() > 16")
	}

	parsedOptions := response.ParseOptions()
	_, ok := parsedOptions[dhcp4.OptionDHCPMessageType]
	if !ok {
		return false, wrapErrPrint(err, "got malformed packet without DHCP message type")
	}

	// that's a DHCP server there
	return true, nil
}
