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
	log.Info("DNS: %s %s: %s", r.Method, r.URL, text)
	http.Error(w, text, code)
}

type dnsConfigJSON struct {
	ProtectionEnabled bool   `json:"protection_enabled"`
	RateLimit         uint32 `json:"ratelimit"`
	BlockingMode      string `json:"blocking_mode"`
}

func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	resp := dnsConfigJSON{}
	s.RLock()
	resp.ProtectionEnabled = s.conf.ProtectionEnabled
	resp.BlockingMode = s.conf.BlockingMode
	resp.RateLimit = s.conf.Ratelimit
	s.RUnlock()

	js, err := json.Marshal(resp)
	if err != nil {
		httpError(r, w, http.StatusInternalServerError, "json.Marshal: %s", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(js)
}

func (s *Server) handleSetConfig(w http.ResponseWriter, r *http.Request) {
	req := dnsConfigJSON{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		httpError(r, w, http.StatusBadRequest, "json.Decode: %s", err)
		return
	}

	if !(req.BlockingMode == "nxdomain" || req.BlockingMode == "null_ip") {
		httpError(r, w, http.StatusBadRequest, "blocking_mode: value not supported")
		return
	}

	restart := false
	s.Lock()
	s.conf.ProtectionEnabled = req.ProtectionEnabled
	s.conf.BlockingMode = req.BlockingMode
	if s.conf.Ratelimit != req.RateLimit {
		restart = true
	}
	s.conf.Ratelimit = req.RateLimit
	s.Unlock()
	s.conf.ConfigModified()

	if restart {
		err = s.Restart()
		if err != nil {
			httpError(r, w, http.StatusInternalServerError, "%s", err)
			return
		}
	}
}

func (s *Server) handleProtectionEnable(w http.ResponseWriter, r *http.Request) {
	s.conf.ProtectionEnabled = true
	s.conf.ConfigModified()
}

func (s *Server) handleProtectionDisable(w http.ResponseWriter, r *http.Request) {
	s.conf.ProtectionEnabled = false
	s.conf.ConfigModified()
}

type upstreamJSON struct {
	Upstreams    []string `json:"upstream_dns"`  // Upstreams
	BootstrapDNS []string `json:"bootstrap_dns"` // Bootstrap DNS
	AllServers   bool     `json:"all_servers"`   // --all-servers param for dnsproxy
}

func (s *Server) handleSetUpstreamConfig(w http.ResponseWriter, r *http.Request) {
	req := upstreamJSON{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		httpError(r, w, http.StatusBadRequest, "Failed to parse new upstreams config json: %s", err)
		return
	}

	err = ValidateUpstreams(req.Upstreams)
	if err != nil {
		httpError(r, w, http.StatusBadRequest, "wrong upstreams specification: %s", err)
		return
	}

	newconf := FilteringConfig{}
	newconf.UpstreamDNS = req.Upstreams

	// bootstrap servers are plain DNS only
	for _, host := range req.BootstrapDNS {
		if err := checkPlainDNS(host); err != nil {
			httpError(r, w, http.StatusBadRequest, "%s can not be used as bootstrap dns cause: %s", host, err)
			return
		}
	}
	newconf.BootstrapDNS = req.BootstrapDNS

	newconf.AllServers = req.AllServers

	s.Lock()
	s.conf.UpstreamDNS = newconf.UpstreamDNS
	s.conf.BootstrapDNS = newconf.BootstrapDNS
	s.conf.AllServers = newconf.AllServers
	s.Unlock()
	s.conf.ConfigModified()

	err = s.Restart()
	if err != nil {
		httpError(r, w, http.StatusInternalServerError, "%s", err)
		return
	}
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

func (s *Server) registerHandlers() {
	s.conf.HTTPRegister("GET", "/control/dns_info", s.handleGetConfig)
	s.conf.HTTPRegister("POST", "/control/dns_config", s.handleSetConfig)
	s.conf.HTTPRegister("POST", "/control/enable_protection", s.handleProtectionEnable)
	s.conf.HTTPRegister("POST", "/control/disable_protection", s.handleProtectionDisable)
	s.conf.HTTPRegister("POST", "/control/set_upstreams_config", s.handleSetUpstreamConfig)
	s.conf.HTTPRegister("POST", "/control/test_upstream_dns", s.handleTestUpstreamDNS)

	s.conf.HTTPRegister("GET", "/control/access/list", s.handleAccessList)
	s.conf.HTTPRegister("POST", "/control/access/set", s.handleAccessSet)
}
