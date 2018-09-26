package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"runtime"
	"sort"
	"strings"
	"time"
)

func clamp(value, low, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

// ----------------------------------
// helper functions for HTTP handlers
// ----------------------------------
func ensure(method string, handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != method {
			http.Error(w, "This request must be "+method, 405)
			return
		}
		handler(w, r)
	}
}

func ensurePOST(handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return ensure("POST", handler)
}

func ensureGET(handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return ensure("GET", handler)
}

func ensurePUT(handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return ensure("PUT", handler)
}

func ensureDELETE(handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return ensure("DELETE", handler)
}

func optionalAuth(handler func(http.ResponseWriter, *http.Request)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if config.AuthName == "" || config.AuthPass == "" {
			handler(w, r)
			return
		}
		user, pass, ok := r.BasicAuth()
		if !ok || user != config.AuthName || pass != config.AuthPass {
			w.Header().Set("WWW-Authenticate", `Basic realm="dnsfilter"`)
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorised.\n"))
			return
		}
		handler(w, r)
	}
}

type authHandler struct {
	handler http.Handler
}

func (a *authHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if config.AuthName == "" || config.AuthPass == "" {
		a.handler.ServeHTTP(w, r)
		return
	}
	user, pass, ok := r.BasicAuth()
	if !ok || user != config.AuthName || pass != config.AuthPass {
		w.Header().Set("WWW-Authenticate", `Basic realm="dnsfilter"`)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Unauthorised.\n"))
		return
	}
	a.handler.ServeHTTP(w, r)
}

func optionalAuthHandler(handler http.Handler) http.Handler {
	return &authHandler{handler}
}

// --------------------------
// helper functions for stats
// --------------------------
func getReversedSlice(input [statsHistoryElements]float64, start int, end int) []float64 {
	output := make([]float64, 0)
	for i := start; i <= end; i++ {
		output = append([]float64{input[i]}, output...)
	}
	return output
}

func generateMapFromStats(stats *periodicStats, start int, end int) map[string]interface{} {
	// clamp
	start = clamp(start, 0, statsHistoryElements)
	end = clamp(end, 0, statsHistoryElements)

	avgProcessingTime := make([]float64, 0)

	count := getReversedSlice(stats.entries[processingTimeCount], start, end)
	sum := getReversedSlice(stats.entries[processingTimeSum], start, end)
	for i := 0; i < len(count); i++ {
		var avg float64
		if count[i] != 0 {
			avg = sum[i] / count[i]
			avg *= 1000
		}
		avgProcessingTime = append(avgProcessingTime, avg)
	}

	result := map[string]interface{}{
		"dns_queries":           getReversedSlice(stats.entries[totalRequests], start, end),
		"blocked_filtering":     getReversedSlice(stats.entries[filteredTotal], start, end),
		"replaced_safebrowsing": getReversedSlice(stats.entries[filteredSafebrowsing], start, end),
		"replaced_safesearch":   getReversedSlice(stats.entries[filteredSafesearch], start, end),
		"replaced_parental":     getReversedSlice(stats.entries[filteredParental], start, end),
		"avg_processing_time":   avgProcessingTime,
	}
	return result
}

// -------------------------------------
// helper functions for querylog parsing
// -------------------------------------
func sortByValue(m map[string]int) []string {
	type kv struct {
		k string
		v int
	}
	var ss []kv
	for k, v := range m {
		ss = append(ss, kv{k, v})
	}
	sort.Slice(ss, func(l, r int) bool {
		return ss[l].v > ss[r].v
	})

	sorted := []string{}
	for _, v := range ss {
		sorted = append(sorted, v.k)
	}
	return sorted
}

func getHost(entry map[string]interface{}) string {
	q, ok := entry["question"]
	if !ok {
		return ""
	}
	question, ok := q.(map[string]interface{})
	if !ok {
		return ""
	}
	h, ok := question["host"]
	if !ok {
		return ""
	}
	host, ok := h.(string)
	if !ok {
		return ""
	}
	return host
}

func getReason(entry map[string]interface{}) string {
	r, ok := entry["reason"]
	if !ok {
		return ""
	}
	reason, ok := r.(string)
	if !ok {
		return ""
	}
	return reason
}

func getClient(entry map[string]interface{}) string {
	c, ok := entry["client"]
	if !ok {
		return ""
	}
	client, ok := c.(string)
	if !ok {
		return ""
	}
	return client
}

func getTime(entry map[string]interface{}) time.Time {
	t, ok := entry["time"]
	if !ok {
		return time.Time{}
	}
	tstr, ok := t.(string)
	if !ok {
		return time.Time{}
	}
	value, err := time.Parse(time.RFC3339, tstr)
	if err != nil {
		return time.Time{}
	}
	return value
}

// -------------------------------------------------
// helper functions for parsing parameters from body
// -------------------------------------------------
func parseParametersFromBody(r io.Reader) (map[string]string, error) {
	parameters := map[string]string{}

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 {
			// skip empty lines
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return parameters, errors.New("Got invalid request body")
		}
		parameters[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}

	return parameters, nil
}

// ---------------------
// debug logging helpers
// ---------------------
func _Func() string {
	pc := make([]uintptr, 10) // at least 1 entry needed
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	return path.Base(f.Name())
}

func trace(format string, args ...interface{}) {
	pc := make([]uintptr, 10) // at least 1 entry needed
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("%s(): ", path.Base(f.Name())))
	text := fmt.Sprintf(format, args...)
	buf.WriteString(text)
	if len(text) == 0 || text[len(text)-1] != '\n' {
		buf.WriteRune('\n')
	}
	fmt.Fprint(os.Stderr, buf.String())
}
