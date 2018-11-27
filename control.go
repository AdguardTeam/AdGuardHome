package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/upstream"

	corednsplugin "github.com/AdguardTeam/AdGuardHome/coredns_plugin"
	"gopkg.in/asaskevich/govalidator.v4"
)

const updatePeriod = time.Minute * 30

var filterTitleRegexp = regexp.MustCompile(`^! Title: +(.*)$`)

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
	corednsplugin.Reload <- true
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
		"bootstrap_dns":      config.CoreDNS.BootstrapDNS,
		"upstream_dns":       config.CoreDNS.UpstreamDNS,
		"version":            VersionString,
		"language":           config.Language,
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
		errorText := fmt.Sprintf("Failed to read request body: %s", err)
		log.Println(errorText)
		http.Error(w, errorText, http.StatusBadRequest)
		return
	}
	// if empty body -- user is asking for default servers
	hosts := strings.Fields(string(body))

	if len(hosts) == 0 {
		config.CoreDNS.UpstreamDNS = defaultDNS
	} else {
		config.CoreDNS.UpstreamDNS = hosts
	}

	err = writeAllConfigs()
	if err != nil {
		errorText := fmt.Sprintf("Couldn't write config file: %s", err)
		log.Println(errorText)
		http.Error(w, errorText, http.StatusInternalServerError)
		return
	}
	tellCoreDNSToReload()
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
	u, err := upstream.NewUpstream(input, config.CoreDNS.BootstrapDNS)

	if err != nil {
		return err
	}
	defer u.Close()

	alive, err := upstream.IsAlive(u)

	if err != nil {
		return fmt.Errorf("couldn't communicate with DNS server %s: %s", input, err)
	}

	if !alive {
		return fmt.Errorf("DNS server has not passed the healthcheck: %s", input)
	}

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

	// URL is deemed valid, append it to filters, update config, write new filter file and tell coredns to reload it
	config.Filters = append(config.Filters, filter)
	err = writeAllConfigs()
	if err != nil {
		errorText := fmt.Sprintf("Couldn't write config file: %s", err)
		log.Println(errorText)
		http.Error(w, errorText, http.StatusInternalServerError)
		return
	}

	tellCoreDNSToReload()

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
			if err != nil {
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
	refreshFiltersIfNeccessary(false)
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
	updated := refreshFiltersIfNeccessary(force != "")
	fmt.Fprintf(w, "OK %d filters updated\n", updated)
}

// Sets up a timer that will be checking for filters updates periodically
func periodicallyRefreshFilters() {
	for range time.Tick(time.Minute) {
		refreshFiltersIfNeccessary(false)
	}
}

// Checks filters updates if necessary
// If force is true, it ignores the filter.LastUpdated field value
func refreshFiltersIfNeccessary(force bool) int {
	config.Lock()

	// fetch URLs
	updateCount := 0
	for i := range config.Filters {
		filter := &config.Filters[i] // otherwise we will be operating on a copy

		if filter.ID == 0 { // protect against users modifying the yaml and removing the ID
			filter.ID = assignUniqueFilterID()
		}

		updated, err := filter.update(force)
		if err != nil {
			log.Printf("Failed to update filter %s: %s\n", filter.URL, err)
			continue
		}
		if updated {
			// Saving it to the filters dir now
			err = filter.save()
			if err != nil {
				log.Printf("Failed to save the updated filter %d: %s", filter.ID, err)
				continue
			}

			updateCount++
		}
	}
	config.Unlock()

	if updateCount > 0 {
		tellCoreDNSToReload()
	}
	return updateCount
}

// A helper function that parses filter contents and returns a number of rules and a filter name (if there's any)
func parseFilterContents(contents []byte) (int, string) {
	lines := strings.Split(string(contents), "\n")
	rulesCount := 0
	name := ""
	seenTitle := false

	// Count lines in the filter
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 0 && line[0] == '!' {
			if m := filterTitleRegexp.FindAllStringSubmatch(line, -1); len(m) > 0 && len(m[0]) >= 2 && !seenTitle {
				name = m[0][1]
				seenTitle = true
			}
		} else if len(line) != 0 {
			rulesCount++
		}
	}

	return rulesCount, name
}

// Checks for filters updates
// If "force" is true -- does not check the filter's LastUpdated field
// Call "save" to persist the filter contents
func (filter *filter) update(force bool) (bool, error) {
	if filter.ID == 0 { // protect against users deleting the ID
		filter.ID = assignUniqueFilterID()
	}
	if !filter.Enabled {
		return false, nil
	}
	if !force && time.Since(filter.LastUpdated) <= updatePeriod {
		return false, nil
	}

	log.Printf("Downloading update for filter %d from %s", filter.ID, filter.URL)

	// use the same update period for failed filter downloads to avoid flooding with requests
	filter.LastUpdated = time.Now()

	resp, err := client.Get(filter.URL)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		log.Printf("Couldn't request filter from URL %s, skipping: %s", filter.URL, err)
		return false, err
	}

	if resp.StatusCode != 200 {
		log.Printf("Got status code %d from URL %s, skipping", resp.StatusCode, filter.URL)
		return false, fmt.Errorf("got status code != 200: %d", resp.StatusCode)
	}

	contentType := strings.ToLower(resp.Header.Get("content-type"))
	if !strings.HasPrefix(contentType, "text/plain") {
		log.Printf("Non-text response %s from %s, skipping", contentType, filter.URL)
		return false, fmt.Errorf("non-text response %s", contentType)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Couldn't fetch filter contents from URL %s, skipping: %s", filter.URL, err)
		return false, err
	}

	// Extract filter name and count number of rules
	rulesCount, filterName := parseFilterContents(body)

	if filterName != "" {
		filter.Name = filterName
	}

	// Check if the filter has been really changed
	if bytes.Equal(filter.Contents, body) {
		log.Printf("The filter %d text has not changed", filter.ID)
		return false, nil
	}

	log.Printf("Filter %d has been updated: %d bytes, %d rules", filter.ID, len(body), rulesCount)
	filter.RulesCount = rulesCount
	filter.Contents = body

	return true, nil
}

// saves filter contents to the file in dataDir
func (filter *filter) save() error {
	filterFilePath := filter.Path()
	log.Printf("Saving filter %d contents to: %s", filter.ID, filterFilePath)

	err := safeWriteFile(filterFilePath, filter.Contents)
	if err != nil {
		return err
	}

	return nil
}

// loads filter contents from the file in dataDir
func (filter *filter) load() error {
	if !filter.Enabled {
		// No need to load a filter that is not enabled
		return nil
	}

	filterFilePath := filter.Path()
	log.Printf("Loading filter %d contents to: %s", filter.ID, filterFilePath)

	if _, err := os.Stat(filterFilePath); os.IsNotExist(err) {
		// do nothing, file doesn't exist
		return err
	}

	filterFileContents, err := ioutil.ReadFile(filterFilePath)
	if err != nil {
		return err
	}

	log.Printf("Filter %d length is %d", filter.ID, len(filterFileContents))
	filter.Contents = filterFileContents

	// Now extract the rules count
	rulesCount, _ := parseFilterContents(filter.Contents)
	filter.RulesCount = rulesCount

	return nil
}

// Path to the filter contents
func (filter *filter) Path() string {
	return filepath.Join(config.ourBinaryDir, dataDir, filterDir, strconv.FormatInt(filter.ID, 10)+".txt")
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
	http.HandleFunc("/control/querylog", optionalAuth(ensureGET(corednsplugin.HandleQueryLog)))
	http.HandleFunc("/control/querylog_enable", optionalAuth(ensurePOST(handleQueryLogEnable)))
	http.HandleFunc("/control/querylog_disable", optionalAuth(ensurePOST(handleQueryLogDisable)))
	http.HandleFunc("/control/set_upstream_dns", optionalAuth(ensurePOST(handleSetUpstreamDNS)))
	http.HandleFunc("/control/test_upstream_dns", optionalAuth(ensurePOST(handleTestUpstreamDNS)))
	http.HandleFunc("/control/i18n/change_language", optionalAuth(ensurePOST(handleI18nChangeLanguage)))
	http.HandleFunc("/control/i18n/current_language", optionalAuth(ensureGET(handleI18nCurrentLanguage)))
	http.HandleFunc("/control/stats_top", optionalAuth(ensureGET(corednsplugin.HandleStatsTop)))
	http.HandleFunc("/control/stats", optionalAuth(ensureGET(corednsplugin.HandleStats)))
	http.HandleFunc("/control/stats_history", optionalAuth(ensureGET(corednsplugin.HandleStatsHistory)))
	http.HandleFunc("/control/stats_reset", optionalAuth(ensurePOST(corednsplugin.HandleStatsReset)))
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
