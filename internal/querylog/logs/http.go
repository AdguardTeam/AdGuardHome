package logs

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/stringutil"
	"github.com/AdguardTeam/golibs/timeutil"
	"golang.org/x/net/idna"
)

func RegisterHTTP(api Api, fn aghhttp.RegisterFunc) {
	fn(http.MethodGet, "/control/querylog", handleHttpQuery(api))
	fn(http.MethodGet, "/control/querylog_info", handleHttpInfo(api))
	fn(http.MethodPost, "/control/querylog_clear", handleHttpClear(api))
	fn(http.MethodPost, "/control/querylog_config", handleHttpConfig(api))
}

func handleHttpInfo(api Api) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_ = aghhttp.WriteJSONResponse(w, r, api.ConfigInfo())
	}
}

func handleHttpClear(api Api) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		api.Clear()
	}
}

func handleHttpConfig(api Api) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		newConf := &ConfigPayload{
			Interval: math.NaN(),
		}
		err := json.NewDecoder(r.Body).Decode(newConf)
		if err != nil {
			aghhttp.Error(r, w, http.StatusBadRequest, "%s", err)

			return
		}
		api.ApplyConfig(newConf)
	}
}
func handleHttpQuery(api Api) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		params, err := ParseSearchParams(r)
		if err != nil {
			aghhttp.Error(r, w, http.StatusBadRequest, "failed to parse params: %s", err)

			return
		}
		// search for the log entries
		data := api.Search(params)
		_ = aghhttp.WriteJSONResponse(w, r, data)
	}
}

func ParseSearchParams(r *http.Request) (p *SearchParams, err error) {
	p = newSearchParams()

	q := r.URL.Query()
	olderThan := q.Get("older_than")
	if len(olderThan) != 0 {
		p.OlderThan, err = time.Parse(time.RFC3339Nano, olderThan)
		if err != nil {
			return nil, err
		}
	}

	var limit64 int64
	if limit64, err = strconv.ParseInt(q.Get("limit"), 10, 64); err == nil {
		p.Limit = int(limit64)
	}

	var offset64 int64
	if offset64, err = strconv.ParseInt(q.Get("offset"), 10, 64); err == nil {
		p.Offset = int(offset64)

		// If we don't use "olderThan" and use offset/limit instead, we should change the default behavior
		// and scan all log records until we found enough log entries
		p.MaxFileScanEntries = 0
	}

	for _, v := range []struct {
		urlField string
		ct       CriterionType
	}{{
		urlField: "search",
		ct:       CtTerm,
	}, {
		urlField: "response_status",
		ct:       CtFilteringStatus,
	}} {
		var ok bool
		var c SearchCriterion
		ok, c, err = ParseSearchCriterion(q, v.urlField, v.ct)
		if err != nil {
			return nil, err
		}

		if ok {
			p.SearchCriteria = append(p.SearchCriteria, c)
		}
	}

	return p, nil
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

func ParseSearchCriterion(q url.Values, name string, ct CriterionType) (
	ok bool,
	sc SearchCriterion,
	err error,
) {
	val := q.Get(name)
	if val == "" {
		return false, sc, nil
	}

	strict := getDoubleQuotesEnclosedValue(&val)

	var asciiVal string
	switch ct {
	case CtTerm:
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
	case CtFilteringStatus:
		if !stringutil.InSlice(filteringStatusValues, val) {
			return false, sc, fmt.Errorf("invalid value %s", val)
		}
	default:
		return false, sc, fmt.Errorf(
			"invalid criterion type %v: should be one of %v",
			ct,
			[]CriterionType{CtTerm, CtFilteringStatus},
		)
	}

	sc = SearchCriterion{
		CriterionType: ct,
		Value:         val,
		AsciiVal:      asciiVal,
		Strict:        strict,
	}

	return true, sc, nil
}

func CheckInterval(ivl time.Duration) (ok bool) {
	// The constants for possible values of query log's rotation interval.
	const (
		quarterDay  = timeutil.Day / 4
		day         = timeutil.Day
		week        = timeutil.Day * 7
		month       = timeutil.Day * 30
		threeMonths = timeutil.Day * 90
	)

	return ivl == quarterDay || ivl == day || ivl == week || ivl == month || ivl == threeMonths
}
