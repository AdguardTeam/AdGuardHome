package dnsfilter

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/AdguardTeam/golibs/cache"
	"github.com/AdguardTeam/golibs/log"
)

/*
expire byte[4]
res Result
*/
func (d *Dnsfilter) setCacheResult(cache cache.Cache, host string, res Result) int {
	var buf bytes.Buffer

	expire := uint(time.Now().Unix()) + d.Config.CacheTime*60
	var exp []byte
	exp = make([]byte, 4)
	binary.BigEndian.PutUint32(exp, uint32(expire))
	_, _ = buf.Write(exp)

	enc := gob.NewEncoder(&buf)
	err := enc.Encode(res)
	if err != nil {
		log.Error("gob.Encode(): %s", err)
		return 0
	}
	val := buf.Bytes()
	_ = cache.Set([]byte(host), val)
	return len(val)
}

func getCachedResult(cache cache.Cache, host string) (Result, bool) {
	data := cache.Get([]byte(host))
	if data == nil {
		return Result{}, false
	}

	exp := int(binary.BigEndian.Uint32(data[:4]))
	if exp <= int(time.Now().Unix()) {
		cache.Del([]byte(host))
		return Result{}, false
	}

	var buf bytes.Buffer
	buf.Write(data[4:])
	dec := gob.NewDecoder(&buf)
	r := Result{}
	err := dec.Decode(&r)
	if err != nil {
		log.Debug("gob.Decode(): %s", err)
		return Result{}, false
	}

	return r, true
}

// SafeSearchDomain returns replacement address for search engine
func (d *Dnsfilter) SafeSearchDomain(host string) (string, bool) {
	val, ok := safeSearchDomains[host]
	return val, ok
}

func (d *Dnsfilter) checkSafeSearch(host string) (Result, error) {
	if log.GetLevel() >= log.DEBUG {
		timer := log.StartTimer()
		defer timer.LogElapsed("SafeSearch: lookup for %s", host)
	}

	// Check cache. Return cached result if it was found
	cachedValue, isFound := getCachedResult(gctx.safeSearchCache, host)
	if isFound {
		// atomic.AddUint64(&gctx.stats.Safesearch.CacheHits, 1)
		log.Tracef("SafeSearch: found in cache: %s", host)
		return cachedValue, nil
	}

	safeHost, ok := d.SafeSearchDomain(host)
	if !ok {
		return Result{}, nil
	}

	res := Result{IsFiltered: true, Reason: FilteredSafeSearch}
	if ip := net.ParseIP(safeHost); ip != nil {
		res.IP = ip
		valLen := d.setCacheResult(gctx.safeSearchCache, host, res)
		log.Debug("SafeSearch: stored in cache: %s (%d bytes)", host, valLen)
		return res, nil
	}

	// TODO this address should be resolved with upstream that was configured in dnsforward
	addrs, err := net.LookupIP(safeHost)
	if err != nil {
		log.Tracef("SafeSearchDomain for %s was found but failed to lookup for %s cause %s", host, safeHost, err)
		return Result{}, err
	}

	for _, i := range addrs {
		if ipv4 := i.To4(); ipv4 != nil {
			res.IP = ipv4
			break
		}
	}

	if len(res.IP) == 0 {
		return Result{}, fmt.Errorf("no ipv4 addresses in safe search response for %s", safeHost)
	}

	// Cache result
	valLen := d.setCacheResult(gctx.safeSearchCache, host, res)
	log.Debug("SafeSearch: stored in cache: %s (%d bytes)", host, valLen)
	return res, nil
}

func (d *Dnsfilter) handleSafeSearchEnable(w http.ResponseWriter, r *http.Request) {
	d.Config.SafeSearchEnabled = true
	d.Config.ConfigModified()
}

func (d *Dnsfilter) handleSafeSearchDisable(w http.ResponseWriter, r *http.Request) {
	d.Config.SafeSearchEnabled = false
	d.Config.ConfigModified()
}

func (d *Dnsfilter) handleSafeSearchStatus(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"enabled": d.Config.SafeSearchEnabled,
	}
	jsonVal, err := json.Marshal(data)
	if err != nil {
		httpError(r, w, http.StatusInternalServerError, "Unable to marshal status json: %s", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonVal)
	if err != nil {
		httpError(r, w, http.StatusInternalServerError, "Unable to write response json: %s", err)
		return
	}
}
