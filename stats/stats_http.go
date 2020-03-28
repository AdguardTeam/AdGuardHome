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

// Return data
func (s *statsCtx) handleStats(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	d := s.getData()
	log.Debug("Stats: prepared data in %v", time.Since(start))

	if d == nil {
		httpError(r, w, http.StatusInternalServerError, "Couldn't get statistics data")
		return
	}

	data, err := json.Marshal(d)
	if err != nil {
		httpError(r, w, http.StatusInternalServerError, "json encode: %s", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
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

	s.conf.HTTPRegister("GET", "/control/stats", s.handleStats)
	s.conf.HTTPRegister("POST", "/control/stats_reset", s.handleStatsReset)
	s.conf.HTTPRegister("POST", "/control/stats_config", s.handleStatsConfig)
	s.conf.HTTPRegister("GET", "/control/stats_info", s.handleStatsInfo)
}
