package home

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/utils"
)

const (
	clientsUpdatePeriod = 1 * time.Hour
)

// Client information
type Client struct {
	IP                  string
	MAC                 string
	Name                string
	UseOwnSettings      bool // false: use global settings
	FilteringEnabled    bool
	SafeSearchEnabled   bool
	SafeBrowsingEnabled bool
	ParentalEnabled     bool
	WhoisInfo           [][]string // [[key,value], ...]

	UseOwnBlockedServices bool // false: use global settings
	BlockedServices       []string
}

type clientJSON struct {
	IP                  string `json:"ip"`
	MAC                 string `json:"mac"`
	Name                string `json:"name"`
	UseGlobalSettings   bool   `json:"use_global_settings"`
	FilteringEnabled    bool   `json:"filtering_enabled"`
	ParentalEnabled     bool   `json:"parental_enabled"`
	SafeSearchEnabled   bool   `json:"safebrowsing_enabled"`
	SafeBrowsingEnabled bool   `json:"safesearch_enabled"`

	WhoisInfo map[string]interface{} `json:"whois_info"`

	UseGlobalBlockedServices bool     `json:"use_global_blocked_services"`
	BlockedServices          []string `json:"blocked_services"`
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
	list    map[string]*Client    // name -> client
	ipIndex map[string]*Client    // IP -> client
	ipHost  map[string]ClientHost // IP -> Hostname
	lock    sync.Mutex
}

// Init initializes clients container
// Note: this function must be called only once
func (clients *clientsContainer) Init() {
	if clients.list != nil {
		log.Fatal("clients.list != nil")
	}
	clients.list = make(map[string]*Client)
	clients.ipIndex = make(map[string]*Client)
	clients.ipHost = make(map[string]ClientHost)

	go clients.periodicUpdate()
}

func (clients *clientsContainer) periodicUpdate() {
	for {
		clients.addFromHostsFile()
		clients.addFromSystemARP()
		clients.addFromDHCP()
		time.Sleep(clientsUpdatePeriod)
	}
}

// GetList returns the pointer to clients list
func (clients *clientsContainer) GetList() map[string]*Client {
	return clients.list
}

// Exists checks if client with this IP already exists
func (clients *clientsContainer) Exists(ip string, source clientSource) bool {
	clients.lock.Lock()
	defer clients.lock.Unlock()

	_, ok := clients.ipIndex[ip]
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

// Find searches for a client by IP
func (clients *clientsContainer) Find(ip string) (Client, bool) {
	clients.lock.Lock()
	defer clients.lock.Unlock()

	c, ok := clients.ipIndex[ip]
	if ok {
		return *c, true
	}

	for _, c = range clients.list {
		if len(c.MAC) != 0 {
			mac, err := net.ParseMAC(c.MAC)
			if err != nil {
				continue
			}
			ipAddr := config.dhcpServer.FindIPbyMAC(mac)
			if ipAddr == nil {
				continue
			}
			if ip == ipAddr.String() {
				return *c, true
			}
		}
	}

	return Client{}, false
}

// Check if Client object's fields are correct
func (c *Client) check() error {
	if len(c.Name) == 0 {
		return fmt.Errorf("Invalid Name")
	}

	if (len(c.IP) == 0 && len(c.MAC) == 0) ||
		(len(c.IP) != 0 && len(c.MAC) != 0) {
		return fmt.Errorf("IP or MAC required")
	}

	if len(c.IP) != 0 {
		ip := net.ParseIP(c.IP)
		if ip == nil {
			return fmt.Errorf("Invalid IP")
		}
		c.IP = ip.String()
	} else {
		_, err := net.ParseMAC(c.MAC)
		if err != nil {
			return fmt.Errorf("Invalid MAC: %s", err)
		}
	}
	return nil
}

// Add a new client object
// Return true: success;  false: client exists.
func (clients *clientsContainer) Add(c Client) (bool, error) {
	e := c.check()
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

	// check IP index
	if len(c.IP) != 0 {
		c2, ok := clients.ipIndex[c.IP]
		if ok {
			return false, fmt.Errorf("Another client uses the same IP address: %s", c2.Name)
		}
	}

	ch, ok := clients.ipHost[c.IP]
	if ok {
		c.WhoisInfo = ch.WhoisInfo
		delete(clients.ipHost, c.IP)
	}

	clients.list[c.Name] = &c
	if len(c.IP) != 0 {
		clients.ipIndex[c.IP] = &c
	}

	log.Tracef("'%s': '%s' | '%s' -> [%d]", c.Name, c.IP, c.MAC, len(clients.list))
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

	delete(clients.list, name)
	delete(clients.ipIndex, c.IP)
	return true
}

// Update a client
func (clients *clientsContainer) Update(name string, c Client) error {
	err := c.check()
	if err != nil {
		return err
	}

	clients.lock.Lock()
	defer clients.lock.Unlock()

	old, ok := clients.list[name]
	if !ok {
		return fmt.Errorf("Client not found")
	}

	// check Name index
	if old.Name != c.Name {
		_, ok = clients.list[c.Name]
		if ok {
			return fmt.Errorf("Client already exists")
		}
	}

	// check IP index
	if old.IP != c.IP && len(c.IP) != 0 {
		c2, ok := clients.ipIndex[c.IP]
		if ok {
			return fmt.Errorf("Another client uses the same IP address: %s", c2.Name)
		}
	}

	// update Name index
	if old.Name != c.Name {
		delete(clients.list, old.Name)
	}
	clients.list[c.Name] = &c

	// update IP index
	if old.IP != c.IP {
		delete(clients.ipIndex, old.IP)
	}
	if len(c.IP) != 0 {
		clients.ipIndex[c.IP] = &c
	}

	return nil
}

// SetWhoisInfo - associate WHOIS information with a client
func (clients *clientsContainer) SetWhoisInfo(ip string, info [][]string) {
	clients.lock.Lock()
	defer clients.lock.Unlock()

	c, ok := clients.ipIndex[ip]
	if ok {
		c.WhoisInfo = info
		log.Debug("Clients: set WHOIS info for client %s: %v", c.Name, c.WhoisInfo)
		return
	}

	ch, ok := clients.ipHost[ip]
	if ok {
		ch.WhoisInfo = info
		log.Debug("Clients: set WHOIS info for auto-client %s: %v", ch.Host, ch.WhoisInfo)
		return
	}

	ch = ClientHost{
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
	defer clients.lock.Unlock()

	// check index
	_, ok := clients.ipIndex[ip]
	if ok {
		return false, nil
	}

	// check index
	c, ok := clients.ipHost[ip]
	if ok && c.Source > source {
		return false, nil
	}

	clients.ipHost[ip] = ClientHost{
		Host:      host,
		Source:    source,
		WhoisInfo: c.WhoisInfo,
	}
	log.Tracef("'%s' -> '%s' [%d]", ip, host, len(clients.ipHost))
	return true, nil
}

// Parse system 'hosts' file and fill clients array
func (clients *clientsContainer) addFromHostsFile() {
	hostsFn := "/etc/hosts"
	if runtime.GOOS == "windows" {
		hostsFn = os.ExpandEnv("$SystemRoot\\system32\\drivers\\etc\\hosts")
	}

	d, e := ioutil.ReadFile(hostsFn)
	if e != nil {
		log.Info("Can't read file %s: %v", hostsFn, e)
		return
	}

	lines := strings.Split(string(d), "\n")
	n := 0
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)
		if len(ln) == 0 || ln[0] == '#' {
			continue
		}

		fields := strings.Fields(ln)
		if len(fields) < 2 {
			continue
		}

		ok, e := clients.AddHost(fields[0], fields[1], ClientSourceHostsFile)
		if e != nil {
			log.Tracef("%s", e)
		}
		if ok {
			n++
		}
	}

	log.Info("Added %d client aliases from %s", n, hostsFn)
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

		ok, e := clients.AddHost(ip, host, ClientSourceARP)
		if e != nil {
			log.Tracef("%s", e)
		}
		if ok {
			n++
		}
	}

	log.Info("Added %d client aliases from 'arp -a' command output", n)
}

// add clients from DHCP that have non-empty Hostname property
func (clients *clientsContainer) addFromDHCP() {
	leases := config.dhcpServer.Leases()
	for _, l := range leases {
		if len(l.Hostname) == 0 {
			continue
		}
		_, _ = config.clients.AddHost(l.IP.String(), l.Hostname, ClientSourceDHCP)
	}
}

type clientHostJSON struct {
	IP     string `json:"ip"`
	Name   string `json:"name"`
	Source string `json:"source"`

	WhoisInfo map[string]interface{} `json:"whois_info"`
}

type clientListJSON struct {
	Clients     []clientJSON     `json:"clients"`
	AutoClients []clientHostJSON `json:"auto_clients"`
}

// respond with information about configured clients
func handleGetClients(w http.ResponseWriter, r *http.Request) {
	data := clientListJSON{}

	config.clients.lock.Lock()
	for _, c := range config.clients.list {
		cj := clientJSON{
			IP:                  c.IP,
			MAC:                 c.MAC,
			Name:                c.Name,
			UseGlobalSettings:   !c.UseOwnSettings,
			FilteringEnabled:    c.FilteringEnabled,
			ParentalEnabled:     c.ParentalEnabled,
			SafeSearchEnabled:   c.SafeSearchEnabled,
			SafeBrowsingEnabled: c.SafeBrowsingEnabled,

			UseGlobalBlockedServices: !c.UseOwnBlockedServices,
			BlockedServices:          c.BlockedServices,
		}

		if len(c.MAC) != 0 {
			hwAddr, _ := net.ParseMAC(c.MAC)
			ipAddr := config.dhcpServer.FindIPbyMAC(hwAddr)
			if ipAddr != nil {
				cj.IP = ipAddr.String()
			}
		}

		cj.WhoisInfo = make(map[string]interface{})
		for _, wi := range c.WhoisInfo {
			cj.WhoisInfo[wi[0]] = wi[1]
		}

		data.Clients = append(data.Clients, cj)
	}
	for ip, ch := range config.clients.ipHost {
		cj := clientHostJSON{
			IP:   ip,
			Name: ch.Host,
		}

		cj.Source = "etc/hosts"
		switch ch.Source {
		case ClientSourceDHCP:
			cj.Source = "DHCP"
		case ClientSourceRDNS:
			cj.Source = "rDNS"
		case ClientSourceARP:
			cj.Source = "ARP"
		case ClientSourceWHOIS:
			cj.Source = "WHOIS"
		}

		cj.WhoisInfo = make(map[string]interface{})
		for _, wi := range ch.WhoisInfo {
			cj.WhoisInfo[wi[0]] = wi[1]
		}

		data.AutoClients = append(data.AutoClients, cj)
	}
	config.clients.lock.Unlock()

	w.Header().Set("Content-Type", "application/json")
	e := json.NewEncoder(w).Encode(data)
	if e != nil {
		httpError(w, http.StatusInternalServerError, "Failed to encode to json: %v", e)
		return
	}
}

// Convert JSON object to Client object
func jsonToClient(cj clientJSON) (*Client, error) {
	c := Client{
		IP:                  cj.IP,
		MAC:                 cj.MAC,
		Name:                cj.Name,
		UseOwnSettings:      !cj.UseGlobalSettings,
		FilteringEnabled:    cj.FilteringEnabled,
		ParentalEnabled:     cj.ParentalEnabled,
		SafeSearchEnabled:   cj.SafeSearchEnabled,
		SafeBrowsingEnabled: cj.SafeBrowsingEnabled,

		UseOwnBlockedServices: !cj.UseGlobalBlockedServices,
		BlockedServices:       cj.BlockedServices,
	}
	return &c, nil
}

// Add a new client
func handleAddClient(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		httpError(w, http.StatusBadRequest, "failed to read request body: %s", err)
		return
	}

	cj := clientJSON{}
	err = json.Unmarshal(body, &cj)
	if err != nil {
		httpError(w, http.StatusBadRequest, "JSON parse: %s", err)
		return
	}

	c, err := jsonToClient(cj)
	if err != nil {
		httpError(w, http.StatusBadRequest, "%s", err)
		return
	}
	ok, err := config.clients.Add(*c)
	if err != nil {
		httpError(w, http.StatusBadRequest, "%s", err)
		return
	}
	if !ok {
		httpError(w, http.StatusBadRequest, "Client already exists")
		return
	}

	_ = writeAllConfigsAndReloadDNS()
	returnOK(w)
}

// Remove client
func handleDelClient(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		httpError(w, http.StatusBadRequest, "failed to read request body: %s", err)
		return
	}

	cj := clientJSON{}
	err = json.Unmarshal(body, &cj)
	if err != nil || len(cj.Name) == 0 {
		httpError(w, http.StatusBadRequest, "JSON parse: %s", err)
		return
	}

	if !config.clients.Del(cj.Name) {
		httpError(w, http.StatusBadRequest, "Client not found")
		return
	}

	_ = writeAllConfigsAndReloadDNS()
	returnOK(w)
}

type updateJSON struct {
	Name string     `json:"name"`
	Data clientJSON `json:"data"`
}

// Update client's properties
func handleUpdateClient(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		httpError(w, http.StatusBadRequest, "failed to read request body: %s", err)
		return
	}

	var dj updateJSON
	err = json.Unmarshal(body, &dj)
	if err != nil {
		httpError(w, http.StatusBadRequest, "JSON parse: %s", err)
		return
	}
	if len(dj.Name) == 0 {
		httpError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	c, err := jsonToClient(dj.Data)
	if err != nil {
		httpError(w, http.StatusBadRequest, "%s", err)
		return
	}

	err = config.clients.Update(dj.Name, *c)
	if err != nil {
		httpError(w, http.StatusBadRequest, "%s", err)
		return
	}

	_ = writeAllConfigsAndReloadDNS()
	returnOK(w)
}

// RegisterClientsHandlers registers HTTP handlers
func RegisterClientsHandlers() {
	httpRegister(http.MethodGet, "/control/clients", handleGetClients)
	httpRegister(http.MethodPost, "/control/clients/add", handleAddClient)
	httpRegister(http.MethodPost, "/control/clients/delete", handleDelClient)
	httpRegister(http.MethodPost, "/control/clients/update", handleUpdateClient)
}
