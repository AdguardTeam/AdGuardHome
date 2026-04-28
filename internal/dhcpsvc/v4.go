package dhcpsvc

import (
	"cmp"
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"slices"
	"strings"
	"time"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/AdguardTeam/golibs/validate"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

// Port numbers for DHCPv4.
//
// See RFC 2131 Section 4.1.
const (
	// ServerPortV4 is the standard DHCPv4 server port.
	ServerPortV4 layers.UDPPort = 67

	// ClientPortV4 is the standard DHCPv4 client port.
	ClientPortV4 layers.UDPPort = 68
)

const (
	// IPv4DefaultTTL is the default Time to Live value in seconds as
	// recommended by RFC 1700.
	IPv4DefaultTTL = 64

	// IPProtoVersion is the IP internetwork general protocol version number as
	// defined by RFC 1700.
	IPProtoVersion = 4
)

// IPv4Config is the interface-specific configuration for DHCPv4.
type IPv4Config struct {
	// Clock is used to get current time.  It should not be nil.
	Clock timeutil.Clock

	// GatewayIP is the IPv4 address of the network's gateway.  It is used as
	// the default gateway for DHCP clients and also used for calculating the
	// network-specific broadcast address.  It should be a valid IPv4 address,
	// should be within the subnet, and should be outside the address range.
	GatewayIP netip.Addr

	// SubnetMask is the IPv4 subnet mask of the network.  It should be a valid
	// IPv4 CIDR (i.e. all 1s followed by all 0s).
	SubnetMask netip.Addr

	// RangeStart is the first address in the range to assign to DHCP clients.
	// It should be a valid IPv4 address, should be within the subnet, and
	// should be less or equal to RangeEnd.
	RangeStart netip.Addr

	// RangeEnd is the last address in the range to assign to DHCP clients.  It
	// should be a valid IPv4 address, should be within the subnet, and should
	// be greater or equal to RangeStart.
	RangeEnd netip.Addr

	// Options is the list of explicitly configured DHCP options to send to
	// clients.  Options with nil Data field are removed from responses.
	//
	// TODO(e.burkov):  Validate.
	Options layers.DHCPOptions

	// LeaseDuration is the TTL of a DHCP lease.  It should be positive.
	LeaseDuration time.Duration

	// Enabled is the state of the DHCPv4 service, whether it is enabled or not
	// on the specific interface.
	Enabled bool
}

// type check
var _ validate.Interface = (*IPv4Config)(nil)

// Validate implements the [validate.Interface] interface for *IPv4Config.
func (c *IPv4Config) Validate() (err error) {
	if c == nil {
		return errors.ErrNoValue
	} else if !c.Enabled {
		// Don't validate the configuration for disabled interface.
		return nil
	}

	errs := []error{
		validate.NotNilInterface("clock", c.Clock),
		validate.Positive("lease duration", c.LeaseDuration),
	}

	errs = c.validateSubnet(errs)

	return errors.Join(errs...)
}

// validateSubnet validates the subnet configuration.
//
// TODO(e.burkov):  Use [validate].
func (c *IPv4Config) validateSubnet(orig []error) (errs []error) {
	errs = orig

	if !c.GatewayIP.Is4() {
		err := newMustErr("gateway ip", "be a valid ipv4", c.GatewayIP)
		errs = append(errs, err)
	}

	if !c.SubnetMask.Is4() {
		err := newMustErr("subnet mask", "be a valid ipv4 cidr mask", c.SubnetMask)
		errs = append(errs, err)
	}

	if !c.RangeStart.Is4() {
		err := newMustErr("range start", "be a valid ipv4", c.RangeStart)
		errs = append(errs, err)
	}

	if !c.RangeEnd.Is4() {
		err := newMustErr("range end", "be a valid ipv4", c.RangeEnd)
		errs = append(errs, err)
	}

	maskLen, _ := net.IPMask(c.SubnetMask.AsSlice()).Size()
	subnet := netip.PrefixFrom(c.GatewayIP, maskLen)

	switch {
	case !subnet.Contains(c.RangeStart):
		errs = append(errs, fmt.Errorf("range start %s is not within %s", c.RangeStart, subnet))
	case !subnet.Contains(c.RangeEnd):
		errs = append(errs, fmt.Errorf("range end %s is not within %s", c.RangeEnd, subnet))
	}

	addrSpace, err := newIPRange(c.RangeStart, c.RangeEnd)
	if err != nil {
		errs = append(errs, err)
	} else if addrSpace.contains(c.GatewayIP) {
		errs = append(errs, fmt.Errorf("gateway ip %s in the ip range %s", c.GatewayIP, addrSpace))
	}

	return errs
}

// dhcpInterfaceV4 is a DHCP interface for IPv4 address family.
type dhcpInterfaceV4 struct {
	// common is the common part of any network interface within the DHCP
	// server.
	common *netInterface

	// clock used to get current time.
	//
	// TODO(e.burkov):  Move to [netInterface].
	clock timeutil.Clock

	// addrChecker checks addresses for availability.
	addrChecker addressChecker

	// gateway is the IP address of the network gateway.
	gateway netip.Addr

	// subnet is the network subnet of the interface.
	subnet netip.Prefix

	// implicitOpts are the options listed in Appendix A of RFC 2131 and
	// initialized with default values.  It must not have intersections with
	// explicitOpts.
	implicitOpts layers.DHCPOptions

	// explicitOpts are the user-configured options.  It must not have
	// intersections with implicitOpts.  Options with nil Data field are removed
	// from responses.
	explicitOpts layers.DHCPOptions
}

// newDHCPInterfaceV4 creates a new DHCP interface for IPv4 address family with
// the given configuration.  If the interface is disabled, it returns nil.
// baseLogger must not be nil, name must be a valid network interface name, conf
// must be valid.
func (srv *DHCPServer) newDHCPInterfaceV4(
	ctx context.Context,
	baseLogger *slog.Logger,
	name string,
	conf *IPv4Config,
) (iface *dhcpInterfaceV4) {
	if !conf.Enabled {
		baseLogger.DebugContext(ctx, "disabled")

		return nil
	}

	// TODO(e.burkov):  Add a helper for converting [netip.Addr] to subnet mask
	// to [netutil].
	maskLen, _ := net.IPMask(conf.SubnetMask.AsSlice()).Size()

	// Ignore the error since it's already checked in [IPv4Config.Validate].
	addrSpace, _ := newIPRange(conf.RangeStart, conf.RangeEnd)

	iface = &dhcpInterfaceV4{
		// TODO(e.burkov):  Use an ICMP implementation.
		addrChecker: noopAddressChecker{},
		gateway:     conf.GatewayIP,
		clock:       conf.Clock,
		subnet:      netip.PrefixFrom(conf.GatewayIP, maskLen),
		common: &netInterface{
			logger:        baseLogger,
			indexMu:       srv.leasesMu,
			index:         srv.leases,
			leases:        map[macKey]*Lease{},
			leasedOffsets: newBitSet(),
			name:          name,
			addrSpace:     addrSpace,
			leaseTTL:      conf.LeaseDuration,
		},
	}
	iface.implicitOpts, iface.explicitOpts = conf.options(ctx, baseLogger)

	return iface
}

// updateLease updates lease in the database.  lease must be valid and not
// expired.
func (iface *dhcpInterfaceV4) updateLease(ctx context.Context, lease *Lease) (err error) {
	return iface.common.index.update(ctx, iface.common.logger, lease, iface.common)
}

// respondOffer sends a DHCPOFFER message to the client.  idOpt is expected to
// be the value of the DHCP option Client Identifier, nil if not present.  req
// and lease must not be nil, fd must be valid
//
// TODO(e.burkov):  Consider merging with [respondACK].
func (iface *dhcpInterfaceV4) respondOffer(
	ctx context.Context,
	req *layers.DHCPv4,
	fd *frameData4,
	lease *Lease,
	idOpt []byte,
) {
	opts := newRespOptions(layers.DHCPMsgTypeOffer, fd, idOpt)
	opts = iface.appendTimeOptions(opts, lease)
	opts = appendHostnameOption(opts, lease.Hostname)
	opts = iface.appendRequestedOptions(opts, req)

	resp := buildResponse(req, lease.IP, net.IPv4zero, req.Flags, opts)
	err := respond4(fd, req, resp)
	if err != nil {
		iface.common.logger.ErrorContext(ctx, "writing offer", "error", err)
	}
}

// respondACK sends a DHCPACK message to the client.  idOpt is expected to be
// the value of the DHCP option Client Identifier, nil if not present.  req and
// lease must not be nil, fd must be valid.
//
// TODO(e.burkov):  Implement according to RFC, answer to DHCPINFORM
// differently, when it's supported.
func (iface *dhcpInterfaceV4) respondACK(
	ctx context.Context,
	req *layers.DHCPv4,
	fd *frameData4,
	lease *Lease,
	idOpt []byte,
) {
	opts := newRespOptions(layers.DHCPMsgTypeAck, fd, idOpt)
	opts = iface.appendTimeOptions(opts, lease)
	opts = iface.appendRequestedOptions(opts, req)
	opts = appendHostnameOption(opts, lease.Hostname)

	resp := buildResponse(req, lease.IP, req.ClientIP, req.Flags, opts)
	err := respond4(fd, req, resp)
	if err != nil {
		iface.common.logger.ErrorContext(ctx, "writing ack", "error", err)
	}
}

// respondNAK constructs and sends a DHCPNAK message to the client.  idOpt is
// expected to be the value of the DHCP option Client Identifier, nil if not
// present.  req and resp must not be nil, fd must be valid.
//
// See https://datatracker.ietf.org/doc/html/rfc2131#section-4.3.1.
func (iface *dhcpInterfaceV4) respondNAK(
	ctx context.Context,
	req *layers.DHCPv4,
	fd *frameData4,
	idOpt []byte,
) {
	// TODO(e.burkov):  According to RFC 2131 we should add a message.
	opts := newRespOptions(layers.DHCPMsgTypeNak, fd, idOpt)

	// If 'giaddr' is set in the DHCPREQUEST message, the client is on a
	// different subnet.  The server MUST set the broadcast bit in the DHCPNAK,
	// so that the relay agent will broadcast the DHCPNAK to the client, because
	// the client may not have a correct network address or subnet mask, and the
	// client may not be answering ARP requests.
	flags := req.Flags
	if isSpecified(req.RelayAgentIP) {
		flags = flags | FlagsBroadcast
	}

	resp := buildResponse(req, netip.Addr{}, net.IPv4zero, flags, opts)
	err := respond4(fd, req, resp)
	if err != nil {
		iface.common.logger.ErrorContext(ctx, "writing nak", "error", err)
	}
}

// buildResponse constructs a DHCPv4 response message.  req must not be nil.
// Note that in order to be a valid response, opts must contain some mandatory
// options, e.g. a message type.
func buildResponse(
	req *layers.DHCPv4,
	yiaddr netip.Addr,
	ciaddr net.IP,
	flags uint16,
	opts layers.DHCPOptions,
) (resp *layers.DHCPv4) {
	return &layers.DHCPv4{
		Operation:    layers.DHCPOpReply,
		HardwareType: layers.LinkTypeEthernet,
		HardwareLen:  uint8(len(req.ClientHWAddr)),
		Xid:          req.Xid,
		Secs:         0,
		Flags:        flags,
		ClientIP:     ciaddr,
		RelayAgentIP: req.RelayAgentIP,
		ClientHWAddr: req.ClientHWAddr,
		YourClientIP: yiaddr.AsSlice(),
		Options:      opts,
	}
}

// dhcpInterfacesV4 is a slice of network interfaces of IPv4 address family.
type dhcpInterfacesV4 []*dhcpInterfaceV4

// find returns the first network interface within ifaces containing ip.  It
// returns false if there is no such interface.  ip must be valid.
func (ifaces dhcpInterfacesV4) find(ip netip.Addr) (iface4 *netInterface, ok bool) {
	i := slices.IndexFunc(ifaces, func(iface *dhcpInterfaceV4) (contains bool) {
		return iface.subnet.Contains(ip)
	})
	if i < 0 {
		return nil, false
	}

	return ifaces[i].common, true
}

// allocateLease allocates a new lease for the MAC address.  If there are no IP
// addresses left, both lease and err are nil.  mac must be a valid according to
// [netutil.ValidateMAC].
//
// TODO(e.burkov):  Pass the precalculated macKey.
func (iface *dhcpInterfaceV4) allocateLease(
	ctx context.Context,
	mac net.HardwareAddr,
) (lease *Lease, err error) {
	for {
		lease, err = iface.reserveLease(ctx, mac)
		if err != nil {
			return nil, err
		}

		var ok bool
		ok, err = iface.addrChecker.IsAvailable(lease.IP)
		if err != nil {
			return nil, fmt.Errorf("checking address availability: %w", err)
		}

		if ok {
			iface.common.leases[macToKey(mac)] = lease

			off, _ := iface.common.addrSpace.offset(lease.IP)
			iface.common.leasedOffsets.set(off, true)

			return lease, nil
		}

		iface.common.logger.DebugContext(ctx, "address not available", "ip", lease.IP)

		err = iface.common.blockLease(ctx, lease, iface.clock)
		if err != nil {
			return nil, fmt.Errorf("blocking unavailable address: %w", err)
		}
	}
}

// reserveLease reserves a lease for a client by its MAC-address.  lease is nil
// if a new lease can't be allocated.  mac must be a valid according to
// [netutil.ValidateMAC].  index mutex must be locked.
func (iface *dhcpInterfaceV4) reserveLease(
	ctx context.Context,
	mac net.HardwareAddr,
) (lease *Lease, err error) {
	nextIP := iface.common.nextIP()
	if nextIP != (netip.Addr{}) {
		lease = &Lease{
			HWAddr: slices.Clone(mac),
			IP:     nextIP,
			Expiry: iface.clock.Now().Add(iface.common.leaseTTL),
		}

		return lease, nil
	}

	lease = iface.common.findExpiredLease(iface.clock.Now())
	if lease == nil {
		return nil, errors.Error("no addresses available to lease")
	}

	// TODO(e.burkov):  Move validation from index methods into server's
	// methods and use index here.
	delete(iface.common.leases, macToKey(lease.HWAddr))

	idx := iface.common.index
	delete(idx.byAddr, lease.IP)
	delete(idx.byName, strings.ToLower(lease.Hostname))

	err = idx.dbStore(ctx, iface.common.logger)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return nil, err
	}

	lease.HWAddr = slices.Clone(mac)
	lease.Hostname = ""
	lease.IsStatic = false
	lease.updateExpiry(iface.clock, iface.common.leaseTTL)

	iface.common.leases[macToKey(mac)] = lease

	return lease, nil
}

// updateAndRespond updates the lease and sends a DHCPACK or DHCPNAK response to
// the client according to the update result.  idOpt is an expected to be the
// value of the DHCP option Client Identifier, nil if not present.  req must be
// a DHCPREQUEST message, lease, and l must not be nil, fd must be valid.
func (iface *dhcpInterfaceV4) updateAndRespond(
	ctx context.Context,
	l *slog.Logger,
	req *layers.DHCPv4,
	lease *Lease,
	fd *frameData4,
	idOpt []byte,
) {
	lease.Hostname = cmp.Or(hostname4(req), lease.Hostname)

	err := iface.updateLease(ctx, lease)
	if err != nil {
		l.ErrorContext(ctx, "init-reboot request failed", slogutil.KeyError, err)
		iface.respondNAK(ctx, req, fd, idOpt)

		return
	}

	iface.respondACK(ctx, req, fd, lease, idOpt)
}

// FlagsBroadcast is the DHCPv4 message flags field with the broadcast bit set.
const FlagsBroadcast uint16 = 1 << 15

// respond4 sends a DHCPv4 response.  req and resp must not be nil, fd must be
// valid.
func respond4(fd *frameData4, req, resp *layers.DHCPv4) (err error) {
	// TODO(e.burkov):  Use pools for buffer and layers.
	buf := gopacket.NewSerializeBuffer()

	eth := &layers.Ethernet{
		SrcMAC:       fd.ether.DstMAC,
		DstMAC:       fd.ether.SrcMAC,
		EthernetType: layers.EthernetTypeIPv4,
	}

	ip, udp := newIPv4UDPLayers(fd, req, resp)

	opts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}

	err = gopacket.SerializeLayers(buf, opts, eth, ip, udp, resp)
	if err != nil {
		return fmt.Errorf("constructing dhcp v4 response: %w", err)
	}

	return fd.device.WritePacketData(buf.Bytes())
}

// newIPv4UDPLayers creates new UDP and IP layers for DHCPv4 response.  req and
// resp must not be nil, fd must be valid.
func newIPv4UDPLayers(fd *frameData4, req, resp *layers.DHCPv4) (ip *layers.IPv4, udp *layers.UDP) {
	var dstIP net.IP
	dstPort := ClientPortV4
	switch {
	case isSpecified(req.RelayAgentIP.To4()):
		// If the 'giaddr' field in a DHCP message from a client is non-zero,
		// the server sends any return messages to the 'DHCP server' port on the
		// BOOTP relay agent whose address appears in 'giaddr'.
		dstIP, dstPort = req.RelayAgentIP.To4(), ServerPortV4
	case isSpecified(req.ClientIP.To4()):
		// If the 'giaddr' field is zero and the 'ciaddr' field is nonzero, then
		// the server unicasts DHCPOFFER and DHCPACK messages to the address in
		// 'ciaddr'.
		dstIP = req.ClientIP.To4()
	case req.Flags&FlagsBroadcast != 0:
		// If 'giaddr' is zero and 'ciaddr' is zero, and the broadcast bit is
		// set, then the server broadcasts DHCPOFFER and DHCPACK messages to
		// 0xffffffff.
		dstIP = net.IPv4bcast.To4()
	case isSpecified(resp.YourClientIP.To4()):
		// If the broadcast bit is not set and 'giaddr' is zero and 'ciaddr' is
		// zero, then the server unicasts DHCPOFFER and DHCPACK messages to the
		// client's hardware address and 'yiaddr' address.
		dstIP = resp.YourClientIP.To4()
	default:
		// Unicast to the client's hardware address only.
		dstIP = netip.IPv4Unspecified().AsSlice()
	}

	ip = &layers.IPv4{
		Version:  IPProtoVersion,
		TTL:      IPv4DefaultTTL,
		SrcIP:    fd.localAddr.AsSlice(),
		DstIP:    dstIP,
		Protocol: layers.IPProtocolUDP,
	}
	udp = &layers.UDP{
		SrcPort: ServerPortV4,
		DstPort: dstPort,
	}

	// It only returns an error if the network layer is not an IP layer.
	_ = udp.SetNetworkLayerForChecksum(ip)

	return ip, udp
}

// isSpecified checks if the IP is not nil and not unspecified.
func isSpecified(ip net.IP) (ok bool) {
	return ip != nil && !ip.IsUnspecified()
}
