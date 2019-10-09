package querylog

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/AdguardTeam/golibs/log"
	"github.com/miekg/dns"
)

func httpError(r *http.Request, w http.ResponseWriter, code int, format string, args ...interface{}) {
	text := fmt.Sprintf(format, args...)

	log.Info("QueryLog: %s %s: %s", r.Method, r.URL, text)

	http.Error(w, text, code)
}

type filterJSON struct {
	Domain         string `json:"domain"`
	Client         string `json:"client"`
	QuestionType   string `json:"question_type"`
	ResponseStatus string `json:"response_status"`
}

type request struct {
	OlderThan string     `json:"older_than"`
	Filter    filterJSON `json:"filter"`
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

func (l *queryLog) handleQueryLog(w http.ResponseWriter, r *http.Request) {
	req := request{}
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		httpError(r, w, http.StatusBadRequest, "json decode: %s", err)
		return
	}

	params := getDataParams{
		Domain:         req.Filter.Domain,
		Client:         req.Filter.Client,
		ResponseStatus: responseStatusAll,
	}
	if len(req.OlderThan) != 0 {
		params.OlderThan, err = time.Parse(time.RFC3339Nano, req.OlderThan)
		if err != nil {
			httpError(r, w, http.StatusBadRequest, "invalid time stamp: %s", err)
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
			httpError(r, w, http.StatusBadRequest, "invalid question_type")
			return
		}
		params.QuestionType = qtype
	}

	if len(req.Filter.ResponseStatus) != 0 {
		switch req.Filter.ResponseStatus {
		case "filtered":
			params.ResponseStatus = responseStatusFiltered
		default:
			httpError(r, w, http.StatusBadRequest, "invalid response_status")
			return
		}
	}

	data := l.getData(params)

	jsonVal, err := json.Marshal(data)
	if err != nil {
		httpError(r, w, http.StatusInternalServerError, "Couldn't marshal data into json: %s", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonVal)
	if err != nil {
		httpError(r, w, http.StatusInternalServerError, "Unable to write response json: %s", err)
	}
}

func (l *queryLog) handleQueryLogClear(w http.ResponseWriter, r *http.Request) {
	l.clear()
}

type qlogConfig struct {
	Enabled  bool   `json:"enabled"`
	Interval uint32 `json:"interval"`
}

// Get configuration
func (l *queryLog) handleQueryLogInfo(w http.ResponseWriter, r *http.Request) {
	resp := qlogConfig{}
	resp.Enabled = l.conf.Enabled
	resp.Interval = l.conf.Interval

	jsonVal, err := json.Marshal(resp)
	if err != nil {
		httpError(r, w, http.StatusInternalServerError, "json encode: %s", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonVal)
	if err != nil {
		httpError(r, w, http.StatusInternalServerError, "http write: %s", err)
	}
}

// Set configuration
func (l *queryLog) handleQueryLogConfig(w http.ResponseWriter, r *http.Request) {

	reqData := qlogConfig{}
	err := json.NewDecoder(r.Body).Decode(&reqData)
	if err != nil {
		httpError(r, w, http.StatusBadRequest, "json decode: %s", err)
		return
	}

	if !checkInterval(reqData.Interval) {
		httpError(r, w, http.StatusBadRequest, "Unsupported interval")
		return
	}

	conf := Config{
		Enabled:  reqData.Enabled,
		Interval: reqData.Interval,
	}
	l.configure(conf)

	l.conf.ConfigModified()
}

// Register web handlers
func (l *queryLog) initWeb() {
	l.conf.HTTPRegister("POST", "/control/querylog", l.handleQueryLog)
	l.conf.HTTPRegister("GET", "/control/querylog_info", l.handleQueryLogInfo)
	l.conf.HTTPRegister("POST", "/control/querylog_clear", l.handleQueryLogClear)
	l.conf.HTTPRegister("POST", "/control/querylog_config", l.handleQueryLogConfig)
}
