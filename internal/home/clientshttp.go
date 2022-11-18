package home

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
)

// clientJSON is a common structure used by several handlers to deal with
// clients.  Some of the fields are only necessary in one or two handlers and
// are thus made pointers with an omitempty tag.
//
// TODO(a.garipov): Consider using nullbool and an optional string here?  Or
// split into several structs?
type clientJSON struct {
	// Disallowed, if non-nil and false, means that the client's IP is
	// allowed.  Otherwise, the IP is blocked.
	Disallowed *bool `json:"disallowed,omitempty"`

	// DisallowedRule is the rule due to which the client is disallowed.
	// If Disallowed is true and this string is empty, the client IP is
	// disallowed by the "allowed IP list", that is it is not included in
	// the allowlist.
	DisallowedRule *string `json:"disallowed_rule,omitempty"`

	WHOISInfo *RuntimeClientWHOISInfo `json:"whois_info,omitempty"`

	Name string `json:"name"`

	BlockedServices []string `json:"blocked_services"`
	IDs             []string `json:"ids"`
	Tags            []string `json:"tags"`
	Upstreams       []string `json:"upstreams"`

	FilteringEnabled         bool `json:"filtering_enabled"`
	ParentalEnabled          bool `json:"parental_enabled"`
	SafeBrowsingEnabled      bool `json:"safebrowsing_enabled"`
	SafeSearchEnabled        bool `json:"safesearch_enabled"`
	UseGlobalBlockedServices bool `json:"use_global_blocked_services"`
	UseGlobalSettings        bool `json:"use_global_settings"`
}

type runtimeClientJSON struct {
	WHOISInfo *RuntimeClientWHOISInfo `json:"whois_info"`

	Name   string       `json:"name"`
	IP     netip.Addr   `json:"ip"`
	Source clientSource `json:"source"`
}

type clientListJSON struct {
	Clients        []*clientJSON       `json:"clients"`
	RuntimeClients []runtimeClientJSON `json:"auto_clients"`
	Tags           []string            `json:"supported_tags"`
}

// respond with information about configured clients
func (clients *clientsContainer) handleGetClients(w http.ResponseWriter, r *http.Request) {
	data := clientListJSON{}

	clients.lock.Lock()
	defer clients.lock.Unlock()

	for _, c := range clients.list {
		cj := clientToJSON(c)
		data.Clients = append(data.Clients, cj)
	}

	for ip, rc := range clients.ipToRC {
		cj := runtimeClientJSON{
			WHOISInfo: rc.WHOISInfo,

			Name:   rc.Host,
			Source: rc.Source,
			IP:     ip,
		}

		data.RuntimeClients = append(data.RuntimeClients, cj)
	}

	data.Tags = clientTags

	_ = aghhttp.WriteJSONResponse(w, r, data)
}

// Convert JSON object to Client object
func jsonToClient(cj clientJSON) (c *Client) {
	return &Client{
		Name:                cj.Name,
		IDs:                 cj.IDs,
		Tags:                cj.Tags,
		UseOwnSettings:      !cj.UseGlobalSettings,
		FilteringEnabled:    cj.FilteringEnabled,
		ParentalEnabled:     cj.ParentalEnabled,
		SafeSearchEnabled:   cj.SafeSearchEnabled,
		SafeBrowsingEnabled: cj.SafeBrowsingEnabled,

		UseOwnBlockedServices: !cj.UseGlobalBlockedServices,
		BlockedServices:       cj.BlockedServices,

		Upstreams: cj.Upstreams,
	}
}

// Convert Client object to JSON
func clientToJSON(c *Client) (cj *clientJSON) {
	return &clientJSON{
		Name:                c.Name,
		IDs:                 c.IDs,
		Tags:                c.Tags,
		UseGlobalSettings:   !c.UseOwnSettings,
		FilteringEnabled:    c.FilteringEnabled,
		ParentalEnabled:     c.ParentalEnabled,
		SafeSearchEnabled:   c.SafeSearchEnabled,
		SafeBrowsingEnabled: c.SafeBrowsingEnabled,

		UseGlobalBlockedServices: !c.UseOwnBlockedServices,
		BlockedServices:          c.BlockedServices,

		Upstreams: c.Upstreams,
	}
}

// Add a new client
func (clients *clientsContainer) handleAddClient(w http.ResponseWriter, r *http.Request) {
	cj := clientJSON{}
	err := json.NewDecoder(r.Body).Decode(&cj)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "failed to process request body: %s", err)

		return
	}

	c := jsonToClient(cj)
	ok, err := clients.Add(c)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "%s", err)

		return
	}

	if !ok {
		aghhttp.Error(r, w, http.StatusBadRequest, "Client already exists")

		return
	}

	onConfigModified()
}

// Remove client
func (clients *clientsContainer) handleDelClient(w http.ResponseWriter, r *http.Request) {
	cj := clientJSON{}
	err := json.NewDecoder(r.Body).Decode(&cj)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "failed to process request body: %s", err)

		return
	}

	if len(cj.Name) == 0 {
		aghhttp.Error(r, w, http.StatusBadRequest, "client's name must be non-empty")

		return
	}

	if !clients.Del(cj.Name) {
		aghhttp.Error(r, w, http.StatusBadRequest, "Client not found")

		return
	}

	onConfigModified()
}

type updateJSON struct {
	Name string     `json:"name"`
	Data clientJSON `json:"data"`
}

// Update client's properties
func (clients *clientsContainer) handleUpdateClient(w http.ResponseWriter, r *http.Request) {
	dj := updateJSON{}
	err := json.NewDecoder(r.Body).Decode(&dj)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "failed to process request body: %s", err)

		return
	}

	if len(dj.Name) == 0 {
		aghhttp.Error(r, w, http.StatusBadRequest, "Invalid request")

		return
	}

	c := jsonToClient(dj.Data)
	err = clients.Update(dj.Name, c)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "%s", err)

		return
	}

	onConfigModified()
}

// Get the list of clients by IP address list
func (clients *clientsContainer) handleFindClient(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	data := []map[string]*clientJSON{}
	for i := 0; i < len(q); i++ {
		idStr := q.Get(fmt.Sprintf("ip%d", i))
		if idStr == "" {
			break
		}

		ip, _ := netip.ParseAddr(idStr)
		c, ok := clients.Find(idStr)
		var cj *clientJSON
		if !ok {
			cj = clients.findRuntime(ip, idStr)
		} else {
			cj = clientToJSON(c)
			disallowed, rule := clients.dnsServer.IsBlockedClient(ip, idStr)
			cj.Disallowed, cj.DisallowedRule = &disallowed, &rule
		}

		data = append(data, map[string]*clientJSON{
			idStr: cj,
		})
	}

	_ = aghhttp.WriteJSONResponse(w, r, data)
}

// findRuntime looks up the IP in runtime and temporary storages, like
// /etc/hosts tables, DHCP leases, or blocklists.  cj is guaranteed to be
// non-nil.
func (clients *clientsContainer) findRuntime(ip netip.Addr, idStr string) (cj *clientJSON) {
	rc, ok := clients.findRuntimeClient(ip)
	if !ok {
		// It is still possible that the IP used to be in the runtime clients
		// list, but then the server was reloaded.  So, check the DNS server's
		// blocked IP list.
		//
		// See https://github.com/AdguardTeam/AdGuardHome/issues/2428.
		disallowed, rule := clients.dnsServer.IsBlockedClient(ip, idStr)
		cj = &clientJSON{
			IDs:            []string{idStr},
			Disallowed:     &disallowed,
			DisallowedRule: &rule,
			WHOISInfo:      &RuntimeClientWHOISInfo{},
		}

		return cj
	}

	cj = &clientJSON{
		Name:      rc.Host,
		IDs:       []string{idStr},
		WHOISInfo: rc.WHOISInfo,
	}

	disallowed, rule := clients.dnsServer.IsBlockedClient(ip, idStr)
	cj.Disallowed, cj.DisallowedRule = &disallowed, &rule

	return cj
}

// RegisterClientsHandlers registers HTTP handlers
func (clients *clientsContainer) registerWebHandlers() {
	httpRegister(http.MethodGet, "/control/clients", clients.handleGetClients)
	httpRegister(http.MethodPost, "/control/clients/add", clients.handleAddClient)
	httpRegister(http.MethodPost, "/control/clients/delete", clients.handleDelClient)
	httpRegister(http.MethodPost, "/control/clients/update", clients.handleUpdateClient)
	httpRegister(http.MethodGet, "/control/clients/find", clients.handleFindClient)
}
