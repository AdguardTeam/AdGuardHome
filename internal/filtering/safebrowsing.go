package filtering

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/cache"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/stringutil"
	"github.com/miekg/dns"
	"golang.org/x/exp/slices"
	"golang.org/x/net/publicsuffix"
)

// Safe browsing and parental control methods.

// TODO(a.garipov): Make configurable.
const (
	dnsTimeout                = 3 * time.Second
	defaultSafebrowsingServer = `https://family.adguard-dns.com/dns-query`
	defaultParentalServer     = `https://family.adguard-dns.com/dns-query`
	sbTXTSuffix               = `sb.dns.adguard.com.`
	pcTXTSuffix               = `pc.dns.adguard.com.`
)

// SetParentalUpstream sets the parental upstream for *DNSFilter.
//
// TODO(e.burkov): Remove this in v1 API to forbid the direct access.
func (d *DNSFilter) SetParentalUpstream(u upstream.Upstream) {
	d.parentalUpstream = u
}

// SetSafeBrowsingUpstream sets the safe browsing upstream for *DNSFilter.
//
// TODO(e.burkov): Remove this in v1 API to forbid the direct access.
func (d *DNSFilter) SetSafeBrowsingUpstream(u upstream.Upstream) {
	d.safeBrowsingUpstream = u
}

func (d *DNSFilter) initSecurityServices() error {
	var err error
	d.safeBrowsingServer = defaultSafebrowsingServer
	d.parentalServer = defaultParentalServer
	opts := &upstream.Options{
		Timeout: dnsTimeout,
		ServerIPAddrs: []net.IP{
			{94, 140, 14, 15},
			{94, 140, 15, 16},
			net.ParseIP("2a10:50c0::bad1:ff"),
			net.ParseIP("2a10:50c0::bad2:ff"),
		},
	}

	parUps, err := upstream.AddressToUpstream(d.parentalServer, opts)
	if err != nil {
		return fmt.Errorf("converting parental server: %w", err)
	}
	d.SetParentalUpstream(parUps)

	sbUps, err := upstream.AddressToUpstream(d.safeBrowsingServer, opts)
	if err != nil {
		return fmt.Errorf("converting safe browsing server: %w", err)
	}
	d.SetSafeBrowsingUpstream(sbUps)

	return nil
}

/*
expire byte[4]
hash byte[32]
...
*/
func (c *sbCtx) setCache(prefix, hashes []byte) {
	d := make([]byte, 4+len(hashes))
	expire := uint(time.Now().Unix()) + c.cacheTime*60
	binary.BigEndian.PutUint32(d[:4], uint32(expire))
	copy(d[4:], hashes)
	c.cache.Set(prefix, d)
	log.Debug("%s: stored in cache: %v", c.svc, prefix)
}

// findInHash returns 32-byte hash if it's found in hashToHost.
func (c *sbCtx) findInHash(val []byte) (hash32 [32]byte, found bool) {
	for i := 4; i < len(val); i += 32 {
		hash := val[i : i+32]

		copy(hash32[:], hash[0:32])

		_, found = c.hashToHost[hash32]
		if found {
			return hash32, found
		}
	}

	return [32]byte{}, false
}

func (c *sbCtx) getCached() int {
	now := time.Now().Unix()
	hashesToRequest := map[[32]byte]string{}
	for k, v := range c.hashToHost {
		// nolint:looppointer // The subsilce is used for a safe cache lookup.
		val := c.cache.Get(k[0:2])
		if val == nil || now >= int64(binary.BigEndian.Uint32(val)) {
			hashesToRequest[k] = v
			continue
		}
		if hash32, found := c.findInHash(val); found {
			log.Debug("%s: found in cache: %s: blocked by %v", c.svc, c.host, hash32)
			return 1
		}
	}

	if len(hashesToRequest) == 0 {
		log.Debug("%s: found in cache: %s: not blocked", c.svc, c.host)
		return -1
	}

	c.hashToHost = hashesToRequest
	return 0
}

type sbCtx struct {
	host       string
	svc        string
	hashToHost map[[32]byte]string
	cache      cache.Cache
	cacheTime  uint
}

func hostnameToHashes(host string) map[[32]byte]string {
	hashes := map[[32]byte]string{}
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
		hashes[sum] = curhost

		pos := strings.IndexByte(curhost, byte('.'))
		if pos < 0 {
			break
		}
		curhost = curhost[pos+1:]
	}
	return hashes
}

// convert hash array to string
func (c *sbCtx) getQuestion() string {
	b := &strings.Builder{}

	for hash := range c.hashToHost {
		// nolint:looppointer // The subsilce is used for safe hex encoding.
		stringutil.WriteToBuilder(b, hex.EncodeToString(hash[0:2]), ".")
	}

	if c.svc == "SafeBrowsing" {
		stringutil.WriteToBuilder(b, sbTXTSuffix)

		return b.String()
	}

	stringutil.WriteToBuilder(b, pcTXTSuffix)

	return b.String()
}

// Find the target hash in TXT response
func (c *sbCtx) processTXT(resp *dns.Msg) (bool, [][]byte) {
	matched := false
	hashes := [][]byte{}
	for _, a := range resp.Answer {
		txt, ok := a.(*dns.TXT)
		if !ok {
			continue
		}
		log.Debug("%s: received hashes for %s: %v", c.svc, c.host, txt.Txt)

		for _, t := range txt.Txt {
			if len(t) != 32*2 {
				continue
			}
			hash, err := hex.DecodeString(t)
			if err != nil {
				continue
			}

			hashes = append(hashes, hash)

			if !matched {
				var hash32 [32]byte
				copy(hash32[:], hash)

				var hashHost string
				hashHost, ok = c.hashToHost[hash32]
				if ok {
					log.Debug("%s: matched %s by %s/%s", c.svc, c.host, hashHost, t)
					matched = true
				}
			}
		}
	}

	return matched, hashes
}

func (c *sbCtx) storeCache(hashes [][]byte) {
	slices.SortFunc(hashes, func(a, b []byte) (sortsBefore bool) {
		return bytes.Compare(a, b) == -1
	})

	var curData []byte
	var prevPrefix []byte
	for i, hash := range hashes {
		// nolint:looppointer // The subsilce is used for a safe comparison.
		if !bytes.Equal(hash[0:2], prevPrefix) {
			if i != 0 {
				c.setCache(prevPrefix, curData)
				curData = nil
			}
			prevPrefix = hashes[i][0:2]
		}
		curData = append(curData, hash...)
	}

	if len(prevPrefix) != 0 {
		c.setCache(prevPrefix, curData)
	}

	for hash := range c.hashToHost {
		// nolint:looppointer // The subsilce is used for a safe cache lookup.
		prefix := hash[0:2]
		val := c.cache.Get(prefix)
		if val == nil {
			c.setCache(prefix, nil)
		}
	}
}

func check(c *sbCtx, r Result, u upstream.Upstream) (Result, error) {
	c.hashToHost = hostnameToHashes(c.host)
	switch c.getCached() {
	case -1:
		return Result{}, nil
	case 1:
		return r, nil
	}

	question := c.getQuestion()

	log.Tracef("%s: checking %s: %s", c.svc, c.host, question)
	req := (&dns.Msg{}).SetQuestion(question, dns.TypeTXT)

	resp, err := u.Exchange(req)
	if err != nil {
		return Result{}, err
	}

	matched, receivedHashes := c.processTXT(resp)

	c.storeCache(receivedHashes)
	if matched {
		return r, nil
	}

	return Result{}, nil
}

// TODO(a.garipov): Unify with checkParental.
func (d *DNSFilter) checkSafeBrowsing(
	host string,
	_ uint16,
	setts *Settings,
) (res Result, err error) {
	if !setts.ProtectionEnabled || !setts.SafeBrowsingEnabled {
		return Result{}, nil
	}

	if log.GetLevel() >= log.DEBUG {
		timer := log.StartTimer()
		defer timer.LogElapsed("safebrowsing lookup for %q", host)
	}

	sctx := &sbCtx{
		host:      host,
		svc:       "SafeBrowsing",
		cache:     d.safebrowsingCache,
		cacheTime: d.Config.CacheTime,
	}

	res = Result{
		Rules: []*ResultRule{{
			Text:         "adguard-malware-shavar",
			FilterListID: SafeBrowsingListID,
		}},
		Reason:     FilteredSafeBrowsing,
		IsFiltered: true,
	}

	return check(sctx, res, d.safeBrowsingUpstream)
}

// TODO(a.garipov): Unify with checkSafeBrowsing.
func (d *DNSFilter) checkParental(
	host string,
	_ uint16,
	setts *Settings,
) (res Result, err error) {
	if !setts.ProtectionEnabled || !setts.ParentalEnabled {
		return Result{}, nil
	}

	if log.GetLevel() >= log.DEBUG {
		timer := log.StartTimer()
		defer timer.LogElapsed("parental lookup for %q", host)
	}

	sctx := &sbCtx{
		host:      host,
		svc:       "Parental",
		cache:     d.parentalCache,
		cacheTime: d.Config.CacheTime,
	}

	res = Result{
		Rules: []*ResultRule{{
			Text:         "parental CATEGORY_BLACKLISTED",
			FilterListID: ParentalListID,
		}},
		Reason:     FilteredParental,
		IsFiltered: true,
	}

	return check(sctx, res, d.parentalUpstream)
}

// setProtectedBool sets the value of a boolean pointer under a lock.  l must
// protect the value under ptr.
//
// TODO(e.burkov):  Make it generic?
func setProtectedBool(mu *sync.RWMutex, ptr *bool, val bool) {
	mu.Lock()
	defer mu.Unlock()

	*ptr = val
}

// protectedBool gets the value of a boolean pointer under a read lock.  l must
// protect the value under ptr.
//
// TODO(e.burkov):  Make it generic?
func protectedBool(mu *sync.RWMutex, ptr *bool) (val bool) {
	mu.RLock()
	defer mu.RUnlock()

	return *ptr
}

func (d *DNSFilter) handleSafeBrowsingEnable(w http.ResponseWriter, r *http.Request) {
	setProtectedBool(&d.confLock, &d.Config.SafeBrowsingEnabled, true)
	d.Config.ConfigModified()
}

func (d *DNSFilter) handleSafeBrowsingDisable(w http.ResponseWriter, r *http.Request) {
	setProtectedBool(&d.confLock, &d.Config.SafeBrowsingEnabled, false)
	d.Config.ConfigModified()
}

func (d *DNSFilter) handleSafeBrowsingStatus(w http.ResponseWriter, r *http.Request) {
	resp := &struct {
		Enabled bool `json:"enabled"`
	}{
		Enabled: protectedBool(&d.confLock, &d.Config.SafeBrowsingEnabled),
	}

	_ = aghhttp.WriteJSONResponse(w, r, resp)
}

func (d *DNSFilter) handleParentalEnable(w http.ResponseWriter, r *http.Request) {
	setProtectedBool(&d.confLock, &d.Config.ParentalEnabled, true)
	d.Config.ConfigModified()
}

func (d *DNSFilter) handleParentalDisable(w http.ResponseWriter, r *http.Request) {
	setProtectedBool(&d.confLock, &d.Config.ParentalEnabled, false)
	d.Config.ConfigModified()
}

func (d *DNSFilter) handleParentalStatus(w http.ResponseWriter, r *http.Request) {
	resp := &struct {
		Enabled bool `json:"enabled"`
	}{
		Enabled: protectedBool(&d.confLock, &d.Config.ParentalEnabled),
	}

	_ = aghhttp.WriteJSONResponse(w, r, resp)
}
