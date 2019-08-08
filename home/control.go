package home

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/dnsforward"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/utils"
	"github.com/NYTimes/gziphandler"
	"github.com/miekg/dns"
)

const updatePeriod = time.Hour * 24

var protocols = []string{"tls://", "https://", "tcp://", "sdns://"}

// ----------------
// helper functions
// ----------------

func returnOK(w http.ResponseWriter) {
	_, err := fmt.Fprintf(w, "OK\n")
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Couldn't write body: %s", err)
	}
}

func httpError(w http.ResponseWriter, code int, format string, args ...interface{}) {
	text := fmt.Sprintf(format, args...)
	log.Info(text)
	http.Error(w, text, code)
}

// ---------------
// dns run control
// ---------------
func writeAllConfigsAndReloadDNS() error {
	err := writeAllConfigs()
	if err != nil {
		log.Error("Couldn't write all configs: %s", err)
		return err
	}
	return reconfigureDNSServer()
}

func httpUpdateConfigReloadDNSReturnOK(w http.ResponseWriter, r *http.Request) {
	err := writeAllConfigsAndReloadDNS()
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Couldn't write config file: %s", err)
		return
	}
	returnOK(w)
}

func addDNSAddress(dnsAddresses *[]string, addr string) {
	if config.DNS.Port != 53 {
		addr = fmt.Sprintf("%s:%d", addr, config.DNS.Port)
	}
	*dnsAddresses = append(*dnsAddresses, addr)
}

// Get the list of DNS addresses the server is listening on
func getDNSAddresses() []string {
	dnsAddresses := []string{}

	if config.DNS.BindHost == "0.0.0.0" {

		ifaces, e := getValidNetInterfacesForWeb()
		if e != nil {
			log.Error("Couldn't get network interfaces: %v", e)
			return []string{}
		}

		for _, iface := range ifaces {
			for _, addr := range iface.Addresses {
				addDNSAddress(&dnsAddresses, addr)
			}
		}

	} else {
		addDNSAddress(&dnsAddresses, config.DNS.BindHost)
	}

	if config.TLS.Enabled && len(config.TLS.ServerName) != 0 {

		if config.TLS.PortHTTPS != 0 {
			addr := config.TLS.ServerName
			if config.TLS.PortHTTPS != 443 {
				addr = fmt.Sprintf("%s:%d", addr, config.TLS.PortHTTPS)
			}
			addr = fmt.Sprintf("https://%s/dns-query", addr)
			dnsAddresses = append(dnsAddresses, addr)
		}

		if config.TLS.PortDNSOverTLS != 0 {
			addr := fmt.Sprintf("tls://%s:%d", config.TLS.ServerName, config.TLS.PortDNSOverTLS)
			dnsAddresses = append(dnsAddresses, addr)
		}
	}

	return dnsAddresses
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"dns_addresses":      getDNSAddresses(),
		"http_port":          config.BindPort,
		"dns_port":           config.DNS.Port,
		"protection_enabled": config.DNS.ProtectionEnabled,
		"querylog_enabled":   config.DNS.QueryLogEnabled,
		"running":            isRunning(),
		"bootstrap_dns":      config.DNS.BootstrapDNS,
		"upstream_dns":       config.DNS.UpstreamDNS,
		"all_servers":        config.DNS.AllServers,
		"version":            versionString,
		"language":           config.Language,
	}

	jsonVal, err := json.Marshal(data)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Unable to marshal status json: %s", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonVal)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Unable to write response json: %s", err)
		return
	}
}

func handleProtectionEnable(w http.ResponseWriter, r *http.Request) {
	config.DNS.ProtectionEnabled = true
	httpUpdateConfigReloadDNSReturnOK(w, r)
}

func handleProtectionDisable(w http.ResponseWriter, r *http.Request) {
	config.DNS.ProtectionEnabled = false
	httpUpdateConfigReloadDNSReturnOK(w, r)
}

// -----
// stats
// -----
func handleQueryLogEnable(w http.ResponseWriter, r *http.Request) {
	config.DNS.QueryLogEnabled = true
	httpUpdateConfigReloadDNSReturnOK(w, r)
}

func handleQueryLogDisable(w http.ResponseWriter, r *http.Request) {
	config.DNS.QueryLogEnabled = false
	httpUpdateConfigReloadDNSReturnOK(w, r)
}

func handleQueryLog(w http.ResponseWriter, r *http.Request) {
	data := config.dnsServer.GetQueryLog()

	jsonVal, err := json.Marshal(data)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Couldn't marshal data into json: %s", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonVal)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Unable to write response json: %s", err)
	}
}

// -----------------------
// upstreams configuration
// -----------------------

// TODO this struct will become unnecessary after config file rework
type upstreamConfig struct {
	Upstreams    []string `json:"upstream_dns"`  // Upstreams
	BootstrapDNS []string `json:"bootstrap_dns"` // Bootstrap DNS
	AllServers   bool     `json:"all_servers"`   // --all-servers param for dnsproxy
}

func handleSetUpstreamConfig(w http.ResponseWriter, r *http.Request) {
	newconfig := upstreamConfig{}
	err := json.NewDecoder(r.Body).Decode(&newconfig)
	if err != nil {
		httpError(w, http.StatusBadRequest, "Failed to parse new upstreams config json: %s", err)
		return
	}

	err = validateUpstreams(newconfig.Upstreams)
	if err != nil {
		httpError(w, http.StatusBadRequest, "wrong upstreams specification: %s", err)
		return
	}

	config.DNS.UpstreamDNS = defaultDNS
	if len(newconfig.Upstreams) > 0 {
		config.DNS.UpstreamDNS = newconfig.Upstreams
	}

	// bootstrap servers are plain DNS only.
	for _, host := range newconfig.BootstrapDNS {
		if err := checkPlainDNS(host); err != nil {
			httpError(w, http.StatusBadRequest, "%s can not be used as bootstrap dns cause: %s", host, err)
			return
		}
	}

	config.DNS.BootstrapDNS = defaultBootstrap
	if len(newconfig.BootstrapDNS) > 0 {
		config.DNS.BootstrapDNS = newconfig.BootstrapDNS
	}

	config.DNS.AllServers = newconfig.AllServers
	httpUpdateConfigReloadDNSReturnOK(w, r)
}

// validateUpstreams validates each upstream and returns an error if any upstream is invalid or if there are no default upstreams specified
func validateUpstreams(upstreams []string) error {
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

func handleTestUpstreamDNS(w http.ResponseWriter, r *http.Request) {
	upstreamConfig := upstreamConfig{}
	err := json.NewDecoder(r.Body).Decode(&upstreamConfig)
	if err != nil {
		httpError(w, http.StatusBadRequest, "Failed to read request body: %s", err)
		return
	}

	if len(upstreamConfig.Upstreams) == 0 {
		httpError(w, http.StatusBadRequest, "No servers specified")
		return
	}

	result := map[string]string{}

	for _, host := range upstreamConfig.Upstreams {
		err = checkDNS(host, upstreamConfig.BootstrapDNS)
		if err != nil {
			log.Info("%v", err)
			result[host] = err.Error()
		} else {
			result[host] = "OK"
		}
	}

	jsonVal, err := json.Marshal(result)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Unable to marshal status json: %s", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonVal)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Couldn't write body: %s", err)
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
	u, err := upstream.AddressToUpstream(input, upstream.Options{Bootstrap: bootstrap, Timeout: dnsforward.DefaultTimeout})
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

// ------------
// safebrowsing
// ------------

func handleSafeBrowsingEnable(w http.ResponseWriter, r *http.Request) {
	config.DNS.SafeBrowsingEnabled = true
	httpUpdateConfigReloadDNSReturnOK(w, r)
}

func handleSafeBrowsingDisable(w http.ResponseWriter, r *http.Request) {
	config.DNS.SafeBrowsingEnabled = false
	httpUpdateConfigReloadDNSReturnOK(w, r)
}

func handleSafeBrowsingStatus(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"enabled": config.DNS.SafeBrowsingEnabled,
	}
	jsonVal, err := json.Marshal(data)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Unable to marshal status json: %s", err)
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonVal)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Unable to write response json: %s", err)
		return
	}
}

// --------
// parental
// --------
func handleParentalEnable(w http.ResponseWriter, r *http.Request) {
	parameters, err := parseParametersFromBody(r.Body)
	if err != nil {
		httpError(w, http.StatusBadRequest, "failed to parse parameters from body: %s", err)
		return
	}

	sensitivity, ok := parameters["sensitivity"]
	if !ok {
		http.Error(w, "Sensitivity parameter was not specified", 400)
		return
	}

	switch sensitivity {
	case "3":
		break
	case "EARLY_CHILDHOOD":
		sensitivity = "3"
	case "10":
		break
	case "YOUNG":
		sensitivity = "10"
	case "13":
		break
	case "TEEN":
		sensitivity = "13"
	case "17":
		break
	case "MATURE":
		sensitivity = "17"
	default:
		http.Error(w, "Sensitivity must be set to valid value", 400)
		return
	}
	i, err := strconv.Atoi(sensitivity)
	if err != nil {
		http.Error(w, "Sensitivity must be set to valid value", 400)
		return
	}
	config.DNS.ParentalSensitivity = i
	config.DNS.ParentalEnabled = true
	httpUpdateConfigReloadDNSReturnOK(w, r)
}

func handleParentalDisable(w http.ResponseWriter, r *http.Request) {
	config.DNS.ParentalEnabled = false
	httpUpdateConfigReloadDNSReturnOK(w, r)
}

func handleParentalStatus(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"enabled": config.DNS.ParentalEnabled,
	}
	if config.DNS.ParentalEnabled {
		data["sensitivity"] = config.DNS.ParentalSensitivity
	}
	jsonVal, err := json.Marshal(data)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Unable to marshal status json: %s", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonVal)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Unable to write response json: %s", err)
		return
	}
}

// ------------
// safebrowsing
// ------------

func handleSafeSearchEnable(w http.ResponseWriter, r *http.Request) {
	config.DNS.SafeSearchEnabled = true
	httpUpdateConfigReloadDNSReturnOK(w, r)
}

func handleSafeSearchDisable(w http.ResponseWriter, r *http.Request) {
	config.DNS.SafeSearchEnabled = false
	httpUpdateConfigReloadDNSReturnOK(w, r)
}

func handleSafeSearchStatus(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"enabled": config.DNS.SafeSearchEnabled,
	}
	jsonVal, err := json.Marshal(data)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Unable to marshal status json: %s", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonVal)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Unable to write response json: %s", err)
		return
	}
}

// --------------
// DNS-over-HTTPS
// --------------
func handleDOH(w http.ResponseWriter, r *http.Request) {
	if r.TLS == nil {
		httpError(w, http.StatusNotFound, "Not Found")
		return
	}

	if !isRunning() {
		httpError(w, http.StatusInternalServerError, "DNS server is not running")
		return
	}

	config.dnsServer.ServeHTTP(w, r)
}

// ------------------------
// registration of handlers
// ------------------------
func registerControlHandlers() {
	httpRegister(http.MethodGet, "/control/status", handleStatus)
	httpRegister(http.MethodPost, "/control/enable_protection", handleProtectionEnable)
	httpRegister(http.MethodPost, "/control/disable_protection", handleProtectionDisable)
	httpRegister(http.MethodGet, "/control/querylog", handleQueryLog)
	httpRegister(http.MethodPost, "/control/querylog_enable", handleQueryLogEnable)
	httpRegister(http.MethodPost, "/control/querylog_disable", handleQueryLogDisable)
	httpRegister(http.MethodPost, "/control/set_upstreams_config", handleSetUpstreamConfig)
	httpRegister(http.MethodPost, "/control/test_upstream_dns", handleTestUpstreamDNS)
	httpRegister(http.MethodPost, "/control/i18n/change_language", handleI18nChangeLanguage)
	httpRegister(http.MethodGet, "/control/i18n/current_language", handleI18nCurrentLanguage)
	http.HandleFunc("/control/version.json", postInstall(optionalAuth(handleGetVersionJSON)))
	httpRegister(http.MethodPost, "/control/update", handleUpdate)
	httpRegister(http.MethodPost, "/control/filtering/enable", handleFilteringEnable)
	httpRegister(http.MethodPost, "/control/filtering/disable", handleFilteringDisable)
	httpRegister(http.MethodPost, "/control/filtering/add_url", handleFilteringAddURL)
	httpRegister(http.MethodPost, "/control/filtering/remove_url", handleFilteringRemoveURL)
	httpRegister(http.MethodPost, "/control/filtering/enable_url", handleFilteringEnableURL)
	httpRegister(http.MethodPost, "/control/filtering/disable_url", handleFilteringDisableURL)
	httpRegister(http.MethodPost, "/control/filtering/refresh", handleFilteringRefresh)
	httpRegister(http.MethodGet, "/control/filtering/status", handleFilteringStatus)
	httpRegister(http.MethodPost, "/control/filtering/set_rules", handleFilteringSetRules)
	httpRegister(http.MethodPost, "/control/safebrowsing/enable", handleSafeBrowsingEnable)
	httpRegister(http.MethodPost, "/control/safebrowsing/disable", handleSafeBrowsingDisable)
	httpRegister(http.MethodGet, "/control/safebrowsing/status", handleSafeBrowsingStatus)
	httpRegister(http.MethodPost, "/control/parental/enable", handleParentalEnable)
	httpRegister(http.MethodPost, "/control/parental/disable", handleParentalDisable)
	httpRegister(http.MethodGet, "/control/parental/status", handleParentalStatus)
	httpRegister(http.MethodPost, "/control/safesearch/enable", handleSafeSearchEnable)
	httpRegister(http.MethodPost, "/control/safesearch/disable", handleSafeSearchDisable)
	httpRegister(http.MethodGet, "/control/safesearch/status", handleSafeSearchStatus)
	httpRegister(http.MethodGet, "/control/dhcp/status", handleDHCPStatus)
	httpRegister(http.MethodGet, "/control/dhcp/interfaces", handleDHCPInterfaces)
	httpRegister(http.MethodPost, "/control/dhcp/set_config", handleDHCPSetConfig)
	httpRegister(http.MethodPost, "/control/dhcp/find_active_dhcp", handleDHCPFindActiveServer)
	httpRegister(http.MethodPost, "/control/dhcp/add_static_lease", handleDHCPAddStaticLease)
	httpRegister(http.MethodPost, "/control/dhcp/remove_static_lease", handleDHCPRemoveStaticLease)

	httpRegister(http.MethodGet, "/control/access/list", handleAccessList)
	httpRegister(http.MethodPost, "/control/access/set", handleAccessSet)

	RegisterTLSHandlers()
	RegisterClientsHandlers()
	registerRewritesHandlers()
	RegisterBlockedServicesHandlers()
	RegisterStatsHandlers()

	http.HandleFunc("/dns-query", postInstall(handleDOH))
}

type httpHandlerType func(http.ResponseWriter, *http.Request)

func httpRegister(method string, url string, handler httpHandlerType) {
	http.Handle(url, postInstallHandler(optionalAuthHandler(gziphandler.GzipHandler(ensureHandler(method, handler)))))
}
