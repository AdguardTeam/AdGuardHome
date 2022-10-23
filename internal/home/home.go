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
	"net/http/pprof"
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
	"github.com/AdguardTeam/AdGuardHome/internal/querylog"
	"github.com/AdguardTeam/AdGuardHome/internal/stats"
	"github.com/AdguardTeam/AdGuardHome/internal/updater"
	"github.com/AdguardTeam/AdGuardHome/internal/version"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil"
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
	whois      *WHOIS               // WHOIS module
	dhcpServer dhcpd.Interface      // DHCP module
	auth       *Auth                // HTTP authentication module
	filters    *filtering.DNSFilter // DNS filtering module
	web        *Web                 // Web (HTTP, HTTPS) module
	tls        *tlsManager          // TLS module
	// etcHosts is an IP-hostname pairs set taken from system configuration
	// (e.g. /etc/hosts) files.
	etcHosts *aghnet.HostsContainer
	// hostsWatcher is the watcher to detect changes in the hosts files.
	hostsWatcher aghos.FSWatcher

	updater *updater.Updater

	// mux is our custom http.ServeMux.
	mux *http.ServeMux

	// Runtime properties
	// --

	configFilename   string // Config filename (can be overridden via the command line arguments)
	workDir          string // Location of our directory, used to protect against CWD being somewhere else
	firstRun         bool   // if set to true, don't run any services except HTTP web interface, and serve only first-run html
	pidFileName      string // PID file name.  Empty if no PID file was created.
	disableUpdate    bool   // If set, don't check for updates
	controlLock      sync.Mutex
	tlsRoots         *x509.CertPool // list of root CAs for TLSv1.2
	transport        *http.Transport
	client           *http.Client
	appSignalChannel chan os.Signal // Channel for receiving OS signals by the console app

	// tlsCipherIDs are the ID of the cipher suites that AdGuard Home must use.
	tlsCipherIDs []uint16

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
				Context.clients.Reload()
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

func setupContext(opts options) {
	setupContextFlags(opts)

	switch version.Channel() {
	case version.ChannelEdge, version.ChannelDevelopment:
		config.BetaBindPort = 3001
	default:
		// Go on.
	}

	Context.tlsRoots = aghtls.SystemRootCAs()
	Context.transport = &http.Transport{
		DialContext: customDialContext,
		Proxy:       getHTTPProxy,
		TLSClientConfig: &tls.Config{
			RootCAs:      Context.tlsRoots,
			CipherSuites: Context.tlsCipherIDs,
			MinVersion:   tls.VersionTLS12,
		},
	}
	Context.client = &http.Client{
		Timeout:   time.Minute * 5,
		Transport: Context.transport,
	}

	if !Context.firstRun {
		// Do the upgrade if necessary.
		err := upgradeConfig()
		fatalOnError(err)

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
			fatalOnError(err)
		}
	}

	Context.mux = http.NewServeMux()
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
	Context.hostsWatcher, err = aghos.NewOSWritesWatcher()
	if err != nil {
		return fmt.Errorf("initing hosts watcher: %w", err)
	}

	Context.etcHosts, err = aghnet.NewHostsContainer(
		filtering.SysHostsListID,
		aghos.RootDirFS(),
		Context.hostsWatcher,
		aghnet.DefaultHostsPaths()...,
	)
	if err != nil {
		cerr := Context.hostsWatcher.Close()
		if errors.Is(err, aghnet.ErrNoHostsPaths) && cerr == nil {
			log.Info("warning: initing hosts container: %s", err)

			return nil
		}

		return errors.WithDeferred(fmt.Errorf("initing hosts container: %w", err), cerr)
	}

	return nil
}

func setupConfig(opts options) (err error) {
	config.DNS.DnsfilterConf.EtcHosts = Context.etcHosts
	config.DNS.DnsfilterConf.ConfigModified = onConfigModified
	config.DNS.DnsfilterConf.HTTPRegister = httpRegister
	config.DNS.DnsfilterConf.DataDir = Context.getDataDir()
	config.DNS.DnsfilterConf.Filters = slices.Clone(config.Filters)
	config.DNS.DnsfilterConf.WhitelistFilters = slices.Clone(config.WhitelistFilters)
	config.DNS.DnsfilterConf.UserRules = slices.Clone(config.UserRules)
	config.DNS.DnsfilterConf.HTTPClient = Context.client

	config.DHCP.WorkDir = Context.workDir
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

	Context.clients.Init(config.Clients.Persistent, Context.dhcpServer, Context.etcHosts, arpdb)

	if opts.bindPort != 0 {
		tcpPorts := aghalg.UniqChecker[tcpPort]{}
		addPorts(tcpPorts, tcpPort(opts.bindPort), tcpPort(config.BetaBindPort))

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

		config.BindPort = opts.bindPort
	}

	// override bind host/port from the console
	if opts.bindHost.IsValid() {
		config.BindHost = opts.bindHost
	}
	if len(opts.pidFile) != 0 && writePIDFile(opts.pidFile) {
		Context.pidFileName = opts.pidFile
	}

	return nil
}

func initWeb(opts options, clientBuildFS fs.FS) (web *Web, err error) {
	var clientFS, clientBetaFS fs.FS
	if opts.localFrontend {
		log.Info("warning: using local frontend files")

		clientFS = os.DirFS("build/static")
		clientBetaFS = os.DirFS("build2/static")
	} else {
		clientFS, err = fs.Sub(clientBuildFS, "build/static")
		if err != nil {
			return nil, fmt.Errorf("getting embedded client subdir: %w", err)
		}

		clientBetaFS, err = fs.Sub(clientBuildFS, "build2/static")
		if err != nil {
			return nil, fmt.Errorf("getting embedded beta client subdir: %w", err)
		}
	}

	webConf := webConfig{
		firstRun:     Context.firstRun,
		BindHost:     config.BindHost,
		BindPort:     config.BindPort,
		BetaBindPort: config.BetaBindPort,

		ReadTimeout:       readTimeout,
		ReadHeaderTimeout: readHdrTimeout,
		WriteTimeout:      writeTimeout,

		clientFS:     clientFS,
		clientBetaFS: clientBetaFS,

		serveHTTP3: config.DNS.ServeHTTP3,
	}

	web = newWeb(&webConf)
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
	// configure config filename
	initConfigFilename(opts)

	// configure working dir and config path
	initWorkingDir(opts)

	// configure log level and output
	configureLogger(opts)

	// Print the first message after logger is configured.
	log.Info(version.Full())
	log.Debug("current working directory is %s", Context.workDir)
	if opts.runningAsService {
		log.Info("AdGuard Home is running as a service")
	}

	setupContext(opts)

	err := configureOS(config)
	fatalOnError(err)

	// clients package uses filtering package's static data (filtering.BlockedSvcKnown()),
	//  so we have to initialize filtering's static data first,
	//  but also avoid relying on automatic Go init() function
	filtering.InitModule()

	err = setupConfig(opts)
	fatalOnError(err)

	if !Context.firstRun {
		// Save the updated config
		err = config.write()
		fatalOnError(err)

		if config.DebugPProf {
			mux := http.NewServeMux()
			mux.HandleFunc("/debug/pprof/", pprof.Index)
			mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
			mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
			mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
			mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
			go func() {
				log.Info("pprof: listening on localhost:6060")
				lerr := http.ListenAndServe("localhost:6060", mux)
				log.Error("Error while running the pprof server: %s", lerr)
			}()
		}
	}

	err = os.MkdirAll(Context.getDataDir(), 0o755)
	if err != nil {
		log.Fatalf("Cannot create DNS data dir at %s: %s", Context.getDataDir(), err)
	}

	sessFilename := filepath.Join(Context.getDataDir(), "sessions.db")
	GLMode = opts.glinetMode
	var rateLimiter *authRateLimiter
	if config.AuthAttempts > 0 && config.AuthBlockMin > 0 {
		rateLimiter = newAuthRateLimiter(
			time.Duration(config.AuthBlockMin)*time.Minute,
			config.AuthAttempts,
		)
	} else {
		log.Info("authratelimiter is disabled")
	}

	Context.auth = InitAuth(
		sessFilename,
		config.Users,
		config.WebSessionTTLHours*60*60,
		rateLimiter,
	)
	if Context.auth == nil {
		log.Fatalf("Couldn't initialize Auth module")
	}
	config.Users = nil

	Context.tls, err = newTLSManager(config.TLS)
	if err != nil {
		log.Fatalf("initializing tls: %s", err)
	}

	Context.web, err = initWeb(opts, clientBuildFS)
	fatalOnError(err)

	if !Context.firstRun {
		err = initDNSServer()
		fatalOnError(err)

		Context.tls.start()

		go func() {
			serr := startDNSServer()
			if serr != nil {
				closeDNSServer()
				fatalOnError(serr)
			}
		}()

		if Context.dhcpServer != nil {
			err = Context.dhcpServer.Start()
			if err != nil {
				log.Error("starting dhcp server: %s", err)
			}
		}
	}

	Context.web.Start()

	// wait indefinitely for other go-routines to complete their job
	select {}
}

// startMods initializes and starts the DNS server after installation.
func startMods() error {
	err := initDNSServer()
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
	err := aghnet.CheckPort("tcp", netip.AddrPortFrom(aghnet.IPv4Localhost(), defaultPortDNS))
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

func initConfigFilename(opts options) {
	// config file path can be overridden by command-line arguments:
	if opts.confFilename != "" {
		Context.configFilename = opts.confFilename
	} else {
		// Default config file name
		Context.configFilename = "AdGuardHome.yaml"
	}
}

// initWorkingDir initializes the workDir
// if no command-line arguments specified, we use the directory where our binary file is located
func initWorkingDir(opts options) {
	execPath, err := os.Executable()
	if err != nil {
		panic(err)
	}

	if opts.workDir != "" {
		// If there is a custom config file, use it's directory as our working dir
		Context.workDir = opts.workDir
	} else {
		Context.workDir = filepath.Dir(execPath)
	}

	workDir, err := filepath.EvalSymlinks(Context.workDir)
	if err != nil {
		panic(err)
	}

	Context.workDir = workDir
}

// configureLogger configures logger level and output
func configureLogger(opts options) {
	ls := getLogSettings()

	// command-line arguments can override config settings
	if opts.verbose || config.Verbose {
		ls.Verbose = true
	}
	if opts.logFile != "" {
		ls.File = opts.logFile
	} else if config.File != "" {
		ls.File = config.File
	}

	// Handle default log settings overrides
	ls.Compress = config.Compress
	ls.LocalTime = config.LocalTime
	ls.MaxBackups = config.MaxBackups
	ls.MaxSize = config.MaxSize
	ls.MaxAge = config.MaxAge

	// log.SetLevel(log.INFO) - default
	if ls.Verbose {
		log.SetLevel(log.DEBUG)
	}

	// Make sure that we see the microseconds in logs, as networking stuff can
	// happen pretty quickly.
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	if opts.runningAsService && ls.File == "" && runtime.GOOS == "windows" {
		// When running as a Windows service, use eventlog by default if nothing
		// else is configured.  Otherwise, we'll simply lose the log output.
		ls.File = configSyslog
	}

	// logs are written to stdout (default)
	if ls.File == "" {
		return
	}

	if ls.File == configSyslog {
		// Use syslog where it is possible and eventlog on Windows
		err := aghos.ConfigureSyslog(serviceName)
		if err != nil {
			log.Fatalf("cannot initialize syslog: %s", err)
		}
	} else {
		logFilePath := ls.File
		if !filepath.IsAbs(logFilePath) {
			logFilePath = filepath.Join(Context.workDir, logFilePath)
		}

		log.SetOutput(&lumberjack.Logger{
			Filename:   logFilePath,
			Compress:   ls.Compress, // disabled by default
			LocalTime:  ls.LocalTime,
			MaxBackups: ls.MaxBackups,
			MaxSize:    ls.MaxSize, // megabytes
			MaxAge:     ls.MaxAge,  // days
		})
	}
}

// cleanup stops and resets all the modules.
func cleanup(ctx context.Context) {
	log.Info("stopping AdGuard Home")

	if Context.web != nil {
		Context.web.Close(ctx)
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
		// Currently Context.hostsWatcher is only used in Context.etcHosts and
		// needs closing only in case of the successful initialization of
		// Context.etcHosts.
		if err = Context.hostsWatcher.Close(); err != nil {
			log.Error("closing hosts watcher: %s", err)
		}

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
// port.  At least one address is printed with the value of port.  If the value
// of betaPort is 0, the second address is not printed.  Output example:
//
//	Go to http://127.0.0.1:80
//	Go to http://127.0.0.1:3000 (BETA)
func printWebAddrs(proto, addr string, port, betaPort int) {
	const (
		hostMsg     = "Go to %s://%s"
		hostBetaMsg = hostMsg + " (BETA)"
	)

	log.Printf(hostMsg, proto, netutil.JoinHostPort(addr, port))
	if betaPort == 0 {
		return
	}

	log.Printf(hostBetaMsg, proto, netutil.JoinHostPort(addr, config.BetaBindPort))
}

// printHTTPAddresses prints the IP addresses which user can use to access the
// admin interface.  proto is either schemeHTTP or schemeHTTPS.
func printHTTPAddresses(proto string) {
	tlsConf := tlsConfigSettings{}
	if Context.tls != nil {
		Context.tls.WriteDiskConfig(&tlsConf)
	}

	port := config.BindPort
	if proto == aghhttp.SchemeHTTPS {
		port = tlsConf.PortHTTPS
	}

	// TODO(e.burkov): Inspect and perhaps merge with the previous condition.
	if proto == aghhttp.SchemeHTTPS && tlsConf.ServerName != "" {
		printWebAddrs(proto, tlsConf.ServerName, tlsConf.PortHTTPS, 0)

		return
	}

	bindhost := config.BindHost
	if !bindhost.IsUnspecified() {
		printWebAddrs(proto, bindhost.String(), port, config.BetaBindPort)

		return
	}

	ifaces, err := aghnet.GetValidNetInterfacesForWeb()
	if err != nil {
		log.Error("web: getting iface ips: %s", err)
		// That's weird, but we'll ignore it.
		//
		// TODO(e.burkov): Find out when it happens.
		printWebAddrs(proto, bindhost.String(), port, config.BetaBindPort)

		return
	}

	for _, iface := range ifaces {
		for _, addr := range iface.Addresses {
			printWebAddrs(proto, addr.String(), config.BindPort, config.BetaBindPort)
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
func customDialContext(ctx context.Context, network, addr string) (conn net.Conn, err error) {
	log.Tracef("network:%v  addr:%v", network, addr)

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
	log.Debug("dnsServer.Resolve: %s: %v", host, addrs)
	if err != nil {
		return nil, err
	}

	if len(addrs) == 0 {
		return nil, fmt.Errorf("couldn't lookup host: %s", host)
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
// TODO(a.garipov): Merge together with the implementations in .../dhcpd and
// other packages after refactoring the web handler registering.
type jsonError struct {
	// Message is the error message, an opaque string.
	Message string `json:"message"`
}
