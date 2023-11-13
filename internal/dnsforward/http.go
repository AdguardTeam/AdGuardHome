package dnsforward

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/stringutil"
	"github.com/miekg/dns"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

// jsonDNSConfig is the JSON representation of the DNS server configuration.
//
// TODO(s.chzhen):  Split it into smaller pieces.  Use aghalg.NullBool instead
// of *bool.
type jsonDNSConfig struct {
	// Upstreams is the list of upstream DNS servers.
	Upstreams *[]string `json:"upstream_dns"`

	// UpstreamsFile is the file containing upstream DNS servers.
	UpstreamsFile *string `json:"upstream_dns_file"`

	// Bootstraps is the list of DNS servers resolving IP addresses of the
	// upstream DoH/DoT resolvers.
	Bootstraps *[]string `json:"bootstrap_dns"`

	// Fallbacks is the list of fallback DNS servers used when upstream DNS
	// servers are not responding.
	Fallbacks *[]string `json:"fallback_dns"`

	// ProtectionEnabled defines if protection is enabled.
	ProtectionEnabled *bool `json:"protection_enabled"`

	// RateLimit is the number of requests per second allowed per client.
	RateLimit *uint32 `json:"ratelimit"`

	// BlockingMode defines the way blocked responses are constructed.
	BlockingMode *filtering.BlockingMode `json:"blocking_mode"`

	// EDNSCSEnabled defines if EDNS Client Subnet is enabled.
	EDNSCSEnabled *bool `json:"edns_cs_enabled"`

	// EDNSCSUseCustom defines if EDNSCSCustomIP should be used.
	EDNSCSUseCustom *bool `json:"edns_cs_use_custom"`

	// DNSSECEnabled defines if DNSSEC is enabled.
	DNSSECEnabled *bool `json:"dnssec_enabled"`

	// DisableIPv6 defines if IPv6 addresses should be dropped.
	DisableIPv6 *bool `json:"disable_ipv6"`

	// UpstreamMode defines the way DNS requests are constructed.
	UpstreamMode *string `json:"upstream_mode"`

	// BlockedResponseTTL is the TTL for blocked responses.
	BlockedResponseTTL *uint32 `json:"blocked_response_ttl"`

	// CacheSize in bytes.
	CacheSize *uint32 `json:"cache_size"`

	// CacheMinTTL is custom minimum TTL for cached DNS responses.
	CacheMinTTL *uint32 `json:"cache_ttl_min"`

	// CacheMaxTTL is custom maximum TTL for cached DNS responses.
	CacheMaxTTL *uint32 `json:"cache_ttl_max"`

	// CacheOptimistic defines if expired entries should be served.
	CacheOptimistic *bool `json:"cache_optimistic"`

	// ResolveClients defines if clients IPs should be resolved into hostnames.
	ResolveClients *bool `json:"resolve_clients"`

	// UsePrivateRDNS defines if privates DNS resolvers should be used.
	UsePrivateRDNS *bool `json:"use_private_ptr_resolvers"`

	// LocalPTRUpstreams is the list of local private DNS resolvers.
	LocalPTRUpstreams *[]string `json:"local_ptr_upstreams"`

	// BlockingIPv4 is custom IPv4 address for blocked A requests.
	BlockingIPv4 netip.Addr `json:"blocking_ipv4"`

	// BlockingIPv6 is custom IPv6 address for blocked AAAA requests.
	BlockingIPv6 netip.Addr `json:"blocking_ipv6"`

	// DisabledUntil is a timestamp until when the protection is disabled.
	DisabledUntil *time.Time `json:"protection_disabled_until"`

	// EDNSCSCustomIP is custom IP for EDNS Client Subnet.
	EDNSCSCustomIP netip.Addr `json:"edns_cs_custom_ip"`

	// DefaultLocalPTRUpstreams is used to pass the addresses from
	// systemResolvers to the front-end.  It's not a pointer to the slice since
	// there is no need to omit it while decoding from JSON.
	DefaultLocalPTRUpstreams []string `json:"default_local_ptr_upstreams,omitempty"`
}

func (s *Server) getDNSConfig() (c *jsonDNSConfig) {
	protectionEnabled, protectionDisabledUntil := s.UpdatedProtectionStatus()

	s.serverLock.RLock()
	defer s.serverLock.RUnlock()

	upstreams := stringutil.CloneSliceOrEmpty(s.conf.UpstreamDNS)
	upstreamFile := s.conf.UpstreamDNSFileName
	bootstraps := stringutil.CloneSliceOrEmpty(s.conf.BootstrapDNS)
	fallbacks := stringutil.CloneSliceOrEmpty(s.conf.FallbackDNS)
	blockingMode, blockingIPv4, blockingIPv6 := s.dnsFilter.BlockingMode()
	blockedResponseTTL := s.dnsFilter.BlockedResponseTTL()
	ratelimit := s.conf.Ratelimit

	customIP := s.conf.EDNSClientSubnet.CustomIP
	enableEDNSClientSubnet := s.conf.EDNSClientSubnet.Enabled
	useCustom := s.conf.EDNSClientSubnet.UseCustom

	enableDNSSEC := s.conf.EnableDNSSEC
	aaaaDisabled := s.conf.AAAADisabled
	cacheSize := s.conf.CacheSize
	cacheMinTTL := s.conf.CacheMinTTL
	cacheMaxTTL := s.conf.CacheMaxTTL
	cacheOptimistic := s.conf.CacheOptimistic
	resolveClients := s.conf.AddrProcConf.UseRDNS
	usePrivateRDNS := s.conf.UsePrivateRDNS
	localPTRUpstreams := stringutil.CloneSliceOrEmpty(s.conf.LocalPTRResolvers)

	var upstreamMode string
	if s.conf.FastestAddr {
		upstreamMode = "fastest_addr"
	} else if s.conf.AllServers {
		upstreamMode = "parallel"
	}

	defPTRUps, err := s.defaultLocalPTRUpstreams()
	if err != nil {
		log.Error("dnsforward: %s", err)
	}

	return &jsonDNSConfig{
		Upstreams:                &upstreams,
		UpstreamsFile:            &upstreamFile,
		Bootstraps:               &bootstraps,
		Fallbacks:                &fallbacks,
		ProtectionEnabled:        &protectionEnabled,
		BlockingMode:             &blockingMode,
		BlockingIPv4:             blockingIPv4,
		BlockingIPv6:             blockingIPv6,
		RateLimit:                &ratelimit,
		EDNSCSCustomIP:           customIP,
		EDNSCSEnabled:            &enableEDNSClientSubnet,
		EDNSCSUseCustom:          &useCustom,
		DNSSECEnabled:            &enableDNSSEC,
		DisableIPv6:              &aaaaDisabled,
		BlockedResponseTTL:       &blockedResponseTTL,
		CacheSize:                &cacheSize,
		CacheMinTTL:              &cacheMinTTL,
		CacheMaxTTL:              &cacheMaxTTL,
		CacheOptimistic:          &cacheOptimistic,
		UpstreamMode:             &upstreamMode,
		ResolveClients:           &resolveClients,
		UsePrivateRDNS:           &usePrivateRDNS,
		LocalPTRUpstreams:        &localPTRUpstreams,
		DefaultLocalPTRUpstreams: defPTRUps,
		DisabledUntil:            protectionDisabledUntil,
	}
}

// defaultLocalPTRUpstreams returns the list of default local PTR resolvers
// filtered of AdGuard Home's own DNS server addresses.  It may appear empty.
func (s *Server) defaultLocalPTRUpstreams() (ups []string, err error) {
	matcher, err := s.conf.ourAddrsMatcher()
	if err != nil {
		// Don't wrap the error because it's informative enough as is.
		return nil, err
	}

	sysResolvers := slices.DeleteFunc(s.sysResolvers.Addrs(), matcher)
	ups = make([]string, 0, len(sysResolvers))
	for _, r := range sysResolvers {
		ups = append(ups, r.String())
	}

	return ups, nil
}

// handleGetConfig handles requests to the GET /control/dns_info endpoint.
func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	resp := s.getDNSConfig()
	aghhttp.WriteJSONResponseOK(w, r, resp)
}

func (req *jsonDNSConfig) checkBlockingMode() (err error) {
	if req.BlockingMode == nil {
		return nil
	}

	return validateBlockingMode(*req.BlockingMode, req.BlockingIPv4, req.BlockingIPv6)
}

func (req *jsonDNSConfig) checkUpstreamsMode() bool {
	valid := []string{"", "fastest_addr", "parallel"}

	return req.UpstreamMode == nil || stringutil.InSlice(valid, *req.UpstreamMode)
}

func (req *jsonDNSConfig) checkBootstrap() (err error) {
	if req.Bootstraps == nil {
		return nil
	}

	var b string
	defer func() { err = errors.Annotate(err, "checking bootstrap %s: invalid address: %w", b) }()

	for _, b = range *req.Bootstraps {
		if b == "" {
			return errors.Error("empty")
		}

		if _, err = upstream.NewUpstreamResolver(b, nil); err != nil {
			return err
		}
	}

	return nil
}

// checkFallbacks returns an error if any fallback address is invalid.
func (req *jsonDNSConfig) checkFallbacks() (err error) {
	if req.Fallbacks == nil {
		return nil
	}

	err = ValidateUpstreams(*req.Fallbacks)
	if err != nil {
		return fmt.Errorf("validating fallback servers: %w", err)
	}

	return nil
}

// validate returns an error if any field of req is invalid.
func (req *jsonDNSConfig) validate(privateNets netutil.SubnetSet) (err error) {
	if req.Upstreams != nil {
		err = ValidateUpstreams(*req.Upstreams)
		if err != nil {
			return fmt.Errorf("validating upstream servers: %w", err)
		}
	}

	if req.LocalPTRUpstreams != nil {
		err = ValidateUpstreamsPrivate(*req.LocalPTRUpstreams, privateNets)
		if err != nil {
			return fmt.Errorf("validating private upstream servers: %w", err)
		}
	}

	err = req.checkBootstrap()
	if err != nil {
		return err
	}

	err = req.checkFallbacks()
	if err != nil {
		return err
	}

	err = req.checkBlockingMode()
	if err != nil {
		return err
	}

	switch {
	case !req.checkUpstreamsMode():
		return errors.Error("upstream_mode: incorrect value")
	case !req.checkCacheTTL():
		return errors.Error("cache_ttl_min must be less or equal than cache_ttl_max")
	default:
		return nil
	}
}

func (req *jsonDNSConfig) checkCacheTTL() bool {
	if req.CacheMinTTL == nil && req.CacheMaxTTL == nil {
		return true
	}

	var min, max uint32
	if req.CacheMinTTL != nil {
		min = *req.CacheMinTTL
	}
	if req.CacheMaxTTL != nil {
		max = *req.CacheMaxTTL
	}

	return min <= max
}

// handleSetConfig handles requests to the POST /control/dns_config endpoint.
func (s *Server) handleSetConfig(w http.ResponseWriter, r *http.Request) {
	req := &jsonDNSConfig{}
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "decoding request: %s", err)

		return
	}

	err = req.validate(s.privateNets)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "%s", err)

		return
	}

	restart := s.setConfig(req)
	s.conf.ConfigModified()

	if restart {
		err = s.Reconfigure(nil)
		if err != nil {
			aghhttp.Error(r, w, http.StatusInternalServerError, "%s", err)
		}
	}
}

// setConfig sets the server parameters.  shouldRestart is true if the server
// should be restarted to apply changes.
func (s *Server) setConfig(dc *jsonDNSConfig) (shouldRestart bool) {
	s.serverLock.Lock()
	defer s.serverLock.Unlock()

	if dc.BlockingMode != nil {
		s.dnsFilter.SetBlockingMode(*dc.BlockingMode, dc.BlockingIPv4, dc.BlockingIPv6)
	}

	if dc.BlockedResponseTTL != nil {
		s.dnsFilter.SetBlockedResponseTTL(*dc.BlockedResponseTTL)
	}

	if dc.ProtectionEnabled != nil {
		s.dnsFilter.SetProtectionEnabled(*dc.ProtectionEnabled)
	}

	if dc.UpstreamMode != nil {
		s.conf.AllServers = *dc.UpstreamMode == "parallel"
		s.conf.FastestAddr = *dc.UpstreamMode == "fastest_addr"
	}

	if dc.EDNSCSUseCustom != nil && *dc.EDNSCSUseCustom {
		s.conf.EDNSClientSubnet.CustomIP = dc.EDNSCSCustomIP
	}

	setIfNotNil(&s.conf.EnableDNSSEC, dc.DNSSECEnabled)
	setIfNotNil(&s.conf.AAAADisabled, dc.DisableIPv6)

	return s.setConfigRestartable(dc)
}

// setIfNotNil sets the value pointed at by currentPtr to the value pointed at
// by newPtr if newPtr is not nil.  currentPtr must not be nil.
func setIfNotNil[T any](currentPtr, newPtr *T) (hasSet bool) {
	if newPtr == nil {
		return false
	}

	*currentPtr = *newPtr

	return true
}

// setConfigRestartable sets the parameters which trigger a restart.
// shouldRestart is true if the server should be restarted to apply changes.
// s.serverLock is expected to be locked.
//
// TODO(a.garipov): Some of these could probably be updated without a restart.
// Inspect and consider refactoring.
func (s *Server) setConfigRestartable(dc *jsonDNSConfig) (shouldRestart bool) {
	for _, hasSet := range []bool{
		setIfNotNil(&s.conf.UpstreamDNS, dc.Upstreams),
		setIfNotNil(&s.conf.LocalPTRResolvers, dc.LocalPTRUpstreams),
		setIfNotNil(&s.conf.UpstreamDNSFileName, dc.UpstreamsFile),
		setIfNotNil(&s.conf.BootstrapDNS, dc.Bootstraps),
		setIfNotNil(&s.conf.FallbackDNS, dc.Fallbacks),
		setIfNotNil(&s.conf.EDNSClientSubnet.Enabled, dc.EDNSCSEnabled),
		setIfNotNil(&s.conf.EDNSClientSubnet.UseCustom, dc.EDNSCSUseCustom),
		setIfNotNil(&s.conf.CacheSize, dc.CacheSize),
		setIfNotNil(&s.conf.CacheMinTTL, dc.CacheMinTTL),
		setIfNotNil(&s.conf.CacheMaxTTL, dc.CacheMaxTTL),
		setIfNotNil(&s.conf.CacheOptimistic, dc.CacheOptimistic),
		setIfNotNil(&s.conf.AddrProcConf.UseRDNS, dc.ResolveClients),
		setIfNotNil(&s.conf.UsePrivateRDNS, dc.UsePrivateRDNS),
	} {
		shouldRestart = shouldRestart || hasSet
		if shouldRestart {
			break
		}
	}

	if dc.RateLimit != nil && s.conf.Ratelimit != *dc.RateLimit {
		s.conf.Ratelimit = *dc.RateLimit
		shouldRestart = true
	}

	return shouldRestart
}

// upstreamJSON is a request body for handleTestUpstreamDNS endpoint.
type upstreamJSON struct {
	Upstreams        []string `json:"upstream_dns"`
	BootstrapDNS     []string `json:"bootstrap_dns"`
	FallbackDNS      []string `json:"fallback_dns"`
	PrivateUpstreams []string `json:"private_upstream"`
}

// IsCommentOrEmpty returns true if s starts with a "#" character or is empty.
// This function is useful for filtering out non-upstream lines from upstream
// configs.
func IsCommentOrEmpty(s string) (ok bool) {
	return len(s) == 0 || s[0] == '#'
}

// newUpstreamConfig validates upstreams and returns an appropriate upstream
// configuration or nil if it can't be built.
//
// TODO(e.burkov):  Perhaps proxy.ParseUpstreamsConfig should validate upstreams
// slice already so that this function may be considered useless.
func newUpstreamConfig(upstreams []string) (conf *proxy.UpstreamConfig, err error) {
	// No need to validate comments and empty lines.
	upstreams = stringutil.FilterOut(upstreams, IsCommentOrEmpty)
	if len(upstreams) == 0 {
		// Consider this case valid since it means the default server should be
		// used.
		return nil, nil
	}

	err = validateUpstreamConfig(upstreams)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return nil, err
	}

	conf, err = proxy.ParseUpstreamsConfig(
		upstreams,
		&upstream.Options{
			Bootstrap: []string{},
			Timeout:   DefaultTimeout,
		},
	)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return nil, err
	} else if len(conf.Upstreams) == 0 {
		return nil, errors.Error("no default upstreams specified")
	}

	return conf, nil
}

// validateUpstreamConfig validates each upstream from the upstream
// configuration and returns an error if any upstream is invalid.
//
// TODO(e.burkov):  Move into aghnet or even into dnsproxy.
func validateUpstreamConfig(conf []string) (err error) {
	for _, u := range conf {
		var ups []string
		var domains []string
		ups, domains, err = separateUpstream(u)
		if err != nil {
			// Don't wrap the error since it's informative enough as is.
			return err
		}

		for _, addr := range ups {
			_, err = validateUpstream(addr, domains)
			if err != nil {
				return fmt.Errorf("validating upstream %q: %w", addr, err)
			}
		}
	}

	return nil
}

// ValidateUpstreams validates each upstream and returns an error if any
// upstream is invalid or if there are no default upstreams specified.
//
// TODO(e.burkov):  Move into aghnet or even into dnsproxy.
func ValidateUpstreams(upstreams []string) (err error) {
	_, err = newUpstreamConfig(upstreams)

	return err
}

// ValidateUpstreamsPrivate validates each upstream and returns an error if any
// upstream is invalid or if there are no default upstreams specified.  It also
// checks each domain of domain-specific upstreams for being ARPA pointing to
// a locally-served network.  privateNets must not be nil.
func ValidateUpstreamsPrivate(upstreams []string, privateNets netutil.SubnetSet) (err error) {
	conf, err := newUpstreamConfig(upstreams)
	if err != nil {
		return fmt.Errorf("creating config: %w", err)
	}

	if conf == nil {
		return nil
	}

	keys := maps.Keys(conf.DomainReservedUpstreams)
	slices.Sort(keys)

	var errs []error
	for _, domain := range keys {
		var subnet netip.Prefix
		subnet, err = extractARPASubnet(domain)
		if err != nil {
			errs = append(errs, err)

			continue
		}

		if !privateNets.Contains(subnet.Addr().AsSlice()) {
			errs = append(
				errs,
				fmt.Errorf("arpa domain %q should point to a locally-served network", domain),
			)
		}
	}

	return errors.Annotate(errors.Join(errs...), "checking domain-specific upstreams: %w")
}

var protocols = []string{
	"h3://",
	"https://",
	"quic://",
	"sdns://",
	"tcp://",
	"tls://",
	"udp://",
}

// validateUpstream returns an error if u alongside with domains is not a valid
// upstream configuration.  useDefault is true if the upstream is
// domain-specific and is configured to point at the default upstream server
// which is validated separately.  The upstream is considered domain-specific
// only if domains is at least not nil.
func validateUpstream(u string, domains []string) (useDefault bool, err error) {
	// The special server address '#' means that default server must be used.
	if useDefault = u == "#" && domains != nil; useDefault {
		return useDefault, nil
	}

	// Check if the upstream has a valid protocol prefix.
	//
	// TODO(e.burkov):  Validate the domain name.
	for _, proto := range protocols {
		if strings.HasPrefix(u, proto) {
			return false, nil
		}
	}

	if proto, _, ok := strings.Cut(u, "://"); ok {
		return false, fmt.Errorf("bad protocol %q", proto)
	}

	// Check if upstream is either an IP or IP with port.
	if _, err = netip.ParseAddr(u); err == nil {
		return false, nil
	} else if _, err = netip.ParseAddrPort(u); err == nil {
		return false, nil
	}

	return false, err
}

// separateUpstream returns the upstreams and the specified domains.  domains
// is nil when the upstream is not domains-specific.  Otherwise it may also be
// empty.
func separateUpstream(upstreamStr string) (upstreams, domains []string, err error) {
	if !strings.HasPrefix(upstreamStr, "[/") {
		return []string{upstreamStr}, nil, nil
	}

	defer func() { err = errors.Annotate(err, "bad upstream for domain %q: %w", upstreamStr) }()

	parts := strings.Split(upstreamStr[2:], "/]")
	switch len(parts) {
	case 2:
		// Go on.
	case 1:
		return nil, nil, errors.Error("missing separator")
	default:
		return nil, nil, errors.Error("duplicated separator")
	}

	for i, host := range strings.Split(parts[0], "/") {
		if host == "" {
			continue
		}

		err = netutil.ValidateDomainName(strings.TrimPrefix(host, "*."))
		if err != nil {
			return nil, nil, fmt.Errorf("domain at index %d: %w", i, err)
		}

		domains = append(domains, host)
	}

	return strings.Fields(parts[1]), domains, nil
}

// healthCheckFunc is a signature of function to check if upstream exchanges
// properly.
type healthCheckFunc func(u upstream.Upstream) (err error)

// checkDNSUpstreamExc checks if the DNS upstream exchanges correctly.
func checkDNSUpstreamExc(u upstream.Upstream) (err error) {
	// testTLD is the special-use fully-qualified domain name for testing the
	// DNS server reachability.
	//
	// See https://datatracker.ietf.org/doc/html/rfc6761#section-6.2.
	const testTLD = "test."

	req := &dns.Msg{
		MsgHdr: dns.MsgHdr{
			Id:               dns.Id(),
			RecursionDesired: true,
		},
		Question: []dns.Question{{
			Name:   testTLD,
			Qtype:  dns.TypeA,
			Qclass: dns.ClassINET,
		}},
	}

	var reply *dns.Msg
	reply, err = u.Exchange(req)
	if err != nil {
		return fmt.Errorf("couldn't communicate with upstream: %w", err)
	} else if len(reply.Answer) != 0 {
		return errors.Error("wrong response")
	}

	return nil
}

// checkPrivateUpstreamExc checks if the upstream for resolving private
// addresses exchanges correctly.
//
// TODO(e.burkov):  Think about testing the ip6.arpa. as well.
func checkPrivateUpstreamExc(u upstream.Upstream) (err error) {
	// inAddrArpaTLD is the special-use fully-qualified domain name for PTR IP
	// address resolution.
	//
	// See https://datatracker.ietf.org/doc/html/rfc1035#section-3.5.
	const inAddrArpaTLD = "in-addr.arpa."

	req := &dns.Msg{
		MsgHdr: dns.MsgHdr{
			Id:               dns.Id(),
			RecursionDesired: true,
		},
		Question: []dns.Question{{
			Name:   inAddrArpaTLD,
			Qtype:  dns.TypePTR,
			Qclass: dns.ClassINET,
		}},
	}

	if _, err = u.Exchange(req); err != nil {
		return fmt.Errorf("couldn't communicate with upstream: %w", err)
	}

	return nil
}

// domainSpecificTestError is a wrapper for errors returned by checkDNS to mark
// the tested upstream domain-specific and therefore consider its errors
// non-critical.
//
// TODO(a.garipov):  Some common mechanism of distinguishing between errors and
// warnings (non-critical errors) is desired.
type domainSpecificTestError struct {
	error
}

// Error implements the [error] interface for domainSpecificTestError.
func (err domainSpecificTestError) Error() (msg string) {
	return fmt.Sprintf("WARNING: %s", err.error)
}

// checkDNS parses line, creates DNS upstreams using opts, and checks if the
// upstreams are exchanging correctly.  It saves the result into a sync.Map
// where key is an upstream address and value is "OK", if the upstream
// exchanges correctly, or text of the error.  It is intended to be used as a
// goroutine.
//
// TODO(s.chzhen):  Separate to a different structure/file.
func (s *Server) checkDNS(
	line string,
	opts *upstream.Options,
	check healthCheckFunc,
	wg *sync.WaitGroup,
	m *sync.Map,
) {
	defer wg.Done()
	defer log.OnPanic("dnsforward: checking upstreams")

	upstreams, domains, err := separateUpstream(line)
	if err != nil {
		err = fmt.Errorf("wrong upstream format: %w", err)
		m.Store(line, err.Error())

		return
	}

	specific := len(domains) > 0

	for _, upstreamAddr := range upstreams {
		var useDefault bool
		useDefault, err = validateUpstream(upstreamAddr, domains)
		if err != nil {
			err = fmt.Errorf("wrong upstream format: %w", err)
			m.Store(upstreamAddr, err.Error())

			continue
		}

		if useDefault {
			continue
		}

		log.Debug("dnsforward: checking if upstream %q works", upstreamAddr)

		err = s.checkUpstreamAddr(upstreamAddr, specific, opts, check)
		if err != nil {
			m.Store(upstreamAddr, err.Error())
		} else {
			m.Store(upstreamAddr, "OK")
		}
	}
}

// checkUpstreamAddr creates the DNS upstream using opts and information from
// [s.dnsFilter.EtcHosts].  Checks if the DNS upstream exchanges correctly.  It
// returns an error if addr is not valid DNS upstream address or the upstream
// is not exchanging correctly.
func (s *Server) checkUpstreamAddr(
	addr string,
	specific bool,
	opts *upstream.Options,
	check healthCheckFunc,
) (err error) {
	defer func() {
		if err != nil && specific {
			err = domainSpecificTestError{error: err}
		}
	}()

	opts = &upstream.Options{
		Bootstrap:  opts.Bootstrap,
		Timeout:    opts.Timeout,
		PreferIPv6: opts.PreferIPv6,
	}

	// dnsFilter can be nil during application update.
	if s.dnsFilter != nil {
		recs := s.dnsFilter.EtcHostsRecords(extractUpstreamHost(addr))
		for _, rec := range recs {
			opts.ServerIPAddrs = append(opts.ServerIPAddrs, rec.Addr.AsSlice())
		}
		sortNetIPAddrs(opts.ServerIPAddrs, opts.PreferIPv6)
	}

	u, err := upstream.AddressToUpstream(addr, opts)
	if err != nil {
		return fmt.Errorf("creating upstream for %q: %w", addr, err)
	}

	defer func() { err = errors.WithDeferred(err, u.Close()) }()

	return check(u)
}

// handleTestUpstreamDNS handles requests to the POST /control/test_upstream_dns
// endpoint.
func (s *Server) handleTestUpstreamDNS(w http.ResponseWriter, r *http.Request) {
	req := &upstreamJSON{}
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "Failed to read request body: %s", err)

		return
	}

	req.Upstreams = stringutil.FilterOut(req.Upstreams, IsCommentOrEmpty)
	req.FallbackDNS = stringutil.FilterOut(req.FallbackDNS, IsCommentOrEmpty)
	req.PrivateUpstreams = stringutil.FilterOut(req.PrivateUpstreams, IsCommentOrEmpty)

	opts := &upstream.Options{
		Bootstrap:  req.BootstrapDNS,
		Timeout:    s.conf.UpstreamTimeout,
		PreferIPv6: s.conf.BootstrapPreferIPv6,
	}
	if len(opts.Bootstrap) == 0 {
		opts.Bootstrap = defaultBootstrap
	}

	wg := &sync.WaitGroup{}
	m := &sync.Map{}

	wg.Add(len(req.Upstreams) + len(req.FallbackDNS) + len(req.PrivateUpstreams))

	for _, ups := range req.Upstreams {
		go s.checkDNS(ups, opts, checkDNSUpstreamExc, wg, m)
	}
	for _, ups := range req.FallbackDNS {
		go s.checkDNS(ups, opts, checkDNSUpstreamExc, wg, m)
	}
	for _, ups := range req.PrivateUpstreams {
		go s.checkDNS(ups, opts, checkPrivateUpstreamExc, wg, m)
	}

	wg.Wait()

	result := map[string]string{}
	m.Range(func(k, v any) bool {
		// TODO(e.burkov):  The upstreams used for both common and private
		// resolving should be reported separately.
		ups := k.(string)
		status := v.(string)

		result[ups] = status

		return true
	})

	aghhttp.WriteJSONResponseOK(w, r, result)
}

// handleCacheClear is the handler for the POST /control/cache_clear HTTP API.
func (s *Server) handleCacheClear(w http.ResponseWriter, _ *http.Request) {
	s.dnsProxy.ClearCache()
	_, _ = io.WriteString(w, "OK")
}

// protectionJSON is an object for /control/protection endpoint.
type protectionJSON struct {
	Enabled  bool `json:"enabled"`
	Duration uint `json:"duration"`
}

// handleSetProtection is a handler for the POST /control/protection HTTP API.
func (s *Server) handleSetProtection(w http.ResponseWriter, r *http.Request) {
	protectionReq := &protectionJSON{}
	err := json.NewDecoder(r.Body).Decode(protectionReq)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "reading req: %s", err)

		return
	}

	var disabledUntil *time.Time
	if protectionReq.Duration > 0 {
		if protectionReq.Enabled {
			aghhttp.Error(
				r,
				w,
				http.StatusBadRequest,
				"Setting a duration is only allowed with protection disabling",
			)

			return
		}

		calcTime := time.Now().Add(time.Duration(protectionReq.Duration) * time.Millisecond)
		disabledUntil = &calcTime
	}

	func() {
		s.serverLock.Lock()
		defer s.serverLock.Unlock()

		s.dnsFilter.SetProtectionStatus(protectionReq.Enabled, disabledUntil)
	}()

	s.conf.ConfigModified()

	aghhttp.OK(w)
}

// handleDoH is the DNS-over-HTTPs handler.
//
// Control flow:
//
//	HTTP server
//	-> dnsforward.handleDoH
//	-> dnsforward.ServeHTTP
//	-> proxy.ServeHTTP
//	-> proxy.handleDNSRequest
//	-> dnsforward.handleDNSRequest
func (s *Server) handleDoH(w http.ResponseWriter, r *http.Request) {
	if !s.conf.TLSAllowUnencryptedDoH && r.TLS == nil {
		aghhttp.Error(r, w, http.StatusNotFound, "Not Found")

		return
	}

	if !s.IsRunning() {
		aghhttp.Error(r, w, http.StatusInternalServerError, "dns server is not running")

		return
	}

	s.ServeHTTP(w, r)
}

func (s *Server) registerHandlers() {
	if webRegistered || s.conf.HTTPRegister == nil {
		return
	}

	s.conf.HTTPRegister(http.MethodGet, "/control/dns_info", s.handleGetConfig)
	s.conf.HTTPRegister(http.MethodPost, "/control/dns_config", s.handleSetConfig)
	s.conf.HTTPRegister(http.MethodPost, "/control/test_upstream_dns", s.handleTestUpstreamDNS)
	s.conf.HTTPRegister(http.MethodPost, "/control/protection", s.handleSetProtection)

	s.conf.HTTPRegister(http.MethodGet, "/control/access/list", s.handleAccessList)
	s.conf.HTTPRegister(http.MethodPost, "/control/access/set", s.handleAccessSet)

	s.conf.HTTPRegister(http.MethodPost, "/control/cache_clear", s.handleCacheClear)

	// Register both versions, with and without the trailing slash, to
	// prevent a 301 Moved Permanently redirect when clients request the
	// path without the trailing slash.  Those redirects break some clients.
	//
	// See go doc net/http.ServeMux.
	//
	// See also https://github.com/AdguardTeam/AdGuardHome/issues/2628.
	s.conf.HTTPRegister("", "/dns-query", s.handleDoH)
	s.conf.HTTPRegister("", "/dns-query/", s.handleDoH)

	webRegistered = true
}
