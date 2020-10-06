package dnsforward

import (
	"net"
	"strings"

	"github.com/AdguardTeam/AdGuardHome/util"
	"github.com/AdguardTeam/golibs/log"
	"github.com/miekg/dns"
)

type ipsetCtx struct {
	ipsetList   map[string][]string // domain -> []ipset_name
	ipsetCache  map[[4]byte]bool    // cache for IP[] to prevent duplicate calls to ipset program
	ipset6Cache map[[16]byte]bool   // cache for IP[] to prevent duplicate calls to ipset program
}

// Convert configuration settings to an internal map
// DOMAIN[,DOMAIN].../IPSET1_NAME[,IPSET2_NAME]...
func (c *ipsetCtx) init(ipsetConfig []string) {
	c.ipsetList = make(map[string][]string)
	c.ipsetCache = make(map[[4]byte]bool)
	c.ipset6Cache = make(map[[16]byte]bool)

	for _, it := range ipsetConfig {
		it = strings.TrimSpace(it)
		hostsAndNames := strings.Split(it, "/")
		if len(hostsAndNames) != 2 {
			log.Debug("IPSET: invalid value '%s'", it)
			continue
		}

		ipsetNames := strings.Split(hostsAndNames[1], ",")
		if len(ipsetNames) == 0 {
			log.Debug("IPSET: invalid value '%s'", it)
			continue
		}
		bad := false
		for i := range ipsetNames {
			ipsetNames[i] = strings.TrimSpace(ipsetNames[i])
			if len(ipsetNames[i]) == 0 {
				bad = true
				break
			}
		}
		if bad {
			log.Debug("IPSET: invalid value '%s'", it)
			continue
		}

		hosts := strings.Split(hostsAndNames[0], ",")
		for _, host := range hosts {
			host = strings.TrimSpace(host)
			host = strings.ToLower(host)
			if len(host) == 0 {
				log.Debug("IPSET: invalid value '%s'", it)
				continue
			}
			c.ipsetList[host] = ipsetNames
		}
	}
	log.Debug("IPSET: added %d hosts", len(c.ipsetList))
}

func (c *ipsetCtx) getIP(rr dns.RR) net.IP {
	switch a := rr.(type) {
	case *dns.A:
		var ip4 [4]byte
		copy(ip4[:], a.A.To4())
		_, found := c.ipsetCache[ip4]
		if found {
			return nil // this IP was added before
		}
		c.ipsetCache[ip4] = false
		return a.A

	case *dns.AAAA:
		var ip6 [16]byte
		copy(ip6[:], a.AAAA)
		_, found := c.ipset6Cache[ip6]
		if found {
			return nil // this IP was added before
		}
		c.ipset6Cache[ip6] = false
		return a.AAAA

	default:
		return nil
	}
}

// Find the ipsets for a given host (accounting for subdomain wildcards)
func (c *ipsetCtx) getIpsetNames(host string) ([]string, bool) {
	var ipsetNames []string
	var found bool

	// search for matching ipset hosts starting with most specific subdomain
	i := 0
	for i != -1 {
		host = host[i:]

		ipsetNames, found = c.ipsetList[host]
		if found {
			break
		}

		// move slice up to the parent domain
		i = strings.Index(host, ".")
		if i != -1 {
			i++
		}
	}

	return ipsetNames, found
}

func addToIpset(host string, ipsetName string, ipStr string) {
	code, out, err := util.RunCommand("ipset", "add", ipsetName, ipStr)
	if err != nil {
		log.Info("IPSET: %s(%s) -> %s: %s", host, ipStr, ipsetName, err)
		return
	}
	if code != 0 {
		log.Info("IPSET: ipset add:  code:%d  output:'%s'", code, out)
		return
	}
	log.Debug("IPSET: added %s(%s) -> %s", host, ipStr, ipsetName)
}

// Compute which addresses to add to which ipsets for a particular DNS query response
// Call addMember for each (host, ipset, ip) triple
func (c *ipsetCtx) processMembers(ctx *dnsContext, addMember func(string, string, string)) int {
	req := ctx.proxyCtx.Req
	if !(req.Question[0].Qtype == dns.TypeA ||
		req.Question[0].Qtype == dns.TypeAAAA) ||
		!ctx.responseFromUpstream {
		return resultDone
	}

	host := req.Question[0].Name
	host = strings.TrimSuffix(host, ".")
	host = strings.ToLower(host)
	ipsetNames, found := c.getIpsetNames(host)
	if !found {
		return resultDone
	}

	log.Debug("IPSET: found ipsets %v for host %s", ipsetNames, host)

	if ctx.proxyCtx.Res != nil {
		for _, it := range ctx.proxyCtx.Res.Answer {
			ip := c.getIP(it)
			if ip == nil {
				continue
			}

			ipStr := ip.String()
			for _, name := range ipsetNames {
				addMember(host, name, ipStr)
			}
		}
	}

	return resultDone
}

// Add IP addresses of the specified in configuration domain names to an ipset list
func (c *ipsetCtx) process(ctx *dnsContext) int {
	return c.processMembers(ctx, addToIpset)
}
