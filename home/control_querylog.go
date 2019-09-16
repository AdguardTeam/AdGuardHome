package home

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/AdguardTeam/AdGuardHome/querylog"
	"github.com/miekg/dns"
)

type qlogFilterJSON struct {
	Domain         string `json:"domain"`
	Client         string `json:"client"`
	QuestionType   string `json:"question_type"`
	ResponseStatus string `json:"response_status"`
}

type queryLogRequest struct {
	OlderThan string         `json:"older_than"`
	Filter    qlogFilterJSON `json:"filter"`
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

func handleQueryLog(w http.ResponseWriter, r *http.Request) {
	req := queryLogRequest{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		httpError(w, http.StatusBadRequest, "json decode: %s", err)
		return
	}

	params := querylog.GetDataParams{
		Domain: req.Filter.Domain,
		Client: req.Filter.Client,
	}
	if len(req.OlderThan) != 0 {
		params.OlderThan, err = time.Parse(time.RFC3339Nano, req.OlderThan)
		if err != nil {
			httpError(w, http.StatusBadRequest, "invalid time stamp: %s", err)
			return
		}
	}

	if getDoubleQuotesEnclosedValue(&params.Domain) {
		params.StrictMatchDomain = true
	}
	if getDoubleQuotesEnclosedValue(&params.Client) {
		params.StrictMatchClient = true
	}

	if len(req.Filter.QuestionType) != 0 {
		qtype, ok := dns.StringToType[req.Filter.QuestionType]
		if !ok {
			httpError(w, http.StatusBadRequest, "invalid question_type")
			return
		}
		params.QuestionType = qtype
	}

	if len(req.Filter.ResponseStatus) != 0 {
		switch req.Filter.ResponseStatus {
		case "filtered":
			params.ResponseStatus = querylog.ResponseStatusFiltered
		default:
			httpError(w, http.StatusBadRequest, "invalid response_status")
			return
		}
	}

	data := config.queryLog.GetData(params)

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
	httpRegister("POST", "/control/querylog", handleQueryLog)
	httpRegister(http.MethodGet, "/control/querylog_info", handleQueryLogInfo)
	httpRegister(http.MethodPost, "/control/querylog_clear", handleQueryLogClear)
	httpRegister(http.MethodPost, "/control/querylog_config", handleQueryLogConfig)
}
