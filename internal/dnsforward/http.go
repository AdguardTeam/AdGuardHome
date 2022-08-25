package dnsforward

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
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

type dnsConfig struct {
	Upstreams     *[]string `json:"upstream_dns"`
	UpstreamsFile *string   `json:"upstream_dns_file"`
	Bootstraps    *[]string `json:"bootstrap_dns"`

	ProtectionEnabled *bool         `json:"protection_enabled"`
	RateLimit         *uint32       `json:"ratelimit"`
	BlockingMode      *BlockingMode `json:"blocking_mode"`
	BlockingIPv4      net.IP        `json:"blocking_ipv4"`
	BlockingIPv6      net.IP        `json:"blocking_ipv6"`
	EDNSCSEnabled     *bool         `json:"edns_cs_enabled"`
	DNSSECEnabled     *bool         `json:"dnssec_enabled"`
	DisableIPv6       *bool         `json:"disable_ipv6"`
	UpstreamMode      *string       `json:"upstream_mode"`
	CacheSize         *uint32       `json:"cache_size"`
	CacheMinTTL       *uint32       `json:"cache_ttl_min"`
	CacheMaxTTL       *uint32       `json:"cache_ttl_max"`
	CacheOptimistic   *bool         `json:"cache_optimistic"`
	ResolveClients    *bool         `json:"resolve_clients"`
	UsePrivateRDNS    *bool         `json:"use_private_ptr_resolvers"`
	LocalPTRUpstreams *[]string     `json:"local_ptr_upstreams"`
}

func (s *Server) getDNSConfig() (c *dnsConfig) {
	s.serverLock.RLock()
	defer s.serverLock.RUnlock()

	upstreams := stringutil.CloneSliceOrEmpty(s.conf.UpstreamDNS)
	upstreamFile := s.conf.UpstreamDNSFileName
	bootstraps := stringutil.CloneSliceOrEmpty(s.conf.BootstrapDNS)
	protectionEnabled := s.conf.ProtectionEnabled
	blockingMode := s.conf.BlockingMode
	blockingIPv4 := s.conf.BlockingIPv4
	blockingIPv6 := s.conf.BlockingIPv6
	ratelimit := s.conf.Ratelimit
	enableEDNSClientSubnet := s.conf.EnableEDNSClientSubnet
	enableDNSSEC := s.conf.EnableDNSSEC
	aaaaDisabled := s.conf.AAAADisabled
	cacheSize := s.conf.CacheSize
	cacheMinTTL := s.conf.CacheMinTTL
	cacheMaxTTL := s.conf.CacheMaxTTL
	cacheOptimistic := s.conf.CacheOptimistic
	resolveClients := s.conf.ResolveClients
	usePrivateRDNS := s.conf.UsePrivateRDNS
	localPTRUpstreams := stringutil.CloneSliceOrEmpty(s.conf.LocalPTRResolvers)
	var upstreamMode string
	if s.conf.FastestAddr {
		upstreamMode = "fastest_addr"
	} else if s.conf.AllServers {
		upstreamMode = "parallel"
	}

	return &dnsConfig{
		Upstreams:         &upstreams,
		UpstreamsFile:     &upstreamFile,
		Bootstraps:        &bootstraps,
		ProtectionEnabled: &protectionEnabled,
		BlockingMode:      &blockingMode,
		BlockingIPv4:      blockingIPv4,
		BlockingIPv6:      blockingIPv6,
		RateLimit:         &ratelimit,
		EDNSCSEnabled:     &enableEDNSClientSubnet,
		DNSSECEnabled:     &enableDNSSEC,
		DisableIPv6:       &aaaaDisabled,
		CacheSize:         &cacheSize,
		CacheMinTTL:       &cacheMinTTL,
		CacheMaxTTL:       &cacheMaxTTL,
		CacheOptimistic:   &cacheOptimistic,
		UpstreamMode:      &upstreamMode,
		ResolveClients:    &resolveClients,
		UsePrivateRDNS:    &usePrivateRDNS,
		LocalPTRUpstreams: &localPTRUpstreams,
	}
}

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	defLocalPTRUps, err := s.filterOurDNSAddrs(s.sysResolvers.Get())
	if err != nil {
		log.Debug("getting dns configuration: %s", err)
	}

	resp := struct {
		dnsConfig
		// DefautLocalPTRUpstreams is used to pass the addresses from
		// systemResolvers to the front-end.  It's not a pointer to the slice
		// since there is no need to omit it while decoding from JSON.
		DefautLocalPTRUpstreams []string `json:"default_local_ptr_upstreams,omitempty"`
	}{
		dnsConfig:               *s.getDNSConfig(),
		DefautLocalPTRUpstreams: defLocalPTRUps,
	}

	w.Header().Set("Content-Type", "application/json")

	if err = json.NewEncoder(w).Encode(resp); err != nil {
		aghhttp.Error(r, w, http.StatusInternalServerError, "json.Encoder: %s", err)

		return
	}
}

func (req *dnsConfig) checkBlockingMode() bool {
	if req.BlockingMode == nil {
		return true
	}

	switch bm := *req.BlockingMode; bm {
	case BlockingModeDefault,
		BlockingModeREFUSED,
		BlockingModeNXDOMAIN,
		BlockingModeNullIP:
		return true
	case BlockingModeCustomIP:
		return req.BlockingIPv4.To4() != nil && req.BlockingIPv6 != nil
	default:
		return false
	}
}

func (req *dnsConfig) checkUpstreamsMode() bool {
	valid := []string{"", "fastest_addr", "parallel"}

	return req.UpstreamMode == nil || stringutil.InSlice(valid, *req.UpstreamMode)
}

func (req *dnsConfig) checkBootstrap() (err error) {
	if req.Bootstraps == nil {
		return nil
	}

	var b string
	defer func() { err = errors.Annotate(err, "checking bootstrap %s: invalid address: %w", b) }()

	for _, b = range *req.Bootstraps {
		if b == "" {
			return errors.Error("empty")
		}

		if _, err = upstream.NewResolver(b, nil); err != nil {
			return err
		}
	}

	return nil
}

// validate returns an error if any field of req is invalid.
func (req *dnsConfig) validate(privateNets netutil.SubnetSet) (err error) {
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

	switch {
	case !req.checkBlockingMode():
		return errors.Error("blocking_mode: incorrect value")
	case !req.checkUpstreamsMode():
		return errors.Error("upstream_mode: incorrect value")
	case !req.checkCacheTTL():
		return errors.Error("cache_ttl_min must be less or equal than cache_ttl_max")
	default:
		return nil
	}
}

func (req *dnsConfig) checkCacheTTL() bool {
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

func (s *Server) handleSetConfig(w http.ResponseWriter, r *http.Request) {
	req := &dnsConfig{}
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

func (s *Server) setConfigRestartable(dc *dnsConfig) (restart bool) {
	if dc.Upstreams != nil {
		s.conf.UpstreamDNS = *dc.Upstreams
		restart = true
	}

	if dc.LocalPTRUpstreams != nil {
		s.conf.LocalPTRResolvers = *dc.LocalPTRUpstreams
		restart = true
	}

	if dc.UpstreamsFile != nil {
		s.conf.UpstreamDNSFileName = *dc.UpstreamsFile
		restart = true
	}

	if dc.Bootstraps != nil {
		s.conf.BootstrapDNS = *dc.Bootstraps
		restart = true
	}

	if dc.RateLimit != nil && s.conf.Ratelimit != *dc.RateLimit {
		s.conf.Ratelimit = *dc.RateLimit
		restart = true
	}

	if dc.EDNSCSEnabled != nil {
		s.conf.EnableEDNSClientSubnet = *dc.EDNSCSEnabled
		restart = true
	}

	if dc.CacheSize != nil {
		s.conf.CacheSize = *dc.CacheSize
		restart = true
	}

	if dc.CacheMinTTL != nil {
		s.conf.CacheMinTTL = *dc.CacheMinTTL
		restart = true
	}

	if dc.CacheMaxTTL != nil {
		s.conf.CacheMaxTTL = *dc.CacheMaxTTL
		restart = true
	}

	if dc.CacheOptimistic != nil {
		s.conf.CacheOptimistic = *dc.CacheOptimistic
		restart = true
	}

	return restart
}

func (s *Server) setConfig(dc *dnsConfig) (restart bool) {
	s.serverLock.Lock()
	defer s.serverLock.Unlock()

	if dc.ProtectionEnabled != nil {
		s.conf.ProtectionEnabled = *dc.ProtectionEnabled
	}

	if dc.BlockingMode != nil {
		s.conf.BlockingMode = *dc.BlockingMode
		if *dc.BlockingMode == "custom_ip" {
			s.conf.BlockingIPv4 = dc.BlockingIPv4.To4()
			s.conf.BlockingIPv6 = dc.BlockingIPv6.To16()
		}
	}

	if dc.DNSSECEnabled != nil {
		s.conf.EnableDNSSEC = *dc.DNSSECEnabled
	}

	if dc.DisableIPv6 != nil {
		s.conf.AAAADisabled = *dc.DisableIPv6
	}

	if dc.UpstreamMode != nil {
		s.conf.AllServers = *dc.UpstreamMode == "parallel"
		s.conf.FastestAddr = *dc.UpstreamMode == "fastest_addr"
	}

	if dc.ResolveClients != nil {
		s.conf.ResolveClients = *dc.ResolveClients
	}

	if dc.UsePrivateRDNS != nil {
		s.conf.UsePrivateRDNS = *dc.UsePrivateRDNS
	}

	return s.setConfigRestartable(dc)
}

// upstreamJSON is a request body for handleTestUpstreamDNS endpoint.
type upstreamJSON struct {
	Upstreams        []string `json:"upstream_dns"`
	BootstrapDNS     []string `json:"bootstrap_dns"`
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

	for _, u := range upstreams {
		var ups string
		var domains []string
		ups, domains, err = separateUpstream(u)
		if err != nil {
			// Don't wrap the error since it's informative enough as is.
			return nil, err
		}

		_, err = validateUpstream(ups, domains)
		if err != nil {
			return nil, fmt.Errorf("validating upstream %q: %w", u, err)
		}
	}

	conf, err = proxy.ParseUpstreamsConfig(
		upstreams,
		&upstream.Options{Bootstrap: []string{}, Timeout: DefaultTimeout},
	)
	if err != nil {
		return nil, err
	} else if len(conf.Upstreams) == 0 {
		return nil, errors.Error("no default upstreams specified")
	}

	return conf, nil
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
		return err
	}

	if conf == nil {
		return nil
	}

	keys := maps.Keys(conf.DomainReservedUpstreams)
	slices.Sort(keys)

	var errs []error
	for _, domain := range keys {
		var subnet *net.IPNet
		subnet, err = netutil.SubnetFromReversedAddr(domain)
		if err != nil {
			errs = append(errs, err)

			continue
		}

		if !privateNets.Contains(subnet.IP) {
			errs = append(
				errs,
				fmt.Errorf("arpa domain %q should point to a locally-served network", domain),
			)
		}
	}

	if len(errs) > 0 {
		return errors.List("checking domain-specific upstreams", errs...)
	}

	return nil
}

var protocols = []string{"udp://", "tcp://", "tls://", "https://", "sdns://", "quic://"}

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
			return useDefault, nil
		}
	}

	if strings.Contains(u, "://") {
		return useDefault, errors.Error("wrong protocol")
	}

	// Check if upstream is either an IP or IP with port.
	if net.ParseIP(u) != nil {
		return useDefault, nil
	} else if _, err = netutil.ParseIPPort(u); err != nil {
		return useDefault, err
	}

	return useDefault, nil
}

// separateUpstream returns the upstream and the specified domains.  domains is
// nil when the upstream is not domains-specific.  Otherwise it may also be
// empty.
func separateUpstream(upstreamStr string) (ups string, domains []string, err error) {
	if !strings.HasPrefix(upstreamStr, "[/") {
		return upstreamStr, nil, nil
	}

	defer func() { err = errors.Annotate(err, "bad upstream for domain %q: %w", upstreamStr) }()

	parts := strings.Split(upstreamStr[2:], "/]")
	switch len(parts) {
	case 2:
		// Go on.
	case 1:
		return "", nil, errors.Error("missing separator")
	default:
		return "", []string{}, errors.Error("duplicated separator")
	}

	for i, host := range strings.Split(parts[0], "/") {
		if host == "" {
			continue
		}

		err = netutil.ValidateDomainName(strings.TrimPrefix(host, "*."))
		if err != nil {
			return "", domains, fmt.Errorf("domain at index %d: %w", i, err)
		}

		domains = append(domains, host)
	}

	return parts[1], domains, nil
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

// checkDNS checks the upstream server defined by upstreamConfigStr using
// healthCheck for actually exchange messages.  It uses bootstrap to resolve the
// upstream's address.
func checkDNS(
	upstreamConfigStr string,
	bootstrap []string,
	timeout time.Duration,
	healthCheck healthCheckFunc,
) (err error) {
	if IsCommentOrEmpty(upstreamConfigStr) {
		return nil
	}

	// Separate upstream from domains list.
	upstreamAddr, domains, err := separateUpstream(upstreamConfigStr)
	if err != nil {
		return fmt.Errorf("wrong upstream format: %w", err)
	}

	useDefault, err := validateUpstream(upstreamAddr, domains)
	if err != nil {
		return fmt.Errorf("wrong upstream format: %w", err)
	} else if useDefault {
		return nil
	}

	if len(bootstrap) == 0 {
		bootstrap = defaultBootstrap
	}

	log.Debug("dnsforward: checking if upstream %q works", upstreamAddr)

	u, err := upstream.AddressToUpstream(upstreamAddr, &upstream.Options{
		Bootstrap: bootstrap,
		Timeout:   timeout,
	})
	if err != nil {
		return fmt.Errorf("failed to choose upstream for %q: %w", upstreamAddr, err)
	}

	if err = healthCheck(u); err != nil {
		err = fmt.Errorf("upstream %q fails to exchange: %w", upstreamAddr, err)
		if domains != nil {
			return domainSpecificTestError{error: err}
		}

		return err
	}

	log.Debug("dnsforward: upstream %q is ok", upstreamAddr)

	return nil
}

func (s *Server) handleTestUpstreamDNS(w http.ResponseWriter, r *http.Request) {
	req := &upstreamJSON{}
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "Failed to read request body: %s", err)

		return
	}

	result := map[string]string{}
	bootstraps := req.BootstrapDNS

	timeout := s.conf.UpstreamTimeout
	for _, host := range req.Upstreams {
		err = checkDNS(host, bootstraps, timeout, checkDNSUpstreamExc)
		if err != nil {
			log.Info("%v", err)
			result[host] = err.Error()
			if _, ok := err.(domainSpecificTestError); ok {
				result[host] = fmt.Sprintf("WARNING: %s", result[host])
			}

			continue
		}

		result[host] = "OK"
	}

	for _, host := range req.PrivateUpstreams {
		err = checkDNS(host, bootstraps, timeout, checkPrivateUpstreamExc)
		if err != nil {
			log.Info("%v", err)
			// TODO(e.burkov): If passed upstream have already written an error
			// above, we rewriting the error for it.  These cases should be
			// handled properly instead.
			result[host] = err.Error()
			if _, ok := err.(domainSpecificTestError); ok {
				result[host] = fmt.Sprintf("WARNING: %s", result[host])
			}

			continue
		}

		result[host] = "OK"
	}

	jsonVal, err := json.Marshal(result)
	if err != nil {
		aghhttp.Error(
			r,
			w,
			http.StatusInternalServerError,
			"Unable to marshal status json: %s",
			err,
		)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonVal)
	if err != nil {
		aghhttp.Error(r, w, http.StatusInternalServerError, "Couldn't write body: %s", err)
	}
}

// Control flow:
// web
//  -> dnsforward.handleDoH -> dnsforward.ServeHTTP
//  -> proxy.ServeHTTP -> proxy.handleDNSRequest
//  -> dnsforward.handleDNSRequest
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
	s.conf.HTTPRegister(http.MethodGet, "/control/dns_info", s.handleGetConfig)
	s.conf.HTTPRegister(http.MethodPost, "/control/dns_config", s.handleSetConfig)
	s.conf.HTTPRegister(http.MethodPost, "/control/test_upstream_dns", s.handleTestUpstreamDNS)

	s.conf.HTTPRegister(http.MethodGet, "/control/access/list", s.handleAccessList)
	s.conf.HTTPRegister(http.MethodPost, "/control/access/set", s.handleAccessSet)

	// Register both versions, with and without the trailing slash, to
	// prevent a 301 Moved Permanently redirect when clients request the
	// path without the trailing slash.  Those redirects break some clients.
	//
	// See go doc net/http.ServeMux.
	//
	// See also https://github.com/AdguardTeam/AdGuardHome/issues/2628.
	s.conf.HTTPRegister("", "/dns-query", s.handleDoH)
	s.conf.HTTPRegister("", "/dns-query/", s.handleDoH)
}
