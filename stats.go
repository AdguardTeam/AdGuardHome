package main

import (
	"bufio"
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
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
	filteredSafebrowsing = `coredns_dnsfilter_filtered_safebrowsing_total`
	filteredSafesearch   = `coredns_dnsfilter_safesearch_total`
	filteredParental     = `coredns_dnsfilter_filtered_parental_total`
	processingTimeSum    = `coredns_dns_request_duration_seconds_sum`
	processingTimeCount  = `coredns_dns_request_duration_seconds_count`
)

var entryWhiteList = map[string]bool{
	totalRequests:        true,
	filteredTotal:        true,
	filteredSafebrowsing: true,
	filteredSafesearch:   true,
	filteredParental:     true,
	processingTimeSum:    true,
	processingTimeCount:  true,
}

type periodicStats struct {
	Entries    statsEntries
	LastRotate time.Time // last time this data was rotated
}

type stats struct {
	PerSecond periodicStats
	PerMinute periodicStats
	PerHour   periodicStats
	PerDay    periodicStats

	LastSeen statsEntry
}

var statistics stats

func initPeriodicStats(periodic *periodicStats) {
	periodic.Entries = statsEntries{}
	periodic.LastRotate = time.Time{}
}

func init() {
	purgeStats()
}

func purgeStats() {
	initPeriodicStats(&statistics.PerSecond)
	initPeriodicStats(&statistics.PerMinute)
	initPeriodicStats(&statistics.PerHour)
	initPeriodicStats(&statistics.PerDay)
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

func statsRotate(periodic *periodicStats, now time.Time, rotations int64) {
	// calculate how many times we should rotate
	for r := int64(0); r < rotations; r++ {
		for key, values := range periodic.Entries {
			newValues := [statsHistoryElements]float64{}
			for i := 1; i < len(values); i++ {
				newValues[i] = values[i-1]
			}
			periodic.Entries[key] = newValues
		}
	}
	if rotations > 0 {
		periodic.LastRotate = now
	}
}

// called every second, accumulates stats for each second, minute, hour and day
func collectStats() {
	now := time.Now()
	statsRotate(&statistics.PerSecond, now, int64(now.Sub(statistics.PerSecond.LastRotate)/time.Second))
	statsRotate(&statistics.PerMinute, now, int64(now.Sub(statistics.PerMinute.LastRotate)/time.Minute))
	statsRotate(&statistics.PerHour, now, int64(now.Sub(statistics.PerHour.LastRotate)/time.Hour))
	statsRotate(&statistics.PerDay, now, int64(now.Sub(statistics.PerDay.LastRotate)/time.Hour/24))

	// grab HTTP from prometheus
	resp, err := client.Get("http://127.0.0.1:9153/metrics")
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		if isConnRefused(err) {
			return
		}
		log.Printf("Couldn't get coredns metrics: %T %s\n", err, err)
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

		// keys not in whitelist are not ok
		if entryWhiteList[key] == false {
			continue
		}

		got, ok := entry[key]
		if ok {
			value += got
		}
		entry[key] = value
	}

	// calculate delta
	delta := calcDelta(entry, statistics.LastSeen)

	// apply delta to second/minute/hour/day
	applyDelta(&statistics.PerSecond, delta)
	applyDelta(&statistics.PerMinute, delta)
	applyDelta(&statistics.PerHour, delta)
	applyDelta(&statistics.PerDay, delta)

	// save last seen
	statistics.LastSeen = entry
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
		currentValues := current.Entries[key]
		currentValues[0] += deltaValue
		current.Entries[key] = currentValues
	}
}

func loadStats() error {
	statsFile := filepath.Join(config.ourBinaryDir, "stats.json")
	if _, err := os.Stat(statsFile); os.IsNotExist(err) {
		log.Printf("Stats JSON does not exist, skipping: %s", statsFile)
		return nil
	}
	log.Printf("Loading JSON stats: %s", statsFile)
	jsonText, err := ioutil.ReadFile(statsFile)
	if err != nil {
		log.Printf("Couldn't read JSON stats: %s", err)
		return err
	}
	err = json.Unmarshal(jsonText, &statistics)
	if err != nil {
		log.Printf("Couldn't parse JSON stats: %s", err)
		return err
	}

	return nil
}

func writeStats() error {
	statsFile := filepath.Join(config.ourBinaryDir, "stats.json")
	log.Printf("Writing JSON file: %s", statsFile)
	json, err := json.MarshalIndent(statistics, "", "  ")
	if err != nil {
		log.Printf("Couldn't generate JSON: %s", err)
		return err
	}
	err = ioutil.WriteFile(statsFile+".tmp", json, 0644)
	if err != nil {
		log.Printf("Couldn't write stats in JSON: %s", err)
		return err
	}
	err = os.Rename(statsFile+".tmp", statsFile)
	if err != nil {
		log.Printf("Couldn't rename stats JSON: %s", err)
		return err
	}
	return nil
}
