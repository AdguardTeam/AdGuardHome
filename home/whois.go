package home

import (
	"strings"
	"sync"

	"github.com/AdguardTeam/golibs/log"
	whois "github.com/likexian/whois-go"
)

// Whois - module context
type Whois struct {
	clients *clientsContainer
	ips     map[string]bool
	lock    sync.Mutex
	ipChan  chan string
}

// Create module context
func initWhois(clients *clientsContainer) *Whois {
	w := Whois{}
	w.clients = clients
	w.ips = make(map[string]bool)
	w.ipChan = make(chan string, 255)
	go w.workerLoop()
	return &w
}

// Parse plain-text data from the response
func whoisParse(data string) map[string]string {
	m := map[string]string{}
	lines := strings.Split(data, "\n")
	for _, ln := range lines {
		ln = strings.TrimSpace(ln)

		if len(ln) == 0 || ln[0] == '#' {
			continue
		}

		kv := strings.SplitN(ln, ":", 2)
		if len(kv) != 2 {
			continue
		}
		k := strings.TrimSpace(kv[0])
		k = strings.ToLower(k)
		v := strings.TrimSpace(kv[1])

		if k == "orgname" || k == "org-name" {
			m["orgname"] = v
		} else if k == "city" {
			m["city"] = v
		} else if k == "country" {
			m["country"] = v
		}
	}
	return m
}

// Request WHOIS information
func whoisProcess(ip string) [][]string {
	data := [][]string{}
	resp, err := whois.Whois(ip)
	if err != nil {
		log.Debug("Whois: error: %s  IP:%s", err, ip)
		return data
	}

	log.Debug("Whois: IP:%s  response: %d bytes", ip, len(resp))

	m := whoisParse(resp)

	keys := []string{"orgname", "country", "city"}
	for _, k := range keys {
		v, found := m[k]
		if !found {
			continue
		}
		pair := []string{k, v}
		data = append(data, pair)
	}

	return data
}

// Begin - begin requesting WHOIS info
func (w *Whois) Begin(ip string) {
	w.lock.Lock()
	_, found := w.ips[ip]
	if found {
		w.lock.Unlock()
		return
	}
	w.ips[ip] = true
	w.lock.Unlock()

	log.Debug("Whois: adding %s", ip)
	select {
	case w.ipChan <- ip:
		//
	default:
		log.Debug("Whois: queue is full")
	}
}

// Get IP address from channel; get WHOIS info; associate info with a client
func (w *Whois) workerLoop() {
	for {
		var ip string
		ip = <-w.ipChan

		info := whoisProcess(ip)
		if len(info) == 0 {
			continue
		}

		w.clients.SetWhoisInfo(ip, info)
	}
}
