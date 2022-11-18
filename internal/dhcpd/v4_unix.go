//go:build darwin || freebsd || linux || openbsd

package dhcpd

import (
	"bytes"
	"fmt"
	"net"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/stringutil"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/go-ping/ping"
	"github.com/insomniacslk/dhcp/dhcpv4"
	"github.com/insomniacslk/dhcp/dhcpv4/server4"
	"golang.org/x/exp/slices"

	//lint:ignore SA1019 See the TODO in go.mod.
	"github.com/mdlayher/raw"
)

// v4Server is a DHCPv4 server.
//
// TODO(a.garipov): Think about unifying this and v6Server.
type v4Server struct {
	conf *V4ServerConf

	srv *server4.Server

	// implicitOpts are the options listed in Appendix A of RFC 2131 initialized
	// with default values.  It must not have intersections with [explicitOpts].
	implicitOpts dhcpv4.Options

	// explicitOpts are the options parsed from the configuration.  It must not
	// have intersections with [implicitOpts].
	explicitOpts dhcpv4.Options

	// leasesLock protects leases, leaseHosts, and leasedOffsets.
	leasesLock sync.Mutex

	// leasedOffsets contains offsets from conf.ipRange.start that have been
	// leased.
	leasedOffsets *bitSet

	// leaseHosts is the set of all hostnames of all known DHCP clients.
	leaseHosts *stringutil.Set

	// leases contains all dynamic and static leases.
	leases []*Lease
}

func (s *v4Server) enabled() (ok bool) {
	return s.conf != nil && s.conf.Enabled
}

// WriteDiskConfig4 - write configuration
func (s *v4Server) WriteDiskConfig4(c *V4ServerConf) {
	if s.conf != nil {
		*c = *s.conf
	}
}

// WriteDiskConfig6 - write configuration
func (s *v4Server) WriteDiskConfig6(c *V6ServerConf) {
}

// normalizeHostname normalizes a hostname sent by the client.  If err is not
// nil, norm is an empty string.
func normalizeHostname(hostname string) (norm string, err error) {
	defer func() { err = errors.Annotate(err, "normalizing %q: %w", hostname) }()

	if hostname == "" {
		return "", nil
	}

	norm = strings.ToLower(hostname)
	parts := strings.FieldsFunc(norm, func(c rune) (ok bool) {
		return c != '.' && !netutil.IsValidHostOuterRune(c)
	})

	if len(parts) == 0 {
		return "", fmt.Errorf("no valid parts")
	}

	norm = strings.Join(parts, "-")
	norm = strings.TrimSuffix(norm, "-")

	return norm, nil
}

// validHostnameForClient accepts the hostname sent by the client and its IP and
// returns either a normalized version of that hostname, or a new hostname
// generated from the IP address, or an empty string.
func (s *v4Server) validHostnameForClient(cliHostname string, ip net.IP) (hostname string) {
	hostname, err := normalizeHostname(cliHostname)
	if err != nil {
		log.Info("dhcpv4: %s", err)
	}

	if hostname == "" {
		hostname = aghnet.GenerateHostname(ip)
	}

	err = netutil.ValidateDomainName(hostname)
	if err != nil {
		log.Info("dhcpv4: %s", err)
		hostname = ""
	}

	return hostname
}

// ResetLeases resets leases.
func (s *v4Server) ResetLeases(leases []*Lease) (err error) {
	defer func() { err = errors.Annotate(err, "dhcpv4: %w") }()

	if s.conf == nil {
		return nil
	}

	s.leasedOffsets = newBitSet()
	s.leaseHosts = stringutil.NewSet()
	s.leases = nil

	for _, l := range leases {
		if !l.IsStatic() {
			l.Hostname = s.validHostnameForClient(l.Hostname, l.IP)
		}
		err = s.addLease(l)
		if err != nil {
			// TODO(a.garipov): Wrap and bubble up the error.
			log.Error("dhcpv4: reset: re-adding a lease for %s (%s): %s", l.IP, l.HWAddr, err)

			continue
		}
	}

	return nil
}

// getLeasesRef returns the actual leases slice.  For internal use only.
func (s *v4Server) getLeasesRef() []*Lease {
	return s.leases
}

// isBlocklisted returns true if this lease holds a blocklisted IP.
//
// TODO(a.garipov): Make a method of *Lease?
func (s *v4Server) isBlocklisted(l *Lease) (ok bool) {
	if len(l.HWAddr) == 0 {
		return false
	}

	ok = true
	for _, b := range l.HWAddr {
		if b != 0 {
			ok = false

			break
		}
	}

	return ok
}

// GetLeases returns the list of current DHCP leases.  It is safe for concurrent
// use.
func (s *v4Server) GetLeases(flags GetLeasesFlags) (leases []*Lease) {
	// The function shouldn't return nil, because zero-length slice behaves
	// differently in cases like marshalling.  Our front-end also requires
	// a non-nil value in the response.
	leases = []*Lease{}

	getDynamic := flags&LeasesDynamic != 0
	getStatic := flags&LeasesStatic != 0

	s.leasesLock.Lock()
	defer s.leasesLock.Unlock()

	now := time.Now()
	for _, l := range s.leases {
		if getDynamic && l.Expiry.After(now) && !s.isBlocklisted(l) {
			leases = append(leases, l.Clone())

			continue
		}

		if getStatic && l.IsStatic() {
			leases = append(leases, l.Clone())
		}
	}

	return leases
}

// FindMACbyIP - find a MAC address by IP address in the currently active DHCP leases
func (s *v4Server) FindMACbyIP(ip net.IP) net.HardwareAddr {
	now := time.Now()

	s.leasesLock.Lock()
	defer s.leasesLock.Unlock()

	ip4 := ip.To4()
	if ip4 == nil {
		return nil
	}

	for _, l := range s.leases {
		if l.IP.Equal(ip4) {
			if l.Expiry.After(now) || l.IsStatic() {
				return l.HWAddr
			}
		}
	}

	return nil
}

// defaultHwAddrLen is the default length of a hardware (MAC) address.
const defaultHwAddrLen = 6

// Add the specified IP to the black list for a time period
func (s *v4Server) blocklistLease(l *Lease) {
	l.HWAddr = make(net.HardwareAddr, defaultHwAddrLen)
	l.Hostname = ""
	l.Expiry = time.Now().Add(s.conf.leaseTime)
}

// rmLeaseByIndex removes a lease by its index in the leases slice.
func (s *v4Server) rmLeaseByIndex(i int) {
	n := len(s.leases)
	if i >= n {
		// TODO(a.garipov): Better error handling.
		log.Debug("dhcpv4: can't remove lease at index %d: no such lease", i)

		return
	}

	l := s.leases[i]
	s.leases = append(s.leases[:i], s.leases[i+1:]...)

	r := s.conf.ipRange
	offset, ok := r.offset(l.IP)
	if ok {
		s.leasedOffsets.set(offset, false)
	}

	s.leaseHosts.Del(l.Hostname)

	log.Debug("dhcpv4: removed lease %s (%s)", l.IP, l.HWAddr)
}

// Remove a dynamic lease with the same properties
// Return error if a static lease is found
func (s *v4Server) rmDynamicLease(lease *Lease) (err error) {
	for i, l := range s.leases {
		isStatic := l.IsStatic()

		if bytes.Equal(l.HWAddr, lease.HWAddr) || l.IP.Equal(lease.IP) {
			if isStatic {
				return errors.Error("static lease already exists")
			}

			s.rmLeaseByIndex(i)
			if i == len(s.leases) {
				break
			}

			l = s.leases[i]
		}

		if !isStatic && l.Hostname == lease.Hostname {
			l.Hostname = ""
		}
	}

	return nil
}

// ErrDupHostname is returned by addLease when the added lease has a not empty
// non-unique hostname.
const ErrDupHostname = errors.Error("hostname is not unique")

// addLease adds a dynamic or static lease.
func (s *v4Server) addLease(l *Lease) (err error) {
	r := s.conf.ipRange
	offset, inOffset := r.offset(l.IP)

	if l.IsStatic() {
		// TODO(a.garipov, d.seregin): Subnet can be nil when dhcp server is
		// disabled.
		addr := netip.AddrFrom4(*(*[4]byte)(l.IP.To4()))
		if sn := s.conf.subnet; !sn.Contains(addr) {
			return fmt.Errorf("subnet %s does not contain the ip %q", sn, l.IP)
		}
	} else if !inOffset {
		return fmt.Errorf("lease %s (%s) out of range, not adding", l.IP, l.HWAddr)
	}

	if l.Hostname != "" {
		if s.leaseHosts.Has(l.Hostname) {
			return ErrDupHostname
		}

		s.leaseHosts.Add(l.Hostname)
	}

	s.leases = append(s.leases, l)
	s.leasedOffsets.set(offset, true)

	return nil
}

// rmLease removes a lease with the same properties.
func (s *v4Server) rmLease(lease *Lease) (err error) {
	if len(s.leases) == 0 {
		return nil
	}

	for i, l := range s.leases {
		if l.IP.Equal(lease.IP) {
			if !bytes.Equal(l.HWAddr, lease.HWAddr) || l.Hostname != lease.Hostname {
				return fmt.Errorf("lease for ip %s is different: %+v", lease.IP, l)
			}

			s.rmLeaseByIndex(i)

			return nil
		}
	}

	return errors.Error("lease not found")
}

// ErrUnconfigured is returned from the server's method when it requires the
// server to be configured and it's not.
const ErrUnconfigured errors.Error = "server is unconfigured"

// AddStaticLease implements the DHCPServer interface for *v4Server.  It is safe
// for concurrent use.
func (s *v4Server) AddStaticLease(l *Lease) (err error) {
	defer func() { err = errors.Annotate(err, "dhcpv4: adding static lease: %w") }()

	if s.conf == nil {
		return ErrUnconfigured
	}

	ip := l.IP.To4()
	if ip == nil {
		return fmt.Errorf("invalid ip %q, only ipv4 is supported", l.IP)
	} else if gwIP := s.conf.GatewayIP; gwIP == netip.AddrFrom4(*(*[4]byte)(ip)) {
		return fmt.Errorf("can't assign the gateway IP %s to the lease", gwIP)
	}

	l.Expiry = time.Unix(leaseExpireStatic, 0)

	err = netutil.ValidateMAC(l.HWAddr)
	if err != nil {
		return err
	}

	if hostname := l.Hostname; hostname != "" {
		hostname, err = normalizeHostname(hostname)
		if err != nil {
			return err
		}

		err = netutil.ValidateDomainName(hostname)
		if err != nil {
			return fmt.Errorf("validating hostname: %w", err)
		}

		// Don't check for hostname uniqueness, since we try to emulate dnsmasq
		// here, which means that rmDynamicLease below will simply empty the
		// hostname of the dynamic lease if there even is one.  In case a static
		// lease with the same name already exists, addLease will return an
		// error and the lease won't be added.

		l.Hostname = hostname
	}

	// Perform the following actions in an anonymous function to make sure
	// that the lock gets unlocked before the notification step.
	func() {
		s.leasesLock.Lock()
		defer s.leasesLock.Unlock()

		err = s.rmDynamicLease(l)
		if err != nil {
			err = fmt.Errorf(
				"removing dynamic leases for %s (%s): %w",
				ip,
				l.HWAddr,
				err,
			)

			return
		}

		err = s.addLease(l)
		if err != nil {
			err = fmt.Errorf("adding static lease for %s (%s): %w", ip, l.HWAddr, err)

			return
		}
	}()
	if err != nil {
		return err
	}

	s.conf.notify(LeaseChangedDBStore)
	s.conf.notify(LeaseChangedAddedStatic)

	return nil
}

// RemoveStaticLease removes a static lease.  It is safe for concurrent use.
func (s *v4Server) RemoveStaticLease(l *Lease) (err error) {
	defer func() { err = errors.Annotate(err, "dhcpv4: %w") }()

	if s.conf == nil {
		return ErrUnconfigured
	}

	if len(l.IP) != 4 {
		return fmt.Errorf("invalid IP")
	}

	err = netutil.ValidateMAC(l.HWAddr)
	if err != nil {
		return fmt.Errorf("validating lease: %w", err)
	}

	defer func() {
		if err != nil {
			return
		}

		s.conf.notify(LeaseChangedDBStore)
		s.conf.notify(LeaseChangedRemovedStatic)
	}()

	s.leasesLock.Lock()
	defer s.leasesLock.Unlock()

	return s.rmLease(l)
}

// addrAvailable sends an ICP request to the specified IP address.  It returns
// true if the remote host doesn't reply, which probably means that the IP
// address is available.
//
// TODO(a.garipov): I'm not sure that this is the best way to do this.
func (s *v4Server) addrAvailable(target net.IP) (avail bool) {
	if s.conf.ICMPTimeout == 0 {
		return true
	}

	pinger, err := ping.NewPinger(target.String())
	if err != nil {
		log.Error("dhcpv4: ping.NewPinger(): %s", err)

		return true
	}

	pinger.SetPrivileged(true)
	pinger.Timeout = time.Duration(s.conf.ICMPTimeout) * time.Millisecond
	pinger.Count = 1
	reply := false
	pinger.OnRecv = func(_ *ping.Packet) {
		reply = true
	}

	log.Debug("dhcpv4: sending icmp echo to %s", target)

	err = pinger.Run()
	if err != nil {
		log.Error("dhcpv4: pinger.Run(): %s", err)

		return true
	}

	if reply {
		log.Info("dhcpv4: ip conflict: %s is already used by another device", target)

		return false
	}

	log.Debug("dhcpv4: icmp procedure is complete: %q", target)

	return true
}

// findLease finds a lease by its MAC-address.
func (s *v4Server) findLease(mac net.HardwareAddr) (l *Lease) {
	for _, l = range s.leases {
		if bytes.Equal(mac, l.HWAddr) {
			return l
		}
	}

	return nil
}

// nextIP generates a new free IP.
func (s *v4Server) nextIP() (ip net.IP) {
	r := s.conf.ipRange
	ip = r.find(func(next net.IP) (ok bool) {
		offset, ok := r.offset(next)
		if !ok {
			// Shouldn't happen.
			return false
		}

		return !s.leasedOffsets.isSet(offset)
	})

	return ip.To4()
}

// Find an expired lease and return its index or -1
func (s *v4Server) findExpiredLease() int {
	now := time.Now()
	for i, lease := range s.leases {
		if !lease.IsStatic() && lease.Expiry.Before(now) {
			return i
		}
	}

	return -1
}

// reserveLease reserves a lease for a client by its MAC-address.  It returns
// nil if it couldn't allocate a new lease.
func (s *v4Server) reserveLease(mac net.HardwareAddr) (l *Lease, err error) {
	l = &Lease{HWAddr: slices.Clone(mac)}

	l.IP = s.nextIP()
	if l.IP == nil {
		i := s.findExpiredLease()
		if i < 0 {
			return nil, nil
		}

		copy(s.leases[i].HWAddr, mac)

		return s.leases[i], nil
	}

	err = s.addLease(l)
	if err != nil {
		return nil, err
	}

	return l, nil
}

// commitLease refreshes l's values.  It takes the desired hostname into account
// when setting it into the lease, but generates a unique one if the provided
// can't be used.
func (s *v4Server) commitLease(l *Lease, hostname string) {
	prev := l.Hostname
	hostname = s.validHostnameForClient(hostname, l.IP)

	if s.leaseHosts.Has(hostname) {
		log.Info("dhcpv4: hostname %q already exists", hostname)

		if prev == "" {
			// The lease is just allocated due to DHCPDISCOVER.
			hostname = aghnet.GenerateHostname(l.IP)
		} else {
			hostname = prev
		}
	}
	if l.Hostname != hostname {
		l.Hostname = hostname
	}

	l.Expiry = time.Now().Add(s.conf.leaseTime)
	if prev != "" && prev != l.Hostname {
		s.leaseHosts.Del(prev)
	}
	if l.Hostname != "" {
		s.leaseHosts.Add(l.Hostname)
	}
}

// allocateLease allocates a new lease for the MAC address.  If there are no IP
// addresses left, both l and err are nil.
func (s *v4Server) allocateLease(mac net.HardwareAddr) (l *Lease, err error) {
	for {
		l, err = s.reserveLease(mac)
		if err != nil {
			return nil, fmt.Errorf("reserving a lease: %w", err)
		} else if l == nil {
			return nil, nil
		}

		if s.addrAvailable(l.IP) {
			return l, nil
		}

		s.blocklistLease(l)
	}
}

// handleDiscover is the handler for the DHCP Discover request.
func (s *v4Server) handleDiscover(req, resp *dhcpv4.DHCPv4) (l *Lease, err error) {
	mac := req.ClientHWAddr

	defer s.conf.notify(LeaseChangedDBStore)

	s.leasesLock.Lock()
	defer s.leasesLock.Unlock()

	l = s.findLease(mac)
	if l != nil {
		reqIP := req.RequestedIPAddress()
		if len(reqIP) != 0 && !reqIP.Equal(l.IP) {
			log.Debug("dhcpv4: different RequestedIP: %s != %s", reqIP, l.IP)
		}

		resp.UpdateOption(dhcpv4.OptMessageType(dhcpv4.MessageTypeOffer))

		return l, nil
	}

	l, err = s.allocateLease(mac)
	if err != nil {
		return nil, err
	} else if l == nil {
		log.Debug("dhcpv4: no more ip addresses")

		return nil, nil
	}

	resp.UpdateOption(dhcpv4.OptMessageType(dhcpv4.MessageTypeOffer))

	return l, nil
}

// OptionFQDN returns a DHCPv4 option for sending the FQDN to the client
// requested another hostname.
//
// See https://datatracker.ietf.org/doc/html/rfc4702.
func OptionFQDN(fqdn string) (opt dhcpv4.Option) {
	optData := []byte{
		// Set only S and O DHCP client FQDN option flags.
		//
		// See https://datatracker.ietf.org/doc/html/rfc4702#section-2.1.
		1<<0 | 1<<1,
		// The RCODE fields should be set to 0xFF in the server responses.
		//
		// See https://datatracker.ietf.org/doc/html/rfc4702#section-2.2.
		0xFF,
		0xFF,
	}
	optData = append(optData, fqdn...)

	return dhcpv4.OptGeneric(dhcpv4.OptionFQDN, optData)
}

// checkLease checks if the pair of mac and ip is already leased.  The mismatch
// is true when the existing lease has the same hardware address but differs in
// its IP address.
func (s *v4Server) checkLease(mac net.HardwareAddr, ip net.IP) (lease *Lease, mismatch bool) {
	s.leasesLock.Lock()
	defer s.leasesLock.Unlock()

	for _, l := range s.leases {
		if !bytes.Equal(l.HWAddr, mac) {
			continue
		}

		if l.IP.Equal(ip) {
			return l, false
		}

		log.Debug(
			`dhcpv4: mismatched OptionRequestedIPAddress in req msg for %s`,
			mac,
		)

		return nil, true
	}

	return nil, false
}

// handleSelecting handles the DHCPREQUEST generated during SELECTING state.
func (s *v4Server) handleSelecting(
	req *dhcpv4.DHCPv4,
	reqIP net.IP,
	sid net.IP,
) (l *Lease, needsReply bool) {
	// Client inserts the address of the selected server in server identifier,
	// ciaddr MUST be zero.
	mac := req.ClientHWAddr

	if !sid.Equal(s.conf.dnsIPAddrs[0].AsSlice()) {
		log.Debug("dhcpv4: bad server identifier in req msg for %s: %s", mac, sid)

		return nil, false
	} else if ciaddr := req.ClientIPAddr; ciaddr != nil && !ciaddr.IsUnspecified() {
		log.Debug("dhcpv4: non-zero ciaddr in selecting req msg for %s", mac)

		return nil, false
	}

	// Requested IP address MUST be filled in with the yiaddr value from the
	// chosen DHCPOFFER.
	if ip4 := reqIP.To4(); ip4 == nil {
		log.Debug("dhcpv4: bad requested address in req msg for %s: %s", mac, reqIP)

		return nil, false
	}

	var mismatch bool
	if l, mismatch = s.checkLease(mac, reqIP); mismatch {
		return nil, true
	} else if l == nil {
		log.Debug("dhcpv4: no reserved lease for %s", mac)
	}

	return l, true
}

// handleInitReboot handles the DHCPREQUEST generated during INIT-REBOOT state.
func (s *v4Server) handleInitReboot(req *dhcpv4.DHCPv4, reqIP net.IP) (l *Lease, needsReply bool) {
	mac := req.ClientHWAddr

	ip4 := reqIP.To4()
	if ip4 == nil {
		log.Debug("dhcpv4: bad requested address in req msg for %s: %s", mac, reqIP)

		return nil, false
	}

	// ciaddr MUST be zero.  The client is seeking to verify a previously
	// allocated, cached configuration.
	if ciaddr := req.ClientIPAddr; ciaddr != nil && !ciaddr.IsUnspecified() {
		log.Debug("dhcpv4: non-zero ciaddr in init-reboot req msg for %s", mac)

		return nil, false
	}

	if !s.conf.subnet.Contains(netip.AddrFrom4(*(*[4]byte)(ip4))) {
		// If the DHCP server detects that the client is on the wrong net then
		// the server SHOULD send a DHCPNAK message to the client.
		log.Debug("dhcpv4: wrong subnet in init-reboot req msg for %s: %s", mac, reqIP)

		return nil, true
	}

	var mismatch bool
	if l, mismatch = s.checkLease(mac, reqIP); mismatch {
		return nil, true
	} else if l == nil {
		// If the DHCP server has no record of this client, then it MUST remain
		// silent, and MAY output a warning to the network administrator.
		log.Info("dhcpv4: warning: no existing lease for %s", mac)

		return nil, false
	}

	return l, true
}

// handleRenew handles the DHCPREQUEST generated during RENEWING or REBINDING
// state.
func (s *v4Server) handleRenew(req *dhcpv4.DHCPv4) (l *Lease, needsReply bool) {
	mac := req.ClientHWAddr

	// ciaddr MUST be filled in with client's IP address.
	ciaddr := req.ClientIPAddr
	if ciaddr == nil || ciaddr.IsUnspecified() || ciaddr.To4() == nil {
		log.Debug("dhcpv4: bad ciaddr in renew req msg for %s: %s", mac, ciaddr)

		return nil, false
	}

	var mismatch bool
	if l, mismatch = s.checkLease(mac, ciaddr); mismatch {
		return nil, true
	} else if l == nil {
		// If the DHCP server has no record of this client, then it MUST remain
		// silent, and MAY output a warning to the network administrator.
		log.Info("dhcpv4: warning: no existing lease for %s", mac)

		return nil, false
	}

	return l, true
}

// handleByRequestType handles the DHCPREQUEST according to the state during
// which it's generated by client.
func (s *v4Server) handleByRequestType(req *dhcpv4.DHCPv4) (lease *Lease, needsReply bool) {
	reqIP, sid := req.RequestedIPAddress(), req.ServerIdentifier()

	if sid != nil && !sid.IsUnspecified() {
		// If the DHCPREQUEST message contains a server identifier option, the
		// message is in response to a DHCPOFFER message.  Otherwise, the
		// message is a request to verify or extend an existing lease.
		return s.handleSelecting(req, reqIP, sid)
	}

	if reqIP != nil && !reqIP.IsUnspecified() {
		// Requested IP address option MUST be filled in with client's notion of
		// its previously assigned address.
		return s.handleInitReboot(req, reqIP)
	}

	// Server identifier MUST NOT be filled in, requested IP address option MUST
	// NOT be filled in.
	return s.handleRenew(req)
}

// handleRequest is the handler for a DHCPREQUEST message.
//
// See https://datatracker.ietf.org/doc/html/rfc2131#section-4.3.2.
func (s *v4Server) handleRequest(req, resp *dhcpv4.DHCPv4) (lease *Lease, needsReply bool) {
	lease, needsReply = s.handleByRequestType(req)
	if lease == nil {
		return nil, needsReply
	}

	resp.UpdateOption(dhcpv4.OptMessageType(dhcpv4.MessageTypeAck))

	hostname := req.HostName()
	isRequested := hostname != "" || req.ParameterRequestList().Has(dhcpv4.OptionHostName)

	defer func() {
		s.conf.notify(LeaseChangedAdded)
		s.conf.notify(LeaseChangedDBStore)
	}()

	s.leasesLock.Lock()
	defer s.leasesLock.Unlock()

	if lease.IsStatic() {
		if lease.Hostname != "" {
			// TODO(e.burkov):  This option is used to update the server's DNS
			// mapping.  The option should only be answered when it has been
			// requested.
			resp.UpdateOption(OptionFQDN(lease.Hostname))
		}

		return lease, needsReply
	}

	s.commitLease(lease, hostname)

	if isRequested {
		resp.UpdateOption(dhcpv4.OptHostName(lease.Hostname))
	}

	return lease, needsReply
}

// handleDecline is the handler for the DHCP Decline request.
func (s *v4Server) handleDecline(req, resp *dhcpv4.DHCPv4) (err error) {
	s.conf.notify(LeaseChangedDBStore)

	s.leasesLock.Lock()
	defer s.leasesLock.Unlock()

	mac := req.ClientHWAddr
	reqIP := req.RequestedIPAddress()
	if reqIP == nil {
		reqIP = req.ClientIPAddr
	}

	var oldLease *Lease
	for _, l := range s.leases {
		if bytes.Equal(l.HWAddr, mac) && l.IP.Equal(reqIP) {
			oldLease = l

			break
		}
	}

	if oldLease == nil {
		log.Info("dhcpv4: lease with ip %s for %s not found", reqIP, mac)

		return nil
	}

	err = s.rmDynamicLease(oldLease)
	if err != nil {
		return fmt.Errorf("removing old lease for %s: %w", mac, err)
	}

	newLease, err := s.allocateLease(mac)
	if err != nil {
		return fmt.Errorf("allocating new lease for %s: %w", mac, err)
	} else if newLease == nil {
		log.Info("dhcpv4: allocating new lease for %s: no more ip addresses", mac)

		resp.YourIPAddr = make([]byte, 4)
		resp.UpdateOption(dhcpv4.OptMessageType(dhcpv4.MessageTypeAck))

		return nil
	}

	newLease.Hostname = oldLease.Hostname
	newLease.Expiry = time.Now().Add(s.conf.leaseTime)

	err = s.addLease(newLease)
	if err != nil {
		return fmt.Errorf("adding new lease for %s: %w", mac, err)
	}

	log.Info("dhcpv4: changed ip from %s to %s for %s", reqIP, newLease.IP, mac)

	resp.YourIPAddr = make([]byte, 4)
	copy(resp.YourIPAddr, newLease.IP)

	resp.UpdateOption(dhcpv4.OptMessageType(dhcpv4.MessageTypeAck))

	return nil
}

// handleRelease is the handler for the DHCP Release request.
func (s *v4Server) handleRelease(req, resp *dhcpv4.DHCPv4) (err error) {
	mac := req.ClientHWAddr
	reqIP := req.RequestedIPAddress()
	if reqIP == nil {
		reqIP = req.ClientIPAddr
	}

	// TODO(a.garipov): Add a separate notification type for dynamic lease
	// removal?
	defer s.conf.notify(LeaseChangedDBStore)

	n := 0
	s.leasesLock.Lock()
	defer s.leasesLock.Unlock()

	for _, l := range s.leases {
		if !bytes.Equal(l.HWAddr, mac) || !l.IP.Equal(reqIP) {
			continue
		}

		err = s.rmDynamicLease(l)
		if err != nil {
			err = fmt.Errorf("removing dynamic lease for %s: %w", mac, err)

			return
		}

		n++
	}

	log.Info("dhcpv4: released %d dynamic leases for %s", n, mac)

	resp.UpdateOption(dhcpv4.OptMessageType(dhcpv4.MessageTypeAck))

	return nil
}

// Find a lease associated with MAC and prepare response
// Return 1: OK
// Return 0: error; reply with Nak
// Return -1: error; don't reply
func (s *v4Server) handle(req, resp *dhcpv4.DHCPv4) int {
	var err error

	// Include server's identifier option since any reply should contain it.
	//
	// See https://datatracker.ietf.org/doc/html/rfc2131#page-29.
	resp.UpdateOption(dhcpv4.OptServerIdentifier(s.conf.dnsIPAddrs[0].AsSlice()))

	// TODO(a.garipov): Refactor this into handlers.
	var l *Lease
	switch mt := req.MessageType(); mt {
	case dhcpv4.MessageTypeDiscover:
		l, err = s.handleDiscover(req, resp)
		if err != nil {
			log.Error("dhcpv4: handling discover: %s", err)

			return 0
		}

		if l == nil {
			return 0
		}
	case dhcpv4.MessageTypeRequest:
		var toReply bool
		l, toReply = s.handleRequest(req, resp)
		if l == nil {
			if toReply {
				return 0
			}
			return -1 // drop packet
		}
	case dhcpv4.MessageTypeDecline:
		err = s.handleDecline(req, resp)
		if err != nil {
			log.Error("dhcpv4: handling decline: %s", err)

			return 0
		}
	case dhcpv4.MessageTypeRelease:
		err = s.handleRelease(req, resp)
		if err != nil {
			log.Error("dhcpv4: handling release: %s", err)

			return 0
		}
	}

	if l != nil {
		resp.YourIPAddr = slices.Clone(l.IP)
	}

	s.updateOptions(req, resp)

	return 1
}

// updateOptions updates the options of the response in accordance with the
// request and RFC 2131.
//
// See https://datatracker.ietf.org/doc/html/rfc2131#section-4.3.1.
func (s *v4Server) updateOptions(req, resp *dhcpv4.DHCPv4) {
	// Set IP address lease time for all DHCPOFFER messages and DHCPACK messages
	// replied for DHCPREQUEST.
	//
	// TODO(e.burkov):  Inspect why this is always set to configured value.
	resp.UpdateOption(dhcpv4.OptIPAddressLeaseTime(s.conf.leaseTime))

	// If the server recognizes the parameter as a parameter defined in the Host
	// Requirements Document, the server MUST include the default value for that
	// parameter.
	for _, code := range req.ParameterRequestList() {
		if val := s.implicitOpts.Get(code); val != nil {
			resp.UpdateOption(dhcpv4.OptGeneric(code, val))
		}
	}

	// If the server has been explicitly configured with a default value for the
	// parameter or the parameter has a non-default value on the client's
	// subnet, the server MUST include that value in an appropriate option.
	for code, val := range s.explicitOpts {
		if val != nil {
			resp.Options[code] = val
		} else {
			// Delete options explicitly configured to be removed.
			delete(resp.Options, code)
		}
	}
}

// client(0.0.0.0:68) -> (Request:ClientMAC,Type=Discover,ClientID,ReqIP,HostName) -> server(255.255.255.255:67)
// client(255.255.255.255:68) <- (Reply:YourIP,ClientMAC,Type=Offer,ServerID,SubnetMask,LeaseTime) <- server(<IP>:67)
// client(0.0.0.0:68) -> (Request:ClientMAC,Type=Request,ClientID,ReqIP||ClientIP,HostName,ServerID,ParamReqList) -> server(255.255.255.255:67)
// client(255.255.255.255:68) <- (Reply:YourIP,ClientMAC,Type=ACK,ServerID,SubnetMask,LeaseTime) <- server(<IP>:67)
func (s *v4Server) packetHandler(conn net.PacketConn, peer net.Addr, req *dhcpv4.DHCPv4) {
	log.Debug("dhcpv4: received message: %s", req.Summary())

	switch req.MessageType() {
	case
		dhcpv4.MessageTypeDiscover,
		dhcpv4.MessageTypeRequest,
		dhcpv4.MessageTypeDecline,
		dhcpv4.MessageTypeRelease:
		// Go on.
	default:
		log.Debug("dhcpv4: unsupported message type %d", req.MessageType())

		return
	}

	resp, err := dhcpv4.NewReplyFromRequest(req)
	if err != nil {
		log.Debug("dhcpv4: dhcpv4.New: %s", err)

		return
	}

	err = netutil.ValidateMAC(req.ClientHWAddr)
	if err != nil {
		log.Error("dhcpv4: invalid ClientHWAddr: %s", err)

		return
	}

	r := s.handle(req, resp)
	if r < 0 {
		return
	} else if r == 0 {
		resp.Options.Update(dhcpv4.OptMessageType(dhcpv4.MessageTypeNak))
	}

	s.send(peer, conn, req, resp)
}

// send writes resp for peer to conn considering the req's parameters according
// to RFC-2131.
//
// See https://datatracker.ietf.org/doc/html/rfc2131#section-4.1.
func (s *v4Server) send(peer net.Addr, conn net.PacketConn, req, resp *dhcpv4.DHCPv4) {
	switch giaddr, ciaddr, mtype := req.GatewayIPAddr, req.ClientIPAddr, resp.MessageType(); {
	case giaddr != nil && !giaddr.IsUnspecified():
		// Send any return messages to the server port on the BOOTP
		// relay agent whose address appears in giaddr.
		peer = &net.UDPAddr{
			IP:   giaddr,
			Port: dhcpv4.ServerPort,
		}
		if mtype == dhcpv4.MessageTypeNak {
			// Set the broadcast bit in the DHCPNAK, so that the relay agent
			// broadcasts it to the client, because the client may not have
			// a correct network address or subnet mask, and the client may not
			// be answering ARP requests.
			resp.SetBroadcast()
		}
	case mtype == dhcpv4.MessageTypeNak:
		// Broadcast any DHCPNAK messages to 0xffffffff.
	case ciaddr != nil && !ciaddr.IsUnspecified():
		// Unicast DHCPOFFER and DHCPACK messages to the address in
		// ciaddr.
		peer = &net.UDPAddr{
			IP:   ciaddr,
			Port: dhcpv4.ClientPort,
		}
	case !req.IsBroadcast() && req.ClientHWAddr != nil:
		// Unicast DHCPOFFER and DHCPACK messages to the client's
		// hardware address and yiaddr.
		peer = &dhcpUnicastAddr{
			Addr:   raw.Addr{HardwareAddr: req.ClientHWAddr},
			yiaddr: resp.YourIPAddr,
		}
	default:
		// Go on since peer is already set to broadcast.
	}

	pktData := resp.ToBytes()

	log.Debug("dhcpv4: sending %d bytes to %s: %s", len(pktData), peer, resp.Summary())

	_, err := conn.WriteTo(pktData, peer)
	if err != nil {
		log.Error("dhcpv4: conn.Write to %s failed: %s", peer, err)
	}
}

// Start starts the IPv4 DHCP server.
func (s *v4Server) Start() (err error) {
	defer func() { err = errors.Annotate(err, "dhcpv4: %w") }()

	if !s.enabled() {
		return nil
	}

	ifaceName := s.conf.InterfaceName
	iface, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return fmt.Errorf("finding interface %s by name: %w", ifaceName, err)
	}

	log.Debug("dhcpv4: starting...")

	dnsIPAddrs, err := aghnet.IfaceDNSIPAddrs(
		iface,
		aghnet.IPVersion4,
		defaultMaxAttempts,
		defaultBackoff,
	)
	if err != nil {
		return fmt.Errorf("interface %s: %w", ifaceName, err)
	}

	if len(dnsIPAddrs) == 0 {
		// No available IP addresses which may appear later.
		return nil
	}
	// Update the value of Domain Name Server option separately from others if
	// not assigned yet since its value is available only at server's start.
	//
	// TODO(e.burkov):  Initialize as implicit option with the rest of default
	// options when it will be possible to do before the call to Start.
	if !s.explicitOpts.Has(dhcpv4.OptionDomainNameServer) {
		s.implicitOpts.Update(dhcpv4.OptDNS(dnsIPAddrs...))
	}

	for _, ip := range dnsIPAddrs {
		ip = ip.To4()
		if ip == nil {
			continue
		}

		s.conf.dnsIPAddrs = append(s.conf.dnsIPAddrs, netip.AddrFrom4(*(*[4]byte)(ip)))
	}

	var c net.PacketConn
	if c, err = s.newDHCPConn(iface); err != nil {
		return err
	}

	s.srv, err = server4.NewServer(
		iface.Name,
		nil,
		s.packetHandler,
		server4.WithConn(c),
		server4.WithDebugLogger(),
	)
	if err != nil {
		return err
	}

	log.Info("dhcpv4: listening")

	go func() {
		if serr := s.srv.Serve(); errors.Is(serr, net.ErrClosed) {
			log.Info("dhcpv4: server is closed")
		} else if serr != nil {
			log.Error("dhcpv4: srv.Serve: %s", serr)
		}
	}()

	// Signal to the clients containers in packages home and dnsforward that
	// it should reload the DHCP clients.
	s.conf.notify(LeaseChangedAdded)

	return nil
}

// Stop - stop server
func (s *v4Server) Stop() (err error) {
	if s.srv == nil {
		return
	}

	log.Debug("dhcpv4: stopping")
	err = s.srv.Close()
	if err != nil {
		return fmt.Errorf("closing dhcpv4 srv: %w", err)
	}

	// Signal to the clients containers in packages home and dnsforward that
	// it should remove all DHCP clients.
	s.conf.notify(LeaseChangedRemovedAll)

	s.srv = nil

	return nil
}

// Create DHCPv4 server
func v4Create(conf *V4ServerConf) (srv *v4Server, err error) {
	s := &v4Server{
		leaseHosts: stringutil.NewSet(),
	}

	err = conf.Validate()
	if err != nil {
		// TODO(a.garipov): Don't use a disabled server in other places or just
		// use an interface.
		return s, err
	}

	s.conf = &V4ServerConf{}
	*s.conf = *conf

	// TODO(a.garipov, d.seregin): Check that every lease is inside the IPRange.
	s.leasedOffsets = newBitSet()

	if conf.LeaseDuration == 0 {
		s.conf.leaseTime = timeutil.Day
		s.conf.LeaseDuration = uint32(s.conf.leaseTime.Seconds())
	} else {
		s.conf.leaseTime = time.Second * time.Duration(conf.LeaseDuration)
	}

	s.prepareOptions()

	return s, nil
}
