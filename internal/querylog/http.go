package querylog

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghalg"
	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/timeutil"
	"golang.org/x/net/idna"
)

// configJSON is the JSON structure for the querylog configuration.
type configJSON struct {
	// Interval is the querylog rotation interval.  Use float64 here to support
	// fractional numbers and not mess the API users by changing the units.
	Interval float64 `json:"interval"`

	// Enabled shows if the querylog is enabled.  It is an aghalg.NullBool to
	// be able to tell when it's set without using pointers.
	Enabled aghalg.NullBool `json:"enabled"`

	// AnonymizeClientIP shows if the clients' IP addresses must be anonymized.
	// It is an [aghalg.NullBool] to be able to tell when it's set without using
	// pointers.
	AnonymizeClientIP aghalg.NullBool `json:"anonymize_client_ip"`
}

// getConfigResp is the JSON structure for the querylog configuration.
type getConfigResp struct {
	// Ignored is the list of host names, which should not be written to log.
	Ignored []string `json:"ignored"`

	// Interval is the querylog rotation interval in milliseconds.
	Interval float64 `json:"interval"`

	// Enabled shows if the querylog is enabled.  It is an aghalg.NullBool to
	// be able to tell when it's set without using pointers.
	Enabled aghalg.NullBool `json:"enabled"`

	// AnonymizeClientIP shows if the clients' IP addresses must be anonymized.
	// It is an aghalg.NullBool to be able to tell when it's set without using
	// pointers.
	//
	// TODO(a.garipov): Consider using separate setting for statistics.
	AnonymizeClientIP aghalg.NullBool `json:"anonymize_client_ip"`
}

// Register web handlers
func (l *queryLog) initWeb() {
	l.conf.HTTPRegister(http.MethodGet, "/control/querylog", l.handleQueryLog)
	l.conf.HTTPRegister(http.MethodPost, "/control/querylog_clear", l.handleQueryLogClear)
	l.conf.HTTPRegister(http.MethodGet, "/control/querylog/config", l.handleGetQueryLogConfig)
	l.conf.HTTPRegister(
		http.MethodPut,
		"/control/querylog/config/update",
		l.handlePutQueryLogConfig,
	)

	// Deprecated handlers.
	l.conf.HTTPRegister(http.MethodGet, "/control/querylog_info", l.handleQueryLogInfo)
	l.conf.HTTPRegister(http.MethodPost, "/control/querylog_config", l.handleQueryLogConfig)
}

// handleQueryLog is the handler for the GET /control/querylog HTTP API.
func (l *queryLog) handleQueryLog(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	params, err := l.parseSearchParams(ctx, r)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "parsing params: %s", err)

		return
	}

	var entries []*logEntry
	var oldest time.Time
	func() {
		l.confMu.RLock()
		defer l.confMu.RUnlock()

		entries, oldest = l.search(ctx, params)
	}()

	resp := l.entriesToJSON(ctx, entries, oldest, l.anonymizer.Load())

	aghhttp.WriteJSONResponseOK(w, r, resp)
}

// handleQueryLogClear is the handler for the POST /control/querylog/clear HTTP
// API.
func (l *queryLog) handleQueryLogClear(_ http.ResponseWriter, r *http.Request) {
	l.clear(r.Context())
}

// handleQueryLogInfo is the handler for the GET /control/querylog_info HTTP
// API.
//
// Deprecated:  Remove it when migration to the new API is over.
func (l *queryLog) handleQueryLogInfo(w http.ResponseWriter, r *http.Request) {
	l.confMu.RLock()
	defer l.confMu.RUnlock()

	ivl := l.conf.RotationIvl

	if !checkInterval(ivl) {
		// NOTE: If interval is custom we set it to 90 days for compatibility
		// with old API.
		ivl = timeutil.Day * 90
	}

	aghhttp.WriteJSONResponseOK(w, r, configJSON{
		Enabled:           aghalg.BoolToNullBool(l.conf.Enabled),
		Interval:          ivl.Hours() / 24,
		AnonymizeClientIP: aghalg.BoolToNullBool(l.conf.AnonymizeClientIP),
	})
}

// handleGetQueryLogConfig is the handler for the GET /control/querylog/config
// HTTP API.
func (l *queryLog) handleGetQueryLogConfig(w http.ResponseWriter, r *http.Request) {
	var resp *getConfigResp
	func() {
		l.confMu.RLock()
		defer l.confMu.RUnlock()

		resp = &getConfigResp{
			Interval:          float64(l.conf.RotationIvl.Milliseconds()),
			Enabled:           aghalg.BoolToNullBool(l.conf.Enabled),
			AnonymizeClientIP: aghalg.BoolToNullBool(l.conf.AnonymizeClientIP),
			Ignored:           l.conf.Ignored.Values(),
		}
	}()

	aghhttp.WriteJSONResponseOK(w, r, resp)
}

// AnonymizeIP masks ip to anonymize the client if the ip is a valid one.
func AnonymizeIP(ip net.IP) {
	// zeroes is a slice of zero bytes from which the IP address tail is copied.
	// Using constant string as source of copying is more efficient than byte
	// slice, see https://github.com/golang/go/issues/49997.
	const zeroes = "\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00"

	if ip4 := ip.To4(); ip4 != nil {
		copy(ip4[net.IPv4len-2:net.IPv4len], zeroes)
	} else if len(ip) == net.IPv6len {
		copy(ip[net.IPv6len-10:net.IPv6len], zeroes)
	}
}

// handleQueryLogConfig is the handler for the POST /control/querylog_config
// HTTP API.
//
// Deprecated:  Remove it when migration to the new API is over.
func (l *queryLog) handleQueryLogConfig(w http.ResponseWriter, r *http.Request) {
	// Set NaN as initial value to be able to know if it changed later by
	// comparing it to NaN.
	newConf := &configJSON{
		Interval: math.NaN(),
	}

	err := json.NewDecoder(r.Body).Decode(newConf)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "%s", err)

		return
	}

	ivl := time.Duration(float64(timeutil.Day) * newConf.Interval)

	hasIvl := !math.IsNaN(newConf.Interval)
	if hasIvl && !checkInterval(ivl) {
		aghhttp.Error(r, w, http.StatusBadRequest, "unsupported interval")

		return
	}

	defer l.conf.ConfigModified()

	l.confMu.Lock()
	defer l.confMu.Unlock()

	conf := *l.conf
	if newConf.Enabled != aghalg.NBNull {
		conf.Enabled = newConf.Enabled == aghalg.NBTrue
	}

	if hasIvl {
		conf.RotationIvl = ivl
	}

	if newConf.AnonymizeClientIP != aghalg.NBNull {
		conf.AnonymizeClientIP = newConf.AnonymizeClientIP == aghalg.NBTrue
		if conf.AnonymizeClientIP {
			l.anonymizer.Store(AnonymizeIP)
		} else {
			l.anonymizer.Store(nil)
		}
	}

	l.conf = &conf
}

// handlePutQueryLogConfig is the handler for the PUT
// /control/querylog/config/update HTTP API.
func (l *queryLog) handlePutQueryLogConfig(w http.ResponseWriter, r *http.Request) {
	newConf := &getConfigResp{}
	err := json.NewDecoder(r.Body).Decode(newConf)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "%s", err)

		return
	}

	engine, err := aghnet.NewIgnoreEngine(newConf.Ignored)
	if err != nil {
		aghhttp.Error(r, w, http.StatusUnprocessableEntity, "ignored: %s", err)

		return
	}

	ivl := time.Duration(newConf.Interval) * time.Millisecond
	err = validateIvl(ivl)
	if err != nil {
		aghhttp.Error(r, w, http.StatusUnprocessableEntity, "unsupported interval: %s", err)

		return
	}

	if newConf.Enabled == aghalg.NBNull {
		aghhttp.Error(r, w, http.StatusUnprocessableEntity, "enabled is null")

		return
	}

	if newConf.AnonymizeClientIP == aghalg.NBNull {
		aghhttp.Error(r, w, http.StatusUnprocessableEntity, "anonymize_client_ip is null")

		return
	}

	defer l.conf.ConfigModified()

	l.confMu.Lock()
	defer l.confMu.Unlock()

	conf := *l.conf

	conf.Ignored = engine
	conf.RotationIvl = ivl
	conf.Enabled = newConf.Enabled == aghalg.NBTrue

	conf.AnonymizeClientIP = newConf.AnonymizeClientIP == aghalg.NBTrue
	if conf.AnonymizeClientIP {
		l.anonymizer.Store(AnonymizeIP)
	} else {
		l.anonymizer.Store(nil)
	}

	l.conf = &conf
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
func (l *queryLog) parseSearchCriterion(
	ctx context.Context,
	q url.Values,
	name string,
	ct criterionType,
) (ok bool, sc searchCriterion, err error) {
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
			l.logger.DebugContext(ctx, "converting  to ascii", "value", val, slogutil.KeyError, err)
		} else if asciiVal == loweredVal {
			// Purge asciiVal to prevent checking the same value
			// twice.
			asciiVal = ""
		}
	case ctFilteringStatus:
		if !slices.Contains(filteringStatusValues, val) {
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

// parseSearchParams parses search parameters from the HTTP request's query
// string.
func (l *queryLog) parseSearchParams(
	ctx context.Context,
	r *http.Request,
) (p *searchParams, err error) {
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
		ok, c, err = l.parseSearchCriterion(ctx, q, v.urlField, v.ct)
		if err != nil {
			return nil, err
		}

		if ok {
			p.searchCriteria = append(p.searchCriteria, c)
		}
	}

	return p, nil
}
