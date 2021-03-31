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

	"github.com/AdguardTeam/AdGuardHome/internal/agherr"
	"github.com/AdguardTeam/AdGuardHome/internal/dhcpd"
	"github.com/AdguardTeam/AdGuardHome/internal/dnsfilter"
	"github.com/AdguardTeam/AdGuardHome/internal/dnsforward"
	"github.com/AdguardTeam/AdGuardHome/internal/util"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/utils"
)

const clientsUpdatePeriod = 10 * time.Minute

var webHandlersRegistered = false

// Client contains information about persistent clients.
type Client struct {
	IDs                 []string
	Tags                []string
	Name                string
	UseOwnSettings      bool // false: use global settings
	FilteringEnabled    bool
	SafeSearchEnabled   bool
	SafeBrowsingEnabled bool
	ParentalEnabled     bool

	UseOwnBlockedServices bool // false: use global settings
	BlockedServices       []string

	Upstreams []string // list of upstream servers to be used for the client's requests

	// Custom upstream config for this client
	// nil: not yet initialized
	// not nil, but empty: initialized, no good upstreams
	// not nil, not empty: Upstreams ready to be used
	upstreamConfig *proxy.UpstreamConfig
}

type clientSource uint

// Client sources.  The order determines the priority.
const (
	ClientSourceWHOIS clientSource = iota
	ClientSourceRDNS
	ClientSourceDHCP
	ClientSourceARP
	ClientSourceHostsFile
)

// ClientHost information
type ClientHost struct {
	Host      string
	Source    clientSource
	WhoisInfo [][]string // [[key,value], ...]
}

type clientsContainer struct {
	// TODO(a.garipov): Perhaps use a number of separate indices for
	// different types (string, net.IP, and so on).
	list    map[string]*Client     // name -> client
	idIndex map[string]*Client     // ID -> client
	ipHost  map[string]*ClientHost // IP -> Hostname
	lock    sync.Mutex

	allTags map[string]bool

	// dhcpServer is used for looking up clients IP addresses by MAC addresses
	dhcpServer *dhcpd.Server

	// dnsServer is used for checking clients IP status access list status
	dnsServer *dnsforward.Server

	autoHosts *util.AutoHosts // get entries from system hosts-files

	testing bool // if TRUE, this object is used for internal tests
}

// Init initializes clients container
// dhcpServer: optional
// Note: this function must be called only once
func (clients *clientsContainer) Init(objects []clientObject, dhcpServer *dhcpd.Server, autoHosts *util.AutoHosts) {
	if clients.list != nil {
		log.Fatal("clients.list != nil")
	}
	clients.list = make(map[string]*Client)
	clients.idIndex = make(map[string]*Client)
	clients.ipHost = make(map[string]*ClientHost)

	clients.allTags = make(map[string]bool)
	for _, t := range clientTags {
		clients.allTags[t] = false
	}

	clients.dhcpServer = dhcpServer
	clients.autoHosts = autoHosts
	clients.addFromConfig(objects)

	if !clients.testing {
		clients.addFromDHCP()
		if clients.dhcpServer != nil {
			clients.dhcpServer.SetOnLeaseChanged(clients.onDHCPLeaseChanged)
		}
		clients.autoHosts.SetOnChanged(clients.onHostsChanged)
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

// Reload - reload auto-clients
func (clients *clientsContainer) Reload() {
	clients.addFromSystemARP()
}

type clientObject struct {
	Name                string   `yaml:"name"`
	Tags                []string `yaml:"tags"`
	IDs                 []string `yaml:"ids"`
	UseGlobalSettings   bool     `yaml:"use_global_settings"`
	FilteringEnabled    bool     `yaml:"filtering_enabled"`
	ParentalEnabled     bool     `yaml:"parental_enabled"`
	SafeSearchEnabled   bool     `yaml:"safesearch_enabled"`
	SafeBrowsingEnabled bool     `yaml:"safebrowsing_enabled"`

	UseGlobalBlockedServices bool     `yaml:"use_global_blocked_services"`
	BlockedServices          []string `yaml:"blocked_services"`

	Upstreams []string `yaml:"upstreams"`
}

func (clients *clientsContainer) tagKnown(tag string) bool {
	_, ok := clients.allTags[tag]
	return ok
}

func (clients *clientsContainer) addFromConfig(objects []clientObject) {
	for _, cy := range objects {
		cli := &Client{
			Name:                cy.Name,
			IDs:                 cy.IDs,
			UseOwnSettings:      !cy.UseGlobalSettings,
			FilteringEnabled:    cy.FilteringEnabled,
			ParentalEnabled:     cy.ParentalEnabled,
			SafeSearchEnabled:   cy.SafeSearchEnabled,
			SafeBrowsingEnabled: cy.SafeBrowsingEnabled,

			UseOwnBlockedServices: !cy.UseGlobalBlockedServices,

			Upstreams: cy.Upstreams,
		}

		for _, s := range cy.BlockedServices {
			if !dnsfilter.BlockedSvcKnown(s) {
				log.Debug("clients: skipping unknown blocked-service %q", s)
				continue
			}
			cli.BlockedServices = append(cli.BlockedServices, s)
		}

		for _, t := range cy.Tags {
			if !clients.tagKnown(t) {
				log.Debug("clients: skipping unknown tag %q", t)
				continue
			}
			cli.Tags = append(cli.Tags, t)
		}
		sort.Strings(cli.Tags)

		_, err := clients.Add(cli)
		if err != nil {
			log.Tracef("clientAdd: %s", err)
		}
	}
}

// WriteDiskConfig - write configuration
func (clients *clientsContainer) WriteDiskConfig(objects *[]clientObject) {
	clients.lock.Lock()
	for _, cli := range clients.list {
		cy := clientObject{
			Name:                     cli.Name,
			UseGlobalSettings:        !cli.UseOwnSettings,
			FilteringEnabled:         cli.FilteringEnabled,
			ParentalEnabled:          cli.ParentalEnabled,
			SafeSearchEnabled:        cli.SafeSearchEnabled,
			SafeBrowsingEnabled:      cli.SafeBrowsingEnabled,
			UseGlobalBlockedServices: !cli.UseOwnBlockedServices,
		}

		cy.Tags = copyStrings(cli.Tags)
		cy.IDs = copyStrings(cli.IDs)
		cy.BlockedServices = copyStrings(cli.BlockedServices)
		cy.Upstreams = copyStrings(cli.Upstreams)

		*objects = append(*objects, cy)
	}
	clients.lock.Unlock()
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
		clients.addFromDHCP()
	}
}

func (clients *clientsContainer) onHostsChanged() {
	clients.addFromHostsFile()
}

// Exists checks if client with this ID already exists.
func (clients *clientsContainer) Exists(id string, source clientSource) (ok bool) {
	clients.lock.Lock()
	defer clients.lock.Unlock()

	_, ok = clients.findLocked(id)
	if ok {
		return true
	}

	var ch *ClientHost
	ch, ok = clients.ipHost[id]
	if !ok {
		return false
	}

	// Return false if the new source has higher priority.
	return source <= ch.Source
}

func copyStrings(a []string) (b []string) {
	return append(b, a...)
}

// Find searches for a client by its ID.
func (clients *clientsContainer) Find(id string) (c *Client, ok bool) {
	clients.lock.Lock()
	defer clients.lock.Unlock()

	c, ok = clients.findLocked(id)
	if !ok {
		return nil, false
	}

	c.IDs = copyStrings(c.IDs)
	c.Tags = copyStrings(c.Tags)
	c.BlockedServices = copyStrings(c.BlockedServices)
	c.Upstreams = copyStrings(c.Upstreams)
	return c, true
}

// FindUpstreams looks for upstreams configured for the client
// If no client found for this IP, or if no custom upstreams are configured,
// this method returns nil
func (clients *clientsContainer) FindUpstreams(ip string) *proxy.UpstreamConfig {
	clients.lock.Lock()
	defer clients.lock.Unlock()

	c, ok := clients.findLocked(ip)
	if !ok {
		return nil
	}

	if len(c.Upstreams) == 0 {
		return nil
	}

	if c.upstreamConfig == nil {
		conf, err := proxy.ParseUpstreamsConfig(
			c.Upstreams,
			upstream.Options{
				Bootstrap: config.DNS.BootstrapDNS,
				Timeout:   dnsforward.DefaultTimeout,
			},
		)
		if err == nil {
			c.upstreamConfig = &conf
		}
	}

	return c.upstreamConfig
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

// FindAutoClient - search for an auto-client by IP
func (clients *clientsContainer) FindAutoClient(ip string) (ClientHost, bool) {
	ipAddr := net.ParseIP(ip)
	if ipAddr == nil {
		return ClientHost{}, false
	}

	clients.lock.Lock()
	defer clients.lock.Unlock()

	ch, ok := clients.ipHost[ip]
	if ok {
		return *ch, true
	}
	return ClientHost{}, false
}

// check validates the client.
func (clients *clientsContainer) check(c *Client) (err error) {
	switch {
	case c == nil:
		return agherr.Error("client is nil")
	case c.Name == "":
		return agherr.Error("invalid name")
	case len(c.IDs) == 0:
		return agherr.Error("id required")
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
		if !clients.tagKnown(t) {
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
		return agherr.Error("client not found")
	}

	// First, check the name index.
	if prev.Name != c.Name {
		_, ok = clients.list[c.Name]
		if ok {
			return agherr.Error("client already exists")
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

// SetWhoisInfo sets the WHOIS information for a client.
//
// TODO(a.garipov): Perhaps replace [][]string with map[string]string.
func (clients *clientsContainer) SetWhoisInfo(ip string, info [][]string) {
	clients.lock.Lock()
	defer clients.lock.Unlock()

	_, ok := clients.findLocked(ip)
	if ok {
		log.Debug("clients: client for %s is already created, ignore whois info", ip)
		return
	}

	ch, ok := clients.ipHost[ip]
	if ok {
		ch.WhoisInfo = info
		log.Debug("clients: set whois info for auto-client %s: %q", ch.Host, info)

		return
	}

	// Create a ClientHost implicitly so that we don't do this check again
	ch = &ClientHost{
		Source: ClientSourceWHOIS,
	}
	ch.WhoisInfo = info
	clients.ipHost[ip] = ch
	log.Debug("clients: set whois info for auto-client with IP %s: %q", ip, info)
}

// AddHost adds a new IP-hostname pairing.  The priorities of the sources is
// taken into account.  ok is true if the pairing was added.
func (clients *clientsContainer) AddHost(ip, host string, src clientSource) (ok bool, err error) {
	clients.lock.Lock()
	defer clients.lock.Unlock()

	ok = clients.addHostLocked(ip, host, src)

	return ok, nil
}

// addHostLocked adds a new IP-hostname pairing.  For internal use only.
func (clients *clientsContainer) addHostLocked(ip, host string, src clientSource) (ok bool) {
	var ch *ClientHost
	ch, ok = clients.ipHost[ip]
	if ok {
		if ch.Source > src {
			return false
		}

		ch.Source = src
	} else {
		ch = &ClientHost{
			Host:   host,
			Source: src,
		}

		clients.ipHost[ip] = ch
	}

	log.Debug("clients: added %q -> %q [%d]", ip, host, len(clients.ipHost))

	return true
}

// rmHostsBySrc removes all entries that match the specified source.
func (clients *clientsContainer) rmHostsBySrc(src clientSource) {
	n := 0
	for k, v := range clients.ipHost {
		if v.Source == src {
			delete(clients.ipHost, k)
			n++
		}
	}

	log.Debug("clients: removed %d client aliases", n)
}

// addFromHostsFile fills the client-hostname pairing index from the system's
// hosts files.
func (clients *clientsContainer) addFromHostsFile() {
	hosts := clients.autoHosts.List()

	clients.lock.Lock()
	defer clients.lock.Unlock()

	clients.rmHostsBySrc(ClientSourceHostsFile)

	n := 0
	for ip, names := range hosts {
		for _, name := range names {
			ok := clients.addHostLocked(ip, name, ClientSourceHostsFile)
			if ok {
				n++
			}
		}
	}

	log.Debug("Clients: added %d client aliases from system hosts-file", n)
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
		open := strings.Index(ln, " (")
		close := strings.Index(ln, ") ")
		if open == -1 || close == -1 || open >= close {
			continue
		}

		host := ln[:open]
		ip := ln[open+2 : close]
		if utils.IsValidHostname(host) != nil || net.ParseIP(ip) == nil {
			continue
		}

		ok := clients.addHostLocked(ip, host, ClientSourceARP)
		if ok {
			n++
		}
	}

	log.Debug("clients: added %d client aliases from 'arp -a' command output", n)
}

// addFromDHCP adds the clients that have a non-empty hostname from the DHCP
// server.
func (clients *clientsContainer) addFromDHCP() {
	if clients.dhcpServer == nil {
		return
	}

	clients.lock.Lock()
	defer clients.lock.Unlock()

	clients.rmHostsBySrc(ClientSourceDHCP)

	leases := clients.dhcpServer.Leases(dhcpd.LeasesAll)
	n := 0
	for _, l := range leases {
		if l.Hostname == "" {
			continue
		}

		ok := clients.addHostLocked(l.IP.String(), l.Hostname, ClientSourceDHCP)
		if ok {
			n++
		}
	}

	log.Debug("clients: added %d client aliases from dhcp", n)
}
