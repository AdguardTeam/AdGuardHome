package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/dnsforward"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/utils"
	"github.com/NYTimes/gziphandler"
	"github.com/miekg/dns"
	govalidator "gopkg.in/asaskevich/govalidator.v4"
)

const updatePeriod = time.Minute * 30

// cached version.json to avoid hammering github.io for each page reload
var versionCheckJSON []byte
var versionCheckLastTime time.Time

var protocols = []string{"tls://", "https://", "tcp://", "sdns://"}

const versionCheckURL = "https://adguardteam.github.io/AdGuardHome/version.json"
const versionCheckPeriod = time.Hour * 8

var transport = &http.Transport{
	DialContext: customDialContext,
}

var client = &http.Client{
	Timeout:   time.Minute * 5,
	Transport: transport,
}

var controlLock sync.Mutex

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

func handleStatus(w http.ResponseWriter, r *http.Request) {
	log.Tracef("%s %v", r.Method, r.URL)

	dnsAddresses := []string{}
	if config.DNS.BindHost == "0.0.0.0" {
		ifaces, e := getValidNetInterfacesForWeb()
		if e != nil {
			log.Error("Couldn't get network interfaces: %v", e)
		}
		for _, iface := range ifaces {
			for _, addr := range iface.Addresses {
				dnsAddresses = append(dnsAddresses, addr)
			}
		}
	}
	if len(dnsAddresses) == 0 {
		dnsAddresses = append(dnsAddresses, config.DNS.BindHost)
	}

	data := map[string]interface{}{
		"dns_addresses":      dnsAddresses,
		"http_port":          config.BindPort,
		"dns_port":           config.DNS.Port,
		"protection_enabled": config.DNS.ProtectionEnabled,
		"querylog_enabled":   config.DNS.QueryLogEnabled,
		"running":            isRunning(),
		"bootstrap_dns":      config.DNS.BootstrapDNS,
		"upstream_dns":       config.DNS.UpstreamDNS,
		"all_servers":        config.DNS.AllServers,
		"version":            VersionString,
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
	log.Tracef("%s %v", r.Method, r.URL)
	config.DNS.ProtectionEnabled = true
	httpUpdateConfigReloadDNSReturnOK(w, r)
}

func handleProtectionDisable(w http.ResponseWriter, r *http.Request) {
	log.Tracef("%s %v", r.Method, r.URL)
	config.DNS.ProtectionEnabled = false
	httpUpdateConfigReloadDNSReturnOK(w, r)
}

// -----
// stats
// -----
func handleQueryLogEnable(w http.ResponseWriter, r *http.Request) {
	log.Tracef("%s %v", r.Method, r.URL)
	config.DNS.QueryLogEnabled = true
	httpUpdateConfigReloadDNSReturnOK(w, r)
}

func handleQueryLogDisable(w http.ResponseWriter, r *http.Request) {
	log.Tracef("%s %v", r.Method, r.URL)
	config.DNS.QueryLogEnabled = false
	httpUpdateConfigReloadDNSReturnOK(w, r)
}

func handleQueryLog(w http.ResponseWriter, r *http.Request) {
	log.Tracef("%s %v", r.Method, r.URL)
	data := dnsServer.GetQueryLog()

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

func handleStatsTop(w http.ResponseWriter, r *http.Request) {
	log.Tracef("%s %v", r.Method, r.URL)
	s := dnsServer.GetStatsTop()

	// use manual json marshalling because we want maps to be sorted by value
	statsJSON := bytes.Buffer{}
	statsJSON.WriteString("{\n")

	gen := func(json *bytes.Buffer, name string, top map[string]int, addComma bool) {
		json.WriteString("  ")
		json.WriteString(fmt.Sprintf("%q", name))
		json.WriteString(": {\n")
		sorted := sortByValue(top)
		// no more than 50 entries
		if len(sorted) > 50 {
			sorted = sorted[:50]
		}
		for i, key := range sorted {
			json.WriteString("    ")
			json.WriteString(fmt.Sprintf("%q", key))
			json.WriteString(": ")
			json.WriteString(strconv.Itoa(top[key]))
			if i+1 != len(sorted) {
				json.WriteByte(',')
			}
			json.WriteByte('\n')
		}
		json.WriteString("  }")
		if addComma {
			json.WriteByte(',')
		}
		json.WriteByte('\n')
	}
	gen(&statsJSON, "top_queried_domains", s.Domains, true)
	gen(&statsJSON, "top_blocked_domains", s.Blocked, true)
	gen(&statsJSON, "top_clients", s.Clients, true)
	statsJSON.WriteString("  \"stats_period\": \"24 hours\"\n")
	statsJSON.WriteString("}\n")

	w.Header().Set("Content-Type", "application/json")
	_, err := w.Write(statsJSON.Bytes())
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Couldn't write body: %s", err)
	}
}

// handleStatsReset resets the stats caches
func handleStatsReset(w http.ResponseWriter, r *http.Request) {
	log.Tracef("%s %v", r.Method, r.URL)
	dnsServer.PurgeStats()
	_, err := fmt.Fprintf(w, "OK\n")
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Couldn't write body: %s", err)
	}
}

// handleStats returns aggregated stats data for the 24 hours
func handleStats(w http.ResponseWriter, r *http.Request) {
	log.Tracef("%s %v", r.Method, r.URL)
	summed := dnsServer.GetAggregatedStats()

	statsJSON, err := json.Marshal(summed)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Unable to marshal status json: %s", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(statsJSON)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Unable to write response json: %s", err)
		return
	}
}

// HandleStatsHistory returns historical stats data for the 24 hours
func handleStatsHistory(w http.ResponseWriter, r *http.Request) {
	log.Tracef("%s %v", r.Method, r.URL)
	// handle time unit and prepare our time window size
	timeUnitString := r.URL.Query().Get("time_unit")
	var timeUnit time.Duration
	switch timeUnitString {
	case "seconds":
		timeUnit = time.Second
	case "minutes":
		timeUnit = time.Minute
	case "hours":
		timeUnit = time.Hour
	case "days":
		timeUnit = time.Hour * 24
	default:
		http.Error(w, "Must specify valid time_unit parameter", http.StatusBadRequest)
		return
	}

	// parse start and end time
	startTime, err := time.Parse(time.RFC3339, r.URL.Query().Get("start_time"))
	if err != nil {
		httpError(w, http.StatusBadRequest, "Must specify valid start_time parameter: %s", err)
		return
	}
	endTime, err := time.Parse(time.RFC3339, r.URL.Query().Get("end_time"))
	if err != nil {
		httpError(w, http.StatusBadRequest, "Must specify valid end_time parameter: %s", err)
		return
	}

	data, err := dnsServer.GetStatsHistory(timeUnit, startTime, endTime)
	if err != nil {
		httpError(w, http.StatusBadRequest, "Cannot get stats history: %s", err)
		return
	}

	statsJSON, err := json.Marshal(data)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Unable to marshal status json: %s", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(statsJSON)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Unable to write response json: %s", err)
		return
	}
}

// sortByValue is a helper function for querylog API
func sortByValue(m map[string]int) []string {
	type kv struct {
		k string
		v int
	}
	var ss []kv
	for k, v := range m {
		ss = append(ss, kv{k, v})
	}
	sort.Slice(ss, func(l, r int) bool {
		return ss[l].v > ss[r].v
	})

	sorted := []string{}
	for _, v := range ss {
		sorted = append(sorted, v.k)
	}
	return sorted
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
	log.Tracef("%s %v", r.Method, r.URL)
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
	log.Tracef("%s %v", r.Method, r.URL)
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

// ---------
// filtering
// ---------

func handleFilteringEnable(w http.ResponseWriter, r *http.Request) {
	log.Tracef("%s %v", r.Method, r.URL)
	config.DNS.FilteringEnabled = true
	httpUpdateConfigReloadDNSReturnOK(w, r)
}

func handleFilteringDisable(w http.ResponseWriter, r *http.Request) {
	log.Tracef("%s %v", r.Method, r.URL)
	config.DNS.FilteringEnabled = false
	httpUpdateConfigReloadDNSReturnOK(w, r)
}

func handleFilteringStatus(w http.ResponseWriter, r *http.Request) {
	log.Tracef("%s %v", r.Method, r.URL)
	data := map[string]interface{}{
		"enabled": config.DNS.FilteringEnabled,
	}

	config.RLock()
	data["filters"] = config.Filters
	data["user_rules"] = config.UserRules
	jsonVal, err := json.Marshal(data)
	config.RUnlock()

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

func handleFilteringAddURL(w http.ResponseWriter, r *http.Request) {
	log.Tracef("%s %v", r.Method, r.URL)
	f := filter{}
	err := json.NewDecoder(r.Body).Decode(&f)
	if err != nil {
		httpError(w, http.StatusBadRequest, "Failed to parse request body json: %s", err)
		return
	}

	if len(f.URL) == 0 {
		http.Error(w, "URL parameter was not specified", http.StatusBadRequest)
		return
	}

	if valid := govalidator.IsRequestURL(f.URL); !valid {
		http.Error(w, "URL parameter is not valid request URL", http.StatusBadRequest)
		return
	}

	// Check for duplicates
	if filterExists(f.URL) {
		httpError(w, http.StatusBadRequest, "Filter URL already added -- %s", f.URL)
		return
	}

	// Set necessary properties
	f.ID = assignUniqueFilterID()
	f.Enabled = true

	// Download the filter contents
	ok, err := f.update()
	if err != nil {
		httpError(w, http.StatusBadRequest, "Couldn't fetch filter from url %s: %s", f.URL, err)
		return
	}
	if f.RulesCount == 0 {
		httpError(w, http.StatusBadRequest, "Filter at the url %s has no rules (maybe it points to blank page?)", f.URL)
		return
	}
	if !ok {
		httpError(w, http.StatusBadRequest, "Filter at the url %s is invalid (maybe it points to blank page?)", f.URL)
		return
	}

	// Save the filter contents
	err = f.save()
	if err != nil {
		httpError(w, http.StatusBadRequest, "Failed to save filter %d due to %s", f.ID, err)
		return
	}

	// URL is deemed valid, append it to filters, update config, write new filter file and tell dns to reload it
	// TODO: since we directly feed filters in-memory, revisit if writing configs is always necessary
	if !filterAdd(f) {
		httpError(w, http.StatusBadRequest, "Filter URL already added -- %s", f.URL)
		return
	}

	err = writeAllConfigs()
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Couldn't write config file: %s", err)
		return
	}

	err = reconfigureDNSServer()
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Couldn't reconfigure the DNS server: %s", err)
		return
	}

	_, err = fmt.Fprintf(w, "OK %d rules\n", f.RulesCount)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Couldn't write body: %s", err)
	}
}

func handleFilteringRemoveURL(w http.ResponseWriter, r *http.Request) {
	log.Tracef("%s %v", r.Method, r.URL)
	parameters, err := parseParametersFromBody(r.Body)
	if err != nil {
		httpError(w, http.StatusBadRequest, "failed to parse parameters from body: %s", err)
		return
	}

	url, ok := parameters["url"]
	if !ok {
		http.Error(w, "URL parameter was not specified", http.StatusBadRequest)
		return
	}

	if valid := govalidator.IsRequestURL(url); !valid {
		http.Error(w, "URL parameter is not valid request URL", http.StatusBadRequest)
		return
	}

	// go through each element and delete if url matches
	config.Lock()
	newFilters := config.Filters[:0]
	for _, filter := range config.Filters {
		if filter.URL != url {
			newFilters = append(newFilters, filter)
		} else {
			// Remove the filter file
			err := os.Remove(filter.Path())
			if err != nil && !os.IsNotExist(err) {
				httpError(w, http.StatusInternalServerError, "Couldn't remove the filter file: %s", err)
				return
			}
		}
	}
	// Update the configuration after removing filter files
	config.Filters = newFilters
	config.Unlock()
	httpUpdateConfigReloadDNSReturnOK(w, r)
}

func handleFilteringEnableURL(w http.ResponseWriter, r *http.Request) {
	log.Tracef("%s %v", r.Method, r.URL)
	parameters, err := parseParametersFromBody(r.Body)
	if err != nil {
		httpError(w, http.StatusBadRequest, "failed to parse parameters from body: %s", err)
		return
	}

	url, ok := parameters["url"]
	if !ok {
		http.Error(w, "URL parameter was not specified", http.StatusBadRequest)
		return
	}

	if valid := govalidator.IsRequestURL(url); !valid {
		http.Error(w, "URL parameter is not valid request URL", http.StatusBadRequest)
		return
	}

	found := filterEnable(url, true)
	if !found {
		http.Error(w, "URL parameter was not previously added", http.StatusBadRequest)
		return
	}

	httpUpdateConfigReloadDNSReturnOK(w, r)
}

func handleFilteringDisableURL(w http.ResponseWriter, r *http.Request) {
	log.Tracef("%s %v", r.Method, r.URL)
	parameters, err := parseParametersFromBody(r.Body)
	if err != nil {
		httpError(w, http.StatusBadRequest, "failed to parse parameters from body: %s", err)
		return
	}

	url, ok := parameters["url"]
	if !ok {
		http.Error(w, "URL parameter was not specified", http.StatusBadRequest)
		return
	}

	if valid := govalidator.IsRequestURL(url); !valid {
		http.Error(w, "URL parameter is not valid request URL", http.StatusBadRequest)
		return
	}

	found := filterEnable(url, false)
	if !found {
		http.Error(w, "URL parameter was not previously added", http.StatusBadRequest)
		return
	}

	httpUpdateConfigReloadDNSReturnOK(w, r)
}

func handleFilteringSetRules(w http.ResponseWriter, r *http.Request) {
	log.Tracef("%s %v", r.Method, r.URL)
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		httpError(w, http.StatusBadRequest, "Failed to read request body: %s", err)
		return
	}

	config.UserRules = strings.Split(string(body), "\n")
	httpUpdateConfigReloadDNSReturnOK(w, r)
}

func handleFilteringRefresh(w http.ResponseWriter, r *http.Request) {
	log.Tracef("%s %v", r.Method, r.URL)
	updated := refreshFiltersIfNecessary(true)
	fmt.Fprintf(w, "OK %d filters updated\n", updated)
}

// ------------
// safebrowsing
// ------------

func handleSafeBrowsingEnable(w http.ResponseWriter, r *http.Request) {
	log.Tracef("%s %v", r.Method, r.URL)
	config.DNS.SafeBrowsingEnabled = true
	httpUpdateConfigReloadDNSReturnOK(w, r)
}

func handleSafeBrowsingDisable(w http.ResponseWriter, r *http.Request) {
	log.Tracef("%s %v", r.Method, r.URL)
	config.DNS.SafeBrowsingEnabled = false
	httpUpdateConfigReloadDNSReturnOK(w, r)
}

func handleSafeBrowsingStatus(w http.ResponseWriter, r *http.Request) {
	log.Tracef("%s %v", r.Method, r.URL)
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
	log.Tracef("%s %v", r.Method, r.URL)
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
	log.Tracef("%s %v", r.Method, r.URL)
	config.DNS.ParentalEnabled = false
	httpUpdateConfigReloadDNSReturnOK(w, r)
}

func handleParentalStatus(w http.ResponseWriter, r *http.Request) {
	log.Tracef("%s %v", r.Method, r.URL)
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
	log.Tracef("%s %v", r.Method, r.URL)
	config.DNS.SafeSearchEnabled = true
	httpUpdateConfigReloadDNSReturnOK(w, r)
}

func handleSafeSearchDisable(w http.ResponseWriter, r *http.Request) {
	log.Tracef("%s %v", r.Method, r.URL)
	config.DNS.SafeSearchEnabled = false
	httpUpdateConfigReloadDNSReturnOK(w, r)
}

func handleSafeSearchStatus(w http.ResponseWriter, r *http.Request) {
	log.Tracef("%s %v", r.Method, r.URL)
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
	log.Tracef("%s %v", r.Method, r.URL)
	if r.TLS == nil {
		httpError(w, http.StatusNotFound, "Not Found")
		return
	}

	if !isRunning() {
		httpError(w, http.StatusInternalServerError, "DNS server is not running")
		return
	}

	dnsServer.ServeHTTP(w, r)
}

// ------------------------
// registration of handlers
// ------------------------
func registerControlHandlers() {
	http.HandleFunc("/control/status", postInstall(optionalAuth(ensureGET(handleStatus))))
	http.HandleFunc("/control/enable_protection", postInstall(optionalAuth(ensurePOST(handleProtectionEnable))))
	http.HandleFunc("/control/disable_protection", postInstall(optionalAuth(ensurePOST(handleProtectionDisable))))
	http.Handle("/control/querylog", postInstallHandler(optionalAuthHandler(gziphandler.GzipHandler(ensureGETHandler(handleQueryLog)))))
	http.HandleFunc("/control/querylog_enable", postInstall(optionalAuth(ensurePOST(handleQueryLogEnable))))
	http.HandleFunc("/control/querylog_disable", postInstall(optionalAuth(ensurePOST(handleQueryLogDisable))))
	http.HandleFunc("/control/set_upstreams_config", postInstall(optionalAuth(ensurePOST(handleSetUpstreamConfig))))
	http.HandleFunc("/control/test_upstream_dns", postInstall(optionalAuth(ensurePOST(handleTestUpstreamDNS))))
	http.HandleFunc("/control/i18n/change_language", postInstall(optionalAuth(ensurePOST(handleI18nChangeLanguage))))
	http.HandleFunc("/control/i18n/current_language", postInstall(optionalAuth(ensureGET(handleI18nCurrentLanguage))))
	http.HandleFunc("/control/stats_top", postInstall(optionalAuth(ensureGET(handleStatsTop))))
	http.HandleFunc("/control/stats", postInstall(optionalAuth(ensureGET(handleStats))))
	http.HandleFunc("/control/stats_history", postInstall(optionalAuth(ensureGET(handleStatsHistory))))
	http.HandleFunc("/control/stats_reset", postInstall(optionalAuth(ensurePOST(handleStatsReset))))
	http.HandleFunc("/control/version.json", postInstall(optionalAuth(handleGetVersionJSON)))
	http.HandleFunc("/control/update", postInstall(optionalAuth(ensurePOST(handleUpdate))))
	http.HandleFunc("/control/filtering/enable", postInstall(optionalAuth(ensurePOST(handleFilteringEnable))))
	http.HandleFunc("/control/filtering/disable", postInstall(optionalAuth(ensurePOST(handleFilteringDisable))))
	http.HandleFunc("/control/filtering/add_url", postInstall(optionalAuth(ensurePOST(handleFilteringAddURL))))
	http.HandleFunc("/control/filtering/remove_url", postInstall(optionalAuth(ensurePOST(handleFilteringRemoveURL))))
	http.HandleFunc("/control/filtering/enable_url", postInstall(optionalAuth(ensurePOST(handleFilteringEnableURL))))
	http.HandleFunc("/control/filtering/disable_url", postInstall(optionalAuth(ensurePOST(handleFilteringDisableURL))))
	http.HandleFunc("/control/filtering/refresh", postInstall(optionalAuth(ensurePOST(handleFilteringRefresh))))
	http.HandleFunc("/control/filtering/status", postInstall(optionalAuth(ensureGET(handleFilteringStatus))))
	http.HandleFunc("/control/filtering/set_rules", postInstall(optionalAuth(ensurePOST(handleFilteringSetRules))))
	http.HandleFunc("/control/safebrowsing/enable", postInstall(optionalAuth(ensurePOST(handleSafeBrowsingEnable))))
	http.HandleFunc("/control/safebrowsing/disable", postInstall(optionalAuth(ensurePOST(handleSafeBrowsingDisable))))
	http.HandleFunc("/control/safebrowsing/status", postInstall(optionalAuth(ensureGET(handleSafeBrowsingStatus))))
	http.HandleFunc("/control/parental/enable", postInstall(optionalAuth(ensurePOST(handleParentalEnable))))
	http.HandleFunc("/control/parental/disable", postInstall(optionalAuth(ensurePOST(handleParentalDisable))))
	http.HandleFunc("/control/parental/status", postInstall(optionalAuth(ensureGET(handleParentalStatus))))
	http.HandleFunc("/control/safesearch/enable", postInstall(optionalAuth(ensurePOST(handleSafeSearchEnable))))
	http.HandleFunc("/control/safesearch/disable", postInstall(optionalAuth(ensurePOST(handleSafeSearchDisable))))
	http.HandleFunc("/control/safesearch/status", postInstall(optionalAuth(ensureGET(handleSafeSearchStatus))))
	http.HandleFunc("/control/dhcp/status", postInstall(optionalAuth(ensureGET(handleDHCPStatus))))
	http.HandleFunc("/control/dhcp/interfaces", postInstall(optionalAuth(ensureGET(handleDHCPInterfaces))))
	http.HandleFunc("/control/dhcp/set_config", postInstall(optionalAuth(ensurePOST(handleDHCPSetConfig))))
	http.HandleFunc("/control/dhcp/find_active_dhcp", postInstall(optionalAuth(ensurePOST(handleDHCPFindActiveServer))))

	RegisterTLSHandlers()
	RegisterClientsHandlers()

	http.HandleFunc("/dns-query", postInstall(handleDOH))
}
