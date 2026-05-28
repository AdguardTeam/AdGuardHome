package dhcpsvc

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"slices"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/AdguardTeam/golibs/validate"
	"github.com/google/gopacket/layers"
)

// Port numbers for DHCPv6.
//
// See RFC 9915 Section 7.2.
const (
	// ServerPortV6 is the standard DHCPv6 server port.
	ServerPortV6 layers.UDPPort = 547

	// ClientPortV6 is the standard DHCPv6 client port.
	ClientPortV6 layers.UDPPort = 546
)

// HardwareTypeEthernet is the IANA hardware type number for Ethernet, used in
// DUID-LL and DUID-LLT construction.  Its value is 1, encoded as a big-endian
// uint16.
//
// See https://www.iana.org/assignments/arp-parameters/arp-parameters.xhtml#arp-parameters-2.
//
// TODO(e.burkov):  Use.
var HardwareTypeEthernet = []byte{0x00, 0x01}

// DHCPv6 multicast addresses.
//
// See RFC 9915 Section 7.1.
var (
	// AllDHCPRelayAgentsAndServers is the well-known IPv6 multicast address
	// All_DHCP_Relay_Agents_and_Servers.  Clients send messages to this address
	// to reach all servers on the local link.
	AllDHCPRelayAgentsAndServers = netip.MustParseAddr("ff02::1:2")

	// AllDHCPServers is the well-known IPv6 multicast address All_DHCP_Servers.
	// Relay agents use this to reach all servers.
	AllDHCPServers = netip.MustParseAddr("ff05::1:3")
)

// v6PrefLen is the length of prefix to match ip against.
//
// TODO(e.burkov):  DHCPv6 inherits the weird behavior of legacy implementation
// where the allocated range constrained by the first address and the first
// address with last byte set to 0xff.  Proper prefixes should be used instead.
const v6PrefLen = netutil.IPv6BitLen - 8

// IPv6Config is the interface-specific configuration for DHCPv6.
//
// TODO(e.burkov):  Add RangeEnd and SubnetPrefix fields, and validate them.
type IPv6Config struct {
	// Clock is used to get the current time.  It should not be nil.
	Clock timeutil.Clock

	// RangeStart is the first address in the range to assign to DHCP clients.
	// It should be a valid IPv6 address.
	RangeStart netip.Addr

	// Options is the list of explicit DHCP options to send to clients.  The
	// options with zero length are treated as deletions of the corresponding
	// options, either implicit or explicit.
	Options layers.DHCPv6Options

	// LeaseDuration is the TTL of a DHCP lease.  It should be positive.
	LeaseDuration time.Duration

	// RASlaacOnly defines whether the DHCP clients should only use SLAAC for
	// address assignment.
	RASLAACOnly bool

	// RAAllowSlaac defines whether the DHCP clients may use SLAAC for address
	// assignment.
	RAAllowSLAAC bool

	// Enabled is the state of the DHCPv6 service, whether it is enabled or not
	// on the specific interface.
	Enabled bool
}

// type check
var _ validate.Interface = (*IPv6Config)(nil)

// Validate implements the [validate.Interface] interface for *IPv6Config.
func (c *IPv6Config) Validate() (err error) {
	if c == nil {
		return errors.ErrNoValue
	} else if !c.Enabled {
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
func (c *IPv6Config) validateSubnet(orig []error) (errs []error) {
	errs = orig

	if !c.RangeStart.Is6() {
		err := newMustErr("range start", "be a valid ipv6", c.RangeStart)
		errs = append(errs, err)
	}

	return errs
}

// dhcpInterfaceV6 is a DHCP interface for IPv6 address family.
type dhcpInterfaceV6 struct {
	// common is the common part of any network interface within the DHCP
	// server.
	common *netInterface

	// clock is used to get the current time.
	clock timeutil.Clock

	// addrChecker checks if an address is available for leasing in current
	// network.
	addrChecker addressChecker

	// subnetPrefix is the network prefix of the interface's IPv6 subnet.  It is
	// used for on-link address determination.
	subnetPrefix netip.Prefix

	// implicitOpts are the DHCPv6 options listed in RFC 8415 (and others) and
	// initialized with default values.  It must not have intersections with
	// explicitOpts.
	implicitOpts layers.DHCPv6Options

	// explicitOpts are the user-configured options.  It must not have
	// intersections with implicitOpts.
	explicitOpts layers.DHCPv6Options

	// t1 is the pre-computed T1 value (0.5 × LeaseDuration) per RFC 9915 §21.4.
	// It is the time after which the client should contact the same server to
	// extend the lease.
	t1 time.Duration

	// t2 is the pre-computed T2 value (0.8 × LeaseDuration) per RFC 9915 §21.4.
	// It is the time after which the client may contact any server to extend
	// the lease.
	t2 time.Duration

	// raSLAACOnly defines if DHCP should send ICMPv6.RA packets without MO
	// flags.
	raSLAACOnly bool

	// raAllowSLAAC defines if DHCP should send ICMPv6.RA packets with MO flags.
	raAllowSLAAC bool
}

// newDHCPInterfaceV6 creates a new DHCP interface for IPv6 address family with
// the given configuration.  If the interface is disabled, it returns nil.  conf
// must be valid.  hwAddr must not be empty.
func (srv *DHCPServer) newDHCPInterfaceV6(
	ctx context.Context,
	l *slog.Logger,
	name string,
	conf *IPv6Config,
) (iface *dhcpInterfaceV6) {
	if !conf.Enabled {
		l.DebugContext(ctx, "disabled")

		return nil
	}

	// TODO(e.burkov):  Migrate the configuration to use proper range start,
	// end, and subnet prefix.
	rangeEndData := conf.RangeStart.As16()
	rangeEndData[15] = 0xff

	// TODO(e.burkov):  Validate the range end and subnet prefix against the
	// range start during configuration validation.
	addrSpace, _ := newIPRange(conf.RangeStart, netip.AddrFrom16(rangeEndData))

	iface = &dhcpInterfaceV6{
		common: &netInterface{
			logger:        l,
			leases:        map[macKey]*Lease{},
			indexMu:       srv.leasesMu,
			index:         srv.leases,
			name:          name,
			addrSpace:     addrSpace,
			leasedOffsets: newBitSet(),
			leaseTTL:      conf.LeaseDuration,
		},
		clock: conf.Clock,
		// TODO(e.burkov):  Use an ICMP implementation.
		addrChecker:  noopAddressChecker{},
		subnetPrefix: netip.PrefixFrom(conf.RangeStart, v6PrefLen),
		t1:           conf.LeaseDuration / 2,
		t2:           conf.LeaseDuration * 4 / 5,
		raSLAACOnly:  conf.RASLAACOnly,
		raAllowSLAAC: conf.RAAllowSLAAC,
	}
	iface.implicitOpts, iface.explicitOpts = conf.options(ctx, l)

	return iface
}

// dhcpInterfacesV6 is a slice of network interfaces of IPv6 address family.
type dhcpInterfacesV6 []*dhcpInterfaceV6

// find returns the first network interface within ifaces whose subnet prefix
// contains ip.  It returns false if there is no such interface.
func (ifaces dhcpInterfacesV6) find(ip netip.Addr) (iface6 *netInterface, ok bool) {
	i := slices.IndexFunc(ifaces, func(iface *dhcpInterfaceV6) (contains bool) {
		return iface.subnetPrefix.Contains(ip)
	})
	if i < 0 {
		return nil, false
	}

	return ifaces[i].common, true
}

// options returns the implicit and explicit options for the interface.  The two
// lists are disjoint and the implicit options are initialized with default
// values.
//
// TODO(e.burkov):  Add implicit options according to RFC.
func (c *IPv6Config) options(ctx context.Context, l *slog.Logger) (imp, exp layers.DHCPv6Options) {
	// Set default values of host configuration parameters listed in RFC 8415.
	imp = layers.DHCPv6Options{}
	slices.SortFunc(imp, compareV6OptionCodes)

	// Set values for explicitly configured options.
	for _, e := range c.Options {
		i, found := slices.BinarySearchFunc(imp, e, compareV6OptionCodes)
		if found {
			imp = slices.Delete(imp, i, i+1)
		}

		exp = append(exp, e)
	}

	l.DebugContext(ctx, "options", "implicit", imp, "explicit", exp)

	return imp, exp
}

// compareV6OptionCodes compares option codes of a and b.
func compareV6OptionCodes(a, b layers.DHCPv6Option) (res int) {
	return int(a.Code) - int(b.Code)
}

// appendRequestedOptions adds the options to opts in accordance with the
// requested parameters.  req must not be nil.
//
// See RFC 9915 Section 21.7.
func (iface *dhcpInterfaceV6) appendRequestedOptions(
	opts layers.DHCPv6Options,
	req *layers.DHCPv6,
) (res layers.DHCPv6Options) {
	optWithCode := layers.DHCPv6Option{}
	for _, code := range requestedOptions6(req) {
		optWithCode.Code = code
		i, has := slices.BinarySearchFunc(iface.implicitOpts, optWithCode, compareV6OptionCodes)
		if has {
			opts = append(opts, iface.implicitOpts[i])
		}
	}

	for _, opt := range iface.explicitOpts {
		if len(opt.Data) > 0 {
			opts = append(opts, opt)

			continue
		}

		// Remove options explicitly configured to be removed, in case they are
		// already set.
		opts = slices.DeleteFunc(opts, func(o layers.DHCPv6Option) (ok bool) {
			return o.Code == opt.Code
		})
	}

	return opts
}

// clientIDNoServer extracts the client identifier from opts and checks that
// there is no server identifier.  It returns an error if the client identifier
// is not found or if the server identifier is found.
func clientIDNoServer(opts layers.DHCPv6Options) (cliID *layers.DHCPv6DUID, err error) {
	_, ok := serverDUID6(opts)
	if ok {
		return nil, fmt.Errorf("dhcpv6: server id: %w", errors.ErrUnexpectedValue)
	}

	cliIDData, ok := clientDUID6(opts)
	if !ok {
		return nil, fmt.Errorf("dhcpv6: client id: %w", errors.ErrNoValue)
	}

	cliID = &layers.DHCPv6DUID{}
	err = cliID.DecodeFromBytes(cliIDData)
	if err != nil {
		return nil, fmt.Errorf("dhcpv6: client id: %w", err)
	}

	return cliID, nil
}

// clientIDMatchingServer extracts the client identifier from opts and checks
// that the server identifier matches serverDUID.  It returns an error if the
// client identifier is not found, if the server identifier is not found, or if
// the server identifier does not match serverDUID.
func clientIDMatchingServer(
	opts layers.DHCPv6Options,
	serverDUID []byte,
) (cliID *layers.DHCPv6DUID, err error) {
	srvID, ok := serverDUID6(opts)
	if !ok {
		return nil, fmt.Errorf("dhcpv6: server id: %w", errors.ErrNoValue)
	}

	// TODO(e.burkov):  Add validate.EqualFunc.
	if !bytes.Equal(srvID, serverDUID) {
		return nil, fmt.Errorf(
			"dhcpv6: server id: got %v, want %v: %w",
			srvID,
			serverDUID,
			errors.ErrNotEqual,
		)
	}

	cliIDData, ok := clientDUID6(opts)
	if !ok {
		return nil, fmt.Errorf("dhcpv6: client id: %w", errors.ErrNoValue)
	}

	cliID = &layers.DHCPv6DUID{}
	err = cliID.DecodeFromBytes(cliIDData)
	if err != nil {
		return nil, fmt.Errorf("dhcpv6: client id: %w", err)
	}

	return cliID, nil
}

// defaultHopLimit is the default hop limit for relaying DHCPv6 response
// packets.
//
// See RFC 9915 Section 7.6.
const defaultHopLimit = 8

// respond6 constructs and sends a DHCPv6 response to the client.
func respond6(fd *frameData6, resp *layers.DHCPv6) (err error) {
	eth := &layers.Ethernet{
		SrcMAC:       fd.ether.DstMAC,
		DstMAC:       fd.ether.SrcMAC,
		EthernetType: layers.EthernetTypeIPv6,
	}

	ip := &layers.IPv6{
		Version:    6,
		NextHeader: layers.IPProtocolUDP,
		HopLimit:   defaultHopLimit,
		SrcIP:      fd.localAddr.AsSlice(),
		// If the original message was received directly by the server, the
		// server unicasts the Advertise or Reply message directly to the client
		// using the address in the source address field from the IP datagram in
		// which the original message was received.
		//
		// See RFC 9915 Section 18.3.10.
		DstIP: fd.ip.SrcIP,
	}

	udp := &layers.UDP{
		SrcPort: ServerPortV6,
		DstPort: ClientPortV6,
	}

	// It only returns an error if the network layer is not an IP layer.
	err = udp.SetNetworkLayerForChecksum(ip)
	if err != nil {
		panic(err)
	}

	err = respond(fd.device, eth, udp, ip, resp)
	if err != nil {
		return fmt.Errorf("writing dhcpv6 response: %w", err)
	}

	return nil
}

// allocateForSolicit allocates a lease for the first IA_NA option in req and
// returns it.  It returns a zero iaid if there is no IA_NA option, if the
// option is malformed.  lease is nil if there is no address available for
// leasing.  mac must be a valid MAC address according to [netutil.ValidateMAC],
// req must be a valid DHCPv6 message of SOLICIT type, iface.common.indexMu
// mutex must be locked.
//
// TODO(e.burkov):  Support allocating several leases at a time when the
// database will migrate, see the BUG at [Lease]'s documentation.
func (iface *dhcpInterfaceV6) allocateForSolicit(
	ctx context.Context,
	mac net.HardwareAddr,
	req *layers.DHCPv6,
) (lease *Lease, iaid uint32) {
	l := iface.common.logger

	for _, reqOpt := range req.Options {
		if reqOpt.Code != layers.DHCPv6OptIANA {
			continue
		}

		var iana iaNAOption
		err := iana.UnmarshalBinary(reqOpt.Data)
		if err != nil {
			// TODO(e.burkov):  Recheck the logic on malformed IA_NA options.
			l.DebugContext(ctx, "malformed ia_na in solicit", slogutil.KeyError, err)

			continue
		}

		// TODO(e.burkov):  Test the case, where the lease exists and is
		// expired.
		//
		// TODO(e.burkov):  Support allocating the exact requested address if it
		// is available.
		lease, err = iface.common.allocateLease(ctx, mac, iface.addrChecker, iface.clock)
		if err != nil {
			l.DebugContext(ctx, "no address available", "iaid", iana.iaid, slogutil.KeyError, err)

			continue
		}

		return lease, iana.iaid
	}

	return nil, 0
}

// newSolicitRespOpts returns the common option list for Advertise and
// rapid-commit Reply responses to a Solicit request.  cliID must not be nil.
func (iface *dhcpInterfaceV6) newSolicitRespOpts(
	fd *frameData6,
	req *layers.DHCPv6,
	cliID *layers.DHCPv6DUID,
	iaid uint32,
	lease *Lease,
	rapidCommit bool,
) (opts layers.DHCPv6Options) {
	cliIDData := cliID.Encode()

	opts = append(opts, layers.NewDHCPv6Option(layers.DHCPv6OptServerID, fd.duidData))
	opts = append(opts, layers.NewDHCPv6Option(layers.DHCPv6OptClientID, cliIDData))

	// For Solicit without IA_NA options, respond with safe Advertise with no
	// IA_NA options and Status Code NoAddrsAvail.
	if iaid == 0 {
		opts = append(opts, newStatusCodeOption(layers.DHCPv6StatusCodeNoAddrsAvail))
	} else {
		opts = append(opts, iface.iaNAFromLease(lease, iaid))
	}

	// The server preference value MUST default to 0 unless otherwise configured
	// by the server administrator.
	//
	// See RFC 9915 Section 18.3.9.
	opts = append(opts, newPreferenceOption(0))
	opts = append(opts, newSOLMaxRTOption(solMaxRT))

	if rapidCommit {
		opts = append(opts, layers.NewDHCPv6Option(layers.DHCPv6OptRapidCommit, nil))
	}

	return iface.appendRequestedOptions(opts, req)
}

// iaNAFromLease returns an IA_NA option with a single IA Address sub-option
// corresponding to lease and with the given iaid.  The T1 and T2 values are set
// according to iface.t1 and iface.t2.  If lease is nil, it returns an IA_NA
// option with the Status Code [layers.NoAddrsAvail].  iaid must not be zero.
func (iface *dhcpInterfaceV6) iaNAFromLease(lease *Lease, iaid uint32) (iana layers.DHCPv6Option) {
	if lease == nil {
		return newIANAWithStatus(iaid, layers.DHCPv6StatusCodeNoAddrsAvail)
	}

	return iaNAOption{
		nested: []iaAddrOption{{
			addr:              lease.IP,
			preferredLifetime: iface.common.leaseTTL,
			validLifetime:     iface.common.leaseTTL,
		}},
		iaid: iaid,
		t1:   iface.t1,
		t2:   iface.t2,
	}.Encode()
}

// commit updates the lease allocated previously via a SOLICIT, or during
// handling the Rapid Commit option, assigning a hostname according to req.  It
// deallocates the lease if the one fails to be committed.  lease must be
// non-nil and allocated for the client corresponding to req,
// iface.common.indexMu mutex must be locked.
//
// TODO(e.burkov):  Support committing several leases at a time when the
// database will migrate, see the BUG at [Lease]'s documentation.
func (iface *dhcpInterfaceV6) commit(
	ctx context.Context,
	req *layers.DHCPv6,
	lease *Lease,
) (err error) {
	if hostname := clientFQDN6(req); hostname != "" {
		lease.Hostname = hostname
	} else {
		lease.Hostname = aghnet.GenerateHostname(lease.IP)
	}

	err = iface.common.index.update(ctx, iface.common.logger, lease, iface.common)
	if err != nil {
		rmErr := iface.common.removeLease(lease)
		err = errors.WithDeferred(err, rmErr)

		return fmt.Errorf("committing rapid lease for ip %s: %w", lease.IP, err)

	}

	return nil
}
