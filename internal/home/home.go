// Package home contains AdGuard Home's HTTP API methods.
package home

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghalg"
	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/AdGuardHome/internal/aghtls"
	"github.com/AdguardTeam/AdGuardHome/internal/dhcpd"
	"github.com/AdguardTeam/AdGuardHome/internal/dnsforward"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering/hashprefix"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering/safesearch"
	"github.com/AdguardTeam/AdGuardHome/internal/querylog"
	"github.com/AdguardTeam/AdGuardHome/internal/stats"
	"github.com/AdguardTeam/AdGuardHome/internal/updater"
	"github.com/AdguardTeam/AdGuardHome/internal/version"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/stringutil"
	"golang.org/x/exp/slices"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	// Used in config to indicate that syslog or eventlog (win) should be used for logger output
	configSyslog = "syslog"
)

// Global context
type homeContext struct {
	// Modules
	// --

	clients    clientsContainer     // per-client-settings module
	stats      stats.Interface      // statistics module
	queryLog   querylog.QueryLog    // query log module
	dnsServer  *dnsforward.Server   // DNS module
	rdns       *RDNS                // rDNS module
	dhcpServer dhcpd.Interface      // DHCP module
	auth       *Auth                // HTTP authentication module
	filters    *filtering.DNSFilter // DNS filtering module
	web        *webAPI              // Web (HTTP, HTTPS) module
	tls        *tlsManager          // TLS module

	// etcHosts contains IP-hostname mappings taken from the OS-specific hosts
	// configuration files, for example /etc/hosts.
	etcHosts *aghnet.HostsContainer

	updater *updater.Updater

	// mux is our custom http.ServeMux.
	mux *http.ServeMux

	// Runtime properties
	// --

	configFilename   string // Config filename (can be overridden via the command line arguments)
	workDir          string // Location of our directory, used to protect against CWD being somewhere else
	pidFileName      string // PID file name.  Empty if no PID file was created.
	controlLock      sync.Mutex
	tlsRoots         *x509.CertPool // list of root CAs for TLSv1.2
	client           *http.Client
	appSignalChannel chan os.Signal // Channel for receiving OS signals by the console app

	// whoisCh is the channel for receiving IPs for WHOIS processing.
	whoisCh chan netip.Addr

	// tlsCipherIDs are the ID of the cipher suites that AdGuard Home must use.
	tlsCipherIDs []uint16

	// disableUpdate, if true, tells AdGuard Home to not check for updates.
	disableUpdate bool

	// firstRun, if true, tells AdGuard Home to only start the web interface
	// service, and only serve the first-run APIs.
	firstRun bool

	// runningAsService flag is set to true when options are passed from the service runner
	runningAsService bool
}

// getDataDir returns path to the directory where we store databases and filters
func (c *homeContext) getDataDir() string {
	return filepath.Join(c.workDir, dataDir)
}

// Context - a global context object
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

	Context.appSignalChannel = make(chan os.Signal)
	signal.Notify(Context.appSignalChannel, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)
	go func() {
		for {
			sig := <-Context.appSignalChannel
			log.Info("Received signal %q", sig)
			switch sig {
			case syscall.SIGHUP:
				Context.clients.reloadARP()
				Context.tls.reload()
			default:
				cleanup(context.Background())
				cleanupAlways()
				os.Exit(0)
			}
		}
	}()

	if opts.serviceControlAction != "" {
		handleServiceControlAction(opts, clientBuildFS)

		return
	}

	// run the protection
	run(opts, clientBuildFS)
}

// setupContext initializes [Context] fields.  It also reads and upgrades
// config file if necessary.
func setupContext(opts options) (err error) {
	setupContextFlags(opts)

	Context.tlsRoots = aghtls.SystemRootCAs()
	Context.client = &http.Client{
		Timeout: time.Minute * 5,
		Transport: &http.Transport{
			DialContext: customDialContext,
			Proxy:       getHTTPProxy,
			TLSClientConfig: &tls.Config{
				RootCAs:      Context.tlsRoots,
				CipherSuites: Context.tlsCipherIDs,
				MinVersion:   tls.VersionTLS12,
			},
		},
	}

	Context.mux = http.NewServeMux()

	if !Context.firstRun {
		// Do the upgrade if necessary.
		err = upgradeConfig()
		if err != nil {
			// Don't wrap the error, because it's informative enough as is.
			return err
		}

		if err = parseConfig(); err != nil {
			log.Error("parsing configuration file: %s", err)

			os.Exit(1)
		}

		if opts.checkConfig {
			log.Info("configuration file is ok")

			os.Exit(0)
		}

		if !opts.noEtcHosts && config.Clients.Sources.HostsFile {
			err = setupHostsContainer()
			if err != nil {
				// Don't wrap the error, because it's informative enough as is.
				return err
			}
		}
	}

	return nil
}

// setupContextFlags sets global flags and prints their status to the log.
func setupContextFlags(opts options) {
	Context.firstRun = detectFirstRun()
	if Context.firstRun {
		log.Info("This is the first time AdGuard Home is launched")
		checkPermissions()
	}

	Context.runningAsService = opts.runningAsService
	// Don't print the runningAsService flag, since that has already been done
	// in [run].

	Context.disableUpdate = opts.disableUpdate || version.Channel() == version.ChannelDevelopment
	if Context.disableUpdate {
		log.Info("AdGuard Home updates are disabled")
	}
}

// logIfUnsupported logs a formatted warning if the error is one of the
// unsupported errors and returns nil.  If err is nil, logIfUnsupported returns
// nil.  Otherwise, it returns err.
func logIfUnsupported(msg string, err error) (outErr error) {
	if errors.As(err, new(*aghos.UnsupportedError)) {
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
		return fmt.Errorf("initing hosts watcher: %w", err)
	}

	Context.etcHosts, err = aghnet.NewHostsContainer(
		filtering.SysHostsListID,
		aghos.RootDirFS(),
		hostsWatcher,
		aghnet.DefaultHostsPaths()...,
	)
	if err != nil {
		closeErr := hostsWatcher.Close()
		if errors.Is(err, aghnet.ErrNoHostsPaths) && closeErr == nil {
			log.Info("warning: initing hosts container: %s", err)

			return nil
		}

		return errors.WithDeferred(fmt.Errorf("initing hosts container: %w", err), closeErr)
	}

	return nil
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
func initContextClients() (err error) {
	err = setupDNSFilteringConf(config.DNS.DnsfilterConf)
	if err != nil {
		// Don't wrap the error, because it's informative enough as is.
		return err
	}

	//lint:ignore SA1019  Migration is not over.
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

	Context.updater = updater.NewUpdater(&updater.Config{
		Client:   Context.client,
		Version:  version.Version(),
		Channel:  version.Channel(),
		GOARCH:   runtime.GOARCH,
		GOOS:     runtime.GOOS,
		GOARM:    version.GOARM(),
		GOMIPS:   version.GOMIPS(),
		WorkDir:  Context.workDir,
		ConfName: config.getConfigFilename(),
	})

	var arpdb aghnet.ARPDB
	if config.Clients.Sources.ARP {
		arpdb = aghnet.NewARPDB()
	}

	err = Context.clients.Init(
		config.Clients.Persistent,
		Context.dhcpServer,
		Context.etcHosts,
		arpdb,
		config.DNS.DnsfilterConf,
	)
	if err != nil {
		// Don't wrap the error, because it's informative enough as is.
		return err
	}

	return nil
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
			uint16(opts.bindPort),
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
func setupDNSFilteringConf(conf *filtering.Config) (err error) {
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
	conf.ConfigModified = onConfigModified
	conf.HTTPRegister = httpRegister
	conf.DataDir = Context.getDataDir()
	conf.Filters = slices.Clone(config.Filters)
	conf.WhitelistFilters = slices.Clone(config.WhitelistFilters)
	conf.UserRules = slices.Clone(config.UserRules)
	conf.HTTPClient = Context.client

	cacheTime := time.Duration(conf.CacheTime) * time.Minute

	upsOpts := &upstream.Options{
		Timeout: dnsTimeout,
		ServerIPAddrs: []net.IP{
			{94, 140, 14, 15},
			{94, 140, 15, 16},
			net.ParseIP("2a10:50c0::bad1:ff"),
			net.ParseIP("2a10:50c0::bad2:ff"),
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

	parUps, err := upstream.AddressToUpstream(defaultParentalServer, upsOpts)
	if err != nil {
		return fmt.Errorf("converting parental server: %w", err)
	}

	conf.ParentalControlChecker = hashprefix.New(&hashprefix.Config{
		Upstream:    parUps,
		ServiceName: pcService,
		TXTSuffix:   pcTXTSuffix,
		CacheTime:   cacheTime,
		CacheSize:   conf.SafeBrowsingCacheSize,
	})

	conf.SafeSearchConf.CustomResolver = safeSearchResolver{}
	conf.SafeSearch, err = safesearch.NewDefault(
		conf.SafeSearchConf,
		"default",
		conf.SafeSearchCacheSize,
		cacheTime,
	)
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

func initWeb(opts options, clientBuildFS fs.FS) (web *webAPI, err error) {
	var clientFS fs.FS
	if opts.localFrontend {
		log.Info("warning: using local frontend files")

		clientFS = os.DirFS("build/static")
	} else {
		clientFS, err = fs.Sub(clientBuildFS, "build/static")
		if err != nil {
			return nil, fmt.Errorf("getting embedded client subdir: %w", err)
		}
	}

	webConf := webConfig{
		firstRun: Context.firstRun,
		BindHost: config.HTTPConfig.Address.Addr(),
		BindPort: int(config.HTTPConfig.Address.Port()),

		ReadTimeout:       readTimeout,
		ReadHeaderTimeout: readHdrTimeout,
		WriteTimeout:      writeTimeout,

		clientFS: clientFS,

		serveHTTP3: config.DNS.ServeHTTP3,
	}

	web = newWebAPI(&webConf)
	if web == nil {
		return nil, fmt.Errorf("initializing web: %w", err)
	}

	return web, nil
}

func fatalOnError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// run configures and starts AdGuard Home.
func run(opts options, clientBuildFS fs.FS) {
	// Configure config filename.
	initConfigFilename(opts)

	// Configure working dir and config path.
	err := initWorkingDir(opts)
	fatalOnError(err)

	// Configure log level and output.
	err = configureLogger(opts)
	fatalOnError(err)

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

	err = initContextClients()
	fatalOnError(err)

	err = setupOpts(opts)
	fatalOnError(err)

	// TODO(e.burkov): This could be made earlier, probably as the option's
	// effect.
	cmdlineUpdate(opts)

	if !Context.firstRun {
		// Save the updated config.
		err = config.write()
		fatalOnError(err)

		if config.DebugPProf {
			// TODO(a.garipov): Make the address configurable.
			startPprof("localhost:6060")
		}
	}

	dir := Context.getDataDir()
	err = os.MkdirAll(dir, 0o755)
	fatalOnError(errors.Annotate(err, "creating DNS data dir at %s: %w", dir))

	GLMode = opts.glinetMode

	// Init auth module.
	Context.auth, err = initUsers()
	fatalOnError(err)

	Context.tls, err = newTLSManager(config.TLS)
	if err != nil {
		log.Error("initializing tls: %s", err)
		onConfigModified()
	}

	Context.web, err = initWeb(opts, clientBuildFS)
	fatalOnError(err)

	if !Context.firstRun {
		err = initDNS()
		fatalOnError(err)

		Context.tls.start()

		go func() {
			sErr := startDNSServer()
			if sErr != nil {
				closeDNSServer()
				fatalOnError(sErr)
			}
		}()

		if Context.dhcpServer != nil {
			err = Context.dhcpServer.Start()
			if err != nil {
				log.Error("starting dhcp server: %s", err)
			}
		}
	}

	Context.web.start()

	// Wait indefinitely for other goroutines to complete their job.
	select {}
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

	sessionTTL := config.HTTPConfig.SessionTTL.Seconds()
	auth = InitAuth(sessFilename, config.Users, uint32(sessionTTL), rateLimiter)
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
func startMods() (err error) {
	err = initDNS()
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

// Check if the current user permissions are enough to run AdGuard Home
func checkPermissions() {
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
// overridden by command-line arguments, or is set to default.
func initConfigFilename(opts options) {
	Context.configFilename = stringutil.Coalesce(opts.confFilename, "AdGuardHome.yaml")
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

// configureLogger configures logger level and output.
func configureLogger(opts options) (err error) {
	ls := getLogSettings(opts)

	// Configure logger level.
	if ls.Verbose {
		log.SetLevel(log.DEBUG)
	}

	// Make sure that we see the microseconds in logs, as networking stuff can
	// happen pretty quickly.
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	// Write logs to stdout by default.
	if ls.File == "" {
		return nil
	}

	if ls.File == configSyslog {
		// Use syslog where it is possible and eventlog on Windows.
		err = aghos.ConfigureSyslog(serviceName)
		if err != nil {
			return fmt.Errorf("cannot initialize syslog: %w", err)
		}

		return nil
	}

	logFilePath := ls.File
	if !filepath.IsAbs(logFilePath) {
		logFilePath = filepath.Join(Context.workDir, logFilePath)
	}

	log.SetOutput(&lumberjack.Logger{
		Filename:   logFilePath,
		Compress:   ls.Compress,
		LocalTime:  ls.LocalTime,
		MaxBackups: ls.MaxBackups,
		MaxSize:    ls.MaxSize,
		MaxAge:     ls.MaxAge,
	})

	return nil
}

// getLogSettings returns a log settings object properly initialized from opts.
func getLogSettings(opts options) (ls *logSettings) {
	ls = readLogSettings()
	configLogSettings := config.Log

	// Command-line arguments can override config settings.
	if opts.verbose || configLogSettings.Verbose {
		ls.Verbose = true
	}

	ls.File = stringutil.Coalesce(opts.logFile, configLogSettings.File, ls.File)

	// Handle default log settings overrides.
	ls.Compress = configLogSettings.Compress
	ls.LocalTime = configLogSettings.LocalTime
	ls.MaxBackups = configLogSettings.MaxBackups
	ls.MaxSize = configLogSettings.MaxSize
	ls.MaxAge = configLogSettings.MaxAge

	if opts.runningAsService && ls.File == "" && runtime.GOOS == "windows" {
		// When running as a Windows service, use eventlog by default if
		// nothing else is configured.  Otherwise, we'll lose the log output.
		ls.File = configSyslog
	}

	return ls
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

		os.Exit(0)
	}

	return opts
}

// printWebAddrs prints addresses built from proto, addr, and an appropriate
// port.  At least one address is printed with the value of port.  Output
// example:
//
//	go to http://127.0.0.1:80
func printWebAddrs(proto, addr string, port int) {
	log.Printf("go to %s://%s", proto, netutil.JoinHostPort(addr, port))
}

// printHTTPAddresses prints the IP addresses which user can use to access the
// admin interface.  proto is either schemeHTTP or schemeHTTPS.
func printHTTPAddresses(proto string) {
	tlsConf := tlsConfigSettings{}
	if Context.tls != nil {
		Context.tls.WriteDiskConfig(&tlsConf)
	}

	port := int(config.HTTPConfig.Address.Port())
	if proto == aghhttp.SchemeHTTPS {
		port = tlsConf.PortHTTPS
	}

	// TODO(e.burkov): Inspect and perhaps merge with the previous condition.
	if proto == aghhttp.SchemeHTTPS && tlsConf.ServerName != "" {
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

// -------------------
// first run / install
// -------------------
func detectFirstRun() bool {
	configfile := Context.configFilename
	if !filepath.IsAbs(configfile) {
		configfile = filepath.Join(Context.workDir, Context.configFilename)
	}
	_, err := os.Stat(configfile)
	return errors.Is(err, os.ErrNotExist)
}

// Connect to a remote server resolving hostname using our own DNS server.
//
// TODO(e.burkov): This messy logic should be decomposed and clarified.
//
// TODO(a.garipov): Support network.
func customDialContext(ctx context.Context, network, addr string) (conn net.Conn, err error) {
	log.Debug("home: customdial: dialing addr %q for network %s", addr, network)

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	dialer := &net.Dialer{
		Timeout: time.Minute * 5,
	}

	if net.ParseIP(host) != nil || config.DNS.Port == 0 {
		return dialer.DialContext(ctx, network, addr)
	}

	addrs, err := Context.dnsServer.Resolve(host)
	if err != nil {
		return nil, fmt.Errorf("resolving %q: %w", host, err)
	}

	log.Debug("dnsServer.Resolve: %q: %v", host, addrs)

	if len(addrs) == 0 {
		return nil, fmt.Errorf("couldn't lookup host: %q", host)
	}

	var dialErrs []error
	for _, a := range addrs {
		addr = net.JoinHostPort(a.String(), port)
		conn, err = dialer.DialContext(ctx, network, addr)
		if err != nil {
			dialErrs = append(dialErrs, err)

			continue
		}

		return conn, err
	}

	return nil, errors.List(fmt.Sprintf("couldn't dial to %s", addr), dialErrs...)
}

func getHTTPProxy(_ *http.Request) (*url.URL, error) {
	if config.ProxyURL == "" {
		return nil, nil
	}

	return url.Parse(config.ProxyURL)
}

// jsonError is a generic JSON error response.
//
// TODO(a.garipov): Merge together with the implementations in [dhcpd] and other
// packages after refactoring the web handler registering.
type jsonError struct {
	// Message is the error message, an opaque string.
	Message string `json:"message"`
}

// cmdlineUpdate updates current application and exits.
func cmdlineUpdate(opts options) {
	if !opts.performUpdate {
		return
	}

	// Initialize the DNS server to use the internal resolver which the updater
	// needs to be able to resolve the update source hostname.
	//
	// TODO(e.burkov):  We could probably initialize the internal resolver
	// separately.
	err := initDNSServer(nil, nil, nil, nil, nil, nil, &tlsConfigSettings{})
	fatalOnError(err)

	log.Info("cmdline update: performing update")

	updater := Context.updater
	info, err := updater.VersionInfo(true)
	if err != nil {
		vcu := updater.VersionCheckURL()
		log.Error("getting version info from %s: %s", vcu, err)

		os.Exit(1)
	}

	if info.NewVersion == version.Version() {
		log.Info("no updates available")

		os.Exit(0)
	}

	err = updater.Update(Context.firstRun)
	fatalOnError(err)

	err = restartService()
	if err != nil {
		log.Debug("restarting service: %s", err)
		log.Info("AdGuard Home was not installed as a service. " +
			"Please restart running instances of AdGuardHome manually.")
	}

	os.Exit(0)
}
