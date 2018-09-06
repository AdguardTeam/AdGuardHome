package main

import (
	"bufio"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var client = &http.Client{
	Timeout: time.Second * 30,
}

// as seen over HTTP
type statsEntry map[string]float64
type statsEntries map[string][statsHistoryElements]float64

const (
	statsHistoryElements = 60 + 1 // +1 for calculating delta
	totalRequests        = `coredns_dns_request_count_total`
	filteredTotal        = `coredns_dnsfilter_filtered_total`
	filteredLists        = `coredns_dnsfilter_filtered_lists_total`
	filteredSafebrowsing = `coredns_dnsfilter_filtered_safebrowsing_total`
	filteredSafesearch   = `coredns_dnsfilter_safesearch_total`
	filteredParental     = `coredns_dnsfilter_filtered_parental_total`
	processingTimeSum    = `coredns_dns_request_duration_seconds_sum`
	processingTimeCount  = `coredns_dns_request_duration_seconds_count`
)

type periodicStats struct {
	entries    statsEntries
	lastRotate time.Time // last time this data was rotated
}

type stats struct {
	perSecond periodicStats
	perMinute periodicStats
	perHour   periodicStats
	perDay    periodicStats

	lastSeen statsEntry
}

var statistics stats

func initPeriodicStats(periodic *periodicStats) {
	periodic.entries = statsEntries{}
}

func init() {
	initPeriodicStats(&statistics.perSecond)
	initPeriodicStats(&statistics.perMinute)
	initPeriodicStats(&statistics.perHour)
	initPeriodicStats(&statistics.perDay)
}

func runStatsCollectors() {
	go statsCollector(time.Second)
}

func statsCollector(t time.Duration) {
	for range time.Tick(t) {
		collectStats()
	}
}

func isConnRefused(err error) bool {
	if err != nil {
		if uerr, ok := err.(*url.Error); ok {
			if noerr, ok := uerr.Err.(*net.OpError); ok {
				if scerr, ok := noerr.Err.(*os.SyscallError); ok {
					if scerr.Err == syscall.ECONNREFUSED {
						return true
					}
				}
			}
		}
	}
	return false
}

func statsRotate(periodic *periodicStats, now time.Time) {
	for key, values := range periodic.entries {
		newValues := [statsHistoryElements]float64{}
		for i := 1; i < len(values); i++ {
			newValues[i] = values[i-1]
		}
		periodic.entries[key] = newValues
	}
	periodic.lastRotate = now
}

// called every second, accumulates stats for each second, minute, hour and day
func collectStats() {
	now := time.Now()
	// rotate each second
	// NOTE: since we are called every second, always rotate perSecond, otherwise aliasing problems cause the rotation to skip
	if true {
		statsRotate(&statistics.perSecond, now)
	}
	// if minute elapsed, rotate
	if now.Sub(statistics.perMinute.lastRotate).Minutes() >= 1 {
		statsRotate(&statistics.perMinute, now)
	}
	// if hour elapsed, rotate
	if now.Sub(statistics.perHour.lastRotate).Hours() >= 1 {
		statsRotate(&statistics.perHour, now)
	}
	// if day elapsed, rotate
	if now.Sub(statistics.perDay.lastRotate).Hours()/24.0 >= 1 {
		statsRotate(&statistics.perDay, now)
	}

	// grab HTTP from prometheus
	resp, err := client.Get("http://127.0.0.1:9153/metrics")
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		if isConnRefused(err) == false {
			log.Printf("Couldn't get coredns metrics: %T %s\n", err, err)
		}
		return
	}

	// read the body entirely
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("Couldn't read response body:", err)
		return
	}

	entry := statsEntry{}

	// handle body
	scanner := bufio.NewScanner(strings.NewReader(string(body)))
	for scanner.Scan() {
		line := scanner.Text()
		// ignore comments
		if line[0] == '#' {
			continue
		}
		splitted := strings.Split(line, " ")
		if len(splitted) < 2 {
			continue
		}

		value, err := strconv.ParseFloat(splitted[1], 64)
		if err != nil {
			log.Printf("Failed to parse number input %s: %s", splitted[1], err)
			continue
		}

		key := splitted[0]
		index := strings.IndexByte(key, '{')
		if index >= 0 {
			key = key[:index]
		}

		// empty keys are not ok
		if key == "" {
			continue
		}

		got, ok := entry[key]
		if ok {
			value += got
		}
		entry[key] = value
	}

	// calculate delta
	delta := calcDelta(entry, statistics.lastSeen)

	// apply delta to second/minute/hour/day
	applyDelta(&statistics.perSecond, delta)
	applyDelta(&statistics.perMinute, delta)
	applyDelta(&statistics.perHour, delta)
	applyDelta(&statistics.perDay, delta)

	// save last seen
	statistics.lastSeen = entry
}

func calcDelta(current, seen statsEntry) statsEntry {
	delta := statsEntry{}
	for key, currentValue := range current {
		seenValue := seen[key]
		deltaValue := currentValue - seenValue
		delta[key] = deltaValue
	}
	return delta
}

func applyDelta(current *periodicStats, delta statsEntry) {
	for key, deltaValue := range delta {
		currentValues := current.entries[key]
		currentValues[0] += deltaValue
		current.entries[key] = currentValues
	}
}
