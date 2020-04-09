package home

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/util"
	"github.com/AdguardTeam/golibs/log"
	"github.com/miekg/dns"
)

// IsValidURL - return TRUE if URL or file path is valid
func IsValidURL(rawurl string) bool {
	if filepath.IsAbs(rawurl) {
		// this is a file path
		return util.FileExists(rawurl)
	}

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

func (f *Filtering) handleFilteringAddURL(w http.ResponseWriter, r *http.Request) {
	fj := filterAddJSON{}
	err := json.NewDecoder(r.Body).Decode(&fj)
	if err != nil {
		httpError(w, http.StatusBadRequest, "Failed to parse request body json: %s", err)
		return
	}

	if !IsValidURL(fj.URL) {
		http.Error(w, "Invalid URL or file path", http.StatusBadRequest)
		return
	}

	// Check for duplicates
	if filterExists(fj.URL) {
		httpError(w, http.StatusBadRequest, "Filter URL already added -- %s", fj.URL)
		return
	}

	// Set necessary properties
	filt := filter{
		Enabled: true,
		URL:     fj.URL,
		Name:    fj.Name,
		white:   fj.Whitelist,
	}
	filt.ID = assignUniqueFilterID()

	// Download the filter contents
	ok, err := f.update(&filt)
	if err != nil {
		httpError(w, http.StatusBadRequest, "Couldn't fetch filter from url %s: %s", filt.URL, err)
		return
	}
	if !ok {
		httpError(w, http.StatusBadRequest, "Filter at the url %s is invalid (maybe it points to blank page?)", filt.URL)
		return
	}

	// URL is deemed valid, append it to filters, update config, write new filter file and tell dns to reload it
	if !filterAdd(filt) {
		httpError(w, http.StatusBadRequest, "Filter URL already added -- %s", filt.URL)
		return
	}

	onConfigModified()
	enableFilters(true)

	_, err = fmt.Fprintf(w, "OK %d rules\n", filt.RulesCount)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Couldn't write body: %s", err)
	}
}

func (f *Filtering) handleFilteringRemoveURL(w http.ResponseWriter, r *http.Request) {

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

func (f *Filtering) handleFilteringSetURL(w http.ResponseWriter, r *http.Request) {
	fj := filterURLReq{}
	err := json.NewDecoder(r.Body).Decode(&fj)
	if err != nil {
		httpError(w, http.StatusBadRequest, "json decode: %s", err)
		return
	}

	if !IsValidURL(fj.URL) {
		http.Error(w, "invalid URL or file path", http.StatusBadRequest)
		return
	}

	filt := filter{
		Enabled: fj.Data.Enabled,
		Name:    fj.Data.Name,
		URL:     fj.Data.URL,
	}
	status := f.filterSetProperties(fj.URL, filt, fj.Whitelist)
	if (status & statusFound) == 0 {
		http.Error(w, "URL doesn't exist", http.StatusBadRequest)
		return
	}
	if (status & statusURLExists) != 0 {
		http.Error(w, "URL already exists", http.StatusBadRequest)
		return
	}

	onConfigModified()
	restart := false
	if (status & statusEnabledChanged) != 0 {
		// we must add or remove filter rules
		restart = true
	}
	if (status&statusUpdateRequired) != 0 && fj.Data.Enabled {
		// download new filter and apply its rules
		flags := FilterRefreshBlocklists
		if fj.Whitelist {
			flags = FilterRefreshAllowlists
		}
		nUpdated, _ := f.refreshFilters(flags, true)
		// if at least 1 filter has been updated, refreshFilters() restarts the filtering automatically
		// if not - we restart the filtering ourselves
		restart = false
		if nUpdated == 0 {
			restart = true
		}
	}
	if restart {
		enableFilters(true)
	}
}

func (f *Filtering) handleFilteringSetRules(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		httpError(w, http.StatusBadRequest, "Failed to read request body: %s", err)
		return
	}

	config.UserRules = strings.Split(string(body), "\n")
	onConfigModified()
	enableFilters(true)
}

func (f *Filtering) handleFilteringRefresh(w http.ResponseWriter, r *http.Request) {
	type Req struct {
		White bool `json:"whitelist"`
	}
	type Resp struct {
		Updated int `json:"updated"`
	}
	resp := Resp{}
	var err error

	req := Req{}
	err = json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		httpError(w, http.StatusBadRequest, "json decode: %s", err)
		return
	}

	Context.controlLock.Unlock()
	flags := FilterRefreshBlocklists
	if req.White {
		flags = FilterRefreshAllowlists
	}
	resp.Updated, err = f.refreshFilters(flags|FilterRefreshForce, false)
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
func (f *Filtering) handleFilteringStatus(w http.ResponseWriter, r *http.Request) {
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
func (f *Filtering) handleFilteringConfig(w http.ResponseWriter, r *http.Request) {
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

func (f *Filtering) handleCheckHost(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	host := q.Get("name")

	setts := Context.dnsFilter.GetConfig()
	setts.FilteringEnabled = true
	Context.dnsFilter.ApplyBlockedServices(&setts, nil, true)
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
func (f *Filtering) RegisterFilteringHandlers() {
	httpRegister("GET", "/control/filtering/status", f.handleFilteringStatus)
	httpRegister("POST", "/control/filtering/config", f.handleFilteringConfig)
	httpRegister("POST", "/control/filtering/add_url", f.handleFilteringAddURL)
	httpRegister("POST", "/control/filtering/remove_url", f.handleFilteringRemoveURL)
	httpRegister("POST", "/control/filtering/set_url", f.handleFilteringSetURL)
	httpRegister("POST", "/control/filtering/refresh", f.handleFilteringRefresh)
	httpRegister("POST", "/control/filtering/set_rules", f.handleFilteringSetRules)
	httpRegister("GET", "/control/filtering/check_host", f.handleCheckHost)
}

func checkFiltersUpdateIntervalHours(i uint32) bool {
	return i == 0 || i == 1 || i == 12 || i == 1*24 || i == 3*24 || i == 7*24
}
