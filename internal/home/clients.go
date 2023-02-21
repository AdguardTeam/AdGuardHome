package home

import (
	"bytes"
	"encoding"
	"fmt"
	"net"
	"net/netip"
	"strings"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/dhcpd"
	"github.com/AdguardTeam/AdGuardHome/internal/dnsforward"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/AdGuardHome/internal/querylog"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/stringutil"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
)

const clientsUpdatePeriod = 10 * time.Minute

var webHandlersRegistered = false

// Client contains information about persistent clients.
type Client struct {
	// upstreamConfig is the custom upstream config for this client.  If
	// it's nil, it has not been initialized yet.  If it's non-nil and
	// empty, there are no valid upstreams.  If it's non-nil and non-empty,
	// these upstream must be used.
	upstreamConfig *proxy.UpstreamConfig

	Name string

	IDs             []string
	Tags            []string
	BlockedServices []string
	Upstreams       []string

	UseOwnSettings        bool
	FilteringEnabled      bool
	SafeSearchEnabled     bool
	SafeBrowsingEnabled   bool
	ParentalEnabled       bool
	UseOwnBlockedServices bool
}

// closeUpstreams closes the client-specific upstream config of c if any.
func (c *Client) closeUpstreams() (err error) {
	if c.upstreamConfig != nil {
		err = c.upstreamConfig.Close()
		if err != nil {
			return fmt.Errorf("closing upstreams of client %q: %w", c.Name, err)
		}
	}

	return nil
}

type clientSource uint

// Clients information sources.  The order determines the priority.
const (
	ClientSourceNone clientSource = iota
	ClientSourceWHOIS
	ClientSourceARP
	ClientSourceRDNS
	ClientSourceDHCP
	ClientSourceHostsFile
	ClientSourcePersistent
)

// type check
var _ fmt.Stringer = clientSource(0)

// String returns a human-readable name of cs.
func (cs clientSource) String() (s string) {
	switch cs {
	case ClientSourceWHOIS:
		return "WHOIS"
	case ClientSourceARP:
		return "ARP"
	case ClientSourceRDNS:
		return "rDNS"
	case ClientSourceDHCP:
		return "DHCP"
	case ClientSourceHostsFile:
		return "etc/hosts"
	default:
		return ""
	}
}

// type check
var _ encoding.TextMarshaler = clientSource(0)

// MarshalText implements encoding.TextMarshaler for the clientSource.
func (cs clientSource) MarshalText() (text []byte, err error) {
	return []byte(cs.String()), nil
}

// clientSourceConf is used to configure where the runtime clients will be
// obtained from.
type clientSourcesConf struct {
	WHOIS     bool `yaml:"whois"`
	ARP       bool `yaml:"arp"`
	RDNS      bool `yaml:"rdns"`
	DHCP      bool `yaml:"dhcp"`
	HostsFile bool `yaml:"hosts"`
}

// RuntimeClient information
type RuntimeClient struct {
	WHOISInfo *RuntimeClientWHOISInfo
	Host      string
	Source    clientSource
}

// RuntimeClientWHOISInfo is the filtered WHOIS data for a runtime client.
type RuntimeClientWHOISInfo struct {
	City    string `json:"city,omitempty"`
	Country string `json:"country,omitempty"`
	Orgname string `json:"orgname,omitempty"`
}

type clientsContainer struct {
	// TODO(a.garipov): Perhaps use a number of separate indices for
	// different types (string, netip.Addr, and so on).
	list    map[string]*Client // name -> client
	idIndex map[string]*Client // ID -> client

	// ipToRC is the IP address to *RuntimeClient map.
	ipToRC map[netip.Addr]*RuntimeClient

	lock sync.Mutex

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

	testing bool // if TRUE, this object is used for internal tests
}

// Init initializes clients container
// dhcpServer: optional
// Note: this function must be called only once
func (clients *clientsContainer) Init(
	objects []*clientObject,
	dhcpServer dhcpd.Interface,
	etcHosts *aghnet.HostsContainer,
	arpdb aghnet.ARPDB,
) {
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
	clients.addFromConfig(objects)

	if clients.testing {
		return
	}

	clients.updateFromDHCP(true)
	if clients.dhcpServer != nil {
		clients.dhcpServer.SetOnLeaseChanged(clients.onDHCPLeaseChanged)
	}

	if clients.etcHosts != nil {
		go clients.handleHostsUpdates()
	}
}

func (clients *clientsContainer) handleHostsUpdates() {
	for upd := range clients.etcHosts.Upd() {
		clients.addFromHostsFile(upd)
	}
}

// Start - start the module
func (clients *clientsContainer) Start() {
	if !clients.testing {
		if !webHandlersRegistered {
			webHandlersRegistered = true
			clients.registerWebHandlers()
		}
		go clients.periodicUpdate()
	}
}

// Reload reloads runtime clients.
func (clients *clientsContainer) Reload() {
	if clients.arpdb != nil {
		clients.addFromSystemARP()
	}
}

type clientObject struct {
	Name string `yaml:"name"`

	Tags            []string `yaml:"tags"`
	IDs             []string `yaml:"ids"`
	BlockedServices []string `yaml:"blocked_services"`
	Upstreams       []string `yaml:"upstreams"`

	UseGlobalSettings        bool `yaml:"use_global_settings"`
	FilteringEnabled         bool `yaml:"filtering_enabled"`
	ParentalEnabled          bool `yaml:"parental_enabled"`
	SafeSearchEnabled        bool `yaml:"safesearch_enabled"`
	SafeBrowsingEnabled      bool `yaml:"safebrowsing_enabled"`
	UseGlobalBlockedServices bool `yaml:"use_global_blocked_services"`
}

// addFromConfig initializes the clients container with objects from the
// configuration file.
func (clients *clientsContainer) addFromConfig(objects []*clientObject) {
	for _, o := range objects {
		cli := &Client{
			Name: o.Name,

			IDs:       o.IDs,
			Upstreams: o.Upstreams,

			UseOwnSettings:        !o.UseGlobalSettings,
			FilteringEnabled:      o.FilteringEnabled,
			ParentalEnabled:       o.ParentalEnabled,
			SafeSearchEnabled:     o.SafeSearchEnabled,
			SafeBrowsingEnabled:   o.SafeBrowsingEnabled,
			UseOwnBlockedServices: !o.UseGlobalBlockedServices,
		}

		for _, s := range o.BlockedServices {
			if filtering.BlockedSvcKnown(s) {
				cli.BlockedServices = append(cli.BlockedServices, s)
			} else {
				log.Info("clients: skipping unknown blocked service %q", s)
			}
		}

		for _, t := range o.Tags {
			if clients.allTags.Has(t) {
				cli.Tags = append(cli.Tags, t)
			} else {
				log.Info("clients: skipping unknown tag %q", t)
			}
		}

		slices.Sort(cli.Tags)

		_, err := clients.Add(cli)
		if err != nil {
			log.Error("clients: adding clients %s: %s", cli.Name, err)
		}
	}
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

			Tags:            stringutil.CloneSlice(cli.Tags),
			IDs:             stringutil.CloneSlice(cli.IDs),
			BlockedServices: stringutil.CloneSlice(cli.BlockedServices),
			Upstreams:       stringutil.CloneSlice(cli.Upstreams),

			UseGlobalSettings:        !cli.UseOwnSettings,
			FilteringEnabled:         cli.FilteringEnabled,
			ParentalEnabled:          cli.ParentalEnabled,
			SafeSearchEnabled:        cli.SafeSearchEnabled,
			SafeBrowsingEnabled:      cli.SafeBrowsingEnabled,
			UseGlobalBlockedServices: !cli.UseOwnBlockedServices,
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

func (clients *clientsContainer) periodicUpdate() {
	defer log.OnPanic("clients container")

	for {
		clients.Reload()
		time.Sleep(clientsUpdatePeriod)
	}
}

func (clients *clientsContainer) onDHCPLeaseChanged(flags int) {
	switch flags {
	case dhcpd.LeaseChangedAdded,
		dhcpd.LeaseChangedAddedStatic,
		dhcpd.LeaseChangedRemovedStatic:
		clients.updateFromDHCP(true)
	case dhcpd.LeaseChangedRemovedAll:
		clients.updateFromDHCP(false)
	}
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
	if !ok {
		return ClientSourceNone
	}

	return rc.Source
}

func toQueryLogWHOIS(wi *RuntimeClientWHOISInfo) (cw *querylog.ClientWHOIS) {
	if wi == nil {
		return &querylog.ClientWHOIS{}
	}

	return &querylog.ClientWHOIS{
		City:    wi.City,
		Country: wi.Country,
		Orgname: wi.Orgname,
	}
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
			c.WHOIS = &querylog.ClientWHOIS{}
		}
	}()

	client, ok := clients.Find(id)
	if ok {
		return &querylog.Client{
			Name: client.Name,
		}, false
	}

	var rc *RuntimeClient
	rc, ok = clients.findRuntimeClient(ip)
	if ok {
		return &querylog.Client{
			Name:  rc.Host,
			WHOIS: toQueryLogWHOIS(rc.WHOISInfo),
		}, false
	}

	return &querylog.Client{
		Name: "",
	}, true
}

func (clients *clientsContainer) Find(id string) (c *Client, ok bool) {
	clients.lock.Lock()
	defer clients.lock.Unlock()

	c, ok = clients.findLocked(id)
	if !ok {
		return nil, false
	}

	c.IDs = stringutil.CloneSlice(c.IDs)
	c.Tags = stringutil.CloneSlice(c.Tags)
	c.BlockedServices = stringutil.CloneSlice(c.BlockedServices)
	c.Upstreams = stringutil.CloneSlice(c.Upstreams)

	return c, true
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
		},
	)
	if err != nil {
		return nil, err
	}

	c.upstreamConfig = conf

	return conf, nil
}

// findLocked searches for a client by its ID.  For internal use only.
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
			var n netip.Prefix
			n, err = netip.ParsePrefix(id)
			if err != nil {
				continue
			}

			if n.Contains(ip) {
				return c, true
			}
		}
	}

	if clients.dhcpServer == nil {
		return nil, false
	}

	macFound := clients.dhcpServer.FindMACbyIP(ip.AsSlice())
	if macFound == nil {
		return nil, false
	}

	for _, c = range clients.list {
		for _, id := range c.IDs {
			var mac net.HardwareAddr
			mac, err = net.ParseMAC(id)
			if err != nil {
				continue
			}

			if bytes.Equal(mac, macFound) {
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
		// Normalize structured data.
		var (
			ip  netip.Addr
			n   netip.Prefix
			mac net.HardwareAddr
		)

		if ip, err = netip.ParseAddr(id); err == nil {
			c.IDs[i] = ip.String()
		} else if n, err = netip.ParsePrefix(id); err == nil {
			c.IDs[i] = n.String()
		} else if mac, err = net.ParseMAC(id); err == nil {
			c.IDs[i] = mac.String()
		} else if err = dnsforward.ValidateClientID(id); err == nil {
			c.IDs[i] = strings.ToLower(id)
		} else {
			return fmt.Errorf("invalid clientid at index %d: %q", i, id)
		}
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

	// update Name index
	clients.list[c.Name] = c

	// update ID index
	for _, id := range c.IDs {
		clients.idIndex[id] = c
	}

	log.Debug("clients: added %q: ID:%q [%d]", c.Name, c.IDs, len(clients.list))

	return true, nil
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

	// update Name index
	delete(clients.list, name)

	// update ID index
	for _, id := range c.IDs {
		delete(clients.idIndex, id)
	}

	return true
}

// equalStringSlices returns true if the slices are equal.
func equalStringSlices(a, b []string) (ok bool) {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

// Update updates a client by its name.
func (clients *clientsContainer) Update(name string, c *Client) (err error) {
	err = clients.check(c)
	if err != nil {
		return err
	}

	clients.lock.Lock()
	defer clients.lock.Unlock()

	prev, ok := clients.list[name]
	if !ok {
		return errors.Error("client not found")
	}

	// First, check the name index.
	if prev.Name != c.Name {
		_, ok = clients.list[c.Name]
		if ok {
			return errors.Error("client already exists")
		}
	}

	// Second, check the IP index.
	if !equalStringSlices(prev.IDs, c.IDs) {
		for _, id := range c.IDs {
			c2, ok2 := clients.idIndex[id]
			if ok2 && c2 != prev {
				return fmt.Errorf("another client uses the same id (%q): %q", id, c2.Name)
			}
		}

		//  Update ID index.
		for _, id := range prev.IDs {
			delete(clients.idIndex, id)
		}
		for _, id := range c.IDs {
			clients.idIndex[id] = prev
		}
	}

	// Update name index.
	if prev.Name != c.Name {
		delete(clients.list, prev.Name)
		clients.list[c.Name] = prev
	}

	// Update upstreams cache.
	err = c.closeUpstreams()
	if err != nil {
		return err
	}

	*prev = *c

	return nil
}

// setWHOISInfo sets the WHOIS information for a client.
func (clients *clientsContainer) setWHOISInfo(ip netip.Addr, wi *RuntimeClientWHOISInfo) {
	clients.lock.Lock()
	defer clients.lock.Unlock()

	_, ok := clients.findLocked(ip.String())
	if ok {
		log.Debug("clients: client for %s is already created, ignore whois info", ip)
		return
	}

	rc, ok := clients.ipToRC[ip]
	if ok {
		rc.WHOISInfo = wi
		log.Debug("clients: set whois info for runtime client %s: %+v", rc.Host, wi)

		return
	}

	// Create a RuntimeClient implicitly so that we don't do this check
	// again.
	rc = &RuntimeClient{
		Source: ClientSourceWHOIS,
	}

	rc.WHOISInfo = wi

	clients.ipToRC[ip] = rc

	log.Debug("clients: set whois info for runtime client with ip %s: %+v", ip, wi)
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
	if ok {
		if rc.Source > src {
			return false
		}

		rc.Host = host
		rc.Source = src
	} else {
		rc = &RuntimeClient{
			Host:      host,
			Source:    src,
			WHOISInfo: &RuntimeClientWHOISInfo{},
		}

		clients.ipToRC[ip] = rc
	}

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

// updateFromDHCP adds the clients that have a non-empty hostname from the DHCP
// server.
func (clients *clientsContainer) updateFromDHCP(add bool) {
	if clients.dhcpServer == nil || !config.Clients.Sources.DHCP {
		return
	}

	clients.lock.Lock()
	defer clients.lock.Unlock()

	clients.rmHostsBySrc(ClientSourceDHCP)

	if !add {
		return
	}

	leases := clients.dhcpServer.Leases(dhcpd.LeasesAll)
	n := 0
	for _, l := range leases {
		if l.Hostname == "" {
			continue
		}

		// TODO(a.garipov):  Remove once we switch to netip.Addr more fully.
		ipAddr, err := netutil.IPToAddrNoMapped(l.IP)
		if err != nil {
			log.Error("clients: bad client ip %v from dhcp: %s", l.IP, err)

			continue
		}

		ok := clients.addHostLocked(ipAddr, l.Hostname, ClientSourceDHCP)
		if ok {
			n++
		}
	}

	log.Debug("clients: added %d client aliases from dhcp", n)
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
