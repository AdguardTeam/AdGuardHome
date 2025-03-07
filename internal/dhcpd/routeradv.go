package dhcpd

import (
	"encoding/binary"
	"fmt"
	"net"
	"slices"
	"sync/atomic"
	"time"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv6"
)

// raCtx is a context for the Router Advertisement logic.
type raCtx struct {
	// raAllowSLAAC is used to determine if the ICMP Router Advertisement
	// messages should be sent.
	//
	// If both raAllowSLAAC and raSLAACOnly are false, the Router Advertisement
	// messages aren't sent.
	raAllowSLAAC bool

	// raSLAACOnly is used to determine if the ICMP Router Advertisement
	// messages should set M and O flags, see RFC 4861, section 4.2.
	//
	// If both raAllowSLAAC and raSLAACOnly are false, the Router Advertisement
	// messages aren't sent.
	raSLAACOnly bool

	// ipAddr is an IP address used within the Source Link-Layer Address option.
	// See RFC 4861, section 4.6.1.
	ipAddr net.IP

	// dnsIPAddr is an IP address used within the DNS Server option.
	dnsIPAddr net.IP

	// prefixIPAddr is an IP address used within the Prefix Information option.
	// See RFC 4861, section 4.6.2.
	prefixIPAddr net.IP

	// ifaceName is the name of the interface used as a scope of the IP
	// addresses.
	ifaceName string

	// iface is the network interface used to send the ICMPv6 packets.
	iface *net.Interface

	// packetSendPeriod is the interval between sending the ICMPv6 packets.
	packetSendPeriod time.Duration

	// conn is the ICMPv6 socket.
	conn *icmp.PacketConn

	// stop is used to stop the packet sending loop.
	stop atomic.Value
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

// hwAddrToLinkLayerAddr clones the hardware address and returns it as a byte
// slice suitable for the Source Link-Layer Address option in the ICMPv6
// Router Advertisement packet.
//
// TODO(e.burkov):  Check if it's safe to use the original slice.
func hwAddrToLinkLayerAddr(hwa net.HardwareAddr) (lla []byte, err error) {
	err = netutil.ValidateMAC(hwa)
	if err != nil {
		// Don't wrap the error, because it already contains enough
		// context.
		return nil, err
	}

	return slices.Clone(hwa), nil
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
	lla, err := hwAddrToLinkLayerAddr(params.sourceLinkLayerAddress)
	if err != nil {
		return nil, fmt.Errorf("converting source link-layer address: %w", err)
	}

	// Calculate length of the source link-layer address option.  As per RFC
	// 4861, section 4.6.1, the length should be in units of 8 octets, including
	// the type and length fields.
	//
	// See https://datatracker.ietf.org/doc/html/rfc4861#section-4.6.1.
	srcLLAOptLen := len(lla) + 2
	// Make sure the value is rounded up to the nearest multiple of 8.
	srcLLAOptLenValue := (srcLLAOptLen + 7) / 8
	srcLLAPadLen := srcLLAOptLenValue*8 - srcLLAOptLen

	// TODO(a.garipov): Don't use a magic constant here.  Refactor the code
	// and make all constants named instead of all those comments.
	data = make([]byte, 80+srcLLAOptLen+srcLLAPadLen)
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

	data[i] = 1                         // Type
	data[i+1] = byte(srcLLAOptLenValue) // Length
	i += 2
	copy(data[i:], lla) // Link-Layer Address[8/24]
	i += len(lla) + srcLLAPadLen

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
