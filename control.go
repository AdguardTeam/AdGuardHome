package main

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/dnsforward"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/log"
	"github.com/joomcode/errorx"
	"github.com/miekg/dns"
	govalidator "gopkg.in/asaskevich/govalidator.v4"
)

const updatePeriod = time.Minute * 30

// cached version.json to avoid hammering github.io for each page reload
var versionCheckJSON []byte
var versionCheckLastTime time.Time

const versionCheckURL = "https://adguardteam.github.io/AdGuardHome/version.json"
const versionCheckPeriod = time.Hour * 8

var client = &http.Client{
	Timeout: time.Second * 30,
}

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
	data := map[string]interface{}{
		"dns_address":        config.DNS.BindHost,
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

	for _, u := range newconfig.Upstreams {
		if err = validateUpstream(u); err != nil {
			httpError(w, http.StatusBadRequest, "%s can not be used as upstream cause: %s", u, err)
			return
		}
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

func validateUpstream(upstream string) error {
	if strings.HasPrefix(upstream, "tls://") || strings.HasPrefix(upstream, "https://") || strings.HasPrefix(upstream, "sdns://") || strings.HasPrefix(upstream, "tcp://") {
		return nil
	}

	if strings.Contains(upstream, "://") {
		return fmt.Errorf("wrong protocol")
	}

	return checkPlainDNS(upstream)
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

func handleGetVersionJSON(w http.ResponseWriter, r *http.Request) {
	log.Tracef("%s %v", r.Method, r.URL)
	now := time.Now()
	if now.Sub(versionCheckLastTime) <= versionCheckPeriod && len(versionCheckJSON) != 0 {
		// return cached copy
		w.Header().Set("Content-Type", "application/json")
		w.Write(versionCheckJSON)
		return
	}

	resp, err := client.Get(versionCheckURL)
	if err != nil {
		httpError(w, http.StatusBadGateway, "Couldn't get version check json from %s: %T %s\n", versionCheckURL, err, err)
		return
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	// read the body entirely
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		httpError(w, http.StatusBadGateway, "Couldn't read response body from %s: %s", versionCheckURL, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(body)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Couldn't write body: %s", err)
	}

	versionCheckLastTime = now
	versionCheckJSON = body
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
	for i := range config.Filters {
		if config.Filters[i].URL == f.URL {
			httpError(w, http.StatusBadRequest, "Filter URL already added -- %s", f.URL)
			return
		}
	}

	// Set necessary properties
	f.ID = assignUniqueFilterID()
	f.Enabled = true

	// Download the filter contents
	ok, err := f.update(true)
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
	config.Filters = append(config.Filters, f)
	err = writeAllConfigs()
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Couldn't write config file: %s", err)
		return
	}

	err = reconfigureDNSServer()
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Couldn't reconfigure the DNS server: %s", err)
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

	found := false
	for i := range config.Filters {
		filter := &config.Filters[i] // otherwise we will be operating on a copy
		if filter.URL == url {
			filter.Enabled = true
			found = true
		}
	}

	if !found {
		http.Error(w, "URL parameter was not previously added", http.StatusBadRequest)
		return
	}

	// kick off refresh of rules from new URLs
	refreshFiltersIfNecessary(false)
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

	found := false
	for i := range config.Filters {
		filter := &config.Filters[i] // otherwise we will be operating on a copy
		if filter.URL == url {
			filter.Enabled = false
			found = true
		}
	}

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
	force := r.URL.Query().Get("force")
	updated := refreshFiltersIfNecessary(force != "")
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
		http.Error(w, "URL parameter was not specified", 400)
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

type ipport struct {
	IP      string `json:"ip,omitempty"`
	Port    int    `json:"port"`
	Warning string `json:"warning"`
}

type firstRunData struct {
	Web        ipport                 `json:"web"`
	DNS        ipport                 `json:"dns"`
	Username   string                 `json:"username,omitempty"`
	Password   string                 `json:"password,omitempty"`
	Interfaces map[string]interface{} `json:"interfaces"`
}

func handleInstallGetAddresses(w http.ResponseWriter, r *http.Request) {
	log.Tracef("%s %v", r.Method, r.URL)
	data := firstRunData{}

	// find out if port 80 is available -- if not, fall back to 3000
	if checkPortAvailable("", 80) == nil {
		data.Web.Port = 80
	} else {
		data.Web.Port = 3000
	}

	// find out if port 53 is available -- if not, show a big warning
	data.DNS.Port = 53
	if checkPacketPortAvailable("", 53) != nil {
		data.DNS.Warning = "Port 53 is not available for binding -- this will make DNS clients unable to contact AdGuard Home."
	}

	ifaces, err := getValidNetInterfacesForWeb()
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Couldn't get interfaces: %s", err)
		return
	}

	data.Interfaces = make(map[string]interface{})
	for _, iface := range ifaces {
		data.Interfaces[iface.Name] = iface
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(data)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Unable to marshal default addresses to json: %s", err)
		return
	}
}

func handleInstallConfigure(w http.ResponseWriter, r *http.Request) {
	log.Tracef("%s %v", r.Method, r.URL)
	newSettings := firstRunData{}
	err := json.NewDecoder(r.Body).Decode(&newSettings)
	if err != nil {
		httpError(w, http.StatusBadRequest, "Failed to parse new config json: %s", err)
		return
	}

	restartHTTP := true
	if config.BindHost == newSettings.Web.IP && config.BindPort == newSettings.Web.Port {
		// no need to rebind
		restartHTTP = false
	}

	// validate that hosts and ports are bindable
	if restartHTTP {
		err = checkPortAvailable(newSettings.Web.IP, newSettings.Web.Port)
		if err != nil {
			httpError(w, http.StatusBadRequest, "Impossible to listen on IP:port %s due to %s", net.JoinHostPort(newSettings.Web.IP, strconv.Itoa(newSettings.Web.Port)), err)
			return
		}
	}

	err = checkPacketPortAvailable(newSettings.DNS.IP, newSettings.DNS.Port)
	if err != nil {
		httpError(w, http.StatusBadRequest, "Impossible to listen on IP:port %s due to %s", net.JoinHostPort(newSettings.DNS.IP, strconv.Itoa(newSettings.DNS.Port)), err)
		return
	}

	config.firstRun = false
	config.BindHost = newSettings.Web.IP
	config.BindPort = newSettings.Web.Port
	config.DNS.BindHost = newSettings.DNS.IP
	config.DNS.Port = newSettings.DNS.Port
	config.AuthName = newSettings.Username
	config.AuthPass = newSettings.Password

	if config.DNS.Port != 0 {
		err = startDNSServer()
		if err != nil {
			httpError(w, http.StatusInternalServerError, "Couldn't start DNS server: %s", err)
			return
		}
	}

	httpUpdateConfigReloadDNSReturnOK(w, r)
	// this needs to be done in a goroutine because Shutdown() is a blocking call, and it will block
	// until all requests are finished, and _we_ are inside a request right now, so it will block indefinitely
	if restartHTTP {
		go func() {
			httpServer.Shutdown(context.TODO())
		}()
	}
}

// ---
// TLS
// ---
func handleTLSStatus(w http.ResponseWriter, r *http.Request) {
	log.Tracef("%s %v", r.Method, r.URL)
	marshalTLS(w, config.TLS)
}

func handleTLSValidate(w http.ResponseWriter, r *http.Request) {
	log.Tracef("%s %v", r.Method, r.URL)
	data, err := unmarshalTLS(r)
	if err != nil {
		httpError(w, http.StatusBadRequest, "Failed to unmarshal TLS config: %s", err)
		return
	}

	// check if port is available
	// BUT: if we are already using this port, no need
	alreadyRunning := false
	if httpsServer.server != nil {
		alreadyRunning = true
	}
	if !alreadyRunning {
		err = checkPortAvailable(config.BindHost, data.PortHTTPS)
		if err != nil {
			httpError(w, http.StatusBadRequest, "port %d is not available, cannot enable HTTPS on it", data.PortHTTPS)
			return
		}
	}

	data = validateCertificates(data)
	marshalTLS(w, data)
}

func handleTLSConfigure(w http.ResponseWriter, r *http.Request) {
	log.Tracef("%s %v", r.Method, r.URL)
	data, err := unmarshalTLS(r)
	if err != nil {
		httpError(w, http.StatusBadRequest, "Failed to unmarshal TLS config: %s", err)
		return
	}

	// check if port is available
	// BUT: if we are already using this port, no need
	alreadyRunning := false
	if httpsServer.server != nil {
		alreadyRunning = true
	}
	if !alreadyRunning {
		err = checkPortAvailable(config.BindHost, data.PortHTTPS)
		if err != nil {
			httpError(w, http.StatusBadRequest, "port %d is not available, cannot enable HTTPS on it", data.PortHTTPS)
			return
		}
	}

	restartHTTPS := false
	data = validateCertificates(data)
	if !reflect.DeepEqual(config.TLS.tlsConfigSettings, data.tlsConfigSettings) {
		log.Printf("tls config settings have changed, will restart HTTPS server")
		restartHTTPS = true
	}
	config.TLS = data
	err = writeAllConfigsAndReloadDNS()
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Couldn't write config file: %s", err)
		return
	}
	marshalTLS(w, data)
	// this needs to be done in a goroutine because Shutdown() is a blocking call, and it will block
	// until all requests are finished, and _we_ are inside a request right now, so it will block indefinitely
	if restartHTTPS {
		go func() {
			time.Sleep(time.Second) // TODO: could not find a way to reliably know that data was fully sent to client by https server, so we wait a bit to let response through before closing the server
			httpsServer.cond.L.Lock()
			httpsServer.cond.Broadcast()
			if httpsServer.server != nil {
				httpsServer.server.Shutdown(context.TODO())
			}
			httpsServer.cond.L.Unlock()
		}()
	}
}

func validateCertificates(data tlsConfig) tlsConfig {
	var err error

	// clear out status for certificates
	data.tlsConfigStatus = tlsConfigStatus{}

	// check only public certificate separately from the key
	if data.CertificateChain != "" {
		log.Tracef("got certificate: %s", data.CertificateChain)

		// now do a more extended validation
		var certs []*pem.Block    // PEM-encoded certificates
		var skippedBytes []string // skipped bytes

		pemblock := []byte(data.CertificateChain)
		for {
			var decoded *pem.Block
			decoded, pemblock = pem.Decode(pemblock)
			if decoded == nil {
				break
			}
			if decoded.Type == "CERTIFICATE" {
				certs = append(certs, decoded)
			} else {
				skippedBytes = append(skippedBytes, decoded.Type)
			}
		}

		var parsedCerts []*x509.Certificate

		for _, cert := range certs {
			parsed, err := x509.ParseCertificate(cert.Bytes)
			if err != nil {
				data.WarningValidation = fmt.Sprintf("Failed to parse certificate: %s", err)
				return data
			}
			parsedCerts = append(parsedCerts, parsed)
		}

		if len(parsedCerts) == 0 {
			data.WarningValidation = fmt.Sprintf("You have specified an empty certificate")
			return data
		}

		data.ValidCert = true

		// spew.Dump(parsedCerts)

		opts := x509.VerifyOptions{
			DNSName: data.ServerName,
		}

		log.Printf("number of certs - %d", len(parsedCerts))
		if len(parsedCerts) > 1 {
			// set up an intermediate
			pool := x509.NewCertPool()
			for _, cert := range parsedCerts[1:] {
				log.Printf("got an intermediate cert")
				pool.AddCert(cert)
			}
			opts.Intermediates = pool
		}

		// TODO: save it as a warning rather than error it out -- shouldn't be a big problem
		mainCert := parsedCerts[0]
		_, err := mainCert.Verify(opts)
		if err != nil {
			// let self-signed certs through
			data.WarningValidation = fmt.Sprintf("Your certificate does not verify: %s", err)
		} else {
			data.ValidChain = true
		}
		// spew.Dump(chains)

		// update status
		if mainCert != nil {
			notAfter := mainCert.NotAfter
			data.Subject = mainCert.Subject.String()
			data.Issuer = mainCert.Issuer.String()
			data.NotAfter = notAfter
			data.NotBefore = mainCert.NotBefore
			data.DNSNames = mainCert.DNSNames
		}
	}

	// validate private key (right now the only validation possible is just parsing it)
	if data.PrivateKey != "" {
		// now do a more extended validation
		var key *pem.Block        // PEM-encoded certificates
		var skippedBytes []string // skipped bytes

		// go through all pem blocks, but take first valid pem block and drop the rest
		pemblock := []byte(data.PrivateKey)
		for {
			var decoded *pem.Block
			decoded, pemblock = pem.Decode(pemblock)
			if decoded == nil {
				break
			}
			if decoded.Type == "PRIVATE KEY" || strings.HasSuffix(decoded.Type, " PRIVATE KEY") {
				key = decoded
				break
			} else {
				skippedBytes = append(skippedBytes, decoded.Type)
			}
		}

		if key == nil {
			data.WarningValidation = "No valid keys were found"
			return data
		}

		// parse the decoded key
		_, keytype, err := parsePrivateKey(key.Bytes)
		if err != nil {
			data.WarningValidation = fmt.Sprintf("Failed to parse private key: %s", err)
			return data
		}

		data.ValidKey = true
		data.KeyType = keytype
	}

	// if both are set, validate both in unison
	if data.PrivateKey != "" && data.CertificateChain != "" {
		_, err = tls.X509KeyPair([]byte(data.CertificateChain), []byte(data.PrivateKey))
		if err != nil {
			data.WarningValidation = fmt.Sprintf("Invalid certificate or key: %s", err)
			return data
		}
		data.usable = true
	}

	return data
}

// Attempt to parse the given private key DER block. OpenSSL 0.9.8 generates
// PKCS#1 private keys by default, while OpenSSL 1.0.0 generates PKCS#8 keys.
// OpenSSL ecparam generates SEC1 EC private keys for ECDSA. We try all three.
func parsePrivateKey(der []byte) (crypto.PrivateKey, string, error) {
	if key, err := x509.ParsePKCS1PrivateKey(der); err == nil {
		return key, "RSA", nil
	}
	if key, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		switch key := key.(type) {
		case *rsa.PrivateKey:
			return key, "RSA", nil
		case *ecdsa.PrivateKey:
			return key, "ECDSA", nil
		default:
			return nil, "", errors.New("tls: found unknown private key type in PKCS#8 wrapping")
		}
	}
	if key, err := x509.ParseECPrivateKey(der); err == nil {
		return key, "ECDSA", nil
	}

	return nil, "", errors.New("tls: failed to parse private key")
}

// unmarshalTLS handles base64-encoded certificates transparently
func unmarshalTLS(r *http.Request) (tlsConfig, error) {
	data := tlsConfig{}
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		return data, errorx.Decorate(err, "Failed to parse new TLS config json")
	}

	if data.CertificateChain != "" {
		certPEM, err := base64.StdEncoding.DecodeString(data.CertificateChain)
		if err != nil {
			return data, errorx.Decorate(err, "Failed to base64-decode certificate chain")
		}
		data.CertificateChain = string(certPEM)
	}

	if data.PrivateKey != "" {
		keyPEM, err := base64.StdEncoding.DecodeString(data.PrivateKey)
		if err != nil {
			return data, errorx.Decorate(err, "Failed to base64-decode private key")
		}

		data.PrivateKey = string(keyPEM)
	}

	return data, nil
}

func marshalTLS(w http.ResponseWriter, data tlsConfig) {
	w.Header().Set("Content-Type", "application/json")
	if data.CertificateChain != "" {
		encoded := base64.StdEncoding.EncodeToString([]byte(data.CertificateChain))
		data.CertificateChain = encoded
	}
	if data.PrivateKey != "" {
		encoded := base64.StdEncoding.EncodeToString([]byte(data.PrivateKey))
		data.PrivateKey = encoded
	}
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Failed to marshal json with TLS status: %s", err)
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
func registerInstallHandlers() {
	http.HandleFunc("/control/install/get_addresses", preInstall(ensureGET(handleInstallGetAddresses)))
	http.HandleFunc("/control/install/configure", preInstall(ensurePOST(handleInstallConfigure)))
}

func registerControlHandlers() {
	http.HandleFunc("/control/status", postInstall(optionalAuth(ensureGET(handleStatus))))
	http.HandleFunc("/control/enable_protection", postInstall(optionalAuth(ensurePOST(handleProtectionEnable))))
	http.HandleFunc("/control/disable_protection", postInstall(optionalAuth(ensurePOST(handleProtectionDisable))))
	http.HandleFunc("/control/querylog", postInstall(optionalAuth(ensureGET(handleQueryLog))))
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
	http.HandleFunc("/control/filtering/enable", postInstall(optionalAuth(ensurePOST(handleFilteringEnable))))
	http.HandleFunc("/control/filtering/disable", postInstall(optionalAuth(ensurePOST(handleFilteringDisable))))
	http.HandleFunc("/control/filtering/add_url", postInstall(optionalAuth(ensurePUT(handleFilteringAddURL))))
	http.HandleFunc("/control/filtering/remove_url", postInstall(optionalAuth(ensureDELETE(handleFilteringRemoveURL))))
	http.HandleFunc("/control/filtering/enable_url", postInstall(optionalAuth(ensurePOST(handleFilteringEnableURL))))
	http.HandleFunc("/control/filtering/disable_url", postInstall(optionalAuth(ensurePOST(handleFilteringDisableURL))))
	http.HandleFunc("/control/filtering/refresh", postInstall(optionalAuth(ensurePOST(handleFilteringRefresh))))
	http.HandleFunc("/control/filtering/status", postInstall(optionalAuth(ensureGET(handleFilteringStatus))))
	http.HandleFunc("/control/filtering/set_rules", postInstall(optionalAuth(ensurePUT(handleFilteringSetRules))))
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

	http.HandleFunc("/control/tls/status", postInstall(optionalAuth(ensureGET(handleTLSStatus))))
	http.HandleFunc("/control/tls/configure", postInstall(optionalAuth(ensurePOST(handleTLSConfigure))))
	http.HandleFunc("/control/tls/validate", postInstall(optionalAuth(ensurePOST(handleTLSValidate))))

	http.HandleFunc("/dns-query", postInstall(handleDOH))
}
