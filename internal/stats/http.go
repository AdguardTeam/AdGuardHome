// HTTP request handlers for accessing statistics data and configuration settings

package stats

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/AdguardTeam/golibs/log"
)

func httpError(r *http.Request, w http.ResponseWriter, code int, format string, args ...interface{}) {
	text := fmt.Sprintf(format, args...)

	log.Info("Stats: %s %s: %s", r.Method, r.URL, text)

	http.Error(w, text, code)
}

// statsResponse is a response for getting statistics.
type statsResponse struct {
	TimeUnits string `json:"time_units"`

	NumDNSQueries           uint64 `json:"num_dns_queries"`
	NumBlockedFiltering     uint64 `json:"num_blocked_filtering"`
	NumReplacedSafebrowsing uint64 `json:"num_replaced_safebrowsing"`
	NumReplacedSafesearch   uint64 `json:"num_replaced_safesearch"`
	NumReplacedParental     uint64 `json:"num_replaced_parental"`

	AvgProcessingTime float64 `json:"avg_processing_time"`

	TopQueried []map[string]uint64 `json:"top_queried_domains"`
	TopClients []map[string]uint64 `json:"top_clients"`
	TopBlocked []map[string]uint64 `json:"top_blocked_domains"`

	DNSQueries []uint64 `json:"dns_queries"`

	BlockedFiltering     []uint64 `json:"blocked_filtering"`
	ReplacedSafebrowsing []uint64 `json:"replaced_safebrowsing"`
	ReplacedParental     []uint64 `json:"replaced_parental"`
}

// handleStats is a handler for getting statistics.
func (s *statsCtx) handleStats(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	response, ok := s.getData()
	log.Debug("Stats: prepared data in %v", time.Since(start))

	if !ok {
		httpError(r, w, http.StatusInternalServerError, "Couldn't get statistics data")

		return
	}

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		httpError(r, w, http.StatusInternalServerError, "json encode: %s", err)

		return
	}
}

type config struct {
	IntervalDays uint32 `json:"interval"`
}

// Get configuration
func (s *statsCtx) handleStatsInfo(w http.ResponseWriter, r *http.Request) {
	resp := config{}
	resp.IntervalDays = s.conf.limit / 24

	data, err := json.Marshal(resp)
	if err != nil {
		httpError(r, w, http.StatusInternalServerError, "json encode: %s", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(data)
	if err != nil {
		httpError(r, w, http.StatusInternalServerError, "http write: %s", err)
	}
}

// Set configuration
func (s *statsCtx) handleStatsConfig(w http.ResponseWriter, r *http.Request) {
	reqData := config{}
	err := json.NewDecoder(r.Body).Decode(&reqData)
	if err != nil {
		httpError(r, w, http.StatusBadRequest, "json decode: %s", err)
		return
	}

	if !checkInterval(reqData.IntervalDays) {
		httpError(r, w, http.StatusBadRequest, "Unsupported interval")
		return
	}

	s.setLimit(int(reqData.IntervalDays))
	s.conf.ConfigModified()
}

// Reset data
func (s *statsCtx) handleStatsReset(w http.ResponseWriter, r *http.Request) {
	s.clear()
}

// Register web handlers
func (s *statsCtx) initWeb() {
	if s.conf.HTTPRegister == nil {
		return
	}

	s.conf.HTTPRegister(http.MethodGet, "/control/stats", s.handleStats)
	s.conf.HTTPRegister(http.MethodPost, "/control/stats_reset", s.handleStatsReset)
	s.conf.HTTPRegister(http.MethodPost, "/control/stats_config", s.handleStatsConfig)
	s.conf.HTTPRegister(http.MethodGet, "/control/stats_info", s.handleStatsInfo)
}
