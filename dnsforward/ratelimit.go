package dnsforward

import (
	"log"
	"sort"
	"time"

	"github.com/beefsack/go-rate"
	gocache "github.com/patrickmn/go-cache"
)

func (s *Server) limiterForIP(ip string) interface{} {
	if s.ratelimitBuckets == nil {
		s.ratelimitBuckets = gocache.New(time.Hour, time.Hour)
	}

	// check if ratelimiter for that IP already exists, if not, create
	value, found := s.ratelimitBuckets.Get(ip)
	if !found {
		value = rate.New(s.Ratelimit, time.Second)
		s.ratelimitBuckets.Set(ip, value, time.Hour)
	}

	return value
}

func (s *Server) isRatelimited(ip string) bool {
	if s.Ratelimit == 0 { // 0 -- disabled
		return false
	}
	if len(s.RatelimitWhitelist) > 0 {
		i := sort.SearchStrings(s.RatelimitWhitelist, ip)

		if i < len(s.RatelimitWhitelist) && s.RatelimitWhitelist[i] == ip {
			// found, don't ratelimit
			return false
		}
	}

	value := s.limiterForIP(ip)
	rl, ok := value.(*rate.RateLimiter)
	if !ok {
		log.Println("SHOULD NOT HAPPEN: non-bool entry found in safebrowsing lookup cache")
		return false
	}

	allow, _ := rl.Try()
	return !allow
}

func (s *Server) isRatelimitedForReply(ip string, size int) bool {
	if s.Ratelimit == 0 { // 0 -- disabled
		return false
	}
	if len(s.RatelimitWhitelist) > 0 {
		i := sort.SearchStrings(s.RatelimitWhitelist, ip)

		if i < len(s.RatelimitWhitelist) && s.RatelimitWhitelist[i] == ip {
			// found, don't ratelimit
			return false
		}
	}

	value := s.limiterForIP(ip)
	rl, ok := value.(*rate.RateLimiter)
	if !ok {
		log.Println("SHOULD NOT HAPPEN: non-bool entry found in safebrowsing lookup cache")
		return false
	}

	// For large UDP responses we try more times, effectively limiting per bandwidth
	// The exact number of times depends on the response size
	for i := 0; i < size/1000; i++ {
		allow, _ := rl.Try()
		if !allow { // not allowed -> ratelimited
			return true
		}
	}
	return false
}
