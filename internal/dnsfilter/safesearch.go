package dnsfilter

import (
	"bytes"
	"context"
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
func (d *DNSFilter) setCacheResult(cache cache.Cache, host string, res Result) int {
	var buf bytes.Buffer

	expire := uint(time.Now().Unix()) + d.Config.CacheTime*60
	exp := make([]byte, 4)
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
func (d *DNSFilter) SafeSearchDomain(host string) (string, bool) {
	val, ok := safeSearchDomains[host]
	return val, ok
}

func (d *DNSFilter) checkSafeSearch(
	host string,
	_ uint16,
	setts *FilteringSettings,
) (res Result, err error) {
	if !setts.SafeSearchEnabled {
		return Result{}, nil
	}

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

	res = Result{
		IsFiltered: true,
		Reason:     FilteredSafeSearch,
		Rules:      []*ResultRule{{}},
	}

	if ip := net.ParseIP(safeHost); ip != nil {
		res.Rules[0].IP = ip
		valLen := d.setCacheResult(gctx.safeSearchCache, host, res)
		log.Debug("SafeSearch: stored in cache: %s (%d bytes)", host, valLen)

		return res, nil
	}

	ips, err := d.resolver.LookupIP(context.Background(), "ip", safeHost)
	if err != nil {
		log.Tracef("SafeSearchDomain for %s was found but failed to lookup for %s cause %s", host, safeHost, err)
		return Result{}, err
	}

	for _, ip := range ips {
		if ip = ip.To4(); ip == nil {
			continue
		}

		res.Rules[0].IP = ip

		l := d.setCacheResult(gctx.safeSearchCache, host, res)
		log.Debug("SafeSearch: stored in cache: %s (%d bytes)", host, l)

		return res, nil
	}

	return Result{}, fmt.Errorf("no ipv4 addresses in safe search response for %s", safeHost)
}

func (d *DNSFilter) handleSafeSearchEnable(w http.ResponseWriter, r *http.Request) {
	d.Config.SafeSearchEnabled = true
	d.Config.ConfigModified()
}

func (d *DNSFilter) handleSafeSearchDisable(w http.ResponseWriter, r *http.Request) {
	d.Config.SafeSearchEnabled = false
	d.Config.ConfigModified()
}

func (d *DNSFilter) handleSafeSearchStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(&struct {
		Enabled bool `json:"enabled"`
	}{
		Enabled: d.Config.SafeSearchEnabled,
	})
	if err != nil {
		httpError(r, w, http.StatusInternalServerError, "Unable to write response json: %s", err)
		return
	}
}

var safeSearchDomains = map[string]string{
	"yandex.com":     "213.180.193.56",
	"yandex.ru":      "213.180.193.56",
	"yandex.ua":      "213.180.193.56",
	"yandex.by":      "213.180.193.56",
	"yandex.kz":      "213.180.193.56",
	"www.yandex.com": "213.180.193.56",
	"www.yandex.ru":  "213.180.193.56",
	"www.yandex.ua":  "213.180.193.56",
	"www.yandex.by":  "213.180.193.56",
	"www.yandex.kz":  "213.180.193.56",

	"www.bing.com": "strict.bing.com",

	"duckduckgo.com":       "safe.duckduckgo.com",
	"www.duckduckgo.com":   "safe.duckduckgo.com",
	"start.duckduckgo.com": "safe.duckduckgo.com",

	"www.google.com":    "forcesafesearch.google.com",
	"www.google.ad":     "forcesafesearch.google.com",
	"www.google.ae":     "forcesafesearch.google.com",
	"www.google.com.af": "forcesafesearch.google.com",
	"www.google.com.ag": "forcesafesearch.google.com",
	"www.google.com.ai": "forcesafesearch.google.com",
	"www.google.al":     "forcesafesearch.google.com",
	"www.google.am":     "forcesafesearch.google.com",
	"www.google.co.ao":  "forcesafesearch.google.com",
	"www.google.com.ar": "forcesafesearch.google.com",
	"www.google.as":     "forcesafesearch.google.com",
	"www.google.at":     "forcesafesearch.google.com",
	"www.google.com.au": "forcesafesearch.google.com",
	"www.google.az":     "forcesafesearch.google.com",
	"www.google.ba":     "forcesafesearch.google.com",
	"www.google.com.bd": "forcesafesearch.google.com",
	"www.google.be":     "forcesafesearch.google.com",
	"www.google.bf":     "forcesafesearch.google.com",
	"www.google.bg":     "forcesafesearch.google.com",
	"www.google.com.bh": "forcesafesearch.google.com",
	"www.google.bi":     "forcesafesearch.google.com",
	"www.google.bj":     "forcesafesearch.google.com",
	"www.google.com.bn": "forcesafesearch.google.com",
	"www.google.com.bo": "forcesafesearch.google.com",
	"www.google.com.br": "forcesafesearch.google.com",
	"www.google.bs":     "forcesafesearch.google.com",
	"www.google.bt":     "forcesafesearch.google.com",
	"www.google.co.bw":  "forcesafesearch.google.com",
	"www.google.by":     "forcesafesearch.google.com",
	"www.google.com.bz": "forcesafesearch.google.com",
	"www.google.ca":     "forcesafesearch.google.com",
	"www.google.cd":     "forcesafesearch.google.com",
	"www.google.cf":     "forcesafesearch.google.com",
	"www.google.cg":     "forcesafesearch.google.com",
	"www.google.ch":     "forcesafesearch.google.com",
	"www.google.ci":     "forcesafesearch.google.com",
	"www.google.co.ck":  "forcesafesearch.google.com",
	"www.google.cl":     "forcesafesearch.google.com",
	"www.google.cm":     "forcesafesearch.google.com",
	"www.google.cn":     "forcesafesearch.google.com",
	"www.google.com.co": "forcesafesearch.google.com",
	"www.google.co.cr":  "forcesafesearch.google.com",
	"www.google.com.cu": "forcesafesearch.google.com",
	"www.google.cv":     "forcesafesearch.google.com",
	"www.google.com.cy": "forcesafesearch.google.com",
	"www.google.cz":     "forcesafesearch.google.com",
	"www.google.de":     "forcesafesearch.google.com",
	"www.google.dj":     "forcesafesearch.google.com",
	"www.google.dk":     "forcesafesearch.google.com",
	"www.google.dm":     "forcesafesearch.google.com",
	"www.google.com.do": "forcesafesearch.google.com",
	"www.google.dz":     "forcesafesearch.google.com",
	"www.google.com.ec": "forcesafesearch.google.com",
	"www.google.ee":     "forcesafesearch.google.com",
	"www.google.com.eg": "forcesafesearch.google.com",
	"www.google.es":     "forcesafesearch.google.com",
	"www.google.com.et": "forcesafesearch.google.com",
	"www.google.fi":     "forcesafesearch.google.com",
	"www.google.com.fj": "forcesafesearch.google.com",
	"www.google.fm":     "forcesafesearch.google.com",
	"www.google.fr":     "forcesafesearch.google.com",
	"www.google.ga":     "forcesafesearch.google.com",
	"www.google.ge":     "forcesafesearch.google.com",
	"www.google.gg":     "forcesafesearch.google.com",
	"www.google.com.gh": "forcesafesearch.google.com",
	"www.google.com.gi": "forcesafesearch.google.com",
	"www.google.gl":     "forcesafesearch.google.com",
	"www.google.gm":     "forcesafesearch.google.com",
	"www.google.gp":     "forcesafesearch.google.com",
	"www.google.gr":     "forcesafesearch.google.com",
	"www.google.com.gt": "forcesafesearch.google.com",
	"www.google.gy":     "forcesafesearch.google.com",
	"www.google.com.hk": "forcesafesearch.google.com",
	"www.google.hn":     "forcesafesearch.google.com",
	"www.google.hr":     "forcesafesearch.google.com",
	"www.google.ht":     "forcesafesearch.google.com",
	"www.google.hu":     "forcesafesearch.google.com",
	"www.google.co.id":  "forcesafesearch.google.com",
	"www.google.ie":     "forcesafesearch.google.com",
	"www.google.co.il":  "forcesafesearch.google.com",
	"www.google.im":     "forcesafesearch.google.com",
	"www.google.co.in":  "forcesafesearch.google.com",
	"www.google.iq":     "forcesafesearch.google.com",
	"www.google.is":     "forcesafesearch.google.com",
	"www.google.it":     "forcesafesearch.google.com",
	"www.google.je":     "forcesafesearch.google.com",
	"www.google.com.jm": "forcesafesearch.google.com",
	"www.google.jo":     "forcesafesearch.google.com",
	"www.google.co.jp":  "forcesafesearch.google.com",
	"www.google.co.ke":  "forcesafesearch.google.com",
	"www.google.com.kh": "forcesafesearch.google.com",
	"www.google.ki":     "forcesafesearch.google.com",
	"www.google.kg":     "forcesafesearch.google.com",
	"www.google.co.kr":  "forcesafesearch.google.com",
	"www.google.com.kw": "forcesafesearch.google.com",
	"www.google.kz":     "forcesafesearch.google.com",
	"www.google.la":     "forcesafesearch.google.com",
	"www.google.com.lb": "forcesafesearch.google.com",
	"www.google.li":     "forcesafesearch.google.com",
	"www.google.lk":     "forcesafesearch.google.com",
	"www.google.co.ls":  "forcesafesearch.google.com",
	"www.google.lt":     "forcesafesearch.google.com",
	"www.google.lu":     "forcesafesearch.google.com",
	"www.google.lv":     "forcesafesearch.google.com",
	"www.google.com.ly": "forcesafesearch.google.com",
	"www.google.co.ma":  "forcesafesearch.google.com",
	"www.google.md":     "forcesafesearch.google.com",
	"www.google.me":     "forcesafesearch.google.com",
	"www.google.mg":     "forcesafesearch.google.com",
	"www.google.mk":     "forcesafesearch.google.com",
	"www.google.ml":     "forcesafesearch.google.com",
	"www.google.com.mm": "forcesafesearch.google.com",
	"www.google.mn":     "forcesafesearch.google.com",
	"www.google.ms":     "forcesafesearch.google.com",
	"www.google.com.mt": "forcesafesearch.google.com",
	"www.google.mu":     "forcesafesearch.google.com",
	"www.google.mv":     "forcesafesearch.google.com",
	"www.google.mw":     "forcesafesearch.google.com",
	"www.google.com.mx": "forcesafesearch.google.com",
	"www.google.com.my": "forcesafesearch.google.com",
	"www.google.co.mz":  "forcesafesearch.google.com",
	"www.google.com.na": "forcesafesearch.google.com",
	"www.google.com.nf": "forcesafesearch.google.com",
	"www.google.com.ng": "forcesafesearch.google.com",
	"www.google.com.ni": "forcesafesearch.google.com",
	"www.google.ne":     "forcesafesearch.google.com",
	"www.google.nl":     "forcesafesearch.google.com",
	"www.google.no":     "forcesafesearch.google.com",
	"www.google.com.np": "forcesafesearch.google.com",
	"www.google.nr":     "forcesafesearch.google.com",
	"www.google.nu":     "forcesafesearch.google.com",
	"www.google.co.nz":  "forcesafesearch.google.com",
	"www.google.com.om": "forcesafesearch.google.com",
	"www.google.com.pa": "forcesafesearch.google.com",
	"www.google.com.pe": "forcesafesearch.google.com",
	"www.google.com.pg": "forcesafesearch.google.com",
	"www.google.com.ph": "forcesafesearch.google.com",
	"www.google.com.pk": "forcesafesearch.google.com",
	"www.google.pl":     "forcesafesearch.google.com",
	"www.google.pn":     "forcesafesearch.google.com",
	"www.google.com.pr": "forcesafesearch.google.com",
	"www.google.ps":     "forcesafesearch.google.com",
	"www.google.pt":     "forcesafesearch.google.com",
	"www.google.com.py": "forcesafesearch.google.com",
	"www.google.com.qa": "forcesafesearch.google.com",
	"www.google.ro":     "forcesafesearch.google.com",
	"www.google.ru":     "forcesafesearch.google.com",
	"www.google.rw":     "forcesafesearch.google.com",
	"www.google.com.sa": "forcesafesearch.google.com",
	"www.google.com.sb": "forcesafesearch.google.com",
	"www.google.sc":     "forcesafesearch.google.com",
	"www.google.se":     "forcesafesearch.google.com",
	"www.google.com.sg": "forcesafesearch.google.com",
	"www.google.sh":     "forcesafesearch.google.com",
	"www.google.si":     "forcesafesearch.google.com",
	"www.google.sk":     "forcesafesearch.google.com",
	"www.google.com.sl": "forcesafesearch.google.com",
	"www.google.sn":     "forcesafesearch.google.com",
	"www.google.so":     "forcesafesearch.google.com",
	"www.google.sm":     "forcesafesearch.google.com",
	"www.google.sr":     "forcesafesearch.google.com",
	"www.google.st":     "forcesafesearch.google.com",
	"www.google.com.sv": "forcesafesearch.google.com",
	"www.google.td":     "forcesafesearch.google.com",
	"www.google.tg":     "forcesafesearch.google.com",
	"www.google.co.th":  "forcesafesearch.google.com",
	"www.google.com.tj": "forcesafesearch.google.com",
	"www.google.tk":     "forcesafesearch.google.com",
	"www.google.tl":     "forcesafesearch.google.com",
	"www.google.tm":     "forcesafesearch.google.com",
	"www.google.tn":     "forcesafesearch.google.com",
	"www.google.to":     "forcesafesearch.google.com",
	"www.google.com.tr": "forcesafesearch.google.com",
	"www.google.tt":     "forcesafesearch.google.com",
	"www.google.com.tw": "forcesafesearch.google.com",
	"www.google.co.tz":  "forcesafesearch.google.com",
	"www.google.com.ua": "forcesafesearch.google.com",
	"www.google.co.ug":  "forcesafesearch.google.com",
	"www.google.co.uk":  "forcesafesearch.google.com",
	"www.google.com.uy": "forcesafesearch.google.com",
	"www.google.co.uz":  "forcesafesearch.google.com",
	"www.google.com.vc": "forcesafesearch.google.com",
	"www.google.co.ve":  "forcesafesearch.google.com",
	"www.google.vg":     "forcesafesearch.google.com",
	"www.google.co.vi":  "forcesafesearch.google.com",
	"www.google.com.vn": "forcesafesearch.google.com",
	"www.google.vu":     "forcesafesearch.google.com",
	"www.google.ws":     "forcesafesearch.google.com",
	"www.google.rs":     "forcesafesearch.google.com",

	"www.youtube.com":          "restrictmoderate.youtube.com",
	"m.youtube.com":            "restrictmoderate.youtube.com",
	"youtubei.googleapis.com":  "restrictmoderate.youtube.com",
	"youtube.googleapis.com":   "restrictmoderate.youtube.com",
	"www.youtube-nocookie.com": "restrictmoderate.youtube.com",

	"pixabay.com": "safesearch.pixabay.com",
}
