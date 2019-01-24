package dnsforward

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/dnsfilter"
	"github.com/hmage/golibs/log"
)

var (
	requests             = newDNSCounter("requests_total")
	filtered             = newDNSCounter("filtered_total")
	filteredLists        = newDNSCounter("filtered_lists_total")
	filteredSafebrowsing = newDNSCounter("filtered_safebrowsing_total")
	filteredParental     = newDNSCounter("filtered_parental_total")
	whitelisted          = newDNSCounter("whitelisted_total")
	safesearch           = newDNSCounter("safesearch_total")
	errorsTotal          = newDNSCounter("errors_total")
	elapsedTime          = newDNSHistogram("request_duration")
)

// entries for single time period (for example all per-second entries)
type statsEntries map[string][statsHistoryElements]float64

// how far back to keep the stats
const statsHistoryElements = 60 + 1 // +1 for calculating delta

// each periodic stat is a map of arrays
type periodicStats struct {
	Entries    statsEntries
	period     time.Duration // how long one entry lasts
	LastRotate time.Time     // last time this data was rotated

	sync.RWMutex
}

type stats struct {
	PerSecond periodicStats
	PerMinute periodicStats
	PerHour   periodicStats
	PerDay    periodicStats
}

// per-second/per-minute/per-hour/per-day stats
var statistics stats

func initPeriodicStats(periodic *periodicStats, period time.Duration) {
	periodic.Entries = statsEntries{}
	periodic.LastRotate = time.Now()
	periodic.period = period
}

func init() {
	purgeStats()
}

func purgeStats() {
	initPeriodicStats(&statistics.PerSecond, time.Second)
	initPeriodicStats(&statistics.PerMinute, time.Minute)
	initPeriodicStats(&statistics.PerHour, time.Hour)
	initPeriodicStats(&statistics.PerDay, time.Hour*24)
}

func (p *periodicStats) Inc(name string, when time.Time) {
	// calculate how many periods ago this happened
	elapsed := int64(time.Since(when) / p.period)
	// log.Tracef("%s: %v as %v -> [%v]", name, time.Since(when), p.period, elapsed)
	if elapsed >= statsHistoryElements {
		return // outside of our timeframe
	}
	p.Lock()
	currentValues := p.Entries[name]
	currentValues[elapsed]++
	p.Entries[name] = currentValues
	p.Unlock()
}

func (p *periodicStats) Observe(name string, when time.Time, value float64) {
	// calculate how many periods ago this happened
	elapsed := int64(time.Since(when) / p.period)
	// log.Tracef("%s: %v as %v -> [%v]", name, time.Since(when), p.period, elapsed)
	if elapsed >= statsHistoryElements {
		return // outside of our timeframe
	}
	p.Lock()
	{
		countname := name + "_count"
		currentValues := p.Entries[countname]
		value := currentValues[elapsed]
		// log.Tracef("Will change p.Entries[%s][%d] from %v to %v", countname, elapsed, value, value+1)
		value++
		currentValues[elapsed] = value
		p.Entries[countname] = currentValues
	}
	{
		totalname := name + "_sum"
		currentValues := p.Entries[totalname]
		currentValues[elapsed] += value
		p.Entries[totalname] = currentValues
	}
	p.Unlock()
}

func (p *periodicStats) statsRotate(now time.Time) {
	p.Lock()
	rotations := int64(now.Sub(p.LastRotate) / p.period)
	if rotations > statsHistoryElements {
		rotations = statsHistoryElements
	}
	// calculate how many times we should rotate
	for r := int64(0); r < rotations; r++ {
		for key, values := range p.Entries {
			newValues := [statsHistoryElements]float64{}
			for i := 1; i < len(values); i++ {
				newValues[i] = values[i-1]
			}
			p.Entries[key] = newValues
		}
	}
	if rotations > 0 {
		p.LastRotate = now
	}
	p.Unlock()
}

func statsRotator() {
	for range time.Tick(time.Second) {
		now := time.Now()
		statistics.PerSecond.statsRotate(now)
		statistics.PerMinute.statsRotate(now)
		statistics.PerHour.statsRotate(now)
		statistics.PerDay.statsRotate(now)
	}
}

// counter that wraps around prometheus Counter but also adds to periodic stats
type counter struct {
	name  string // used as key in periodic stats
	value int64

	sync.Mutex
}

func newDNSCounter(name string) *counter {
	// log.Tracef("called")
	return &counter{
		name: name,
	}
}

func (c *counter) IncWithTime(when time.Time) {
	statistics.PerSecond.Inc(c.name, when)
	statistics.PerMinute.Inc(c.name, when)
	statistics.PerHour.Inc(c.name, when)
	statistics.PerDay.Inc(c.name, when)
	c.Lock()
	c.value++
	c.Unlock()
}

func (c *counter) Inc() {
	c.IncWithTime(time.Now())
}

type histogram struct {
	name  string // used as key in periodic stats
	count int64
	total float64

	sync.Mutex
}

func newDNSHistogram(name string) *histogram {
	return &histogram{
		name: name,
	}
}

func (h *histogram) ObserveWithTime(value float64, when time.Time) {
	statistics.PerSecond.Observe(h.name, when, value)
	statistics.PerMinute.Observe(h.name, when, value)
	statistics.PerHour.Observe(h.name, when, value)
	statistics.PerDay.Observe(h.name, when, value)
	h.Lock()
	h.count++
	h.total += value
	h.Unlock()
}

func (h *histogram) Observe(value float64) {
	h.ObserveWithTime(value, time.Now())
}

// -----
// stats
// -----
func incrementCounters(entry *logEntry) {
	requests.IncWithTime(entry.Time)
	if entry.Result.IsFiltered {
		filtered.IncWithTime(entry.Time)
	}

	switch entry.Result.Reason {
	case dnsfilter.NotFilteredWhiteList:
		whitelisted.IncWithTime(entry.Time)
	case dnsfilter.NotFilteredError:
		errorsTotal.IncWithTime(entry.Time)
	case dnsfilter.FilteredBlackList:
		filteredLists.IncWithTime(entry.Time)
	case dnsfilter.FilteredSafeBrowsing:
		filteredSafebrowsing.IncWithTime(entry.Time)
	case dnsfilter.FilteredParental:
		filteredParental.IncWithTime(entry.Time)
	case dnsfilter.FilteredInvalid:
		// do nothing
	case dnsfilter.FilteredSafeSearch:
		safesearch.IncWithTime(entry.Time)
	}
	elapsedTime.ObserveWithTime(entry.Elapsed.Seconds(), entry.Time)
}

// HandleStats returns aggregated stats data for the 24 hours
func HandleStats(w http.ResponseWriter, r *http.Request) {
	const numHours = 24
	histrical := generateMapFromStats(&statistics.PerHour, 0, numHours)
	// sum them up
	summed := map[string]interface{}{}
	for key, values := range histrical {
		summedValue := 0.0
		floats, ok := values.([]float64)
		if !ok {
			continue
		}
		for _, v := range floats {
			summedValue += v
		}
		summed[key] = summedValue
	}
	// don't forget to divide by number of elements in returned slice
	if val, ok := summed["avg_processing_time"]; ok {
		if flval, flok := val.(float64); flok {
			flval /= numHours
			summed["avg_processing_time"] = flval
		}
	}

	summed["stats_period"] = "24 hours"

	json, err := json.Marshal(summed)
	if err != nil {
		errorText := fmt.Sprintf("Unable to marshal status json: %s", err)
		log.Println(errorText)
		http.Error(w, errorText, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(json)
	if err != nil {
		errorText := fmt.Sprintf("Unable to write response json: %s", err)
		log.Println(errorText)
		http.Error(w, errorText, 500)
		return
	}
}

func generateMapFromStats(stats *periodicStats, start int, end int) map[string]interface{} {
	// clamp
	start = clamp(start, 0, statsHistoryElements)
	end = clamp(end, 0, statsHistoryElements)

	avgProcessingTime := make([]float64, 0)

	count := getReversedSlice(stats.Entries[elapsedTime.name+"_count"], start, end)
	sum := getReversedSlice(stats.Entries[elapsedTime.name+"_sum"], start, end)
	for i := 0; i < len(count); i++ {
		var avg float64
		if count[i] != 0 {
			avg = sum[i] / count[i]
			avg *= 1000
		}
		avgProcessingTime = append(avgProcessingTime, avg)
	}

	result := map[string]interface{}{
		"dns_queries":           getReversedSlice(stats.Entries[requests.name], start, end),
		"blocked_filtering":     getReversedSlice(stats.Entries[filtered.name], start, end),
		"replaced_safebrowsing": getReversedSlice(stats.Entries[filteredSafebrowsing.name], start, end),
		"replaced_safesearch":   getReversedSlice(stats.Entries[safesearch.name], start, end),
		"replaced_parental":     getReversedSlice(stats.Entries[filteredParental.name], start, end),
		"avg_processing_time":   avgProcessingTime,
	}
	return result
}

// HandleStatsHistory returns historical stats data for the 24 hours
func HandleStatsHistory(w http.ResponseWriter, r *http.Request) {
	// handle time unit and prepare our time window size
	now := time.Now()
	timeUnitString := r.URL.Query().Get("time_unit")
	var stats *periodicStats
	var timeUnit time.Duration
	switch timeUnitString {
	case "seconds":
		timeUnit = time.Second
		stats = &statistics.PerSecond
	case "minutes":
		timeUnit = time.Minute
		stats = &statistics.PerMinute
	case "hours":
		timeUnit = time.Hour
		stats = &statistics.PerHour
	case "days":
		timeUnit = time.Hour * 24
		stats = &statistics.PerDay
	default:
		http.Error(w, "Must specify valid time_unit parameter", 400)
		return
	}

	// parse start and end time
	startTime, err := time.Parse(time.RFC3339, r.URL.Query().Get("start_time"))
	if err != nil {
		errorText := fmt.Sprintf("Must specify valid start_time parameter: %s", err)
		log.Println(errorText)
		http.Error(w, errorText, 400)
		return
	}
	endTime, err := time.Parse(time.RFC3339, r.URL.Query().Get("end_time"))
	if err != nil {
		errorText := fmt.Sprintf("Must specify valid end_time parameter: %s", err)
		log.Println(errorText)
		http.Error(w, errorText, 400)
		return
	}

	// check if start and time times are within supported time range
	timeRange := timeUnit * statsHistoryElements
	if startTime.Add(timeRange).Before(now) {
		http.Error(w, "start_time parameter is outside of supported range", 501)
		return
	}
	if endTime.Add(timeRange).Before(now) {
		http.Error(w, "end_time parameter is outside of supported range", 501)
		return
	}

	// calculate start and end of our array
	// basically it's how many hours/minutes/etc have passed since now
	start := int(now.Sub(endTime) / timeUnit)
	end := int(now.Sub(startTime) / timeUnit)

	// swap them around if they're inverted
	if start > end {
		start, end = end, start
	}

	data := generateMapFromStats(stats, start, end)
	json, err := json.Marshal(data)
	if err != nil {
		errorText := fmt.Sprintf("Unable to marshal status json: %s", err)
		log.Println(errorText)
		http.Error(w, errorText, 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(json)
	if err != nil {
		errorText := fmt.Sprintf("Unable to write response json: %s", err)
		log.Println(errorText)
		http.Error(w, errorText, 500)
		return
	}
}

// HandleStatsReset resets the stats caches
func HandleStatsReset(w http.ResponseWriter, r *http.Request) {
	purgeStats()
	_, err := fmt.Fprintf(w, "OK\n")
	if err != nil {
		errorText := fmt.Sprintf("Couldn't write body: %s", err)
		log.Println(errorText)
		http.Error(w, errorText, http.StatusInternalServerError)
	}
}

func clamp(value, low, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
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
