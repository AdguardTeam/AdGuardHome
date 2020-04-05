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

	"github.com/AdguardTeam/AdGuardHome/dhcpd"
	"github.com/AdguardTeam/AdGuardHome/dnsfilter"
	"github.com/AdguardTeam/AdGuardHome/dnsforward"
	"github.com/AdguardTeam/AdGuardHome/util"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/utils"
)

const (
	clientsUpdatePeriod = 1 * time.Hour
)

var webHandlersRegistered = false

// Client information
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
	// Upstream objects:
	// nil: not yet initialized
	// not nil, but empty: initialized, no good upstreams
	// not nil, not empty: Upstreams ready to be used
	upstreamObjects []upstream.Upstream
}

type clientSource uint

// Client sources
const (
	// Priority: etc/hosts > DHCP > ARP > rDNS > WHOIS
	ClientSourceWHOIS     clientSource = iota // from WHOIS
	ClientSourceRDNS                          // from rDNS
	ClientSourceDHCP                          // from DHCP
	ClientSourceARP                           // from 'arp -a'
	ClientSourceHostsFile                     // from /etc/hosts
)

// ClientHost information
type ClientHost struct {
	Host      string
	Source    clientSource
	WhoisInfo [][]string // [[key,value], ...]
}

type clientsContainer struct {
	list    map[string]*Client     // name -> client
	idIndex map[string]*Client     // IP -> client
	ipHost  map[string]*ClientHost // IP -> Hostname
	lock    sync.Mutex

	allTags map[string]bool

	// dhcpServer is used for looking up clients IP addresses by MAC addresses
	dhcpServer *dhcpd.Server

	autoHosts *util.AutoHosts // get entries from system hosts-files

	testing bool // if TRUE, this object is used for internal tests
}

// Init initializes clients container
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
		clients.dhcpServer.SetOnLeaseChanged(clients.onDHCPLeaseChanged)
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
		cli := Client{
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
				log.Debug("Clients: skipping unknown blocked-service '%s'", s)
				continue
			}
			cli.BlockedServices = append(cli.BlockedServices, s)
		}

		for _, t := range cy.Tags {
			if !clients.tagKnown(t) {
				log.Debug("Clients: skipping unknown tag '%s'", t)
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

		cy.Tags = stringArrayDup(cli.Tags)
		cy.IDs = stringArrayDup(cli.IDs)
		cy.BlockedServices = stringArrayDup(cli.BlockedServices)
		cy.Upstreams = stringArrayDup(cli.Upstreams)

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

// Exists checks if client with this IP already exists
func (clients *clientsContainer) Exists(ip string, source clientSource) bool {
	clients.lock.Lock()
	defer clients.lock.Unlock()

	_, ok := clients.findByIP(ip)
	if ok {
		return true
	}

	ch, ok := clients.ipHost[ip]
	if !ok {
		return false
	}
	if source > ch.Source {
		return false // we're going to overwrite this client's info with a stronger source
	}
	return true
}

func stringArrayDup(a []string) []string {
	a2 := make([]string, len(a))
	copy(a2, a)
	return a2
}

// Find searches for a client by IP
func (clients *clientsContainer) Find(ip string) (Client, bool) {
	clients.lock.Lock()
	defer clients.lock.Unlock()

	c, ok := clients.findByIP(ip)
	if !ok {
		return Client{}, false
	}
	c.IDs = stringArrayDup(c.IDs)
	c.Tags = stringArrayDup(c.Tags)
	c.BlockedServices = stringArrayDup(c.BlockedServices)
	c.Upstreams = stringArrayDup(c.Upstreams)
	return c, true
}

func upstreamArrayCopy(a []upstream.Upstream) []upstream.Upstream {
	a2 := make([]upstream.Upstream, len(a))
	copy(a2, a)
	return a2
}

// FindUpstreams looks for upstreams configured for the client
// If no client found for this IP, or if no custom upstreams are configured,
// this method returns nil
func (clients *clientsContainer) FindUpstreams(ip string) []upstream.Upstream {
	clients.lock.Lock()
	defer clients.lock.Unlock()

	c, ok := clients.findByIP(ip)
	if !ok {
		return nil
	}

	if c.upstreamObjects == nil {
		c.upstreamObjects = make([]upstream.Upstream, 0)
		for _, us := range c.Upstreams {
			u, err := upstream.AddressToUpstream(us, upstream.Options{Timeout: dnsforward.DefaultTimeout})
			if err != nil {
				log.Error("upstream.AddressToUpstream: %s: %s", us, err)
				continue
			}
			c.upstreamObjects = append(c.upstreamObjects, u)
		}
	}

	if len(c.upstreamObjects) == 0 {
		return nil
	}
	return upstreamArrayCopy(c.upstreamObjects)
}

// Find searches for a client by IP (and does not lock anything)
func (clients *clientsContainer) findByIP(ip string) (Client, bool) {
	ipAddr := net.ParseIP(ip)
	if ipAddr == nil {
		return Client{}, false
	}

	c, ok := clients.idIndex[ip]
	if ok {
		return *c, true
	}

	for _, c = range clients.list {
		for _, id := range c.IDs {
			_, ipnet, err := net.ParseCIDR(id)
			if err != nil {
				continue
			}
			if ipnet.Contains(ipAddr) {
				return *c, true
			}
		}
	}

	if clients.dhcpServer == nil {
		return Client{}, false
	}
	macFound := clients.dhcpServer.FindMACbyIP(ipAddr)
	if macFound == nil {
		return Client{}, false
	}
	for _, c = range clients.list {
		for _, id := range c.IDs {
			hwAddr, err := net.ParseMAC(id)
			if err != nil {
				continue
			}
			if bytes.Equal(hwAddr, macFound) {
				return *c, true
			}
		}
	}

	return Client{}, false
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

// Check if Client object's fields are correct
func (clients *clientsContainer) check(c *Client) error {
	if len(c.Name) == 0 {
		return fmt.Errorf("invalid Name")
	}

	if len(c.IDs) == 0 {
		return fmt.Errorf("ID required")
	}

	for i, id := range c.IDs {
		ip := net.ParseIP(id)
		if ip != nil {
			c.IDs[i] = ip.String() // normalize IP address
			continue
		}

		_, _, err := net.ParseCIDR(id)
		if err == nil {
			continue
		}

		_, err = net.ParseMAC(id)
		if err == nil {
			continue
		}

		return fmt.Errorf("invalid ID: %s", id)
	}

	for _, t := range c.Tags {
		if !clients.tagKnown(t) {
			return fmt.Errorf("invalid tag: %s", t)
		}
	}
	sort.Strings(c.Tags)

	if len(c.Upstreams) != 0 {
		err := dnsforward.ValidateUpstreams(c.Upstreams)
		if err != nil {
			return fmt.Errorf("invalid upstream servers: %s", err)
		}
	}

	return nil
}

// Add a new client object
// Return true: success;  false: client exists.
func (clients *clientsContainer) Add(c Client) (bool, error) {
	e := clients.check(&c)
	if e != nil {
		return false, e
	}

	clients.lock.Lock()
	defer clients.lock.Unlock()

	// check Name index
	_, ok := clients.list[c.Name]
	if ok {
		return false, nil
	}

	// check ID index
	for _, id := range c.IDs {
		c2, ok := clients.idIndex[id]
		if ok {
			return false, fmt.Errorf("another client uses the same ID (%s): %s", id, c2.Name)
		}
	}

	// update Name index
	clients.list[c.Name] = &c

	// update ID index
	for _, id := range c.IDs {
		clients.idIndex[id] = &c
	}

	log.Debug("Clients: added '%s': ID:%v [%d]", c.Name, c.IDs, len(clients.list))
	return true, nil
}

// Del removes a client
func (clients *clientsContainer) Del(name string) bool {
	clients.lock.Lock()
	defer clients.lock.Unlock()

	c, ok := clients.list[name]
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

// Return TRUE if arrays are equal
func arraysEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i != len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Update a client
func (clients *clientsContainer) Update(name string, c Client) error {
	err := clients.check(&c)
	if err != nil {
		return err
	}

	clients.lock.Lock()
	defer clients.lock.Unlock()

	old, ok := clients.list[name]
	if !ok {
		return fmt.Errorf("client not found")
	}

	// check Name index
	if old.Name != c.Name {
		_, ok = clients.list[c.Name]
		if ok {
			return fmt.Errorf("client already exists")
		}
	}

	// check IP index
	if !arraysEqual(old.IDs, c.IDs) {
		for _, id := range c.IDs {
			c2, ok := clients.idIndex[id]
			if ok && c2 != old {
				return fmt.Errorf("another client uses the same ID (%s): %s", id, c2.Name)
			}
		}

		// update ID index
		for _, id := range old.IDs {
			delete(clients.idIndex, id)
		}
		for _, id := range c.IDs {
			clients.idIndex[id] = old
		}
	}

	// update Name index
	if old.Name != c.Name {
		delete(clients.list, old.Name)
		clients.list[c.Name] = old
	}

	// update upstreams cache
	c.upstreamObjects = nil

	*old = c
	return nil
}

// SetWhoisInfo - associate WHOIS information with a client
func (clients *clientsContainer) SetWhoisInfo(ip string, info [][]string) {
	clients.lock.Lock()
	defer clients.lock.Unlock()

	_, ok := clients.findByIP(ip)
	if ok {
		log.Debug("Clients: client for %s is already created, ignore WHOIS info", ip)
		return
	}

	ch, ok := clients.ipHost[ip]
	if ok {
		ch.WhoisInfo = info
		log.Debug("Clients: set WHOIS info for auto-client %s: %v", ch.Host, ch.WhoisInfo)
		return
	}

	// Create a ClientHost implicitly so that we don't do this check again
	ch = &ClientHost{
		Source: ClientSourceWHOIS,
	}
	ch.WhoisInfo = info
	clients.ipHost[ip] = ch
	log.Debug("Clients: set WHOIS info for auto-client with IP %s: %v", ip, ch.WhoisInfo)
}

// AddHost adds new IP -> Host pair
// Use priority of the source (etc/hosts > ARP > rDNS)
//  so we overwrite existing entries with an equal or higher priority
func (clients *clientsContainer) AddHost(ip, host string, source clientSource) (bool, error) {
	clients.lock.Lock()
	b, e := clients.addHost(ip, host, source)
	clients.lock.Unlock()
	return b, e
}

func (clients *clientsContainer) addHost(ip, host string, source clientSource) (bool, error) {
	// check auto-clients index
	ch, ok := clients.ipHost[ip]
	if ok && ch.Source > source {
		return false, nil
	} else if ok {
		ch.Source = source
	} else {
		ch = &ClientHost{
			Host:   host,
			Source: source,
		}
		clients.ipHost[ip] = ch
	}
	log.Debug("Clients: added '%s' -> '%s' [%d]", ip, host, len(clients.ipHost))
	return true, nil
}

// Remove all entries that match the specified source
func (clients *clientsContainer) rmHosts(source clientSource) int {
	n := 0
	for k, v := range clients.ipHost {
		if v.Source == source {
			delete(clients.ipHost, k)
			n++
		}
	}
	log.Debug("Clients: removed %d client aliases", n)
	return n
}

// Fill clients array from system hosts-file
func (clients *clientsContainer) addFromHostsFile() {
	hosts := clients.autoHosts.List()

	clients.lock.Lock()
	defer clients.lock.Unlock()
	_ = clients.rmHosts(ClientSourceHostsFile)

	n := 0
	for ip, names := range hosts {
		for _, name := range names {
			ok, err := clients.addHost(ip, name.String(), ClientSourceHostsFile)
			if err != nil {
				log.Debug("Clients: %s", err)
			}
			if ok {
				n++
			}
		}
	}

	log.Debug("Clients: added %d client aliases from system hosts-file", n)
}

// Add IP -> Host pairs from the system's `arp -a` command output
// The command's output is:
// HOST (IP) at MAC on IFACE
func (clients *clientsContainer) addFromSystemARP() {
	if runtime.GOOS == "windows" {
		return
	}

	cmd := exec.Command("arp", "-a")
	log.Tracef("executing %s %v", cmd.Path, cmd.Args)
	data, err := cmd.Output()
	if err != nil || cmd.ProcessState.ExitCode() != 0 {
		log.Debug("command %s has failed: %v code:%d",
			cmd.Path, err, cmd.ProcessState.ExitCode())
		return
	}

	clients.lock.Lock()
	defer clients.lock.Unlock()
	_ = clients.rmHosts(ClientSourceARP)

	n := 0
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

		ok, e := clients.addHost(ip, host, ClientSourceARP)
		if e != nil {
			log.Tracef("%s", e)
		}
		if ok {
			n++
		}
	}

	log.Debug("Clients: added %d client aliases from 'arp -a' command output", n)
}

// Add clients from DHCP that have non-empty Hostname property
func (clients *clientsContainer) addFromDHCP() {
	if clients.dhcpServer == nil {
		return
	}

	clients.lock.Lock()
	defer clients.lock.Unlock()

	_ = clients.rmHosts(ClientSourceDHCP)

	leases := clients.dhcpServer.Leases(dhcpd.LeasesAll)
	n := 0
	for _, l := range leases {
		if len(l.Hostname) == 0 {
			continue
		}
		ok, _ := clients.addHost(l.IP.String(), l.Hostname, ClientSourceDHCP)
		if ok {
			n++
		}
	}
	log.Debug("Clients: added %d client aliases from DHCP", n)
}
