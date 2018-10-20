package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	coredns_plugin "github.com/AdguardTeam/AdGuardHome/coredns_plugin"
	"github.com/AdguardTeam/AdGuardHome/dnsfilter"
	"github.com/miekg/dns"
	"gopkg.in/asaskevich/govalidator.v4"
)

const updatePeriod = time.Minute * 30

var filterTitle = regexp.MustCompile(`^! Title: +(.*)$`)

// cached version.json to avoid hammering github.io for each page reload
var versionCheckJSON []byte
var versionCheckLastTime time.Time

const versionCheckURL = "https://adguardteam.github.io/AdGuardHome/version.json"
const versionCheckPeriod = time.Hour * 8

var client = &http.Client{
	Timeout: time.Second * 30,
}

// -------------------
// coredns run control
// -------------------
func tellCoreDNSToReload() {
	coredns_plugin.Reload <- true
}

func writeAllConfigsAndReloadCoreDNS() error {
	err := writeAllConfigs()
	if err != nil {
		log.Printf("Couldn't write all configs: %s", err)
		return err
	}
	tellCoreDNSToReload()
	return nil
}

func httpUpdateConfigReloadDNSReturnOK(w http.ResponseWriter, r *http.Request) {
	err := writeAllConfigsAndReloadCoreDNS()
	if err != nil {
		errortext := fmt.Sprintf("Couldn't write config file: %s", err)
		log.Println(errortext)
		http.Error(w, errortext, http.StatusInternalServerError)
		return
	}
	returnOK(w, r)
}

func returnOK(w http.ResponseWriter, r *http.Request) {
	_, err := fmt.Fprintf(w, "OK\n")
	if err != nil {
		errortext := fmt.Sprintf("Couldn't write body: %s", err)
		log.Println(errortext)
		http.Error(w, errortext, http.StatusInternalServerError)
	}
}

func handleStatus(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"dns_address":        config.BindHost,
		"dns_port":           config.CoreDNS.Port,
		"protection_enabled": config.CoreDNS.ProtectionEnabled,
		"querylog_enabled":   config.CoreDNS.QueryLogEnabled,
		"running":            isRunning(),
		"upstream_dns":       config.CoreDNS.UpstreamDNS,
		"version":            VersionString,
	}

	jsonVal, err := json.Marshal(data)
	if err != nil {
		errortext := fmt.Sprintf("Unable to marshal status json: %s", err)
		log.Println(errortext)
		http.Error(w, errortext, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonVal)
	if err != nil {
		errortext := fmt.Sprintf("Unable to write response json: %s", err)
		log.Println(errortext)
		http.Error(w, errortext, 500)
		return
	}
}

func handleProtectionEnable(w http.ResponseWriter, r *http.Request) {
	config.CoreDNS.ProtectionEnabled = true
	httpUpdateConfigReloadDNSReturnOK(w, r)
}

func handleProtectionDisable(w http.ResponseWriter, r *http.Request) {
	config.CoreDNS.ProtectionEnabled = false
	httpUpdateConfigReloadDNSReturnOK(w, r)
}

// -----
// stats
// -----
func handleQueryLogEnable(w http.ResponseWriter, r *http.Request) {
	config.CoreDNS.QueryLogEnabled = true
	httpUpdateConfigReloadDNSReturnOK(w, r)
}

func handleQueryLogDisable(w http.ResponseWriter, r *http.Request) {
	config.CoreDNS.QueryLogEnabled = false
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
		errortext := fmt.Sprintf("Failed to read request body: %s", err)
		log.Println(errortext)
		http.Error(w, errortext, http.StatusBadRequest)
		return
	}
	// if empty body -- user is asking for default servers
	hosts, err := sanitiseDNSServers(string(body))
	if err != nil {
		httpError(w, http.StatusBadRequest, "Invalid DNS servers were given: %s", err)
		return
	}
	if len(hosts) == 0 {
		config.CoreDNS.UpstreamDNS = defaultDNS
	} else {
		config.CoreDNS.UpstreamDNS = hosts
	}

	err = writeAllConfigs()
	if err != nil {
		errortext := fmt.Sprintf("Couldn't write config file: %s", err)
		log.Println(errortext)
		http.Error(w, errortext, http.StatusInternalServerError)
		return
	}
	tellCoreDNSToReload()
	_, err = fmt.Fprintf(w, "OK %d servers\n", len(hosts))
	if err != nil {
		errortext := fmt.Sprintf("Couldn't write body: %s", err)
		log.Println(errortext)
		http.Error(w, errortext, http.StatusInternalServerError)
	}
}

func handleTestUpstreamDNS(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		errortext := fmt.Sprintf("Failed to read request body: %s", err)
		log.Println(errortext)
		http.Error(w, errortext, 400)
		return
	}
	hosts := strings.Fields(string(body))

	if len(hosts) == 0 {
		errortext := fmt.Sprintf("No servers specified")
		log.Println(errortext)
		http.Error(w, errortext, http.StatusBadRequest)
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
		errortext := fmt.Sprintf("Unable to marshal status json: %s", err)
		log.Println(errortext)
		http.Error(w, errortext, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonVal)
	if err != nil {
		errortext := fmt.Sprintf("Couldn't write body: %s", err)
		log.Println(errortext)
		http.Error(w, errortext, http.StatusInternalServerError)
	}
}

func checkDNS(input string) error {
	input, err := sanitizeDNSServer(input)
	if err != nil {
		return err
	}

	req := dns.Msg{}
	req.Id = dns.Id()
	req.RecursionDesired = true
	req.Question = []dns.Question{
		{"google-public-dns-a.google.com.", dns.TypeA, dns.ClassINET},
	}

	prefix, host := splitDNSServerPrefixServer(input)

	c := dns.Client{
		Timeout: time.Minute,
	}
	switch prefix {
	case "tls://":
		c.Net = "tcp-tls"
	}

	resp, rtt, err := c.Exchange(&req, host)
	if err != nil {
		return fmt.Errorf("Couldn't communicate with DNS server %s: %s", input, err)
	}
	trace("exchange with %s took %v", input, rtt)
	if len(resp.Answer) != 1 {
		return fmt.Errorf("DNS server %s returned wrong answer", input)
	}
	if t, ok := resp.Answer[0].(*dns.A); ok {
		if !net.IPv4(8, 8, 8, 8).Equal(t.A) {
			return fmt.Errorf("DNS server %s returned wrong answer: %v", input, t.A)
		}
	}

	return nil
}

func sanitiseDNSServers(input string) ([]string, error) {
	fields := strings.Fields(input)
	hosts := []string{}
	for _, field := range fields {
		sanitized, err := sanitizeDNSServer(field)
		if err != nil {
			return hosts, err
		}
		hosts = append(hosts, sanitized)
	}
	return hosts, nil
}

func getDNSServerPrefix(input string) string {
	prefix := ""
	switch {
	case strings.HasPrefix(input, "dns://"):
		prefix = "dns://"
	case strings.HasPrefix(input, "tls://"):
		prefix = "tls://"
	}
	return prefix
}

func splitDNSServerPrefixServer(input string) (string, string) {
	prefix := getDNSServerPrefix(input)
	host := strings.TrimPrefix(input, prefix)
	return prefix, host
}

func sanitizeDNSServer(input string) (string, error) {
	prefix, host := splitDNSServerPrefixServer(input)
	host = appendPortIfMissing(prefix, host)
	{
		h, _, err := net.SplitHostPort(host)
		if err != nil {
			return "", err
		}
		ip := net.ParseIP(h)
		if ip == nil {
			return "", fmt.Errorf("Invalid DNS server field: %s", h)
		}
	}
	return prefix + host, nil
}

func appendPortIfMissing(prefix, input string) string {
	port := "53"
	switch prefix {
	case "tls://":
		port = "853"
	}
	_, _, err := net.SplitHostPort(input)
	if err == nil {
		return input
	}
	return net.JoinHostPort(input, port)
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
		errortext := fmt.Sprintf("Couldn't get querylog from coredns: %T %s\n", err, err)
		log.Println(errortext)
		http.Error(w, errortext, http.StatusBadGateway)
		return
	}
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	// read the body entirely
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		errortext := fmt.Sprintf("Couldn't read response body: %s", err)
		log.Println(errortext)
		http.Error(w, errortext, http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(body)
	if err != nil {
		errortext := fmt.Sprintf("Couldn't write body: %s", err)
		log.Println(errortext)
		http.Error(w, errortext, http.StatusInternalServerError)
	}

	versionCheckLastTime = now
	versionCheckJSON = body
}

// ---------
// filtering
// ---------

func handleFilteringEnable(w http.ResponseWriter, r *http.Request) {
	config.CoreDNS.FilteringEnabled = true
	httpUpdateConfigReloadDNSReturnOK(w, r)
}

func handleFilteringDisable(w http.ResponseWriter, r *http.Request) {
	config.CoreDNS.FilteringEnabled = false
	httpUpdateConfigReloadDNSReturnOK(w, r)
}

func handleFilteringStatus(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"enabled": config.CoreDNS.FilteringEnabled,
	}

	config.RLock()
	data["filters"] = config.Filters
	data["user_rules"] = config.UserRules
	jsonVal, err := json.Marshal(data)
	config.RUnlock()

	if err != nil {
		errortext := fmt.Sprintf("Unable to marshal status json: %s", err)
		log.Println(errortext)
		http.Error(w, errortext, 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonVal)
	if err != nil {
		errortext := fmt.Sprintf("Unable to write response json: %s", err)
		log.Println(errortext)
		http.Error(w, errortext, 500)
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

	filter.Enabled = true
	if len(filter.URL) == 0 {
		http.Error(w, "URL parameter was not specified", 400)
		return
	}

	if valid := govalidator.IsRequestURL(filter.URL); !valid {
		http.Error(w, "URL parameter is not valid request URL", 400)
		return
	}

	// check for duplicates
	for i := range config.Filters {
		if config.Filters[i].URL == filter.URL {
			errortext := fmt.Sprintf("Filter URL already added -- %s", filter.URL)
			log.Println(errortext)
			http.Error(w, errortext, http.StatusBadRequest)
			return
		}
	}

	ok, err := filter.update(time.Now())
	if err != nil {
		errortext := fmt.Sprintf("Couldn't fetch filter from url %s: %s", filter.URL, err)
		log.Println(errortext)
		http.Error(w, errortext, http.StatusBadRequest)
		return
	}
	if filter.RulesCount == 0 {
		errortext := fmt.Sprintf("Filter at url %s has no rules (maybe it points to blank page?)", filter.URL)
		log.Println(errortext)
		http.Error(w, errortext, http.StatusBadRequest)
		return
	}
	if !ok {
		errortext := fmt.Sprintf("Filter at url %s is invalid (maybe it points to blank page?)", filter.URL)
		log.Println(errortext)
		http.Error(w, errortext, http.StatusBadRequest)
		return
	}

	// URL is deemed valid, append it to filters, update config, write new filter file and tell coredns to reload it
	config.Filters = append(config.Filters, filter)
	err = writeAllConfigs()
	if err != nil {
		errortext := fmt.Sprintf("Couldn't write config file: %s", err)
		log.Println(errortext)
		http.Error(w, errortext, http.StatusInternalServerError)
		return
	}
	err = writeFilterFile()
	if err != nil {
		errortext := fmt.Sprintf("Couldn't write filter file: %s", err)
		log.Println(errortext)
		http.Error(w, errortext, http.StatusInternalServerError)
		return
	}
	tellCoreDNSToReload()
	_, err = fmt.Fprintf(w, "OK %d rules\n", filter.RulesCount)
	if err != nil {
		errortext := fmt.Sprintf("Couldn't write body: %s", err)
		log.Println(errortext)
		http.Error(w, errortext, http.StatusInternalServerError)
	}
}

func handleFilteringRemoveURL(w http.ResponseWriter, r *http.Request) {
	parameters, err := parseParametersFromBody(r.Body)
	if err != nil {
		errortext := fmt.Sprintf("failed to parse parameters from body: %s", err)
		log.Println(errortext)
		http.Error(w, errortext, 400)
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
		}
	}
	config.Filters = newFilters
	err = writeFilterFile()
	if err != nil {
		errortext := fmt.Sprintf("Couldn't write filter file: %s", err)
		log.Println(errortext)
		http.Error(w, errortext, http.StatusInternalServerError)
		return
	}
	httpUpdateConfigReloadDNSReturnOK(w, r)
}

func handleFilteringEnableURL(w http.ResponseWriter, r *http.Request) {
	parameters, err := parseParametersFromBody(r.Body)
	if err != nil {
		errortext := fmt.Sprintf("failed to parse parameters from body: %s", err)
		log.Println(errortext)
		http.Error(w, errortext, 400)
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
	refreshFiltersIfNeccessary()
	err = writeFilterFile()
	if err != nil {
		errortext := fmt.Sprintf("Couldn't write filter file: %s", err)
		log.Println(errortext)
		http.Error(w, errortext, http.StatusInternalServerError)
		return
	}
	httpUpdateConfigReloadDNSReturnOK(w, r)
}

func handleFilteringDisableURL(w http.ResponseWriter, r *http.Request) {
	parameters, err := parseParametersFromBody(r.Body)
	if err != nil {
		errortext := fmt.Sprintf("failed to parse parameters from body: %s", err)
		log.Println(errortext)
		http.Error(w, errortext, 400)
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

	err = writeFilterFile()
	if err != nil {
		errortext := fmt.Sprintf("Couldn't write filter file: %s", err)
		log.Println(errortext)
		http.Error(w, errortext, http.StatusInternalServerError)
		return
	}
	httpUpdateConfigReloadDNSReturnOK(w, r)
}

func handleFilteringSetRules(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		errortext := fmt.Sprintf("Failed to read request body: %s", err)
		log.Println(errortext)
		http.Error(w, errortext, 400)
		return
	}

	config.UserRules = strings.Split(string(body), "\n")
	err = writeFilterFile()
	if err != nil {
		errortext := fmt.Sprintf("Couldn't write filter file: %s", err)
		log.Println(errortext)
		http.Error(w, errortext, http.StatusInternalServerError)
		return
	}
	httpUpdateConfigReloadDNSReturnOK(w, r)
}

func handleFilteringRefresh(w http.ResponseWriter, r *http.Request) {
	force := r.URL.Query().Get("force")
	if force != "" {
		config.Lock()
		for i := range config.Filters {
			filter := &config.Filters[i] // otherwise we will be operating on a copy
			filter.LastUpdated = time.Unix(0, 0)
		}
		config.Unlock() // not defer because refreshFiltersIfNeccessary locks it too
	}
	updated := refreshFiltersIfNeccessary()
	fmt.Fprintf(w, "OK %d filters updated\n", updated)
}

func runFilterRefreshers() {
	go func() {
		for range time.Tick(time.Second) {
			refreshFiltersIfNeccessary()
		}
	}()
}

func refreshFiltersIfNeccessary() int {
	now := time.Now()
	config.Lock()

	// deduplicate
	// TODO: move it somewhere else
	{
		i := 0 // output index, used for deletion later
		urls := map[string]bool{}
		for _, filter := range config.Filters {
			if _, ok := urls[filter.URL]; !ok {
				// we didn't see it before, keep it
				urls[filter.URL] = true // remember the URL
				config.Filters[i] = filter
				i++
			}
		}
		// all entries we want to keep are at front, delete the rest
		config.Filters = config.Filters[:i]
	}

	// fetch URLs
	updateCount := 0
	for i := range config.Filters {
		filter := &config.Filters[i] // otherwise we will be operating on a copy
		updated, err := filter.update(now)
		if err != nil {
			log.Printf("Failed to update filter %s: %s\n", filter.URL, err)
			continue
		}
		if updated {
			updateCount++
		}
	}
	config.Unlock()

	if updateCount > 0 {
		err := writeFilterFile()
		if err != nil {
			errortext := fmt.Sprintf("Couldn't write filter file: %s", err)
			log.Println(errortext)
		}
		tellCoreDNSToReload()
	}
	return updateCount
}

func (filter *filter) update(now time.Time) (bool, error) {
	if !filter.Enabled {
		return false, nil
	}
	elapsed := time.Since(filter.LastUpdated)
	if elapsed <= updatePeriod {
		return false, nil
	}

	// use same update period for failed filter downloads to avoid flooding with requests
	filter.LastUpdated = now

	resp, err := client.Get(filter.URL)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		log.Printf("Couldn't request filter from URL %s, skipping: %s", filter.URL, err)
		return false, err
	}

	if resp.StatusCode >= 400 {
		log.Printf("Got status code %d from URL %s, skipping", resp.StatusCode, filter.URL)
		return false, fmt.Errorf("Got status code >= 400: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Couldn't fetch filter contents from URL %s, skipping: %s", filter.URL, err)
		return false, err
	}

	// extract filter name and count number of rules
	lines := strings.Split(string(body), "\n")
	rulesCount := 0
	seenTitle := false
	d := dnsfilter.New()
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 0 && line[0] == '!' {
			if m := filterTitle.FindAllStringSubmatch(line, -1); len(m) > 0 && len(m[0]) >= 2 && !seenTitle {
				filter.Name = m[0][1]
				seenTitle = true
			}
		} else if len(line) != 0 {
			err = d.AddRule(line, 0)
			if err == dnsfilter.ErrInvalidSyntax {
				continue
			}
			if err != nil {
				log.Printf("Couldn't add rule %s: %s", filter.URL, err)
				return false, err
			}
			rulesCount++
		}
	}
	if bytes.Equal(filter.contents, body) {
		return false, nil
	}
	log.Printf("Filter %s updated: %d bytes, %d rules", filter.URL, len(body), rulesCount)
	filter.RulesCount = rulesCount
	filter.contents = body
	return true, nil
}

// write filter file
func writeFilterFile() error {
	filterpath := filepath.Join(config.ourBinaryDir, config.CoreDNS.FilterFile)
	log.Printf("Writing filter file: %s", filterpath)
	// TODO: check if file contents have modified
	data := []byte{}
	config.RLock()
	filters := config.Filters
	for _, filter := range filters {
		if !filter.Enabled {
			continue
		}
		data = append(data, filter.contents...)
		data = append(data, '\n')
	}
	for _, rule := range config.UserRules {
		data = append(data, []byte(rule)...)
		data = append(data, '\n')
	}
	config.RUnlock()
	err := ioutil.WriteFile(filterpath+".tmp", data, 0644)
	if err != nil {
		log.Printf("Couldn't write filter file: %s", err)
		return err
	}

	err = os.Rename(filterpath+".tmp", filterpath)
	if err != nil {
		log.Printf("Couldn't rename filter file: %s", err)
		return err
	}
	return nil
}

// ------------
// safebrowsing
// ------------

func handleSafeBrowsingEnable(w http.ResponseWriter, r *http.Request) {
	config.CoreDNS.SafeBrowsingEnabled = true
	httpUpdateConfigReloadDNSReturnOK(w, r)
}

func handleSafeBrowsingDisable(w http.ResponseWriter, r *http.Request) {
	config.CoreDNS.SafeBrowsingEnabled = false
	httpUpdateConfigReloadDNSReturnOK(w, r)
}

func handleSafeBrowsingStatus(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"enabled": config.CoreDNS.SafeBrowsingEnabled,
	}
	jsonVal, err := json.Marshal(data)
	if err != nil {
		errortext := fmt.Sprintf("Unable to marshal status json: %s", err)
		log.Println(errortext)
		http.Error(w, errortext, 500)
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonVal)
	if err != nil {
		errortext := fmt.Sprintf("Unable to write response json: %s", err)
		log.Println(errortext)
		http.Error(w, errortext, 500)
		return
	}
}

// --------
// parental
// --------
func handleParentalEnable(w http.ResponseWriter, r *http.Request) {
	parameters, err := parseParametersFromBody(r.Body)
	if err != nil {
		errortext := fmt.Sprintf("failed to parse parameters from body: %s", err)
		log.Println(errortext)
		http.Error(w, errortext, 400)
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
	config.CoreDNS.ParentalSensitivity = i
	config.CoreDNS.ParentalEnabled = true
	httpUpdateConfigReloadDNSReturnOK(w, r)
}

func handleParentalDisable(w http.ResponseWriter, r *http.Request) {
	config.CoreDNS.ParentalEnabled = false
	httpUpdateConfigReloadDNSReturnOK(w, r)
}

func handleParentalStatus(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"enabled": config.CoreDNS.ParentalEnabled,
	}
	if config.CoreDNS.ParentalEnabled {
		data["sensitivity"] = config.CoreDNS.ParentalSensitivity
	}
	jsonVal, err := json.Marshal(data)
	if err != nil {
		errortext := fmt.Sprintf("Unable to marshal status json: %s", err)
		log.Println(errortext)
		http.Error(w, errortext, 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonVal)
	if err != nil {
		errortext := fmt.Sprintf("Unable to write response json: %s", err)
		log.Println(errortext)
		http.Error(w, errortext, 500)
		return
	}
}

// ------------
// safebrowsing
// ------------

func handleSafeSearchEnable(w http.ResponseWriter, r *http.Request) {
	config.CoreDNS.SafeSearchEnabled = true
	httpUpdateConfigReloadDNSReturnOK(w, r)
}

func handleSafeSearchDisable(w http.ResponseWriter, r *http.Request) {
	config.CoreDNS.SafeSearchEnabled = false
	httpUpdateConfigReloadDNSReturnOK(w, r)
}

func handleSafeSearchStatus(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"enabled": config.CoreDNS.SafeSearchEnabled,
	}
	jsonVal, err := json.Marshal(data)
	if err != nil {
		errortext := fmt.Sprintf("Unable to marshal status json: %s", err)
		log.Println(errortext)
		http.Error(w, errortext, 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonVal)
	if err != nil {
		errortext := fmt.Sprintf("Unable to write response json: %s", err)
		log.Println(errortext)
		http.Error(w, errortext, 500)
		return
	}
}

func registerControlHandlers() {
	http.HandleFunc("/control/status", optionalAuth(ensureGET(handleStatus)))
	http.HandleFunc("/control/enable_protection", optionalAuth(ensurePOST(handleProtectionEnable)))
	http.HandleFunc("/control/disable_protection", optionalAuth(ensurePOST(handleProtectionDisable)))
	http.HandleFunc("/control/querylog", optionalAuth(ensureGET(coredns_plugin.HandleQueryLog)))
	http.HandleFunc("/control/querylog_enable", optionalAuth(ensurePOST(handleQueryLogEnable)))
	http.HandleFunc("/control/querylog_disable", optionalAuth(ensurePOST(handleQueryLogDisable)))
	http.HandleFunc("/control/set_upstream_dns", optionalAuth(ensurePOST(handleSetUpstreamDNS)))
	http.HandleFunc("/control/test_upstream_dns", optionalAuth(ensurePOST(handleTestUpstreamDNS)))
	http.HandleFunc("/control/stats_top", optionalAuth(ensureGET(coredns_plugin.HandleStatsTop)))
	http.HandleFunc("/control/stats", optionalAuth(ensureGET(coredns_plugin.HandleStats)))
	http.HandleFunc("/control/stats_history", optionalAuth(ensureGET(coredns_plugin.HandleStatsHistory)))
	http.HandleFunc("/control/stats_reset", optionalAuth(ensurePOST(coredns_plugin.HandleStatsReset)))
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
}
