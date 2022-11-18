//go:build linux

package aghnet

import (
	"fmt"
	"net"
	"strings"
	"sync"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/digineo/go-ipset/v2"
	"github.com/mdlayher/netlink"
	"github.com/ti-mo/netfilter"
	"golang.org/x/sys/unix"
)

// How to test on a real Linux machine:
//
//  1. Run "sudo ipset create example_set hash:ip family ipv4".
//
//  2. Run "sudo ipset list example_set".  The Members field should be empty.
//
//  3. Add the line "example.com/example_set" to your AdGuardHome.yaml.
//
//  4. Start AdGuardHome.
//
//  5. Make requests to example.com and its subdomains.
//
//  6. Run "sudo ipset list example_set".  The Members field should contain the
//     resolved IP addresses.

// newIpsetMgr returns a new Linux ipset manager.
func newIpsetMgr(ipsetConf []string) (set IpsetManager, err error) {
	return newIpsetMgrWithDialer(ipsetConf, defaultDial)
}

// defaultDial is the default netfilter dialing function.
func defaultDial(pf netfilter.ProtoFamily, conf *netlink.Config) (conn ipsetConn, err error) {
	conn, err = ipset.Dial(pf, conf)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

// ipsetConn is the ipset conn interface.
type ipsetConn interface {
	Add(name string, entries ...*ipset.Entry) (err error)
	Close() (err error)
	Header(name string) (p *ipset.HeaderPolicy, err error)
}

// ipsetDialer creates an ipsetConn.
type ipsetDialer func(pf netfilter.ProtoFamily, conf *netlink.Config) (conn ipsetConn, err error)

// ipsetProps contains one Linux Netfilter ipset properties.
type ipsetProps struct {
	name   string
	family netfilter.ProtoFamily
}

// unit is a convenient alias for struct{}.
type unit = struct{}

// ipsInIpset is the type of a set of IP-address-to-ipset mappings.
type ipsInIpset map[ipInIpsetEntry]unit

// ipInIpsetEntry is the type for entries in an ipsInIpset set.
type ipInIpsetEntry struct {
	ipsetName string
	ipArr     [net.IPv6len]byte
}

// ipsetMgr is the Linux Netfilter ipset manager.
type ipsetMgr struct {
	nameToIpset    map[string]ipsetProps
	domainToIpsets map[string][]ipsetProps

	dial ipsetDialer

	// mu protects all properties below.
	mu *sync.Mutex

	// TODO(a.garipov): Currently, the ipset list is static, and we don't
	// read the IPs already in sets, so we can assume that all incoming IPs
	// are either added to all corresponding ipsets or not.  When that stops
	// being the case, for example if we add dynamic reconfiguration of
	// ipsets, this map will need to become a per-ipset-name one.
	addedIPs ipsInIpset

	ipv4Conn ipsetConn
	ipv6Conn ipsetConn
}

// dialNetfilter establishes connections to Linux's netfilter module.
func (m *ipsetMgr) dialNetfilter(conf *netlink.Config) (err error) {
	// The kernel API does not actually require two sockets but package
	// github.com/digineo/go-ipset does.
	//
	// TODO(a.garipov): Perhaps we can ditch package ipset altogether and just
	// use packages netfilter and netlink.
	m.ipv4Conn, err = m.dial(netfilter.ProtoIPv4, conf)
	if err != nil {
		return fmt.Errorf("dialing v4: %w", err)
	}

	m.ipv6Conn, err = m.dial(netfilter.ProtoIPv6, conf)
	if err != nil {
		return fmt.Errorf("dialing v6: %w", err)
	}

	return nil
}

// parseIpsetConfig parses one ipset configuration string.
func parseIpsetConfig(confStr string) (hosts, ipsetNames []string, err error) {
	confStr = strings.TrimSpace(confStr)
	hostsAndNames := strings.Split(confStr, "/")
	if len(hostsAndNames) != 2 {
		return nil, nil, fmt.Errorf("invalid value %q: expected one slash", confStr)
	}

	hosts = strings.Split(hostsAndNames[0], ",")
	ipsetNames = strings.Split(hostsAndNames[1], ",")

	if len(ipsetNames) == 0 {
		return nil, nil, nil
	}

	for i := range ipsetNames {
		ipsetNames[i] = strings.TrimSpace(ipsetNames[i])
		if len(ipsetNames[i]) == 0 {
			return nil, nil, fmt.Errorf("invalid value %q: empty ipset name", confStr)
		}
	}

	for i := range hosts {
		hosts[i] = strings.ToLower(strings.TrimSpace(hosts[i]))
	}

	return hosts, ipsetNames, nil
}

// ipsetProps returns the properties of an ipset with the given name.
func (m *ipsetMgr) ipsetProps(name string) (set ipsetProps, err error) {
	// The family doesn't seem to matter when we use a header query, so
	// query only the IPv4 one.
	//
	// TODO(a.garipov): Find out if this is a bug or a feature.
	var res *ipset.HeaderPolicy
	res, err = m.ipv4Conn.Header(name)
	if err != nil {
		return set, err
	}

	if res == nil || res.Family == nil {
		return set, errors.Error("empty response or no family data")
	}

	family := netfilter.ProtoFamily(res.Family.Value)
	if family != netfilter.ProtoIPv4 && family != netfilter.ProtoIPv6 {
		return set, fmt.Errorf("unexpected ipset family %d", family)
	}

	return ipsetProps{
		name:   name,
		family: family,
	}, nil
}

// ipsets returns currently known ipsets.
func (m *ipsetMgr) ipsets(names []string) (sets []ipsetProps, err error) {
	for _, name := range names {
		set, ok := m.nameToIpset[name]
		if ok {
			sets = append(sets, set)

			continue
		}

		set, err = m.ipsetProps(name)
		if err != nil {
			return nil, fmt.Errorf("querying ipset %q: %w", name, err)
		}

		m.nameToIpset[name] = set
		sets = append(sets, set)
	}

	return sets, nil
}

// newIpsetMgrWithDialer returns a new Linux ipset manager using the provided
// dialer.
func newIpsetMgrWithDialer(ipsetConf []string, dial ipsetDialer) (mgr IpsetManager, err error) {
	defer func() { err = errors.Annotate(err, "ipset: %w") }()

	m := &ipsetMgr{
		mu: &sync.Mutex{},

		nameToIpset:    make(map[string]ipsetProps),
		domainToIpsets: make(map[string][]ipsetProps),

		dial: dial,

		addedIPs: make(ipsInIpset),
	}

	err = m.dialNetfilter(&netlink.Config{})
	if err != nil {
		if errors.Is(err, unix.EPROTONOSUPPORT) {
			// The implementation doesn't support this protocol version.  Just
			// issue a warning.
			log.Info("ipset: dialing netfilter: warning: %s", err)

			return nil, nil
		}

		return nil, fmt.Errorf("dialing netfilter: %w", err)
	}

	for i, confStr := range ipsetConf {
		var hosts, ipsetNames []string
		hosts, ipsetNames, err = parseIpsetConfig(confStr)
		if err != nil {
			return nil, fmt.Errorf("config line at idx %d: %w", i, err)
		}

		var ipsets []ipsetProps
		ipsets, err = m.ipsets(ipsetNames)
		if err != nil {
			return nil, fmt.Errorf(
				"getting ipsets from config line at idx %d: %w",
				i,
				err,
			)
		}

		for _, host := range hosts {
			m.domainToIpsets[host] = append(m.domainToIpsets[host], ipsets...)
		}
	}

	return m, nil
}

// lookupHost find the ipsets for the host, taking subdomain wildcards into
// account.
func (m *ipsetMgr) lookupHost(host string) (sets []ipsetProps) {
	// Search for matching ipset hosts starting with most specific domain.
	// We could use a trie here but the simple, inefficient solution isn't
	// that expensive: ~10 ns for TLD + SLD vs. ~140 ns for 10 subdomains on
	// an AMD Ryzen 7 PRO 4750U CPU; ~120 ns vs. ~ 1500 ns on a Raspberry
	// Pi's ARMv7 rev 4 CPU.
	for i := 0; ; i++ {
		host = host[i:]
		sets = m.domainToIpsets[host]
		if sets != nil {
			return sets
		}

		i = strings.Index(host, ".")
		if i == -1 {
			break
		}
	}

	// Check the root catch-all one.
	return m.domainToIpsets[""]
}

// addIPs adds the IP addresses for the host to the ipset.  set must be same
// family as set's family.
func (m *ipsetMgr) addIPs(host string, set ipsetProps, ips []net.IP) (n int, err error) {
	if len(ips) == 0 {
		return 0, nil
	}

	var entries []*ipset.Entry
	var newAddedEntries []ipInIpsetEntry
	for _, ip := range ips {
		e := ipInIpsetEntry{
			ipsetName: set.name,
		}
		copy(e.ipArr[:], ip.To16())

		if _, added := m.addedIPs[e]; added {
			continue
		}

		entries = append(entries, ipset.NewEntry(ipset.EntryIP(ip)))
		newAddedEntries = append(newAddedEntries, e)
	}

	n = len(entries)
	if n == 0 {
		return 0, nil
	}

	var conn ipsetConn
	switch set.family {
	case netfilter.ProtoIPv4:
		conn = m.ipv4Conn
	case netfilter.ProtoIPv6:
		conn = m.ipv6Conn
	default:
		return 0, fmt.Errorf("unexpected family %s for ipset %q", set.family, set.name)
	}

	err = conn.Add(set.name, entries...)
	if err != nil {
		return 0, fmt.Errorf("adding %q%s to ipset %q: %w", host, ips, set.name, err)
	}

	// Only add these to the cache once we're sure that all of them were
	// actually sent to the ipset.
	for _, e := range newAddedEntries {
		m.addedIPs[e] = unit{}
	}

	return n, nil
}

// addToSets adds the IP addresses to the corresponding ipset.
func (m *ipsetMgr) addToSets(
	host string,
	ip4s []net.IP,
	ip6s []net.IP,
	sets []ipsetProps,
) (n int, err error) {
	for _, set := range sets {
		var nn int
		switch set.family {
		case netfilter.ProtoIPv4:
			nn, err = m.addIPs(host, set, ip4s)
			if err != nil {
				return n, err
			}
		case netfilter.ProtoIPv6:
			nn, err = m.addIPs(host, set, ip6s)
			if err != nil {
				return n, err
			}
		default:
			return n, fmt.Errorf("unexpected family %s for ipset %q", set.family, set.name)
		}

		log.Debug("ipset: added %d ips to set %s", nn, set.name)

		n += nn
	}

	return n, nil
}

// Add implements the IpsetManager interface for *ipsetMgr
func (m *ipsetMgr) Add(host string, ip4s, ip6s []net.IP) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	sets := m.lookupHost(host)
	if len(sets) == 0 {
		return 0, nil
	}

	log.Debug("ipset: found %d sets", len(sets))

	return m.addToSets(host, ip4s, ip6s, sets)
}

// Close implements the IpsetManager interface for *ipsetMgr.
func (m *ipsetMgr) Close() (err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []error

	// Close both and collect errors so that the errors from closing one
	// don't interfere with closing the other.
	err = m.ipv4Conn.Close()
	if err != nil {
		errs = append(errs, err)
	}

	err = m.ipv6Conn.Close()
	if err != nil {
		errs = append(errs, err)
	}

	if len(errs) != 0 {
		return errors.List("closing ipsets", errs...)
	}

	return nil
}
