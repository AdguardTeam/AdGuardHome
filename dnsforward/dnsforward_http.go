package dnsforward

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/jsonutil"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/utils"
	"github.com/miekg/dns"
)

func httpError(r *http.Request, w http.ResponseWriter, code int, format string, args ...interface{}) {
	text := fmt.Sprintf(format, args...)
	log.Info("DNS: %s %s: %s", r.Method, r.URL, text)
	http.Error(w, text, code)
}

type dnsConfigJSON struct {
	Upstreams  []string `json:"upstream_dns"`
	Bootstraps []string `json:"bootstrap_dns"`

	ProtectionEnabled bool   `json:"protection_enabled"`
	RateLimit         uint32 `json:"ratelimit"`
	BlockingMode      string `json:"blocking_mode"`
	BlockingIPv4      string `json:"blocking_ipv4"`
	BlockingIPv6      string `json:"blocking_ipv6"`
	EDNSCSEnabled     bool   `json:"edns_cs_enabled"`
	DNSSECEnabled     bool   `json:"dnssec_enabled"`
	DisableIPv6       bool   `json:"disable_ipv6"`
	FastestAddr       bool   `json:"fastest_addr"`
	ParallelRequests  bool   `json:"parallel_requests"`
}

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	resp := dnsConfigJSON{}
	s.RLock()
	resp.Upstreams = stringArrayDup(s.conf.UpstreamDNS)
	resp.Bootstraps = stringArrayDup(s.conf.BootstrapDNS)

	resp.ProtectionEnabled = s.conf.ProtectionEnabled
	resp.BlockingMode = s.conf.BlockingMode
	resp.BlockingIPv4 = s.conf.BlockingIPv4
	resp.BlockingIPv6 = s.conf.BlockingIPv6
	resp.RateLimit = s.conf.Ratelimit
	resp.EDNSCSEnabled = s.conf.EnableEDNSClientSubnet
	resp.DNSSECEnabled = s.conf.EnableDNSSEC
	resp.DisableIPv6 = s.conf.AAAADisabled
	resp.FastestAddr = s.conf.FastestAddr
	resp.ParallelRequests = s.conf.AllServers
	s.RUnlock()

	js, err := json.Marshal(resp)
	if err != nil {
		httpError(r, w, http.StatusInternalServerError, "json.Marshal: %s", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(js)
}

func checkBlockingMode(req dnsConfigJSON) bool {
	bm := req.BlockingMode
	if !(bm == "default" || bm == "nxdomain" || bm == "null_ip" || bm == "custom_ip") {
		return false
	}

	if bm == "custom_ip" {
		ip := net.ParseIP(req.BlockingIPv4)
		if ip == nil || ip.To4() == nil {
			return false
		}

		ip = net.ParseIP(req.BlockingIPv6)
		if ip == nil {
			return false
		}
	}

	return true
}

// nolint(gocyclo) - we need to check each JSON field separately
func (s *Server) handleSetConfig(w http.ResponseWriter, r *http.Request) {
	req := dnsConfigJSON{}
	js, err := jsonutil.DecodeObject(&req, r.Body)
	if err != nil {
		httpError(r, w, http.StatusBadRequest, "json.Decode: %s", err)
		return
	}

	if js.Exists("upstream_dns") {
		if len(req.Upstreams) != 0 {
			err = ValidateUpstreams(req.Upstreams)
			if err != nil {
				httpError(r, w, http.StatusBadRequest, "wrong upstreams specification: %s", err)
				return
			}
		}
	}

	if js.Exists("bootstrap_dns") {
		for _, host := range req.Bootstraps {
			if err := checkPlainDNS(host); err != nil {
				httpError(r, w, http.StatusBadRequest, "%s can not be used as bootstrap dns cause: %s", host, err)
				return
			}
		}
	}

	if js.Exists("blocking_mode") && !checkBlockingMode(req) {
		httpError(r, w, http.StatusBadRequest, "blocking_mode: incorrect value")
		return
	}

	restart := false
	s.Lock()

	if js.Exists("upstream_dns") {
		s.conf.UpstreamDNS = req.Upstreams
		restart = true
	}

	if js.Exists("bootstrap_dns") {
		s.conf.BootstrapDNS = req.Bootstraps
		restart = true
	}

	if js.Exists("protection_enabled") {
		s.conf.ProtectionEnabled = req.ProtectionEnabled
	}

	if js.Exists("blocking_mode") {
		s.conf.BlockingMode = req.BlockingMode
		if req.BlockingMode == "custom_ip" {
			if js.Exists("blocking_ipv4") {
				s.conf.BlockingIPv4 = req.BlockingIPv4
				s.conf.BlockingIPAddrv4 = net.ParseIP(req.BlockingIPv4)
			}
			if js.Exists("blocking_ipv6") {
				s.conf.BlockingIPv6 = req.BlockingIPv6
				s.conf.BlockingIPAddrv6 = net.ParseIP(req.BlockingIPv6)
			}
		}
	}

	if js.Exists("ratelimit") {
		if s.conf.Ratelimit != req.RateLimit {
			restart = true
		}
		s.conf.Ratelimit = req.RateLimit
	}

	if js.Exists("edns_cs_enabled") {
		s.conf.EnableEDNSClientSubnet = req.EDNSCSEnabled
		restart = true
	}

	if js.Exists("dnssec_enabled") {
		s.conf.EnableDNSSEC = req.DNSSECEnabled
	}

	if js.Exists("disable_ipv6") {
		s.conf.AAAADisabled = req.DisableIPv6
	}

	if js.Exists("fastest_addr") {
		s.conf.FastestAddr = req.FastestAddr
	}

	if js.Exists("parallel_requests") {
		s.conf.AllServers = req.ParallelRequests
	}

	s.Unlock()
	s.conf.ConfigModified()

	if restart {
		err = s.Reconfigure(nil)
		if err != nil {
			httpError(r, w, http.StatusInternalServerError, "%s", err)
			return
		}
	}
}

type upstreamJSON struct {
	Upstreams    []string `json:"upstream_dns"`  // Upstreams
	BootstrapDNS []string `json:"bootstrap_dns"` // Bootstrap DNS
}

// ValidateUpstreams validates each upstream and returns an error if any upstream is invalid or if there are no default upstreams specified
func ValidateUpstreams(upstreams []string) error {
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

var protocols = []string{"tls://", "https://", "tcp://", "sdns://"}

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
			return "", defaultUpstream, fmt.Errorf("wrong DNS upstream per domain specification: %s", upstream)
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
		return fmt.Errorf("%s is not a valid port: %s", port, err)
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

	if len(req.Upstreams) == 0 {
		httpError(r, w, http.StatusBadRequest, "No servers specified")
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
	// separate upstream from domains list
	input, defaultUpstream, err := separateUpstream(input)
	if err != nil {
		return fmt.Errorf("wrong upstream format: %s", err)
	}

	// No need to check this entrance
	if input == "#" && !defaultUpstream {
		return nil
	}

	if _, err := validateUpstream(input); err != nil {
		return fmt.Errorf("wrong upstream format: %s", err)
	}

	if len(bootstrap) == 0 {
		bootstrap = defaultBootstrap
	}

	log.Debug("Checking if DNS %s works...", input)
	u, err := upstream.AddressToUpstream(input, upstream.Options{Bootstrap: bootstrap, Timeout: DefaultTimeout})
	if err != nil {
		return fmt.Errorf("failed to choose upstream for %s: %s", input, err)
	}

	req := dns.Msg{}
	req.Id = dns.Id()
	req.RecursionDesired = true
	req.Question = []dns.Question{
		{Name: "google-public-dns-a.google.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET},
	}
	reply, err := u.Exchange(&req)
	if err != nil {
		return fmt.Errorf("couldn't communicate with DNS server %s: %s", input, err)
	}
	if len(reply.Answer) != 1 {
		return fmt.Errorf("DNS server %s returned wrong answer", input)
	}
	if t, ok := reply.Answer[0].(*dns.A); ok {
		if !net.IPv4(8, 8, 8, 8).Equal(t.A) {
			return fmt.Errorf("DNS server %s returned wrong answer: %v", input, t.A)
		}
	}

	log.Debug("DNS %s works OK", input)
	return nil
}

func (s *Server) handleDOH(w http.ResponseWriter, r *http.Request) {
	if !s.conf.TLSAllowUnencryptedDOH && r.TLS == nil {
		httpError(r, w, http.StatusNotFound, "Not Found")
		return
	}

	if !s.IsRunning() {
		httpError(r, w, http.StatusInternalServerError, "DNS server is not running")
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
