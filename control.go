package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/dnsforward"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/hmage/golibs/log"
	"github.com/miekg/dns"
	"gopkg.in/asaskevich/govalidator.v4"
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

// -------------------
// dns run control
// -------------------
func writeAllConfigsAndReloadDNS() error {
	err := writeAllConfigs()
	if err != nil {
		log.Printf("Couldn't write all configs: %s", err)
		return err
	}
	return reconfigureDNSServer()
}

func httpUpdateConfigReloadDNSReturnOK(w http.ResponseWriter, r *http.Request) {
	err := writeAllConfigsAndReloadDNS()
	if err != nil {
		errorText := fmt.Sprintf("Couldn't write config file: %s", err)
		log.Println(errorText)
		http.Error(w, errorText, http.StatusInternalServerError)
		return
	}
	returnOK(w)
}

func returnOK(w http.ResponseWriter) {
	_, err := fmt.Fprintf(w, "OK\n")
	if err != nil {
		errorText := fmt.Sprintf("Couldn't write body: %s", err)
		log.Println(errorText)
		http.Error(w, errorText, http.StatusInternalServerError)
	}
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"dns_address":        config.BindHost,
		"dns_port":           config.DNS.Port,
		"protection_enabled": config.DNS.ProtectionEnabled,
		"querylog_enabled":   config.DNS.QueryLogEnabled,
		"running":            isRunning(),
		"bootstrap_dns":      config.DNS.BootstrapDNS,
		"upstream_dns":       config.DNS.UpstreamDNS,
		"version":            VersionString,
		"language":           config.Language,
	}

	jsonVal, err := json.Marshal(data)
	if err != nil {
		errorText := fmt.Sprintf("Unable to marshal status json: %s", err)
		log.Println(errorText)
		http.Error(w, errorText, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonVal)
	if err != nil {
		errorText := fmt.Sprintf("Unable to write response json: %s", err)
		log.Println(errorText)
		http.Error(w, errorText, 500)
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

func httpError(w http.ResponseWriter, code int, format string, args ...interface{}) {
	text := fmt.Sprintf(format, args...)
	log.Println(text)
	http.Error(w, text, code)
}

func handleSetUpstreamDNS(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		errorText := fmt.Sprintf("Failed to read request body: %s", err)
		log.Println(errorText)
		http.Error(w, errorText, http.StatusBadRequest)
		return
	}
	// if empty body -- user is asking for default servers
	hosts := strings.Fields(string(body))

	if len(hosts) == 0 {
		config.DNS.UpstreamDNS = defaultDNS
	} else {
		config.DNS.UpstreamDNS = hosts
	}

	err = writeAllConfigs()
	if err != nil {
		errorText := fmt.Sprintf("Couldn't write config file: %s", err)
		log.Println(errorText)
		http.Error(w, errorText, http.StatusInternalServerError)
		return
	}
	err = reconfigureDNSServer()
	if err != nil {
		errorText := fmt.Sprintf("Couldn't reconfigure the DNS server: %s", err)
		log.Println(errorText)
		http.Error(w, errorText, http.StatusInternalServerError)
		return
	}
	_, err = fmt.Fprintf(w, "OK %d servers\n", len(hosts))
	if err != nil {
		errorText := fmt.Sprintf("Couldn't write body: %s", err)
		log.Println(errorText)
		http.Error(w, errorText, http.StatusInternalServerError)
	}
}

func handleTestUpstreamDNS(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		errorText := fmt.Sprintf("Failed to read request body: %s", err)
		log.Println(errorText)
		http.Error(w, errorText, 400)
		return
	}
	hosts := strings.Fields(string(body))

	if len(hosts) == 0 {
		errorText := fmt.Sprintf("No servers specified")
		log.Println(errorText)
		http.Error(w, errorText, http.StatusBadRequest)
		return
	}

	result := map[string]string{}

	for _, host := range hosts {
		err = checkDNS(host)
		if err != nil {
			log.Println(err)
			result[host] = err.Error()
		} else {
			result[host] = "OK"
		}
	}

	jsonVal, err := json.Marshal(result)
	if err != nil {
		errorText := fmt.Sprintf("Unable to marshal status json: %s", err)
		log.Println(errorText)
		http.Error(w, errorText, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonVal)
	if err != nil {
		errorText := fmt.Sprintf("Couldn't write body: %s", err)
		log.Println(errorText)
		http.Error(w, errorText, http.StatusInternalServerError)
	}
}

func checkDNS(input string) error {
	log.Printf("Checking if DNS %s works...", input)
	u, err := upstream.AddressToUpstream(input, "", dnsforward.DefaultTimeout)
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

	log.Printf("DNS %s works OK", input)
	return nil
}

func handleGetVersionJSON(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	if now.Sub(versionCheckLastTime) <= versionCheckPeriod && len(versionCheckJSON) != 0 {
		// return cached copy
		w.Header().Set("Content-Type", "application/json")
		w.Write(versionCheckJSON)
		return
	}

	resp, err := client.Get(versionCheckURL)
	if err != nil {
		errorText := fmt.Sprintf("Couldn't get version check json from %s: %T %s\n", versionCheckURL, err, err)
		log.Println(errorText)
		http.Error(w, errorText, http.StatusBadGateway)
		return
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	// read the body entirely
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		errorText := fmt.Sprintf("Couldn't read response body from %s: %s", versionCheckURL, err)
		log.Println(errorText)
		http.Error(w, errorText, http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(body)
	if err != nil {
		errorText := fmt.Sprintf("Couldn't write body: %s", err)
		log.Println(errorText)
		http.Error(w, errorText, http.StatusInternalServerError)
	}

	versionCheckLastTime = now
	versionCheckJSON = body
}

// ---------
// filtering
// ---------

func handleFilteringEnable(w http.ResponseWriter, r *http.Request) {
	config.DNS.FilteringEnabled = true
	httpUpdateConfigReloadDNSReturnOK(w, r)
}

func handleFilteringDisable(w http.ResponseWriter, r *http.Request) {
	config.DNS.FilteringEnabled = false
	httpUpdateConfigReloadDNSReturnOK(w, r)
}

func handleFilteringStatus(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"enabled": config.DNS.FilteringEnabled,
	}

	config.RLock()
	data["filters"] = config.Filters
	data["user_rules"] = config.UserRules
	jsonVal, err := json.Marshal(data)
	config.RUnlock()

	if err != nil {
		errorText := fmt.Sprintf("Unable to marshal status json: %s", err)
		log.Println(errorText)
		http.Error(w, errorText, 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonVal)
	if err != nil {
		errorText := fmt.Sprintf("Unable to write response json: %s", err)
		log.Println(errorText)
		http.Error(w, errorText, 500)
		return
	}
}

func handleFilteringAddURL(w http.ResponseWriter, r *http.Request) {
	filter := filter{}
	err := json.NewDecoder(r.Body).Decode(&filter)
	if err != nil {
		httpError(w, http.StatusBadRequest, "Failed to parse request body json: %s", err)
		return
	}

	if len(filter.URL) == 0 {
		http.Error(w, "URL parameter was not specified", 400)
		return
	}

	if valid := govalidator.IsRequestURL(filter.URL); !valid {
		http.Error(w, "URL parameter is not valid request URL", 400)
		return
	}

	// Check for duplicates
	for i := range config.Filters {
		if config.Filters[i].URL == filter.URL {
			errorText := fmt.Sprintf("Filter URL already added -- %s", filter.URL)
			log.Println(errorText)
			http.Error(w, errorText, http.StatusBadRequest)
			return
		}
	}

	// Set necessary properties
	filter.ID = assignUniqueFilterID()
	filter.Enabled = true

	// Download the filter contents
	ok, err := filter.update(true)
	if err != nil {
		errorText := fmt.Sprintf("Couldn't fetch filter from url %s: %s", filter.URL, err)
		log.Println(errorText)
		http.Error(w, errorText, http.StatusBadRequest)
		return
	}
	if filter.RulesCount == 0 {
		errorText := fmt.Sprintf("Filter at the url %s has no rules (maybe it points to blank page?)", filter.URL)
		log.Println(errorText)
		http.Error(w, errorText, http.StatusBadRequest)
		return
	}
	if !ok {
		errorText := fmt.Sprintf("Filter at the url %s is invalid (maybe it points to blank page?)", filter.URL)
		log.Println(errorText)
		http.Error(w, errorText, http.StatusBadRequest)
		return
	}

	// Save the filter contents
	err = filter.save()
	if err != nil {
		errorText := fmt.Sprintf("Failed to save filter %d due to %s", filter.ID, err)
		log.Println(errorText)
		http.Error(w, errorText, http.StatusBadRequest)
		return
	}

	// URL is deemed valid, append it to filters, update config, write new filter file and tell dns to reload it
	// TODO: since we directly feed filters in-memory, revisit if writing configs is always necessary
	config.Filters = append(config.Filters, filter)
	err = writeAllConfigs()
	if err != nil {
		errorText := fmt.Sprintf("Couldn't write config file: %s", err)
		log.Println(errorText)
		http.Error(w, errorText, http.StatusInternalServerError)
		return
	}

	reconfigureDNSServer()

	_, err = fmt.Fprintf(w, "OK %d rules\n", filter.RulesCount)
	if err != nil {
		errorText := fmt.Sprintf("Couldn't write body: %s", err)
		log.Println(errorText)
		http.Error(w, errorText, http.StatusInternalServerError)
	}
}

func handleFilteringRemoveURL(w http.ResponseWriter, r *http.Request) {
	parameters, err := parseParametersFromBody(r.Body)
	if err != nil {
		errorText := fmt.Sprintf("failed to parse parameters from body: %s", err)
		log.Println(errorText)
		http.Error(w, errorText, 400)
		return
	}

	url, ok := parameters["url"]
	if !ok {
		http.Error(w, "URL parameter was not specified", 400)
		return
	}

	if valid := govalidator.IsRequestURL(url); !valid {
		http.Error(w, "URL parameter is not valid request URL", 400)
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
				errorText := fmt.Sprintf("Couldn't remove the filter file: %s", err)
				http.Error(w, errorText, http.StatusInternalServerError)
				return
			}
		}
	}
	// Update the configuration after removing filter files
	config.Filters = newFilters
	httpUpdateConfigReloadDNSReturnOK(w, r)
}

func handleFilteringEnableURL(w http.ResponseWriter, r *http.Request) {
	parameters, err := parseParametersFromBody(r.Body)
	if err != nil {
		errorText := fmt.Sprintf("failed to parse parameters from body: %s", err)
		log.Println(errorText)
		http.Error(w, errorText, 400)
		return
	}

	url, ok := parameters["url"]
	if !ok {
		http.Error(w, "URL parameter was not specified", 400)
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
	parameters, err := parseParametersFromBody(r.Body)
	if err != nil {
		errorText := fmt.Sprintf("failed to parse parameters from body: %s", err)
		log.Println(errorText)
		http.Error(w, errorText, 400)
		return
	}

	url, ok := parameters["url"]
	if !ok {
		http.Error(w, "URL parameter was not specified", 400)
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
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		errorText := fmt.Sprintf("Failed to read request body: %s", err)
		log.Println(errorText)
		http.Error(w, errorText, 400)
		return
	}

	config.UserRules = strings.Split(string(body), "\n")
	httpUpdateConfigReloadDNSReturnOK(w, r)
}

func handleFilteringRefresh(w http.ResponseWriter, r *http.Request) {
	force := r.URL.Query().Get("force")
	updated := refreshFiltersIfNecessary(force != "")
	fmt.Fprintf(w, "OK %d filters updated\n", updated)
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
		errorText := fmt.Sprintf("Unable to marshal status json: %s", err)
		log.Println(errorText)
		http.Error(w, errorText, 500)
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonVal)
	if err != nil {
		errorText := fmt.Sprintf("Unable to write response json: %s", err)
		log.Println(errorText)
		http.Error(w, errorText, 500)
		return
	}
}

// --------
// parental
// --------
func handleParentalEnable(w http.ResponseWriter, r *http.Request) {
	parameters, err := parseParametersFromBody(r.Body)
	if err != nil {
		errorText := fmt.Sprintf("failed to parse parameters from body: %s", err)
		log.Println(errorText)
		http.Error(w, errorText, 400)
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
		errorText := fmt.Sprintf("Unable to marshal status json: %s", err)
		log.Println(errorText)
		http.Error(w, errorText, 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonVal)
	if err != nil {
		errorText := fmt.Sprintf("Unable to write response json: %s", err)
		log.Println(errorText)
		http.Error(w, errorText, 500)
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
		errorText := fmt.Sprintf("Unable to marshal status json: %s", err)
		log.Println(errorText)
		http.Error(w, errorText, 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonVal)
	if err != nil {
		errorText := fmt.Sprintf("Unable to write response json: %s", err)
		log.Println(errorText)
		http.Error(w, errorText, 500)
		return
	}
}

func registerControlHandlers() {
	http.HandleFunc("/control/status", optionalAuth(ensureGET(handleStatus)))
	http.HandleFunc("/control/enable_protection", optionalAuth(ensurePOST(handleProtectionEnable)))
	http.HandleFunc("/control/disable_protection", optionalAuth(ensurePOST(handleProtectionDisable)))
	http.HandleFunc("/control/querylog", optionalAuth(ensureGET(dnsforward.HandleQueryLog)))
	http.HandleFunc("/control/querylog_enable", optionalAuth(ensurePOST(handleQueryLogEnable)))
	http.HandleFunc("/control/querylog_disable", optionalAuth(ensurePOST(handleQueryLogDisable)))
	http.HandleFunc("/control/set_upstream_dns", optionalAuth(ensurePOST(handleSetUpstreamDNS)))
	http.HandleFunc("/control/test_upstream_dns", optionalAuth(ensurePOST(handleTestUpstreamDNS)))
	http.HandleFunc("/control/i18n/change_language", optionalAuth(ensurePOST(handleI18nChangeLanguage)))
	http.HandleFunc("/control/i18n/current_language", optionalAuth(ensureGET(handleI18nCurrentLanguage)))
	http.HandleFunc("/control/stats_top", optionalAuth(ensureGET(dnsforward.HandleStatsTop)))
	http.HandleFunc("/control/stats", optionalAuth(ensureGET(dnsforward.HandleStats)))
	http.HandleFunc("/control/stats_history", optionalAuth(ensureGET(dnsforward.HandleStatsHistory)))
	http.HandleFunc("/control/stats_reset", optionalAuth(ensurePOST(dnsforward.HandleStatsReset)))
	http.HandleFunc("/control/version.json", optionalAuth(handleGetVersionJSON))
	http.HandleFunc("/control/filtering/enable", optionalAuth(ensurePOST(handleFilteringEnable)))
	http.HandleFunc("/control/filtering/disable", optionalAuth(ensurePOST(handleFilteringDisable)))
	http.HandleFunc("/control/filtering/add_url", optionalAuth(ensurePUT(handleFilteringAddURL)))
	http.HandleFunc("/control/filtering/remove_url", optionalAuth(ensureDELETE(handleFilteringRemoveURL)))
	http.HandleFunc("/control/filtering/enable_url", optionalAuth(ensurePOST(handleFilteringEnableURL)))
	http.HandleFunc("/control/filtering/disable_url", optionalAuth(ensurePOST(handleFilteringDisableURL)))
	http.HandleFunc("/control/filtering/refresh", optionalAuth(ensurePOST(handleFilteringRefresh)))
	http.HandleFunc("/control/filtering/status", optionalAuth(ensureGET(handleFilteringStatus)))
	http.HandleFunc("/control/filtering/set_rules", optionalAuth(ensurePUT(handleFilteringSetRules)))
	http.HandleFunc("/control/safebrowsing/enable", optionalAuth(ensurePOST(handleSafeBrowsingEnable)))
	http.HandleFunc("/control/safebrowsing/disable", optionalAuth(ensurePOST(handleSafeBrowsingDisable)))
	http.HandleFunc("/control/safebrowsing/status", optionalAuth(ensureGET(handleSafeBrowsingStatus)))
	http.HandleFunc("/control/parental/enable", optionalAuth(ensurePOST(handleParentalEnable)))
	http.HandleFunc("/control/parental/disable", optionalAuth(ensurePOST(handleParentalDisable)))
	http.HandleFunc("/control/parental/status", optionalAuth(ensureGET(handleParentalStatus)))
	http.HandleFunc("/control/safesearch/enable", optionalAuth(ensurePOST(handleSafeSearchEnable)))
	http.HandleFunc("/control/safesearch/disable", optionalAuth(ensurePOST(handleSafeSearchDisable)))
	http.HandleFunc("/control/safesearch/status", optionalAuth(ensureGET(handleSafeSearchStatus)))
	http.HandleFunc("/control/dhcp/status", optionalAuth(ensureGET(handleDHCPStatus)))
	http.HandleFunc("/control/dhcp/interfaces", optionalAuth(ensureGET(handleDHCPInterfaces)))
	http.HandleFunc("/control/dhcp/set_config", optionalAuth(ensurePOST(handleDHCPSetConfig)))
	http.HandleFunc("/control/dhcp/find_active_dhcp", optionalAuth(ensurePOST(handleDHCPFindActiveServer)))
}
