package dnsforward

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/urlfilter"
	"github.com/AdguardTeam/urlfilter/filterlist"
)

type accessCtx struct {
	lock sync.Mutex

	allowedClients    map[string]bool // IP addresses of whitelist clients
	disallowedClients map[string]bool // IP addresses of clients that should be blocked

	allowedClientsIPNet    []net.IPNet // CIDRs of whitelist clients
	disallowedClientsIPNet []net.IPNet // CIDRs of clients that should be blocked

	blockedHostsEngine *urlfilter.DNSEngine // finds hosts that should be blocked
}

func (a *accessCtx) Init(allowedClients, disallowedClients, blockedHosts []string) error {
	err := processIPCIDRArray(&a.allowedClients, &a.allowedClientsIPNet, allowedClients)
	if err != nil {
		return err
	}

	err = processIPCIDRArray(&a.disallowedClients, &a.disallowedClientsIPNet, disallowedClients)
	if err != nil {
		return err
	}

	buf := strings.Builder{}
	for _, s := range blockedHosts {
		buf.WriteString(s)
		buf.WriteString("\n")
	}

	listArray := []filterlist.RuleList{}
	list := &filterlist.StringRuleList{
		ID:             int(0),
		RulesText:      buf.String(),
		IgnoreCosmetic: true,
	}
	listArray = append(listArray, list)
	rulesStorage, err := filterlist.NewRuleStorage(listArray)
	if err != nil {
		return fmt.Errorf("filterlist.NewRuleStorage(): %s", err)
	}
	a.blockedHostsEngine = urlfilter.NewDNSEngine(rulesStorage)

	return nil
}

// Split array of IP or CIDR into 2 containers for fast search
func processIPCIDRArray(dst *map[string]bool, dstIPNet *[]net.IPNet, src []string) error {
	*dst = make(map[string]bool)

	for _, s := range src {
		ip := net.ParseIP(s)
		if ip != nil {
			(*dst)[s] = true
			continue
		}

		_, ipnet, err := net.ParseCIDR(s)
		if err != nil {
			return err
		}
		*dstIPNet = append(*dstIPNet, *ipnet)
	}

	return nil
}

// IsBlockedIP - return TRUE if this client should be blocked
// Returns the item from the "disallowedClients" list that lead to blocking IP.
// If it returns TRUE and an empty string, it means that the "allowedClients" is not empty,
// but the ip does not belong to it.
func (a *accessCtx) IsBlockedIP(ip string) (bool, string) {
	a.lock.Lock()
	defer a.lock.Unlock()

	if len(a.allowedClients) != 0 || len(a.allowedClientsIPNet) != 0 {
		_, ok := a.allowedClients[ip]
		if ok {
			return false, ""
		}

		if len(a.allowedClientsIPNet) != 0 {
			ipAddr := net.ParseIP(ip)
			for _, ipnet := range a.allowedClientsIPNet {
				if ipnet.Contains(ipAddr) {
					return false, ""
				}
			}
		}

		return true, ""
	}

	_, ok := a.disallowedClients[ip]
	if ok {
		return true, ip
	}

	if len(a.disallowedClientsIPNet) != 0 {
		ipAddr := net.ParseIP(ip)
		for _, ipnet := range a.disallowedClientsIPNet {
			if ipnet.Contains(ipAddr) {
				return true, ipnet.String()
			}
		}
	}

	return false, ""
}

// IsBlockedDomain - return TRUE if this domain should be blocked
func (a *accessCtx) IsBlockedDomain(host string) bool {
	a.lock.Lock()
	_, ok := a.blockedHostsEngine.Match(host)
	a.lock.Unlock()
	return ok
}

type accessListJSON struct {
	AllowedClients    []string `json:"allowed_clients"`
	DisallowedClients []string `json:"disallowed_clients"`
	BlockedHosts      []string `json:"blocked_hosts"`
}

func (s *Server) handleAccessList(w http.ResponseWriter, r *http.Request) {
	s.RLock()
	j := accessListJSON{
		AllowedClients:    s.conf.AllowedClients,
		DisallowedClients: s.conf.DisallowedClients,
		BlockedHosts:      s.conf.BlockedHosts,
	}
	s.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(j)
	if err != nil {
		httpError(r, w, http.StatusInternalServerError, "json.Encode: %s", err)
		return
	}
}

func checkIPCIDRArray(src []string) error {
	for _, s := range src {
		ip := net.ParseIP(s)
		if ip != nil {
			continue
		}

		_, _, err := net.ParseCIDR(s)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) handleAccessSet(w http.ResponseWriter, r *http.Request) {
	j := accessListJSON{}
	err := json.NewDecoder(r.Body).Decode(&j)
	if err != nil {
		httpError(r, w, http.StatusBadRequest, "json.Decode: %s", err)
		return
	}

	err = checkIPCIDRArray(j.AllowedClients)
	if err == nil {
		err = checkIPCIDRArray(j.DisallowedClients)
	}
	if err != nil {
		httpError(r, w, http.StatusBadRequest, "%s", err)
		return
	}

	a := &accessCtx{}
	err = a.Init(j.AllowedClients, j.DisallowedClients, j.BlockedHosts)
	if err != nil {
		httpError(r, w, http.StatusBadRequest, "access.Init: %s", err)
		return
	}

	s.Lock()
	s.conf.AllowedClients = j.AllowedClients
	s.conf.DisallowedClients = j.DisallowedClients
	s.conf.BlockedHosts = j.BlockedHosts
	s.access = a
	s.Unlock()
	s.conf.ConfigModified()

	log.Debug("Access: updated lists: %d, %d, %d",
		len(j.AllowedClients), len(j.DisallowedClients), len(j.BlockedHosts))
}
