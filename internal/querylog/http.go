package querylog

import (
	"encoding/json"
	"fmt"
	"math"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghalg"
	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/stringutil"
	"github.com/AdguardTeam/golibs/timeutil"
	"golang.org/x/net/idna"
)

// configJSON is the JSON structure for the querylog configuration.
type configJSON struct {
	// Interval is the querylog rotation interval.  Use float64 here to support
	// fractional numbers and not mess the API users by changing the units.
	Interval float64 `json:"interval"`

	// Enabled shows if the querylog is enabled.  It is an [aghalg.NullBool]
	// to be able to tell when it's set without using pointers.
	Enabled aghalg.NullBool `json:"enabled"`

	// AnonymizeClientIP shows if the clients' IP addresses must be anonymized.
	// It is an [aghalg.NullBool] to be able to tell when it's set without using
	// pointers.
	AnonymizeClientIP aghalg.NullBool `json:"anonymize_client_ip"`
}

// Register web handlers
func (l *queryLog) initWeb() {
	l.conf.HTTPRegister(http.MethodGet, "/control/querylog", l.handleQueryLog)
	l.conf.HTTPRegister(http.MethodGet, "/control/querylog_info", l.handleQueryLogInfo)
	l.conf.HTTPRegister(http.MethodPost, "/control/querylog_clear", l.handleQueryLogClear)
	l.conf.HTTPRegister(http.MethodPost, "/control/querylog_config", l.handleQueryLogConfig)
}

func (l *queryLog) handleQueryLog(w http.ResponseWriter, r *http.Request) {
	l.lock.Lock()
	defer l.lock.Unlock()

	params, err := l.parseSearchParams(r)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "failed to parse params: %s", err)

		return
	}

	// search for the log entries
	entries, oldest := l.search(params)

	// convert log entries to JSON
	data := l.entriesToJSON(entries, oldest)

	_ = aghhttp.WriteJSONResponse(w, r, data)
}

func (l *queryLog) handleQueryLogClear(_ http.ResponseWriter, _ *http.Request) {
	l.clear()
}

// Get configuration
func (l *queryLog) handleQueryLogInfo(w http.ResponseWriter, r *http.Request) {
	_ = aghhttp.WriteJSONResponse(w, r, configJSON{
		Enabled:           aghalg.BoolToNullBool(l.conf.Enabled),
		Interval:          l.conf.RotationIvl.Hours() / 24,
		AnonymizeClientIP: aghalg.BoolToNullBool(l.conf.AnonymizeClientIP),
	})
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

// handleQueryLogConfig handles the POST /control/querylog_config queries.
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

	l.lock.Lock()
	defer l.lock.Unlock()

	// Copy data, modify it, then activate.  Other threads (readers) don't need
	// to use this lock.
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
		if !stringutil.InSlice(filteringStatusValues, val) {
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
