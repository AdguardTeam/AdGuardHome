package refuseany

import (
	"fmt"
	"log"

	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/metrics"
	"github.com/coredns/coredns/request"
	"github.com/mholt/caddy"
	"github.com/miekg/dns"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/net/context"
)

type plug struct {
	Next plugin.Handler
}

// ServeDNS handles the DNS request and refuses if it's an ANY request
func (p *plug) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	if len(r.Question) != 1 {
		// google DNS, bind and others do the same
		return dns.RcodeFormatError, fmt.Errorf("Got DNS request with != 1 questions")
	}

	q := r.Question[0]
	if q.Qtype == dns.TypeANY {
		log.Printf("Got request with type ANY, will respond with NOTIMP\n")

		state := request.Request{W: w, Req: r, Context: ctx}
		rcode := dns.RcodeNotImplemented

		m := new(dns.Msg)
		m.SetRcode(r, rcode)
		state.SizeAndDo(m)
		err := state.W.WriteMsg(m)
		if err != nil {
			log.Printf("Got error %s\n", err)
			return dns.RcodeServerFailure, err
		}
		return rcode, nil
	}

	return plugin.NextOrFailure(p.Name(), p.Next, ctx, w, r)
}

func init() {
	caddy.RegisterPlugin("refuseany", caddy.Plugin{
		ServerType: "dns",
		Action:     setup,
	})
}

func setup(c *caddy.Controller) error {
	p := &plug{}
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
		Subsystem: "refuseany",
		Name:      name,
		Help:      help,
	})
}

var (
	ratelimited = newDNSCounter("refusedany_total", "Count of ANY requests that have been dropped")
)

// Name returns name of the plugin as seen in Corefile and plugin.cfg
func (p *plug) Name() string { return "refuseany" }
