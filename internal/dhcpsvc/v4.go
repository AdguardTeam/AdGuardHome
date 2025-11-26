package dhcpsvc

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"slices"
	"time"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/AdguardTeam/golibs/validate"
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

// commitLease updates the lease in database, using new hostname if it's valid.
func (iface *dhcpInterfaceV4) commitLease(ctx context.Context, l *Lease, hostname string) {
	// TODO(e.burkov):  Implement.
}

// respondOffer sends a DHCPOFFER message to the client.  rw, req, and l must
// not be nil.
func (iface *dhcpInterfaceV4) respondOffer(
	ctx context.Context,
	rw responseWriter4,
	req *layers.DHCPv4,
	l *Lease,
) {
	resp := iface.buildResponse(req, l, layers.DHCPMsgTypeOffer)
	if err := rw.write(ctx, resp); err != nil {
		iface.common.logger.ErrorContext(ctx, "writing offer", "error", err)
	}
}

// respondACK sends a DHCPACK message to the client.
//
// TODO(e.burkov):  Implement according to RFC, answer to DHCPINFORM
// differently, when it's supported.
func (iface *dhcpInterfaceV4) respondACK(
	ctx context.Context,
	rw responseWriter4,
	req *layers.DHCPv4,
	l *Lease,
) {
	resp := iface.buildResponse(req, l, layers.DHCPMsgTypeAck)
	if err := rw.write(ctx, resp); err != nil {
		iface.common.logger.ErrorContext(ctx, "writing ack", "error", err)
	}
}

// v4OptionMessageTypeNAK is a DHCP option for DHCPNAK message type.
var v4OptionMessageTypeNAK = layers.NewDHCPOption(
	layers.DHCPOptMessageType,
	[]byte{byte(layers.DHCPMsgTypeNak)},
)

// respondNAK constructs and sends a DHCPNAK message to the client.  rw and req
// must not be nil.
//
// See https://datatracker.ietf.org/doc/html/rfc2131#section-4.3.1.
func (iface *dhcpInterfaceV4) respondNAK(
	ctx context.Context,
	rw responseWriter4,
	req *layers.DHCPv4,
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

	if err := rw.write(ctx, resp); err != nil {
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
		layers.NewDHCPOption(layers.DHCPOptServerID, iface.gateway.AsSlice()),
	)

	appendLeaseTime(resp, iface.common.leaseTTL)
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

// handleDiscover handles messages of type DHCPDISCOVER.  rw and req must not be
// nil.
func (iface *dhcpInterfaceV4) handleDiscover(
	ctx context.Context,
	rw responseWriter4,
	req *layers.DHCPv4,
) {
	l := iface.common.logger
	mac := req.ClientHWAddr
	mk := macToKey(mac)

	// Check if there's an existing lease for this MAC address.
	iface.common.indexMu.Lock()
	defer iface.common.indexMu.Unlock()

	lease, hasLease := iface.common.leases[mk]
	if hasLease {
		reqIP, hasReqIP := requestedIPv4(req)
		if hasReqIP && reqIP != lease.IP {
			l.DebugContext(ctx, "different requested ip", "requested", reqIP, "lease", lease.IP)
		}

		iface.respondOffer(ctx, rw, req, lease)

		return
	}

	// TODO(e.burkov):  Allocate a new lease.
	lease, err := iface.allocateLease(ctx, mac)
	if err != nil {
		l.ErrorContext(ctx, "allocating a lease", "error", err)

		return
	}

	// Send DHCPOFFER with new lease.
	iface.respondOffer(ctx, rw, req, lease)
}

// handleSelecting handles messages of type DHCPREQUEST in SELECTING state.  req
// must contain a server identifier option that matches the iface's subnet, and
// client IP address must be empty or unspecified, and requested IP address
// must be filled in with the yiaddr value from the chosen DHCPOFFER.  rw must
// not be nil, reqIP must be a valid IPv4 address.
func (iface *dhcpInterfaceV4) handleSelecting(
	ctx context.Context,
	rw responseWriter4,
	req *layers.DHCPv4,
	reqIP netip.Addr,
) {
	l := iface.common.logger

	if !reqIP.Is4() {
		l.DebugContext(ctx, "bad requested address", "requestedip", reqIP)

		return
	}

	ciaddr, ok := netip.AddrFromSlice(req.ClientIP)
	if ok && !ciaddr.IsUnspecified() {
		l.DebugContext(ctx, "non-zero ciaddr in selecting request", "ciaddr", ciaddr)

		return
	}

	mac := req.ClientHWAddr
	mk := macToKey(mac)

	iface.common.indexMu.Lock()
	defer iface.common.indexMu.Unlock()

	lease, hasLease := iface.common.leases[mk]
	if !hasLease {
		l.DebugContext(ctx, "no reserved lease", "clienthwaddr", mac)
		iface.respondNAK(ctx, rw, req)

		return
	}

	if lease.IP != reqIP {
		l.DebugContext(ctx, "selecting request mismatched", "requested", reqIP, "lease", lease.IP)
		iface.respondNAK(ctx, rw, req)

		return
	}

	// Commit the lease and send ACK.
	iface.commitLease(ctx, lease, hostname4(req))
	iface.respondACK(ctx, rw, req, lease)
}

// handleInitReboot handles messages of type DHCPREQUEST in INIT-REBOOT state.
// req must contain a client IP address option that matches the iface's subnet,
// and requested IP address option must be filled in with the client's IP
// address.  Also req must contain a valid chaddr according to
// [netutil.ValidateMAC].  rw must not be nil, reqIP must be a valid IPv4
// address.
func (iface *dhcpInterfaceV4) handleInitReboot(
	ctx context.Context,
	rw responseWriter4,
	req *layers.DHCPv4,
	reqIP netip.Addr,
) {
	l := iface.common.logger

	if !reqIP.Is4() {
		l.DebugContext(ctx, "bad requested address", "requestedip", reqIP)

		return
	}

	// ciaddr must be zero.  The client is seeking to verify a previously
	// allocated, cached configuration.
	ciaddr, _ := netip.AddrFromSlice(req.ClientIP)
	if ciaddr.IsValid() && !ciaddr.IsUnspecified() {
		l.DebugContext(ctx, "unexpected ciaddr in init-reboot request", "ciaddr", ciaddr)

		return
	}

	// Check if the lease exists and matches.
	mac := req.ClientHWAddr
	mk := macToKey(mac)

	iface.common.indexMu.Lock()
	defer iface.common.indexMu.Unlock()

	lease, hasLease := iface.common.leases[mk]
	if !hasLease {
		// If the DHCP server has no record of this client, then it MUST remain
		// silent, and MAY output a warning to the network administrator.
		l.WarnContext(ctx, "no existing lease", "mac", mac)

		return
	}

	if lease.IP != reqIP {
		l.WarnContext(ctx, "init-reboot request mismatched", "requested", reqIP, "lease", lease.IP)
		iface.respondNAK(ctx, rw, req)

		return
	}

	// Commit the lease and send ACK.
	iface.commitLease(ctx, lease, hostname4(req))
	iface.respondACK(ctx, rw, req, lease)
}

// handleRenew handles messages of type DHCPREQUEST in RENEWING or REBINDING
// state.  rw must not be nil, ip should be a previously leased address, req
// must contain a valid chaddr according to [netutil.ValidateMAC].
func (iface *dhcpInterfaceV4) handleRenew(
	ctx context.Context,
	rw responseWriter4,
	req *layers.DHCPv4,
	ip netip.Addr,
) {
	l := iface.common.logger

	// Check if the lease exists and matches.
	mac := req.ClientHWAddr
	mk := macToKey(mac)

	iface.common.indexMu.Lock()
	defer iface.common.indexMu.Unlock()

	lease, hasLease := iface.common.leases[mk]
	if !hasLease {
		// If the DHCP server has no record of this client, then it MUST remain
		// silent, and MAY output a warning to the network administrator.
		l.InfoContext(ctx, "no existing lease", "mac", mac)

		// TODO(e.burkov):  Investigate if we should respond with NAK.
		return
	}

	if lease.IP != ip {
		l.DebugContext(ctx, "renew request mismatched", "ciaddr", ip, "lease", lease.IP)
		iface.respondNAK(ctx, rw, req)

		return
	}

	// Commit the lease and send ACK.
	iface.commitLease(ctx, lease, hostname4(req))
	iface.respondACK(ctx, rw, req, lease)
}

// handleDecline handles messages of type DHCPDECLINE.  req should contain a
// requested IP address, as previously offered by the server.  ip should be a
// previously leased address, req must contain a valid chaddr according to
// [netutil.ValidateMAC].
//
// TODO(e.burkov):  Log the message option, as the request should include one.
//
// TODO(e.burkov):  Consider DRY'ing this with [dhcpInterfaceV4.handleRelease].
func (iface *dhcpInterfaceV4) handleDecline(
	ctx context.Context,
	ip netip.Addr,
	req *layers.DHCPv4,
) {
	l := iface.common.logger

	// Check if the lease exists and matches.
	mac := req.ClientHWAddr
	mk := macToKey(mac)

	iface.common.indexMu.Lock()
	defer iface.common.indexMu.Unlock()

	lease, hasLease := iface.common.leases[mk]
	if !hasLease {
		l.ErrorContext(ctx, "decline message for non-existing lease", "mac", mac)

		return
	}

	if lease.IP != ip {
		l.ErrorContext(ctx, "decline mismatch", "ip", ip, "lease", lease.IP)

		return
	}

	l.WarnContext(ctx, "lease reported to be unavailable", "ip", lease.IP)

	err := iface.common.index.remove(ctx, l, lease, iface.common)
	if err != nil {
		l.ErrorContext(ctx, "removing lease", slogutil.KeyError, err)

		return
	}

	iface.common.blockLease(lease, iface.clock)
}

// handleRelease handles messages of type DHCPRELEASE.  ip should be valid,
// previously allocated by the server and correspond to the client's lease.  req
// must contain a valid chaddr according to [netutil.ValidateMAC].
//
// TODO(e.burkov):  Consider DRY'ing this with [dhcpInterfaceV4.handleDecline].
func (iface *dhcpInterfaceV4) handleRelease(
	ctx context.Context,
	ip netip.Addr,
	req *layers.DHCPv4,
) {
	l := iface.common.logger

	// Check if the lease exists and matches.
	mac := req.ClientHWAddr
	mk := macToKey(mac)

	iface.common.indexMu.Lock()
	defer iface.common.indexMu.Unlock()

	lease, hasLease := iface.common.leases[mk]
	if !hasLease {
		l.ErrorContext(ctx, "release message for non-existing lease", "mac", mac)

		return
	}

	if lease.IP != ip {
		l.ErrorContext(ctx, "release mismatch", "ip", ip, "lease", lease.IP)

		return
	}

	err := iface.common.index.remove(ctx, l, lease, iface.common)
	if err != nil {
		l.ErrorContext(ctx, "removing lease", slogutil.KeyError, err)

		return
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

// findInterface returns the first DHCPv4 interface within ifaces containing
// ip.  It returns false if there is no such interface.  ip must be valid.
func (ifaces dhcpInterfacesV4) findInterface(ip netip.Addr) (iface *dhcpInterfaceV4, ok bool) {
	i := slices.IndexFunc(ifaces, func(iface *dhcpInterfaceV4) (contains bool) {
		return iface.subnet.Contains(ip)
	})
	if i < 0 {
		return nil, false
	}

	return ifaces[i], true
}

// allocateLease allocates a new lease for the MAC address.  If there are no IP
// addresses left, both l and err are nil.  mac must be a valid according to
// [netutil.ValidateMAC].
func (iface *dhcpInterfaceV4) allocateLease(
	ctx context.Context,
	mac net.HardwareAddr,
) (l *Lease, err error) {
	for {
		l, err = iface.reserveLease(ctx, mac)
		if err != nil {
			return nil, fmt.Errorf("reserving a lease: %w", err)
		}

		var ok bool
		ok, err = iface.addrChecker.IsAvailable(l.IP)
		if err != nil {
			return nil, fmt.Errorf("checking address availability: %w", err)
		} else if ok {
			return l, nil
		}

		iface.common.logger.DebugContext(ctx, "address not available", "ip", l.IP)

		iface.common.blockLease(l, iface.clock)
	}
}

// reserveLease reserves a lease for a client by its MAC-address.  l is nil if a
// new lease can't be allocated.  mac must be a valid according to
// [netutil.ValidateMAC].
func (iface *dhcpInterfaceV4) reserveLease(
	ctx context.Context,
	mac net.HardwareAddr,
) (l *Lease, err error) {
	iface.common.indexMu.Lock()
	defer iface.common.indexMu.Unlock()

	nextIP := iface.common.nextIP()
	if nextIP == (netip.Addr{}) {
		l = iface.common.findExpiredLease(iface.clock.Now())
		if l == nil {
			return nil, nil
		}

		// TODO(e.burkov):  Move validation from index methods into server's
		// methods and use index here.
		delete(iface.common.leases, macToKey(l.HWAddr))

		l.HWAddr = slices.Clone(mac)
		iface.common.leases[macToKey(mac)] = l

		return l, nil
	}

	l = &Lease{
		HWAddr: slices.Clone(mac),
		IP:     nextIP,
	}

	err = iface.common.index.add(ctx, iface.common.logger, l, iface.common)
	if err != nil {
		return nil, err
	}

	return l, nil
}
