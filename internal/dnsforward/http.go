package dnsforward

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/aghstrings"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/miekg/dns"
)

func httpError(r *http.Request, w http.ResponseWriter, code int, format string, args ...interface{}) {
	text := fmt.Sprintf(format, args...)
	log.Info("dns: %s %s: %s", r.Method, r.URL, text)
	http.Error(w, text, code)
}

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

func (s *Server) getDNSConfig() dnsConfig {
	s.serverLock.RLock()
	defer s.serverLock.RUnlock()

	upstreams := aghstrings.CloneSliceOrEmpty(s.conf.UpstreamDNS)
	upstreamFile := s.conf.UpstreamDNSFileName
	bootstraps := aghstrings.CloneSliceOrEmpty(s.conf.BootstrapDNS)
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
	localPTRUpstreams := aghstrings.CloneSliceOrEmpty(s.conf.LocalPTRResolvers)
	var upstreamMode string
	if s.conf.FastestAddr {
		upstreamMode = "fastest_addr"
	} else if s.conf.AllServers {
		upstreamMode = "parallel"
	}

	return dnsConfig{
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
		dnsConfig:               s.getDNSConfig(),
		DefautLocalPTRUpstreams: defLocalPTRUps,
	}

	w.Header().Set("Content-Type", "application/json")

	if err = json.NewEncoder(w).Encode(resp); err != nil {
		httpError(r, w, http.StatusInternalServerError, "json.Encoder: %s", err)
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
	if req.UpstreamMode == nil {
		return true
	}

	for _, valid := range []string{
		"",
		"fastest_addr",
		"parallel",
	} {
		if *req.UpstreamMode == valid {
			return true
		}
	}

	return false
}

func (req *dnsConfig) checkBootstrap() (string, error) {
	if req.Bootstraps == nil {
		return "", nil
	}

	for _, boot := range *req.Bootstraps {
		if boot == "" {
			return boot, fmt.Errorf("invalid bootstrap server address: empty")
		}

		if _, err := upstream.NewResolver(boot, nil); err != nil {
			return boot, fmt.Errorf("invalid bootstrap server address: %w", err)
		}
	}

	return "", nil
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
	req := dnsConfig{}
	dec := json.NewDecoder(r.Body)
	if err := dec.Decode(&req); err != nil {
		httpError(r, w, http.StatusBadRequest, "json Encode: %s", err)
		return
	}

	if req.Upstreams != nil {
		if err := ValidateUpstreams(*req.Upstreams); err != nil {
			httpError(r, w, http.StatusBadRequest, "wrong upstreams specification: %s", err)
			return
		}
	}

	if errBoot, err := req.checkBootstrap(); err != nil {
		httpError(r, w, http.StatusBadRequest, "%s can not be used as bootstrap dns cause: %s", errBoot, err)
		return
	}

	if !req.checkBlockingMode() {
		httpError(r, w, http.StatusBadRequest, "blocking_mode: incorrect value")
		return
	}

	if !req.checkUpstreamsMode() {
		httpError(r, w, http.StatusBadRequest, "upstream_mode: incorrect value")
		return
	}

	if !req.checkCacheTTL() {
		httpError(r, w, http.StatusBadRequest, "cache_ttl_min must be less or equal than cache_ttl_max")
		return
	}

	restart := s.setConfig(req)
	s.conf.ConfigModified()

	if restart {
		if err := s.Reconfigure(nil); err != nil {
			httpError(r, w, http.StatusInternalServerError, "%s", err)
			return
		}
	}
}

func (s *Server) setConfigRestartable(dc dnsConfig) (restart bool) {
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

	if dc.RateLimit != nil {
		restart = restart || s.conf.Ratelimit != *dc.RateLimit
		s.conf.Ratelimit = *dc.RateLimit
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

func (s *Server) setConfig(dc dnsConfig) (restart bool) {
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

// ValidateUpstreams validates each upstream and returns an error if any
// upstream is invalid or if there are no default upstreams specified.
//
// TODO(e.burkov): Move into aghnet or even into dnsproxy.
func ValidateUpstreams(upstreams []string) (err error) {
	// No need to validate comments
	upstreams = aghstrings.FilterOut(upstreams, aghstrings.IsCommentOrEmpty)

	// Consider this case valid because defaultDNS will be used
	if len(upstreams) == 0 {
		return nil
	}

	_, err = proxy.ParseUpstreamsConfig(
		upstreams,
		&upstream.Options{
			Bootstrap: []string{},
			Timeout:   DefaultTimeout,
		},
	)
	if err != nil {
		return err
	}

	var defaultUpstreamFound bool
	for _, u := range upstreams {
		var ok bool
		ok, err = validateUpstream(u)
		if err != nil {
			return err
		}

		if !defaultUpstreamFound {
			defaultUpstreamFound = ok
		}
	}

	if !defaultUpstreamFound {
		return fmt.Errorf("no default upstreams specified")
	}

	return nil
}

var protocols = []string{"tls://", "https://", "tcp://", "sdns://", "quic://"}

func validateUpstream(u string) (bool, error) {
	// Check if the user tries to specify upstream for domain.
	u, useDefault, err := separateUpstream(u)
	if err != nil {
		return useDefault, err
	}

	// The special server address '#' means "use the default servers"
	if u == "#" && !useDefault {
		return useDefault, nil
	}

	// Check if the upstream has a valid protocol prefix
	for _, proto := range protocols {
		if strings.HasPrefix(u, proto) {
			return useDefault, nil
		}
	}

	// Return error if the upstream contains '://' without any valid protocol
	if strings.Contains(u, "://") {
		return useDefault, fmt.Errorf("wrong protocol")
	}

	// Check if upstream is valid plain DNS
	return useDefault, checkPlainDNS(u)
}

// separateUpstream returns the upstream without the specified domains.
// useDefault is true when a default upstream must be used.
func separateUpstream(upstreamStr string) (upstream string, useDefault bool, err error) {
	defer func() { err = errors.Annotate(err, "bad upstream for domain spec %q: %w", upstreamStr) }()

	if !strings.HasPrefix(upstreamStr, "[/") {
		return upstreamStr, true, nil
	}

	parts := strings.Split(upstreamStr[2:], "/]")
	if len(parts) != 2 {
		return "", false, errors.Error("duplicated separator")
	}

	domains := parts[0]
	upstream = parts[1]
	for i, host := range strings.Split(domains, "/") {
		if host == "" {
			continue
		}

		err = aghnet.ValidateDomainName(host)
		if err != nil {
			return "", false, fmt.Errorf("domain at index %d: %w", i, err)
		}
	}

	return upstream, false, nil
}

// checkPlainDNS checks if host is plain DNS
func checkPlainDNS(upstream string) error {
	// Check if host is ip without port
	if net.ParseIP(upstream) != nil {
		return nil
	}

	// Check if host is ip with port
	ip, port, err := net.SplitHostPort(upstream)
	if err != nil {
		return err
	}

	if net.ParseIP(ip) == nil {
		return fmt.Errorf("%s is not a valid IP", ip)
	}

	_, err = strconv.ParseInt(port, 0, 64)
	if err != nil {
		return fmt.Errorf("%s is not a valid port: %w", port, err)
	}

	return nil
}

// excFunc is a signature of function to check if upstream exchanges correctly.
type excFunc func(u upstream.Upstream) (err error)

// checkDNSUpstreamExc checks if the DNS upstream exchanges correctly.
func checkDNSUpstreamExc(u upstream.Upstream) (err error) {
	req := &dns.Msg{
		MsgHdr: dns.MsgHdr{
			Id:               dns.Id(),
			RecursionDesired: true,
		},
		Question: []dns.Question{{
			Name:   "google-public-dns-a.google.com.",
			Qtype:  dns.TypeA,
			Qclass: dns.ClassINET,
		}},
	}

	var reply *dns.Msg
	reply, err = u.Exchange(req)
	if err != nil {
		return fmt.Errorf("couldn't communicate with upstream: %w", err)
	}

	if len(reply.Answer) != 1 {
		return fmt.Errorf("wrong response")
	}

	if t, ok := reply.Answer[0].(*dns.A); ok {
		if !net.IPv4(8, 8, 8, 8).Equal(t.A) {
			return fmt.Errorf("wrong response")
		}
	}

	return nil
}

// checkPrivateUpstreamExc checks if the upstream for resolving private
// addresses exchanges correctly.
func checkPrivateUpstreamExc(u upstream.Upstream) (err error) {
	req := &dns.Msg{
		MsgHdr: dns.MsgHdr{
			Id:               dns.Id(),
			RecursionDesired: true,
		},
		Question: []dns.Question{{
			Name:   "1.0.0.127.in-addr.arpa.",
			Qtype:  dns.TypePTR,
			Qclass: dns.ClassINET,
		}},
	}

	if _, err = u.Exchange(req); err != nil {
		return fmt.Errorf("couldn't communicate with upstream: %w", err)
	}

	return nil
}

func checkDNS(input string, bootstrap []string, timeout time.Duration, ef excFunc) (err error) {
	if aghstrings.IsCommentOrEmpty(input) {
		return nil
	}

	// Separate upstream from domains list.
	var useDefault bool
	if input, useDefault, err = separateUpstream(input); err != nil {
		return fmt.Errorf("wrong upstream format: %w", err)
	}

	// No need to check this DNS server.
	if !useDefault {
		return nil
	}

	if _, err = validateUpstream(input); err != nil {
		return fmt.Errorf("wrong upstream format: %w", err)
	}

	if len(bootstrap) == 0 {
		bootstrap = defaultBootstrap
	}

	log.Debug("checking if dns server %q works...", input)
	var u upstream.Upstream
	u, err = upstream.AddressToUpstream(input, &upstream.Options{
		Bootstrap: bootstrap,
		Timeout:   timeout,
	})
	if err != nil {
		return fmt.Errorf("failed to choose upstream for %q: %w", input, err)
	}

	if err = ef(u); err != nil {
		return fmt.Errorf("upstream %q fails to exchange: %w", input, err)
	}

	log.Debug("dns %s works OK", input)

	return nil
}

func (s *Server) handleTestUpstreamDNS(w http.ResponseWriter, r *http.Request) {
	req := &upstreamJSON{}
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		httpError(r, w, http.StatusBadRequest, "Failed to read request body: %s", err)

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

			continue
		}

		result[host] = "OK"
	}

	for _, host := range req.PrivateUpstreams {
		err = checkDNS(host, bootstraps, timeout, checkPrivateUpstreamExc)
		if err != nil {
			log.Info("%v", err)
			// TODO(e.burkov): If passed upstream have already
			// written an error above, we rewriting the error for
			// it.  These cases should be handled properly instead.
			result[host] = err.Error()

			continue
		}

		result[host] = "OK"
	}

	jsonVal, err := json.Marshal(result)
	if err != nil {
		httpError(r, w, http.StatusInternalServerError, "Unable to marshal status json: %s", err)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonVal)
	if err != nil {
		httpError(r, w, http.StatusInternalServerError, "Couldn't write body: %s", err)

		return
	}
}

// Control flow:
// web
//  -> dnsforward.handleDoH -> dnsforward.ServeHTTP
//  -> proxy.ServeHTTP -> proxy.handleDNSRequest
//  -> dnsforward.handleDNSRequest
func (s *Server) handleDoH(w http.ResponseWriter, r *http.Request) {
	if !s.conf.TLSAllowUnencryptedDoH && r.TLS == nil {
		httpError(r, w, http.StatusNotFound, "Not Found")
		return
	}

	if !s.IsRunning() {
		httpError(r, w, http.StatusInternalServerError, "dns server is not running")
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
