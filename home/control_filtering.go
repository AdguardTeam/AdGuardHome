package home

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/AdguardTeam/golibs/log"
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
	Name string `json:"name"`
	URL  string `json:"url"`
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
	// TODO: since we directly feed filters in-memory, revisit if writing configs is always necessary
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
		URL string `json:"url"`
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
	for _, filter := range config.Filters {
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
	config.Filters = newFilters
	config.Unlock()

	onConfigModified()
	enableFilters(true)

	// Note: the old files "filter.txt.old" aren't deleted - it's not really necessary,
	//  but will require the additional code to run after enableFilters() is finished: i.e. complicated
}

type filterURLJSON struct {
	URL     string `json:"url"`
	Enabled bool   `json:"enabled"`
}

func handleFilteringSetURL(w http.ResponseWriter, r *http.Request) {
	fj := filterURLJSON{}
	err := json.NewDecoder(r.Body).Decode(&fj)
	if err != nil {
		httpError(w, http.StatusBadRequest, "json decode: %s", err)
		return
	}

	if !IsValidURL(fj.URL) {
		http.Error(w, "invalid URL", http.StatusBadRequest)
		return
	}

	found := filterEnable(fj.URL, fj.Enabled)
	if !found {
		http.Error(w, "URL doesn't exist", http.StatusBadRequest)
		return
	}

	onConfigModified()
	enableFilters(true)
}

func handleFilteringSetRules(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		httpError(w, http.StatusBadRequest, "Failed to read request body: %s", err)
		return
	}

	config.UserRules = strings.Split(string(body), "\n")
	_ = writeAllConfigs()
	enableFilters(true)
}

func handleFilteringRefresh(w http.ResponseWriter, r *http.Request) {
	config.controlLock.Unlock()
	nUpdated, err := refreshFilters()
	config.controlLock.Lock()
	if err != nil {
		httpError(w, http.StatusInternalServerError, "%s", err)
		return
	}
	fmt.Fprintf(w, "OK %d filters updated\n", nUpdated)
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
	Enabled   bool         `json:"enabled"`
	Interval  uint32       `json:"interval"` // in hours
	Filters   []filterJSON `json:"filters"`
	UserRules []string     `json:"user_rules"`
}

// Get filtering configuration
func handleFilteringInfo(w http.ResponseWriter, r *http.Request) {
	resp := filteringConfig{}
	config.RLock()
	resp.Enabled = config.DNS.FilteringEnabled
	resp.Interval = config.DNS.FiltersUpdateIntervalHours
	for _, f := range config.Filters {
		fj := filterJSON{
			ID:         f.ID,
			Enabled:    f.Enabled,
			URL:        f.URL,
			Name:       f.Name,
			RulesCount: uint32(f.RulesCount),
		}

		if f.LastUpdated.Second() != 0 {
			fj.LastUpdated = f.LastUpdated.Format(time.RFC3339)
		}

		resp.Filters = append(resp.Filters, fj)
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

// RegisterFilteringHandlers - register handlers
func RegisterFilteringHandlers() {
	httpRegister(http.MethodGet, "/control/filtering_info", handleFilteringInfo)
	httpRegister(http.MethodPost, "/control/filtering_config", handleFilteringConfig)
	httpRegister(http.MethodPost, "/control/filtering/add_url", handleFilteringAddURL)
	httpRegister(http.MethodPost, "/control/filtering/remove_url", handleFilteringRemoveURL)
	httpRegister(http.MethodPost, "/control/filtering/set_url", handleFilteringSetURL)
	httpRegister(http.MethodPost, "/control/filtering/refresh", handleFilteringRefresh)
	httpRegister(http.MethodPost, "/control/filtering/set_rules", handleFilteringSetRules)
}

func checkFiltersUpdateIntervalHours(i uint32) bool {
	return i == 0 || i == 1 || i == 12 || i == 1*24 || i == 3*24 || i == 7*24
}
