package home

import (
	"bytes"
	"fmt"
	"net"
	"os/exec"
	"runtime"
	"sort"
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

type clientSource uint

// Client sources.  The order determines the priority.
const (
	ClientSourceWHOIS clientSource = iota
	ClientSourceRDNS
	ClientSourceARP
	ClientSourceDHCP
	ClientSourceHostsFile
)

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
	// different types (string, net.IP, and so on).
	list    map[string]*Client // name -> client
	idIndex map[string]*Client // ID -> client

	// ipToRC is the IP address to *RuntimeClient map.
	ipToRC *netutil.IPMap

	lock sync.Mutex

	allTags *stringutil.Set

	// dhcpServer is used for looking up clients IP addresses by MAC addresses
	dhcpServer *dhcpd.Server

	// dnsServer is used for checking clients IP status access list status
	dnsServer *dnsforward.Server

	// etcHosts contains list of rewrite rules taken from the operating system's
	// hosts database.
	etcHosts *aghnet.HostsContainer

	testing bool // if TRUE, this object is used for internal tests
}

// Init initializes clients container
// dhcpServer: optional
// Note: this function must be called only once
func (clients *clientsContainer) Init(
	objects []*clientObject,
	dhcpServer *dhcpd.Server,
	etcHosts *aghnet.HostsContainer,
) {
	if clients.list != nil {
		log.Fatal("clients.list != nil")
	}
	clients.list = make(map[string]*Client)
	clients.idIndex = make(map[string]*Client)
	clients.ipToRC = netutil.NewIPMap(0)

	clients.allTags = stringutil.NewSet(clientTags...)

	clients.dhcpServer = dhcpServer
	clients.etcHosts = etcHosts
	clients.addFromConfig(objects)

	if clients.testing {
		return
	}

	clients.updateFromDHCP(true)
	if clients.dhcpServer != nil {
		clients.dhcpServer.SetOnLeaseChanged(clients.onDHCPLeaseChanged)
	}

	go clients.handleHostsUpdates()
}

func (clients *clientsContainer) handleHostsUpdates() {
	if clients.etcHosts != nil {
		for upd := range clients.etcHosts.Upd() {
			clients.addFromHostsFile(upd)
		}
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
	clients.addFromSystemARP()
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

		sort.Strings(cli.Tags)

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
	sort.Slice(objs, func(i, j int) bool { return objs[i].Name < objs[j].Name })

	return objs
}

func (clients *clientsContainer) periodicUpdate() {
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

// Exists checks if client with this IP address already exists.
func (clients *clientsContainer) Exists(ip net.IP, source clientSource) (ok bool) {
	clients.lock.Lock()
	defer clients.lock.Unlock()

	_, ok = clients.findLocked(ip.String())
	if ok {
		return true
	}

	rc, ok := clients.findRuntimeClientLocked(ip)
	if !ok {
		return false
	}

	// Return false if the new source has higher priority.
	return source <= rc.Source
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
		c, art = clients.clientOrArtificial(net.ParseIP(id), id)
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
	ip net.IP,
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

	if ip == nil {
		// Technically should never happen, but still.
		return &querylog.Client{
			Name: "",
		}, true
	}

	var rc *RuntimeClient
	rc, ok = clients.FindRuntimeClient(ip)
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
			Bootstrap: config.DNS.BootstrapDNS,
			Timeout:   config.DNS.UpstreamTimeout.Duration,
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

	ip := net.ParseIP(id)
	if ip == nil {
		return nil, false
	}

	for _, c = range clients.list {
		for _, id := range c.IDs {
			_, ipnet, err := net.ParseCIDR(id)
			if err != nil {
				continue
			}

			if ipnet.Contains(ip) {
				return c, true
			}
		}
	}

	if clients.dhcpServer == nil {
		return nil, false
	}

	macFound := clients.dhcpServer.FindMACbyIP(ip)
	if macFound == nil {
		return nil, false
	}

	for _, c = range clients.list {
		for _, id := range c.IDs {
			hwAddr, err := net.ParseMAC(id)
			if err != nil {
				continue
			}

			if bytes.Equal(hwAddr, macFound) {
				return c, true
			}
		}
	}

	return nil, false
}

// findRuntimeClientLocked finds a runtime client by their IP address.  For
// internal use only.
func (clients *clientsContainer) findRuntimeClientLocked(ip net.IP) (rc *RuntimeClient, ok bool) {
	var v interface{}
	v, ok = clients.ipToRC.Get(ip)
	if !ok {
		return nil, false
	}

	rc, ok = v.(*RuntimeClient)
	if !ok {
		log.Error("clients: bad type %T in ipToRC for %s", v, ip)

		return nil, false
	}

	return rc, true
}

// FindRuntimeClient finds a runtime client by their IP.
func (clients *clientsContainer) FindRuntimeClient(ip net.IP) (rc *RuntimeClient, ok bool) {
	if ip == nil {
		return nil, false
	}

	clients.lock.Lock()
	defer clients.lock.Unlock()

	return clients.findRuntimeClientLocked(ip)
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
		var ip net.IP
		var ipnet *net.IPNet
		var mac net.HardwareAddr
		if ip = net.ParseIP(id); ip != nil {
			c.IDs[i] = ip.String()
		} else if ip, ipnet, err = net.ParseCIDR(id); err == nil {
			ipnet.IP = ip
			c.IDs[i] = ipnet.String()
		} else if mac, err = net.ParseMAC(id); err == nil {
			c.IDs[i] = mac.String()
		} else if err = dnsforward.ValidateClientID(id); err == nil {
			c.IDs[i] = id
		} else {
			return fmt.Errorf("invalid client id at index %d: %q", i, id)
		}
	}

	for _, t := range c.Tags {
		if !clients.allTags.Has(t) {
			return fmt.Errorf("invalid tag: %q", t)
		}
	}

	sort.Strings(c.Tags)

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

		// update ID index
		for _, id := range prev.IDs {
			delete(clients.idIndex, id)
		}
		for _, id := range c.IDs {
			clients.idIndex[id] = prev
		}
	}

	// update Name index
	if prev.Name != c.Name {
		delete(clients.list, prev.Name)
		clients.list[c.Name] = prev
	}

	// update upstreams cache
	c.upstreamConfig = nil

	*prev = *c

	return nil
}

// SetWHOISInfo sets the WHOIS information for a client.
func (clients *clientsContainer) SetWHOISInfo(ip net.IP, wi *RuntimeClientWHOISInfo) {
	clients.lock.Lock()
	defer clients.lock.Unlock()

	_, ok := clients.findLocked(ip.String())
	if ok {
		log.Debug("clients: client for %s is already created, ignore whois info", ip)
		return
	}

	rc, ok := clients.findRuntimeClientLocked(ip)
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

	clients.ipToRC.Set(ip, rc)

	log.Debug("clients: set whois info for runtime client with ip %s: %+v", ip, wi)
}

// AddHost adds a new IP-hostname pairing.  The priorities of the sources are
// taken into account.  ok is true if the pairing was added.
func (clients *clientsContainer) AddHost(ip net.IP, host string, src clientSource) (ok bool, err error) {
	clients.lock.Lock()
	defer clients.lock.Unlock()

	ok = clients.addHostLocked(ip, host, src)

	return ok, nil
}

// addHostLocked adds a new IP-hostname pairing.  For internal use only.
func (clients *clientsContainer) addHostLocked(ip net.IP, host string, src clientSource) (ok bool) {
	var rc *RuntimeClient
	rc, ok = clients.findRuntimeClientLocked(ip)
	if ok {
		if rc.Source > src {
			return false
		}

		rc.Source = src
	} else {
		rc = &RuntimeClient{
			Host:      host,
			Source:    src,
			WHOISInfo: &RuntimeClientWHOISInfo{},
		}

		clients.ipToRC.Set(ip, rc)
	}

	log.Debug("clients: added %s -> %q [%d]", ip, host, clients.ipToRC.Len())

	return true
}

// rmHostsBySrc removes all entries that match the specified source.
func (clients *clientsContainer) rmHostsBySrc(src clientSource) {
	n := 0
	clients.ipToRC.Range(func(ip net.IP, v interface{}) (cont bool) {
		rc, ok := v.(*RuntimeClient)
		if !ok {
			log.Error("clients: bad type %T in ipToRC for %s", v, ip)

			return true
		}

		if rc.Source == src {
			clients.ipToRC.Del(ip)
			n++
		}

		return true
	})

	log.Debug("clients: removed %d client aliases", n)
}

// addFromHostsFile fills the client-hostname pairing index from the system's
// hosts files.
func (clients *clientsContainer) addFromHostsFile(hosts *netutil.IPMap) {
	clients.lock.Lock()
	defer clients.lock.Unlock()

	clients.rmHostsBySrc(ClientSourceHostsFile)

	n := 0
	hosts.Range(func(ip net.IP, v interface{}) (cont bool) {
		hosts, ok := v.(*aghnet.Hosts)
		if !ok {
			log.Error("dns: bad type %T in ipToRC for %s", v, ip)

			return true
		}

		if clients.addHostLocked(ip, hosts.Main, ClientSourceHostsFile) {
			n++
		}
		hosts.Aliases.Range(func(name string) (cont bool) {
			if clients.addHostLocked(ip, name, ClientSourceHostsFile) {
				n++
			}

			return true
		})

		return true
	})

	log.Debug("clients: added %d client aliases from system hosts-file", n)
}

// addFromSystemARP adds the IP-hostname pairings from the output of the arp -a
// command.
func (clients *clientsContainer) addFromSystemARP() {
	if runtime.GOOS == "windows" {
		return
	}

	cmd := exec.Command("arp", "-a")
	log.Tracef("executing %q %q", cmd.Path, cmd.Args)
	data, err := cmd.Output()
	if err != nil || cmd.ProcessState.ExitCode() != 0 {
		log.Debug("command %q has failed: %q code:%d",
			cmd.Path, err, cmd.ProcessState.ExitCode())
		return
	}

	clients.lock.Lock()
	defer clients.lock.Unlock()

	clients.rmHostsBySrc(ClientSourceARP)

	n := 0
	// TODO(a.garipov): Rewrite to use bufio.Scanner.
	lines := strings.Split(string(data), "\n")
	for _, ln := range lines {
		lparen := strings.Index(ln, " (")
		rparen := strings.Index(ln, ") ")
		if lparen == -1 || rparen == -1 || lparen >= rparen {
			continue
		}

		host := ln[:lparen]
		ipStr := ln[lparen+2 : rparen]
		ip := net.ParseIP(ipStr)
		if netutil.ValidateDomainName(host) != nil || ip == nil {
			continue
		}

		ok := clients.addHostLocked(ip, host, ClientSourceARP)
		if ok {
			n++
		}
	}

	log.Debug("clients: added %d client aliases from 'arp -a' command output", n)
}

// updateFromDHCP adds the clients that have a non-empty hostname from the DHCP
// server.
func (clients *clientsContainer) updateFromDHCP(add bool) {
	if clients.dhcpServer == nil {
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

		ok := clients.addHostLocked(l.IP, l.Hostname, ClientSourceDHCP)
		if ok {
			n++
		}
	}

	log.Debug("clients: added %d client aliases from dhcp", n)
}
