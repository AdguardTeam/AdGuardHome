package ratelimit

import (
	"testing"

	"github.com/mholt/caddy"
)

func TestSetup(t *testing.T) {
	for i, testcase := range []struct {
		config  string
		failing bool
	}{
		{`ratelimit`, false},
		{`ratelimit 100`, false},
		{`ratelimit { 
					whitelist 127.0.0.1
				}`, false},
		{`ratelimit 50 {
					whitelist 127.0.0.1 176.103.130.130
				}`, false},
		{`ratelimit test`, true},
	} {
		c := caddy.NewTestController("dns", testcase.config)
		err := setup(c)
		if err != nil {
			if !testcase.failing {
				t.Fatalf("Test #%d expected no errors, but got: %v", i, err)
			}
			continue
		}
		if testcase.failing {
			t.Fatalf("Test #%d expected to fail but it didn't", i)
		}
	}
}

func TestRatelimiting(t *testing.T) {
	// rate limit is 1 per sec
	c := caddy.NewTestController("dns", `ratelimit 1`)
	p, err := setupPlugin(c)

	if err != nil {
		t.Fatal("Failed to initialize the plugin")
	}

	allowed, err := p.allowRequest("127.0.0.1")

	if err != nil || !allowed {
		t.Fatal("First request must have been allowed")
	}

	allowed, err = p.allowRequest("127.0.0.1")

	if err != nil || allowed {
		t.Fatal("Second request must have been ratelimited")
	}
}

func TestWhitelist(t *testing.T) {
	// rate limit is 1 per sec
	c := caddy.NewTestController("dns", `ratelimit 1 { whitelist 127.0.0.2 127.0.0.1 127.0.0.125 }`)
	p, err := setupPlugin(c)

	if err != nil {
		t.Fatal("Failed to initialize the plugin")
	}

	allowed, err := p.allowRequest("127.0.0.1")

	if err != nil || !allowed {
		t.Fatal("First request must have been allowed")
	}

	allowed, err = p.allowRequest("127.0.0.1")

	if err != nil || !allowed {
		t.Fatal("Second request must have been allowed due to whitelist")
	}
}
