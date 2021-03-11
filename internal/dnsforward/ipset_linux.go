// +build linux

package dnsforward

import (
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/AdguardTeam/AdGuardHome/internal/agherr"
	"github.com/AdguardTeam/golibs/log"
	"github.com/digineo/go-ipset/v2"
	"github.com/mdlayher/netlink"
	"github.com/miekg/dns"
	"github.com/ti-mo/netfilter"
)

// TODO(a.garipov): Cover with unit tests as well as document how to test it
// manually.  The original PR by @dsheets on Github contained an integration
// test, but unfortunately I didn't have the time to properly refactor it and
// check it in.
//
// See https://github.com/AdguardTeam/AdGuardHome/issues/2611.

// ipsetProps contains one Linux Netfilter ipset properties.
type ipsetProps struct {
	name   string
	family netfilter.ProtoFamily
}

// ipsetCtx is the Linux Netfilter ipset context.
type ipsetCtx struct {
	// mu protects all properties below.
	mu *sync.Mutex

	nameToIpset    map[string]ipsetProps
	domainToIpsets map[string][]ipsetProps

	// TODO(a.garipov): Currently, the ipset list is static, and we don't
	// read the IPs already in sets, so we can assume that all incoming IPs
	// are either added to all corresponding ipsets or not.  When that stops
	// being the case, for example if we add dynamic reconfiguration of
	// ipsets, this map will need to become a per-ipset-name one.
	addedIPs map[[16]byte]struct{}

	ipv4Conn *ipset.Conn
	ipv6Conn *ipset.Conn
}

// dialNetfilter establishes connections to Linux's netfilter module.
func (c *ipsetCtx) dialNetfilter(config *netlink.Config) (err error) {
	// The kernel API does not actually require two sockets but package
	// github.com/digineo/go-ipset does.
	//
	// TODO(a.garipov): Perhaps we can ditch package ipset altogether and
	// just use packages netfilter and netlink.
	c.ipv4Conn, err = ipset.Dial(netfilter.ProtoIPv4, config)
	if err != nil {
		return fmt.Errorf("dialing v4: %w", err)
	}

	c.ipv6Conn, err = ipset.Dial(netfilter.ProtoIPv6, config)
	if err != nil {
		return fmt.Errorf("dialing v6: %w", err)
	}

	return nil
}

// ipsetProps returns the properties of an ipset with the given name.
func (c *ipsetCtx) ipsetProps(name string) (set ipsetProps, err error) {
	// The family doesn't seem to matter when we use a header query, so
	// query only the IPv4 one.
	//
	// TODO(a.garipov): Find out if this is a bug or a feature.
	var res *ipset.HeaderPolicy
	res, err = c.ipv4Conn.Header(name)
	if err != nil {
		return set, err
	}

	if res == nil || res.Family == nil {
		return set, agherr.Error("empty response or no family data")
	}

	family := netfilter.ProtoFamily(res.Family.Value)
	if family != netfilter.ProtoIPv4 && family != netfilter.ProtoIPv6 {
		return set, fmt.Errorf("unexpected ipset family %s", family)
	}

	return ipsetProps{
		name:   name,
		family: family,
	}, nil
}

// ipsets returns currently known ipsets.
func (c *ipsetCtx) ipsets(names []string) (sets []ipsetProps, err error) {
	for _, name := range names {
		set, ok := c.nameToIpset[name]
		if ok {
			sets = append(sets, set)

			continue
		}

		set, err = c.ipsetProps(name)
		if err != nil {
			return nil, fmt.Errorf("querying ipset %q: %w", name, err)
		}

		c.nameToIpset[name] = set
		sets = append(sets, set)
	}

	return sets, nil
}

// parseIpsetConfig parses one ipset configuration string.
func parseIpsetConfig(cfgStr string) (hosts, ipsetNames []string, err error) {
	cfgStr = strings.TrimSpace(cfgStr)
	hostsAndNames := strings.Split(cfgStr, "/")
	if len(hostsAndNames) != 2 {
		return nil, nil, fmt.Errorf("invalid value %q: expected one slash", cfgStr)
	}

	hosts = strings.Split(hostsAndNames[0], ",")
	ipsetNames = strings.Split(hostsAndNames[1], ",")

	if len(ipsetNames) == 0 {
		log.Info("ipset: resolutions for %q will not be stored", hosts)

		return nil, nil, nil
	}

	for i := range ipsetNames {
		ipsetNames[i] = strings.TrimSpace(ipsetNames[i])
		if len(ipsetNames[i]) == 0 {
			return nil, nil, fmt.Errorf("invalid value %q: empty ipset name", cfgStr)
		}
	}

	for i := range hosts {
		hosts[i] = strings.TrimSpace(hosts[i])
		hosts[i] = strings.ToLower(hosts[i])
		if len(hosts[i]) == 0 {
			log.Info("ipset: root catchall in %q", ipsetNames)
		}
	}

	return hosts, ipsetNames, nil
}

// init initializes the ipset context.  It is not safe for concurrent use.
//
// TODO(a.garipov): Rewrite into a simple constructor?
func (c *ipsetCtx) init(ipsetConfig []string) (err error) {
	c.mu = &sync.Mutex{}
	c.nameToIpset = make(map[string]ipsetProps)
	c.domainToIpsets = make(map[string][]ipsetProps)
	c.addedIPs = make(map[[16]byte]struct{})

	err = c.dialNetfilter(&netlink.Config{})
	if err != nil {
		return fmt.Errorf("ipset: dialing netfilter: %w", err)
	}

	for i, cfgStr := range ipsetConfig {
		var hosts, ipsetNames []string
		hosts, ipsetNames, err = parseIpsetConfig(cfgStr)
		if err != nil {
			return fmt.Errorf("ipset: config line at index %d: %w", i, err)
		}

		var ipsets []ipsetProps
		ipsets, err = c.ipsets(ipsetNames)
		if err != nil {
			return fmt.Errorf("ipset: getting ipsets config line at index %d: %w", i, err)
		}

		for _, host := range hosts {
			c.domainToIpsets[host] = append(c.domainToIpsets[host], ipsets...)
		}
	}

	log.Debug("ipset: added %d domains for %d ipsets", len(c.domainToIpsets), len(c.nameToIpset))

	return nil
}

// Close closes the Linux Netfilter connections.
func (c *ipsetCtx) Close() (err error) {
	var errors []error
	if c.ipv4Conn != nil {
		err = c.ipv4Conn.Close()
		if err != nil {
			errors = append(errors, err)
		}
	}

	if c.ipv6Conn != nil {
		err = c.ipv6Conn.Close()
		if err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) != 0 {
		return agherr.Many("closing ipsets", errors...)
	}

	return nil
}

// ipFromRR returns an IP address from a DNS resource record.
func ipFromRR(rr dns.RR) (ip net.IP) {
	switch a := rr.(type) {
	case *dns.A:
		return a.A
	case *dns.AAAA:
		return a.AAAA
	default:
		return nil
	}
}

// lookupHost find the ipsets for the host, taking subdomain wildcards into
// account.
func (c *ipsetCtx) lookupHost(host string) (sets []ipsetProps) {
	// Search for matching ipset hosts starting with most specific
	// subdomain.  We could use a trie here but the simple, inefficient
	// solution isn't that expensive.  ~75 % for 10 subdomains vs 0, but
	// still sub-microsecond on a Core i7.
	//
	// TODO(a.garipov): Re-add benchmarks from the original PR.
	for i := 0; i != -1; i++ {
		host = host[i:]
		sets = c.domainToIpsets[host]
		if sets != nil {
			return sets
		}

		i = strings.Index(host, ".")
		if i == -1 {
			break
		}
	}

	// Check the root catch-all one.
	return c.domainToIpsets[""]
}

// addIPs adds the IP addresses for the host to the ipset.  set must be same
// family as set's family.
func (c *ipsetCtx) addIPs(host string, set ipsetProps, ips []net.IP) (err error) {
	if len(ips) == 0 {
		return
	}

	entries := make([]*ipset.Entry, 0, len(ips))
	for _, ip := range ips {
		entries = append(entries, ipset.NewEntry(ipset.EntryIP(ip)))
	}

	var conn *ipset.Conn
	switch set.family {
	case netfilter.ProtoIPv4:
		conn = c.ipv4Conn
	case netfilter.ProtoIPv6:
		conn = c.ipv6Conn
	default:
		return fmt.Errorf("unexpected family %s for ipset %q", set.family, set.name)
	}

	err = conn.Add(set.name, entries...)
	if err != nil {
		return fmt.Errorf("adding %q%s to ipset %q: %w", host, ips, set.name, err)
	}

	log.Debug("ipset: added %s%s to ipset %s", host, ips, set.name)

	return nil
}

// skipIpsetProcessing returns true when the ipset processing can be skipped for
// this request.
func (c *ipsetCtx) skipIpsetProcessing(ctx *dnsContext) (ok bool) {
	if len(c.domainToIpsets) == 0 || ctx == nil || !ctx.responseFromUpstream {
		return true
	}

	req := ctx.proxyCtx.Req
	if req == nil || len(req.Question) == 0 {
		return true
	}

	qt := req.Question[0].Qtype
	return qt != dns.TypeA && qt != dns.TypeAAAA && qt != dns.TypeANY
}

// process adds the resolved IP addresses to the domain's ipsets, if any.
func (c *ipsetCtx) process(ctx *dnsContext) (rc resultCode) {
	var err error

	if c == nil {
		return resultCodeSuccess
	}

	log.Debug("ipset: starting processing")

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.skipIpsetProcessing(ctx) {
		log.Debug("ipset: skipped processing for request")

		return resultCodeSuccess
	}

	req := ctx.proxyCtx.Req
	host := req.Question[0].Name
	host = strings.TrimSuffix(host, ".")
	host = strings.ToLower(host)
	sets := c.lookupHost(host)
	if len(sets) == 0 {
		log.Debug("ipset: no ipsets for host %s", host)

		return resultCodeSuccess
	}

	log.Debug("ipset: found ipsets %+v for host %s", sets, host)

	if ctx.proxyCtx.Res == nil {
		return resultCodeSuccess
	}

	ans := ctx.proxyCtx.Res.Answer
	l := len(ans)
	v4s := make([]net.IP, 0, l)
	v6s := make([]net.IP, 0, l)
	for _, rr := range ans {
		ip := ipFromRR(rr)
		if ip == nil {
			continue
		}

		var iparr [16]byte
		copy(iparr[:], ip.To16())
		if _, added := c.addedIPs[iparr]; added {
			continue
		}

		if ip.To4() == nil {
			v6s = append(v6s, ip)

			continue
		}

		v4s = append(v4s, ip)
	}

setLoop:
	for _, set := range sets {
		switch set.family {
		case netfilter.ProtoIPv4:
			err = c.addIPs(host, set, v4s)
			if err != nil {
				break setLoop
			}
		case netfilter.ProtoIPv6:
			err = c.addIPs(host, set, v6s)
			if err != nil {
				break setLoop
			}
		default:
			err = fmt.Errorf("unexpected family %s for ipset %q", set.family, set.name)
			break setLoop
		}
	}
	if err != nil {
		log.Error("ipset: adding host ips: %s", err)
	} else {
		log.Debug("ipset: processed %d new ips", len(v4s)+len(v6s))
	}

	for _, ip := range v4s {
		var iparr [16]byte
		copy(iparr[:], ip.To16())
		c.addedIPs[iparr] = struct{}{}
	}

	for _, ip := range v6s {
		var iparr [16]byte
		copy(iparr[:], ip.To16())
		c.addedIPs[iparr] = struct{}{}
	}

	return resultCodeSuccess
}
