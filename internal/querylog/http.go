package querylog

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghstrings"
	"github.com/AdguardTeam/golibs/jsonutil"
	"github.com/AdguardTeam/golibs/log"
	"golang.org/x/net/idna"
)

type qlogConfig struct {
	Enabled bool `json:"enabled"`
	// Use float64 here to support fractional numbers and not mess the API
	// users by changing the units.
	Interval          float64 `json:"interval"`
	AnonymizeClientIP bool    `json:"anonymize_client_ip"`
}

// Register web handlers
func (l *queryLog) initWeb() {
	l.conf.HTTPRegister(http.MethodGet, "/control/querylog", l.handleQueryLog)
	l.conf.HTTPRegister(http.MethodGet, "/control/querylog_info", l.handleQueryLogInfo)
	l.conf.HTTPRegister(http.MethodPost, "/control/querylog_clear", l.handleQueryLogClear)
	l.conf.HTTPRegister(http.MethodPost, "/control/querylog_config", l.handleQueryLogConfig)
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
	data := l.entriesToJSON(entries, oldest)

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
	resp.Interval = l.conf.RotationIvl.Hours() / 24
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

	ivl := time.Duration(24*d.Interval) * time.Hour
	if req.Exists("interval") && !checkInterval(ivl) {
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
		conf.RotationIvl = ivl
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

// parseSearchCriterion parses a search criterion from the query parameter.
func (l *queryLog) parseSearchCriterion(q url.Values, name string, ct criterionType) (
	ok bool,
	sc searchCriterion,
	err error,
) {
	val := q.Get(name)
	if val == "" {
		return false, sc, nil
	}

	strict := getDoubleQuotesEnclosedValue(&val)

	var asciiVal string
	switch ct {
	case ctTerm:
		// Decode lowercased value from punycode to make EqualFold and
		// friends work properly with IDNAs.
		//
		// TODO(e.burkov):  Make it work with parts of IDNAs somehow.
		loweredVal := strings.ToLower(val)
		if asciiVal, err = idna.ToASCII(loweredVal); err != nil {
			log.Debug("can't convert %q to ascii: %s", val, err)
		} else if asciiVal == loweredVal {
			// Purge asciiVal to prevent checking the same value
			// twice.
			asciiVal = ""
		}
	case ctFilteringStatus:
		if !aghstrings.InSlice(filteringStatusValues, val) {
			return false, sc, fmt.Errorf("invalid value %s", val)
		}
	default:
		return false, sc, fmt.Errorf(
			"invalid criterion type %v: should be one of %v",
			ct,
			[]criterionType{ctTerm, ctFilteringStatus},
		)
	}

	sc = searchCriterion{
		criterionType: ct,
		value:         val,
		asciiVal:      asciiVal,
		strict:        strict,
	}

	return true, sc, nil
}

// parseSearchParams - parses "searchParams" from the HTTP request's query string
func (l *queryLog) parseSearchParams(r *http.Request) (p *searchParams, err error) {
	p = newSearchParams()

	q := r.URL.Query()
	olderThan := q.Get("older_than")
	if len(olderThan) != 0 {
		p.olderThan, err = time.Parse(time.RFC3339Nano, olderThan)
		if err != nil {
			return nil, err
		}
	}

	var limit64 int64
	if limit64, err = strconv.ParseInt(q.Get("limit"), 10, 64); err == nil {
		p.limit = int(limit64)
	}

	var offset64 int64
	if offset64, err = strconv.ParseInt(q.Get("offset"), 10, 64); err == nil {
		p.offset = int(offset64)

		// If we don't use "olderThan" and use offset/limit instead, we should change the default behavior
		// and scan all log records until we found enough log entries
		p.maxFileScanEntries = 0
	}

	for _, v := range []struct {
		urlField string
		ct       criterionType
	}{{
		urlField: "search",
		ct:       ctTerm,
	}, {
		urlField: "response_status",
		ct:       ctFilteringStatus,
	}} {
		var ok bool
		var c searchCriterion
		ok, c, err = l.parseSearchCriterion(q, v.urlField, v.ct)
		if err != nil {
			return nil, err
		}

		if ok {
			p.searchCriteria = append(p.searchCriteria, c)
		}
	}

	return p, nil
}
