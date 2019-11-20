package querylog

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/AdguardTeam/golibs/jsonutil"
	"github.com/AdguardTeam/golibs/log"
	"github.com/miekg/dns"
)

func httpError(r *http.Request, w http.ResponseWriter, code int, format string, args ...interface{}) {
	text := fmt.Sprintf(format, args...)

	log.Info("QueryLog: %s %s: %s", r.Method, r.URL, text)

	http.Error(w, text, code)
}

type request struct {
	olderThan            string
	filterDomain         string
	filterClient         string
	filterQuestionType   string
	filterResponseStatus string
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
	var err error
	req := request{}
	q := r.URL.Query()
	req.olderThan = q.Get("older_than")
	req.filterDomain = q.Get("filter_domain")
	req.filterClient = q.Get("filter_client")
	req.filterQuestionType = q.Get("filter_question_type")
	req.filterResponseStatus = q.Get("filter_response_status")

	params := getDataParams{
		Domain:         req.filterDomain,
		Client:         req.filterClient,
		ResponseStatus: responseStatusAll,
	}
	if len(req.olderThan) != 0 {
		params.OlderThan, err = time.Parse(time.RFC3339Nano, req.olderThan)
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

	if len(req.filterQuestionType) != 0 {
		_, ok := dns.StringToType[req.filterQuestionType]
		if !ok {
			httpError(r, w, http.StatusBadRequest, "invalid question_type")
			return
		}
		params.QuestionType = req.filterQuestionType
	}

	if len(req.filterResponseStatus) != 0 {
		switch req.filterResponseStatus {
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
	d := qlogConfig{}
	req, err := jsonutil.DecodeObject(&d, r.Body)
	if err != nil {
		httpError(r, w, http.StatusBadRequest, "%s", err)
		return
	}

	if req.Exists("interval") && !checkInterval(d.Interval) {
		httpError(r, w, http.StatusBadRequest, "Unsupported interval")
		return
	}

	l.lock.Lock()
	// copy data, modify it, then activate.  Other threads (readers) don't need to use this lock.
	conf := *l.conf
	if req.Exists("enabled") {
		conf.Enabled = d.Enabled
	}
	if req.Exists("interval") {
		conf.Interval = d.Interval
	}
	l.conf = &conf
	l.lock.Unlock()

	l.conf.ConfigModified()
}

// Register web handlers
func (l *queryLog) initWeb() {
	l.conf.HTTPRegister("GET", "/control/querylog", l.handleQueryLog)
	l.conf.HTTPRegister("GET", "/control/querylog_info", l.handleQueryLogInfo)
	l.conf.HTTPRegister("POST", "/control/querylog_clear", l.handleQueryLogClear)
	l.conf.HTTPRegister("POST", "/control/querylog_config", l.handleQueryLogConfig)
}
