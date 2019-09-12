package home

import (
	"encoding/json"
	"net/http"

	"github.com/AdguardTeam/AdGuardHome/querylog"
)

func handleQueryLog(w http.ResponseWriter, r *http.Request) {
	data := config.queryLog.GetData()

	jsonVal, err := json.Marshal(data)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Couldn't marshal data into json: %s", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonVal)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Unable to write response json: %s", err)
	}
}

func handleQueryLogClear(w http.ResponseWriter, r *http.Request) {
	config.queryLog.Clear()
	returnOK(w)
}

type qlogConfig struct {
	Enabled  bool   `json:"enabled"`
	Interval uint32 `json:"interval"`
}

// Get configuration
func handleQueryLogInfo(w http.ResponseWriter, r *http.Request) {
	resp := qlogConfig{}
	resp.Enabled = config.DNS.QueryLogEnabled
	resp.Interval = config.DNS.QueryLogInterval

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

// Set configuration
func handleQueryLogConfig(w http.ResponseWriter, r *http.Request) {

	reqData := qlogConfig{}
	err := json.NewDecoder(r.Body).Decode(&reqData)
	if err != nil {
		httpError(w, http.StatusBadRequest, "json decode: %s", err)
		return
	}

	if !checkQueryLogInterval(reqData.Interval) {
		httpError(w, http.StatusBadRequest, "Unsupported interval")
		return
	}

	config.DNS.QueryLogEnabled = reqData.Enabled
	config.DNS.QueryLogInterval = reqData.Interval
	_ = config.write()

	conf := querylog.Config{
		Interval: config.DNS.QueryLogInterval * 24,
	}
	config.queryLog.Configure(conf)

	returnOK(w)
}

func checkQueryLogInterval(i uint32) bool {
	return i == 1 || i == 7 || i == 30 || i == 90
}

// RegisterQueryLogHandlers - register handlers
func RegisterQueryLogHandlers() {
	httpRegister(http.MethodGet, "/control/querylog", handleQueryLog)
	httpRegister(http.MethodGet, "/control/querylog_info", handleQueryLogInfo)
	httpRegister(http.MethodPost, "/control/querylog_clear", handleQueryLogClear)
	httpRegister(http.MethodPost, "/control/querylog_config", handleQueryLogConfig)
}
