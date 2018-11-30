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

	"github.com/AdguardTeam/AdGuardHome/dnsfilter"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/plugin/metrics"
	"github.com/coredns/coredns/plugin/pkg/dnstest"
	"github.com/coredns/coredns/plugin/pkg/upstream"
	"github.com/coredns/coredns/request"
	"github.com/mholt/caddy"
	"github.com/miekg/dns"
	"github.com/prometheus/client_golang/prometheus"
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

type plugFilter struct {
	ID   int64
	Path string
}

type plugSettings struct {
	SafeBrowsingBlockHost string
	ParentalBlockHost     string
	QueryLogEnabled       bool
	BlockedTTL            uint32 // in seconds, default 3600
	Filters               []plugFilter
}

type plug struct {
	d        *dnsfilter.Dnsfilter
	Next     plugin.Handler
	upstream upstream.Upstream
	settings plugSettings

	sync.RWMutex
}

var defaultPluginSettings = plugSettings{
	SafeBrowsingBlockHost: "safebrowsing.block.dns.adguard.com",
	ParentalBlockHost:     "family.block.dns.adguard.com",
	BlockedTTL:            3600, // in seconds
	Filters:               make([]plugFilter, 0),
}

//
// coredns handling functions
//
func setupPlugin(c *caddy.Controller) (*plug, error) {
	// create new Plugin and copy default values
	p := &plug{
		settings: defaultPluginSettings,
		d:        dnsfilter.New(nil),
	}

	log.Println("Initializing the CoreDNS plugin")

	for c.Next() {
		for c.NextBlock() {
			blockValue := c.Val()
			switch blockValue {
			case "safebrowsing":
				log.Println("Browsing security service is enabled")
				p.d.SafeBrowsingEnabled = true
				if c.NextArg() {
					if len(c.Val()) == 0 {
						return nil, c.ArgErr()
					}
					p.d.SetSafeBrowsingServer(c.Val())
				}
			case "safesearch":
				log.Println("Safe search is enabled")
				p.d.SafeSearchEnabled = true
			case "parental":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				sensitivity, err := strconv.Atoi(c.Val())
				if err != nil {
					return nil, c.ArgErr()
				}

				log.Println("Parental control is enabled")
				if !dnsfilter.IsParentalSensitivityValid(sensitivity) {
					return nil, dnsfilter.ErrInvalidParental
				}
				p.d.ParentalEnabled = true
				p.d.ParentalSensitivity = sensitivity
				if c.NextArg() {
					if len(c.Val()) == 0 {
						return nil, c.ArgErr()
					}
					p.settings.ParentalBlockHost = c.Val()
				}
			case "blocked_ttl":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				blockedTtl, err := strconv.ParseUint(c.Val(), 10, 32)
				if err != nil {
					return nil, c.ArgErr()
				}
				log.Printf("Blocked request TTL is %d", blockedTtl)
				p.settings.BlockedTTL = uint32(blockedTtl)
			case "querylog":
				log.Println("Query log is enabled")
				p.settings.QueryLogEnabled = true
			case "filter":
				if !c.NextArg() {
					return nil, c.ArgErr()
				}

				filterId, err := strconv.ParseInt(c.Val(), 10, 64)
				if err != nil {
					return nil, c.ArgErr()
				}
				if !c.NextArg() {
					return nil, c.ArgErr()
				}
				filterPath := c.Val()

				// Initialize filter and add it to the list
				p.settings.Filters = append(p.settings.Filters, plugFilter{
					ID:   filterId,
					Path: filterPath,
				})
			}
		}
	}

	for _, filter := range p.settings.Filters {
		log.Printf("Loading rules from %s", filter.Path)

		file, err := os.Open(filter.Path)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		count := 0
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			text := scanner.Text()

			err = p.d.AddRule(text, filter.ID)
			if err == dnsfilter.ErrAlreadyExists || err == dnsfilter.ErrInvalidSyntax {
				continue
			}
			if err != nil {
				log.Printf("Cannot add rule %s: %s", text, err)
				// Just ignore invalid rules
				continue
			}
			count++
		}
		log.Printf("Added %d rules from filter ID=%d", count, filter.ID)

		if err = scanner.Err(); err != nil {
			return nil, err
		}
	}

	log.Printf("Loading stats from querylog")
	err := fillStatsFromQueryLog()
	if err != nil {
		log.Printf("Failed to load stats from querylog: %s", err)
		return nil, err
	}

	if p.settings.QueryLogEnabled {
		onceQueryLog.Do(func() {
			go periodicQueryLogRotate()
			go periodicHourlyTopRotate()
			go statsRotator()
		})
	}

	onceHook.Do(func() {
		caddy.RegisterEventHook("dnsfilter-reload", hook)
	})

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
			x.MustRegister(elapsedTime)
			x.MustRegister(p)
		}
		return nil
	})
	c.OnShutdown(p.onShutdown)
	c.OnFinalShutdown(p.onFinalShutdown)

	return nil
}

func (p *plug) onShutdown() error {
	p.Lock()
	p.d.Destroy()
	p.d = nil
	p.Unlock()
	return nil
}

func (p *plug) onFinalShutdown() error {
	logBufferLock.Lock()
	err := flushToFile(logBuffer)
	if err != nil {
		log.Printf("failed to flush to file: %s", err)
		return err
	}
	logBufferLock.Unlock()
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
	p.RLock()
	stats := p.d.GetStats()
	doStatsLookup(ch, doFunc, "safebrowsing", &stats.Safebrowsing)
	doStatsLookup(ch, doFunc, "parental", &stats.Parental)
	p.RUnlock()
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
	// log.Println("Will give", val, "instead of", host) // debug logging
	if addr != nil {
		// this is an IP address, return it
		result, err := dns.NewRR(fmt.Sprintf("%s %d A %s", host, p.settings.BlockedTTL, val))
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
func (p *plug) genSOA(r *dns.Msg) []dns.RR {
	zone := r.Question[0].Name
	header := dns.RR_Header{Name: zone, Rrtype: dns.TypeSOA, Ttl: p.settings.BlockedTTL, Class: dns.ClassINET}

	Mbox := "hostmaster."
	if zone[0] != '.' {
		Mbox += zone
	}
	Ns := "fake-for-negative-caching.adguard.com."

	soa := *defaultSOA
	soa.Hdr = header
	soa.Mbox = Mbox
	soa.Ns = Ns
	soa.Serial = 100500 // faster than uint32(time.Now().Unix())
	return []dns.RR{&soa}
}

func (p *plug) writeNXdomain(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r, Context: ctx}
	m := new(dns.Msg)
	m.SetRcode(state.Req, dns.RcodeNameError)
	m.Authoritative, m.RecursionAvailable, m.Compress = true, true, true
	m.Ns = p.genSOA(r)

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
		return dns.RcodeFormatError, dnsfilter.Result{}, fmt.Errorf("got a DNS request with more than one Question")
	}
	for _, question := range r.Question {
		host := strings.ToLower(strings.TrimSuffix(question.Name, "."))
		// is it a safesearch domain?
		p.RLock()
		if val, ok := p.d.SafeSearchDomain(host); ok {
			rcode, err := p.replaceHostWithValAndReply(ctx, w, r, host, val, question)
			if err != nil {
				p.RUnlock()
				return rcode, dnsfilter.Result{}, err
			}
			p.RUnlock()
			return rcode, dnsfilter.Result{Reason: dnsfilter.FilteredSafeSearch}, err
		}
		p.RUnlock()

		// needs to be filtered instead
		p.RLock()
		result, err := p.d.CheckHost(host)
		if err != nil {
			log.Printf("plugin/dnsfilter: %s\n", err)
			p.RUnlock()
			return dns.RcodeServerFailure, dnsfilter.Result{}, fmt.Errorf("plugin/dnsfilter: %s", err)
		}
		p.RUnlock()

		if result.IsFiltered {
			switch result.Reason {
			case dnsfilter.FilteredSafeBrowsing:
				// return cname safebrowsing.block.dns.adguard.com
				val := p.settings.SafeBrowsingBlockHost
				rcode, err := p.replaceHostWithValAndReply(ctx, w, r, host, val, question)
				if err != nil {
					return rcode, dnsfilter.Result{}, err
				}
				return rcode, result, err
			case dnsfilter.FilteredParental:
				// return cname family.block.dns.adguard.com
				val := p.settings.ParentalBlockHost
				rcode, err := p.replaceHostWithValAndReply(ctx, w, r, host, val, question)
				if err != nil {
					return rcode, dnsfilter.Result{}, err
				}
				return rcode, result, err
			case dnsfilter.FilteredBlackList:

				if result.Ip == nil {
					// return NXDomain
					rcode, err := p.writeNXdomain(ctx, w, r)
					if err != nil {
						return rcode, dnsfilter.Result{}, err
					}
					return rcode, result, err
				} else {
					// This is a hosts-syntax rule
					rcode, err := p.replaceHostWithValAndReply(ctx, w, r, host, result.Ip.String(), question)
					if err != nil {
						return rcode, dnsfilter.Result{}, err
					}
					return rcode, result, err
				}
			case dnsfilter.FilteredInvalid:
				// return NXdomain
				rcode, err := p.writeNXdomain(ctx, w, r)
				if err != nil {
					return rcode, dnsfilter.Result{}, err
				}
				return rcode, result, err
			default:
				log.Printf("SHOULD NOT HAPPEN -- got unknown reason for filtering host \"%s\": %v, %+v", host, result.Reason, result)
			}
		} else {
			switch result.Reason {
			case dnsfilter.NotFilteredWhiteList:
				rcode, err := plugin.NextOrFailure(p.Name(), p.Next, ctx, w, r)
				return rcode, result, err
			case dnsfilter.NotFilteredNotFound:
				// do nothing, pass through to lower code
			default:
				log.Printf("SHOULD NOT HAPPEN -- got unknown reason for not filtering host \"%s\": %v, %+v", host, result.Reason, result)
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
	elapsed := time.Since(start)
	elapsedTime.Observe(elapsed.Seconds())
	if p.settings.QueryLogEnabled {
		logRequest(r, rrw.Msg, result, time.Since(start), ip)
	}
	return rcode, err
}

// Name returns name of the plugin as seen in Corefile and plugin.cfg
func (p *plug) Name() string { return "dnsfilter" }

var onceHook sync.Once
var onceQueryLog sync.Once
