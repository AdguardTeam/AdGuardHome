package home

import (
	"context"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"net"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/util"

	"github.com/AdguardTeam/golibs/cache"
	"github.com/AdguardTeam/golibs/log"
)

const (
	defaultServer  = "whois.arin.net"
	defaultPort    = "43"
	maxValueLength = 250
	whoisTTL       = 1 * 60 * 60 // 1 hour
)

// Whois - module context
type Whois struct {
	clients     *clientsContainer
	ipChan      chan string
	timeoutMsec uint

	// Contains IP addresses of clients
	// An active IP address is resolved once again after it expires.
	// If IP address couldn't be resolved, it stays here for some time to prevent further attempts to resolve the same IP.
	ipAddrs cache.Cache
}

// Create module context
func initWhois(clients *clientsContainer) *Whois {
	w := Whois{}
	w.timeoutMsec = 5000
	w.clients = clients

	cconf := cache.Config{}
	cconf.EnableLRU = true
	cconf.MaxCount = 10000
	w.ipAddrs = cache.New(cconf)

	w.ipChan = make(chan string, 255)
	go w.workerLoop()
	return &w
}

// If the value is too large - cut it and append "..."
func trimValue(s string) string {
	if len(s) <= maxValueLength {
		return s
	}
	return s[:maxValueLength-3] + "..."
}

// Parse plain-text data from the response
func whoisParse(data string) map[string]string {
	m := map[string]string{}
	descr := ""
	netname := ""
	for len(data) != 0 {
		ln := util.SplitNext(&data, '\n')
		if len(ln) == 0 || ln[0] == '#' || ln[0] == '%' {
			continue
		}

		kv := strings.SplitN(ln, ":", 2)
		if len(kv) != 2 {
			continue
		}
		k := strings.TrimSpace(kv[0])
		k = strings.ToLower(k)
		v := strings.TrimSpace(kv[1])

		switch k {
		case "org-name":
			m["orgname"] = trimValue(v)
		case "orgname":
			fallthrough
		case "city":
			fallthrough
		case "country":
			m[k] = trimValue(v)

		case "descr":
			if len(descr) == 0 {
				descr = v
			}
		case "netname":
			netname = v

		case "whois": // "whois: whois.arin.net"
			m["whois"] = v

		case "referralserver": // "ReferralServer:  whois://whois.ripe.net"
			if strings.HasPrefix(v, "whois://") {
				m["whois"] = v[len("whois://"):]
			}
		}
	}

	// descr or netname -> orgname
	_, ok := m["orgname"]
	if !ok && len(descr) != 0 {
		m["orgname"] = trimValue(descr)
	} else if !ok && len(netname) != 0 {
		m["orgname"] = trimValue(netname)
	}

	return m
}

// Send request to a server and receive the response
func (w *Whois) query(target string, serverAddr string) (string, error) {
	addr, _, _ := net.SplitHostPort(serverAddr)
	if addr == "whois.arin.net" {
		target = "n + " + target
	}
	conn, err := customDialContext(context.TODO(), "tcp", serverAddr)
	if err != nil {
		return "", err
	}
	defer conn.Close()

	_ = conn.SetReadDeadline(time.Now().Add(time.Duration(w.timeoutMsec) * time.Millisecond))
	_, err = conn.Write([]byte(target + "\r\n"))
	if err != nil {
		return "", err
	}

	data, err := ioutil.ReadAll(conn)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// Query WHOIS servers (handle redirects)
func (w *Whois) queryAll(target string) (string, error) {
	server := net.JoinHostPort(defaultServer, defaultPort)
	const maxRedirects = 5
	for i := 0; i != maxRedirects; i++ {
		resp, err := w.query(target, server)
		if err != nil {
			return "", err
		}
		log.Debug("Whois: received response (%d bytes) from %s  IP:%s", len(resp), server, target)

		m := whoisParse(resp)
		redir, ok := m["whois"]
		if !ok {
			return resp, nil
		}
		redir = strings.ToLower(redir)

		_, _, err = net.SplitHostPort(redir)
		if err != nil {
			server = net.JoinHostPort(redir, defaultPort)
		} else {
			server = redir
		}

		log.Debug("Whois: redirected to %s  IP:%s", redir, target)
	}
	return "", fmt.Errorf("Whois: redirect loop")
}

// Request WHOIS information
func (w *Whois) process(ip string) [][]string {
	data := [][]string{}
	resp, err := w.queryAll(ip)
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
	now := uint64(time.Now().Unix())
	expire := w.ipAddrs.Get([]byte(ip))
	if len(expire) != 0 {
		exp := binary.BigEndian.Uint64(expire)
		if exp > now {
			return
		}
		// TTL expired
	}
	expire = make([]byte, 8)
	binary.BigEndian.PutUint64(expire, now+whoisTTL)
	_ = w.ipAddrs.Set([]byte(ip), expire)

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

		info := w.process(ip)
		if len(info) == 0 {
			continue
		}

		w.clients.SetWhoisInfo(ip, info)
	}
}
