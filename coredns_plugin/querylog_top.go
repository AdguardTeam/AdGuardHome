package dnsfilter

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bluele/gcache"
	"github.com/miekg/dns"
)

// top domains/clients/blocked stats in the last 24 hours

// on start we read the saved stats from the last 24 hours and add them to the stats

// stats are counted using hourly LRU, rotating hourly and keeping last 24 hours

type hourTop struct {
	domains gcache.Cache
	blocked gcache.Cache
	clients gcache.Cache
	sync.RWMutex
}

func (top *hourTop) init() {
	top.domains = gcache.New(500).LRU().Build()
	top.blocked = gcache.New(500).LRU().Build()
	top.clients = gcache.New(500).LRU().Build()
}

type dayTop struct {
	hours        []*hourTop
	loaded       bool
	sync.RWMutex // write -- rotating hourTop, read -- anything else
}

var runningTop dayTop

func init() {
	runningTop.Lock()
	for i := 0; i < 24; i++ {
		hour := hourTop{}
		hour.init()
		runningTop.hours = append(runningTop.hours, &hour)
	}
	runningTop.Unlock()
}

func rotateHourlyTop() {
	log.Printf("Rotating hourly top")
	hour := &hourTop{}
	hour.init()
	runningTop.Lock()
	runningTop.hours = append([]*hourTop{hour}, runningTop.hours...)
	runningTop.hours = runningTop.hours[:24]
	runningTop.Unlock()
}

func periodicHourlyTopRotate() {
	t := time.Hour
	for range time.Tick(t) {
		rotateHourlyTop()
	}
}

func (top *hourTop) incrementValue(key string, cache gcache.Cache) error {
	top.Lock()
	defer top.Unlock()
	ivalue, err := cache.Get(key)
	if err == gcache.KeyNotFoundError {
		// we just set it and we're done
		err = cache.Set(key, 1)
		if err != nil {
			log.Printf("Failed to set hourly top value: %s", err)
			return err
		}
		return nil
	}

	if err != nil {
		log.Printf("gcache encountered an error during get: %s", err)
		return err
	}

	cachedValue, ok := ivalue.(int)
	if !ok {
		err = fmt.Errorf("SHOULD NOT HAPPEN: gcache has non-int as value: %v", ivalue)
		log.Println(err)
		return err
	}

	err = cache.Set(key, cachedValue+1)
	if err != nil {
		log.Printf("Failed to set hourly top value: %s", err)
		return err
	}
	return nil
}

func (top *hourTop) incrementDomains(key string) error {
	return top.incrementValue(key, top.domains)
}

func (top *hourTop) incrementBlocked(key string) error {
	return top.incrementValue(key, top.blocked)
}

func (top *hourTop) incrementClients(key string) error {
	return top.incrementValue(key, top.clients)
}

// if does not exist -- return 0
func (top *hourTop) lockedGetValue(key string, cache gcache.Cache) (int, error) {
	ivalue, err := cache.Get(key)
	if err == gcache.KeyNotFoundError {
		return 0, nil
	}

	if err != nil {
		log.Printf("gcache encountered an error during get: %s", err)
		return 0, err
	}

	value, ok := ivalue.(int)
	if !ok {
		err := fmt.Errorf("SHOULD NOT HAPPEN: gcache has non-int as value: %v", ivalue)
		log.Println(err)
		return 0, err
	}

	return value, nil
}

func (top *hourTop) lockedGetDomains(key string) (int, error) {
	return top.lockedGetValue(key, top.domains)
}

func (top *hourTop) lockedGetBlocked(key string) (int, error) {
	return top.lockedGetValue(key, top.blocked)
}

func (top *hourTop) lockedGetClients(key string) (int, error) {
	return top.lockedGetValue(key, top.clients)
}

func (r *dayTop) addEntry(entry *logEntry, now time.Time) error {
	if len(entry.Question) == 0 {
		log.Printf("entry question is absent, skipping")
		return nil
	}

	if entry.Time.After(now) {
		log.Printf("t %v vs %v is in the future, ignoring", entry.Time, now)
		return nil
	}
	// figure out which hour bucket it belongs to
	hour := int(now.Sub(entry.Time).Hours())
	if hour >= 24 {
		log.Printf("t %v is >24 hours ago, ignoring", entry.Time)
		return nil
	}

	q := new(dns.Msg)
	if err := q.Unpack(entry.Question); err != nil {
		log.Printf("failed to unpack dns message question: %s", err)
		return err
	}

	if len(q.Question) != 1 {
		log.Printf("malformed dns message, has no questions, skipping")
		return nil
	}

	hostname := strings.ToLower(strings.TrimSuffix(q.Question[0].Name, "."))

	// get value, if not set, crate one
	runningTop.RLock()
	defer runningTop.RUnlock()
	err := runningTop.hours[hour].incrementDomains(hostname)
	if err != nil {
		log.Printf("Failed to increment value: %s", err)
		return err
	}

	if entry.Result.IsFiltered {
		err := runningTop.hours[hour].incrementBlocked(hostname)
		if err != nil {
			log.Printf("Failed to increment value: %s", err)
			return err
		}
	}

	if len(entry.IP) > 0 {
		err := runningTop.hours[hour].incrementClients(entry.IP)
		if err != nil {
			log.Printf("Failed to increment value: %s", err)
			return err
		}
	}

	return nil
}

func loadTopFromFiles() error {
	now := time.Now()
	runningTop.RLock()
	if runningTop.loaded {
		return nil
	}
	defer runningTop.RUnlock()
	onEntry := func(entry *logEntry) error {
		err := runningTop.addEntry(entry, now)
		if err != nil {
			log.Printf("Failed to add entry to running top: %s", err)
			return err
		}
		return nil
	}

	needMore := func() bool { return true }
	err := genericLoader(onEntry, needMore, time.Hour*24)
	if err != nil {
		log.Printf("Failed to load entries from querylog: %s", err)
		return err
	}

	runningTop.loaded = true

	return nil
}

func handleStatsTop(w http.ResponseWriter, r *http.Request) {
	domains := map[string]int{}
	blocked := map[string]int{}
	clients := map[string]int{}

	do := func(keys []interface{}, getter func(key string) (int, error), result map[string]int) {
		for _, ikey := range keys {
			key, ok := ikey.(string)
			if !ok {
				continue
			}
			value, err := getter(key)
			if err != nil {
				log.Printf("Failed to get top domains value for %v: %s", key, err)
				return
			}
			result[key] += value
		}
	}

	runningTop.RLock()
	for hour := 0; hour < 24; hour++ {
		runningTop.hours[hour].RLock()
		do(runningTop.hours[hour].domains.Keys(), runningTop.hours[hour].lockedGetDomains, domains)
		do(runningTop.hours[hour].blocked.Keys(), runningTop.hours[hour].lockedGetBlocked, blocked)
		do(runningTop.hours[hour].clients.Keys(), runningTop.hours[hour].lockedGetClients, clients)
		runningTop.hours[hour].RUnlock()
	}
	runningTop.RUnlock()

	// use manual json marshalling because we want maps to be sorted by value
	json := bytes.Buffer{}
	json.WriteString("{\n")

	gen := func(json *bytes.Buffer, name string, top map[string]int, addComma bool) {
		json.WriteString("  \"")
		json.WriteString(name)
		json.WriteString("\": {\n")
		sorted := sortByValue(top)
		for i, key := range sorted {
			// no more than 50 entries
			if i >= 50 {
				break
			}
			json.WriteString("    \"")
			json.WriteString(key)
			json.WriteString("\": ")
			json.WriteString(strconv.Itoa(top[key]))
			if i+1 != len(sorted) {
				json.WriteByte(',')
			}
			json.WriteByte('\n')
		}
		json.WriteString("  }")
		if addComma {
			json.WriteByte(',')
		}
		json.WriteByte('\n')
	}
	gen(&json, "top_queried_domains", domains, true)
	gen(&json, "top_blocked_domains", blocked, true)
	gen(&json, "top_clients", clients, true)
	json.WriteString("  \"stats_period\": \"24 hours\"\n")
	json.WriteString("}\n")

	w.Header().Set("Content-Type", "application/json")
	_, err := w.Write(json.Bytes())
	if err != nil {
		errortext := fmt.Sprintf("Couldn't write body: %s", err)
		log.Println(errortext)
		http.Error(w, errortext, http.StatusInternalServerError)
	}
}

// helper function for querylog API
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
