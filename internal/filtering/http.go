package filtering

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/miekg/dns"
)

// validateFilterURL validates the filter list URL or file name.
func validateFilterURL(urlStr string) (err error) {
	if filepath.IsAbs(urlStr) {
		_, err = os.Stat(urlStr)
		if err != nil {
			return fmt.Errorf("checking filter file: %w", err)
		}

		return nil
	}

	url, err := url.ParseRequestURI(urlStr)
	if err != nil {
		return fmt.Errorf("checking filter url: %w", err)
	}

	if s := url.Scheme; s != aghhttp.SchemeHTTP && s != aghhttp.SchemeHTTPS {
		return fmt.Errorf("checking filter url: invalid scheme %q", s)
	}

	return nil
}

type filterAddJSON struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	Whitelist bool   `json:"whitelist"`
}

func (d *DNSFilter) handleFilteringAddURL(w http.ResponseWriter, r *http.Request) {
	fj := filterAddJSON{}
	err := json.NewDecoder(r.Body).Decode(&fj)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "Failed to parse request body json: %s", err)

		return
	}

	err = validateFilterURL(fj.URL)
	if err != nil {
		err = fmt.Errorf("invalid url: %s", err)
		aghhttp.Error(r, w, http.StatusBadRequest, "%s", err)

		return
	}

	// Check for duplicates
	if d.filterExists(fj.URL) {
		aghhttp.Error(r, w, http.StatusBadRequest, "Filter URL already added -- %s", fj.URL)

		return
	}

	// Set necessary properties
	filt := FilterYAML{
		Enabled: true,
		URL:     fj.URL,
		Name:    fj.Name,
		white:   fj.Whitelist,
	}
	filt.ID = assignUniqueFilterID()

	// Download the filter contents
	ok, err := d.update(&filt)
	if err != nil {
		aghhttp.Error(
			r,
			w,
			http.StatusBadRequest,
			"Couldn't fetch filter from url %s: %s",
			filt.URL,
			err,
		)

		return
	}

	if !ok {
		aghhttp.Error(
			r,
			w,
			http.StatusBadRequest,
			"Filter at the url %s is invalid (maybe it points to blank page?)",
			filt.URL,
		)

		return
	}

	// URL is assumed valid so append it to filters, update config, write new
	// file and reload it to engines.
	if !d.filterAdd(filt) {
		aghhttp.Error(r, w, http.StatusBadRequest, "Filter URL already added -- %s", filt.URL)

		return
	}

	d.ConfigModified()
	d.EnableFilters(true)

	_, err = fmt.Fprintf(w, "OK %d rules\n", filt.RulesCount)
	if err != nil {
		aghhttp.Error(r, w, http.StatusInternalServerError, "Couldn't write body: %s", err)
	}
}

func (d *DNSFilter) handleFilteringRemoveURL(w http.ResponseWriter, r *http.Request) {
	type request struct {
		URL       string `json:"url"`
		Whitelist bool   `json:"whitelist"`
	}

	req := request{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "failed to parse request body json: %s", err)

		return
	}

	d.filtersMu.Lock()
	filters := &d.Filters
	if req.Whitelist {
		filters = &d.WhitelistFilters
	}

	var deleted FilterYAML
	var newFilters []FilterYAML
	for _, flt := range *filters {
		if flt.URL != req.URL {
			newFilters = append(newFilters, flt)

			continue
		}

		deleted = flt
		path := flt.Path(d.DataDir)
		err = os.Rename(path, path+".old")
		if err != nil {
			log.Error("deleting filter %q: %s", path, err)
		}
	}

	*filters = newFilters
	d.filtersMu.Unlock()

	d.ConfigModified()
	d.EnableFilters(true)

	// NOTE: The old files "filter.txt.old" aren't deleted.  It's not really
	// necessary, but will require the additional complicated code to run
	// after enableFilters is done.
	//
	// TODO(a.garipov): Make sure the above comment is true.

	_, err = fmt.Fprintf(w, "OK %d rules\n", deleted.RulesCount)
	if err != nil {
		aghhttp.Error(r, w, http.StatusInternalServerError, "couldn't write body: %s", err)
	}
}

type filterURLReqData struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	Enabled bool   `json:"enabled"`
}

type filterURLReq struct {
	Data      *filterURLReqData `json:"data"`
	URL       string            `json:"url"`
	Whitelist bool              `json:"whitelist"`
}

func (d *DNSFilter) handleFilteringSetURL(w http.ResponseWriter, r *http.Request) {
	fj := filterURLReq{}
	err := json.NewDecoder(r.Body).Decode(&fj)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "decoding request: %s", err)

		return
	}

	if fj.Data == nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "%s", errors.Error("data is absent"))

		return
	}

	err = validateFilterURL(fj.Data.URL)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "invalid url: %s", err)

		return
	}

	filt := FilterYAML{
		Enabled: fj.Data.Enabled,
		Name:    fj.Data.Name,
		URL:     fj.Data.URL,
	}
	status := d.filterSetProperties(fj.URL, filt, fj.Whitelist)
	if (status & statusFound) == 0 {
		aghhttp.Error(r, w, http.StatusBadRequest, "URL doesn't exist")

		return
	}
	if (status & statusURLExists) != 0 {
		aghhttp.Error(r, w, http.StatusBadRequest, "URL already exists")

		return
	}

	d.ConfigModified()

	restart := (status & statusEnabledChanged) != 0
	if (status&statusUpdateRequired) != 0 && fj.Data.Enabled {
		// download new filter and apply its rules.
		nUpdated := d.refreshFilters(!fj.Whitelist, fj.Whitelist, false)
		// if at least 1 filter has been updated, refreshFilters() restarts the filtering automatically
		// if not - we restart the filtering ourselves
		restart = false
		if nUpdated == 0 {
			restart = true
		}
	}

	if restart {
		d.EnableFilters(true)
	}
}

// filteringRulesReq is the JSON structure for settings custom filtering rules.
type filteringRulesReq struct {
	Rules []string `json:"rules"`
}

func (d *DNSFilter) handleFilteringSetRules(w http.ResponseWriter, r *http.Request) {
	if aghhttp.WriteTextPlainDeprecated(w, r) {
		return
	}

	req := &filteringRulesReq{}
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "reading req: %s", err)

		return
	}

	d.UserRules = req.Rules
	d.ConfigModified()
	d.EnableFilters(true)
}

func (d *DNSFilter) handleFilteringRefresh(w http.ResponseWriter, r *http.Request) {
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
		aghhttp.Error(r, w, http.StatusBadRequest, "json decode: %s", err)

		return
	}

	var ok bool
	resp.Updated, _, ok = d.tryRefreshFilters(!req.White, req.White, true)
	if !ok {
		aghhttp.Error(
			r,
			w,
			http.StatusInternalServerError,
			"filters update procedure is already running",
		)

		return
	}

	_ = aghhttp.WriteJSONResponse(w, r, resp)
}

type filterJSON struct {
	URL         string `json:"url"`
	Name        string `json:"name"`
	LastUpdated string `json:"last_updated,omitempty"`
	ID          int64  `json:"id"`
	RulesCount  uint32 `json:"rules_count"`
	Enabled     bool   `json:"enabled"`
}

type filteringConfig struct {
	Filters          []filterJSON `json:"filters"`
	WhitelistFilters []filterJSON `json:"whitelist_filters"`
	UserRules        []string     `json:"user_rules"`
	Interval         uint32       `json:"interval"` // in hours
	Enabled          bool         `json:"enabled"`
}

func filterToJSON(f FilterYAML) filterJSON {
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
func (d *DNSFilter) handleFilteringStatus(w http.ResponseWriter, r *http.Request) {
	resp := filteringConfig{}
	d.filtersMu.RLock()
	resp.Enabled = d.FilteringEnabled
	resp.Interval = d.FiltersUpdateIntervalHours
	for _, f := range d.Filters {
		fj := filterToJSON(f)
		resp.Filters = append(resp.Filters, fj)
	}
	for _, f := range d.WhitelistFilters {
		fj := filterToJSON(f)
		resp.WhitelistFilters = append(resp.WhitelistFilters, fj)
	}
	resp.UserRules = d.UserRules
	d.filtersMu.RUnlock()

	_ = aghhttp.WriteJSONResponse(w, r, resp)
}

// Set filtering configuration
func (d *DNSFilter) handleFilteringConfig(w http.ResponseWriter, r *http.Request) {
	req := filteringConfig{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "json decode: %s", err)

		return
	}

	if !ValidateUpdateIvl(req.Interval) {
		aghhttp.Error(r, w, http.StatusBadRequest, "Unsupported interval")

		return
	}

	func() {
		d.filtersMu.Lock()
		defer d.filtersMu.Unlock()

		d.FilteringEnabled = req.Enabled
		d.FiltersUpdateIntervalHours = req.Interval
	}()

	d.ConfigModified()
	d.EnableFilters(true)
}

type checkHostRespRule struct {
	Text         string `json:"text"`
	FilterListID int64  `json:"filter_list_id"`
}

type checkHostResp struct {
	Reason string `json:"reason"`

	// Rule is the text of the matched rule.
	//
	// Deprecated: Use Rules[*].Text.
	Rule string `json:"rule"`

	Rules []*checkHostRespRule `json:"rules"`

	// for FilteredBlockedService:
	SvcName string `json:"service_name"`

	// for Rewrite:
	CanonName string   `json:"cname"`    // CNAME value
	IPList    []net.IP `json:"ip_addrs"` // list of IP addresses

	// FilterID is the ID of the rule's filter list.
	//
	// Deprecated: Use Rules[*].FilterListID.
	FilterID int64 `json:"filter_id"`
}

func (d *DNSFilter) handleCheckHost(w http.ResponseWriter, r *http.Request) {
	host := r.URL.Query().Get("name")

	setts := d.GetConfig()
	setts.FilteringEnabled = true
	setts.ProtectionEnabled = true

	d.ApplyBlockedServices(&setts, nil)
	result, err := d.CheckHost(host, dns.TypeA, &setts)
	if err != nil {
		aghhttp.Error(
			r,
			w,
			http.StatusInternalServerError,
			"couldn't apply filtering: %s: %s",
			host,
			err,
		)

		return
	}

	rulesLen := len(result.Rules)
	resp := checkHostResp{
		Reason:    result.Reason.String(),
		SvcName:   result.ServiceName,
		CanonName: result.CanonName,
		IPList:    result.IPList,
		Rules:     make([]*checkHostRespRule, len(result.Rules)),
	}

	if rulesLen > 0 {
		resp.FilterID = result.Rules[0].FilterListID
		resp.Rule = result.Rules[0].Text
	}

	for i, r := range result.Rules {
		resp.Rules[i] = &checkHostRespRule{
			FilterListID: r.FilterListID,
			Text:         r.Text,
		}
	}

	_ = aghhttp.WriteJSONResponse(w, r, resp)
}

// RegisterFilteringHandlers - register handlers
func (d *DNSFilter) RegisterFilteringHandlers() {
	registerHTTP := d.HTTPRegister
	if registerHTTP == nil {
		return
	}

	registerHTTP(http.MethodPost, "/control/safebrowsing/enable", d.handleSafeBrowsingEnable)
	registerHTTP(http.MethodPost, "/control/safebrowsing/disable", d.handleSafeBrowsingDisable)
	registerHTTP(http.MethodGet, "/control/safebrowsing/status", d.handleSafeBrowsingStatus)

	registerHTTP(http.MethodPost, "/control/parental/enable", d.handleParentalEnable)
	registerHTTP(http.MethodPost, "/control/parental/disable", d.handleParentalDisable)
	registerHTTP(http.MethodGet, "/control/parental/status", d.handleParentalStatus)

	registerHTTP(http.MethodPost, "/control/safesearch/enable", d.handleSafeSearchEnable)
	registerHTTP(http.MethodPost, "/control/safesearch/disable", d.handleSafeSearchDisable)
	registerHTTP(http.MethodGet, "/control/safesearch/status", d.handleSafeSearchStatus)

	registerHTTP(http.MethodGet, "/control/rewrite/list", d.handleRewriteList)
	registerHTTP(http.MethodPost, "/control/rewrite/add", d.handleRewriteAdd)
	registerHTTP(http.MethodPost, "/control/rewrite/delete", d.handleRewriteDelete)

	registerHTTP(http.MethodGet, "/control/blocked_services/services", d.handleBlockedServicesAvailableServices)
	registerHTTP(http.MethodGet, "/control/blocked_services/list", d.handleBlockedServicesList)
	registerHTTP(http.MethodPost, "/control/blocked_services/set", d.handleBlockedServicesSet)

	registerHTTP(http.MethodGet, "/control/filtering/status", d.handleFilteringStatus)
	registerHTTP(http.MethodPost, "/control/filtering/config", d.handleFilteringConfig)
	registerHTTP(http.MethodPost, "/control/filtering/add_url", d.handleFilteringAddURL)
	registerHTTP(http.MethodPost, "/control/filtering/remove_url", d.handleFilteringRemoveURL)
	registerHTTP(http.MethodPost, "/control/filtering/set_url", d.handleFilteringSetURL)
	registerHTTP(http.MethodPost, "/control/filtering/refresh", d.handleFilteringRefresh)
	registerHTTP(http.MethodPost, "/control/filtering/set_rules", d.handleFilteringSetRules)
	registerHTTP(http.MethodGet, "/control/filtering/check_host", d.handleCheckHost)
}

// ValidateUpdateIvl returns false if i is not a valid filters update interval.
func ValidateUpdateIvl(i uint32) bool {
	return i == 0 || i == 1 || i == 12 || i == 1*24 || i == 3*24 || i == 7*24
}
