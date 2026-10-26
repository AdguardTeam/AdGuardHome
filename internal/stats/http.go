// HTTP request handlers for accessing statistics data and configuration settings

package stats

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghalg"
	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/golibs/validate"
)

// queryKeyRecent is the key of the query parameter that contains the lookback
// interval for statistics.
const queryKeyRecent = "recent"

// millisecondsInHour contains number of milliseconds in one hour.
const millisecondsInHour = int64(time.Hour / time.Millisecond)

// topAddrs is an alias for the types of the TopFoo fields of statsResponse.
// The key is either a client's address or a requested address.
type topAddrs = map[string]uint64

// topAddrsFloat is like [topAddrs] but the value is float64 number.
type topAddrsFloat = map[string]float64

// StatsResp is a response to the GET /control/stats.
type StatsResp struct {
	TimeUnits string `json:"time_units"`

	TopQueried []topAddrs `json:"top_queried_domains"`
	TopClients []topAddrs `json:"top_clients"`
	TopBlocked []topAddrs `json:"top_blocked_domains"`

	TopUpstreamsResponses []topAddrs      `json:"top_upstreams_responses"`
	TopUpstreamsAvgTime   []topAddrsFloat `json:"top_upstreams_avg_time"`

	DNSQueries []uint64 `json:"dns_queries"`

	BlockedFiltering     []uint64 `json:"blocked_filtering"`
	ReplacedSafebrowsing []uint64 `json:"replaced_safebrowsing"`
	ReplacedParental     []uint64 `json:"replaced_parental"`

	NumDNSQueries           uint64 `json:"num_dns_queries"`
	NumBlockedFiltering     uint64 `json:"num_blocked_filtering"`
	NumReplacedSafebrowsing uint64 `json:"num_replaced_safebrowsing"`
	NumReplacedSafesearch   uint64 `json:"num_replaced_safesearch"`
	NumReplacedParental     uint64 `json:"num_replaced_parental"`

	AvgProcessingTime float64 `json:"avg_processing_time"`
}

// handleStats is the handler for the GET /control/stats HTTP API.
func (s *StatsCtx) handleStats(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	ctx := r.Context()
	l := s.logger

	var limit time.Duration
	func() {
		s.confMu.RLock()
		defer s.confMu.RUnlock()

		limit = s.limit
	}()

	recent := r.URL.Query().Get(queryKeyRecent)

	limit, err := parseRecent(recent, limit)
	if err != nil {
		aghhttp.ErrorAndLog(ctx, l, r, w, http.StatusBadRequest, "%s", err)

		return
	}

	resp, ok := s.getData(uint32(limit.Hours()))

	l.DebugContext(ctx, "prepared data", "elapsed", time.Since(start))

	if !ok {
		// Don't bring the message to the lower case since it's a part of UI
		// text for the moment.
		const msg = "Couldn't get statistics data"
		aghhttp.ErrorAndLog(ctx, l, r, w, http.StatusInternalServerError, msg)

		return
	}

	aghhttp.WriteJSONResponseOK(ctx, l, w, r, resp)
}

// parseRecent parses and validates the value of the recent URL parameter.  If
// the parameter is empty, the original limit is returned.
func parseRecent(recent string, limit time.Duration) (parsedLimit time.Duration, err error) {
	if recent == "" {
		return limit, nil
	}

	recentMs, err := strconv.ParseInt(recent, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s: parsing interval: %s", queryKeyRecent, err)
	}

	err = validate.InRange(queryKeyRecent, recentMs, millisecondsInHour, limit.Milliseconds())
	if err != nil {
		// Don't wrap the error since it's already informative enough as is.
		return 0, err
	}

	if recentMs%millisecondsInHour != 0 {
		return 0, fmt.Errorf("%s: must be a multiple of 1 hour", queryKeyRecent)
	}

	return time.Duration(recentMs) * time.Millisecond, nil
}

// getConfigResp is the response for the GET /control/stats/config HTTP API.
type getConfigResp struct {
	// Ignored is the list of host names, which should not be counted.
	Ignored []string `json:"ignored"`

	// Interval is the statistics rotation interval in milliseconds.
	Interval float64 `json:"interval"`

	// Enabled shows if statistics are enabled.  It is an aghalg.NullBool to be
	// able to tell when it's set without using pointers.
	Enabled aghalg.NullBool `json:"enabled"`

	// IgnoredEnabled defines if ignored list is enabled.
	IgnoredEnabled aghalg.NullBool `json:"ignored_enabled"`
}

// handleGetStatsConfig is the handler for the GET /control/stats/config HTTP
// API.
func (s *StatsCtx) handleGetStatsConfig(w http.ResponseWriter, r *http.Request) {
	var resp *getConfigResp
	func() {
		s.confMu.RLock()
		defer s.confMu.RUnlock()

		resp = &getConfigResp{
			Ignored:        s.ignored.Values(),
			IgnoredEnabled: aghalg.BoolToNullBool(s.ignored.IsEnabled()),
			Interval:       float64(s.limit.Milliseconds()),
			Enabled:        aghalg.BoolToNullBool(s.enabled),
		}
	}()

	aghhttp.WriteJSONResponseOK(r.Context(), s.logger, w, r, resp)
}

// handlePutStatsConfig is the handler for the PUT /control/stats/config/update
// HTTP API.
func (s *StatsCtx) handlePutStatsConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	reqData := getConfigResp{}
	err := json.NewDecoder(r.Body).Decode(&reqData)
	if err != nil {
		aghhttp.ErrorAndLog(ctx, s.logger, r, w, http.StatusBadRequest, "json decode: %s", err)

		return
	}

	var ignoredEnabled bool
	if reqData.IgnoredEnabled == aghalg.NBNull {
		ignoredEnabled = len(reqData.Ignored) > 0
	} else {
		ignoredEnabled = reqData.IgnoredEnabled == aghalg.NBTrue
	}

	engine, err := aghnet.NewIgnoreEngine(reqData.Ignored, ignoredEnabled)
	if err != nil {
		aghhttp.ErrorAndLog(ctx, s.logger, r, w, http.StatusUnprocessableEntity, "ignored: %s", err)

		return
	}

	ivl := time.Duration(reqData.Interval) * time.Millisecond
	err = validateIvl(ivl)
	if err != nil {
		aghhttp.ErrorAndLog(
			ctx,
			s.logger,
			r,
			w,
			http.StatusUnprocessableEntity,
			"unsupported interval: %s",
			err,
		)

		return
	}

	if reqData.Enabled == aghalg.NBNull {
		aghhttp.ErrorAndLog(ctx, s.logger, r, w, http.StatusUnprocessableEntity, "enabled is null")

		return
	}

	defer s.configModifier.Apply(ctx)

	s.confMu.Lock()
	defer s.confMu.Unlock()

	s.ignored = engine
	s.limit = ivl
	s.enabled = reqData.Enabled == aghalg.NBTrue
}

// handleStatsReset is the handler for the POST /control/stats_reset HTTP API.
func (s *StatsCtx) handleStatsReset(w http.ResponseWriter, r *http.Request) {
	err := s.clear()
	if err != nil {
		aghhttp.ErrorAndLog(
			r.Context(),
			s.logger,
			r,
			w,
			http.StatusInternalServerError,
			"stats: %s",
			err,
		)
	}
}

// initWeb registers the handlers for web endpoints of statistics module.
func (s *StatsCtx) initWeb() {
	s.httpReg.Register(http.MethodGet, "/control/stats", s.handleStats)
	s.httpReg.Register(http.MethodPost, "/control/stats_reset", s.handleStatsReset)
	s.httpReg.Register(http.MethodGet, "/control/stats/config", s.handleGetStatsConfig)
	s.httpReg.Register(http.MethodPut, "/control/stats/config/update", s.handlePutStatsConfig)
}
