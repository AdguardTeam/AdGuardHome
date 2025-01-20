// HTTP request handlers for accessing statistics data and configuration settings

package stats

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghalg"
	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/golibs/timeutil"
)

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

	var (
		resp *StatsResp
		ok   bool
	)
	func() {
		s.confMu.RLock()
		defer s.confMu.RUnlock()

		resp, ok = s.getData(uint32(s.limit.Hours()))
	}()

	s.logger.DebugContext(ctx, "prepared data", "elapsed", time.Since(start))

	if !ok {
		// Don't bring the message to the lower case since it's a part of UI
		// text for the moment.
		const msg = "Couldn't get statistics data"
		aghhttp.ErrorAndLog(ctx, s.logger, r, w, http.StatusInternalServerError, msg)

		return
	}

	aghhttp.WriteJSONResponseOK(w, r, resp)
}

// configResp is the response to the GET /control/stats_info.
type configResp struct {
	IntervalDays uint32 `json:"interval"`
}

// getConfigResp is the response to the GET /control/stats_info.
type getConfigResp struct {
	// Ignored is the list of host names, which should not be counted.
	Ignored []string `json:"ignored"`

	// Interval is the statistics rotation interval in milliseconds.
	Interval float64 `json:"interval"`

	// Enabled shows if statistics are enabled.  It is an aghalg.NullBool to be
	// able to tell when it's set without using pointers.
	Enabled aghalg.NullBool `json:"enabled"`
}

// handleStatsInfo is the handler for the GET /control/stats_info HTTP API.
//
// Deprecated:  Remove it when migration to the new API is over.
func (s *StatsCtx) handleStatsInfo(w http.ResponseWriter, r *http.Request) {
	var (
		enabled bool
		limit   time.Duration
	)
	func() {
		s.confMu.RLock()
		defer s.confMu.RUnlock()

		enabled, limit = s.enabled, s.limit
	}()

	days := uint32(limit / timeutil.Day)
	ok := checkInterval(days)
	if !ok || (enabled && days == 0) {
		// NOTE: If interval is custom we set it to 90 days for compatibility
		// with old API.
		days = 90
	}

	resp := configResp{IntervalDays: days}
	if !enabled {
		resp.IntervalDays = 0
	}

	aghhttp.WriteJSONResponseOK(w, r, resp)
}

// handleGetStatsConfig is the handler for the GET /control/stats/config HTTP
// API.
func (s *StatsCtx) handleGetStatsConfig(w http.ResponseWriter, r *http.Request) {
	var resp *getConfigResp
	func() {
		s.confMu.RLock()
		defer s.confMu.RUnlock()

		resp = &getConfigResp{
			Ignored:  s.ignored.Values(),
			Interval: float64(s.limit.Milliseconds()),
			Enabled:  aghalg.BoolToNullBool(s.enabled),
		}
	}()

	aghhttp.WriteJSONResponseOK(w, r, resp)
}

// handleStatsConfig is the handler for the POST /control/stats_config HTTP API.
//
// Deprecated:  Remove it when migration to the new API is over.
func (s *StatsCtx) handleStatsConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	reqData := configResp{}
	err := json.NewDecoder(r.Body).Decode(&reqData)
	if err != nil {
		aghhttp.ErrorAndLog(ctx, s.logger, r, w, http.StatusBadRequest, "json decode: %s", err)

		return
	}

	if !checkInterval(reqData.IntervalDays) {
		aghhttp.ErrorAndLog(ctx, s.logger, r, w, http.StatusBadRequest, "Unsupported interval")

		return
	}

	limit := time.Duration(reqData.IntervalDays) * timeutil.Day

	defer s.configModified()

	s.confMu.Lock()
	defer s.confMu.Unlock()

	s.setLimit(limit)
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

	engine, err := aghnet.NewIgnoreEngine(reqData.Ignored)
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

	defer s.configModified()

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
	if s.httpRegister == nil {
		return
	}

	s.httpRegister(http.MethodGet, "/control/stats", s.handleStats)
	s.httpRegister(http.MethodPost, "/control/stats_reset", s.handleStatsReset)
	s.httpRegister(http.MethodGet, "/control/stats/config", s.handleGetStatsConfig)
	s.httpRegister(http.MethodPut, "/control/stats/config/update", s.handlePutStatsConfig)

	// Deprecated handlers.
	s.httpRegister(http.MethodGet, "/control/stats_info", s.handleStatsInfo)
	s.httpRegister(http.MethodPost, "/control/stats_config", s.handleStatsConfig)
}
