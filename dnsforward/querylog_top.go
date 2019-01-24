package dnsforward

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"path"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bluele/gcache"
	"github.com/hmage/golibs/log"
	"github.com/miekg/dns"
)

type hourTop struct {
	domains gcache.Cache
	blocked gcache.Cache
	clients gcache.Cache

	mutex sync.RWMutex
}

func (h *hourTop) init() {
	h.domains = gcache.New(queryLogTopSize).LRU().Build()
	h.blocked = gcache.New(queryLogTopSize).LRU().Build()
	h.clients = gcache.New(queryLogTopSize).LRU().Build()
}

type dayTop struct {
	hours     []*hourTop
	hoursLock sync.RWMutex // writelock this lock ONLY WHEN rotating or intializing hours!

	loaded     bool
	loadedLock sync.Mutex
}

var runningTop dayTop

func init() {
	runningTop.hoursWriteLock()
	for i := 0; i < 24; i++ {
		hour := hourTop{}
		hour.init()
		runningTop.hours = append(runningTop.hours, &hour)
	}
	runningTop.hoursWriteUnlock()
}

func rotateHourlyTop() {
	log.Printf("Rotating hourly top")
	hour := &hourTop{}
	hour.init()
	runningTop.hoursWriteLock()
	runningTop.hours = append([]*hourTop{hour}, runningTop.hours...)
	runningTop.hours = runningTop.hours[:24]
	runningTop.hoursWriteUnlock()
}

func periodicHourlyTopRotate() {
	t := time.Hour
	for range time.Tick(t) {
		rotateHourlyTop()
	}
}

func (h *hourTop) incrementValue(key string, cache gcache.Cache) error {
	h.Lock()
	defer h.Unlock()
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

func (h *hourTop) incrementDomains(key string) error {
	return h.incrementValue(key, h.domains)
}

func (h *hourTop) incrementBlocked(key string) error {
	return h.incrementValue(key, h.blocked)
}

func (h *hourTop) incrementClients(key string) error {
	return h.incrementValue(key, h.clients)
}

// if does not exist -- return 0
func (h *hourTop) lockedGetValue(key string, cache gcache.Cache) (int, error) {
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

func (h *hourTop) lockedGetDomains(key string) (int, error) {
	return h.lockedGetValue(key, h.domains)
}

func (h *hourTop) lockedGetBlocked(key string) (int, error) {
	return h.lockedGetValue(key, h.blocked)
}

func (h *hourTop) lockedGetClients(key string) (int, error) {
	return h.lockedGetValue(key, h.clients)
}

func (d *dayTop) addEntry(entry *logEntry, q *dns.Msg, now time.Time) error {
	// figure out which hour bucket it belongs to
	hour := int(now.Sub(entry.Time).Hours())
	if hour >= 24 {
		log.Printf("t %v is >24 hours ago, ignoring", entry.Time)
		return nil
	}

	// if a DNS query doesn't have questions, do nothing
	if len(q.Question) == 0 {
		return nil
	}

	hostname := strings.ToLower(strings.TrimSuffix(q.Question[0].Name, "."))

	// get value, if not set, crate one
	runningTop.hoursReadLock()
	defer runningTop.hoursReadUnlock()
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

func fillStatsFromQueryLog() error {
	now := time.Now()
	runningTop.loadedWriteLock()
	defer runningTop.loadedWriteUnlock()
	if runningTop.loaded {
		return nil
	}
	onEntry := func(entry *logEntry) error {
		if len(entry.Question) == 0 {
			log.Printf("entry question is absent, skipping")
			return nil
		}

		if entry.Time.After(now) {
			log.Printf("t %v vs %v is in the future, ignoring", entry.Time, now)
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

		err := runningTop.addEntry(entry, q, now)
		if err != nil {
			log.Printf("Failed to add entry to running top: %s", err)
			return err
		}

		queryLogLock.Lock()
		queryLogCache = append(queryLogCache, entry)
		if len(queryLogCache) > queryLogSize {
			toremove := len(queryLogCache) - queryLogSize
			queryLogCache = queryLogCache[toremove:]
		}
		queryLogLock.Unlock()

		incrementCounters(entry)

		return nil
	}

	needMore := func() bool { return true }
	err := genericLoader(onEntry, needMore, queryLogTimeLimit)
	if err != nil {
		log.Printf("Failed to load entries from querylog: %s", err)
		return err
	}

	runningTop.loaded = true

	return nil
}

// HandleStatsTop returns the current top stats
func HandleStatsTop(w http.ResponseWriter, r *http.Request) {
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

	runningTop.hoursReadLock()
	for hour := 0; hour < 24; hour++ {
		runningTop.hours[hour].RLock()
		do(runningTop.hours[hour].domains.Keys(), runningTop.hours[hour].lockedGetDomains, domains)
		do(runningTop.hours[hour].blocked.Keys(), runningTop.hours[hour].lockedGetBlocked, blocked)
		do(runningTop.hours[hour].clients.Keys(), runningTop.hours[hour].lockedGetClients, clients)
		runningTop.hours[hour].RUnlock()
	}
	runningTop.hoursReadUnlock()

	// use manual json marshalling because we want maps to be sorted by value
	json := bytes.Buffer{}
	json.WriteString("{\n")

	gen := func(json *bytes.Buffer, name string, top map[string]int, addComma bool) {
		json.WriteString("  ")
		json.WriteString(fmt.Sprintf("%q", name))
		json.WriteString(": {\n")
		sorted := sortByValue(top)
		// no more than 50 entries
		if len(sorted) > 50 {
			sorted = sorted[:50]
		}
		for i, key := range sorted {
			json.WriteString("    ")
			json.WriteString(fmt.Sprintf("%q", key))
			json.WriteString(": ")
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
		errorText := fmt.Sprintf("Couldn't write body: %s", err)
		log.Println(errorText)
		http.Error(w, errorText, http.StatusInternalServerError)
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

func (d *dayTop) hoursWriteLock()    { tracelock(); d.hoursLock.Lock() }
func (d *dayTop) hoursWriteUnlock()  { tracelock(); d.hoursLock.Unlock() }
func (d *dayTop) hoursReadLock()     { tracelock(); d.hoursLock.RLock() }
func (d *dayTop) hoursReadUnlock()   { tracelock(); d.hoursLock.RUnlock() }
func (d *dayTop) loadedWriteLock()   { tracelock(); d.loadedLock.Lock() }
func (d *dayTop) loadedWriteUnlock() { tracelock(); d.loadedLock.Unlock() }

func (h *hourTop) Lock()    { tracelock(); h.mutex.Lock() }
func (h *hourTop) RLock()   { tracelock(); h.mutex.RLock() }
func (h *hourTop) RUnlock() { tracelock(); h.mutex.RUnlock() }
func (h *hourTop) Unlock()  { tracelock(); h.mutex.Unlock() }

func tracelock() {
	if false { // not commented out to make code checked during compilation
		pc := make([]uintptr, 10) // at least 1 entry needed
		runtime.Callers(2, pc)
		f := path.Base(runtime.FuncForPC(pc[1]).Name())
		lockf := path.Base(runtime.FuncForPC(pc[0]).Name())
		fmt.Fprintf(os.Stderr, "%s(): %s\n", f, lockf)
	}
}
