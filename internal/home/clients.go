package home

import (
	"context"
	"fmt"
	"log/slog"
	"net/netip"
	"slices"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/agh"
	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/arpdb"
	"github.com/AdguardTeam/AdGuardHome/internal/client"
	"github.com/AdguardTeam/AdGuardHome/internal/configmgr"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering/safesearch"
	"github.com/AdguardTeam/AdGuardHome/internal/querylog"
	"github.com/AdguardTeam/AdGuardHome/internal/schedule"
	"github.com/AdguardTeam/AdGuardHome/internal/whois"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/c2h5oh/datasize"
)

// clientsContainer is the storage of all runtime and persistent clients.
type clientsContainer struct {
	// baseLogger is used to create loggers with custom prefixes for safe search
	// filter.  It must not be nil.
	baseLogger *slog.Logger

	// logger is used for logging the operation of the client container.  It
	// must not be nil.
	logger *slog.Logger

	// storage stores information about persistent clients.
	storage *client.Storage

	// clientChecker checks if a client is blocked by the current access
	// settings.
	clientChecker BlockedClientChecker

	// confModifier is used to update the global configuration.  It must not be
	// nil.
	confModifier agh.ConfigModifier

	// httpReg registers HTTP handlers.  It must not be nil.
	httpReg aghhttp.Registrar

	// lock protects all fields.
	//
	// TODO(a.garipov): Use a pointer and describe which fields are protected in
	// more detail.  Use sync.RWMutex.
	lock sync.Mutex

	// safeSearchCacheSize is the size of the safe search cache to use for
	// persistent clients.
	safeSearchCacheSize uint

	// safeSearchCacheTTL is the TTL of the safe search cache to use for
	// persistent clients.
	safeSearchCacheTTL time.Duration
}

// BlockedClientChecker checks if a client is blocked by the current access
// settings.
type BlockedClientChecker interface {
	// TODO(s.chzhen):  Accept [client.FindParams].
	IsBlockedClient(ip netip.Addr, clientID string) (blocked bool, rule string)
}

// Init initializes the clients container.  All arguments must not be nil.
//
// NOTE:  This function must be called only once.
//
// TODO(s.chzhen):  Use a configuration structure.
func (clients *clientsContainer) Init(
	ctx context.Context,
	baseLogger *slog.Logger,
	conf *configmgr.ClientsConfig,
	dhcpServer client.DHCP,
	etcHosts *aghnet.HostsContainer,
	arpDB arpdb.Interface,
	filteringConf *filtering.Config,
	sigHdlr *signalHandler,
	confModifier agh.ConfigModifier,
	httpReg aghhttp.Registrar,
) (err error) {
	// TODO(s.chzhen):  Refactor it.
	if clients.storage != nil {
		return errors.Error("clients container already initialized")
	}

	clients.baseLogger = baseLogger
	clients.logger = baseLogger.With(slogutil.KeyPrefix, "client_container")
	clients.safeSearchCacheSize = filteringConf.SafeSearchCacheSize
	clients.safeSearchCacheTTL = time.Minute * time.Duration(filteringConf.CacheTime)
	clients.confModifier = confModifier
	clients.httpReg = httpReg

	confClients := make([]*client.Persistent, 0, len(conf.Persistent))
	for i, o := range conf.Persistent {
		if o == nil {
			return fmt.Errorf("init persistent client at index %d: %w", i, errors.ErrNoValue)
		}

		var p *client.Persistent
		p, err = newPersistentClient(
			ctx,
			o,
			baseLogger,
			clients.safeSearchCacheSize,
			clients.safeSearchCacheTTL,
		)
		if err != nil {
			return fmt.Errorf("init persistent client at index %d: %w", i, err)
		}

		confClients = append(confClients, p)
	}

	var hosts client.HostsContainer
	if conf.Sources.HostsFile && etcHosts != nil {
		// Prevent nil-pointer non-nil interface.
		hosts = etcHosts
	}

	clients.storage, err = client.NewStorage(ctx, &client.StorageConfig{
		BaseLogger:             baseLogger,
		Logger:                 baseLogger.With(slogutil.KeyPrefix, "client_storage"),
		Clock:                  timeutil.SystemClock{},
		InitialClients:         confClients,
		DHCP:                   dhcpServer,
		EtcHosts:               hosts,
		ARPDB:                  arpDB,
		ARPClientsUpdatePeriod: arpClientsUpdatePeriod,
		RuntimeSourceDHCP:      config.Clients.Sources.DHCP,
	})
	if err != nil {
		return fmt.Errorf("init client storage: %w", err)
	}

	sigHdlr.addClientStorage(clients.storage)

	filteringConf.ApplyClientFiltering = clients.storage.ApplyClientFiltering

	return nil
}

// webHandlersRegistered prevents a [clientsContainer] from registering its web
// handlers more than once.
//
// TODO(a.garipov): Refactor HTTP handler registration logic.
var webHandlersRegistered = false

// Start starts the clients container.
func (clients *clientsContainer) Start(ctx context.Context) (err error) {
	if !webHandlersRegistered {
		webHandlersRegistered = true
		clients.registerWebHandlers()
	}

	return clients.storage.Start(ctx)
}

// newPersistentClient returns an initialized persistent client if there are no
// errors.  baseLogger must not be nil, conf must be valid.
func newPersistentClient(
	ctx context.Context,
	conf *configmgr.Client,
	baseLogger *slog.Logger,
	safeSearchCacheSize uint,
	safeSearchCacheTTL time.Duration,
) (cli *client.Persistent, err error) {
	safeSearch := conf.SafeSearchConf
	if safeSearch == nil {
		safeSearch = &configmgr.SafeSearch{}
	}

	safeSearchConf := &filtering.SafeSearchConfig{
		Enabled:    safeSearch.Enabled,
		Bing:       safeSearch.Bing,
		DuckDuckGo: safeSearch.DuckDuckGo,
		Ecosia:     safeSearch.Ecosia,
		Google:     safeSearch.Google,
		Pixabay:    safeSearch.Pixabay,
		Yandex:     safeSearch.Yandex,
		YouTube:    safeSearch.YouTube,
	}

	cli = &client.Persistent{
		SafeSearchConf:        safeSearchConf,
		Name:                  conf.Name,
		Upstreams:             conf.Upstreams,
		UID:                   conf.UID,
		UseOwnSettings:        !conf.UseGlobalSettings,
		FilteringEnabled:      conf.FilteringEnabled,
		ParentalEnabled:       conf.ParentalEnabled,
		SafeBrowsingEnabled:   conf.SafeBrowsingEnabled,
		UseOwnBlockedServices: !conf.UseGlobalBlockedServices,
		IgnoreQueryLog:        conf.IgnoreQueryLog,
		IgnoreStatistics:      conf.IgnoreStatistics,
		UpstreamsCacheEnabled: conf.UpstreamsCacheEnabled,
		UpstreamsCacheSize:    uint32(conf.UpstreamsCacheSize),
	}

	err = cli.SetIDs(conf.IDs)
	if err != nil {
		return nil, fmt.Errorf("parsing ids: %w", err)
	}

	if (cli.UID == client.UID{}) {
		cli.UID = client.NewUID()
	}

	if safeSearch.Enabled {
		ss, sErr := newSafeSearch(
			ctx,
			baseLogger,
			cli.Name,
			safeSearchConf,
			safeSearchCacheSize,
			safeSearchCacheTTL,
		)
		if sErr != nil {
			return nil, fmt.Errorf("init safesearch %q: %w", cli.Name, sErr)
		}

		cli.SafeSearch = ss
	}

	bs, err := newBlockedServices(ctx, baseLogger, conf.BlockedServices)
	if err != nil {
		return nil, fmt.Errorf("init blocked services %q: %w", cli.Name, err)
	}

	cli.BlockedServices = bs

	cli.Tags = slices.Clone(conf.Tags)

	return cli, nil
}

// newSafeSearch returns an initialized safe search filter if there are no
// errors.  baseLogger and conf must not be nil.
func newSafeSearch(
	ctx context.Context,
	baseLogger *slog.Logger,
	cliName string,
	conf *filtering.SafeSearchConfig,
	cacheSize uint,
	cacheTTL time.Duration,
) (ss *safesearch.Default, err error) {
	logger := baseLogger.With(
		slogutil.KeyPrefix, safesearch.LogPrefix,
		safesearch.LogKeyClient, cliName,
	)

	return safesearch.NewDefault(ctx, &safesearch.DefaultConfig{
		Logger:         logger,
		ServicesConfig: conf,
		ClientName:     cliName,
		CacheSize:      cacheSize,
		CacheTTL:       cacheTTL,
	})
}

// newBlockedServices returns an initialized blocked services configuration if
// there are no errors.  baseLogger must not be nil.
func newBlockedServices(
	ctx context.Context,
	baseLogger *slog.Logger,
	conf *configmgr.BlockedServices,
) (bs *filtering.BlockedServices, err error) {
	bs = &filtering.BlockedServices{}
	if conf != nil {
		bs.IDs = conf.IDs
		bs.Schedule = conf.Schedule
	}

	if bs.Schedule == nil {
		bs.Schedule = schedule.EmptyWeekly()
	}

	bs.FilterUnknownIDs(ctx, baseLogger)
	err = bs.Validate()
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return nil, err
	}

	return bs.Clone(), nil
}

// WriteDiskConfig writes the clients configuration to the given c.  c must not
// be nil.
func (clients *clientsContainer) WriteDiskConfig(c *configmgr.ClientsConfig) {
	clients.lock.Lock()
	defer clients.lock.Unlock()

	objs := make([]*configmgr.Client, 0, clients.storage.Size())
	clients.storage.RangeByName(func(cli *client.Persistent) (cont bool) {
		var bs *configmgr.BlockedServices
		if cli.BlockedServices != nil {
			bs = &configmgr.BlockedServices{
				Schedule: cli.BlockedServices.Schedule.Clone(),
				IDs:      slices.Clone(cli.BlockedServices.IDs),
			}
		}

		objs = append(objs, &configmgr.Client{
			BlockedServices: bs,
			SafeSearchConf: &configmgr.SafeSearch{
				Enabled:    cli.SafeSearchConf.Enabled,
				Bing:       cli.SafeSearchConf.Bing,
				DuckDuckGo: cli.SafeSearchConf.DuckDuckGo,
				Ecosia:     cli.SafeSearchConf.Ecosia,
				Google:     cli.SafeSearchConf.Google,
				Pixabay:    cli.SafeSearchConf.Pixabay,
				Yandex:     cli.SafeSearchConf.Yandex,
				YouTube:    cli.SafeSearchConf.YouTube,
			},

			Name: cli.Name,

			IDs:       cli.Identifiers(),
			Tags:      slices.Clone(cli.Tags),
			Upstreams: slices.Clone(cli.Upstreams),

			UID: cli.UID,

			UpstreamsCacheSize: datasize.ByteSize(cli.UpstreamsCacheSize),

			FilteringEnabled:         cli.FilteringEnabled,
			ParentalEnabled:          cli.ParentalEnabled,
			SafeBrowsingEnabled:      cli.SafeBrowsingEnabled,
			UseGlobalBlockedServices: !cli.UseOwnBlockedServices,
			IgnoreQueryLog:           cli.IgnoreQueryLog,
			IgnoreStatistics:         cli.IgnoreStatistics,
			UpstreamsCacheEnabled:    cli.UpstreamsCacheEnabled,
			UseGlobalSettings:        !cli.UseOwnSettings,
		})

		return true
	})

	c.Persistent = objs
}

// arpClientsUpdatePeriod defines how often ARP clients are updated.
const arpClientsUpdatePeriod = 10 * time.Minute

// findMultiple is a wrapper around [clientsContainer.find] to make it a valid
// client finder for the query log.  c is never nil; if no information about the
// client is found, it returns an artificial client record by only setting the
// blocking-related fields.  err is always nil.
func (clients *clientsContainer) findMultiple(ids []string) (c *querylog.Client, err error) {
	var artClient *querylog.Client
	var art bool
	for _, id := range ids {
		ip, _ := netip.ParseAddr(id)
		c, art = clients.clientOrArtificial(ip, id)
		if art {
			artClient = c

			continue
		}

		return c, nil
	}

	return artClient, nil
}

// clientOrArtificial returns information about one client.  If art is true,
// this is an artificial client record, meaning that we currently don't have any
// records about this client besides maybe whether or not it is blocked.  c is
// never nil.
func (clients *clientsContainer) clientOrArtificial(
	ip netip.Addr,
	id string,
) (c *querylog.Client, art bool) {
	defer func() {
		c.Disallowed, c.DisallowedRule = clients.clientChecker.IsBlockedClient(ip, id)
		if c.WHOIS == nil {
			c.WHOIS = &whois.Info{}
		}
	}()

	cli, ok := clients.storage.FindLoose(ip, id)
	if ok {
		return &querylog.Client{
			Name:           cli.Name,
			IgnoreQueryLog: cli.IgnoreQueryLog,
		}, false
	}

	rc := clients.storage.ClientRuntime(ip)
	if rc != nil {
		_, host := rc.Info()

		return &querylog.Client{
			Name:  host,
			WHOIS: rc.WHOIS(),
		}, false
	}

	return &querylog.Client{
		Name: "",
	}, true
}

// shouldCountClient is a wrapper around [client.Storage.Find] to make it a
// valid client information finder for the statistics.  If no information about
// the client is found, it returns true.  Values of ids must be either a valid
// ClientID or a valid IP address.
//
// TODO(s.chzhen):  Accept [client.FindParams].
func (clients *clientsContainer) shouldCountClient(ids []string) (y bool) {
	clients.lock.Lock()
	defer clients.lock.Unlock()

	params := &client.FindParams{}
	for _, id := range ids {
		err := params.Set(id)
		if err != nil {
			// Should not happen.
			clients.logger.Warn("parsing find params", slogutil.KeyError, err)

			continue
		}

		client, ok := clients.storage.Find(params)
		if ok {
			return !client.IgnoreStatistics
		}
	}

	return true
}

// type check
var _ client.AddressUpdater = (*clientsContainer)(nil)

// UpdateAddress implements the [client.AddressUpdater] interface for
// *clientsContainer
func (clients *clientsContainer) UpdateAddress(
	ctx context.Context,
	ip netip.Addr,
	host string,
	info *whois.Info,
) {
	clients.storage.UpdateAddress(ctx, ip, host, info)
}

// close gracefully closes all the client-specific upstream configurations of
// the persistent clients.
func (clients *clientsContainer) close(ctx context.Context) (err error) {
	return clients.storage.Shutdown(ctx)
}
