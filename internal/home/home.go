// Package home contains AdGuard Home's HTTP API methods.
package home

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/agh"
	"github.com/AdguardTeam/AdGuardHome/internal/aghalg"
	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/AdGuardHome/internal/aghslog"
	"github.com/AdguardTeam/AdGuardHome/internal/aghtls"
	"github.com/AdguardTeam/AdGuardHome/internal/arpdb"
	"github.com/AdguardTeam/AdGuardHome/internal/dhcpd"
	"github.com/AdguardTeam/AdGuardHome/internal/dnsforward"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering/hashprefix"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering/safesearch"
	"github.com/AdguardTeam/AdGuardHome/internal/permcheck"
	"github.com/AdguardTeam/AdGuardHome/internal/querylog"
	"github.com/AdguardTeam/AdGuardHome/internal/stats"
	"github.com/AdguardTeam/AdGuardHome/internal/updater"
	"github.com/AdguardTeam/AdGuardHome/internal/version"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/hostsfile"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/netutil/urlutil"
	"github.com/AdguardTeam/golibs/osutil"
	"github.com/AdguardTeam/golibs/osutil/executil"
)

// Global context
type homeContext struct {
	// Modules
	// --

	clients    clientsContainer   // per-client-settings module
	stats      stats.Interface    // statistics module
	queryLog   querylog.QueryLog  // query log module
	dnsServer  *dnsforward.Server // DNS module
	dhcpServer dhcpd.Interface    // DHCP module

	filters *filtering.DNSFilter // DNS filtering module
	web     *webAPI              // Web (HTTP, HTTPS) module

	// etcHosts contains IP-hostname mappings taken from the OS-specific hosts
	// configuration files, for example /etc/hosts.
	etcHosts *aghnet.HostsContainer

	// Runtime properties
	// --

	pidFileName string // PID file name.  Empty if no PID file was created.
	controlLock sync.Mutex
}

// globalContext is a global context object.
//
// TODO(a.garipov): Refactor.
var globalContext homeContext

// Main is the entry point
func Main(clientBuildFS fs.FS) {
	ctx := context.Background()

	initCmdLineOpts()

	// The configuration file path can be overridden, but other command-line
	// options have to override config values.  Therefore, do it manually
	// instead of using package flag.
	//
	// TODO(a.garipov): The comment above is most likely false.  Replace with
	// package flag.
	opts := loadCmdLineOpts()

	// TODO(s.chzhen):  Construct logger from command-line options.
	l := slog.Default()
	workDir, err := initWorkingDir(opts)
	if err != nil {
		l.ErrorContext(ctx, "failed to init working directory", slogutil.KeyError, err)

		os.Exit(osutil.ExitCodeFailure)
	}

	confPath := initConfigFilename(ctx, l, opts, workDir)

	ls := getLogSettings(ctx, l, opts, workDir, confPath)

	// TODO(a.garipov): Use slog everywhere.
	baseLogger := newSlogLogger(ls)

	done := make(chan struct{})

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)

	sigHdlrLogger := baseLogger.With(slogutil.KeyPrefix, "signalhdlr")
	sigHdlr := newSignalHandler(sigHdlrLogger, signals, func(ctx context.Context) {
		cleanup(ctx)
		cleanupAlways()
		close(done)
	})

	go sigHdlr.handle(ctx)

	if opts.serviceControlAction != "" {
		svcLogger := baseLogger.With(slogutil.KeyPrefix, "service")
		err = handleServiceControlAction(
			ctx,
			baseLogger,
			svcLogger,
			opts,
			clientBuildFS,
			signals,
			done,
			sigHdlr,
			workDir,
			confPath,
		)
		if err != nil {
			svcLogger.ErrorContext(ctx, "action failed", slogutil.KeyError, err)
			os.Exit(osutil.ExitCodeFailure)
		}

		return
	}

	// run the protection
	run(ctx, baseLogger, opts, clientBuildFS, done, sigHdlr, workDir, confPath)
}

// setupContext initializes [globalContext] fields.  It also reads and upgrades
// config file if necessary.  baseLogger must not be nil.
func setupContext(
	ctx context.Context,
	baseLogger *slog.Logger,
	opts options,
	workDir string,
	confPath string,
	isFirstRun bool,
) (err error) {
	if !opts.noEtcHosts {
		err = setupHostsContainer(ctx, baseLogger)
		if err != nil {
			// Don't wrap the error, because it's informative enough as is.
			return err
		}
	}

	if isFirstRun {
		baseLogger.InfoContext(ctx, "this is the first time adguard home has been launched")
		checkNetworkPermissions(ctx, baseLogger)

		return nil
	}

	// TODO(s.chzhen):  Consider adding a key prefix.
	err = parseConfig(ctx, baseLogger, workDir, confPath)
	if err != nil {
		baseLogger.ErrorContext(ctx, "failed to parse configuration file", slogutil.KeyError, err)

		os.Exit(osutil.ExitCodeFailure)
	}

	if opts.checkConfig {
		baseLogger.InfoContext(ctx, "configuration file is ok")

		os.Exit(osutil.ExitCodeSuccess)
	}

	return nil
}

// logIfUnsupported logs a formatted warning if the error is one of the
// unsupported errors and returns nil.  If err is nil, logIfUnsupported returns
// nil.  Otherwise, it returns err.
func logIfUnsupported(msg string, err error) (outErr error) {
	if errors.Is(err, errors.ErrUnsupported) {
		log.Debug(msg, err)

		return nil
	}

	return err
}

// configureOS sets the OS-related configuration.
func configureOS(conf *configuration) (err error) {
	osConf := conf.OSConfig
	if osConf == nil {
		return nil
	}

	if osConf.Group != "" {
		err = aghos.SetGroup(osConf.Group)
		err = logIfUnsupported("warning: setting group", err)
		if err != nil {
			return fmt.Errorf("setting group: %w", err)
		}

		log.Info("group set to %s", osConf.Group)
	}

	if osConf.User != "" {
		err = aghos.SetUser(osConf.User)
		err = logIfUnsupported("warning: setting user", err)
		if err != nil {
			return fmt.Errorf("setting user: %w", err)
		}

		log.Info("user set to %s", osConf.User)
	}

	if osConf.RlimitNoFile != 0 {
		err = aghos.SetRlimit(osConf.RlimitNoFile)
		err = logIfUnsupported("warning: setting rlimit", err)
		if err != nil {
			return fmt.Errorf("setting rlimit: %w", err)
		}

		log.Info("rlimit_nofile set to %d", osConf.RlimitNoFile)
	}

	return nil
}

// setupHostsContainer initializes the structures to keep up-to-date the hosts
// provided by the OS.  baseLogger must not be nil.
func setupHostsContainer(ctx context.Context, baseLogger *slog.Logger) (err error) {
	l := baseLogger.With(slogutil.KeyPrefix, "hosts")

	var hostsWatcher aghos.FSWatcher
	hostsWatcher, err = aghos.NewOSWatcher(&aghos.OSWatcherConfig{
		Logger: baseLogger.With(slogutil.KeyPrefix, "hosts_watcher"),
	})
	if err != nil {
		l.WarnContext(
			ctx,
			"initializing filesystem watcher; not watching for changes",
			slogutil.KeyError,
			err,
		)

		hostsWatcher = aghos.EmptyFSWatcher{}
	}

	paths, err := hostsfile.DefaultHostsPaths()
	if err != nil {
		return fmt.Errorf("getting default system hosts paths: %w", err)
	}

	globalContext.etcHosts, err = aghnet.NewHostsContainer(
		ctx,
		l,
		osutil.RootDirFS(),
		hostsWatcher,
		paths...,
	)
	if err != nil {
		closeErr := hostsWatcher.Shutdown(ctx)
		if errors.Is(err, aghnet.ErrNoHostsPaths) {
			l.WarnContext(ctx, "initializing hosts container", slogutil.KeyError, err)

			return closeErr
		}

		return errors.Join(fmt.Errorf("initializing hosts container: %w", err), closeErr)
	}

	return hostsWatcher.Start(ctx)
}

// setupOpts sets up command-line options.
func setupOpts(opts options) (err error) {
	err = setupBindOpts(opts)
	if err != nil {
		// Don't wrap the error, because it's informative enough as is.
		return err
	}

	if len(opts.pidFile) != 0 && writePIDFile(opts.pidFile) {
		globalContext.pidFileName = opts.pidFile
	}

	return nil
}

// initContextClients initializes Context clients and related fields.  All
// arguments must not be nil.
func initContextClients(
	ctx context.Context,
	logger *slog.Logger,
	sigHdlr *signalHandler,
	confModifier agh.ConfigModifier,
	httpReg aghhttp.Registrar,
	workDir string,
) (err error) {
	//lint:ignore SA1019 Migration is not over.
	config.DHCP.WorkDir = workDir
	config.DHCP.DataDir = filepath.Join(workDir, dataDir)
	config.DHCP.HTTPReg = httpReg
	config.DHCP.CommandConstructor = executil.SystemCommandConstructor{}
	config.DHCP.Logger = logger.With(slogutil.KeyPrefix, "dhcpd")
	config.DHCP.ConfModifier = confModifier

	globalContext.dhcpServer, err = dhcpd.Create(ctx, config.DHCP)
	if globalContext.dhcpServer == nil || err != nil {
		// TODO(a.garipov): There are a lot of places in the code right
		// now which assume that the DHCP server can be nil despite this
		// condition.  Inspect them and perhaps rewrite them to use
		// Enabled() instead.
		return fmt.Errorf("initing dhcp: %w", err)
	}

	var arpDB arpdb.Interface
	if config.Clients.Sources.ARP {
		arpDB = arpdb.New(logger.With(slogutil.KeyError, "arpdb"))
	}

	return globalContext.clients.Init(
		ctx,
		logger,
		config.Clients.Persistent,
		globalContext.dhcpServer,
		globalContext.etcHosts,
		arpDB,
		config.Filtering,
		sigHdlr,
		confModifier,
		httpReg,
	)
}

// setupBindOpts overrides bind host/port from the opts.
func setupBindOpts(opts options) (err error) {
	bindAddr := opts.bindAddr
	if bindAddr != (netip.AddrPort{}) {
		config.HTTPConfig.Address = bindAddr

		if config.HTTPConfig.Address.Port() != 0 {
			err = checkPorts()
			if err != nil {
				// Don't wrap the error, because it's informative enough as is.
				return err
			}
		}

		return nil
	}

	if opts.bindPort != 0 {
		config.HTTPConfig.Address = netip.AddrPortFrom(
			config.HTTPConfig.Address.Addr(),
			opts.bindPort,
		)

		err = checkPorts()
		if err != nil {
			// Don't wrap the error, because it's informative enough as is.
			return err
		}
	}

	if opts.bindHost.IsValid() {
		config.HTTPConfig.Address = netip.AddrPortFrom(
			opts.bindHost,
			config.HTTPConfig.Address.Port(),
		)
	}

	return nil
}

// setupDNSFilteringConf sets up DNS filtering configuration settings.  All
// arguments must not be nil.
func setupDNSFilteringConf(
	ctx context.Context,
	baseLogger *slog.Logger,
	conf *filtering.Config,
	tlsMgr *tlsManager,
	confModifier agh.ConfigModifier,
	httpReg aghhttp.Registrar,
	workDir string,
) (err error) {
	const (
		dnsTimeout = 3 * time.Second

		sbService                 = "safe_browsing"
		defaultSafeBrowsingServer = `https://family.adguard-dns.com/dns-query`
		sbTXTSuffix               = `sb.dns.adguard.com.`

		pcService             = "parental_control"
		defaultParentalServer = `https://family.adguard-dns.com/dns-query`
		pcTXTSuffix           = `pc.dns.adguard.com.`
	)

	conf.Logger = baseLogger.With(slogutil.KeyPrefix, "filtering")

	conf.EtcHosts = globalContext.etcHosts
	// TODO(s.chzhen):  Use empty interface.
	if globalContext.etcHosts == nil || !config.DNS.HostsFileEnabled {
		conf.EtcHosts = nil
	}

	conf.ConfModifier = confModifier
	conf.HTTPReg = httpReg
	conf.DataDir = filepath.Join(workDir, dataDir)
	conf.Filters = slices.Clone(config.Filters)
	conf.WhitelistFilters = slices.Clone(config.WhitelistFilters)
	conf.UserRules = slices.Clone(config.UserRules)
	conf.HTTPClient = httpClient(tlsMgr)

	cacheTime := time.Duration(conf.CacheTime) * time.Minute

	upsOpts := &upstream.Options{
		Logger:  aghslog.NewForUpstream(baseLogger, aghslog.UpstreamTypeService),
		Timeout: dnsTimeout,
		Bootstrap: upstream.StaticResolver{
			// 94.140.14.15.
			netip.AddrFrom4([4]byte{94, 140, 14, 15}),
			// 94.140.14.16.
			netip.AddrFrom4([4]byte{94, 140, 14, 16}),
			// 2a10:50c0::bad1:ff.
			netip.AddrFrom16([16]byte{42, 16, 80, 192, 12: 186, 209, 0, 255}),
			// 2a10:50c0::bad2:ff.
			netip.AddrFrom16([16]byte{42, 16, 80, 192, 12: 186, 210, 0, 255}),
		},
	}

	sbUps, err := upstream.AddressToUpstream(defaultSafeBrowsingServer, upsOpts)
	if err != nil {
		return fmt.Errorf("converting safe browsing server: %w", err)
	}

	conf.SafeBrowsingChecker = hashprefix.New(&hashprefix.Config{
		Logger:    baseLogger.With(slogutil.KeyPrefix, sbService),
		Upstream:  sbUps,
		TXTSuffix: sbTXTSuffix,
		CacheTime: cacheTime,
		CacheSize: conf.SafeBrowsingCacheSize,
	})

	// Protect against invalid configuration, see #6181.
	//
	// TODO(a.garipov): Validate against an empty host instead of setting it to
	// default.
	if conf.SafeBrowsingBlockHost == "" {
		host := defaultSafeBrowsingBlockHost
		baseLogger.WarnContext(ctx,
			"empty blocking host; set default",
			"service", sbService,
			"host", host,
		)

		conf.SafeBrowsingBlockHost = host
	}

	parUps, err := upstream.AddressToUpstream(defaultParentalServer, upsOpts)
	if err != nil {
		return fmt.Errorf("converting parental server: %w", err)
	}

	conf.ParentalControlChecker = hashprefix.New(&hashprefix.Config{
		Logger:    baseLogger.With(slogutil.KeyPrefix, pcService),
		Upstream:  parUps,
		TXTSuffix: pcTXTSuffix,
		CacheTime: cacheTime,
		CacheSize: conf.ParentalCacheSize,
	})

	// Protect against invalid configuration, see #6181.
	//
	// TODO(a.garipov): Validate against an empty host instead of setting it to
	// default.
	if conf.ParentalBlockHost == "" {
		host := defaultParentalBlockHost
		baseLogger.WarnContext(ctx,
			"empty blocking host; set default",
			"service", pcService,
			"host", host,
		)

		conf.ParentalBlockHost = host
	}

	logger := baseLogger.With(slogutil.KeyPrefix, safesearch.LogPrefix)
	conf.SafeSearch, err = safesearch.NewDefault(ctx, &safesearch.DefaultConfig{
		Logger:         logger,
		ServicesConfig: conf.SafeSearchConf,
		CacheSize:      conf.SafeSearchCacheSize,
		CacheTTL:       cacheTime,
	})
	if err != nil {
		return fmt.Errorf("initializing safesearch: %w", err)
	}

	return nil
}

// checkPorts is a helper for ports validation in config.
func checkPorts() (err error) {
	tcpPorts := aghalg.UniqChecker[tcpPort]{}
	addPorts(tcpPorts, tcpPort(config.HTTPConfig.Address.Port()))

	udpPorts := aghalg.UniqChecker[udpPort]{}
	addPorts(udpPorts, udpPort(config.DNS.Port))

	if config.TLS.Enabled {
		addPorts(
			tcpPorts,
			tcpPort(config.TLS.PortHTTPS),
			tcpPort(config.TLS.PortDNSOverTLS),
			tcpPort(config.TLS.PortDNSCrypt),
		)

		addPorts(udpPorts, udpPort(config.TLS.PortDNSOverQUIC))
	}

	if err = tcpPorts.Validate(); err != nil {
		return fmt.Errorf("validating tcp ports: %w", err)
	} else if err = udpPorts.Validate(); err != nil {
		return fmt.Errorf("validating udp ports: %w", err)
	}

	return nil
}

// isUpdateEnabled returns true if the update is enabled for current
// configuration.  It also logs the decision.  isCustomURL should be true if the
// updater is using a custom URL.
func isUpdateEnabled(
	ctx context.Context,
	l *slog.Logger,
	opts *options,
	isCustomURL bool,
) (ok bool) {
	if opts.disableUpdate {
		l.DebugContext(ctx, "updates are disabled by command-line option")

		return false
	}

	switch version.Channel() {
	case
		version.ChannelDevelopment,
		version.ChannelCandidate:
		if isCustomURL {
			l.DebugContext(ctx, "updates are enabled because custom url is used")
		} else {
			l.DebugContext(ctx, "updates are disabled for development and candidate builds")
		}

		return isCustomURL
	default:
		l.DebugContext(ctx, "updates are enabled")

		return true
	}
}

// webConfig is a configuration structure for webAPI.
type webConfig struct {
	// opts are used to determine if update is enabled.
	opts options

	// clientBuildFS is used for initializing client FS.  If opts.localFrontend
	// is false, then this field must not be nil.
	clientBuildFS fs.FS

	// updater is used for handling updates.  It must not be nil.
	updater *updater.Updater

	// baseLogger is used for logging init process and for logging inside web
	// api.  It must not be nil.
	baseLogger *slog.Logger

	// tlsManager contains the current configuration and state of TLS
	// encryption. It must not be nil.
	tlsManager *tlsManager

	// auth stores web user information and handles authentication.  It must not
	// be nil.
	auth *auth

	// mux is the default *http.ServeMux, the same as [globalContext.mux]. It
	// must not be nil.
	mux *http.ServeMux

	// configModifier is used to update the global configuration.
	configModifier agh.ConfigModifier

	// httpReg registers HTTP handlers. It must not be nil.
	httpReg aghhttp.Registrar

	// workDir is a base working directory.
	workDir string

	// confPath is a config path.
	confPath string

	// isCustomUpdURL defines if updater should use custom url.
	isCustomUpdURL bool

	// isFirstRun defines if current run is the first run.
	isFirstRun bool
}

// newWeb initializes the web module.  conf must not be nil.
func newWeb(ctx context.Context, conf *webConfig) (web *webAPI, err error) {
	logger := conf.baseLogger.With(slogutil.KeyPrefix, "webapi")

	webPort := suggestedWebPort(ctx, logger)

	var clientFS fs.FS
	if conf.opts.localFrontend {
		logger.WarnContext(ctx, "using local frontend files")

		clientFS = os.DirFS("build/static")
	} else {
		clientFS, err = fs.Sub(conf.clientBuildFS, "build/static")
		if err != nil {
			return nil, fmt.Errorf("getting embedded client subdir: %w", err)
		}
	}

	disableUpdate := !isUpdateEnabled(ctx, conf.baseLogger, &conf.opts, conf.isCustomUpdURL)

	webConf := &webAPIConfig{
		CommandConstructor: executil.SystemCommandConstructor{},
		updater:            conf.updater,
		logger:             logger,
		baseLogger:         conf.baseLogger,
		confModifier:       conf.configModifier,
		httpReg:            conf.httpReg,
		tlsManager:         conf.tlsManager,
		auth:               conf.auth,
		mux:                conf.mux,

		clientFS: clientFS,

		BindAddr: config.HTTPConfig.Address,

		workDir:  conf.workDir,
		confPath: conf.confPath,

		ReadTimeout:       readTimeout,
		ReadHeaderTimeout: readHdrTimeout,
		WriteTimeout:      writeTimeout,

		defaultWebPort: webPort,

		firstRun:         conf.isFirstRun,
		disableUpdate:    disableUpdate,
		runningAsService: conf.opts.runningAsService,
		serveHTTP3:       config.DNS.ServeHTTP3,
	}

	web = newWebAPI(ctx, webConf)
	if web == nil {
		return nil, errors.Error("can not initialize web")
	}

	return web, nil
}

// suggestedWebPort returns the suggested default HTTP port for the installation
// wizard, using the port provided via an environment variable.  It falls back
// to [defaultPortHTTP] on error.
func suggestedWebPort(ctx context.Context, l *slog.Logger) (p uint16) {
	const webPortEnv = "ADGUARD_HOME_DEFAULT_WEB_PORT"

	s := os.Getenv(webPortEnv)
	if s == "" {
		return defaultPortHTTP
	}

	v, err := strconv.ParseUint(s, 10, 16)
	if err == nil && v == 0 {
		err = errors.ErrOutOfRange
	}

	if err != nil {
		l.WarnContext(
			ctx,
			"invalid web port; using default",
			"env", webPortEnv,
			"val", s,
			slogutil.KeyError, err,
		)

		return defaultPortHTTP
	}

	return uint16(v)
}

func fatalOnError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// run configures and starts AdGuard Home.
//
// TODO(e.burkov):  Make opts a pointer.
func run(
	ctx context.Context,
	baseLogger *slog.Logger,
	opts options,
	clientBuildFS fs.FS,
	done chan struct{},
	sigHdlr *signalHandler,
	workDir string,
	confPath string,
) {
	initEnvironment(ctx, opts, baseLogger, workDir, confPath)

	isFirstRun := detectFirstRun(ctx, baseLogger, workDir, confPath)

	mw := &webMw{}
	mux := http.NewServeMux()
	httpReg := aghhttp.NewDefaultRegistrar(mux, mw.wrap)

	err := setupContext(ctx, baseLogger, opts, workDir, confPath, isFirstRun)
	fatalOnError(err)

	err = configureOS(config)
	fatalOnError(err)

	// Clients package uses filtering package's static data
	// (filtering.BlockedSvcKnown()), so we have to initialize filtering static
	// data first, but also to avoid relying on automatic Go init() function.
	filtering.InitModule(ctx, baseLogger)

	confModifier := newDefaultConfigModifier(
		config,
		baseLogger.With(slogutil.KeyPrefix, "config_modifier"),
		workDir,
		confPath,
	)

	err = initContextClients(ctx, baseLogger, sigHdlr, confModifier, httpReg, workDir)
	fatalOnError(err)

	tlsMgr, err := initTLS(ctx, baseLogger, sigHdlr, confModifier, httpReg)
	fatalOnError(err)

	err = setupDNSFilteringConf(
		ctx,
		baseLogger,
		config.Filtering,
		tlsMgr,
		confModifier,
		httpReg,
		workDir,
	)
	fatalOnError(err)

	err = setupOpts(opts)
	fatalOnError(err)

	upd, isCustomURL := initUpdate(ctx, baseLogger, opts, tlsMgr, isFirstRun, workDir, confPath)

	dataDirPath := filepath.Join(workDir, dataDir)
	err = os.MkdirAll(dataDirPath, aghos.DefaultPermDir)
	fatalOnError(errors.Annotate(err, "creating DNS data dir at %s: %w", dataDirPath))

	auth, err := initUsers(ctx, baseLogger, workDir, opts.glinetMode)
	fatalOnError(err)

	confModifier.setAuth(auth)

	conf := &webConfig{
		clientBuildFS:  clientBuildFS,
		updater:        upd,
		opts:           opts,
		baseLogger:     baseLogger,
		tlsManager:     tlsMgr,
		auth:           auth,
		mux:            mux,
		configModifier: confModifier,
		httpReg:        httpReg,
		workDir:        workDir,
		confPath:       confPath,
		isCustomUpdURL: isCustomURL,
		isFirstRun:     isFirstRun,
	}

	web, err := newWeb(ctx, conf)
	fatalOnError(err)

	mw.set(web)

	globalContext.web = web

	tlsMgr.setWebAPI(web)

	statsDir, querylogDir, err := checkStatsAndQuerylogDirs(config, workDir)
	fatalOnError(err)

	if !isFirstRun {
		runDNSServer(ctx, baseLogger, tlsMgr, confModifier, statsDir, querylogDir, httpReg)
	}

	if !opts.noPermCheck {
		checkPermissions(ctx, baseLogger, workDir, confPath, dataDirPath, statsDir, querylogDir)
	}

	web.start(ctx)

	// Wait for other goroutines to complete their job.
	<-done
}

// runDNSServer initializes and starts DNS and DHCP servers if this is not the
// first run.  httpReg, slogLogger, tlsMgr and confModifier must not be nil.
func runDNSServer(
	ctx context.Context,
	slogLogger *slog.Logger,
	tlsMgr *tlsManager,
	confModifier *defaultConfigModifier,
	statsDir string,
	querylogDir string,
	httpReg *aghhttp.DefaultRegistrar,
) {
	err := initDNS(ctx, slogLogger, tlsMgr, confModifier, httpReg, statsDir, querylogDir)
	fatalOnError(err)

	tlsMgr.start(ctx)

	go func() {
		startErr := startDNSServer()
		if startErr != nil {
			closeDNSServer(ctx)
			fatalOnError(startErr)
		}
	}()

	if globalContext.dhcpServer != nil {
		err = globalContext.dhcpServer.Start(ctx)
		if err != nil {
			slogLogger.ErrorContext(ctx, "starting dhcp server", slogutil.KeyError, err)
		}
	}
}

// initTLS initializes TLS manager.  baseLogger, sigHdlr, confModifier, and
// httpReg must not be nil.
func initTLS(
	ctx context.Context,
	baseLogger *slog.Logger,
	sigHdlr *signalHandler,
	confModifier *defaultConfigModifier,
	httpReg *aghhttp.DefaultRegistrar,
) (tlsMgr *tlsManager, err error) {
	tlsMgrLogger := baseLogger.With(slogutil.KeyPrefix, "tls_manager")

	var watcher aghos.FSWatcher
	watcher, err = aghos.NewOSWatcher(&aghos.OSWatcherConfig{
		Logger: tlsMgrLogger.With(slogutil.KeyPrefix, "cert_watcher"),
	})
	if err != nil {
		tlsMgrLogger.ErrorContext(ctx, "initializing watcher", slogutil.KeyError, err)
		watcher = aghos.EmptyFSWatcher{}
	}

	aghtlsMgr := aghtls.NewDefaultManager(&aghtls.DefaultManagerConfig{
		Logger:  baseLogger.With(slogutil.KeyPrefix, "aghtls_manager"),
		Watcher: watcher,
	})
	err = aghtlsMgr.Start(ctx)
	if err != nil {
		return nil, fmt.Errorf("starting tls manager: %w", err)
	}

	sigHdlr.addTLSManager(aghtlsMgr)

	tlsMgr, err = newTLSManager(ctx, &tlsManagerConfig{
		logger:        tlsMgrLogger,
		confModifier:  confModifier,
		manager:       aghtlsMgr,
		httpReg:       httpReg,
		tlsSettings:   config.TLS,
		servePlainDNS: config.DNS.ServePlainDNS,
	})
	if err != nil {
		tlsMgrLogger.ErrorContext(ctx, "initializing", slogutil.KeyError, err)
		confModifier.Apply(ctx)
	}

	confModifier.setTLSManager(tlsMgr)

	return tlsMgr, nil
}

// initUpdate configures and runs update of this application.  logger and tlsMgr
// must not be nil.
func initUpdate(
	ctx context.Context,
	baseLogger *slog.Logger,
	opts options,
	tlsMgr *tlsManager,
	isFirstRun bool,
	workDir string,
	confPath string,
) (upd *updater.Updater, isCustomURL bool) {
	execPath, err := os.Executable()
	fatalOnError(errors.Annotate(err, "getting executable path: %w"))

	updLogger := baseLogger.With(slogutil.KeyPrefix, "updater")
	upd, isCustomURL = newUpdater(
		ctx,
		updLogger,
		config,
		workDir,
		confPath,
		execPath,
	)

	// TODO(e.burkov): This could be made earlier, probably as the option's
	// effect.
	cmdlineUpdate(ctx, baseLogger, opts, upd, tlsMgr, isFirstRun)

	if !isFirstRun {
		// Save the updated config.
		err = config.write(ctx, baseLogger, nil, nil, workDir, confPath)
		fatalOnError(err)

		if config.HTTPConfig.Pprof.Enabled {
			startPprof(baseLogger, config.HTTPConfig.Pprof.Port)
		}
	}

	return upd, isCustomURL
}

// initEnvironment inits working environment.  opts and slogLogger must not be
// nil.
func initEnvironment(
	ctx context.Context,
	opts options,
	slogLogger *slog.Logger,
	workDir,
	confPath string,
) {
	ls := getLogSettings(ctx, slogLogger, opts, workDir, confPath)

	// Configure log level and output.
	err := configureLogger(ls, workDir)
	fatalOnError(err)

	// Print the first message after logger is configured.
	slogLogger.InfoContext(ctx, "starting adguard home", "version", version.Full())
	slogLogger.DebugContext(ctx, "current working directory", "path", workDir)
	if opts.runningAsService {
		slogLogger.InfoContext(ctx, "adguard home is running as a service")
	}

	aghtls.Init(ctx, slogLogger.With(slogutil.KeyPrefix, "aghtls"))
}

// newUpdater creates a new AdGuard Home updater.  l and conf must not be nil.
// workDir, confPath, and execPath must not be empty.  isCustomURL is true if
// the user has specified a custom version announcement URL.
func newUpdater(
	ctx context.Context,
	l *slog.Logger,
	conf *configuration,
	workDir string,
	confPath string,
	execPath string,
) (upd *updater.Updater, isCustomURL bool) {
	// envName is the name of the environment variable that can be used to
	// override the default version check URL.
	const envName = "ADGUARD_HOME_TEST_UPDATE_VERSION_URL"

	customURLStr := os.Getenv(envName)

	var versionURL *url.URL
	switch {
	case version.Channel() == version.ChannelRelease:
		// Only enable custom version URL for development builds.
		l.DebugContext(ctx, "custom version url is disabled for release builds")
	case !conf.UnsafeUseCustomUpdateIndexURL:
		l.DebugContext(ctx, "custom version url is disabled in config")
	default:
		versionURL, _ = url.Parse(customURLStr)
	}

	err := urlutil.ValidateHTTPURL(versionURL)
	if isCustomURL = err == nil; !isCustomURL {
		l.DebugContext(ctx, "parsing custom version url", slogutil.KeyError, err)

		versionURL = updater.DefaultVersionURL()
	}

	l.DebugContext(ctx, "creating updater", "config_path", confPath)

	return updater.NewUpdater(&updater.Config{
		Client:             conf.Filtering.HTTPClient,
		Logger:             l,
		CommandConstructor: executil.SystemCommandConstructor{},
		Version:            version.Version(),
		Channel:            version.Channel(),
		GOARCH:             runtime.GOARCH,
		GOOS:               runtime.GOOS,
		GOARM:              version.GOARM(),
		GOMIPS:             version.GOMIPS(),
		WorkDir:            workDir,
		ConfName:           confPath,
		ExecPath:           execPath,
		VersionCheckURL:    versionURL,
	}), isCustomURL
}

// checkPermissions checks and migrates permissions of the files and directories
// used by AdGuard Home, if needed.
func checkPermissions(
	ctx context.Context,
	baseLogger *slog.Logger,
	workDir string,
	confPath string,
	dataDirPath string,
	statsDir string,
	querylogDir string,
) {
	l := baseLogger.With(slogutil.KeyPrefix, "permcheck")

	if permcheck.NeedsMigration(ctx, l, workDir, confPath) {
		permcheck.Migrate(ctx, l, workDir, dataDirPath, statsDir, querylogDir, confPath)
	}

	permcheck.Check(ctx, l, workDir, dataDirPath, statsDir, querylogDir, confPath)
}

// initUsers initializes authentication module and clears the [config.Users]
// field.
func initUsers(
	ctx context.Context,
	baseLogger *slog.Logger,
	workDir string,
	isGLiNet bool,
) (auth *auth, err error) {
	var rateLimiter loginRateLimiter
	if config.AuthAttempts > 0 && config.AuthBlockMin > 0 {
		blockDur := time.Duration(config.AuthBlockMin) * time.Minute
		rateLimiter = newAuthRateLimiter(blockDur, config.AuthAttempts)
	} else {
		baseLogger.WarnContext(ctx, "authratelimiter is disabled")
		rateLimiter = emptyRateLimiter{}
	}

	dataDirPath := filepath.Join(workDir, dataDir)
	auth, err = newAuth(ctx, &authConfig{
		baseLogger:     baseLogger,
		rateLimiter:    rateLimiter,
		trustedProxies: netutil.SliceSubnetSet(netutil.UnembedPrefixes(config.DNS.TrustedProxies)),
		dbFilename:     filepath.Join(dataDirPath, sessionsDBName),
		users:          config.Users,
		sessionTTL:     time.Duration(config.HTTPConfig.SessionTTL),
		isGLiNet:       isGLiNet,
	})
	if err != nil {
		return nil, fmt.Errorf("initializing auth module: %w", err)
	}

	config.Users = nil

	return auth, nil
}

func (c *configuration) anonymizer() (ipmut *aghnet.IPMut) {
	var anonFunc aghnet.IPMutFunc
	if c.DNS.AnonymizeClientIP {
		anonFunc = querylog.AnonymizeIP
	}

	return aghnet.NewIPMut(anonFunc)
}

// permCheckHelp is printed when binding to privileged ports is not permitted.
const permCheckHelp = `Permission check failed.

AdGuard Home is not allowed to bind to privileged ports (for instance, port 53).
Please note that this is crucial for a server to be able to use privileged ports.

You have two options:
1. Run AdGuard Home with root privileges.
2. On Linux you can grant the CAP_NET_BIND_SERVICE capability:
https://github.com/AdguardTeam/AdGuardHome/wiki/Getting-Started#running-without-superuser`

// checkNetworkPermissions checks if the current user permissions are enough to
// use the required networking functionality.  l must not be nil.
func checkNetworkPermissions(ctx context.Context, l *slog.Logger) {
	l.InfoContext(ctx, "checking if adguard home has the necessary permissions")

	if ok, err := aghnet.CanBindPrivilegedPorts(ctx, l); !ok || err != nil {
		l.ErrorContext(
			ctx,
			"this is the first launch of adguard home; you must run it as administrator.",
		)

		os.Exit(osutil.ExitCodeFailure)
	}

	// We should check if AdGuard Home is able to bind to port 53
	err := aghnet.CheckPort("tcp", netip.AddrPortFrom(netutil.IPv4Localhost(), defaultPortDNS))
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			slogutil.PrintLines(ctx, l, slog.LevelError, "", permCheckHelp)

			os.Exit(osutil.ExitCodeFailure)
		}

		l.ErrorContext(
			ctx,
			"failed to bind to port 53; binding to port 53 is required for a dns server",
			slogutil.KeyError, err,
		)
	}

	l.InfoContext(ctx, "adguard home can bind to port 53")
}

// Write PID to a file
func writePIDFile(fn string) bool {
	data := fmt.Sprintf("%d", os.Getpid())
	err := os.WriteFile(fn, []byte(data), 0o644)
	if err != nil {
		log.Error("Couldn't write PID to file %s: %v", fn, err)
		return false
	}
	return true
}

// initConfigFilename returns the configuration file path.  If a path is
// provided via command-line argument, it is used; otherwise a default within
// workDir is returned.  l must not be nil.
func initConfigFilename(
	ctx context.Context,
	l *slog.Logger,
	opts options,
	workDir string,
) (confPath string) {
	confPath = opts.confFilename
	if confPath != "" {
		l.DebugContext(ctx, "config path overridden from cmdline", "path", confPath)

		return confPath
	}

	confPath = filepath.Join(workDir, "AdGuardHome.yaml")

	return confPath
}

// initWorkingDir returns the working directory path.  If no command-line
// argument is provided, it uses the executable's directory.
func initWorkingDir(opts options) (workDir string, err error) {
	if opts.workDir != "" {
		workDir = opts.workDir
	} else {
		var execPath string
		execPath, err = os.Executable()
		if err != nil {
			// Don't wrap the error, because it's informative enough as is.
			return "", err
		}

		workDir = filepath.Dir(execPath)
	}

	workDir, err = filepath.EvalSymlinks(workDir)
	if err != nil {
		// Don't wrap the error, because it's informative enough as is.
		return "", err
	}

	return workDir, nil
}

// cleanup stops and resets all the modules.
func cleanup(ctx context.Context) {
	log.Info("stopping AdGuard Home")

	if globalContext.web != nil {
		globalContext.web.close(ctx)
		globalContext.web = nil
	}

	err := stopDNSServer(ctx)
	if err != nil {
		log.Error("stopping dns server: %s", err)
	}

	if globalContext.dhcpServer != nil {
		err = globalContext.dhcpServer.Stop()
		if err != nil {
			log.Error("stopping dhcp server: %s", err)
		}
	}

	if globalContext.etcHosts != nil {
		if err = globalContext.etcHosts.Close(); err != nil {
			log.Error("closing hosts container: %s", err)
		}
	}
}

// This function is called before application exits
func cleanupAlways() {
	if len(globalContext.pidFileName) != 0 {
		_ = os.Remove(globalContext.pidFileName)
	}

	log.Info("stopped")
}

func exitWithError() {
	os.Exit(64)
}

// loadCmdLineOpts reads command line arguments and initializes configuration
// from them.  If there is an error or an effect, loadCmdLineOpts processes them
// and exits.
func loadCmdLineOpts() (opts options) {
	opts, eff, err := parseCmdOpts(os.Args[0], os.Args[1:])
	if err != nil {
		log.Error("%s", err)
		printHelp(os.Args[0])

		exitWithError()
	}

	if eff != nil {
		err = eff()
		if err != nil {
			log.Error("%s", err)
			exitWithError()
		}

		os.Exit(osutil.ExitCodeSuccess)
	}

	return opts
}

// printWebAddrs prints addresses built from proto, addr, and an appropriate
// port.  At least one address is printed with the value of port.  Output
// example:
//
//	go to http://127.0.0.1:80
func printWebAddrs(proto, addr string, port uint16) {
	log.Printf("go to %s://%s", proto, netutil.JoinHostPort(addr, port))
}

// printHTTPAddresses prints the IP addresses which user can use to access the
// admin interface.  proto is either schemeHTTP or schemeHTTPS.
//
// TODO(s.chzhen):  Implement separate functions for HTTP and HTTPS.
func printHTTPAddresses(proto string, tlsMgr *tlsManager) {
	var tlsConf *tlsConfigSettings
	if tlsMgr != nil {
		tlsConf = tlsMgr.config()
	}

	port := config.HTTPConfig.Address.Port()
	if proto == urlutil.SchemeHTTPS {
		port = tlsConf.PortHTTPS
	}

	if proto == urlutil.SchemeHTTPS && tlsConf.ServerName != "" {
		printWebAddrs(proto, tlsConf.ServerName, tlsConf.PortHTTPS)

		return
	}

	bindHost := config.HTTPConfig.Address.Addr()
	if !bindHost.IsUnspecified() {
		printWebAddrs(proto, bindHost.String(), port)

		return
	}

	ifaces, err := aghnet.GetValidNetInterfacesForWeb()
	if err != nil {
		log.Error("web: getting iface ips: %s", err)
		// That's weird, but we'll ignore it.
		//
		// TODO(e.burkov): Find out when it happens.
		printWebAddrs(proto, bindHost.String(), port)

		return
	}

	for _, iface := range ifaces {
		for _, addr := range iface.Addresses {
			printWebAddrs(proto, addr.String(), port)
		}
	}
}

// detectFirstRun returns true if this is the first run of AdGuard Home.  l must
// not be nil.
func detectFirstRun(ctx context.Context, l *slog.Logger, workDir, confPath string) (ok bool) {
	if !filepath.IsAbs(confPath) {
		confPath = filepath.Join(workDir, confPath)
	}

	_, err := os.Stat(confPath)
	if err == nil {
		return false
	} else if errors.Is(err, os.ErrNotExist) {
		return true
	}

	l.ErrorContext(ctx, "failed to detect first run; considering first run", slogutil.KeyError, err)

	return true
}

// jsonError is a generic JSON error response.
//
// TODO(a.garipov): Merge together with the implementations in [dhcpd] and other
// packages after refactoring the web handler registering.
type jsonError struct {
	// Message is the error message, an opaque string.
	Message string `json:"message"`
}

// cmdlineUpdate updates current application and exits.  l, upd, and tlsMgr must
// not be nil.
func cmdlineUpdate(
	ctx context.Context,
	l *slog.Logger,
	opts options,
	upd *updater.Updater,
	tlsMgr *tlsManager,
	isFirstRun bool,
) {
	if !opts.performUpdate {
		return
	}

	// Initialize the DNS server to use the internal resolver which the updater
	// needs to be able to resolve the update source hostname.
	//
	// TODO(e.burkov):  We could probably initialize the internal resolver
	// separately.
	err := initDNSServer(ctx, nil, nil, nil, nil, nil, nil, tlsMgr, l, agh.EmptyConfigModifier{})
	fatalOnError(err)

	l.InfoContext(ctx, "performing update via cli")

	info, err := upd.VersionInfo(ctx, true)
	if err != nil {
		l.ErrorContext(ctx, "getting version info", slogutil.KeyError, err)

		os.Exit(osutil.ExitCodeFailure)
	}

	if info.NewVersion == version.Version() {
		l.InfoContext(ctx, "no updates available")

		os.Exit(osutil.ExitCodeSuccess)
	}

	err = upd.Update(ctx, isFirstRun)
	fatalOnError(err)

	err = restartService(ctx, l)
	if err != nil {
		l.DebugContext(ctx, "restarting service", slogutil.KeyError, err)
		l.InfoContext(ctx, "AdGuard Home was not installed as a service. "+
			"Please restart running instances of AdGuardHome manually.")
	}

	os.Exit(osutil.ExitCodeSuccess)
}
