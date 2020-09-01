package querylog

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/AdguardTeam/AdGuardHome/util"

	"github.com/AdguardTeam/golibs/jsonutil"
	"github.com/AdguardTeam/golibs/log"
)

type qlogConfig struct {
	Enabled           bool   `json:"enabled"`
	Interval          uint32 `json:"interval"`
	AnonymizeClientIP bool   `json:"anonymize_client_ip"`
}

// Register web handlers
func (l *queryLog) initWeb() {
	l.conf.HTTPRegister("GET", "/control/querylog", l.handleQueryLog)
	l.conf.HTTPRegister("GET", "/control/querylog_info", l.handleQueryLogInfo)
	l.conf.HTTPRegister("POST", "/control/querylog_clear", l.handleQueryLogClear)
	l.conf.HTTPRegister("POST", "/control/querylog_config", l.handleQueryLogConfig)
}

func httpError(r *http.Request, w http.ResponseWriter, code int, format string, args ...interface{}) {
	text := fmt.Sprintf(format, args...)

	log.Info("QueryLog: %s %s: %s", r.Method, r.URL, text)

	http.Error(w, text, code)
}

func (l *queryLog) handleQueryLog(w http.ResponseWriter, r *http.Request) {
	params, err := l.parseSearchParams(r)
	if err != nil {
		httpError(r, w, http.StatusBadRequest, "failed to parse params: %s", err)
		return
	}

	// search for the log entries
	entries, oldest := l.search(params)

	// convert log entries to JSON
	var data = l.entriesToJSON(entries, oldest)

	jsonVal, err := json.Marshal(data)
	if err != nil {
		httpError(r, w, http.StatusInternalServerError, "Couldn't marshal data into json: %s", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonVal)
	if err != nil {
		httpError(r, w, http.StatusInternalServerError, "Unable to write response json: %s", err)
	}
}

func (l *queryLog) handleQueryLogClear(_ http.ResponseWriter, _ *http.Request) {
	l.clear()
}

// Get configuration
func (l *queryLog) handleQueryLogInfo(w http.ResponseWriter, r *http.Request) {
	resp := qlogConfig{}
	resp.Enabled = l.conf.Enabled
	resp.Interval = l.conf.Interval
	resp.AnonymizeClientIP = l.conf.AnonymizeClientIP

	jsonVal, err := json.Marshal(resp)
	if err != nil {
		httpError(r, w, http.StatusInternalServerError, "json encode: %s", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonVal)
	if err != nil {
		httpError(r, w, http.StatusInternalServerError, "http write: %s", err)
	}
}

// Set configuration
func (l *queryLog) handleQueryLogConfig(w http.ResponseWriter, r *http.Request) {
	d := qlogConfig{}
	req, err := jsonutil.DecodeObject(&d, r.Body)
	if err != nil {
		httpError(r, w, http.StatusBadRequest, "%s", err)
		return
	}

	if req.Exists("interval") && !checkInterval(d.Interval) {
		httpError(r, w, http.StatusBadRequest, "Unsupported interval")
		return
	}

	l.lock.Lock()
	// copy data, modify it, then activate.  Other threads (readers) don't need to use this lock.
	conf := *l.conf
	if req.Exists("enabled") {
		conf.Enabled = d.Enabled
	}
	if req.Exists("interval") {
		conf.Interval = d.Interval
	}
	if req.Exists("anonymize_client_ip") {
		conf.AnonymizeClientIP = d.AnonymizeClientIP
	}
	l.conf = &conf
	l.lock.Unlock()

	l.conf.ConfigModified()
}

// "value" -> value, return TRUE
func getDoubleQuotesEnclosedValue(s *string) bool {
	t := *s
	if len(t) >= 2 && t[0] == '"' && t[len(t)-1] == '"' {
		*s = t[1 : len(t)-1]
		return true
	}
	return false
}

// parseSearchCriteria - parses "searchCriteria" from the specified query parameter
func (l *queryLog) parseSearchCriteria(q url.Values, name string, ct criteriaType) (bool, searchCriteria, error) {
	val := q.Get(name)
	if len(val) == 0 {
		return false, searchCriteria{}, nil
	}

	c := searchCriteria{
		criteriaType: ct,
		value:        val,
	}
	if getDoubleQuotesEnclosedValue(&c.value) {
		c.strict = true
	}

	if ct == ctFilteringStatus && !util.ContainsString(filteringStatusValues, c.value) {
		return false, c, fmt.Errorf("invalid value %s", c.value)
	}

	return true, c, nil
}

// parseSearchParams - parses "searchParams" from the HTTP request's query string
func (l *queryLog) parseSearchParams(r *http.Request) (*searchParams, error) {
	p := newSearchParams()

	var err error
	q := r.URL.Query()
	olderThan := q.Get("older_than")
	if len(olderThan) != 0 {
		p.olderThan, err = time.Parse(time.RFC3339Nano, olderThan)
		if err != nil {
			return nil, err
		}
	}

	if limit, err := strconv.ParseInt(q.Get("limit"), 10, 64); err == nil {
		p.limit = int(limit)
	}
	if offset, err := strconv.ParseInt(q.Get("offset"), 10, 64); err == nil {
		p.offset = int(offset)

		// If we don't use "olderThan" and use offset/limit instead, we should change the default behavior
		// and scan all log records until we found enough log entries
		p.maxFileScanEntries = 0
	}

	paramNames := map[string]criteriaType{
		"search":          ctDomainOrClient,
		"response_status": ctFilteringStatus,
	}

	for k, v := range paramNames {
		ok, c, err := l.parseSearchCriteria(q, k, v)
		if err != nil {
			return nil, err
		}

		if ok {
			p.searchCriteria = append(p.searchCriteria, c)
		}
	}

	return p, nil
}
