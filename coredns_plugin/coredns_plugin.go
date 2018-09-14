package dnsfilter

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/metrics"
	"github.com/coredns/coredns/plugin/pkg/dnstest"
	"github.com/coredns/coredns/plugin/pkg/upstream"
	"github.com/coredns/coredns/request"

	"github.com/mholt/caddy"

	"github.com/miekg/dns"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/AdguardTeam/AdguardDNS/dnsfilter"
	"golang.org/x/net/context"
)

var defaultSOA = &dns.SOA{
	// values copied from verisign's nonexistent .com domain
	// their exact values are not important in our use case because they are used for domain transfers between primary/secondary DNS servers
	Refresh: 1800,
	Retry:   900,
	Expire:  604800,
	Minttl:  86400,
}

func init() {
	caddy.RegisterPlugin("dnsfilter", caddy.Plugin{
		ServerType: "dns",
		Action:     setup,
	})
}

type plug struct {
	d        *dnsfilter.Dnsfilter
	Next     plugin.Handler
	upstream upstream.Upstream
	hosts    map[string]net.IP

	SafeBrowsingBlockHost string
	ParentalBlockHost     string
	QueryLogEnabled       bool
}

var defaultPlugin = plug{
	SafeBrowsingBlockHost: "safebrowsing.block.dns.adguard.com",
	ParentalBlockHost:     "family.block.dns.adguard.com",
}

func newDNSCounter(name string, help string) prometheus.Counter {
	return prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: plugin.Namespace,
		Subsystem: "dnsfilter",
		Name:      name,
		Help:      help,
	})
}

var (
	requests             = newDNSCounter("requests_total", "Count of requests seen by dnsfilter.")
	filtered             = newDNSCounter("filtered_total", "Count of requests filtered by dnsfilter.")
	filteredLists        = newDNSCounter("filtered_lists_total", "Count of requests filtered by dnsfilter using lists.")
	filteredSafebrowsing = newDNSCounter("filtered_safebrowsing_total", "Count of requests filtered by dnsfilter using safebrowsing.")
	filteredParental     = newDNSCounter("filtered_parental_total", "Count of requests filtered by dnsfilter using parental.")
	filteredInvalid      = newDNSCounter("filtered_invalid_total", "Count of requests filtered by dnsfilter because they were invalid.")
	whitelisted          = newDNSCounter("whitelisted_total", "Count of requests not filtered by dnsfilter because they are whitelisted.")
	safesearch           = newDNSCounter("safesearch_total", "Count of requests replaced by dnsfilter safesearch.")
	errorsTotal          = newDNSCounter("errors_total", "Count of requests that dnsfilter couldn't process because of transitive errors.")
)

//
// coredns handling functions
//
func setupPlugin(c *caddy.Controller) (*plug, error) {
	// create new Plugin and copy default values
	var p = new(plug)
	*p = defaultPlugin
	p.d = dnsfilter.New()
	p.hosts = make(map[string]net.IP)

	var filterFileName string
	for c.Next() {
		args := c.RemainingArgs()
		if len(args) == 0 {
			// must have at least one argument
			return nil, c.ArgErr()
		}
		filterFileName = args[0]
		for c.NextBlock() {
			switch c.Val() {
			case "safebrowsing":
				p.d.EnableSafeBrowsing()
				if c.NextArg() {
					if len(c.Val()) == 0 {
						return nil, c.ArgErr()
					}
					p.d.SetSafeBrowsingServer(c.Val())
				}
			case "safesearch":
				p.d.EnableSafeSearch()
			case "parental":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				sensitivity, err := strconv.Atoi(c.Val())
				if err != nil {
					return nil, c.ArgErr()
				}
				err = p.d.EnableParental(sensitivity)
				if err != nil {
					return nil, c.ArgErr()
				}
				if c.NextArg() {
					if len(c.Val()) == 0 {
						return nil, c.ArgErr()
					}
					p.ParentalBlockHost = c.Val()
				}
			case "querylog":
				p.QueryLogEnabled = true
				onceQueryLog.Do(func() {
					go startQueryLogServer() // TODO: how to handle errors?
				})
			}
		}
	}

	file, err := os.Open(filterFileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	count := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		text := scanner.Text()
		if p.parseEtcHosts(text) {
			continue
		}
		err = p.d.AddRule(text, 0)
		if err == dnsfilter.ErrInvalidSyntax {
			continue
		}
		if err != nil {
			return nil, err
		}
		count++
	}
	log.Printf("Added %d rules from %s", count, filterFileName)

	if err = scanner.Err(); err != nil {
		return nil, err
	}

	p.upstream, err = upstream.New(nil)
	if err != nil {
		return nil, err
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
			x.MustRegister(requests)
			x.MustRegister(filtered)
			x.MustRegister(filteredLists)
			x.MustRegister(filteredSafebrowsing)
			x.MustRegister(filteredParental)
			x.MustRegister(whitelisted)
			x.MustRegister(safesearch)
			x.MustRegister(errorsTotal)
			x.MustRegister(p)
		}
		return nil
	})
	c.OnShutdown(p.onShutdown)

	return nil
}

func (p *plug) parseEtcHosts(text string) bool {
	if pos := strings.IndexByte(text, '#'); pos != -1 {
		text = text[0:pos]
	}
	fields := strings.Fields(text)
	if len(fields) < 2 {
		return false
	}
	addr := net.ParseIP(fields[0])
	if addr == nil {
		return false
	}
	for _, host := range fields[1:] {
		if val, ok := p.hosts[host]; ok {
			log.Printf("warning: host %s already has value %s, will overwrite it with %s", host, val, addr)
		}
		p.hosts[host] = addr
	}
	return true
}

func (p *plug) onShutdown() error {
	p.d.Destroy()
	p.d = nil
	return nil
}

type statsFunc func(ch interface{}, name string, text string, value float64, valueType prometheus.ValueType)

func doDesc(ch interface{}, name string, text string, value float64, valueType prometheus.ValueType) {
	realch, ok := ch.(chan<- *prometheus.Desc)
	if !ok {
		log.Printf("Couldn't convert ch to chan<- *prometheus.Desc\n")
		return
	}
	realch <- prometheus.NewDesc(name, text, nil, nil)
}

func doMetric(ch interface{}, name string, text string, value float64, valueType prometheus.ValueType) {
	realch, ok := ch.(chan<- prometheus.Metric)
	if !ok {
		log.Printf("Couldn't convert ch to chan<- prometheus.Metric\n")
		return
	}
	desc := prometheus.NewDesc(name, text, nil, nil)
	realch <- prometheus.MustNewConstMetric(desc, valueType, value)
}

func gen(ch interface{}, doFunc statsFunc, name string, text string, value float64, valueType prometheus.ValueType) {
	doFunc(ch, name, text, value, valueType)
}

func doStatsLookup(ch interface{}, doFunc statsFunc, name string, lookupstats *dnsfilter.LookupStats) {
	gen(ch, doFunc, fmt.Sprintf("coredns_dnsfilter_%s_requests", name), fmt.Sprintf("Number of %s HTTP requests that were sent", name), float64(lookupstats.Requests), prometheus.CounterValue)
	gen(ch, doFunc, fmt.Sprintf("coredns_dnsfilter_%s_cachehits", name), fmt.Sprintf("Number of %s lookups that didn't need HTTP requests", name), float64(lookupstats.CacheHits), prometheus.CounterValue)
	gen(ch, doFunc, fmt.Sprintf("coredns_dnsfilter_%s_pending", name), fmt.Sprintf("Number of currently pending %s HTTP requests", name), float64(lookupstats.Pending), prometheus.GaugeValue)
	gen(ch, doFunc, fmt.Sprintf("coredns_dnsfilter_%s_pending_max", name), fmt.Sprintf("Maximum number of pending %s HTTP requests", name), float64(lookupstats.PendingMax), prometheus.GaugeValue)
}

func (p *plug) doStats(ch interface{}, doFunc statsFunc) {
	stats := p.d.GetStats()
	doStatsLookup(ch, doFunc, "safebrowsing", &stats.Safebrowsing)
	doStatsLookup(ch, doFunc, "parental", &stats.Parental)
}

// Describe is called by prometheus handler to know stat types
func (p *plug) Describe(ch chan<- *prometheus.Desc) {
	p.doStats(ch, doDesc)
}

// Collect is called by prometheus handler to collect stats
func (p *plug) Collect(ch chan<- prometheus.Metric) {
	p.doStats(ch, doMetric)
}

func (p *plug) replaceHostWithValAndReply(ctx context.Context, w dns.ResponseWriter, r *dns.Msg, host string, val string, question dns.Question) (int, error) {
	// check if it's a domain name or IP address
	addr := net.ParseIP(val)
	var records []dns.RR
	log.Println("Will give", val, "instead of", host)
	if addr != nil {
		// this is an IP address, return it
		result, err := dns.NewRR(host + " A " + val)
		if err != nil {
			log.Printf("Got error %s\n", err)
			return dns.RcodeServerFailure, fmt.Errorf("plugin/dnsfilter: %s", err)
		}
		records = append(records, result)
	} else {
		// this is a domain name, need to look it up
		req := new(dns.Msg)
		req.SetQuestion(dns.Fqdn(val), question.Qtype)
		req.RecursionDesired = true
		reqstate := request.Request{W: w, Req: req, Context: ctx}
		result, err := p.upstream.Lookup(reqstate, dns.Fqdn(val), reqstate.QType())
		if err != nil {
			log.Printf("Got error %s\n", err)
			return dns.RcodeServerFailure, fmt.Errorf("plugin/dnsfilter: %s", err)
		}
		if result != nil {
			for _, answer := range result.Answer {
				answer.Header().Name = question.Name
			}
			records = result.Answer
		}
	}
	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative, m.RecursionAvailable, m.Compress = true, true, true
	m.Answer = append(m.Answer, records...)
	state := request.Request{W: w, Req: r, Context: ctx}
	state.SizeAndDo(m)
	err := state.W.WriteMsg(m)
	if err != nil {
		log.Printf("Got error %s\n", err)
		return dns.RcodeServerFailure, fmt.Errorf("plugin/dnsfilter: %s", err)
	}
	return dns.RcodeSuccess, nil
}

// generate SOA record that makes DNS clients cache NXdomain results
// the only value that is important is TTL in header, other values like refresh, retry, expire and minttl are irrelevant
func genSOA(r *dns.Msg) []dns.RR {
	zone := r.Question[0].Name
	header := dns.RR_Header{Name: zone, Rrtype: dns.TypeSOA, Ttl: 3600, Class: dns.ClassINET}

	Mbox := "hostmaster."
	if zone[0] != '.' {
		Mbox += zone
	}
	Ns := "fake-for-negative-caching.adguard.com."

	soa := defaultSOA
	soa.Hdr = header
	soa.Mbox = Mbox
	soa.Ns = Ns
	soa.Serial = uint32(time.Now().Unix())
	return []dns.RR{soa}
}

func writeNXdomain(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r, Context: ctx}
	m := new(dns.Msg)
	m.SetRcode(state.Req, dns.RcodeNameError)
	m.Authoritative, m.RecursionAvailable, m.Compress = true, true, true
	m.Ns = genSOA(r)

	state.SizeAndDo(m)
	err := state.W.WriteMsg(m)
	if err != nil {
		log.Printf("Got error %s\n", err)
		return dns.RcodeServerFailure, err
	}
	return dns.RcodeNameError, nil
}

func (p *plug) serveDNSInternal(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, dnsfilter.Result, error) {
	if len(r.Question) != 1 {
		// google DNS, bind and others do the same
		return dns.RcodeFormatError, dnsfilter.Result{}, fmt.Errorf("Got DNS request with != 1 questions")
	}
	for _, question := range r.Question {
		host := strings.ToLower(strings.TrimSuffix(question.Name, "."))
		// is it a safesearch domain?
		if val, ok := p.d.SafeSearchDomain(host); ok {
			rcode, err := p.replaceHostWithValAndReply(ctx, w, r, host, val, question)
			if err != nil {
				return rcode, dnsfilter.Result{}, err
			}
			return rcode, dnsfilter.Result{Reason: dnsfilter.FilteredSafeSearch}, err
		}

		// is it in hosts?
		if val, ok := p.hosts[host]; ok {
			// it is, if it's a loopback host, reply with NXDOMAIN
			if val.IsLoopback() {
				rcode, err := writeNXdomain(ctx, w, r)
				if err != nil {
					return rcode, dnsfilter.Result{}, err
				}
				return rcode, dnsfilter.Result{Reason: dnsfilter.FilteredInvalid}, err
			}
			// it's not a loopback host, replace it with value specified
			rcode, err := p.replaceHostWithValAndReply(ctx, w, r, host, val.String(), question)
			if err != nil {
				return rcode, dnsfilter.Result{}, err
			}
			return rcode, dnsfilter.Result{Reason: dnsfilter.FilteredSafeSearch}, err
		}

		// needs to be filtered instead
		result, err := p.d.CheckHost(host)
		if err != nil {
			log.Printf("plugin/dnsfilter: %s\n", err)
			return dns.RcodeServerFailure, dnsfilter.Result{}, fmt.Errorf("plugin/dnsfilter: %s", err)
		}

		if result.IsFiltered {
			switch result.Reason {
			case dnsfilter.FilteredSafeBrowsing:
				// return cname safebrowsing.block.dns.adguard.com
				val := p.SafeBrowsingBlockHost
				rcode, err := p.replaceHostWithValAndReply(ctx, w, r, host, val, question)
				if err != nil {
					return rcode, dnsfilter.Result{}, err
				}
				return rcode, result, err
			case dnsfilter.FilteredParental:
				// return cname family.block.dns.adguard.com
				val := p.ParentalBlockHost
				rcode, err := p.replaceHostWithValAndReply(ctx, w, r, host, val, question)
				if err != nil {
					return rcode, dnsfilter.Result{}, err
				}
				return rcode, result, err
			case dnsfilter.FilteredBlackList:
				// return NXdomain
				rcode, err := writeNXdomain(ctx, w, r)
				if err != nil {
					return rcode, dnsfilter.Result{}, err
				}
				return rcode, result, err
			default:
				log.Printf("SHOULD NOT HAPPEN -- got unknown reason for filtering: %T %v %s", result.Reason, result.Reason, result.Reason.String())
			}
		} else {
			switch result.Reason {
			case dnsfilter.NotFilteredWhiteList:
				rcode, err := plugin.NextOrFailure(p.Name(), p.Next, ctx, w, r)
				return rcode, result, err
			case dnsfilter.NotFilteredNotFound:
				// do nothing, pass through to lower code
			default:
				log.Printf("SHOULD NOT HAPPEN -- got unknown reason for not filtering: %T %v %s", result.Reason, result.Reason, result.Reason.String())
			}
		}
	}
	rcode, err := plugin.NextOrFailure(p.Name(), p.Next, ctx, w, r)
	return rcode, dnsfilter.Result{}, err
}

// ServeDNS handles the DNS request and refuses if it's in filterlists
func (p *plug) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	start := time.Now()
	requests.Inc()
	state := request.Request{W: w, Req: r}
	ip := state.IP()

	// capture the written answer
	rrw := dnstest.NewRecorder(w)
	rcode, result, err := p.serveDNSInternal(ctx, rrw, r)
	if rcode > 0 {
		// actually send the answer if we have one
		answer := new(dns.Msg)
		answer.SetRcode(r, rcode)
		state.SizeAndDo(answer)
		err = w.WriteMsg(answer)
		if err != nil {
			return dns.RcodeServerFailure, err
		}
	}

	// increment counters
	switch {
	case err != nil:
		errorsTotal.Inc()
	case result.Reason == dnsfilter.FilteredBlackList:
		filtered.Inc()
		filteredLists.Inc()
	case result.Reason == dnsfilter.FilteredSafeBrowsing:
		filtered.Inc()
		filteredSafebrowsing.Inc()
	case result.Reason == dnsfilter.FilteredParental:
		filtered.Inc()
		filteredParental.Inc()
	case result.Reason == dnsfilter.FilteredInvalid:
		filtered.Inc()
		filteredInvalid.Inc()
	case result.Reason == dnsfilter.FilteredSafeSearch:
		// the request was passsed through but not filtered, don't increment filtered
		safesearch.Inc()
	case result.Reason == dnsfilter.NotFilteredWhiteList:
		whitelisted.Inc()
	case result.Reason == dnsfilter.NotFilteredNotFound:
		// do nothing
	case result.Reason == dnsfilter.NotFilteredError:
		text := "SHOULD NOT HAPPEN: got DNSFILTER_NOTFILTERED_ERROR without err != nil!"
		log.Println(text)
		err = errors.New(text)
		rcode = dns.RcodeServerFailure
	}

	// log
	if p.QueryLogEnabled {
		logRequest(r, rrw.Msg, result, time.Since(start), ip)
	}
	return rcode, err
}

// Name returns name of the plugin as seen in Corefile and plugin.cfg
func (p *plug) Name() string { return "dnsfilter" }

var onceQueryLog sync.Once
