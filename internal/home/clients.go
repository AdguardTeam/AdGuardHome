package home

import (
	"bytes"
	"fmt"
	"net"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/dhcpd"
	"github.com/AdguardTeam/AdGuardHome/internal/dhcpsvc"
	"github.com/AdguardTeam/AdGuardHome/internal/dnsforward"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/AdGuardHome/internal/querylog"
	"github.com/AdguardTeam/AdGuardHome/internal/whois"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/stringutil"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

// DHCP is an interface for accesing DHCP lease data the [clientsContainer]
// needs.
type DHCP interface {
	// Leases returns all the DHCP leases.
	Leases() (leases []*dhcpsvc.Lease)

	// HostByIP returns the hostname of the DHCP client with the given IP
	// address.  The address will be netip.Addr{} if there is no such client,
	// due to an assumption that a DHCP client must always have an IP address.
	HostByIP(ip netip.Addr) (host string)

	// MACByIP returns the MAC address for the given IP address leased.  It
	// returns nil if there is no such client, due to an assumption that a DHCP
	// client must always have a MAC address.
	MACByIP(ip netip.Addr) (mac net.HardwareAddr)
}

// clientsContainer is the storage of all runtime and persistent clients.
type clientsContainer struct {
	// TODO(a.garipov): Perhaps use a number of separate indices for different
	// types (string, netip.Addr, and so on).
	list    map[string]*Client // name -> client
	idIndex map[string]*Client // ID -> client

	// ipToRC is the IP address to *RuntimeClient map.
	ipToRC map[netip.Addr]*RuntimeClient

	allTags *stringutil.Set

	// dhcpServer is used for looking up clients IP addresses by MAC addresses
	dhcpServer dhcpd.Interface

	// dnsServer is used for checking clients IP status access list status
	dnsServer *dnsforward.Server

	// etcHosts contains list of rewrite rules taken from the operating system's
	// hosts database.
	etcHosts *aghnet.HostsContainer

	// arpdb stores the neighbors retrieved from ARP.
	arpdb aghnet.ARPDB

	// lock protects all fields.
	//
	// TODO(a.garipov): Use a pointer and describe which fields are protected in
	// more detail.  Use sync.RWMutex.
	lock sync.Mutex

	// safeSearchCacheSize is the size of the safe search cache to use for
	// persistent clients.
	safeSearchCacheSize uint

	// safeSearchCacheTTL is the TTL of the safe search cache to use for
	// persistent clients.
	safeSearchCacheTTL time.Duration

	// testing is a flag that disables some features for internal tests.
	//
	// TODO(a.garipov): Awful.  Remove.
	testing bool
}

// Init initializes clients container
// dhcpServer: optional
// Note: this function must be called only once
func (clients *clientsContainer) Init(
	objects []*clientObject,
	dhcpServer dhcpd.Interface,
	etcHosts *aghnet.HostsContainer,
	arpdb aghnet.ARPDB,
	filteringConf *filtering.Config,
) (err error) {
	if clients.list != nil {
		log.Fatal("clients.list != nil")
	}

	clients.list = make(map[string]*Client)
	clients.idIndex = make(map[string]*Client)
	clients.ipToRC = map[netip.Addr]*RuntimeClient{}

	clients.allTags = stringutil.NewSet(clientTags...)

	clients.dhcpServer = dhcpServer
	clients.etcHosts = etcHosts
	clients.arpdb = arpdb
	err = clients.addFromConfig(objects, filteringConf)
	if err != nil {
		// Don't wrap the error, because it's informative enough as is.
		return err
	}

	clients.safeSearchCacheSize = filteringConf.SafeSearchCacheSize
	clients.safeSearchCacheTTL = time.Minute * time.Duration(filteringConf.CacheTime)

	if clients.testing {
		return nil
	}

	if clients.dhcpServer != nil {
		clients.dhcpServer.SetOnLeaseChanged(clients.onDHCPLeaseChanged)
		clients.onDHCPLeaseChanged(dhcpd.LeaseChangedAdded)
	}

	if clients.etcHosts != nil {
		go clients.handleHostsUpdates()
	}

	return nil
}

func (clients *clientsContainer) handleHostsUpdates() {
	for upd := range clients.etcHosts.Upd() {
		clients.addFromHostsFile(upd)
	}
}

// webHandlersRegistered prevents a [clientsContainer] from regisering its web
// handlers more than once.
//
// TODO(a.garipov): Refactor HTTP handler registration logic.
var webHandlersRegistered = false

// Start starts the clients container.
func (clients *clientsContainer) Start() {
	if clients.testing {
		return
	}

	if !webHandlersRegistered {
		webHandlersRegistered = true
		clients.registerWebHandlers()
	}

	go clients.periodicUpdate()
}

// reloadARP reloads runtime clients from ARP, if configured.
func (clients *clientsContainer) reloadARP() {
	if clients.arpdb != nil {
		clients.addFromSystemARP()
	}
}

// clientObject is the YAML representation of a persistent client.
type clientObject struct {
	SafeSearchConf filtering.SafeSearchConfig `yaml:"safe_search"`

	// BlockedServices is the configuration of blocked services of a client.
	BlockedServices *filtering.BlockedServices `yaml:"blocked_services"`

	Name string `yaml:"name"`

	IDs       []string `yaml:"ids"`
	Tags      []string `yaml:"tags"`
	Upstreams []string `yaml:"upstreams"`

	UseGlobalSettings        bool `yaml:"use_global_settings"`
	FilteringEnabled         bool `yaml:"filtering_enabled"`
	ParentalEnabled          bool `yaml:"parental_enabled"`
	SafeBrowsingEnabled      bool `yaml:"safebrowsing_enabled"`
	UseGlobalBlockedServices bool `yaml:"use_global_blocked_services"`

	IgnoreQueryLog   bool `yaml:"ignore_querylog"`
	IgnoreStatistics bool `yaml:"ignore_statistics"`
}

// addFromConfig initializes the clients container with objects from the
// configuration file.
func (clients *clientsContainer) addFromConfig(
	objects []*clientObject,
	filteringConf *filtering.Config,
) (err error) {
	for _, o := range objects {
		cli := &Client{
			Name: o.Name,

			IDs:       o.IDs,
			Upstreams: o.Upstreams,

			UseOwnSettings:        !o.UseGlobalSettings,
			FilteringEnabled:      o.FilteringEnabled,
			ParentalEnabled:       o.ParentalEnabled,
			safeSearchConf:        o.SafeSearchConf,
			SafeBrowsingEnabled:   o.SafeBrowsingEnabled,
			UseOwnBlockedServices: !o.UseGlobalBlockedServices,
			IgnoreQueryLog:        o.IgnoreQueryLog,
			IgnoreStatistics:      o.IgnoreStatistics,
		}

		if o.SafeSearchConf.Enabled {
			o.SafeSearchConf.CustomResolver = safeSearchResolver{}

			err = cli.setSafeSearch(
				o.SafeSearchConf,
				filteringConf.SafeSearchCacheSize,
				time.Minute*time.Duration(filteringConf.CacheTime),
			)
			if err != nil {
				log.Error("clients: init client safesearch %q: %s", cli.Name, err)

				continue
			}
		}

		err = o.BlockedServices.Validate()
		if err != nil {
			return fmt.Errorf("clients: init client blocked services %q: %w", cli.Name, err)
		}

		cli.BlockedServices = o.BlockedServices.Clone()

		for _, t := range o.Tags {
			if clients.allTags.Has(t) {
				cli.Tags = append(cli.Tags, t)
			} else {
				log.Info("clients: skipping unknown tag %q", t)
			}
		}

		slices.Sort(cli.Tags)

		_, err = clients.Add(cli)
		if err != nil {
			log.Error("clients: adding clients %s: %s", cli.Name, err)
		}
	}

	return nil
}

// forConfig returns all currently known persistent clients as objects for the
// configuration file.
func (clients *clientsContainer) forConfig() (objs []*clientObject) {
	clients.lock.Lock()
	defer clients.lock.Unlock()

	objs = make([]*clientObject, 0, len(clients.list))
	for _, cli := range clients.list {
		o := &clientObject{
			Name: cli.Name,

			BlockedServices: cli.BlockedServices.Clone(),

			IDs:       stringutil.CloneSlice(cli.IDs),
			Tags:      stringutil.CloneSlice(cli.Tags),
			Upstreams: stringutil.CloneSlice(cli.Upstreams),

			UseGlobalSettings:        !cli.UseOwnSettings,
			FilteringEnabled:         cli.FilteringEnabled,
			ParentalEnabled:          cli.ParentalEnabled,
			SafeSearchConf:           cli.safeSearchConf,
			SafeBrowsingEnabled:      cli.SafeBrowsingEnabled,
			UseGlobalBlockedServices: !cli.UseOwnBlockedServices,
			IgnoreQueryLog:           cli.IgnoreQueryLog,
			IgnoreStatistics:         cli.IgnoreStatistics,
		}

		objs = append(objs, o)
	}

	// Maps aren't guaranteed to iterate in the same order each time, so the
	// above loop can generate different orderings when writing to the config
	// file: this produces lots of diffs in config files, so sort objects by
	// name before writing.
	slices.SortStableFunc(objs, func(a, b *clientObject) (sortsBefore bool) {
		return a.Name < b.Name
	})

	return objs
}

// arpClientsUpdatePeriod defines how often ARP clients are updated.
const arpClientsUpdatePeriod = 10 * time.Minute

func (clients *clientsContainer) periodicUpdate() {
	defer log.OnPanic("clients container")

	for {
		clients.reloadARP()
		time.Sleep(arpClientsUpdatePeriod)
	}
}

// onDHCPLeaseChanged is a callback for the DHCP server.  It updates the list of
// runtime clients using the DHCP server's leases.
//
// TODO(e.burkov):  Remove when switched to dhcpsvc.
func (clients *clientsContainer) onDHCPLeaseChanged(flags int) {
	if clients.dhcpServer == nil || !config.Clients.Sources.DHCP {
		return
	}

	clients.lock.Lock()
	defer clients.lock.Unlock()

	clients.rmHostsBySrc(ClientSourceDHCP)

	if flags == dhcpd.LeaseChangedRemovedAll {
		return
	}

	leases := clients.dhcpServer.Leases(dhcpd.LeasesAll)
	n := 0
	for _, l := range leases {
		if l.Hostname == "" {
			continue
		}

		ok := clients.addHostLocked(l.IP, l.Hostname, ClientSourceDHCP)
		if ok {
			n++
		}
	}

	log.Debug("clients: added %d client aliases from dhcp", n)
}

// clientSource checks if client with this IP address already exists and returns
// the source which updated it last.  It returns [ClientSourceNone] if the
// client doesn't exist.
func (clients *clientsContainer) clientSource(ip netip.Addr) (src clientSource) {
	clients.lock.Lock()
	defer clients.lock.Unlock()

	_, ok := clients.findLocked(ip.String())
	if ok {
		return ClientSourcePersistent
	}

	rc, ok := clients.ipToRC[ip]
	if ok {
		return rc.Source
	}

	return ClientSourceNone
}

// findMultiple is a wrapper around Find to make it a valid client finder for
// the query log.  c is never nil; if no information about the client is found,
// it returns an artificial client record by only setting the blocking-related
// fields.  err is always nil.
func (clients *clientsContainer) findMultiple(ids []string) (c *querylog.Client, err error) {
	var artClient *querylog.Client
	var art bool
	for _, id := range ids {
		ip, _ := netip.ParseAddr(id)
		c, art = clients.clientOrArtificial(ip, id)
		if art {
			artClient = c

			continue
		}

		return c, nil
	}

	return artClient, nil
}

// clientOrArtificial returns information about one client.  If art is true,
// this is an artificial client record, meaning that we currently don't have any
// records about this client besides maybe whether or not it is blocked.  c is
// never nil.
func (clients *clientsContainer) clientOrArtificial(
	ip netip.Addr,
	id string,
) (c *querylog.Client, art bool) {
	defer func() {
		c.Disallowed, c.DisallowedRule = clients.dnsServer.IsBlockedClient(ip, id)
		if c.WHOIS == nil {
			c.WHOIS = &whois.Info{}
		}
	}()

	client, ok := clients.Find(id)
	if ok {
		return &querylog.Client{
			Name:           client.Name,
			IgnoreQueryLog: client.IgnoreQueryLog,
		}, false
	}

	var rc *RuntimeClient
	rc, ok = clients.findRuntimeClient(ip)
	if ok {
		return &querylog.Client{
			Name:  rc.Host,
			WHOIS: rc.WHOIS,
		}, false
	}

	return &querylog.Client{
		Name: "",
	}, true
}

// Find returns a shallow copy of the client if there is one found.
func (clients *clientsContainer) Find(id string) (c *Client, ok bool) {
	clients.lock.Lock()
	defer clients.lock.Unlock()

	c, ok = clients.findLocked(id)
	if !ok {
		return nil, false
	}

	return c.ShallowClone(), true
}

// shouldCountClient is a wrapper around Find to make it a valid client
// information finder for the statistics.  If no information about the client
// is found, it returns true.
func (clients *clientsContainer) shouldCountClient(ids []string) (y bool) {
	clients.lock.Lock()
	defer clients.lock.Unlock()

	for _, id := range ids {
		client, ok := clients.findLocked(id)
		if ok {
			return !client.IgnoreStatistics
		}
	}

	return true
}

// findUpstreams returns upstreams configured for the client, identified either
// by its IP address or its ClientID.  upsConf is nil if the client isn't found
// or if the client has no custom upstreams.
func (clients *clientsContainer) findUpstreams(
	id string,
) (upsConf *proxy.UpstreamConfig, err error) {
	clients.lock.Lock()
	defer clients.lock.Unlock()

	c, ok := clients.findLocked(id)
	if !ok {
		return nil, nil
	}

	upstreams := stringutil.FilterOut(c.Upstreams, dnsforward.IsCommentOrEmpty)
	if len(upstreams) == 0 {
		return nil, nil
	}

	if c.upstreamConfig != nil {
		return c.upstreamConfig, nil
	}

	var conf *proxy.UpstreamConfig
	conf, err = proxy.ParseUpstreamsConfig(
		upstreams,
		&upstream.Options{
			Bootstrap:    config.DNS.BootstrapDNS,
			Timeout:      config.DNS.UpstreamTimeout.Duration,
			HTTPVersions: dnsforward.UpstreamHTTPVersions(config.DNS.UseHTTP3Upstreams),
			PreferIPv6:   config.DNS.BootstrapPreferIPv6,
		},
	)
	if err != nil {
		return nil, err
	}

	c.upstreamConfig = conf

	return conf, nil
}

// findLocked searches for a client by its ID.  clients.lock is expected to be
// locked.
func (clients *clientsContainer) findLocked(id string) (c *Client, ok bool) {
	c, ok = clients.idIndex[id]
	if ok {
		return c, true
	}

	ip, err := netip.ParseAddr(id)
	if err != nil {
		return nil, false
	}

	for _, c = range clients.list {
		for _, id := range c.IDs {
			var subnet netip.Prefix
			subnet, err = netip.ParsePrefix(id)
			if err != nil {
				continue
			}

			if subnet.Contains(ip) {
				return c, true
			}
		}
	}

	if clients.dhcpServer != nil {
		return clients.findDHCP(ip)
	}

	return nil, false
}

// findDHCP searches for a client by its MAC, if the DHCP server is active and
// there is such client.  clients.lock is expected to be locked.
func (clients *clientsContainer) findDHCP(ip netip.Addr) (c *Client, ok bool) {
	foundMAC := clients.dhcpServer.FindMACbyIP(ip)
	if foundMAC == nil {
		return nil, false
	}

	for _, c = range clients.list {
		for _, id := range c.IDs {
			mac, err := net.ParseMAC(id)
			if err != nil {
				continue
			}

			if bytes.Equal(mac, foundMAC) {
				return c, true
			}
		}
	}

	return nil, false
}

// findRuntimeClient finds a runtime client by their IP.
func (clients *clientsContainer) findRuntimeClient(ip netip.Addr) (rc *RuntimeClient, ok bool) {
	if ip == (netip.Addr{}) {
		return nil, false
	}

	clients.lock.Lock()
	defer clients.lock.Unlock()

	rc, ok = clients.ipToRC[ip]

	return rc, ok
}

// check validates the client.
func (clients *clientsContainer) check(c *Client) (err error) {
	switch {
	case c == nil:
		return errors.Error("client is nil")
	case c.Name == "":
		return errors.Error("invalid name")
	case len(c.IDs) == 0:
		return errors.Error("id required")
	default:
		// Go on.
	}

	for i, id := range c.IDs {
		var norm string
		norm, err = normalizeClientIdentifier(id)
		if err != nil {
			return fmt.Errorf("client at index %d: %w", i, err)
		}

		c.IDs[i] = norm
	}

	for _, t := range c.Tags {
		if !clients.allTags.Has(t) {
			return fmt.Errorf("invalid tag: %q", t)
		}
	}

	slices.Sort(c.Tags)

	err = dnsforward.ValidateUpstreams(c.Upstreams)
	if err != nil {
		return fmt.Errorf("invalid upstream servers: %w", err)
	}

	return nil
}

// normalizeClientIdentifier returns a normalized version of idStr.  If idStr
// cannot be normalized, it returns an error.
func normalizeClientIdentifier(idStr string) (norm string, err error) {
	if idStr == "" {
		return "", errors.Error("clientid is empty")
	}

	var ip netip.Addr
	if ip, err = netip.ParseAddr(idStr); err == nil {
		return ip.String(), nil
	}

	var subnet netip.Prefix
	if subnet, err = netip.ParsePrefix(idStr); err == nil {
		return subnet.String(), nil
	}

	var mac net.HardwareAddr
	if mac, err = net.ParseMAC(idStr); err == nil {
		return mac.String(), nil
	}

	if err = dnsforward.ValidateClientID(idStr); err == nil {
		return strings.ToLower(idStr), nil
	}

	return "", fmt.Errorf("bad client identifier %q", idStr)
}

// Add adds a new client object.  ok is false if such client already exists or
// if an error occurred.
func (clients *clientsContainer) Add(c *Client) (ok bool, err error) {
	err = clients.check(c)
	if err != nil {
		return false, err
	}

	clients.lock.Lock()
	defer clients.lock.Unlock()

	// check Name index
	_, ok = clients.list[c.Name]
	if ok {
		return false, nil
	}

	// check ID index
	for _, id := range c.IDs {
		var c2 *Client
		c2, ok = clients.idIndex[id]
		if ok {
			return false, fmt.Errorf("another client uses the same ID (%q): %q", id, c2.Name)
		}
	}

	clients.add(c)

	log.Debug("clients: added %q: ID:%q [%d]", c.Name, c.IDs, len(clients.list))

	return true, nil
}

// add c to the indexes. clients.lock is expected to be locked.
func (clients *clientsContainer) add(c *Client) {
	// update Name index
	clients.list[c.Name] = c

	// update ID index
	for _, id := range c.IDs {
		clients.idIndex[id] = c
	}
}

// Del removes a client.  ok is false if there is no such client.
func (clients *clientsContainer) Del(name string) (ok bool) {
	clients.lock.Lock()
	defer clients.lock.Unlock()

	var c *Client
	c, ok = clients.list[name]
	if !ok {
		return false
	}

	if err := c.closeUpstreams(); err != nil {
		log.Error("client container: removing client %s: %s", name, err)
	}

	clients.del(c)

	return true
}

// del removes c from the indexes. clients.lock is expected to be locked.
func (clients *clientsContainer) del(c *Client) {
	// update Name index
	delete(clients.list, c.Name)

	// update ID index
	for _, id := range c.IDs {
		delete(clients.idIndex, id)
	}
}

// Update updates a client by its name.
func (clients *clientsContainer) Update(prev, c *Client) (err error) {
	err = clients.check(c)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return err
	}

	clients.lock.Lock()
	defer clients.lock.Unlock()

	// Check the name index.
	if prev.Name != c.Name {
		_, ok := clients.list[c.Name]
		if ok {
			return errors.Error("client already exists")
		}
	}

	// Check the ID index.
	if !slices.Equal(prev.IDs, c.IDs) {
		for _, id := range c.IDs {
			existing, ok := clients.idIndex[id]
			if ok && existing != prev {
				return fmt.Errorf("id %q is used by client with name %q", id, existing.Name)
			}
		}
	}

	clients.del(prev)
	clients.add(c)

	return nil
}

// setWHOISInfo sets the WHOIS information for a client.
func (clients *clientsContainer) setWHOISInfo(ip netip.Addr, wi *whois.Info) {
	clients.lock.Lock()
	defer clients.lock.Unlock()

	_, ok := clients.findLocked(ip.String())
	if ok {
		log.Debug("clients: client for %s is already created, ignore whois info", ip)

		return
	}

	// TODO(e.burkov):  Consider storing WHOIS information separately and
	// potentially get rid of [RuntimeClient].
	rc, ok := clients.ipToRC[ip]
	if !ok {
		// Create a RuntimeClient implicitly so that we don't do this check
		// again.
		rc = &RuntimeClient{
			Source: ClientSourceWHOIS,
		}
		clients.ipToRC[ip] = rc

		log.Debug("clients: set whois info for runtime client with ip %s: %+v", ip, wi)
	} else {
		log.Debug("clients: set whois info for runtime client %s: %+v", rc.Host, wi)
	}

	rc.WHOIS = wi
}

// AddHost adds a new IP-hostname pairing.  The priorities of the sources are
// taken into account.  ok is true if the pairing was added.
func (clients *clientsContainer) AddHost(
	ip netip.Addr,
	host string,
	src clientSource,
) (ok bool) {
	clients.lock.Lock()
	defer clients.lock.Unlock()

	return clients.addHostLocked(ip, host, src)
}

// addHostLocked adds a new IP-hostname pairing.  clients.lock is expected to be
// locked.
func (clients *clientsContainer) addHostLocked(
	ip netip.Addr,
	host string,
	src clientSource,
) (ok bool) {
	rc, ok := clients.ipToRC[ip]
	if !ok {
		rc = &RuntimeClient{
			WHOIS: &whois.Info{},
		}

		clients.ipToRC[ip] = rc
	} else if src < rc.Source {
		return false
	}

	rc.Host = host
	rc.Source = src

	log.Debug("clients: added %s -> %q [%d]", ip, host, len(clients.ipToRC))

	return true
}

// rmHostsBySrc removes all entries that match the specified source.
func (clients *clientsContainer) rmHostsBySrc(src clientSource) {
	n := 0
	for ip, rc := range clients.ipToRC {
		if rc.Source == src {
			delete(clients.ipToRC, ip)
			n++
		}
	}

	log.Debug("clients: removed %d client aliases", n)
}

// addFromHostsFile fills the client-hostname pairing index from the system's
// hosts files.
func (clients *clientsContainer) addFromHostsFile(hosts aghnet.HostsRecords) {
	clients.lock.Lock()
	defer clients.lock.Unlock()

	clients.rmHostsBySrc(ClientSourceHostsFile)

	n := 0
	for ip, rec := range hosts {
		clients.addHostLocked(ip, rec.Canonical, ClientSourceHostsFile)
		n++
	}

	log.Debug("clients: added %d client aliases from system hosts file", n)
}

// addFromSystemARP adds the IP-hostname pairings from the output of the arp -a
// command.
func (clients *clientsContainer) addFromSystemARP() {
	if err := clients.arpdb.Refresh(); err != nil {
		log.Error("refreshing arp container: %s", err)

		clients.arpdb = aghnet.EmptyARPDB{}

		return
	}

	ns := clients.arpdb.Neighbors()
	if len(ns) == 0 {
		log.Debug("refreshing arp container: the update is empty")

		return
	}

	clients.lock.Lock()
	defer clients.lock.Unlock()

	clients.rmHostsBySrc(ClientSourceARP)

	added := 0
	for _, n := range ns {
		if clients.addHostLocked(n.IP, n.Name, ClientSourceARP) {
			added++
		}
	}

	log.Debug("clients: added %d client aliases from arp neighborhood", added)
}

// close gracefully closes all the client-specific upstream configurations of
// the persistent clients.
func (clients *clientsContainer) close() (err error) {
	persistent := maps.Values(clients.list)
	slices.SortFunc(persistent, func(a, b *Client) (less bool) { return a.Name < b.Name })

	var errs []error

	for _, cli := range persistent {
		if err = cli.closeUpstreams(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return errors.List("closing client specific upstreams", errs...)
	}

	return nil
}
