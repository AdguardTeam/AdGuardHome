//go:build linux

package ipset

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"sync"

	"github.com/AdguardTeam/golibs/container"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
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

// newManager returns a new Linux ipset manager.
func newManager(ctx context.Context, conf *Config) (set Manager, err error) {
	return newManagerWithDialer(ctx, conf, defaultDial)
}

// defaultDial is the default netfilter dialing function.
func defaultDial(pf netfilter.ProtoFamily, conf *netlink.Config) (conn ipsetConn, err error) {
	c, err := ipset.Dial(pf, conf)
	if err != nil {
		return nil, err
	}

	return &queryConn{c}, nil
}

// queryConn is the [ipsetConn] implementation with listAll method, which
// returns the list of properties of all available ipsets.
type queryConn struct {
	*ipset.Conn
}

// type check
var _ ipsetConn = (*queryConn)(nil)

// listAll returns the list of properties of all available ipsets.
//
// TODO(s.chzhen):  Use https://github.com/vishvananda/netlink.
func (qc *queryConn) listAll() (sets []props, err error) {
	msg, err := netfilter.MarshalNetlink(
		netfilter.Header{
			// The family doesn't seem to matter.  See TODO on parseIpsetConfig.
			Family:      qc.Conn.Family,
			SubsystemID: netfilter.NFSubsysIPSet,
			MessageType: netfilter.MessageType(ipset.CmdList),
			Flags:       netlink.Request | netlink.Dump,
		},
		[]netfilter.Attribute{{
			Type: uint16(ipset.AttrProtocol),
			Data: []byte{ipset.Protocol},
		}},
	)
	if err != nil {
		return nil, fmt.Errorf("marshaling netlink msg: %w", err)
	}

	// We assume it's OK to call a method of an unexported type
	// [ipset.connector], since there is no negative effects.
	ms, err := qc.Conn.Conn.Query(msg)
	if err != nil {
		return nil, fmt.Errorf("querying netlink msg: %w", err)
	}

	for i, s := range ms {
		p := props{}
		err = p.unmarshalMessage(s)
		if err != nil {
			return nil, fmt.Errorf("unmarshaling netlink msg at index %d: %w", i, err)
		}

		sets = append(sets, p)
	}

	return sets, nil
}

// ipsetConn is the ipset conn interface.
type ipsetConn interface {
	Add(name string, entries ...*ipset.Entry) (err error)
	Close() (err error)
	Header(name string) (p *ipset.HeaderPolicy, err error)
	listAll() (sets []props, err error)
}

// dialer creates an ipsetConn.
type dialer func(pf netfilter.ProtoFamily, conf *netlink.Config) (conn ipsetConn, err error)

// props contains one Linux Netfilter ipset properties.
type props struct {
	// name of the ipset.
	name string

	// typeName of the ipset.
	typeName string

	// family of the IP addresses in the ipset.
	family netfilter.ProtoFamily

	// isPersistent indicates that ipset has no timeout parameter and all
	// entries are added permanently.
	isPersistent bool
}

// unmarshalMessage unmarshals netlink message and sets the properties of the
// ipset.
func (p *props) unmarshalMessage(msg netlink.Message) (err error) {
	_, attrs, err := netfilter.UnmarshalNetlink(msg)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return err
	}

	// By default ipset has no timeout parameter.
	p.isPersistent = true

	for _, a := range attrs {
		p.parseAttribute(a)
	}

	return nil
}

// parseAttribute parses netfilter attribute and sets the name and family of
// the ipset.
func (p *props) parseAttribute(a netfilter.Attribute) {
	switch ipset.AttributeType(a.Type) {
	case ipset.AttrData:
		p.parseAttrData(a)
	case ipset.AttrSetName:
		// Trim the null character.
		p.name = string(bytes.Trim(a.Data, "\x00"))
	case ipset.AttrTypeName:
		p.typeName = string(bytes.Trim(a.Data, "\x00"))
	case ipset.AttrFamily:
		p.family = netfilter.ProtoFamily(a.Data[0])
	default:
		// Go on.
	}
}

// parseAttrData parses attribute data and sets the timeout of the ipset.
func (p *props) parseAttrData(a netfilter.Attribute) {
	for _, a := range a.Children {
		switch ipset.AttributeType(a.Type) {
		case ipset.AttrTimeout:
			timeout := a.Uint32()
			p.isPersistent = timeout == 0
		default:
			// Go on.
		}
	}
}

// manager is the Linux Netfilter ipset manager.
type manager struct {
	nameToIpset    map[string]props
	domainToIpsets map[string][]props

	logger *slog.Logger

	dial dialer

	// mu protects all properties below.
	mu *sync.Mutex

	// TODO(a.garipov): Currently, the ipset list is static, and we don't read
	// the IPs already in sets, so we can assume that all incoming IPs are
	// either added to all corresponding ipsets or not.  When that stops being
	// the case, for example if we add dynamic reconfiguration of ipsets, this
	// map will need to become a per-ipset-name one.
	addedIPs *container.MapSet[ipInIpsetEntry]

	ipv4Conn ipsetConn
	ipv6Conn ipsetConn
}

// ipInIpsetEntry is the type for entries in [manager.addIPs].
type ipInIpsetEntry struct {
	ipsetName string
	// TODO(schzen):  Use netip.Addr.
	ipArr [net.IPv6len]byte
}

// dialNetfilter establishes connections to Linux's netfilter module.
func (m *manager) dialNetfilter(conf *netlink.Config) (err error) {
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

// parseIpsetConfigLine parses one ipset configuration line.
func parseIpsetConfigLine(confStr string) (hosts, ipsetNames []string, err error) {
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

// parseIpsetConfig parses the ipset configuration and stores ipsets.  It
// returns an error if the configuration can't be used.
func (m *manager) parseIpsetConfig(ctx context.Context, ipsetConf []string) (err error) {
	// The family doesn't seem to matter when we use a header query, so query
	// only the IPv4 one.
	//
	// TODO(a.garipov): Find out if this is a bug or a feature.
	all, err := m.ipv4Conn.listAll()
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return err
	}

	currentlyKnown := map[string]props{}
	for _, p := range all {
		currentlyKnown[p.name] = p
	}

	for i, confStr := range ipsetConf {
		var hosts, ipsetNames []string
		hosts, ipsetNames, err = parseIpsetConfigLine(confStr)
		if err != nil {
			return fmt.Errorf("config line at idx %d: %w", i, err)
		}

		var ipsets []props
		ipsets, err = m.ipsets(ctx, ipsetNames, currentlyKnown)
		if err != nil {
			return fmt.Errorf("getting ipsets from config line at idx %d: %w", i, err)
		}

		for _, host := range hosts {
			m.domainToIpsets[host] = append(m.domainToIpsets[host], ipsets...)
		}
	}

	return nil
}

// ipsetProps returns the properties of an ipset with the given name.
//
// Additional header data query.  See https://github.com/AdguardTeam/AdGuardHome/issues/6420.
//
// TODO(s.chzhen):  Use *props.
func (m *manager) ipsetProps(name string) (p props, err error) {
	// The family doesn't seem to matter when we use a header query, so
	// query only the IPv4 one.
	//
	// TODO(a.garipov): Find out if this is a bug or a feature.
	var res *ipset.HeaderPolicy
	res, err = m.ipv4Conn.Header(name)
	if err != nil {
		return props{}, err
	}

	if res == nil || res.Family == nil {
		return props{}, errors.Error("empty response or no family data")
	}

	family := netfilter.ProtoFamily(res.Family.Value)
	if family != netfilter.ProtoIPv4 && family != netfilter.ProtoIPv6 {
		return props{}, fmt.Errorf("unexpected ipset family %q", family)
	}

	typeName := res.TypeName.Get()

	return props{
		name:         name,
		typeName:     typeName,
		family:       family,
		isPersistent: false,
	}, nil
}

// ipsets returns ipset properties of currently known ipsets.  It also makes an
// additional ipset header data query if needed.
func (m *manager) ipsets(
	ctx context.Context,
	names []string,
	currentlyKnown map[string]props,
) (sets []props, err error) {
	for _, n := range names {
		p, ok := currentlyKnown[n]
		if !ok {
			return nil, fmt.Errorf("unknown ipset %q", n)
		}

		if p.family != netfilter.ProtoIPv4 && p.family != netfilter.ProtoIPv6 {
			m.logger.DebugContext(
				ctx,
				"got unexpected ipset family while getting set properties",
				"set_name", p.name,
				"set_type", p.typeName,
				"set_family", p.family,
			)

			p, err = m.ipsetProps(n)
			if err != nil {
				return nil, fmt.Errorf("%q %q making header query: %w", p.name, p.typeName, err)
			}
		}

		m.nameToIpset[n] = p
		sets = append(sets, p)
	}

	return sets, nil
}

// newManagerWithDialer returns a new Linux ipset manager using the provided
// dialer.
func newManagerWithDialer(ctx context.Context, conf *Config, dial dialer) (mgr Manager, err error) {
	defer func() { err = errors.Annotate(err, "ipset: %w") }()

	m := &manager{
		mu: &sync.Mutex{},

		nameToIpset:    make(map[string]props),
		domainToIpsets: make(map[string][]props),

		logger: conf.Logger,

		dial: dial,

		addedIPs: container.NewMapSet[ipInIpsetEntry](),
	}

	err = m.dialNetfilter(&netlink.Config{})
	if err != nil {
		if errors.Is(err, unix.EPROTONOSUPPORT) {
			// The implementation doesn't support this protocol version.  Just
			// issue a warning.
			m.logger.WarnContext(ctx, "dialing netfilter", slogutil.KeyError, err)

			return nil, nil
		}

		return nil, fmt.Errorf("dialing netfilter: %w", err)
	}

	err = m.parseIpsetConfig(ctx, conf.Lines)
	if err != nil {
		return nil, fmt.Errorf("getting ipsets: %w", err)
	}

	m.logger.DebugContext(ctx, "initialized")

	return m, nil
}

// lookupHost find the ipsets for the host, taking subdomain wildcards into
// account.
func (m *manager) lookupHost(host string) (sets []props) {
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
func (m *manager) addIPs(host string, set props, ips []net.IP) (n int, err error) {
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

		if m.addedIPs.Has(e) {
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
		return 0, fmt.Errorf("adding %q%s to %q %q: %w", host, ips, set.name, set.typeName, err)
	}

	// Only add these to the cache once we're sure that all of them were
	// actually sent to the ipset.
	for _, e := range newAddedEntries {
		s := m.nameToIpset[e.ipsetName]
		if s.isPersistent {
			m.addedIPs.Add(e)
		}
	}

	return n, nil
}

// addToSets adds the IP addresses to the corresponding ipset.
func (m *manager) addToSets(
	ctx context.Context,
	host string,
	ip4s []net.IP,
	ip6s []net.IP,
	sets []props,
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
			return n, fmt.Errorf("%q %q unexpected family %q", set.name, set.typeName, set.family)
		}

		m.logger.DebugContext(
			ctx,
			"added ips to set",
			"ips_num", nn,
			"set_name", set.name,
			"set_type", set.typeName,
		)

		n += nn
	}

	return n, nil
}

// Add implements the [Manager] interface for *manager.
func (m *manager) Add(ctx context.Context, host string, ip4s, ip6s []net.IP) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	sets := m.lookupHost(host)
	if len(sets) == 0 {
		return 0, nil
	}

	m.logger.DebugContext(ctx, "found sets", "set_num", len(sets))

	return m.addToSets(ctx, host, ip4s, ip6s, sets)
}

// Close implements the [Manager] interface for *manager.
func (m *manager) Close() (err error) {
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

	return errors.Annotate(errors.Join(errs...), "closing ipsets: %w")
}
