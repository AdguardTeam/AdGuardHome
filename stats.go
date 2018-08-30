package main

import (
	"bufio"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type periodicStats struct {
	totalRequests []float64

	filteredTotal        []float64
	filteredLists        []float64
	filteredSafebrowsing []float64
	filteredSafesearch   []float64
	filteredParental     []float64

	processingTimeSum   []float64
	processingTimeCount []float64

	lastRotate time.Time // last time this data was rotated
}

type statsSnapshot struct {
	totalRequests float64

	filteredTotal        float64
	filteredLists        float64
	filteredSafebrowsing float64
	filteredSafesearch   float64
	filteredParental     float64

	processingTimeSum   float64
	processingTimeCount float64
}

type statsCollection struct {
	perSecond periodicStats
	perMinute periodicStats
	perHour   periodicStats
	perDay    periodicStats
	lastsnap  statsSnapshot
}

var statistics statsCollection

var client = &http.Client{
	Timeout: time.Second * 30,
}

const statsHistoryElements = 60 + 1 // +1 for calculating delta

var requestCountTotalRegex = regexp.MustCompile(`^coredns_dns_request_count_total`)
var requestDurationSecondsSum = regexp.MustCompile(`^coredns_dns_request_duration_seconds_sum`)
var requestDurationSecondsCount = regexp.MustCompile(`^coredns_dns_request_duration_seconds_count`)

func initPeriodicStats(stats *periodicStats) {
	stats.totalRequests = make([]float64, statsHistoryElements)
	stats.filteredTotal = make([]float64, statsHistoryElements)
	stats.filteredLists = make([]float64, statsHistoryElements)
	stats.filteredSafebrowsing = make([]float64, statsHistoryElements)
	stats.filteredSafesearch = make([]float64, statsHistoryElements)
	stats.filteredParental = make([]float64, statsHistoryElements)
	stats.processingTimeSum = make([]float64, statsHistoryElements)
	stats.processingTimeCount = make([]float64, statsHistoryElements)
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

func sliceRotate(slice *[]float64) {
	a := (*slice)[:len(*slice)-1]
	*slice = append([]float64{0}, a...)
}

func statsRotate(stats *periodicStats, now time.Time) {
	sliceRotate(&stats.totalRequests)
	sliceRotate(&stats.filteredTotal)
	sliceRotate(&stats.filteredLists)
	sliceRotate(&stats.filteredSafebrowsing)
	sliceRotate(&stats.filteredSafesearch)
	sliceRotate(&stats.filteredParental)
	sliceRotate(&stats.processingTimeSum)
	sliceRotate(&stats.processingTimeCount)
	stats.lastRotate = now
}

func handleValue(input string, target *float64) {
	value, err := strconv.ParseFloat(input, 64)
	if err != nil {
		log.Println("Failed to parse number input:", err)
		return
	}
	*target = value
}

// called every second, accumulates stats for each second, minute, hour and day
func collectStats() {
	now := time.Now()
	// rotate each second
	// NOTE: since we are called every second, always rotate, otherwise aliasing problems cause the rotation to skip
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

	// handle body
	scanner := bufio.NewScanner(strings.NewReader(string(body)))
	for scanner.Scan() {
		line := scanner.Text()
		// ignore comments
		if line[0] == '#' {
			continue
		}
		splitted := strings.Split(line, " ")
		switch {
		case splitted[0] == "coredns_dnsfilter_filtered_total":
			handleValue(splitted[1], &statistics.lastsnap.filteredTotal)
		case splitted[0] == "coredns_dnsfilter_filtered_lists_total":
			handleValue(splitted[1], &statistics.lastsnap.filteredLists)
		case splitted[0] == "coredns_dnsfilter_filtered_safebrowsing_total":
			handleValue(splitted[1], &statistics.lastsnap.filteredSafebrowsing)
		case splitted[0] == "coredns_dnsfilter_filtered_parental_total":
			handleValue(splitted[1], &statistics.lastsnap.filteredParental)
		case requestCountTotalRegex.MatchString(splitted[0]):
			handleValue(splitted[1], &statistics.lastsnap.totalRequests)
		case requestDurationSecondsSum.MatchString(splitted[0]):
			handleValue(splitted[1], &statistics.lastsnap.processingTimeSum)
		case requestDurationSecondsCount.MatchString(splitted[0]):
			handleValue(splitted[1], &statistics.lastsnap.processingTimeCount)
		}
	}

	// put the snap into per-second, per-minute, per-hour and per-day
	assignSnapToStats(&statistics.perSecond)
	assignSnapToStats(&statistics.perMinute)
	assignSnapToStats(&statistics.perHour)
	assignSnapToStats(&statistics.perDay)
}

func assignSnapToStats(stats *periodicStats) {
	stats.totalRequests[0] = statistics.lastsnap.totalRequests
	stats.filteredTotal[0] = statistics.lastsnap.filteredTotal
	stats.filteredLists[0] = statistics.lastsnap.filteredLists
	stats.filteredSafebrowsing[0] = statistics.lastsnap.filteredSafebrowsing
	stats.filteredSafesearch[0] = statistics.lastsnap.filteredSafesearch
	stats.filteredParental[0] = statistics.lastsnap.filteredParental
	stats.processingTimeSum[0] = statistics.lastsnap.processingTimeSum
	stats.processingTimeCount[0] = statistics.lastsnap.processingTimeCount
}
