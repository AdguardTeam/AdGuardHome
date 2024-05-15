package dnsforward

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"slices"
	"strings"

	"github.com/AdguardTeam/AdGuardHome/internal/aghalg"
	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/golibs/container"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/stringutil"
	"github.com/AdguardTeam/urlfilter"
	"github.com/AdguardTeam/urlfilter/filterlist"
	"github.com/AdguardTeam/urlfilter/rules"
)

// accessManager controls IP and client blocking that takes place before all
// other processing.  An accessManager is safe for concurrent use.
type accessManager struct {
	allowedIPs *container.MapSet[netip.Addr]
	blockedIPs *container.MapSet[netip.Addr]

	allowedClientIDs *container.MapSet[string]
	blockedClientIDs *container.MapSet[string]

	// TODO(s.chzhen):  Use [aghnet.IgnoreEngine].
	blockedHostsEng *urlfilter.DNSEngine

	// TODO(a.garipov): Create a type for an efficient tree set of IP networks.
	allowedNets []netip.Prefix
	blockedNets []netip.Prefix
}

// processAccessClients is a helper for processing a list of client strings,
// which may be an IP address, a CIDR, or a ClientID.
func processAccessClients(
	clientStrs []string,
	ips *container.MapSet[netip.Addr],
	nets *[]netip.Prefix,
	clientIDs *container.MapSet[string],
) (err error) {
	for i, s := range clientStrs {
		var ip netip.Addr
		var ipnet netip.Prefix
		if ip, err = netip.ParseAddr(s); err == nil {
			ips.Add(ip)
		} else if ipnet, err = netip.ParsePrefix(s); err == nil {
			*nets = append(*nets, ipnet)
		} else {
			err = ValidateClientID(s)
			if err != nil {
				return fmt.Errorf("value %q at index %d: bad ip, cidr, or clientid", s, i)
			}

			clientIDs.Add(s)
		}
	}

	return nil
}

// newAccessCtx creates a new accessCtx.
func newAccessCtx(allowed, blocked, blockedHosts []string) (a *accessManager, err error) {
	a = &accessManager{
		allowedIPs: container.NewMapSet[netip.Addr](),
		blockedIPs: container.NewMapSet[netip.Addr](),

		allowedClientIDs: container.NewMapSet[string](),
		blockedClientIDs: container.NewMapSet[string](),
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
		stringutil.WriteToBuilder(b, strings.ToLower(h), "\n")
	}

	lists := []filterlist.RuleList{
		&filterlist.StringRuleList{
			ID:             0,
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
func (a *accessManager) allowlistMode() (ok bool) {
	return a.allowedIPs.Len() != 0 || a.allowedClientIDs.Len() != 0 || len(a.allowedNets) != 0
}

// isBlockedClientID returns true if the ClientID should be blocked.
func (a *accessManager) isBlockedClientID(id string) (ok bool) {
	allowlistMode := a.allowlistMode()
	if id == "" {
		// In allowlist mode, consider requests without ClientIDs blocked by
		// default.
		return allowlistMode
	}

	if allowlistMode {
		return !a.allowedClientIDs.Has(id)
	}

	return a.blockedClientIDs.Has(id)
}

// isBlockedHost returns true if host should be blocked.
func (a *accessManager) isBlockedHost(host string, qt rules.RRType) (ok bool) {
	_, ok = a.blockedHostsEng.MatchRequest(&urlfilter.DNSRequest{
		Hostname: host,
		DNSType:  qt,
	})

	return ok
}

// isBlockedIP returns the status of the IP address blocking as well as the rule
// that blocked it.
func (a *accessManager) isBlockedIP(ip netip.Addr) (blocked bool, rule string) {
	blocked = true
	ips := a.blockedIPs
	ipnets := a.blockedNets

	if a.allowlistMode() {
		// Enable allowlist mode and use the allowlist sets.
		blocked = false
		ips = a.allowedIPs
		ipnets = a.allowedNets
	}

	if ips.Has(ip) {
		return blocked, ip.String()
	}

	for _, ipnet := range ipnets {
		// Remove zone before checking because prefixes stip zones.
		//
		// TODO(d.kolyshev):  Cover with tests.
		if ipnet.Contains(ip.WithZone("")) {
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
		AllowedClients:    slices.Clone(s.conf.AllowedClients),
		DisallowedClients: slices.Clone(s.conf.DisallowedClients),
		BlockedHosts:      slices.Clone(s.conf.BlockedHosts),
	}
}

// handleAccessList handles requests to the GET /control/access/list endpoint.
func (s *Server) handleAccessList(w http.ResponseWriter, r *http.Request) {
	aghhttp.WriteJSONResponseOK(w, r, s.accessListJSON())
}

// validateAccessSet checks the internal accessListJSON lists.  To search for
// duplicates, we cannot compare the new stringutil.Set and []string, because
// creating a set for a large array can be an unnecessary algorithmic complexity
func validateAccessSet(list *accessListJSON) (err error) {
	allowed, err := validateStrUniq(list.AllowedClients)
	if err != nil {
		return fmt.Errorf("validating allowed clients: %w", err)
	}

	disallowed, err := validateStrUniq(list.DisallowedClients)
	if err != nil {
		return fmt.Errorf("validating disallowed clients: %w", err)
	}

	_, err = validateStrUniq(list.BlockedHosts)
	if err != nil {
		return fmt.Errorf("validating blocked hosts: %w", err)
	}

	merged := allowed.Merge(disallowed)
	err = merged.Validate()
	if err != nil {
		return fmt.Errorf("items in allowed and disallowed clients intersect: %w", err)
	}

	return nil
}

// validateStrUniq returns an informative error if clients are not unique.
func validateStrUniq(clients []string) (uc aghalg.UniqChecker[string], err error) {
	uc = make(aghalg.UniqChecker[string], len(clients))
	for _, c := range clients {
		uc.Add(c)
	}

	return uc, uc.Validate()
}

// handleAccessSet handles requests to the POST /control/access/set endpoint.
func (s *Server) handleAccessSet(w http.ResponseWriter, r *http.Request) {
	list := &accessListJSON{}
	err := json.NewDecoder(r.Body).Decode(&list)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "decoding request: %s", err)

		return
	}

	err = validateAccessSet(list)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, err.Error())

		return
	}

	var a *accessManager
	a, err = newAccessCtx(list.AllowedClients, list.DisallowedClients, list.BlockedHosts)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "creating access ctx: %s", err)

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
