package dhcpd

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"net/netip"
	"slices"
	"sync"
	"time"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv6"
)

const (
	defaultRARouterLifetimeSeconds = uint16(1800)
	defaultRAObservePeriod         = 5 * time.Second
)

// raObserver refreshes IPv6 Router Advertisement state from interface data.
type raObserver func(ctx context.Context) (obs raObservation, err error)

// raActivePrefixHandler is called when the current active prefix or the set of
// advertised prefixes changes.
type raActivePrefixHandler func(active *raPrefixSnapshot, advertised []prefixPIO)

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

	// ifaceName is the name of the interface used as a scope of the IP
	// addresses.
	ifaceName string

	// iface is the network interface used to send the ICMPv6 packets.
	iface *net.Interface

	// packetSendPeriod is the interval between sending the ICMPv6 packets.
	packetSendPeriod time.Duration

	// observePeriod is the interval between refreshing interface-derived RA
	// state.
	observePeriod time.Duration

	// conn is the ICMPv6 socket.
	conn *icmp.PacketConn
	// connSourceAddr is the source address currently bound to conn.
	connSourceAddr netip.Addr

	// observe refreshes interface-derived RA state.  It is nil for static mode.
	observe raObserver

	// onActivePrefixChange applies runtime changes on active-prefix updates.
	onActivePrefixChange raActivePrefixHandler

	// onStateRefresh applies extra state reconciliation after a successful
	// observation merge and before change detection.
	onStateRefresh func(now time.Time, st *raState)

	state              raState
	lastActiveSnapshot *raPrefixSnapshot
	lastAdvertised     []prefixPIO

	cancel func()
	wg     sync.WaitGroup
}

// icmpv6RA describes the contents of one Router Advertisement packet.
type icmpv6RA struct {
	managedAddressConfiguration bool
	otherConfiguration          bool
	prefixes                    []prefixPIO
	sourceLinkLayerAddress      net.HardwareAddr
	recursiveDNSServer          netip.Addr
	mtu                         uint32
}

// hwAddrToLinkLayerAddr clones the hardware address and returns it as a byte
// slice suitable for the Source Link-Layer Address option in the ICMPv6
// Router Advertisement packet.
func hwAddrToLinkLayerAddr(hwa net.HardwareAddr) (lla []byte, err error) {
	err = netutil.ValidateMAC(hwa)
	if err != nil {
		return nil, err
	}

	return slices.Clone(hwa), nil
}

// createICMPv6RAPacket creates an ICMPv6 Router Advertisement packet with the
// supplied prefix and DNS options.
func createICMPv6RAPacket(params icmpv6RA) (data []byte, err error) {
	lla, err := hwAddrToLinkLayerAddr(params.sourceLinkLayerAddress)
	if err != nil {
		return nil, fmt.Errorf("converting source link-layer address: %w", err)
	}

	srcLLAOptLen := len(lla) + 2
	srcLLAOptLenValue := (srcLLAOptLen + 7) / 8
	srcLLAPadLen := srcLLAOptLenValue*8 - srcLLAOptLen

	dataLen := 16 + len(params.prefixes)*32 + 8 + srcLLAOptLen + srcLLAPadLen
	if params.recursiveDNSServer.IsValid() {
		dataLen += 24
	}

	data = make([]byte, dataLen)
	i := 0

	// ICMPv6 header.
	data[i] = 134
	data[i+1] = 0
	i += 4

	// Router Advertisement header.
	data[i] = 64
	i++

	if params.managedAddressConfiguration {
		data[i] |= 0x80
	}
	if params.otherConfiguration {
		data[i] |= 0x40
	}
	i++

	binary.BigEndian.PutUint16(data[i:], defaultRARouterLifetimeSeconds)
	i += 2
	i += 8

	for _, prefix := range params.prefixes {
		data[i] = 3
		data[i+1] = 4
		i += 2

		data[i] = byte(prefix.Prefix.Bits())
		i++
		data[i] = 0xc0
		i++

		binary.BigEndian.PutUint32(data[i:], prefix.ValidSec)
		i += 4
		binary.BigEndian.PutUint32(data[i:], prefix.PreferredSec)
		i += 4
		i += 4

		addr := prefix.Prefix.Masked().Addr().As16()
		copy(data[i:], addr[:])
		i += len(addr)
	}

	// MTU option.
	data[i] = 5
	data[i+1] = 1
	i += 4
	binary.BigEndian.PutUint32(data[i:], params.mtu)
	i += 4

	// Source Link-Layer Address option.
	data[i] = 1
	data[i+1] = byte(srcLLAOptLenValue)
	i += 2
	copy(data[i:], lla)
	i += len(lla) + srcLLAPadLen

	if params.recursiveDNSServer.IsValid() {
		data[i] = 25
		data[i+1] = 3
		i += 4
		binary.BigEndian.PutUint32(data[i:], defaultRARDNSSLifetimeSeconds)
		i += 4

		addr := params.recursiveDNSServer.As16()
		copy(data[i:], addr[:])
	}

	return data, nil
}

// sendingEnabled reports whether RA packets should be transmitted.
func (ra *raCtx) sendingEnabled() (ok bool) {
	return ra.raAllowSLAAC || ra.raSLAACOnly
}

// Init initializes the Router Advertisement state loop.
func (ra *raCtx) Init(initial raState) (err error) {
	ra.state = initial
	ra.conn = nil
	now := time.Now()
	ra.lastActiveSnapshot = clonePrefixSnapshot(initial.activeSnapshot(now))
	ra.lastAdvertised = clonePIOs(initial.pios(now))

	if !ra.sendingEnabled() && ra.observe == nil {
		return nil
	}

	if ra.sendingEnabled() {
		sourceAddr, _ := ra.state.sourceAndRDNSS()
		err = ra.ensureConn(sourceAddr)
		if err != nil {
			return err
		} else if ra.conn == nil && ra.observe == nil {
			return fmt.Errorf("dhcpv6 ra: no source address")
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	ra.cancel = cancel
	ra.wg.Add(1)
	go ra.loop(ctx)

	return nil
}

// ensureConn initializes or refreshes the ICMPv6 socket used to send Router
// Advertisements.
func (ra *raCtx) ensureConn(sourceAddr netip.Addr) (err error) {
	if !sourceAddr.IsValid() {
		if ra.conn != nil {
			err = ra.conn.Close()
		}

		ra.conn = nil
		ra.connSourceAddr = netip.Addr{}

		return err
	}

	if ra.conn != nil && ra.connSourceAddr == sourceAddr {
		return nil
	}

	if ra.conn != nil {
		err = ra.conn.Close()
		ra.conn = nil
		ra.connSourceAddr = netip.Addr{}
		if err != nil {
			return fmt.Errorf("closing previous icmp listener: %w", err)
		}
	}

	ipAndScope := sourceAddr.String() + "%" + ra.ifaceName
	ra.conn, err = icmp.ListenPacket("ip6:ipv6-icmp", ipAndScope)
	if err != nil {
		return fmt.Errorf("dhcpv6 ra: icmp.ListenPacket: %w", err)
	}

	defer func() {
		if err != nil {
			err = errors.WithDeferred(err, ra.conn.Close())
		}
	}()

	con6 := ra.conn.IPv6PacketConn()
	if err = con6.SetHopLimit(255); err != nil {
		return fmt.Errorf("dhcpv6 ra: SetHopLimit: %w", err)
	}

	if err = con6.SetMulticastHopLimit(255); err != nil {
		return fmt.Errorf("dhcpv6 ra: SetMulticastHopLimit: %w", err)
	}

	ra.connSourceAddr = sourceAddr

	return nil
}

// loop runs the Router Advertisement send and observation ticks.
func (ra *raCtx) loop(ctx context.Context) {
	defer ra.wg.Done()

	var sendTicker *time.Ticker
	if ra.sendingEnabled() {
		sendTicker = time.NewTicker(ra.packetSendPeriod)
		defer sendTicker.Stop()
	}

	var observeTicker *time.Ticker
	if ra.observe != nil {
		period := ra.observePeriod
		if period <= 0 {
			period = defaultRAObservePeriod
		}

		observeTicker = time.NewTicker(period)
		defer observeTicker.Stop()
	}

	log.Debug("dhcpv6 ra: starting router advertisement loop")
	for {
		select {
		case <-ctx.Done():
			log.Debug("dhcpv6 ra: loop exit")

			return
		case <-tickerC(sendTicker):
			ra.sendPacket()
		case <-tickerC(observeTicker):
			ra.refresh(ctx)
		}
	}
}

// refresh updates the interface-derived RA state.
func (ra *raCtx) refresh(ctx context.Context) {
	if ra.observe == nil {
		return
	}

	obs, err := ra.observe(ctx)
	if err != nil {
		log.Error("dhcpv6 ra: refreshing prefix state: %s", err)

		return
	}

	now := time.Now()
	_ = ra.state.merge(obs, now)
	if ra.onStateRefresh != nil {
		ra.onStateRefresh(now, &ra.state)
	}
	ra.syncStateChange(now, true)
}

// sendPacket rebuilds and sends the current Router Advertisement packet.
func (ra *raCtx) sendPacket() {
	now := time.Now()
	ra.syncStateChange(now, false)

	sourceAddr, rdnssAddr := ra.state.sourceAndRDNSS()
	err := ra.ensureConn(sourceAddr)
	if err != nil {
		log.Error("dhcpv6 ra: opening listener: %s", err)

		return
	} else if ra.conn == nil {
		return
	}

	if !sourceAddr.IsValid() {
		return
	}

	pios := ra.state.pios(now)
	if len(pios) == 0 {
		return
	}

	pkt, err := createICMPv6RAPacket(icmpv6RA{
		managedAddressConfiguration: !ra.raSLAACOnly,
		otherConfiguration:          !ra.raSLAACOnly,
		prefixes:                    pios,
		recursiveDNSServer:          rdnssAddr,
		sourceLinkLayerAddress:      ra.iface.HardwareAddr,
		mtu:                         uint32(ra.iface.MTU),
	})
	if err != nil {
		log.Error("dhcpv6 ra: creating packet: %s", err)

		return
	}

	msg := &ipv6.ControlMessage{
		HopLimit: 255,
		Src:      net.IP(sourceAddr.AsSlice()),
		IfIndex:  ra.iface.Index,
	}
	addr := &net.UDPAddr{IP: net.ParseIP("ff02::1")}
	_, err = ra.conn.IPv6PacketConn().WriteTo(pkt, msg, addr)
	if err != nil {
		log.Error("dhcpv6 ra: WriteTo: %s", err)
	}
}

// Close closes the Router Advertisement module.
func (ra *raCtx) Close() (err error) {
	log.Debug("dhcpv6 ra: closing")

	if ra.cancel != nil {
		ra.cancel()
		ra.wg.Wait()
		ra.cancel = nil
	}

	if ra.conn != nil {
		err = ra.conn.Close()
		ra.conn = nil
		ra.connSourceAddr = netip.Addr{}
	}

	return err
}

// tickerC safely returns the channel for t.
func tickerC(t *time.Ticker) (c <-chan time.Time) {
	if t == nil {
		return nil
	}

	return t.C
}

// syncStateChange updates the DHCPv6-facing state if the active prefix or the
// advertised prefix set changed.  When compareLifetimes is true, preferred and
// valid lifetime changes are treated as state changes as well.
func (ra *raCtx) syncStateChange(now time.Time, compareLifetimes bool) {
	active := ra.state.activeSnapshot(now)
	advertised := ra.state.pios(now)
	changed := !sameActivePrefix(ra.lastActiveSnapshot, active) ||
		!sameAdvertisedPIOSet(ra.lastAdvertised, advertised, compareLifetimes)

	ra.lastActiveSnapshot = clonePrefixSnapshot(active)
	ra.lastAdvertised = clonePIOs(advertised)

	if !changed || ra.onActivePrefixChange == nil {
		return
	}

	ra.onActivePrefixChange(active, advertised)
}

// sameAdvertisedPIOSet reports whether a and b advertise the same set of
// prefixes.  When compareLifetimes is true, preferred and valid lifetime
// changes are also treated as differences.
func sameAdvertisedPIOSet(a, b []prefixPIO, compareLifetimes bool) (ok bool) {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i].Prefix != b[i].Prefix {
			return false
		}
		if compareLifetimes &&
			(a[i].PreferredSec != b[i].PreferredSec || a[i].ValidSec != b[i].ValidSec) {
			return false
		}
	}

	return true
}

// clonePrefixSnapshot returns a shallow clone of snap.
func clonePrefixSnapshot(snap *raPrefixSnapshot) (cloned *raPrefixSnapshot) {
	if snap == nil {
		return nil
	}

	clone := *snap

	return &clone
}

// clonePIOs returns a shallow clone of pios.
func clonePIOs(pios []prefixPIO) (cloned []prefixPIO) {
	if pios == nil {
		return nil
	}

	return slices.Clone(pios)
}
