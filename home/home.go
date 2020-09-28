package home

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/AdguardTeam/AdGuardHome/update"
	"github.com/AdguardTeam/AdGuardHome/util"

	"github.com/joomcode/errorx"

	"github.com/AdguardTeam/AdGuardHome/isdelve"

	"github.com/AdguardTeam/AdGuardHome/dhcpd"
	"github.com/AdguardTeam/AdGuardHome/dnsfilter"
	"github.com/AdguardTeam/AdGuardHome/dnsforward"
	"github.com/AdguardTeam/AdGuardHome/querylog"
	"github.com/AdguardTeam/AdGuardHome/stats"
	"github.com/AdguardTeam/golibs/log"
)

const (
	// Used in config to indicate that syslog or eventlog (win) should be used for logger output
	configSyslog = "syslog"
)

// Update-related variables
var (
	versionString   = "dev"
	updateChannel   = "none"
	versionCheckURL = ""
	ARMVersion      = ""
)

// Global context
type homeContext struct {
	// Modules
	// --

	clients    clientsContainer     // per-client-settings module
	stats      stats.Stats          // statistics module
	queryLog   querylog.QueryLog    // query log module
	dnsServer  *dnsforward.Server   // DNS module
	rdns       *RDNS                // rDNS module
	whois      *Whois               // WHOIS module
	dnsFilter  *dnsfilter.Dnsfilter // DNS filtering module
	dhcpServer *dhcpd.Server        // DHCP module
	auth       *Auth                // HTTP authentication module
	filters    Filtering            // DNS filtering module
	web        *Web                 // Web (HTTP, HTTPS) module
	tls        *TLSMod              // TLS module
	autoHosts  util.AutoHosts       // IP-hostname pairs taken from system configuration (e.g. /etc/hosts) files
	updater    *update.Updater

	// Runtime properties
	// --

	configFilename   string // Config filename (can be overridden via the command line arguments)
	workDir          string // Location of our directory, used to protect against CWD being somewhere else
	firstRun         bool   // if set to true, don't run any services except HTTP web inteface, and serve only first-run html
	pidFileName      string // PID file name.  Empty if no PID file was created.
	disableUpdate    bool   // If set, don't check for updates
	controlLock      sync.Mutex
	tlsRoots         *x509.CertPool // list of root CAs for TLSv1.2
	tlsCiphers       []uint16       // list of TLS ciphers to use
	transport        *http.Transport
	client           *http.Client
	appSignalChannel chan os.Signal // Channel for receiving OS signals by the console app
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
func Main(version string, channel string, armVer string) {
	// Init update-related global variables
	versionString = version
	updateChannel = channel
	ARMVersion = armVer
	versionCheckURL = "https://static.adguard.com/adguardhome/" + updateChannel + "/version.json"

	// config can be specified, which reads options from there, but other command line flags have to override config values
	// therefore, we must do it manually instead of using a lib
	args := loadOptions()

	Context.appSignalChannel = make(chan os.Signal)
	signal.Notify(Context.appSignalChannel, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)
	go func() {
		for {
			sig := <-Context.appSignalChannel
			log.Info("Received signal '%s'", sig)
			switch sig {
			case syscall.SIGHUP:
				Context.clients.Reload()
				Context.tls.Reload()

			default:
				cleanup()
				cleanupAlways()
				os.Exit(0)
			}
		}
	}()

	if args.serviceControlAction != "" {
		handleServiceControlAction(args)
		return
	}

	// run the protection
	run(args)
}

// version - returns the current version string
func version() string {
	msg := "AdGuard Home, version %s, channel %s, arch %s %s"
	if ARMVersion != "" {
		msg = msg + " v" + ARMVersion
	}
	return fmt.Sprintf(msg, versionString, updateChannel, runtime.GOOS, runtime.GOARCH)
}

// run initializes configuration and runs the AdGuard Home
// run is a blocking method!
// nolint
func run(args options) {
	// configure config filename
	initConfigFilename(args)

	// configure working dir and config path
	initWorkingDir(args)

	// configure log level and output
	configureLogger(args)

	// Go memory hacks
	memoryUsage(args)

	// print the first message after logger is configured
	log.Println(version())
	log.Debug("Current working directory is %s", Context.workDir)
	if args.runningAsService {
		log.Info("AdGuard Home is running as a service")
	}
	Context.runningAsService = args.runningAsService
	Context.disableUpdate = args.disableUpdate

	Context.firstRun = detectFirstRun()
	if Context.firstRun {
		log.Info("This is the first time AdGuard Home is launched")
		checkPermissions()
	}

	initConfig()

	Context.tlsRoots = util.LoadSystemRootCAs()
	Context.tlsCiphers = util.InitTLSCiphers()
	Context.transport = &http.Transport{
		DialContext: customDialContext,
		Proxy:       getHTTPProxy,
		TLSClientConfig: &tls.Config{
			RootCAs: Context.tlsRoots,
		},
	}
	Context.client = &http.Client{
		Timeout:   time.Minute * 5,
		Transport: Context.transport,
	}

	if !Context.firstRun {
		// Do the upgrade if necessary
		err := upgradeConfig()
		if err != nil {
			log.Fatal(err)
		}

		err = parseConfig()
		if err != nil {
			log.Error("Failed to parse configuration, exiting")
			os.Exit(1)
		}

		if args.checkConfig {
			log.Info("Configuration file is OK")
			os.Exit(0)
		}
	}

	// 'clients' module uses 'dnsfilter' module's static data (dnsfilter.BlockedSvcKnown()),
	//  so we have to initialize dnsfilter's static data first,
	//  but also avoid relying on automatic Go init() function
	dnsfilter.InitModule()

	config.DHCP.WorkDir = Context.workDir
	config.DHCP.HTTPRegister = httpRegister
	config.DHCP.ConfigModified = onConfigModified
	if runtime.GOOS != "windows" {
		Context.dhcpServer = dhcpd.Create(config.DHCP)
		if Context.dhcpServer == nil {
			log.Fatalf("Can't initialize DHCP module")
		}
	}
	Context.autoHosts.Init("")

	Context.updater = update.NewUpdater(update.Config{
		Client:        Context.client,
		WorkDir:       Context.workDir,
		VersionURL:    versionCheckURL,
		VersionString: versionString,
		OS:            runtime.GOOS,
		Arch:          runtime.GOARCH,
		ARMVersion:    ARMVersion,
		ConfigName:    config.getConfigFilename(),
	})

	Context.clients.Init(config.Clients, Context.dhcpServer, &Context.autoHosts)
	config.Clients = nil

	if (runtime.GOOS == "linux" || runtime.GOOS == "darwin") &&
		config.RlimitNoFile != 0 {
		util.SetRlimit(config.RlimitNoFile)
	}

	// override bind host/port from the console
	if args.bindHost != "" {
		config.BindHost = args.bindHost
	}
	if args.bindPort != 0 {
		config.BindPort = args.bindPort
	}
	if len(args.pidFile) != 0 && writePIDFile(args.pidFile) {
		Context.pidFileName = args.pidFile
	}

	if !Context.firstRun {
		// Save the updated config
		err := config.write()
		if err != nil {
			log.Fatal(err)
		}

		if config.DebugPProf {
			mux := http.NewServeMux()
			util.PProfRegisterWebHandlers(mux)
			go func() {
				log.Info("pprof: listening on localhost:6060")
				err := http.ListenAndServe("localhost:6060", mux)
				log.Error("Error while running the pprof server: %s", err)
			}()
		}
	}

	err := os.MkdirAll(Context.getDataDir(), 0755)
	if err != nil {
		log.Fatalf("Cannot create DNS data dir at %s: %s", Context.getDataDir(), err)
	}

	sessFilename := filepath.Join(Context.getDataDir(), "sessions.db")
	GLMode = args.glinetMode
	Context.auth = InitAuth(sessFilename, config.Users, config.WebSessionTTLHours*60*60)
	if Context.auth == nil {
		log.Fatalf("Couldn't initialize Auth module")
	}
	config.Users = nil

	Context.tls = tlsCreate(config.TLS)
	if Context.tls == nil {
		log.Fatalf("Can't initialize TLS module")
	}

	webConf := WebConfig{
		firstRun: Context.firstRun,
		BindHost: config.BindHost,
		BindPort: config.BindPort,
	}
	Context.web = CreateWeb(&webConf)
	if Context.web == nil {
		log.Fatalf("Can't initialize Web module")
	}

	if !Context.firstRun {
		err := initDNSServer()
		if err != nil {
			log.Fatalf("%s", err)
		}
		Context.tls.Start()
		Context.autoHosts.Start()

		go func() {
			err := startDNSServer()
			if err != nil {
				log.Fatal(err)
			}
		}()

		if Context.dhcpServer != nil {
			_ = Context.dhcpServer.Start()
		}
	}

	Context.web.Start()

	// wait indefinitely for other go-routines to complete their job
	select {}
}

// StartMods - initialize and start DNS after installation
func StartMods() error {
	err := initDNSServer()
	if err != nil {
		return err
	}

	Context.tls.Start()

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

	if runtime.GOOS == "windows" {
		// On Windows we need to have admin rights to run properly

		admin, _ := util.HaveAdminRights()
		if //noinspection ALL
		admin || isdelve.Enabled {
			// Don't forget that for this to work you need to add "delve" tag explicitly
			// https://stackoverflow.com/questions/47879070/how-can-i-see-if-the-goland-debugger-is-running-in-the-program
			return
		}

		log.Fatal("This is the first launch of AdGuard Home. You must run it as Administrator.")
	}

	// We should check if AdGuard Home is able to bind to port 53
	ok, err := util.CanBindPort(53)

	if ok {
		log.Info("AdGuard Home can bind to port 53")
		return
	}

	if opErr, ok := err.(*net.OpError); ok {
		if sysErr, ok := opErr.Err.(*os.SyscallError); ok {
			if errno, ok := sysErr.Err.(syscall.Errno); ok && errno == syscall.EACCES {
				msg := `Permission check failed.

AdGuard Home is not allowed to bind to privileged ports (for instance, port 53).
Please note, that this is crucial for a server to be able to use privileged ports.

You have two options:
1. Run AdGuard Home with root privileges
2. On Linux you can grant the CAP_NET_BIND_SERVICE capability:
https://github.com/AdguardTeam/AdGuardHome/wiki/Getting-Started#running-without-superuser`

				log.Fatal(msg)
			}
		}
	}

	msg := fmt.Sprintf(`AdGuard failed to bind to port 53 due to %v

Please note, that this is crucial for a DNS server to be able to use that port.`, err)

	log.Info(msg)
}

// Write PID to a file
func writePIDFile(fn string) bool {
	data := fmt.Sprintf("%d", os.Getpid())
	err := ioutil.WriteFile(fn, []byte(data), 0644)
	if err != nil {
		log.Error("Couldn't write PID to file %s: %v", fn, err)
		return false
	}
	return true
}

func initConfigFilename(args options) {
	// config file path can be overridden by command-line arguments:
	if args.configFilename != "" {
		Context.configFilename = args.configFilename
	} else {
		// Default config file name
		Context.configFilename = "AdGuardHome.yaml"
	}
}

// initWorkingDir initializes the workDir
// if no command-line arguments specified, we use the directory where our binary file is located
func initWorkingDir(args options) {
	execPath, err := os.Executable()
	if err != nil {
		panic(err)
	}

	if args.workDir != "" {
		// If there is a custom config file, use it's directory as our working dir
		Context.workDir = args.workDir
	} else {
		Context.workDir = filepath.Dir(execPath)
	}
}

// configureLogger configures logger level and output
func configureLogger(args options) {
	ls := getLogSettings()

	// command-line arguments can override config settings
	if args.verbose || config.Verbose {
		ls.Verbose = true
	}
	if args.logFile != "" {
		ls.LogFile = args.logFile
	} else if config.LogFile != "" {
		ls.LogFile = config.LogFile
	}

	// Handle default log settings overrides
	ls.LogCompress = config.LogCompress
	ls.LogLocalTime = config.LogLocalTime
	ls.LogMaxBackups = config.LogMaxBackups
	ls.LogMaxSize = config.LogMaxSize
	ls.LogMaxAge = config.LogMaxAge

	// log.SetLevel(log.INFO) - default
	if ls.Verbose {
		log.SetLevel(log.DEBUG)
	}

	if args.runningAsService && ls.LogFile == "" && runtime.GOOS == "windows" {
		// When running as a Windows service, use eventlog by default if nothing else is configured
		// Otherwise, we'll simply loose the log output
		ls.LogFile = configSyslog
	}

	// logs are written to stdout (default)
	if ls.LogFile == "" {
		return
	}

	if ls.LogFile == configSyslog {
		// Use syslog where it is possible and eventlog on Windows
		err := util.ConfigureSyslog(serviceName)
		if err != nil {
			log.Fatalf("cannot initialize syslog: %s", err)
		}
	} else {
		logFilePath := filepath.Join(Context.workDir, ls.LogFile)
		if filepath.IsAbs(ls.LogFile) {
			logFilePath = ls.LogFile
		}

		_, err := os.OpenFile(logFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			log.Fatalf("cannot create a log file: %s", err)
		}

		log.SetOutput(&lumberjack.Logger{
			Filename:   logFilePath,
			Compress:   ls.LogCompress, // disabled by default
			LocalTime:  ls.LogLocalTime,
			MaxBackups: ls.LogMaxBackups,
			MaxSize:    ls.LogMaxSize, // megabytes
			MaxAge:     ls.LogMaxAge,  //days
		})
	}
}

func cleanup() {
	log.Info("Stopping AdGuard Home")

	if Context.web != nil {
		Context.web.Close()
		Context.web = nil
	}
	if Context.auth != nil {
		Context.auth.Close()
		Context.auth = nil
	}

	err := stopDNSServer()
	if err != nil {
		log.Error("Couldn't stop DNS server: %s", err)
	}

	if Context.dhcpServer != nil {
		Context.dhcpServer.Stop()
	}

	Context.autoHosts.Close()

	if Context.tls != nil {
		Context.tls.Close()
		Context.tls = nil
	}
}

// This function is called before application exits
func cleanupAlways() {
	if len(Context.pidFileName) != 0 {
		_ = os.Remove(Context.pidFileName)
	}
	log.Info("Stopped")
}

func exitWithError() {
	os.Exit(64)
}

// loadOptions reads command line arguments and initializes configuration
func loadOptions() options {
	o, f, err := parse(os.Args[0], os.Args[1:])

	if err != nil {
		log.Error(err.Error())
		_ = printHelp(os.Args[0])
		exitWithError()
	} else if f != nil {
		err = f()
		if err != nil {
			log.Error(err.Error())
			exitWithError()
		} else {
			os.Exit(0)
		}
	}

	return o
}

// prints IP addresses which user can use to open the admin interface
// proto is either "http" or "https"
func printHTTPAddresses(proto string) {
	var address string

	tlsConf := tlsConfigSettings{}
	if Context.tls != nil {
		Context.tls.WriteDiskConfig(&tlsConf)
	}

	port := strconv.Itoa(config.BindPort)
	if proto == "https" {
		port = strconv.Itoa(tlsConf.PortHTTPS)
	}

	if proto == "https" && tlsConf.ServerName != "" {
		if tlsConf.PortHTTPS == 443 {
			log.Printf("Go to https://%s", tlsConf.ServerName)
		} else {
			log.Printf("Go to https://%s:%s", tlsConf.ServerName, port)
		}
	} else if config.BindHost == "0.0.0.0" {
		log.Println("AdGuard Home is available on the following addresses:")
		ifaces, err := util.GetValidNetInterfacesForWeb()
		if err != nil {
			// That's weird, but we'll ignore it
			address = net.JoinHostPort(config.BindHost, port)
			log.Printf("Go to %s://%s", proto, address)
			return
		}

		for _, iface := range ifaces {
			for _, addr := range iface.Addresses {
				address = net.JoinHostPort(addr, strconv.Itoa(config.BindPort))
				log.Printf("Go to %s://%s", proto, address)
			}
		}
	} else {
		address = net.JoinHostPort(config.BindHost, port)
		log.Printf("Go to %s://%s", proto, address)
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
	if !os.IsNotExist(err) {
		// do nothing, file exists
		return false
	}
	return true
}

// Connect to a remote server resolving hostname using our own DNS server
func customDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	log.Tracef("network:%v  addr:%v", network, addr)

	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, err
	}

	dialer := &net.Dialer{
		Timeout: time.Minute * 5,
	}

	if net.ParseIP(host) != nil || config.DNS.Port == 0 {
		con, err := dialer.DialContext(ctx, network, addr)
		return con, err
	}

	addrs, e := Context.dnsServer.Resolve(host)
	log.Debug("dnsServer.Resolve: %s: %v", host, addrs)
	if e != nil {
		return nil, e
	}

	if len(addrs) == 0 {
		return nil, fmt.Errorf("couldn't lookup host: %s", host)
	}

	var dialErrs []error
	for _, a := range addrs {
		addr = net.JoinHostPort(a.String(), port)
		con, err := dialer.DialContext(ctx, network, addr)
		if err != nil {
			dialErrs = append(dialErrs, err)
			continue
		}
		return con, err
	}
	return nil, errorx.DecorateMany(fmt.Sprintf("couldn't dial to %s", addr), dialErrs...)
}

func getHTTPProxy(req *http.Request) (*url.URL, error) {
	if len(config.ProxyURL) == 0 {
		return nil, nil
	}
	return url.Parse(config.ProxyURL)
}
