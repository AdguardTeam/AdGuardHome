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

// updateLease updates l in the database.  l must be valid and not expired.
func (iface *dhcpInterfaceV4) updateLease(
	ctx context.Context,
	l *Lease,
) (err error) {
	return iface.common.index.update(ctx, iface.common.logger, l, iface.common)
}

// respondOffer sends a DHCPOFFER message to the client.  req, fd, and l must
// not be nil.
func (iface *dhcpInterfaceV4) respondOffer(
	ctx context.Context,
	req *layers.DHCPv4,
	fd *frameData,
	l *Lease,
) {
	resp := iface.buildResponse(req, l, fd.device, layers.DHCPMsgTypeOffer)

	err := respond4(fd, resp)
	if err != nil {
		iface.common.logger.ErrorContext(ctx, "writing offer", "error", err)
	}
}

// respondACK sends a DHCPACK message to the client.  req, fd, and l must not be
// nil.
//
// TODO(e.burkov):  Implement according to RFC, answer to DHCPINFORM
// differently, when it's supported.
func (iface *dhcpInterfaceV4) respondACK(
	ctx context.Context,
	req *layers.DHCPv4,
	fd *frameData,
	l *Lease,
) {
	resp := iface.buildResponse(req, l, fd.device, layers.DHCPMsgTypeAck)
	if err := respond4(fd, resp); err != nil {
		iface.common.logger.ErrorContext(ctx, "writing ack", "error", err)
	}
}

// v4OptionMessageTypeNAK is a DHCP option for DHCPNAK message type.
var v4OptionMessageTypeNAK = layers.NewDHCPOption(
	layers.DHCPOptMessageType,
	[]byte{byte(layers.DHCPMsgTypeNak)},
)

// respondNAK constructs and sends a DHCPNAK message to the client.  req, fd,
// and resp must not be nil.
//
// See https://datatracker.ietf.org/doc/html/rfc2131#section-4.3.1.
func (iface *dhcpInterfaceV4) respondNAK(
	ctx context.Context,
	req *layers.DHCPv4,
	fd *frameData,
) {
	resp := &layers.DHCPv4{
		Operation:    layers.DHCPOpReply,
		HardwareType: layers.LinkTypeEthernet,
		HardwareLen:  uint8(len(req.ClientHWAddr)),
		Xid:          req.Xid,
		RelayAgentIP: req.RelayAgentIP,
		ClientHWAddr: req.ClientHWAddr,
		Options: layers.DHCPOptions{
			v4OptionMessageTypeNAK,
			layers.NewDHCPOption(layers.DHCPOptServerID, iface.gateway.AsSlice()),
			// TODO(e.burkov):  According to RFC 2131 we should add a message.
		},
	}

	if err := respond4(fd, resp); err != nil {
		iface.common.logger.ErrorContext(ctx, "writing nak", "error", err)
	}
}

// buildResponse builds a DHCP response message with the given message type.
// req and l must not be nil.  msgType must be one of:
//   - [layers.DHCPMsgTypeOffer]
//   - [layers.DHCPMsgTypeAck]
func (iface *dhcpInterfaceV4) buildResponse(
	req *layers.DHCPv4,
	l *Lease,
	nd NetworkDevice,
	msgType layers.DHCPMsgType,
) (resp *layers.DHCPv4) {
	resp = &layers.DHCPv4{
		Operation:    layers.DHCPOpReply,
		HardwareType: layers.LinkTypeEthernet,
		HardwareLen:  uint8(len(req.ClientHWAddr)),
		Xid:          req.Xid,
		ClientHWAddr: req.ClientHWAddr,
		YourClientIP: l.IP.AsSlice(),
	}

	resp.Options = append(
		resp.Options,
		layers.NewDHCPOption(layers.DHCPOptMessageType, []byte{byte(msgType)}),
		layers.NewDHCPOption(layers.DHCPOptServerID, nd.Addresses()[0].AsSlice()),
	)

	iface.appendLeaseTime(resp, l)
	iface.updateOptions(req, resp)

	// Add hostname option if the lease has a hostname.
	//
	// TODO(e.burkov):  Lease should always has a hostname, investigate when
	// it isn't the case.
	if l.Hostname != "" {
		resp.Options = append(
			resp.Options,
			layers.NewDHCPOption(layers.DHCPOptHostname, []byte(l.Hostname)),
		)
	}

	return resp
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
// addresses left, both l and err are nil.  mac must be a valid according to
// [netutil.ValidateMAC].
//
// TODO(e.burkov):  Pass the precalculated macKey.
func (iface *dhcpInterfaceV4) allocateLease(
	ctx context.Context,
	mac net.HardwareAddr,
) (l *Lease, err error) {
	for {
		l, err = iface.reserveLease(ctx, mac)
		if err != nil {
			return nil, err
		}

		var ok bool
		ok, err = iface.addrChecker.IsAvailable(l.IP)
		if err != nil {
			return nil, fmt.Errorf("checking address availability: %w", err)
		}

		if ok {
			iface.common.leases[macToKey(mac)] = l

			off, _ := iface.common.addrSpace.offset(l.IP)
			iface.common.leasedOffsets.set(off, true)

			return l, nil
		}

		iface.common.logger.DebugContext(ctx, "address not available", "ip", l.IP)

		err = iface.common.blockLease(ctx, l, iface.clock)
		if err != nil {
			return nil, fmt.Errorf("blocking unavailable address: %w", err)
		}
	}
}

// reserveLease reserves a lease for a client by its MAC-address.  l is nil if a
// new lease can't be allocated.  mac must be a valid according to
// [netutil.ValidateMAC].  index mutex must be locked.
func (iface *dhcpInterfaceV4) reserveLease(
	ctx context.Context,
	mac net.HardwareAddr,
) (l *Lease, err error) {
	nextIP := iface.common.nextIP()
	if nextIP != (netip.Addr{}) {
		l = &Lease{
			HWAddr: slices.Clone(mac),
			IP:     nextIP,
			Expiry: iface.clock.Now().Add(iface.common.leaseTTL),
		}

		return l, nil
	}

	l = iface.common.findExpiredLease(iface.clock.Now())
	if l == nil {
		return nil, errors.Error("no addresses available to lease")
	}

	// TODO(e.burkov):  Move validation from index methods into server's
	// methods and use index here.
	delete(iface.common.leases, macToKey(l.HWAddr))

	idx := iface.common.index
	delete(idx.byAddr, l.IP)
	delete(idx.byName, strings.ToLower(l.Hostname))

	err = idx.dbStore(ctx, iface.common.logger)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return nil, err
	}

	l.HWAddr = slices.Clone(mac)
	l.Hostname = ""
	l.IsStatic = false
	l.updateExpiry(iface.clock, iface.common.leaseTTL)

	iface.common.leases[macToKey(mac)] = l

	return l, nil
}

// updateAndRespond updates the lease and sends a DHCPACK or DHCPNAK response to
// the client according to the update result.  req must be a DHCPREQUEST
// message, lease, l, and fd must not be nil.
func (iface *dhcpInterfaceV4) updateAndRespond(
	ctx context.Context,
	l *slog.Logger,
	req *layers.DHCPv4,
	lease *Lease,
	fd *frameData,
) {
	lease.Hostname = cmp.Or(hostname4(req), lease.Hostname)

	err := iface.updateLease(ctx, lease)
	if err != nil {
		l.ErrorContext(ctx, "init-reboot request failed", slogutil.KeyError, err)
		iface.respondNAK(ctx, req, fd)

		return
	}

	iface.respondACK(ctx, req, fd, lease)
}

const (
	// IPv4DefaultTTL is the default Time to Live value in seconds as
	// recommended by RFC 1700.
	IPv4DefaultTTL = 64

	// IPProtoVersion is the IP internetwork general protocol version number as
	// defined by RFC 1700.
	IPProtoVersion = 4
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

// respond4 sends a DHCPv4 response.  fd and resp must not be nil.
func respond4(fd *frameData, resp *layers.DHCPv4) (err error) {
	// TODO(e.burkov):  Use pools for buffer and layers.
	buf := gopacket.NewSerializeBuffer()

	eth := &layers.Ethernet{
		SrcMAC:       fd.ether.SrcMAC,
		DstMAC:       fd.ether.DstMAC,
		EthernetType: layers.EthernetTypeIPv4,
	}
	ip := &layers.IPv4{
		Version:  IPProtoVersion,
		TTL:      IPv4DefaultTTL,
		SrcIP:    net.IPv4zero.To4(),
		DstIP:    net.IPv4bcast.To4(),
		Protocol: layers.IPProtocolUDP,
	}
	udp := &layers.UDP{
		SrcPort: ServerPortV4,
		DstPort: ClientPortV4,
	}
	_ = udp.SetNetworkLayerForChecksum(ip)

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
