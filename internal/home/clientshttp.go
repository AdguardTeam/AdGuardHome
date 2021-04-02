package home

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
)

type clientJSON struct {
	IDs                 []string `json:"ids"`
	Tags                []string `json:"tags"`
	Name                string   `json:"name"`
	UseGlobalSettings   bool     `json:"use_global_settings"`
	FilteringEnabled    bool     `json:"filtering_enabled"`
	ParentalEnabled     bool     `json:"parental_enabled"`
	SafeSearchEnabled   bool     `json:"safesearch_enabled"`
	SafeBrowsingEnabled bool     `json:"safebrowsing_enabled"`

	UseGlobalBlockedServices bool     `json:"use_global_blocked_services"`
	BlockedServices          []string `json:"blocked_services"`

	Upstreams []string `json:"upstreams"`

	WhoisInfo *RuntimeClientWhoisInfo `json:"whois_info"`

	// Disallowed - if true -- client's IP is not disallowed
	// Otherwise, it is blocked.
	Disallowed bool `json:"disallowed"`

	// DisallowedRule - the rule due to which the client is disallowed
	// If Disallowed is true, and this string is empty - it means that the client IP
	// is disallowed by the "allowed IP list", i.e. it is not included in allowed.
	DisallowedRule string `json:"disallowed_rule"`
}

type runtimeClientJSON struct {
	WhoisInfo *RuntimeClientWhoisInfo `json:"whois_info"`

	IP     string `json:"ip"`
	Name   string `json:"name"`
	Source string `json:"source"`
}

type clientListJSON struct {
	Clients        []clientJSON        `json:"clients"`
	RuntimeClients []runtimeClientJSON `json:"auto_clients"`
	Tags           []string            `json:"supported_tags"`
}

// respond with information about configured clients
func (clients *clientsContainer) handleGetClients(w http.ResponseWriter, _ *http.Request) {
	data := clientListJSON{}

	clients.lock.Lock()
	defer clients.lock.Unlock()

	for _, c := range clients.list {
		cj := clientToJSON(c)
		data.Clients = append(data.Clients, cj)
	}
	for ip, rc := range clients.ipToRC {
		cj := runtimeClientJSON{
			IP:        ip,
			Name:      rc.Host,
			WhoisInfo: rc.WhoisInfo,
		}

		cj.Source = "etc/hosts"
		switch rc.Source {
		case ClientSourceDHCP:
			cj.Source = "DHCP"
		case ClientSourceRDNS:
			cj.Source = "rDNS"
		case ClientSourceARP:
			cj.Source = "ARP"
		case ClientSourceWHOIS:
			cj.Source = "WHOIS"
		}

		data.RuntimeClients = append(data.RuntimeClients, cj)
	}

	data.Tags = clientTags

	w.Header().Set("Content-Type", "application/json")
	e := json.NewEncoder(w).Encode(data)
	if e != nil {
		httpError(w, http.StatusInternalServerError, "Failed to encode to json: %v", e)
		return
	}
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
func clientToJSON(c *Client) clientJSON {
	cj := clientJSON{
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

		WhoisInfo: &RuntimeClientWhoisInfo{},
	}

	return cj
}

// runtimeClientToJSON converts a RuntimeClient into a JSON struct.
func runtimeClientToJSON(ip string, rc RuntimeClient) (cj clientJSON) {
	cj = clientJSON{
		Name:      rc.Host,
		IDs:       []string{ip},
		WhoisInfo: rc.WhoisInfo,
	}

	return cj
}

// Add a new client
func (clients *clientsContainer) handleAddClient(w http.ResponseWriter, r *http.Request) {
	cj := clientJSON{}
	err := json.NewDecoder(r.Body).Decode(&cj)
	if err != nil {
		httpError(w, http.StatusBadRequest, "failed to process request body: %s", err)

		return
	}

	c := jsonToClient(cj)
	ok, err := clients.Add(c)
	if err != nil {
		httpError(w, http.StatusBadRequest, "%s", err)
		return
	}
	if !ok {
		httpError(w, http.StatusBadRequest, "Client already exists")
		return
	}

	onConfigModified()
}

// Remove client
func (clients *clientsContainer) handleDelClient(w http.ResponseWriter, r *http.Request) {
	cj := clientJSON{}
	err := json.NewDecoder(r.Body).Decode(&cj)
	if err != nil {
		httpError(w, http.StatusBadRequest, "failed to process request body: %s", err)

		return
	}

	if len(cj.Name) == 0 {
		httpError(w, http.StatusBadRequest, "client's name must be non-empty")

		return
	}

	if !clients.Del(cj.Name) {
		httpError(w, http.StatusBadRequest, "Client not found")
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
		httpError(w, http.StatusBadRequest, "failed to process request body: %s", err)

		return
	}

	if len(dj.Name) == 0 {
		httpError(w, http.StatusBadRequest, "Invalid request")
		return
	}

	c := jsonToClient(dj.Data)
	err = clients.Update(dj.Name, c)
	if err != nil {
		httpError(w, http.StatusBadRequest, "%s", err)
		return
	}

	onConfigModified()
}

// Get the list of clients by IP address list
func (clients *clientsContainer) handleFindClient(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	data := []map[string]clientJSON{}
	for i := 0; i < len(q); i++ {
		idStr := q.Get(fmt.Sprintf("ip%d", i))
		if idStr == "" {
			break
		}

		ip := net.ParseIP(idStr)
		c, ok := clients.Find(idStr)
		var cj clientJSON
		if !ok {
			var found bool
			cj, found = clients.findTemporary(ip, idStr)
			if !found {
				continue
			}
		} else {
			cj = clientToJSON(c)
			cj.Disallowed, cj.DisallowedRule = clients.dnsServer.IsBlockedIP(ip)
		}

		data = append(data, map[string]clientJSON{
			idStr: cj,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Couldn't write response: %s", err)
	}
}

// findTemporary looks up the IP in temporary storages, like autohosts or
// blocklists.
func (clients *clientsContainer) findTemporary(ip net.IP, idStr string) (cj clientJSON, found bool) {
	if ip == nil {
		return cj, false
	}

	rc, ok := clients.FindRuntimeClient(idStr)
	if !ok {
		// It is still possible that the IP used to be in the runtime
		// clients list, but then the server was reloaded.  So, check
		// the DNS server's blocked IP list.
		//
		// See https://github.com/AdguardTeam/AdGuardHome/issues/2428.
		disallowed, rule := clients.dnsServer.IsBlockedIP(ip)
		if rule == "" {
			return clientJSON{}, false
		}

		cj = clientJSON{
			IDs:            []string{idStr},
			Disallowed:     disallowed,
			DisallowedRule: rule,
			WhoisInfo:      &RuntimeClientWhoisInfo{},
		}

		return cj, true
	}

	cj = runtimeClientToJSON(idStr, rc)
	cj.Disallowed, cj.DisallowedRule = clients.dnsServer.IsBlockedIP(ip)

	return cj, true
}

// RegisterClientsHandlers registers HTTP handlers
func (clients *clientsContainer) registerWebHandlers() {
	httpRegister(http.MethodGet, "/control/clients", clients.handleGetClients)
	httpRegister(http.MethodPost, "/control/clients/add", clients.handleAddClient)
	httpRegister(http.MethodPost, "/control/clients/delete", clients.handleDelClient)
	httpRegister(http.MethodPost, "/control/clients/update", clients.handleUpdateClient)
	httpRegister(http.MethodGet, "/control/clients/find", clients.handleFindClient)
}
