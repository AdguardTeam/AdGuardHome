package dnsforward

import (
	"fmt"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/dnsfilter"
)

// how far back to keep the stats
const statsHistoryElements = 60 + 1 // +1 for calculating delta

// entries for single time period (for example all per-second entries)
type statsEntries map[string][statsHistoryElements]float64

// each periodic stat is a map of arrays
type periodicStats struct {
	entries    statsEntries
	period     time.Duration // how long one entry lasts
	lastRotate time.Time     // last time this data was rotated

	sync.RWMutex
}

// stats is the DNS server historical statistics
type stats struct {
	perSecond periodicStats
	perMinute periodicStats
	perHour   periodicStats
	perDay    periodicStats

	requests             *counter   // total number of requests
	filtered             *counter   // total number of filtered requests
	filteredLists        *counter   // total number of requests blocked by filter lists
	filteredSafebrowsing *counter   // total number of requests blocked by safebrowsing
	filteredParental     *counter   // total number of requests blocked by the parental control
	whitelisted          *counter   // total number of requests whitelisted by filter lists
	safesearch           *counter   // total number of requests for which safe search rules were applied
	errorsTotal          *counter   // total number of errors
	elapsedTime          *histogram // requests duration histogram
}

// initializes an empty stats structure
func newStats() *stats {
	s := &stats{
		requests:             newDNSCounter("requests_total"),
		filtered:             newDNSCounter("filtered_total"),
		filteredLists:        newDNSCounter("filtered_lists_total"),
		filteredSafebrowsing: newDNSCounter("filtered_safebrowsing_total"),
		filteredParental:     newDNSCounter("filtered_parental_total"),
		whitelisted:          newDNSCounter("whitelisted_total"),
		safesearch:           newDNSCounter("safesearch_total"),
		errorsTotal:          newDNSCounter("errors_total"),
		elapsedTime:          newDNSHistogram("request_duration"),
	}

	// Initializes empty per-sec/minute/hour/day stats
	s.purgeStats()
	return s
}

func initPeriodicStats(periodic *periodicStats, period time.Duration) {
	periodic.entries = statsEntries{}
	periodic.lastRotate = time.Now()
	periodic.period = period
}

func (s *stats) purgeStats() {
	initPeriodicStats(&s.perSecond, time.Second)
	initPeriodicStats(&s.perMinute, time.Minute)
	initPeriodicStats(&s.perHour, time.Hour)
	initPeriodicStats(&s.perDay, time.Hour*24)
}

func (p *periodicStats) Inc(name string, when time.Time) {
	// calculate how many periods ago this happened
	elapsed := int64(time.Since(when) / p.period)
	// log.Tracef("%s: %v as %v -> [%v]", name, time.Since(when), p.period, elapsed)
	if elapsed >= statsHistoryElements {
		return // outside of our timeframe
	}
	p.Lock()
	currentValues := p.entries[name]
	currentValues[elapsed]++
	p.entries[name] = currentValues
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
		currentValues := p.entries[countname]
		v := currentValues[elapsed]
		// log.Tracef("Will change p.entries[%s][%d] from %v to %v", countname, elapsed, value, value+1)
		v++
		currentValues[elapsed] = v
		p.entries[countname] = currentValues
	}
	{
		totalname := name + "_sum"
		currentValues := p.entries[totalname]
		currentValues[elapsed] += value
		p.entries[totalname] = currentValues
	}
	p.Unlock()
}

func (p *periodicStats) statsRotate(now time.Time) {
	p.Lock()
	rotations := int64(now.Sub(p.lastRotate) / p.period)
	if rotations > statsHistoryElements {
		rotations = statsHistoryElements
	}
	// calculate how many times we should rotate
	for r := int64(0); r < rotations; r++ {
		for key, values := range p.entries {
			newValues := [statsHistoryElements]float64{}
			for i := 1; i < len(values); i++ {
				newValues[i] = values[i-1]
			}
			p.entries[key] = newValues
		}
	}
	if rotations > 0 {
		p.lastRotate = now
	}
	p.Unlock()
}

func (s *stats) statsRotator() {
	for range time.Tick(time.Second) {
		now := time.Now()
		s.perSecond.statsRotate(now)
		s.perMinute.statsRotate(now)
		s.perHour.statsRotate(now)
		s.perDay.statsRotate(now)
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

func (s *stats) incWithTime(c *counter, when time.Time) {
	s.perSecond.Inc(c.name, when)
	s.perMinute.Inc(c.name, when)
	s.perHour.Inc(c.name, when)
	s.perDay.Inc(c.name, when)
	c.Lock()
	c.value++
	c.Unlock()
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

func (s *stats) observeWithTime(h *histogram, value float64, when time.Time) {
	s.perSecond.Observe(h.name, when, value)
	s.perMinute.Observe(h.name, when, value)
	s.perHour.Observe(h.name, when, value)
	s.perDay.Observe(h.name, when, value)
	h.Lock()
	h.count++
	h.total += value
	h.Unlock()
}

// -----
// stats
// -----
func (s *stats) incrementCounters(entry *logEntry) {
	s.incWithTime(s.requests, entry.Time)
	if entry.Result.IsFiltered {
		s.incWithTime(s.filtered, entry.Time)
	}

	switch entry.Result.Reason {
	case dnsfilter.NotFilteredWhiteList:
		s.incWithTime(s.whitelisted, entry.Time)
	case dnsfilter.NotFilteredError:
		s.incWithTime(s.errorsTotal, entry.Time)
	case dnsfilter.FilteredBlackList:
		s.incWithTime(s.filteredLists, entry.Time)
	case dnsfilter.FilteredSafeBrowsing:
		s.incWithTime(s.filteredSafebrowsing, entry.Time)
	case dnsfilter.FilteredParental:
		s.incWithTime(s.filteredParental, entry.Time)
	case dnsfilter.FilteredInvalid:
		// do nothing
	case dnsfilter.FilteredSafeSearch:
		s.incWithTime(s.safesearch, entry.Time)
	}
	s.observeWithTime(s.elapsedTime, entry.Elapsed.Seconds(), entry.Time)
}

// getAggregatedStats returns aggregated stats data for the 24 hours
func (s *stats) getAggregatedStats() map[string]interface{} {
	const numHours = 24
	historical := s.generateMapFromStats(&s.perHour, 0, numHours)
	// sum them up
	summed := map[string]interface{}{}
	for key, values := range historical {
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
	return summed
}

func (s *stats) generateMapFromStats(stats *periodicStats, start int, end int) map[string]interface{} {
	// clamp
	start = clamp(start, 0, statsHistoryElements)
	end = clamp(end, 0, statsHistoryElements)

	avgProcessingTime := make([]float64, 0)

	count := getReversedSlice(stats.entries[s.elapsedTime.name+"_count"], start, end)
	sum := getReversedSlice(stats.entries[s.elapsedTime.name+"_sum"], start, end)
	for i := 0; i < len(count); i++ {
		var avg float64
		if count[i] != 0 {
			avg = sum[i] / count[i]
			avg *= 1000
		}
		avgProcessingTime = append(avgProcessingTime, avg)
	}

	result := map[string]interface{}{
		"dns_queries":           getReversedSlice(stats.entries[s.requests.name], start, end),
		"blocked_filtering":     getReversedSlice(stats.entries[s.filtered.name], start, end),
		"replaced_safebrowsing": getReversedSlice(stats.entries[s.filteredSafebrowsing.name], start, end),
		"replaced_safesearch":   getReversedSlice(stats.entries[s.safesearch.name], start, end),
		"replaced_parental":     getReversedSlice(stats.entries[s.filteredParental.name], start, end),
		"avg_processing_time":   avgProcessingTime,
	}
	return result
}

// getStatsHistory gets stats history aggregated by the specified time unit
// timeUnit is either time.Second, time.Minute, time.Hour, or 24*time.Hour
// start is start of the time range
// end is end of the time range
// returns nil if time unit is not supported
func (s *stats) getStatsHistory(timeUnit time.Duration, startTime time.Time, endTime time.Time) (map[string]interface{}, error) {
	var stats *periodicStats

	switch timeUnit {
	case time.Second:
		stats = &s.perSecond
	case time.Minute:
		stats = &s.perMinute
	case time.Hour:
		stats = &s.perHour
	case 24 * time.Hour:
		stats = &s.perDay
	}

	if stats == nil {
		return nil, fmt.Errorf("unsupported time unit: %v", timeUnit)
	}

	now := time.Now()

	// check if start and time times are within supported time range
	timeRange := timeUnit * statsHistoryElements
	if startTime.Add(timeRange).Before(now) {
		return nil, fmt.Errorf("start_time parameter is outside of supported range: %s", startTime.String())
	}
	if endTime.Add(timeRange).Before(now) {
		return nil, fmt.Errorf("end_time parameter is outside of supported range: %s", startTime.String())
	}

	// calculate start and end of our array
	// basically it's how many hours/minutes/etc have passed since now
	start := int(now.Sub(endTime) / timeUnit)
	end := int(now.Sub(startTime) / timeUnit)

	// swap them around if they're inverted
	if start > end {
		start, end = end, start
	}

	return s.generateMapFromStats(stats, start, end), nil
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
