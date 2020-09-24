package dhcpd

import (
	"encoding/binary"
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"github.com/AdguardTeam/golibs/log"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv6"
)

type raCtx struct {
	raAllowSlaac     bool   // send RA packets without MO flags
	raSlaacOnly      bool   // send RA packets with MO flags
	ipAddr           net.IP // source IP address (link-local-unicast)
	dnsIPAddr        net.IP // IP address for DNS Server option
	prefixIPAddr     net.IP // IP address for Prefix option
	ifaceName        string
	iface            *net.Interface
	packetSendPeriod time.Duration // how often RA packets are sent

	conn *icmp.PacketConn // ICMPv6 socket
	stop atomic.Value     // stop the packet sending loop
}

type icmpv6RA struct {
	managedAddressConfiguration bool
	otherConfiguration          bool
	prefix                      net.IP
	prefixLen                   int
	sourceLinkLayerAddress      net.HardwareAddr
	recursiveDNSServer          net.IP
	mtu                         uint32
}

// Create an ICMPv6.RouterAdvertisement packet with all necessary options.
//
// ICMPv6:
// type[1]
// code[1]
// chksum[2]
// body (RouterAdvertisement):
//   Cur Hop Limit[1]
//   Flags[1]: MO......
//   Router Lifetime[2]
//   Reachable Time[4]
//   Retrans Timer[4]
//   Option=Prefix Information(3):
//     Type[1]
//     Length * 8bytes[1]
//     Prefix Length[1]
//     Flags[1]: LA......
//     Valid Lifetime[4]
//     Preferred Lifetime[4]
//     Reserved[4]
//     Prefix[16]
//   Option=MTU(5):
//     Type[1]
//     Length * 8bytes[1]
//     Reserved[2]
//     MTU[4]
//   Option=Source link-layer address(1):
//     Link-Layer Address[6]
//   Option=Recursive DNS Server(25):
//     Type[1]
//     Length * 8bytes[1]
//     Reserved[2]
//     Lifetime[4]
//     Addresses of IPv6 Recursive DNS Servers[16]
func createICMPv6RAPacket(params icmpv6RA) []byte {
	data := make([]byte, 88)
	i := 0

	// ICMPv6:

	data[i] = 134 // type
	data[i+1] = 0 // code
	data[i+2] = 0 // chksum
	data[i+3] = 0
	i += 4

	// RouterAdvertisement:

	data[i] = 64 // Cur Hop Limit[1]
	i++

	data[i] = 0 // Flags[1]: MO......
	if params.managedAddressConfiguration {
		data[i] |= 0x80
	}
	if params.otherConfiguration {
		data[i] |= 0x40
	}
	i++

	binary.BigEndian.PutUint16(data[i:], 1800) // Router Lifetime[2]
	i += 2
	binary.BigEndian.PutUint32(data[i:], 0) // Reachable Time[4]
	i += 4
	binary.BigEndian.PutUint32(data[i:], 0) // Retrans Timer[4]
	i += 4

	// Option=Prefix Information:

	data[i] = 3   // Type
	data[i+1] = 4 // Length
	i += 2
	data[i] = byte(params.prefixLen) // Prefix Length[1]
	i++
	data[i] = 0xc0 // Flags[1]
	i++
	binary.BigEndian.PutUint32(data[i:], 3600) // Valid Lifetime[4]
	i += 4
	binary.BigEndian.PutUint32(data[i:], 3600) // Preferred Lifetime[4]
	i += 4
	binary.BigEndian.PutUint32(data[i:], 0) // Reserved[4]
	i += 4
	copy(data[i:], params.prefix[:8]) // Prefix[16]
	binary.BigEndian.PutUint32(data[i+8:], 0)
	binary.BigEndian.PutUint32(data[i+12:], 0)
	i += 16

	// Option=MTU:

	data[i] = 5   // Type
	data[i+1] = 1 // Length
	i += 2
	binary.BigEndian.PutUint16(data[i:], 0) // Reserved[2]
	i += 2
	binary.BigEndian.PutUint32(data[i:], params.mtu) // MTU[4]
	i += 4

	// Option=Source link-layer address:

	data[i] = 1   // Type
	data[i+1] = 1 // Length
	i += 2
	copy(data[i:], params.sourceLinkLayerAddress) // Link-Layer Address[6]
	i += 6

	// Option=Recursive DNS Server:

	data[i] = 25  // Type
	data[i+1] = 3 // Length
	i += 2
	binary.BigEndian.PutUint16(data[i:], 0) // Reserved[2]
	i += 2
	binary.BigEndian.PutUint32(data[i:], 3600) // Lifetime[4]
	i += 4
	copy(data[i:], params.recursiveDNSServer) // Addresses of IPv6 Recursive DNS Servers[16]

	return data
}

// Init - initialize RA module
func (ra *raCtx) Init() error {
	ra.stop.Store(0)
	ra.conn = nil
	if !(ra.raAllowSlaac || ra.raSlaacOnly) {
		return nil
	}

	log.Debug("DHCPv6 RA: source IP address: %s  DNS IP address: %s",
		ra.ipAddr, ra.dnsIPAddr)

	params := icmpv6RA{
		managedAddressConfiguration: !ra.raSlaacOnly,
		otherConfiguration:          !ra.raSlaacOnly,
		mtu:                         uint32(ra.iface.MTU),
		prefixLen:                   64,
		recursiveDNSServer:          ra.dnsIPAddr,
		sourceLinkLayerAddress:      ra.iface.HardwareAddr,
	}
	params.prefix = make([]byte, 16)
	copy(params.prefix, ra.prefixIPAddr[:8]) // /64

	data := createICMPv6RAPacket(params)

	var err error
	ipAndScope := ra.ipAddr.String() + "%" + ra.ifaceName
	ra.conn, err = icmp.ListenPacket("ip6:ipv6-icmp", ipAndScope)
	if err != nil {
		return fmt.Errorf("DHCPv6 RA: icmp.ListenPacket: %s", err)
	}
	success := false
	defer func() {
		if !success {
			ra.Close()
		}
	}()

	con6 := ra.conn.IPv6PacketConn()

	if err := con6.SetHopLimit(255); err != nil {
		return fmt.Errorf("DHCPv6 RA: SetHopLimit: %s", err)
	}

	if err := con6.SetMulticastHopLimit(255); err != nil {
		return fmt.Errorf("DHCPv6 RA: SetMulticastHopLimit: %s", err)
	}

	msg := &ipv6.ControlMessage{
		HopLimit: 255,
		Src:      ra.ipAddr,
		IfIndex:  ra.iface.Index,
	}
	addr := &net.UDPAddr{
		IP: net.ParseIP("ff02::1"),
	}

	go func() {
		log.Debug("DHCPv6 RA: starting to send periodic RouterAdvertisement packets")
		for ra.stop.Load() == 0 {
			_, err = con6.WriteTo(data, msg, addr)
			if err != nil {
				log.Error("DHCPv6 RA: WriteTo: %s", err)
			}
			time.Sleep(ra.packetSendPeriod)
		}
		log.Debug("DHCPv6 RA: loop exit")
	}()

	success = true
	return nil
}

// Close - close module
func (ra *raCtx) Close() {
	log.Debug("DHCPv6 RA: closing")

	ra.stop.Store(1)

	if ra.conn != nil {
		ra.conn.Close()
	}
}
