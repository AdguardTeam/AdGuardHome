// Package home contains AdGuard Home's HTTP API methods.
package home

import (
	"context"
	"crypto/x509"
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
	"sync"
	"syscall"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghalg"
	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
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
)

// Global context
type homeContext struct {
	// Modules
	// --

	clients    clientsContainer     // per-client-settings module
	stats      stats.Interface      // statistics module
	queryLog   querylog.QueryLog    // query log module
	dnsServer  *dnsforward.Server   // DNS module
	dhcpServer dhcpd.Interface      // DHCP module
	auth       *Auth                // HTTP authentication module
	filters    *filtering.DNSFilter // DNS filtering module
	web        *webAPI              // Web (HTTP, HTTPS) module
	tls        *tlsManager          // TLS module

	// etcHosts contains IP-hostname mappings taken from the OS-specific hosts
	// configuration files, for example /etc/hosts.
	etcHosts *aghnet.HostsContainer

	// mux is our custom http.ServeMux.
	mux *http.ServeMux

	// Runtime properties
	// --

	// confFilePath is the configuration file path as set by default or from the
	// command-line options.
	confFilePath string

	workDir     string // Location of our directory, used to protect against CWD being somewhere else
	pidFileName string // PID file name.  Empty if no PID file was created.
	controlLock sync.Mutex
	tlsRoots    *x509.CertPool // list of root CAs for TLSv1.2

	// tlsCipherIDs are the ID of the cipher suites that AdGuard Home must use.
	tlsCipherIDs []uint16

	// firstRun, if true, tells AdGuard Home to only start the web interface
	// service, and only serve the first-run APIs.
	firstRun bool
}

// getDataDir returns path to the directory where we store databases and filters
func (c *homeContext) getDataDir() string {
	return filepath.Join(c.workDir, dataDir)
}

// Context - a global context object
//
// TODO(a.garipov): Refactor.
var Context homeContext

// Main is the entry point
func Main(clientBuildFS fs.FS) {
	initCmdLineOpts()

	// The configuration file path can be overridden, but other command-line
	// options have to override config values.  Therefore, do it manually
	// instead of using package flag.
	//
	// TODO(a.garipov): The comment above is most likely false.  Replace with
	// package flag.
	opts := loadCmdLineOpts()

	done := make(chan struct{})

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)

	go func() {
		ctx := context.Background()
		for {
			sig := <-signals
			log.Info("Received signal %q", sig)
			switch sig {
			case syscall.SIGHUP:
				Context.clients.storage.ReloadARP(ctx)
				Context.tls.reload()
			default:
				cleanup(ctx)
				cleanupAlways()
				close(done)
			}
		}
	}()

	if opts.serviceControlAction != "" {
		handleServiceControlAction(opts, clientBuildFS, signals, done)

		return
	}

	// run the protection
	run(opts, clientBuildFS, done)
}

// setupContext initializes [Context] fields.  It also reads and upgrades
// config file if necessary.
func setupContext(opts options) (err error) {
	Context.firstRun = detectFirstRun()

	Context.tlsRoots = aghtls.SystemRootCAs()
	Context.mux = http.NewServeMux()

	if !opts.noEtcHosts {
		err = setupHostsContainer()
		if err != nil {
			// Don't wrap the error, because it's informative enough as is.
			return err
		}
	}

	if Context.firstRun {
		log.Info("This is the first time AdGuard Home is launched")
		checkNetworkPermissions()

		return nil
	}

	err = parseConfig()
	if err != nil {
		log.Error("parsing configuration file: %s", err)

		os.Exit(osutil.ExitCodeFailure)
	}

	if opts.checkConfig {
		log.Info("configuration file is ok")

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
// provided by the OS.
func setupHostsContainer() (err error) {
	hostsWatcher, err := aghos.NewOSWritesWatcher()
	if err != nil {
		log.Info("WARNING: initializing filesystem watcher: %s; not watching for changes", err)

		hostsWatcher = aghos.EmptyFSWatcher{}
	}

	paths, err := hostsfile.DefaultHostsPaths()
	if err != nil {
		return fmt.Errorf("getting default system hosts paths: %w", err)
	}

	Context.etcHosts, err = aghnet.NewHostsContainer(osutil.RootDirFS(), hostsWatcher, paths...)
	if err != nil {
		closeErr := hostsWatcher.Close()
		if errors.Is(err, aghnet.ErrNoHostsPaths) {
			log.Info("warning: initing hosts container: %s", err)

			return closeErr
		}

		return errors.Join(fmt.Errorf("initializing hosts container: %w", err), closeErr)
	}

	return hostsWatcher.Start()
}

// setupOpts sets up command-line options.
func setupOpts(opts options) (err error) {
	err = setupBindOpts(opts)
	if err != nil {
		// Don't wrap the error, because it's informative enough as is.
		return err
	}

	if len(opts.pidFile) != 0 && writePIDFile(opts.pidFile) {
		Context.pidFileName = opts.pidFile
	}

	return nil
}

// initContextClients initializes Context clients and related fields.
func initContextClients(ctx context.Context, logger *slog.Logger) (err error) {
	err = setupDNSFilteringConf(ctx, logger, config.Filtering)
	if err != nil {
		// Don't wrap the error, because it's informative enough as is.
		return err
	}

	//lint:ignore SA1019 Migration is not over.
	config.DHCP.WorkDir = Context.workDir
	config.DHCP.DataDir = Context.getDataDir()
	config.DHCP.HTTPRegister = httpRegister
	config.DHCP.ConfigModified = onConfigModified

	Context.dhcpServer, err = dhcpd.Create(config.DHCP)
	if Context.dhcpServer == nil || err != nil {
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

	return Context.clients.Init(
		ctx,
		logger,
		config.Clients.Persistent,
		Context.dhcpServer,
		Context.etcHosts,
		arpDB,
		config.Filtering,
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

// setupDNSFilteringConf sets up DNS filtering configuration settings.
func setupDNSFilteringConf(
	ctx context.Context,
	baseLogger *slog.Logger,
	conf *filtering.Config,
) (err error) {
	const (
		dnsTimeout = 3 * time.Second

		sbService                 = "safe browsing"
		defaultSafeBrowsingServer = `https://family.adguard-dns.com/dns-query`
		sbTXTSuffix               = `sb.dns.adguard.com.`

		pcService             = "parental control"
		defaultParentalServer = `https://family.adguard-dns.com/dns-query`
		pcTXTSuffix           = `pc.dns.adguard.com.`
	)

	conf.EtcHosts = Context.etcHosts
	// TODO(s.chzhen):  Use empty interface.
	if Context.etcHosts == nil || !config.DNS.HostsFileEnabled {
		conf.EtcHosts = nil
	}

	conf.ConfigModified = onConfigModified
	conf.HTTPRegister = httpRegister
	conf.DataDir = Context.getDataDir()
	conf.Filters = slices.Clone(config.Filters)
	conf.WhitelistFilters = slices.Clone(config.WhitelistFilters)
	conf.UserRules = slices.Clone(config.UserRules)
	conf.HTTPClient = httpClient()

	cacheTime := time.Duration(conf.CacheTime) * time.Minute

	upsOpts := &upstream.Options{
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
		Upstream:    sbUps,
		ServiceName: sbService,
		TXTSuffix:   sbTXTSuffix,
		CacheTime:   cacheTime,
		CacheSize:   conf.SafeBrowsingCacheSize,
	})

	// Protect against invalid configuration, see #6181.
	//
	// TODO(a.garipov): Validate against an empty host instead of setting it to
	// default.
	if conf.SafeBrowsingBlockHost == "" {
		host := defaultSafeBrowsingBlockHost
		log.Info("%s: warning: empty blocking host; using default: %q", sbService, host)

		conf.SafeBrowsingBlockHost = host
	}

	parUps, err := upstream.AddressToUpstream(defaultParentalServer, upsOpts)
	if err != nil {
		return fmt.Errorf("converting parental server: %w", err)
	}

	conf.ParentalControlChecker = hashprefix.New(&hashprefix.Config{
		Upstream:    parUps,
		ServiceName: pcService,
		TXTSuffix:   pcTXTSuffix,
		CacheTime:   cacheTime,
		CacheSize:   conf.ParentalCacheSize,
	})

	// Protect against invalid configuration, see #6181.
	//
	// TODO(a.garipov): Validate against an empty host instead of setting it to
	// default.
	if conf.ParentalBlockHost == "" {
		host := defaultParentalBlockHost
		log.Info("%s: warning: empty blocking host; using default: %q", pcService, host)

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
// configuration.  It also logs the decision.  customURL should be true if the
// updater is using a custom URL.
func isUpdateEnabled(ctx context.Context, l *slog.Logger, opts *options, customURL bool) (ok bool) {
	if opts.disableUpdate {
		l.DebugContext(ctx, "updates are disabled by command-line option")

		return false
	}

	switch version.Channel() {
	case
		version.ChannelDevelopment,
		version.ChannelCandidate:
		if customURL {
			l.DebugContext(ctx, "updates are enabled because custom url is used")
		} else {
			l.DebugContext(ctx, "updates are disabled for development and candidate builds")
		}

		return customURL
	default:
		l.DebugContext(ctx, "updates are enabled")

		return true
	}
}

// initWeb initializes the web module.  upd and baseLogger must not be nil.
func initWeb(
	ctx context.Context,
	opts options,
	clientBuildFS fs.FS,
	upd *updater.Updater,
	baseLogger *slog.Logger,
	customURL bool,
) (web *webAPI, err error) {
	logger := baseLogger.With(slogutil.KeyPrefix, "webapi")

	var clientFS fs.FS
	if opts.localFrontend {
		logger.WarnContext(ctx, "using local frontend files")

		clientFS = os.DirFS("build/static")
	} else {
		clientFS, err = fs.Sub(clientBuildFS, "build/static")
		if err != nil {
			return nil, fmt.Errorf("getting embedded client subdir: %w", err)
		}
	}

	disableUpdate := !isUpdateEnabled(ctx, baseLogger, &opts, customURL)

	webConf := &webConfig{
		updater:    upd,
		logger:     logger,
		baseLogger: baseLogger,

		clientFS: clientFS,

		BindAddr: config.HTTPConfig.Address,

		ReadTimeout:       readTimeout,
		ReadHeaderTimeout: readHdrTimeout,
		WriteTimeout:      writeTimeout,

		firstRun:         Context.firstRun,
		disableUpdate:    disableUpdate,
		runningAsService: opts.runningAsService,
		serveHTTP3:       config.DNS.ServeHTTP3,
	}

	web = newWebAPI(ctx, webConf)
	if web == nil {
		return nil, errors.Error("can not initialize web")
	}

	return web, nil
}

func fatalOnError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// run configures and starts AdGuard Home.
//
// TODO(e.burkov):  Make opts a pointer.
func run(opts options, clientBuildFS fs.FS, done chan struct{}) {
	// Configure working dir.
	err := initWorkingDir(opts)
	fatalOnError(err)

	// Configure config filename.
	initConfigFilename(opts)

	ls := getLogSettings(opts)

	// Configure log level and output.
	err = configureLogger(ls)
	fatalOnError(err)

	// TODO(a.garipov): Use slog everywhere.
	slogLogger := newSlogLogger(ls)

	// Print the first message after logger is configured.
	log.Info(version.Full())
	log.Debug("current working directory is %s", Context.workDir)
	if opts.runningAsService {
		log.Info("AdGuard Home is running as a service")
	}

	err = setupContext(opts)
	fatalOnError(err)

	err = configureOS(config)
	fatalOnError(err)

	// Clients package uses filtering package's static data
	// (filtering.BlockedSvcKnown()), so we have to initialize filtering static
	// data first, but also to avoid relying on automatic Go init() function.
	filtering.InitModule()

	// TODO(s.chzhen):  Use it for the entire initialization process.
	ctx := context.Background()

	err = initContextClients(ctx, slogLogger)
	fatalOnError(err)

	err = setupOpts(opts)
	fatalOnError(err)

	execPath, err := os.Executable()
	fatalOnError(errors.Annotate(err, "getting executable path: %w"))

	confPath := configFilePath()

	upd, customURL := newUpdater(ctx, slogLogger, Context.workDir, confPath, execPath, config)

	// TODO(e.burkov): This could be made earlier, probably as the option's
	// effect.
	cmdlineUpdate(ctx, slogLogger, opts, upd)

	if !Context.firstRun {
		// Save the updated config.
		err = config.write()
		fatalOnError(err)

		if config.HTTPConfig.Pprof.Enabled {
			startPprof(slogLogger, config.HTTPConfig.Pprof.Port)
		}
	}

	dataDir := Context.getDataDir()
	err = os.MkdirAll(dataDir, aghos.DefaultPermDir)
	fatalOnError(errors.Annotate(err, "creating DNS data dir at %s: %w", dataDir))

	GLMode = opts.glinetMode

	// Init auth module.
	Context.auth, err = initUsers()
	fatalOnError(err)

	Context.tls, err = newTLSManager(config.TLS, config.DNS.ServePlainDNS)
	if err != nil {
		log.Error("initializing tls: %s", err)
		onConfigModified()
	}

	Context.web, err = initWeb(ctx, opts, clientBuildFS, upd, slogLogger, customURL)
	fatalOnError(err)

	statsDir, querylogDir, err := checkStatsAndQuerylogDirs(&Context, config)
	fatalOnError(err)

	if !Context.firstRun {
		err = initDNS(slogLogger, statsDir, querylogDir)
		fatalOnError(err)

		Context.tls.start()

		go func() {
			startErr := startDNSServer()
			if startErr != nil {
				closeDNSServer()
				fatalOnError(startErr)
			}
		}()

		if Context.dhcpServer != nil {
			err = Context.dhcpServer.Start()
			if err != nil {
				log.Error("starting dhcp server: %s", err)
			}
		}
	}

	if !opts.noPermCheck {
		checkPermissions(ctx, slogLogger, Context.workDir, confPath, dataDir, statsDir, querylogDir)
	}

	Context.web.start(ctx)

	// Wait for other goroutines to complete their job.
	<-done
}

// newUpdater creates a new AdGuard Home updater.  customURL is true if the user
// has specified a custom version announcement URL.
func newUpdater(
	ctx context.Context,
	l *slog.Logger,
	workDir string,
	confPath string,
	execPath string,
	config *configuration,
) (upd *updater.Updater, customURL bool) {
	// envName is the name of the environment variable that can be used to
	// override the default version check URL.
	const envName = "ADGUARD_HOME_TEST_UPDATE_VERSION_URL"

	customURLStr := os.Getenv(envName)

	var versionURL *url.URL
	switch {
	case version.Channel() == version.ChannelRelease:
		// Only enable custom version URL for development builds.
		l.DebugContext(ctx, "custom version url is disabled for release builds")
	case !config.UnsafeUseCustomUpdateIndexURL:
		l.DebugContext(ctx, "custom version url is disabled in config")
	default:
		versionURL, _ = url.Parse(customURLStr)
	}

	err := urlutil.ValidateHTTPURL(versionURL)
	if customURL = err == nil; !customURL {
		l.DebugContext(ctx, "parsing custom version url", slogutil.KeyError, err)

		versionURL = updater.DefaultVersionURL()
	}

	l.DebugContext(ctx, "creating updater", "config_path", confPath)

	return updater.NewUpdater(&updater.Config{
		Client:          config.Filtering.HTTPClient,
		Version:         version.Version(),
		Channel:         version.Channel(),
		GOARCH:          runtime.GOARCH,
		GOOS:            runtime.GOOS,
		GOARM:           version.GOARM(),
		GOMIPS:          version.GOMIPS(),
		WorkDir:         workDir,
		ConfName:        confPath,
		ExecPath:        execPath,
		VersionCheckURL: versionURL,
	}), customURL
}

// checkPermissions checks and migrates permissions of the files and directories
// used by AdGuard Home, if needed.
func checkPermissions(
	ctx context.Context,
	baseLogger *slog.Logger,
	workDir string,
	confPath string,
	dataDir string,
	statsDir string,
	querylogDir string,
) {
	l := baseLogger.With(slogutil.KeyPrefix, "permcheck")

	if permcheck.NeedsMigration(ctx, l, workDir, confPath) {
		permcheck.Migrate(ctx, l, workDir, dataDir, statsDir, querylogDir, confPath)
	}

	permcheck.Check(ctx, l, workDir, dataDir, statsDir, querylogDir, confPath)
}

// initUsers initializes context auth module.  Clears config users field.
func initUsers() (auth *Auth, err error) {
	sessFilename := filepath.Join(Context.getDataDir(), "sessions.db")

	var rateLimiter *authRateLimiter
	if config.AuthAttempts > 0 && config.AuthBlockMin > 0 {
		blockDur := time.Duration(config.AuthBlockMin) * time.Minute
		rateLimiter = newAuthRateLimiter(blockDur, config.AuthAttempts)
	} else {
		log.Info("authratelimiter is disabled")
	}

	trustedProxies := netutil.SliceSubnetSet(netutil.UnembedPrefixes(config.DNS.TrustedProxies))

	sessionTTL := time.Duration(config.HTTPConfig.SessionTTL).Seconds()
	auth = InitAuth(sessFilename, config.Users, uint32(sessionTTL), rateLimiter, trustedProxies)
	if auth == nil {
		return nil, errors.Error("initializing auth module failed")
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

// startMods initializes and starts the DNS server after installation.
// baseLogger must not be nil.
func startMods(baseLogger *slog.Logger) (err error) {
	statsDir, querylogDir, err := checkStatsAndQuerylogDirs(&Context, config)
	if err != nil {
		return err
	}

	err = initDNS(baseLogger, statsDir, querylogDir)
	if err != nil {
		return err
	}

	Context.tls.start()

	err = startDNSServer()
	if err != nil {
		closeDNSServer()

		return err
	}

	return nil
}

// checkNetworkPermissions checks if the current user permissions are enough to
// use the required networking functionality.
func checkNetworkPermissions() {
	log.Info("Checking if AdGuard Home has necessary permissions")

	if ok, err := aghnet.CanBindPrivilegedPorts(); !ok || err != nil {
		log.Fatal("This is the first launch of AdGuard Home. You must run it as Administrator.")
	}

	// We should check if AdGuard Home is able to bind to port 53
	err := aghnet.CheckPort("tcp", netip.AddrPortFrom(netutil.IPv4Localhost(), defaultPortDNS))
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			log.Fatal(`Permission check failed.

AdGuard Home is not allowed to bind to privileged ports (for instance, port 53).
Please note, that this is crucial for a server to be able to use privileged ports.

You have two options:
1. Run AdGuard Home with root privileges
2. On Linux you can grant the CAP_NET_BIND_SERVICE capability:
https://github.com/AdguardTeam/AdGuardHome/wiki/Getting-Started#running-without-superuser`)
		}

		log.Info(
			"AdGuard failed to bind to port 53: %s\n\n"+
				"Please note, that this is crucial for a DNS server to be able to use that port.",
			err,
		)
	}

	log.Info("AdGuard Home can bind to port 53")
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

// initConfigFilename sets up context config file path.  This file path can be
// overridden by command-line arguments, or is set to default.  Must only be
// called after initializing the workDir with initWorkingDir.
func initConfigFilename(opts options) {
	confPath := opts.confFilename
	if confPath == "" {
		Context.confFilePath = filepath.Join(Context.workDir, "AdGuardHome.yaml")

		return
	}

	log.Debug("config path overridden to %q from cmdline", confPath)

	Context.confFilePath = confPath
}

// initWorkingDir initializes the workDir.  If no command-line arguments are
// specified, the directory with the binary file is used.
func initWorkingDir(opts options) (err error) {
	execPath, err := os.Executable()
	if err != nil {
		// Don't wrap the error, because it's informative enough as is.
		return err
	}

	if opts.workDir != "" {
		// If there is a custom config file, use it's directory as our working dir
		Context.workDir = opts.workDir
	} else {
		Context.workDir = filepath.Dir(execPath)
	}

	workDir, err := filepath.EvalSymlinks(Context.workDir)
	if err != nil {
		// Don't wrap the error, because it's informative enough as is.
		return err
	}

	Context.workDir = workDir

	return nil
}

// cleanup stops and resets all the modules.
func cleanup(ctx context.Context) {
	log.Info("stopping AdGuard Home")

	if Context.web != nil {
		Context.web.close(ctx)
		Context.web = nil
	}
	if Context.auth != nil {
		Context.auth.Close()
		Context.auth = nil
	}

	err := stopDNSServer()
	if err != nil {
		log.Error("stopping dns server: %s", err)
	}

	if Context.dhcpServer != nil {
		err = Context.dhcpServer.Stop()
		if err != nil {
			log.Error("stopping dhcp server: %s", err)
		}
	}

	if Context.etcHosts != nil {
		if err = Context.etcHosts.Close(); err != nil {
			log.Error("closing hosts container: %s", err)
		}
	}

	if Context.tls != nil {
		Context.tls = nil
	}
}

// This function is called before application exits
func cleanupAlways() {
	if len(Context.pidFileName) != 0 {
		_ = os.Remove(Context.pidFileName)
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
		log.Error(err.Error())
		printHelp(os.Args[0])

		exitWithError()
	}

	if eff != nil {
		err = eff()
		if err != nil {
			log.Error(err.Error())
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
func printHTTPAddresses(proto string) {
	tlsConf := tlsConfigSettings{}
	if Context.tls != nil {
		Context.tls.WriteDiskConfig(&tlsConf)
	}

	port := config.HTTPConfig.Address.Port()
	if proto == urlutil.SchemeHTTPS {
		port = tlsConf.PortHTTPS
	}

	// TODO(e.burkov): Inspect and perhaps merge with the previous condition.
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

// detectFirstRun returns true if this is the first run of AdGuard Home.
func detectFirstRun() (ok bool) {
	confPath := Context.confFilePath
	if !filepath.IsAbs(confPath) {
		confPath = filepath.Join(Context.workDir, Context.confFilePath)
	}

	_, err := os.Stat(confPath)
	if err == nil {
		return false
	} else if errors.Is(err, os.ErrNotExist) {
		return true
	}

	log.Error("detecting first run: %s; considering first run", err)

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

// cmdlineUpdate updates current application and exits.  l must not be nil.
func cmdlineUpdate(ctx context.Context, l *slog.Logger, opts options, upd *updater.Updater) {
	if !opts.performUpdate {
		return
	}

	// Initialize the DNS server to use the internal resolver which the updater
	// needs to be able to resolve the update source hostname.
	//
	// TODO(e.burkov):  We could probably initialize the internal resolver
	// separately.
	err := initDNSServer(nil, nil, nil, nil, nil, nil, &tlsConfigSettings{}, l)
	fatalOnError(err)

	l.InfoContext(ctx, "performing update via cli")

	info, err := upd.VersionInfo(true)
	if err != nil {
		l.ErrorContext(ctx, "getting version info", slogutil.KeyError, err)

		os.Exit(osutil.ExitCodeFailure)
	}

	if info.NewVersion == version.Version() {
		l.InfoContext(ctx, "no updates available")

		os.Exit(osutil.ExitCodeSuccess)
	}

	err = upd.Update(Context.firstRun)
	fatalOnError(err)

	err = restartService()
	if err != nil {
		l.DebugContext(ctx, "restarting service", slogutil.KeyError, err)
		l.InfoContext(ctx, "AdGuard Home was not installed as a service. "+
			"Please restart running instances of AdGuardHome manually.")
	}

	os.Exit(osutil.ExitCodeSuccess)
}
