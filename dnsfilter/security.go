// Parental Control, Safe Browsing, Safe Search

package dnsfilter

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/cache"
	"github.com/AdguardTeam/golibs/log"
	"github.com/miekg/dns"
	"golang.org/x/net/publicsuffix"
)

// Servers to use for resolution of SB/PC server name
var bootstrapServers = []string{"176.103.130.130", "176.103.130.131"}

const dnsTimeout = 3 * time.Second
const defaultSafebrowsingServer = "https://dns-family.adguard.com/dns-query"
const defaultParentalServer = "https://dns-family.adguard.com/dns-query"
const sbTXTSuffix = "sb.dns.adguard.com."
const pcTXTSuffix = "pc.dns.adguard.com."

func (d *Dnsfilter) initSecurityServices() error {
	var err error
	d.safeBrowsingServer = defaultSafebrowsingServer
	d.parentalServer = defaultParentalServer
	opts := upstream.Options{Timeout: dnsTimeout, Bootstrap: bootstrapServers}

	d.parentalUpstream, err = upstream.AddressToUpstream(d.parentalServer, opts)
	if err != nil {
		return err
	}

	d.safeBrowsingUpstream, err = upstream.AddressToUpstream(d.safeBrowsingServer, opts)
	if err != nil {
		return err
	}

	return nil
}

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

// for each dot, hash it and add it to string
// The maximum is 4 components: "a.b.c.d"
func hostnameToHashParam(host string) (string, map[string]bool) {
	var hashparam bytes.Buffer
	hashes := map[string]bool{}
	tld, icann := publicsuffix.PublicSuffix(host)
	if !icann {
		// private suffixes like cloudfront.net
		tld = ""
	}
	curhost := host

	nDots := 0
	for i := len(curhost) - 1; i >= 0; i-- {
		if curhost[i] == '.' {
			nDots++
			if nDots == 4 {
				curhost = curhost[i+1:] // "xxx.a.b.c.d" -> "a.b.c.d"
				break
			}
		}
	}

	for {
		if curhost == "" {
			// we've reached end of string
			break
		}
		if tld != "" && curhost == tld {
			// we've reached the TLD, don't hash it
			break
		}

		sum := sha256.Sum256([]byte(curhost))
		hashes[hex.EncodeToString(sum[:])] = true
		hashparam.WriteString(fmt.Sprintf("%s.", hex.EncodeToString(sum[0:4])))

		pos := strings.IndexByte(curhost, byte('.'))
		if pos < 0 {
			break
		}
		curhost = curhost[pos+1:]
	}
	return hashparam.String(), hashes
}

// Find the target hash in TXT response
func (d *Dnsfilter) processTXT(svc, host string, resp *dns.Msg, hashes map[string]bool) bool {
	for _, a := range resp.Answer {
		txt, ok := a.(*dns.TXT)
		if !ok {
			continue
		}
		log.Tracef("%s: hashes for %s: %v", svc, host, txt.Txt)
		for _, t := range txt.Txt {
			_, ok := hashes[t]
			if ok {
				log.Tracef("%s: matched %s by %s", svc, host, t)
				return true
			}
		}
	}
	return false
}

// Disabling "dupl": the algorithm of SB/PC is similar, but it uses different data
// nolint:dupl
func (d *Dnsfilter) checkSafeBrowsing(host string) (Result, error) {
	if log.GetLevel() >= log.DEBUG {
		timer := log.StartTimer()
		defer timer.LogElapsed("SafeBrowsing lookup for %s", host)
	}

	// check cache
	cachedValue, isFound := getCachedResult(gctx.safebrowsingCache, host)
	if isFound {
		// atomic.AddUint64(&gctx.stats.Safebrowsing.CacheHits, 1)
		log.Tracef("SafeBrowsing: found in cache: %s", host)
		return cachedValue, nil
	}

	result := Result{}
	question, hashes := hostnameToHashParam(host)
	question = question + sbTXTSuffix

	log.Tracef("SafeBrowsing: checking %s: %s", host, question)

	req := dns.Msg{}
	req.SetQuestion(question, dns.TypeTXT)
	resp, err := d.safeBrowsingUpstream.Exchange(&req)
	if err != nil {
		return result, err
	}

	if d.processTXT("SafeBrowsing", host, resp, hashes) {
		result.IsFiltered = true
		result.Reason = FilteredSafeBrowsing
		result.Rule = "adguard-malware-shavar"
	}

	valLen := d.setCacheResult(gctx.safebrowsingCache, host, result)
	log.Debug("SafeBrowsing: stored in cache: %s (%d bytes)", host, valLen)
	return result, nil
}

// Disabling "dupl": the algorithm of SB/PC is similar, but it uses different data
// nolint:dupl
func (d *Dnsfilter) checkParental(host string) (Result, error) {
	if log.GetLevel() >= log.DEBUG {
		timer := log.StartTimer()
		defer timer.LogElapsed("Parental lookup for %s", host)
	}

	// check cache
	cachedValue, isFound := getCachedResult(gctx.parentalCache, host)
	if isFound {
		// atomic.AddUint64(&gctx.stats.Parental.CacheHits, 1)
		log.Tracef("Parental: found in cache: %s", host)
		return cachedValue, nil
	}

	result := Result{}
	question, hashes := hostnameToHashParam(host)
	question = question + pcTXTSuffix

	log.Tracef("Parental: checking %s: %s", host, question)

	req := dns.Msg{}
	req.SetQuestion(question, dns.TypeTXT)
	resp, err := d.parentalUpstream.Exchange(&req)
	if err != nil {
		return result, err
	}

	if d.processTXT("Parental", host, resp, hashes) {
		result.IsFiltered = true
		result.Reason = FilteredParental
		result.Rule = "parental CATEGORY_BLACKLISTED"
	}

	valLen := d.setCacheResult(gctx.parentalCache, host, result)
	log.Debug("Parental: stored in cache: %s (%d bytes)", host, valLen)
	return result, err
}

func httpError(r *http.Request, w http.ResponseWriter, code int, format string, args ...interface{}) {
	text := fmt.Sprintf(format, args...)
	log.Info("DNSFilter: %s %s: %s", r.Method, r.URL, text)
	http.Error(w, text, code)
}

func (d *Dnsfilter) handleSafeBrowsingEnable(w http.ResponseWriter, r *http.Request) {
	d.Config.SafeBrowsingEnabled = true
	d.Config.ConfigModified()
}

func (d *Dnsfilter) handleSafeBrowsingDisable(w http.ResponseWriter, r *http.Request) {
	d.Config.SafeBrowsingEnabled = false
	d.Config.ConfigModified()
}

func (d *Dnsfilter) handleSafeBrowsingStatus(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"enabled": d.Config.SafeBrowsingEnabled,
	}
	jsonVal, err := json.Marshal(data)
	if err != nil {
		httpError(r, w, http.StatusInternalServerError, "Unable to marshal status json: %s", err)
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonVal)
	if err != nil {
		httpError(r, w, http.StatusInternalServerError, "Unable to write response json: %s", err)
		return
	}
}

func (d *Dnsfilter) handleParentalEnable(w http.ResponseWriter, r *http.Request) {
	d.Config.ParentalEnabled = true
	d.Config.ConfigModified()
}

func (d *Dnsfilter) handleParentalDisable(w http.ResponseWriter, r *http.Request) {
	d.Config.ParentalEnabled = false
	d.Config.ConfigModified()
}

func (d *Dnsfilter) handleParentalStatus(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"enabled": d.Config.ParentalEnabled,
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

func (d *Dnsfilter) registerSecurityHandlers() {
	d.Config.HTTPRegister("POST", "/control/safebrowsing/enable", d.handleSafeBrowsingEnable)
	d.Config.HTTPRegister("POST", "/control/safebrowsing/disable", d.handleSafeBrowsingDisable)
	d.Config.HTTPRegister("GET", "/control/safebrowsing/status", d.handleSafeBrowsingStatus)

	d.Config.HTTPRegister("POST", "/control/parental/enable", d.handleParentalEnable)
	d.Config.HTTPRegister("POST", "/control/parental/disable", d.handleParentalDisable)
	d.Config.HTTPRegister("GET", "/control/parental/status", d.handleParentalStatus)

	d.Config.HTTPRegister("POST", "/control/safesearch/enable", d.handleSafeSearchEnable)
	d.Config.HTTPRegister("POST", "/control/safesearch/disable", d.handleSafeSearchDisable)
	d.Config.HTTPRegister("GET", "/control/safesearch/status", d.handleSafeSearchStatus)
}
