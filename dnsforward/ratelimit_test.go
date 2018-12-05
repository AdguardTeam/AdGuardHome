package dnsforward

import (
	"testing"
)

func TestRatelimiting(t *testing.T) {
	// rate limit is 1 per sec
	p := Server{}
	p.Ratelimit = 1

	limited := p.isRatelimited("127.0.0.1")

	if limited {
		t.Fatal("First request must have been allowed")
	}

	limited = p.isRatelimited("127.0.0.1")

	if !limited {
		t.Fatal("Second request must have been ratelimited")
	}
}

func TestWhitelist(t *testing.T) {
	// rate limit is 1 per sec with whitelist
	p := Server{}
	p.Ratelimit = 1
	p.RatelimitWhitelist = []string{"127.0.0.1", "127.0.0.2", "127.0.0.125"}

	limited := p.isRatelimited("127.0.0.1")

	if limited {
		t.Fatal("First request must have been allowed")
	}

	limited = p.isRatelimited("127.0.0.1")

	if limited {
		t.Fatal("Second request must have been allowed due to whitelist")
	}
}
