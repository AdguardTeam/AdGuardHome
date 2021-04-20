package dnsforward

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/AdguardTeam/AdGuardHome/internal/aghstrings"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/urlfilter"
	"github.com/AdguardTeam/urlfilter/filterlist"
)

type accessCtx struct {
	lock sync.Mutex

	// allowedClients are the IP addresses of clients in the allowlist.
	allowedClients *aghstrings.Set

	// disallowedClients are the IP addresses of clients in the blocklist.
	disallowedClients *aghstrings.Set

	allowedClientsIPNet    []net.IPNet // CIDRs of whitelist clients
	disallowedClientsIPNet []net.IPNet // CIDRs of clients that should be blocked

	blockedHostsEngine *urlfilter.DNSEngine // finds hosts that should be blocked
}

func newAccessCtx(allowedClients, disallowedClients, blockedHosts []string) (a *accessCtx, err error) {
	a = &accessCtx{
		allowedClients:    aghstrings.NewSet(),
		disallowedClients: aghstrings.NewSet(),
	}

	err = processIPCIDRArray(a.allowedClients, &a.allowedClientsIPNet, allowedClients)
	if err != nil {
		return nil, fmt.Errorf("processing allowed clients: %w", err)
	}

	err = processIPCIDRArray(a.disallowedClients, &a.disallowedClientsIPNet, disallowedClients)
	if err != nil {
		return nil, fmt.Errorf("processing disallowed clients: %w", err)
	}

	b := &strings.Builder{}
	for _, s := range blockedHosts {
		aghstrings.WriteToBuilder(b, s, "\n")
	}

	listArray := []filterlist.RuleList{}
	list := &filterlist.StringRuleList{
		ID:             int(0),
		RulesText:      b.String(),
		IgnoreCosmetic: true,
	}
	listArray = append(listArray, list)
	rulesStorage, err := filterlist.NewRuleStorage(listArray)
	if err != nil {
		return nil, fmt.Errorf("filterlist.NewRuleStorage(): %w", err)
	}
	a.blockedHostsEngine = urlfilter.NewDNSEngine(rulesStorage)

	return a, nil
}

// Split array of IP or CIDR into 2 containers for fast search
func processIPCIDRArray(dst *aghstrings.Set, dstIPNet *[]net.IPNet, src []string) error {
	for _, s := range src {
		ip := net.ParseIP(s)
		if ip != nil {
			dst.Add(s)

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
func (a *accessCtx) IsBlockedIP(ip net.IP) (bool, string) {
	ipStr := ip.String()

	a.lock.Lock()
	defer a.lock.Unlock()

	if a.allowedClients.Len() != 0 || len(a.allowedClientsIPNet) != 0 {
		if a.allowedClients.Has(ipStr) {
			return false, ""
		}

		if len(a.allowedClientsIPNet) != 0 {
			for _, ipnet := range a.allowedClientsIPNet {
				if ipnet.Contains(ip) {
					return false, ""
				}
			}
		}

		return true, ""
	}

	if a.disallowedClients.Has(ipStr) {
		return true, ipStr
	}

	if len(a.disallowedClientsIPNet) != 0 {
		for _, ipnet := range a.disallowedClientsIPNet {
			if ipnet.Contains(ip) {
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

	var a *accessCtx
	a, err = newAccessCtx(j.AllowedClients, j.DisallowedClients, j.BlockedHosts)
	if err != nil {
		httpError(r, w, http.StatusBadRequest, "creating access ctx: %s", err)

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
