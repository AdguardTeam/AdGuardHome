package home

import (
	"encoding/json"
	"net/http"

	"github.com/AdguardTeam/golibs/log"
)

type statsConfig struct {
	Interval uint `json:"interval"`
}

func handleStatsInfo(w http.ResponseWriter, r *http.Request) {
	log.Tracef("%s %v", r.Method, r.URL)

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

func handleStatsConfig(w http.ResponseWriter, r *http.Request) {
	log.Tracef("%s %v", r.Method, r.URL)

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

	httpUpdateConfigReloadDNSReturnOK(w, r)
}

func checkStatsInterval(i uint) bool {
	return i == 1 || i == 7 || i == 30 || i == 90
}
