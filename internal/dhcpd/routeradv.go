package dhcpd

import (
	"encoding/binary"
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv6"
)

type raCtx struct {
	raAllowSLAAC     bool   // send RA packets without MO flags
	raSLAACOnly      bool   // send RA packets with MO flags
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

// hwAddrToLinkLayerAddr converts a hardware address into a form required by
// RFC4861.  That is, a byte slice of length divisible by 8.
//
// See https://tools.ietf.org/html/rfc4861#section-4.6.1.
func hwAddrToLinkLayerAddr(hwa net.HardwareAddr) (lla []byte, err error) {
	err = netutil.ValidateMAC(hwa)
	if err != nil {
		// Don't wrap the error, because it already contains enough
		// context.
		return nil, err
	}

	if len(hwa) == 6 || len(hwa) == 8 {
		lla = make([]byte, 8)
		copy(lla, hwa)

		return lla, nil
	}

	// Assume that netutil.ValidateMAC prevents lengths other than 20 by
	// now.
	lla = make([]byte, 24)
	copy(lla, hwa)

	return lla, nil
}

// Create an ICMPv6.RouterAdvertisement packet with all necessary options.
// Data scheme:
//
//	ICMPv6:
//	- type[1]
//	- code[1]
//	- chksum[2]
//	- body (RouterAdvertisement):
//	  - Cur Hop Limit[1]
//	  - Flags[1]: MO......
//	  - Router Lifetime[2]
//	  - Reachable Time[4]
//	  - Retrans Timer[4]
//	  - Option=Prefix Information(3):
//	    - Type[1]
//	    - Length * 8bytes[1]
//	    - Prefix Length[1]
//	    - Flags[1]: LA......
//	    - Valid Lifetime[4]
//	    - Preferred Lifetime[4]
//	    - Reserved[4]
//	    - Prefix[16]
//	  - Option=MTU(5):
//	    - Type[1]
//	    - Length * 8bytes[1]
//	    - Reserved[2]
//	    - MTU[4]
//	  - Option=Source link-layer address(1):
//	    - Link-Layer Address[8/24]
//	  - Option=Recursive DNS Server(25):
//	    - Type[1]
//	    - Length * 8bytes[1]
//	    - Reserved[2]
//	    - Lifetime[4]
//	    - Addresses of IPv6 Recursive DNS Servers[16]
//
// TODO(a.garipov): Replace with an existing implementation from a dependency.
func createICMPv6RAPacket(params icmpv6RA) (data []byte, err error) {
	var lla []byte
	lla, err = hwAddrToLinkLayerAddr(params.sourceLinkLayerAddress)
	if err != nil {
		return nil, fmt.Errorf("converting source link layer address: %w", err)
	}

	// TODO(a.garipov): Don't use a magic constant here.  Refactor the code
	// and make all constants named instead of all those comments..
	data = make([]byte, 82+len(lla))
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

	copy(data[i:], lla) // Link-Layer Address[8/24]
	i += len(lla)

	// Option=Recursive DNS Server:

	data[i] = 25  // Type
	data[i+1] = 3 // Length
	i += 2
	binary.BigEndian.PutUint16(data[i:], 0) // Reserved[2]
	i += 2
	binary.BigEndian.PutUint32(data[i:], 3600) // Lifetime[4]
	i += 4
	copy(data[i:], params.recursiveDNSServer) // Addresses of IPv6 Recursive DNS Servers[16]

	return data, nil
}

// Init initializes RA module.
func (ra *raCtx) Init() (err error) {
	ra.stop.Store(0)
	ra.conn = nil
	if !ra.raAllowSLAAC && !ra.raSLAACOnly {
		return nil
	}

	log.Debug("dhcpv6 ra: source IP address: %s  DNS IP address: %s", ra.ipAddr, ra.dnsIPAddr)

	params := icmpv6RA{
		managedAddressConfiguration: !ra.raSLAACOnly,
		otherConfiguration:          !ra.raSLAACOnly,
		mtu:                         uint32(ra.iface.MTU),
		prefixLen:                   64,
		recursiveDNSServer:          ra.dnsIPAddr,
		sourceLinkLayerAddress:      ra.iface.HardwareAddr,
	}
	params.prefix = make([]byte, 16)
	copy(params.prefix, ra.prefixIPAddr[:8]) // /64

	var data []byte
	data, err = createICMPv6RAPacket(params)
	if err != nil {
		return fmt.Errorf("creating packet: %w", err)
	}

	ipAndScope := ra.ipAddr.String() + "%" + ra.ifaceName
	ra.conn, err = icmp.ListenPacket("ip6:ipv6-icmp", ipAndScope)
	if err != nil {
		return fmt.Errorf("dhcpv6 ra: icmp.ListenPacket: %w", err)
	}

	defer func() {
		if err != nil {
			err = errors.WithDeferred(err, ra.Close())
		}
	}()

	con6 := ra.conn.IPv6PacketConn()

	if err = con6.SetHopLimit(255); err != nil {
		return fmt.Errorf("dhcpv6 ra: SetHopLimit: %w", err)
	}

	if err = con6.SetMulticastHopLimit(255); err != nil {
		return fmt.Errorf("dhcpv6 ra: SetMulticastHopLimit: %w", err)
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
		log.Debug("dhcpv6 ra: starting to send periodic RouterAdvertisement packets")
		for ra.stop.Load() == 0 {
			_, err = con6.WriteTo(data, msg, addr)
			if err != nil {
				log.Error("dhcpv6 ra: WriteTo: %s", err)
			}
			time.Sleep(ra.packetSendPeriod)
		}
		log.Debug("dhcpv6 ra: loop exit")
	}()

	return nil
}

// Close closes the module.
func (ra *raCtx) Close() (err error) {
	log.Debug("dhcpv6 ra: closing")

	ra.stop.Store(1)

	if ra.conn != nil {
		return ra.conn.Close()
	}

	return nil
}
