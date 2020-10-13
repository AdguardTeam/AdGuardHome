package dnsforward

import (
	"net"
	"strings"
	"sync"

	"github.com/AdguardTeam/golibs/log"
	"github.com/digineo/go-ipset/v2"
	"github.com/mdlayher/netlink"
	"github.com/miekg/dns"
	"github.com/ti-mo/netfilter"
)

type ipsetProps struct {
	name    string
	family  netfilter.ProtoFamily
	comment bool
}

type ipsetCtx struct {
	ipsetMap  map[string]ipsetProps   // ipset -> props
	domainMap map[string][]ipsetProps // domain -> ipsets
	ipv4Cache map[[4]byte]struct{}
	ipv4Mutex *sync.RWMutex
	ipv6Cache map[[16]byte]struct{}
	ipv6Mutex *sync.RWMutex

	ipv4Conn *ipset.Conn
	ipv6Conn *ipset.Conn
}

func (c *ipsetCtx) clearCache() {
	c.ipv4Cache = make(map[[4]byte]struct{})
	c.ipv6Cache = make(map[[16]byte]struct{})
}

func (c *ipsetCtx) dialNetfilterSockets(config *netlink.Config) error {
	var err error

	// the kernel API does not actually require 2 sockets but the
	// digineo/go-ipset lib does and that's acceptable -- it's not
	// even clear that the family needs to be correct for our use
	// cases
	c.ipv4Conn, err = ipset.Dial(netfilter.ProtoIPv4, config)
	if err != nil {
		return err
	}
	c.ipv6Conn, err = ipset.Dial(netfilter.ProtoIPv6, config)
	if err != nil {
		return err
	}
	return nil
}

func (c *ipsetCtx) queryIpsetProps(name string) (ipsetProps, error) {
	// doesn't matter the family we use for the header query
	set, err := c.ipv4Conn.ListHeader(name)
	if err != nil {
		return ipsetProps{}, err
	}

	var family netfilter.ProtoFamily
	if set != nil && set.Family != nil {
		val := netfilter.ProtoFamily(set.Family.Value)
		if val == netfilter.ProtoIPv4 || val == netfilter.ProtoIPv6 {
			family = val
		}
	}

	var comment bool
	if set.CreateData != nil && set.CreateData.CadtFlags != nil &&
		(set.CreateData.CadtFlags.Value&uint32(ipset.WithComment)) != 0 {
		comment = true
	}

	return ipsetProps{name, family, comment}, nil
}

func (c *ipsetCtx) getIpsets(names []string) []ipsetProps {
	ipsets := make([]ipsetProps, 0, 2)
	for _, name := range names {
		ipset := c.ipsetMap[name]
		if ipset.name == "" {
			var err error
			ipset, err = c.queryIpsetProps(name)
			if err != nil {
				log.Info("IPSET: error querying ipset '%s': %s", name, err)
				continue
			} else if ipset.family == netfilter.ProtoUnspec {
				log.Info("IPSET: could not determine protocol family of ipset '%s'",
					name)
				continue
			} else {
				c.ipsetMap[name] = ipset
			}
		}
		ipsets = append(ipsets, ipset)
	}
	return ipsets
}

func parseIpsetConfig(cfgStr string) ([]string, []string) {
	cfgStr = strings.TrimSpace(cfgStr)
	hostsAndNames := strings.Split(cfgStr, "/")
	if len(hostsAndNames) != 2 {
		log.Info("IPSET: invalid value '%s' (need exactly one /)", cfgStr)
		return nil, nil
	}

	hosts := strings.Split(hostsAndNames[0], ",")
	ipsetNames := strings.Split(hostsAndNames[1], ",")

	if len(ipsetNames) == 0 {
		log.Info("IPSET: resolutions for %v will not be stored", hosts)
	}

	for i := range ipsetNames {
		ipsetNames[i] = strings.TrimSpace(ipsetNames[i])
		if len(ipsetNames[i]) == 0 {
			log.Info("IPSET: invalid value '%s' (zero length ipset name)", cfgStr)
			return nil, nil
		}
	}

	for i := range hosts {
		hosts[i] = strings.TrimSpace(hosts[i])
		hosts[i] = strings.ToLower(hosts[i])
		if len(hosts[i]) == 0 {
			log.Info("IPSET: root catchall in %v", ipsetNames)
		}
	}

	return hosts, ipsetNames
}

// Convert configuration settings to an internal map and check ipsets
// DOMAIN[,DOMAIN].../IPSET1_NAME[,IPSET2_NAME]...
// config parameter may be nil
func (c *ipsetCtx) init(ipsetConfig []string, config *netlink.Config) error {
	c.ipsetMap = make(map[string]ipsetProps)
	c.domainMap = make(map[string][]ipsetProps)
	c.ipv4Mutex = &sync.RWMutex{}
	c.ipv6Mutex = &sync.RWMutex{}
	c.clearCache()

	if config == nil {
		config = &netlink.Config{}
	}
	err := c.dialNetfilterSockets(config)
	if err != nil {
		return err
	}

	for _, cfgStr := range ipsetConfig {
		hosts, ipsetNames := parseIpsetConfig(cfgStr)

		ipsets := c.getIpsets(ipsetNames)

		for _, host := range hosts {
			c.domainMap[host] = append(c.domainMap[host], ipsets...)
		}
	}
	log.Debug("IPSET: added %d domains for %d ipsets", len(c.domainMap), len(c.ipsetMap))

	return nil
}

func (c *ipsetCtx) Uninit() error {
	errv4 := c.ipv4Conn.Close()
	errv6 := c.ipv6Conn.Close()
	if errv4 != nil {
		return errv4
	}
	if errv6 != nil {
		return errv6
	}
	return nil
}

func (c *ipsetCtx) getIP(rr dns.RR) net.IP {
	switch a := rr.(type) {
	case *dns.A:
		var ip4 [4]byte
		copy(ip4[:], a.A.To4())
		c.ipv4Mutex.Lock()
		defer c.ipv4Mutex.Unlock()
		_, found := c.ipv4Cache[ip4]
		if found {
			return nil // this IP was added before
		}
		c.ipv4Cache[ip4] = struct{}{}
		return a.A

	case *dns.AAAA:
		var ip6 [16]byte
		copy(ip6[:], a.AAAA.To16())
		c.ipv6Mutex.Lock()
		defer c.ipv6Mutex.Unlock()
		_, found := c.ipv6Cache[ip6]
		if found {
			return nil // this IP was added before
		}
		c.ipv6Cache[ip6] = struct{}{}
		return a.AAAA

	default:
		return nil
	}
}

// Find the ipsets for a given host (accounting for subdomain wildcards)
func (c *ipsetCtx) lookupHost(host string) []ipsetProps {
	var ipsets []ipsetProps

	// Search for matching ipset hosts starting with most specific
	// subdomain. We could use a trie here but the simple,
	// inefficient solution isn't _that_ expensive (~75% for 10
	// subdomains vs 0 but still sub-microsecond on a Core i7, see
	// BenchmarkIpsetUnbound*).
	i := 0
	for i != -1 {
		host = host[i:]

		ipsets = c.domainMap[host]
		if ipsets != nil {
			break
		}

		// move slice up to the parent domain
		i = strings.Index(host, ".")
		if i == -1 { // check the root
			ipsets = c.domainMap[""]
		} else { // move past .
			i++
		}
	}

	return ipsets
}

func (c *ipsetCtx) getConn(set ipsetProps) *ipset.Conn {
	if set.family == netfilter.ProtoIPv4 {
		return c.ipv4Conn
	}
	return c.ipv6Conn
}

// IPs must be same family (v4/v6) as set's family
func (c *ipsetCtx) addIPs(host string, set ipsetProps, addrs []net.IP) {
	entries := make([]*ipset.Entry, 0, len(addrs))
	for _, ip := range addrs {
		var entry *ipset.Entry
		if set.comment {
			entry = ipset.NewEntry(ipset.EntryIP(ip), ipset.EntryComment(host))
		} else {
			entry = ipset.NewEntry(ipset.EntryIP(ip))
		}
		entries = append(entries, entry)
	}
	err := c.getConn(set).Add(set.name, entries...)
	if err != nil {
		log.Info("IPSET: %s%s -> %s: %s", host, addrs, set.name, err)
	}
	log.Debug("IPSET: added %s%s -> %s", host, addrs, set.name)
}

func addToIpset(c *ipsetCtx, host string, set ipsetProps, addrs []net.IP) {
	c.addIPs(host, set, addrs)
}

func ipsetNames(sets []ipsetProps) []string {
	names := make([]string, 0, len(sets))
	for _, set := range sets {
		names = append(names, set.name)
	}
	return names
}

// Compute which addresses to add to which ipsets for a particular DNS query response
// Call addEntry for each (host, ipset, ip) triple
func (c *ipsetCtx) processEntries(ctx *dnsContext, addEntries func(*ipsetCtx, string, ipsetProps, []net.IP)) int {
	req := ctx.proxyCtx.Req
	if req == nil || len(c.domainMap) == 0 || !ctx.responseFromUpstream ||
		!(req.Question[0].Qtype == dns.TypeA ||
			req.Question[0].Qtype == dns.TypeAAAA ||
			req.Question[0].Qtype == dns.TypeANY) {
		return resultDone
	}

	host := req.Question[0].Name
	host = strings.TrimSuffix(host, ".")
	host = strings.ToLower(host)
	ipsets := c.lookupHost(host)
	if ipsets == nil {
		return resultDone
	}

	// don't bother building the ipset name list if it'll be thrown away
	if log.GetLevel() >= log.DEBUG {
		log.Debug("IPSET: found ipsets %v for host %s", ipsetNames(ipsets), host)
	}

	if ctx.proxyCtx.Res != nil {
		v4s := make([]net.IP, 0, len(ctx.proxyCtx.Res.Answer))
		v6s := make([]net.IP, 0, len(ctx.proxyCtx.Res.Answer))
		for _, it := range ctx.proxyCtx.Res.Answer {
			ip := c.getIP(it)
			if ip == nil {
				continue
			}
			if ip.To4() == nil {
				v6s = append(v6s, ip)
			} else {
				v4s = append(v4s, ip)
			}
		}
		for _, ipset := range ipsets {
			switch ipset.family {
			case netfilter.ProtoIPv4:
				if len(v4s) != 0 {
					addEntries(c, host, ipset, v4s)
				}
				continue
			case netfilter.ProtoIPv6:
				if len(v6s) != 0 {
					addEntries(c, host, ipset, v6s)
				}
				continue
			}
		}
	}

	return resultDone
}

// Add IP addresses of the specified in configuration domain names to an ipset list
func (c *ipsetCtx) process(ctx *dnsContext) int {
	return c.processEntries(ctx, addToIpset)
}
