// HTTP request handlers for accessing statistics data and configuration settings

package stats

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/golibs/log"
)

// topAddrs is an alias for the types of the TopFoo fields of statsResponse.
// The key is either a client's address or a requested address.
type topAddrs = map[string]uint64

// statsResponse is a response for getting statistics.
type statsResponse struct {
	TimeUnits string `json:"time_units"`

	NumDNSQueries           uint64 `json:"num_dns_queries"`
	NumBlockedFiltering     uint64 `json:"num_blocked_filtering"`
	NumReplacedSafebrowsing uint64 `json:"num_replaced_safebrowsing"`
	NumReplacedSafesearch   uint64 `json:"num_replaced_safesearch"`
	NumReplacedParental     uint64 `json:"num_replaced_parental"`

	AvgProcessingTime float64 `json:"avg_processing_time"`

	TopQueried []topAddrs `json:"top_queried_domains"`
	TopClients []topAddrs `json:"top_clients"`
	TopBlocked []topAddrs `json:"top_blocked_domains"`

	DNSQueries []uint64 `json:"dns_queries"`

	BlockedFiltering     []uint64 `json:"blocked_filtering"`
	ReplacedSafebrowsing []uint64 `json:"replaced_safebrowsing"`
	ReplacedParental     []uint64 `json:"replaced_parental"`
}

// handleStats is a handler for getting statistics.
func (s *StatsCtx) handleStats(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	var resp statsResponse
	var ok bool
	resp, ok = s.getData()

	log.Debug("stats: prepared data in %v", time.Since(start))

	if !ok {
		// Don't bring the message to the lower case since it's a part of UI
		// text for the moment.
		aghhttp.Error(r, w, http.StatusInternalServerError, "Couldn't get statistics data")

		return
	}

	w.Header().Set("Content-Type", "application/json")

	err := json.NewEncoder(w).Encode(resp)
	if err != nil {
		aghhttp.Error(r, w, http.StatusInternalServerError, "json encode: %s", err)

		return
	}
}

type config struct {
	IntervalDays uint32 `json:"interval"`
}

// Get configuration
func (s *StatsCtx) handleStatsInfo(w http.ResponseWriter, r *http.Request) {
	resp := config{}
	resp.IntervalDays = s.limitHours / 24

	data, err := json.Marshal(resp)
	if err != nil {
		aghhttp.Error(r, w, http.StatusInternalServerError, "json encode: %s", err)

		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(data)
	if err != nil {
		aghhttp.Error(r, w, http.StatusInternalServerError, "http write: %s", err)
	}
}

// Set configuration
func (s *StatsCtx) handleStatsConfig(w http.ResponseWriter, r *http.Request) {
	reqData := config{}
	err := json.NewDecoder(r.Body).Decode(&reqData)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "json decode: %s", err)

		return
	}

	if !checkInterval(reqData.IntervalDays) {
		aghhttp.Error(r, w, http.StatusBadRequest, "Unsupported interval")

		return
	}

	s.setLimit(int(reqData.IntervalDays))
	s.configModified()
}

// Reset data
func (s *StatsCtx) handleStatsReset(w http.ResponseWriter, r *http.Request) {
	s.clear()
}

// Register web handlers
func (s *StatsCtx) initWeb() {
	if s.httpRegister == nil {
		return
	}

	s.httpRegister(http.MethodGet, "/control/stats", s.handleStats)
	s.httpRegister(http.MethodPost, "/control/stats_reset", s.handleStatsReset)
	s.httpRegister(http.MethodPost, "/control/stats_config", s.handleStatsConfig)
	s.httpRegister(http.MethodGet, "/control/stats_info", s.handleStatsInfo)
}
