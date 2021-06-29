package dnsforward

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/aghstrings"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/urlfilter"
	"github.com/AdguardTeam/urlfilter/filterlist"
)

// accessCtx controls IP and client blocking that takes place before all other
// processing.  An accessCtx is safe for concurrent use.
type accessCtx struct {
	allowedIPs *aghnet.IPMap
	blockedIPs *aghnet.IPMap

	allowedClientIDs *aghstrings.Set
	blockedClientIDs *aghstrings.Set

	blockedHostsEng *urlfilter.DNSEngine

	// TODO(a.garipov): Create a type for a set of IP networks.
	// aghnet.IPNetSet?
	allowedNets []*net.IPNet
	blockedNets []*net.IPNet
}

// unit is a convenient alias for struct{}
type unit = struct{}

// processAccessClients is a helper for processing a list of client strings,
// which may be an IP address, a CIDR, or a ClientID.
func processAccessClients(
	clientStrs []string,
	ips *aghnet.IPMap,
	nets *[]*net.IPNet,
	clientIDs *aghstrings.Set,
) (err error) {
	for i, s := range clientStrs {
		if ip := net.ParseIP(s); ip != nil {
			ips.Set(ip, unit{})
		} else if cidrIP, ipnet, cidrErr := net.ParseCIDR(s); cidrErr == nil {
			ipnet.IP = cidrIP
			*nets = append(*nets, ipnet)
		} else {
			idErr := ValidateClientID(s)
			if idErr != nil {
				return fmt.Errorf(
					"value %q at index %d: bad ip, cidr, or clientid",
					s,
					i,
				)
			}

			clientIDs.Add(s)
		}
	}

	return nil
}

// newAccessCtx creates a new accessCtx.
func newAccessCtx(allowed, blocked, blockedHosts []string) (a *accessCtx, err error) {
	a = &accessCtx{
		allowedIPs: aghnet.NewIPMap(0),
		blockedIPs: aghnet.NewIPMap(0),

		allowedClientIDs: aghstrings.NewSet(),
		blockedClientIDs: aghstrings.NewSet(),
	}

	err = processAccessClients(allowed, a.allowedIPs, &a.allowedNets, a.allowedClientIDs)
	if err != nil {
		return nil, fmt.Errorf("adding allowed: %w", err)
	}

	err = processAccessClients(blocked, a.blockedIPs, &a.blockedNets, a.blockedClientIDs)
	if err != nil {
		return nil, fmt.Errorf("adding blocked: %w", err)
	}

	b := &strings.Builder{}
	for _, h := range blockedHosts {
		aghstrings.WriteToBuilder(b, strings.ToLower(h), "\n")
	}

	lists := []filterlist.RuleList{
		&filterlist.StringRuleList{
			ID:             int(0),
			RulesText:      b.String(),
			IgnoreCosmetic: true,
		},
	}

	rulesStrg, err := filterlist.NewRuleStorage(lists)
	if err != nil {
		return nil, fmt.Errorf("adding blocked hosts: %w", err)
	}

	a.blockedHostsEng = urlfilter.NewDNSEngine(rulesStrg)

	return a, nil
}

// allowlistMode returns true if this *accessCtx is in the allowlist mode.
func (a *accessCtx) allowlistMode() (ok bool) {
	return a.allowedIPs.Len() != 0 || a.allowedClientIDs.Len() != 0 || len(a.allowedNets) != 0
}

// isBlockedClientID returns true if the ClientID should be blocked.
func (a *accessCtx) isBlockedClientID(id string) (ok bool) {
	allowlistMode := a.allowlistMode()
	if id == "" {
		// In allowlist mode, consider requests without client IDs
		// blocked by default.
		return allowlistMode
	}

	if allowlistMode {
		return !a.allowedClientIDs.Has(id)
	}

	return a.blockedClientIDs.Has(id)
}

// isBlockedHost returns true if host should be blocked.
func (a *accessCtx) isBlockedHost(host string) (ok bool) {
	_, ok = a.blockedHostsEng.Match(strings.ToLower(host))

	return ok
}

// isBlockedIP returns the status of the IP address blocking as well as the rule
// that blocked it.
func (a *accessCtx) isBlockedIP(ip net.IP) (blocked bool, rule string) {
	blocked = true
	ips := a.blockedIPs
	ipnets := a.blockedNets

	if a.allowlistMode() {
		// Enable allowlist mode and use the allowlist sets.
		blocked = false
		ips = a.allowedIPs
		ipnets = a.allowedNets
	}

	if _, ok := ips.Get(ip); ok {
		return blocked, ip.String()
	}

	for _, ipnet := range ipnets {
		if ipnet.Contains(ip) {
			return blocked, ipnet.String()
		}
	}

	return !blocked, ""
}

type accessListJSON struct {
	AllowedClients    []string `json:"allowed_clients"`
	DisallowedClients []string `json:"disallowed_clients"`
	BlockedHosts      []string `json:"blocked_hosts"`
}

func (s *Server) accessListJSON() (j accessListJSON) {
	s.serverLock.RLock()
	defer s.serverLock.RUnlock()

	return accessListJSON{
		AllowedClients:    aghstrings.CloneSlice(s.conf.AllowedClients),
		DisallowedClients: aghstrings.CloneSlice(s.conf.DisallowedClients),
		BlockedHosts:      aghstrings.CloneSlice(s.conf.BlockedHosts),
	}
}

func (s *Server) handleAccessList(w http.ResponseWriter, r *http.Request) {
	j := s.accessListJSON()

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(j)
	if err != nil {
		httpError(r, w, http.StatusInternalServerError, "encoding response: %s", err)

		return
	}
}

func (s *Server) handleAccessSet(w http.ResponseWriter, r *http.Request) {
	list := accessListJSON{}
	err := json.NewDecoder(r.Body).Decode(&list)
	if err != nil {
		httpError(r, w, http.StatusBadRequest, "decoding request: %s", err)

		return
	}

	var a *accessCtx
	a, err = newAccessCtx(list.AllowedClients, list.DisallowedClients, list.BlockedHosts)
	if err != nil {
		httpError(r, w, http.StatusBadRequest, "creating access ctx: %s", err)

		return
	}

	defer log.Debug(
		"access: updated lists: %d, %d, %d",
		len(list.AllowedClients),
		len(list.DisallowedClients),
		len(list.BlockedHosts),
	)

	defer s.conf.ConfigModified()

	s.serverLock.Lock()
	defer s.serverLock.Unlock()

	s.conf.AllowedClients = list.AllowedClients
	s.conf.DisallowedClients = list.DisallowedClients
	s.conf.BlockedHosts = list.BlockedHosts
	s.access = a
}
