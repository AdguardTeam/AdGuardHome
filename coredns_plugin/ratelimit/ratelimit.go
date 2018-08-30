package ratelimit

import (
	"errors"
	"log"
	"strconv"
	"sync"
	"time"

	// ratelimiting and per-ip buckets
	"github.com/beefsack/go-rate"
	"github.com/patrickmn/go-cache"

	// coredns plugin
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/metrics"
	"github.com/coredns/coredns/request"
	"github.com/mholt/caddy"
	"github.com/miekg/dns"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/context"
)

const defaultRatelimit = 100
const defaultMaxRateLimitedIPs = 1024 * 1024

var (
	tokenBuckets = cache.New(time.Hour, time.Hour)
)

// main function
func (p *Plugin) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r}
	ip := state.IP()
	allow, err := p.allowRequest(ip)
	if err != nil {
		return 0, err
	}
	if !allow {
		ratelimited.Inc()
		return 0, nil
	}
	return plugin.NextOrFailure(p.Name(), p.Next, ctx, w, r)
}

func (p *Plugin) allowRequest(ip string) (bool, error) {
	if _, found := tokenBuckets.Get(ip); !found {
		tokenBuckets.Set(ip, rate.New(p.ratelimit, time.Second), time.Hour)
	}

	value, found := tokenBuckets.Get(ip)
	if !found {
		// should not happen since we've just inserted it
		text := "SHOULD NOT HAPPEN: just-inserted ratelimiter disappeared"
		log.Println(text)
		err := errors.New(text)
		return true, err
	}

	rl, ok := value.(*rate.RateLimiter)
	if ok == false {
		text := "SHOULD NOT HAPPEN: non-bool entry found in safebrowsing lookup cache"
		log.Println(text)
		err := errors.New(text)
		return true, err
	}

	allow, _ := rl.Try()
	return allow, nil
}

//
// helper functions
//
func init() {
	caddy.RegisterPlugin("ratelimit", caddy.Plugin{
		ServerType: "dns",
		Action:     setup,
	})
}

type Plugin struct {
	Next plugin.Handler

	// configuration for creating above
	ratelimit int // in requests per second per IP
}

func setup(c *caddy.Controller) error {
	p := &Plugin{ratelimit: defaultRatelimit}
	config := dnsserver.GetConfig(c)

	for c.Next() {
		args := c.RemainingArgs()
		if len(args) <= 0 {
			continue
		}
		ratelimit, err := strconv.Atoi(args[0])
		if err != nil {
			return c.ArgErr()
		}
		p.ratelimit = ratelimit
	}

	config.AddPlugin(func(next plugin.Handler) plugin.Handler {
		p.Next = next
		return p
	})

	c.OnStartup(func() error {
		once.Do(func() {
			m := dnsserver.GetConfig(c).Handler("prometheus")
			if m == nil {
				return
			}
			if x, ok := m.(*metrics.Metrics); ok {
				x.MustRegister(ratelimited)
			}
		})
		return nil
	})

	return nil
}

func newDnsCounter(name string, help string) prometheus.Counter {
	return prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: plugin.Namespace,
		Subsystem: "ratelimit",
		Name:      name,
		Help:      help,
	})
}

var (
	ratelimited = newDnsCounter("dropped_total", "Count of requests that have been dropped because of rate limit")
)

func (d *Plugin) Name() string { return "ratelimit" }

var once sync.Once
