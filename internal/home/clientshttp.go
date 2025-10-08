package home

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"

	"github.com/AdguardTeam/AdGuardHome/internal/aghalg"
	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/client"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering/safesearch"
	"github.com/AdguardTeam/AdGuardHome/internal/schedule"
	"github.com/AdguardTeam/AdGuardHome/internal/whois"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
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

	// WHOIS is the filtered WHOIS data of a client.
	WHOIS          *whois.Info                 `json:"whois_info,omitempty"`
	SafeSearchConf *filtering.SafeSearchConfig `json:"safe_search"`

	// Schedule is blocked services schedule for every day of the week.
	Schedule *schedule.Weekly `json:"blocked_services_schedule"`

	Name string `json:"name"`

	// BlockedServices is the names of blocked services.
	BlockedServices []string `json:"blocked_services"`
	IDs             []string `json:"ids"`
	Tags            []string `json:"tags"`
	Upstreams       []string `json:"upstreams"`

	FilteringEnabled    bool `json:"filtering_enabled"`
	ParentalEnabled     bool `json:"parental_enabled"`
	SafeBrowsingEnabled bool `json:"safebrowsing_enabled"`
	// Deprecated: use safeSearchConf.
	SafeSearchEnabled        bool `json:"safesearch_enabled"`
	UseGlobalBlockedServices bool `json:"use_global_blocked_services"`
	UseGlobalSettings        bool `json:"use_global_settings"`

	IgnoreQueryLog   aghalg.NullBool `json:"ignore_querylog"`
	IgnoreStatistics aghalg.NullBool `json:"ignore_statistics"`

	UpstreamsCacheSize    uint32          `json:"upstreams_cache_size"`
	UpstreamsCacheEnabled aghalg.NullBool `json:"upstreams_cache_enabled"`
}

// runtimeClientJSON is a JSON representation of the [client.Runtime].
type runtimeClientJSON struct {
	WHOIS *whois.Info `json:"whois_info"`

	IP     netip.Addr    `json:"ip"`
	Name   string        `json:"name"`
	Source client.Source `json:"source"`
}

// clientListJSON contains lists of persistent clients, runtime clients and also
// supported tags.
type clientListJSON struct {
	Clients        []*clientJSON       `json:"clients"`
	RuntimeClients []runtimeClientJSON `json:"auto_clients"`
	Tags           []string            `json:"supported_tags"`
}

// whoisOrEmpty returns a WHOIS client information or a pointer to an empty
// struct.  Frontend expects a non-nil value.
func whoisOrEmpty(r *client.Runtime) (info *whois.Info) {
	info = r.WHOIS()
	if info != nil {
		return info
	}

	return &whois.Info{}
}

// handleGetClients is the handler for GET /control/clients HTTP API.
func (clients *clientsContainer) handleGetClients(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	data := clientListJSON{}

	clients.lock.Lock()
	defer clients.lock.Unlock()

	clients.storage.RangeByName(func(c *client.Persistent) (cont bool) {
		cj := clientToJSON(c)
		data.Clients = append(data.Clients, cj)

		return true
	})

	clients.storage.UpdateDHCP(ctx)

	clients.storage.RangeRuntime(func(rc *client.Runtime) (cont bool) {
		src, host := rc.Info()
		cj := runtimeClientJSON{
			WHOIS:  whoisOrEmpty(rc),
			Name:   host,
			Source: src,
			IP:     rc.Addr(),
		}

		data.RuntimeClients = append(data.RuntimeClients, cj)

		return true
	})

	data.Tags = clients.storage.AllowedTags()

	aghhttp.WriteJSONResponseOK(ctx, clients.logger, w, r, data)
}

// initPrev initializes the persistent client with the default or previous
// client properties.
func initPrev(cj clientJSON, prev *client.Persistent) (c *client.Persistent, err error) {
	var (
		uid              client.UID
		ignoreQueryLog   bool
		ignoreStatistics bool
		upsCacheEnabled  bool
		upsCacheSize     uint32
	)

	if prev != nil {
		uid = prev.UID
		ignoreQueryLog = prev.IgnoreQueryLog
		ignoreStatistics = prev.IgnoreStatistics
		upsCacheEnabled = prev.UpstreamsCacheEnabled
		upsCacheSize = prev.UpstreamsCacheSize
	}

	if cj.IgnoreQueryLog != aghalg.NBNull {
		ignoreQueryLog = cj.IgnoreQueryLog == aghalg.NBTrue
	}

	if cj.IgnoreStatistics != aghalg.NBNull {
		ignoreStatistics = cj.IgnoreStatistics == aghalg.NBTrue
	}

	if cj.UpstreamsCacheEnabled != aghalg.NBNull {
		upsCacheEnabled = cj.UpstreamsCacheEnabled == aghalg.NBTrue
		upsCacheSize = cj.UpstreamsCacheSize
	}

	svcs, err := copyBlockedServices(cj.Schedule, cj.BlockedServices, prev)
	if err != nil {
		return nil, fmt.Errorf("invalid blocked services: %w", err)
	}

	if (uid == client.UID{}) {
		uid, err = client.NewUID()
		if err != nil {
			return nil, fmt.Errorf("generating uid: %w", err)
		}
	}

	return &client.Persistent{
		BlockedServices:       svcs,
		UID:                   uid,
		IgnoreQueryLog:        ignoreQueryLog,
		IgnoreStatistics:      ignoreStatistics,
		UpstreamsCacheEnabled: upsCacheEnabled,
		UpstreamsCacheSize:    upsCacheSize,
	}, nil
}

// jsonToClient converts JSON object to persistent client object if there are no
// errors.
func (clients *clientsContainer) jsonToClient(
	ctx context.Context,
	cj clientJSON,
	prev *client.Persistent,
) (c *client.Persistent, err error) {
	c, err = initPrev(cj, prev)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return nil, err
	}

	err = c.SetIDs(cj.IDs)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return nil, err
	}

	c.SafeSearchConf = copySafeSearch(cj.SafeSearchConf, cj.SafeSearchEnabled)
	c.Name = cj.Name
	c.Tags = cj.Tags
	c.Upstreams = cj.Upstreams
	c.UseOwnSettings = !cj.UseGlobalSettings
	c.FilteringEnabled = cj.FilteringEnabled
	c.ParentalEnabled = cj.ParentalEnabled
	c.SafeBrowsingEnabled = cj.SafeBrowsingEnabled
	c.UseOwnBlockedServices = !cj.UseGlobalBlockedServices

	if c.SafeSearchConf.Enabled {
		logger := clients.baseLogger.With(
			slogutil.KeyPrefix, safesearch.LogPrefix,
			safesearch.LogKeyClient, c.Name,
		)
		var ss *safesearch.Default
		ss, err = safesearch.NewDefault(ctx, &safesearch.DefaultConfig{
			Logger:         logger,
			ServicesConfig: c.SafeSearchConf,
			ClientName:     c.Name,
			CacheSize:      clients.safeSearchCacheSize,
			CacheTTL:       clients.safeSearchCacheTTL,
		})
		if err != nil {
			return nil, fmt.Errorf("creating safesearch for client %q: %w", c.Name, err)
		}

		c.SafeSearch = ss
	}

	return c, nil
}

// copySafeSearch returns safe search config created from provided parameters.
func copySafeSearch(
	jsonConf *filtering.SafeSearchConfig,
	enabled bool,
) (conf filtering.SafeSearchConfig) {
	if jsonConf != nil {
		return *jsonConf
	}

	// TODO(d.kolyshev): Remove after cleaning the deprecated
	// [clientJSON.SafeSearchEnabled] field.
	conf = filtering.SafeSearchConfig{
		Enabled: enabled,
	}

	// Set default service flags for enabled safesearch.
	if conf.Enabled {
		conf.Bing = true
		conf.DuckDuckGo = true
		conf.Ecosia = true
		conf.Google = true
		conf.Pixabay = true
		conf.Yandex = true
		conf.YouTube = true
	}

	return conf
}

// copyBlockedServices converts a json blocked services to an internal blocked
// services.
func copyBlockedServices(
	sch *schedule.Weekly,
	svcStrs []string,
	prev *client.Persistent,
) (svcs *filtering.BlockedServices, err error) {
	var weekly *schedule.Weekly
	if sch != nil {
		weekly = sch.Clone()
	} else if prev != nil {
		weekly = prev.BlockedServices.Schedule.Clone()
	} else {
		weekly = schedule.EmptyWeekly()
	}

	svcs = &filtering.BlockedServices{
		Schedule: weekly,
		IDs:      svcStrs,
	}

	err = svcs.Validate()
	if err != nil {
		return nil, fmt.Errorf("validating blocked services: %w", err)
	}

	return svcs, nil
}

// clientToJSON converts persistent client object to JSON object.
func clientToJSON(c *client.Persistent) (cj *clientJSON) {
	// TODO(d.kolyshev): Remove after cleaning the deprecated
	// [clientJSON.SafeSearchEnabled] field.
	cloneVal := c.SafeSearchConf
	safeSearchConf := &cloneVal

	return &clientJSON{
		Name:                c.Name,
		IDs:                 c.Identifiers(),
		Tags:                c.Tags,
		UseGlobalSettings:   !c.UseOwnSettings,
		FilteringEnabled:    c.FilteringEnabled,
		ParentalEnabled:     c.ParentalEnabled,
		SafeSearchEnabled:   safeSearchConf.Enabled,
		SafeSearchConf:      safeSearchConf,
		SafeBrowsingEnabled: c.SafeBrowsingEnabled,

		UseGlobalBlockedServices: !c.UseOwnBlockedServices,

		Schedule:        c.BlockedServices.Schedule,
		BlockedServices: c.BlockedServices.IDs,

		Upstreams: c.Upstreams,

		IgnoreQueryLog:   aghalg.BoolToNullBool(c.IgnoreQueryLog),
		IgnoreStatistics: aghalg.BoolToNullBool(c.IgnoreStatistics),

		UpstreamsCacheSize:    c.UpstreamsCacheSize,
		UpstreamsCacheEnabled: aghalg.BoolToNullBool(c.UpstreamsCacheEnabled),
	}
}

// handleAddClient is the handler for POST /control/clients/add HTTP API.
func (clients *clientsContainer) handleAddClient(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := clients.logger

	cj := clientJSON{}
	err := json.NewDecoder(r.Body).Decode(&cj)
	if err != nil {
		aghhttp.ErrorAndLog(
			ctx,
			l,
			r,
			w,
			http.StatusBadRequest,
			"failed to process request body: %s",
			err,
		)

		return
	}

	c, err := clients.jsonToClient(ctx, cj, nil)
	if err != nil {
		aghhttp.ErrorAndLog(ctx, l, r, w, http.StatusBadRequest, "%s", err)

		return
	}

	err = clients.storage.Add(ctx, c)
	if err != nil {
		aghhttp.ErrorAndLog(ctx, l, r, w, http.StatusBadRequest, "%s", err)

		return
	}

	clients.confModifier.Apply(ctx)
}

// handleDelClient is the handler for POST /control/clients/delete HTTP API.
func (clients *clientsContainer) handleDelClient(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := clients.logger

	cj := clientJSON{}
	err := json.NewDecoder(r.Body).Decode(&cj)
	if err != nil {
		aghhttp.ErrorAndLog(
			ctx,
			l,
			r,
			w,
			http.StatusBadRequest,
			"failed to process request body: %s",
			err,
		)

		return
	}

	if len(cj.Name) == 0 {
		aghhttp.ErrorAndLog(ctx, l, r, w, http.StatusBadRequest, "client's name must be non-empty")

		return
	}

	if !clients.storage.RemoveByName(ctx, cj.Name) {
		aghhttp.ErrorAndLog(ctx, l, r, w, http.StatusBadRequest, "Client not found")

		return
	}

	clients.confModifier.Apply(ctx)
}

// updateJSON contains the name and data of the updated persistent client.
type updateJSON struct {
	Name string     `json:"name"`
	Data clientJSON `json:"data"`
}

// handleUpdateClient is the handler for POST /control/clients/update HTTP API.
//
// TODO(s.chzhen):  Accept updated parameters instead of whole structure.
func (clients *clientsContainer) handleUpdateClient(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := clients.logger

	dj := updateJSON{}
	err := json.NewDecoder(r.Body).Decode(&dj)
	if err != nil {
		aghhttp.ErrorAndLog(
			ctx,
			l,
			r,
			w,
			http.StatusBadRequest,
			"failed to process request body: %s",
			err,
		)

		return
	}

	if len(dj.Name) == 0 {
		aghhttp.ErrorAndLog(ctx, l, r, w, http.StatusBadRequest, "Invalid request")

		return
	}

	c, err := clients.jsonToClient(ctx, dj.Data, nil)
	if err != nil {
		aghhttp.ErrorAndLog(ctx, l, r, w, http.StatusBadRequest, "%s", err)

		return
	}

	err = clients.storage.Update(ctx, dj.Name, c)
	if err != nil {
		aghhttp.ErrorAndLog(ctx, l, r, w, http.StatusBadRequest, "%s", err)

		return
	}

	clients.confModifier.Apply(ctx)
}

// handleFindClient is the handler for GET /control/clients/find HTTP API.
//
// Deprecated:  Remove it when migration to the new API is over.
func (clients *clientsContainer) handleFindClient(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := clients.logger

	q := r.URL.Query()
	data := make([]map[string]*clientJSON, 0, len(q))
	params := &client.FindParams{}
	var err error

	for i := range len(q) {
		idStr := q.Get(fmt.Sprintf("ip%d", i))
		if idStr == "" {
			break
		}

		err = params.Set(idStr)
		if err != nil {
			l.DebugContext(ctx, "finding client", "id", idStr, slogutil.KeyError, err)

			continue
		}

		data = append(data, map[string]*clientJSON{
			idStr: clients.findClient(idStr, params),
		})
	}

	aghhttp.WriteJSONResponseOK(ctx, l, w, r, data)
}

// findClient returns available information about a client by params from the
// client's storage or access settings.  idStr is the string representation of
// typed params.  params must not be nil.  cj is guaranteed to be non-nil.
func (clients *clientsContainer) findClient(
	idStr string,
	params *client.FindParams,
) (cj *clientJSON) {
	c, ok := clients.storage.Find(params)
	if !ok {
		return clients.findRuntime(idStr, params)
	}

	cj = clientToJSON(c)
	disallowed, rule := clients.clientChecker.IsBlockedClient(
		params.RemoteIP,
		string(params.ClientID),
	)
	cj.Disallowed = &disallowed

	if disallowed && rule != "" {
		// Since "disallowed_rule" is omitted from JSON unless present, it
		// should only be set when the client is actually blocked.
		cj.DisallowedRule = &rule
	}

	return cj
}

// searchQueryJSON is a request to the POST /control/clients/search HTTP API.
//
// TODO(s.chzhen):  Add UIDs.
type searchQueryJSON struct {
	Clients []searchClientJSON `json:"clients"`
}

// searchClientJSON is a part of [searchQueryJSON] that contains a string
// representation of the client's IP address, CIDR, MAC address, or ClientID.
type searchClientJSON struct {
	ID string `json:"id"`
}

// handleSearchClient is the handler for the POST /control/clients/search HTTP
// API.
func (clients *clientsContainer) handleSearchClient(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	l := clients.logger

	q := searchQueryJSON{}
	err := json.NewDecoder(r.Body).Decode(&q)
	if err != nil {
		aghhttp.ErrorAndLog(
			ctx,
			l,
			r,
			w,
			http.StatusBadRequest,
			"failed to process request body: %s",
			err,
		)

		return
	}

	data := make([]map[string]*clientJSON, 0, len(q.Clients))
	params := &client.FindParams{}

	for _, c := range q.Clients {
		idStr := c.ID
		err = params.Set(idStr)
		if err != nil {
			l.DebugContext(ctx, "searching client", "id", idStr, slogutil.KeyError, err)

			continue
		}

		data = append(data, map[string]*clientJSON{
			idStr: clients.findClient(idStr, params),
		})
	}

	aghhttp.WriteJSONResponseOK(ctx, l, w, r, data)
}

// findRuntime looks up the IP in runtime and temporary storages, like
// /etc/hosts tables, DHCP leases, or blocklists.  params must not be nil.  cj
// is guaranteed to be non-nil.
func (clients *clientsContainer) findRuntime(
	idStr string,
	params *client.FindParams,
) (cj *clientJSON) {
	var host string
	whois := &whois.Info{}

	ip := params.RemoteIP
	rc := clients.storage.ClientRuntime(ip)
	if rc != nil {
		_, host = rc.Info()
		whois = whoisOrEmpty(rc)
	}

	// Check the DNS server's blocked IP list regardless of whether a runtime
	// client was found or not.  This is because it's still possible that the
	// runtime client associated with the IP address was stored previously, but
	// then the server was reloaded.
	//
	// See https://github.com/AdguardTeam/AdGuardHome/issues/2428.
	disallowed, rule := clients.clientChecker.IsBlockedClient(ip, string(params.ClientID))

	var disallowedRule *string
	if disallowed && rule != "" {
		// Since "disallowed_rule" is omitted from JSON unless present, it
		// should only be set when the client is actually blocked.
		disallowedRule = &rule
	}

	return &clientJSON{
		Name:           host,
		IDs:            []string{idStr},
		WHOIS:          whois,
		Disallowed:     &disallowed,
		DisallowedRule: disallowedRule,
	}
}

// registerWebHandlers registers HTTP handlers.
func (clients *clientsContainer) registerWebHandlers() {
	clients.httpReg.Register(http.MethodGet, "/control/clients", clients.handleGetClients)
	clients.httpReg.Register(http.MethodPost, "/control/clients/add", clients.handleAddClient)
	clients.httpReg.Register(http.MethodPost, "/control/clients/delete", clients.handleDelClient)
	clients.httpReg.Register(http.MethodPost, "/control/clients/update", clients.handleUpdateClient)
	clients.httpReg.Register(http.MethodPost, "/control/clients/search", clients.handleSearchClient)

	// Deprecated handler.
	clients.httpReg.Register(http.MethodGet, "/control/clients/find", clients.handleFindClient)
}
