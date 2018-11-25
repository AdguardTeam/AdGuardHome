package ratelimit

import (
	"errors"
	"log"
	"sort"
	"strconv"
	"time"

	// ratelimiting and per-ip buckets
	"github.com/beefsack/go-rate"
	"github.com/patrickmn/go-cache"

	// coredns plugin
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/metrics"
	"github.com/coredns/coredns/plugin/pkg/dnstest"
	"github.com/coredns/coredns/request"
	"github.com/mholt/caddy"
	"github.com/miekg/dns"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/context"
)

const defaultRatelimit = 100
const defaultResponseSize = 1000

var (
	tokenBuckets = cache.New(time.Hour, time.Hour)
)

// ServeDNS handles the DNS request and refuses if it's an beyind specified ratelimit
func (p *plug) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
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

	// Record response to get status code and size of the reply.
	rw := dnstest.NewRecorder(w)
	status, err := plugin.NextOrFailure(p.Name(), p.Next, ctx, rw, r)

	size := rw.Len

	if size > defaultResponseSize && state.Proto() == "udp" {
		// For large UDP responses we call allowRequest more times
		// The exact number of times depends on the response size
		for i := 0; i < size/defaultResponseSize; i++ {
			p.allowRequest(ip)
		}
	}

	return status, err
}

func (p *plug) allowRequest(ip string) (bool, error) {

	if len(p.whitelist) > 0 {
		i := sort.SearchStrings(p.whitelist, ip)

		if i < len(p.whitelist) && p.whitelist[i] == ip {
			return true, nil
		}
	}

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
	if !ok {
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

type plug struct {
	Next plugin.Handler

	// configuration for creating above
	ratelimit int      // in requests per second per IP
	whitelist []string // a list of whitelisted IP addresses
}

func setupPlugin(c *caddy.Controller) (*plug, error) {

	p := &plug{ratelimit: defaultRatelimit}

	for c.Next() {
		args := c.RemainingArgs()
		if len(args) > 0 {
			ratelimit, err := strconv.Atoi(args[0])
			if err != nil {
				return nil, c.ArgErr()
			}
			p.ratelimit = ratelimit
		}
		for c.NextBlock() {
			switch c.Val() {
			case "whitelist":
				p.whitelist = c.RemainingArgs()

				if len(p.whitelist) > 0 {
					sort.Strings(p.whitelist)
				}
			}
		}
	}

	return p, nil
}

func setup(c *caddy.Controller) error {
	p, err := setupPlugin(c)
	if err != nil {
		return err
	}

	config := dnsserver.GetConfig(c)
	config.AddPlugin(func(next plugin.Handler) plugin.Handler {
		p.Next = next
		return p
	})

	c.OnStartup(func() error {
		m := dnsserver.GetConfig(c).Handler("prometheus")
		if m == nil {
			return nil
		}
		if x, ok := m.(*metrics.Metrics); ok {
			x.MustRegister(ratelimited)
		}
		return nil
	})

	return nil
}

func newDNSCounter(name string, help string) prometheus.Counter {
	return prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: plugin.Namespace,
		Subsystem: "ratelimit",
		Name:      name,
		Help:      help,
	})
}

var (
	ratelimited = newDNSCounter("dropped_total", "Count of requests that have been dropped because of rate limit")
)

// Name returns name of the plugin as seen in Corefile and plugin.cfg
func (p *plug) Name() string { return "ratelimit" }
