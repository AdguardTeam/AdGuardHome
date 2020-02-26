package home

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/AdguardTeam/golibs/log"
	"github.com/miekg/dns"
)

// IsValidURL - return TRUE if URL is valid
func IsValidURL(rawurl string) bool {
	url, err := url.ParseRequestURI(rawurl)
	if err != nil {
		return false //Couldn't even parse the rawurl
	}
	if len(url.Scheme) == 0 {
		return false //No Scheme found
	}
	return true
}

type filterAddJSON struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	Whitelist bool   `json:"whitelist"`
}

func handleFilteringAddURL(w http.ResponseWriter, r *http.Request) {
	fj := filterAddJSON{}
	err := json.NewDecoder(r.Body).Decode(&fj)
	if err != nil {
		httpError(w, http.StatusBadRequest, "Failed to parse request body json: %s", err)
		return
	}

	if !IsValidURL(fj.URL) {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	// Check for duplicates
	if filterExists(fj.URL) {
		httpError(w, http.StatusBadRequest, "Filter URL already added -- %s", fj.URL)
		return
	}

	// Set necessary properties
	f := filter{
		Enabled: true,
		URL:     fj.URL,
		Name:    fj.Name,
		white:   fj.Whitelist,
	}
	f.ID = assignUniqueFilterID()

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
	if !filterAdd(f) {
		httpError(w, http.StatusBadRequest, "Filter URL already added -- %s", f.URL)
		return
	}

	onConfigModified()
	enableFilters(true)

	_, err = fmt.Fprintf(w, "OK %d rules\n", f.RulesCount)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Couldn't write body: %s", err)
	}
}

func handleFilteringRemoveURL(w http.ResponseWriter, r *http.Request) {

	type request struct {
		URL       string `json:"url"`
		Whitelist bool   `json:"whitelist"`
	}
	req := request{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		httpError(w, http.StatusBadRequest, "Failed to parse request body json: %s", err)
		return
	}

	if !IsValidURL(req.URL) {
		http.Error(w, "URL parameter is not valid request URL", http.StatusBadRequest)
		return
	}

	// go through each element and delete if url matches
	config.Lock()
	newFilters := []filter{}
	filters := &config.Filters
	if req.Whitelist {
		filters = &config.WhitelistFilters
	}
	for _, filter := range *filters {
		if filter.URL != req.URL {
			newFilters = append(newFilters, filter)
		} else {
			err := os.Rename(filter.Path(), filter.Path()+".old")
			if err != nil {
				log.Error("os.Rename: %s: %s", filter.Path(), err)
			}
		}
	}
	// Update the configuration after removing filter files
	*filters = newFilters
	config.Unlock()

	onConfigModified()
	enableFilters(true)

	// Note: the old files "filter.txt.old" aren't deleted - it's not really necessary,
	//  but will require the additional code to run after enableFilters() is finished: i.e. complicated
}

type filterURLJSON struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	Enabled bool   `json:"enabled"`
}

type filterURLReq struct {
	URL       string        `json:"url"`
	Whitelist bool          `json:"whitelist"`
	Data      filterURLJSON `json:"data"`
}

func handleFilteringSetURL(w http.ResponseWriter, r *http.Request) {
	fj := filterURLReq{}
	err := json.NewDecoder(r.Body).Decode(&fj)
	if err != nil {
		httpError(w, http.StatusBadRequest, "json decode: %s", err)
		return
	}

	if !IsValidURL(fj.URL) {
		http.Error(w, "invalid URL", http.StatusBadRequest)
		return
	}

	f := filter{
		Enabled: fj.Data.Enabled,
		Name:    fj.Data.Name,
		URL:     fj.Data.URL,
	}
	status := filterSetProperties(fj.URL, f, fj.Whitelist)
	if (status & statusFound) == 0 {
		http.Error(w, "URL doesn't exist", http.StatusBadRequest)
		return
	}
	if (status & statusURLExists) != 0 {
		http.Error(w, "URL already exists", http.StatusBadRequest)
		return
	}

	onConfigModified()
	if (status & statusURLChanged) != 0 {
		if fj.Data.Enabled {
			// download new filter and apply its rules
			refreshStatus = 1
			refreshLock.Lock()
			_, _ = refreshFiltersIfNecessary(true)
			refreshLock.Unlock()
		}

	} else if (status & statusEnabledChanged) != 0 {
		enableFilters(true)
	}
}

func handleFilteringSetRules(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		httpError(w, http.StatusBadRequest, "Failed to read request body: %s", err)
		return
	}

	config.UserRules = strings.Split(string(body), "\n")
	onConfigModified()
	userFilter := userFilter()
	err = userFilter.save()
	if err != nil {
		log.Error("Couldn't save the user filter: %s", err)
	}
	enableFilters(true)
}

func handleFilteringRefresh(w http.ResponseWriter, r *http.Request) {
	type Resp struct {
		Updated int `json:"updated"`
	}
	resp := Resp{}
	var err error

	Context.controlLock.Unlock()
	resp.Updated, err = refreshFilters()
	Context.controlLock.Lock()
	if err != nil {
		httpError(w, http.StatusInternalServerError, "%s", err)
		return
	}

	js, err := json.Marshal(resp)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "json encode: %s", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(js)
}

type filterJSON struct {
	ID          int64  `json:"id"`
	Enabled     bool   `json:"enabled"`
	URL         string `json:"url"`
	Name        string `json:"name"`
	RulesCount  uint32 `json:"rules_count"`
	LastUpdated string `json:"last_updated"`
}

type filteringConfig struct {
	Enabled          bool         `json:"enabled"`
	Interval         uint32       `json:"interval"` // in hours
	Filters          []filterJSON `json:"filters"`
	WhitelistFilters []filterJSON `json:"whitelist_filters"`
	UserRules        []string     `json:"user_rules"`
}

func filterToJSON(f filter) filterJSON {
	fj := filterJSON{
		ID:         f.ID,
		Enabled:    f.Enabled,
		URL:        f.URL,
		Name:       f.Name,
		RulesCount: uint32(f.RulesCount),
	}

	if !f.LastUpdated.IsZero() {
		fj.LastUpdated = f.LastUpdated.Format(time.RFC3339)
	}

	return fj
}

// Get filtering configuration
func handleFilteringStatus(w http.ResponseWriter, r *http.Request) {
	resp := filteringConfig{}
	config.RLock()
	resp.Enabled = config.DNS.FilteringEnabled
	resp.Interval = config.DNS.FiltersUpdateIntervalHours
	for _, f := range config.Filters {
		fj := filterToJSON(f)
		resp.Filters = append(resp.Filters, fj)
	}
	for _, f := range config.WhitelistFilters {
		fj := filterToJSON(f)
		resp.WhitelistFilters = append(resp.WhitelistFilters, fj)
	}
	resp.UserRules = config.UserRules
	config.RUnlock()

	jsonVal, err := json.Marshal(resp)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "json encode: %s", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonVal)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "http write: %s", err)
	}
}

// Set filtering configuration
func handleFilteringConfig(w http.ResponseWriter, r *http.Request) {
	req := filteringConfig{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		httpError(w, http.StatusBadRequest, "json decode: %s", err)
		return
	}

	if !checkFiltersUpdateIntervalHours(req.Interval) {
		httpError(w, http.StatusBadRequest, "Unsupported interval")
		return
	}

	config.DNS.FilteringEnabled = req.Enabled
	config.DNS.FiltersUpdateIntervalHours = req.Interval
	onConfigModified()
	enableFilters(true)
}

type checkHostResp struct {
	Reason   string `json:"reason"`
	FilterID int64  `json:"filter_id"`
	Rule     string `json:"rule"`

	// for FilteredBlockedService:
	SvcName string `json:"service_name"`

	// for ReasonRewrite:
	CanonName string   `json:"cname"`    // CNAME value
	IPList    []net.IP `json:"ip_addrs"` // list of IP addresses
}

func handleCheckHost(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	host := q.Get("name")

	setts := Context.dnsFilter.GetConfig()
	setts.FilteringEnabled = true
	ApplyBlockedServices(&setts, config.DNS.BlockedServices)
	result, err := Context.dnsFilter.CheckHost(host, dns.TypeA, &setts)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "couldn't apply filtering: %s: %s", host, err)
		return
	}

	resp := checkHostResp{}
	resp.Reason = result.Reason.String()
	resp.FilterID = result.FilterID
	resp.Rule = result.Rule
	resp.SvcName = result.ServiceName
	resp.CanonName = result.CanonName
	resp.IPList = result.IPList
	js, err := json.Marshal(resp)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "json encode: %s", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(js)
}

// RegisterFilteringHandlers - register handlers
func RegisterFilteringHandlers() {
	httpRegister("GET", "/control/filtering/status", handleFilteringStatus)
	httpRegister("POST", "/control/filtering/config", handleFilteringConfig)
	httpRegister("POST", "/control/filtering/add_url", handleFilteringAddURL)
	httpRegister("POST", "/control/filtering/remove_url", handleFilteringRemoveURL)
	httpRegister("POST", "/control/filtering/set_url", handleFilteringSetURL)
	httpRegister("POST", "/control/filtering/refresh", handleFilteringRefresh)
	httpRegister("POST", "/control/filtering/set_rules", handleFilteringSetRules)
	httpRegister("GET", "/control/filtering/check_host", handleCheckHost)
}

func checkFiltersUpdateIntervalHours(i uint32) bool {
	return i == 0 || i == 1 || i == 12 || i == 1*24 || i == 3*24 || i == 7*24
}
