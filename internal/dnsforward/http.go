package dnsforward

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/utils"
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

	ProtectionEnabled *bool   `json:"protection_enabled"`
	RateLimit         *uint32 `json:"ratelimit"`
	BlockingMode      *string `json:"blocking_mode"`
	BlockingIPv4      *string `json:"blocking_ipv4"`
	BlockingIPv6      *string `json:"blocking_ipv6"`
	EDNSCSEnabled     *bool   `json:"edns_cs_enabled"`
	DNSSECEnabled     *bool   `json:"dnssec_enabled"`
	DisableIPv6       *bool   `json:"disable_ipv6"`
	UpstreamMode      *string `json:"upstream_mode"`
	CacheSize         *uint32 `json:"cache_size"`
	CacheMinTTL       *uint32 `json:"cache_ttl_min"`
	CacheMaxTTL       *uint32 `json:"cache_ttl_max"`
}

func (s *Server) getDNSConfig() dnsConfig {
	s.RLock()
	upstreams := stringArrayDup(s.conf.UpstreamDNS)
	upstreamFile := s.conf.UpstreamDNSFileName
	bootstraps := stringArrayDup(s.conf.BootstrapDNS)
	protectionEnabled := s.conf.ProtectionEnabled
	blockingMode := s.conf.BlockingMode
	BlockingIPv4 := s.conf.BlockingIPv4
	BlockingIPv6 := s.conf.BlockingIPv6
	Ratelimit := s.conf.Ratelimit
	EnableEDNSClientSubnet := s.conf.EnableEDNSClientSubnet
	EnableDNSSEC := s.conf.EnableDNSSEC
	AAAADisabled := s.conf.AAAADisabled
	CacheSize := s.conf.CacheSize
	CacheMinTTL := s.conf.CacheMinTTL
	CacheMaxTTL := s.conf.CacheMaxTTL
	var upstreamMode string
	if s.conf.FastestAddr {
		upstreamMode = "fastest_addr"
	} else if s.conf.AllServers {
		upstreamMode = "parallel"
	}
	s.RUnlock()
	return dnsConfig{
		Upstreams:         &upstreams,
		UpstreamsFile:     &upstreamFile,
		Bootstraps:        &bootstraps,
		ProtectionEnabled: &protectionEnabled,
		BlockingMode:      &blockingMode,
		BlockingIPv4:      &BlockingIPv4,
		BlockingIPv6:      &BlockingIPv6,
		RateLimit:         &Ratelimit,
		EDNSCSEnabled:     &EnableEDNSClientSubnet,
		DNSSECEnabled:     &EnableDNSSEC,
		DisableIPv6:       &AAAADisabled,
		CacheSize:         &CacheSize,
		CacheMinTTL:       &CacheMinTTL,
		CacheMaxTTL:       &CacheMaxTTL,
		UpstreamMode:      &upstreamMode,
	}
}

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	resp := s.getDNSConfig()

	w.Header().Set("Content-Type", "application/json")

	enc := json.NewEncoder(w)
	if err := enc.Encode(resp); err != nil {
		httpError(r, w, http.StatusInternalServerError, "json.Encoder: %s", err)
		return
	}
}

func (req *dnsConfig) checkBlockingMode() bool {
	if req.BlockingMode == nil {
		return true
	}

	bm := *req.BlockingMode
	if bm == "custom_ip" {
		if req.BlockingIPv4 == nil || req.BlockingIPv6 == nil {
			return false
		}

		ip4 := net.ParseIP(*req.BlockingIPv4)
		if ip4 == nil || ip4.To4() == nil {
			return false
		}

		ip6 := net.ParseIP(*req.BlockingIPv6)
		return ip6 != nil
	}

	for _, valid := range []string{
		"default",
		"refused",
		"nxdomain",
		"null_ip",
	} {
		if bm == valid {
			return true
		}
	}

	return false
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

		if _, err := upstream.NewResolver(boot, 0); err != nil {
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
	if req.CacheMaxTTL == nil {
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

	if s.setConfig(req) {
		if err := s.Reconfigure(nil); err != nil {
			httpError(r, w, http.StatusInternalServerError, "%s", err)
			return
		}
	}
}

func (s *Server) setConfig(dc dnsConfig) (restart bool) {
	s.Lock()

	if dc.Upstreams != nil {
		s.conf.UpstreamDNS = *dc.Upstreams
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

	if dc.ProtectionEnabled != nil {
		s.conf.ProtectionEnabled = *dc.ProtectionEnabled
	}

	if dc.BlockingMode != nil {
		s.conf.BlockingMode = *dc.BlockingMode
		if *dc.BlockingMode == "custom_ip" {
			s.conf.BlockingIPv4 = *dc.BlockingIPv4
			s.conf.BlockingIPAddrv4 = net.ParseIP(*dc.BlockingIPv4)
			s.conf.BlockingIPv6 = *dc.BlockingIPv6
			s.conf.BlockingIPAddrv6 = net.ParseIP(*dc.BlockingIPv6)
		}
	}

	if dc.RateLimit != nil {
		if s.conf.Ratelimit != *dc.RateLimit {
			restart = true
		}
		s.conf.Ratelimit = *dc.RateLimit
	}

	if dc.EDNSCSEnabled != nil {
		s.conf.EnableEDNSClientSubnet = *dc.EDNSCSEnabled
		restart = true
	}

	if dc.DNSSECEnabled != nil {
		s.conf.EnableDNSSEC = *dc.DNSSECEnabled
	}

	if dc.DisableIPv6 != nil {
		s.conf.AAAADisabled = *dc.DisableIPv6
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

	if dc.UpstreamMode != nil {
		switch *dc.UpstreamMode {
		case "parallel":
			s.conf.AllServers = true
			s.conf.FastestAddr = false
		case "fastest_addr":
			s.conf.AllServers = false
			s.conf.FastestAddr = true
		default:
			s.conf.AllServers = false
			s.conf.FastestAddr = false
		}
	}
	s.Unlock()
	s.conf.ConfigModified()
	return restart
}

type upstreamJSON struct {
	Upstreams    []string `json:"upstream_dns"`  // Upstreams
	BootstrapDNS []string `json:"bootstrap_dns"` // Bootstrap DNS
}

// ValidateUpstreams validates each upstream and returns an error if any upstream is invalid or if there are no default upstreams specified
func ValidateUpstreams(upstreams []string) error {
	// No need to validate comments
	upstreams = filterOutComments(upstreams)

	// Consider this case valid because defaultDNS will be used
	if len(upstreams) == 0 {
		return nil
	}

	var defaultUpstreamFound bool
	for _, u := range upstreams {
		d, err := validateUpstream(u)
		if err != nil {
			return err
		}

		// Check this flag until default upstream will not be found
		if !defaultUpstreamFound {
			defaultUpstreamFound = d
		}
	}

	// Return error if there are no default upstreams
	if !defaultUpstreamFound {
		return fmt.Errorf("no default upstreams specified")
	}

	return nil
}

var protocols = []string{"tls://", "https://", "tcp://", "sdns://", "quic://"}

func validateUpstream(u string) (bool, error) {
	// Check if user tries to specify upstream for domain
	u, defaultUpstream, err := separateUpstream(u)
	if err != nil {
		return defaultUpstream, err
	}

	// The special server address '#' means "use the default servers"
	if u == "#" && !defaultUpstream {
		return defaultUpstream, nil
	}

	// Check if the upstream has a valid protocol prefix
	for _, proto := range protocols {
		if strings.HasPrefix(u, proto) {
			return defaultUpstream, nil
		}
	}

	// Return error if the upstream contains '://' without any valid protocol
	if strings.Contains(u, "://") {
		return defaultUpstream, fmt.Errorf("wrong protocol")
	}

	// Check if upstream is valid plain DNS
	return defaultUpstream, checkPlainDNS(u)
}

// separateUpstream returns upstream without specified domains and a bool flag that indicates if no domains were specified
// error will be returned if upstream per domain specification is invalid
func separateUpstream(upstream string) (string, bool, error) {
	defaultUpstream := true
	if strings.HasPrefix(upstream, "[/") {
		defaultUpstream = false
		// split domains and upstream string
		domainsAndUpstream := strings.Split(strings.TrimPrefix(upstream, "[/"), "/]")
		if len(domainsAndUpstream) != 2 {
			return "", defaultUpstream, fmt.Errorf("wrong dns upstream per domain specification: %s", upstream)
		}

		// split domains list and validate each one
		for _, host := range strings.Split(domainsAndUpstream[0], "/") {
			if host != "" {
				if err := utils.IsValidHostname(host); err != nil {
					return "", defaultUpstream, err
				}
			}
		}
		upstream = domainsAndUpstream[1]
	}
	return upstream, defaultUpstream, nil
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

func (s *Server) handleTestUpstreamDNS(w http.ResponseWriter, r *http.Request) {
	req := upstreamJSON{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		httpError(r, w, http.StatusBadRequest, "Failed to read request body: %s", err)
		return
	}

	result := map[string]string{}

	for _, host := range req.Upstreams {
		err = checkDNS(host, req.BootstrapDNS)
		if err != nil {
			log.Info("%v", err)
			result[host] = err.Error()
		} else {
			result[host] = "OK"
		}
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

func checkDNS(input string, bootstrap []string) error {
	if !isUpstream(input) {
		return nil
	}

	// separate upstream from domains list
	input, defaultUpstream, err := separateUpstream(input)
	if err != nil {
		return fmt.Errorf("wrong upstream format: %w", err)
	}

	// No need to check this DNS server
	if !defaultUpstream {
		return nil
	}

	if _, err := validateUpstream(input); err != nil {
		return fmt.Errorf("wrong upstream format: %w", err)
	}

	if len(bootstrap) == 0 {
		bootstrap = defaultBootstrap
	}

	log.Debug("checking if dns %s works...", input)
	u, err := upstream.AddressToUpstream(input, upstream.Options{Bootstrap: bootstrap, Timeout: DefaultTimeout})
	if err != nil {
		return fmt.Errorf("failed to choose upstream for %s: %w", input, err)
	}

	req := dns.Msg{}
	req.Id = dns.Id()
	req.RecursionDesired = true
	req.Question = []dns.Question{
		{Name: "google-public-dns-a.google.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET},
	}
	reply, err := u.Exchange(&req)
	if err != nil {
		return fmt.Errorf("couldn't communicate with dns server %s: %w", input, err)
	}
	if len(reply.Answer) != 1 {
		return fmt.Errorf("dns server %s returned wrong answer", input)
	}
	if t, ok := reply.Answer[0].(*dns.A); ok {
		if !net.IPv4(8, 8, 8, 8).Equal(t.A) {
			return fmt.Errorf("dns server %s returned wrong answer: %v", input, t.A)
		}
	}

	log.Debug("dns %s works OK", input)
	return nil
}

// Control flow:
// web
//  -> dnsforward.handleDOH -> dnsforward.ServeHTTP
//  -> proxy.ServeHTTP -> proxy.handleDNSRequest
//  -> dnsforward.handleDNSRequest
func (s *Server) handleDOH(w http.ResponseWriter, r *http.Request) {
	if !s.conf.TLSAllowUnencryptedDOH && r.TLS == nil {
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
	s.conf.HTTPRegister("GET", "/control/dns_info", s.handleGetConfig)
	s.conf.HTTPRegister("POST", "/control/dns_config", s.handleSetConfig)
	s.conf.HTTPRegister("POST", "/control/test_upstream_dns", s.handleTestUpstreamDNS)

	s.conf.HTTPRegister("GET", "/control/access/list", s.handleAccessList)
	s.conf.HTTPRegister("POST", "/control/access/set", s.handleAccessSet)

	s.conf.HTTPRegister("", "/dns-query", s.handleDOH)
}
