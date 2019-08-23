package home

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/AdguardTeam/golibs/log"
	"github.com/asaskevich/govalidator"
)

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

	type request struct {
		URL string `json:"url"`
	}
	req := request{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		httpError(w, http.StatusBadRequest, "Failed to parse request body json: %s", err)
		return
	}

	if valid := govalidator.IsRequestURL(req.URL); !valid {
		http.Error(w, "URL parameter is not valid request URL", http.StatusBadRequest)
		return
	}

	// Stop DNS server:
	//  we close urlfilter object which in turn closes file descriptors to filter files.
	// Otherwise, Windows won't allow us to remove the file which is being currently used.
	_ = config.dnsServer.Stop()

	// go through each element and delete if url matches
	config.Lock()
	newFilters := config.Filters[:0]
	for _, filter := range config.Filters {
		if filter.URL != req.URL {
			newFilters = append(newFilters, filter)
		} else {
			// Remove the filter file
			err := os.Remove(filter.Path())
			if err != nil && !os.IsNotExist(err) {
				config.Unlock()
				httpError(w, http.StatusInternalServerError, "Couldn't remove the filter file: %s", err)
				return
			}
			log.Debug("os.Remove(%s)", filter.Path())
		}
	}
	// Update the configuration after removing filter files
	config.Filters = newFilters
	config.Unlock()
	httpUpdateConfigReloadDNSReturnOK(w, r)
}

func handleFilteringEnableURL(w http.ResponseWriter, r *http.Request) {
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
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		httpError(w, http.StatusBadRequest, "Failed to read request body: %s", err)
		return
	}

	config.UserRules = strings.Split(string(body), "\n")
	httpUpdateConfigReloadDNSReturnOK(w, r)
}

func handleFilteringRefresh(w http.ResponseWriter, r *http.Request) {
	updated := refreshFiltersIfNecessary(true)
	fmt.Fprintf(w, "OK %d filters updated\n", updated)
}
