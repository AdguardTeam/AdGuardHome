package home

import (
	"encoding/json"
	"net/http"

	"github.com/AdguardTeam/AdGuardHome/stats"
	"github.com/AdguardTeam/golibs/log"
)

type statsConfig struct {
	Interval uint `json:"interval"`
}

// Get stats configuration
func handleStatsInfo(w http.ResponseWriter, r *http.Request) {
	resp := statsConfig{}
	resp.Interval = config.DNS.StatsInterval

	jsonVal, err := json.Marshal(resp)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "json encode: %s", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonVal)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "http write: %s", err)
	}
}

// Set stats configuration
func handleStatsConfig(w http.ResponseWriter, r *http.Request) {
	reqData := statsConfig{}
	err := json.NewDecoder(r.Body).Decode(&reqData)
	if err != nil {
		httpError(w, http.StatusBadRequest, "json decode: %s", err)
		return
	}

	if !checkStatsInterval(reqData.Interval) {
		httpError(w, http.StatusBadRequest, "Unsupported interval")
		return
	}

	config.DNS.StatsInterval = reqData.Interval
	config.stats.Configure(int(config.DNS.StatsInterval))

	returnOK(w)
}

// handleStats returns aggregated stats data
func handleStats(w http.ResponseWriter, r *http.Request) {
	units := stats.Hours
	if config.DNS.StatsInterval > 7 {
		units = stats.Days
	}
	counter := log.StartTimer()
	d := config.stats.GetData(units)
	counter.LogElapsed("Stats: prepared data")

	if d == nil {
		httpError(w, http.StatusInternalServerError, "Couldn't get statistics data")
		return
	}

	data, err := json.Marshal(d)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "json encode: %s", err)
		return
	}

	w.Write(data)
}

// handleStatsReset resets the stats
func handleStatsReset(w http.ResponseWriter, r *http.Request) {
	config.stats.Clear()
	returnOK(w)
}

// RegisterStatsHandlers - register handlers
func RegisterStatsHandlers() {
	httpRegister(http.MethodGet, "/control/stats", handleStats)
	httpRegister(http.MethodPost, "/control/stats_reset", handleStatsReset)
	httpRegister(http.MethodPost, "/control/stats_config", handleStatsConfig)
	httpRegister(http.MethodGet, "/control/stats_info", handleStatsInfo)
}

func checkStatsInterval(i uint) bool {
	return i == 1 || i == 7 || i == 30 || i == 90
}
