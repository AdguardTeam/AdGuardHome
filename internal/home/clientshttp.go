package home

import (
	"encoding/json"
	"fmt"
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

	WhoisInfo map[string]interface{} `json:"whois_info"`

	// Disallowed - if true -- client's IP is not disallowed
	// Otherwise, it is blocked.
	Disallowed bool `json:"disallowed"`

	// DisallowedRule - the rule due to which the client is disallowed
	// If Disallowed is true, and this string is empty - it means that the client IP
	// is disallowed by the "allowed IP list", i.e. it is not included in allowed.
	DisallowedRule string `json:"disallowed_rule"`
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
	Tags        []string         `json:"supported_tags"`
}

// respond with information about configured clients
func (clients *clientsContainer) handleGetClients(w http.ResponseWriter, _ *http.Request) {
	data := clientListJSON{}

	clients.lock.Lock()
	for _, c := range clients.list {
		cj := clientToJSON(c)
		data.Clients = append(data.Clients, cj)
	}
	for ip, ch := range clients.ipHost {
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
	clients.lock.Unlock()

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
	}
	return cj
}

// Convert ClientHost object to JSON
func clientHostToJSON(ip string, ch ClientHost) clientJSON {
	cj := clientJSON{
		Name: ch.Host,
		IDs:  []string{ip},
	}

	cj.WhoisInfo = make(map[string]interface{})
	for _, wi := range ch.WhoisInfo {
		cj.WhoisInfo[wi[0]] = wi[1]
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
	ok, err := clients.Add(*c)
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
	err = clients.Update(dj.Name, *c)
	if err != nil {
		httpError(w, http.StatusBadRequest, "%s", err)
		return
	}

	onConfigModified()
}

// Get the list of clients by IP address list
func (clients *clientsContainer) handleFindClient(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	data := []map[string]interface{}{}
	for i := 0; ; i++ {
		ip := q.Get(fmt.Sprintf("ip%d", i))
		if len(ip) == 0 {
			break
		}

		el := map[string]interface{}{}
		c, ok := clients.Find(ip)
		var cj clientJSON
		if !ok {
			var found bool
			cj, found = clients.findTemporary(ip)
			if !found {
				continue
			}
		} else {
			cj = clientToJSON(&c)
			cj.Disallowed, cj.DisallowedRule = clients.dnsServer.IsBlockedIP(ip)
		}

		el[ip] = cj
		data = append(data, el)
	}

	js, err := json.Marshal(data)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "json.Marshal: %s", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(js)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Couldn't write response: %s", err)
	}
}

// findTemporary looks up the IP in temporary storages, like autohosts or
// blocklists.
func (clients *clientsContainer) findTemporary(ip string) (cj clientJSON, found bool) {
	ch, ok := clients.FindAutoClient(ip)
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
			IDs:            []string{ip},
			Disallowed:     disallowed,
			DisallowedRule: rule,
		}

		return cj, true
	}

	cj = clientHostToJSON(ip, ch)
	cj.Disallowed, cj.DisallowedRule = clients.dnsServer.IsBlockedIP(ip)

	return cj, true
}

// RegisterClientsHandlers registers HTTP handlers
func (clients *clientsContainer) registerWebHandlers() {
	httpRegister("GET", "/control/clients", clients.handleGetClients)
	httpRegister("POST", "/control/clients/add", clients.handleAddClient)
	httpRegister("POST", "/control/clients/delete", clients.handleDelClient)
	httpRegister("POST", "/control/clients/update", clients.handleUpdateClient)
	httpRegister("GET", "/control/clients/find", clients.handleFindClient)
}
