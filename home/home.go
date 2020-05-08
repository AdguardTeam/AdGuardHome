package home

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

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

const versionCheckPeriod = time.Hour * 8

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

	if args.serviceControlAction != "" {
		handleServiceControlAction(args.serviceControlAction)
		return
	}

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

	// run the protection
	run(args)
}

// run initializes configuration and runs the AdGuard Home
// run is a blocking method!
// nolint
func run(args options) {
	// config file path can be overridden by command-line arguments:
	if args.configFilename != "" {
		Context.configFilename = args.configFilename
	} else {
		// Default config file name
		Context.configFilename = "AdGuardHome.yaml"
	}

	// configure working dir and config path
	initWorkingDir(args)

	// configure log level and output
	configureLogger(args)

	// print the first message after logger is configured
	msg := "AdGuard Home, version %s, channel %s, arch %s %s"
	if ARMVersion != "" {
		msg = msg + " v" + ARMVersion
	}
	log.Printf(msg, versionString, updateChannel, runtime.GOOS, runtime.GOARCH)
	log.Debug("Current working directory is %s", Context.workDir)
	if args.runningAsService {
		log.Info("AdGuard Home is running as a service")
	}
	Context.runningAsService = args.runningAsService
	Context.disableUpdate = args.disableUpdate

	Context.firstRun = detectFirstRun()
	if Context.firstRun {
		log.Info("This is the first time AdGuard Home is launched")
		requireAdminRights()
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
	Context.dhcpServer = dhcpd.Create(config.DHCP)
	if Context.dhcpServer == nil {
		log.Error("Failed to initialize DHCP server, exiting")
		os.Exit(1)
	}
	Context.autoHosts.Init("")
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

		err = startDHCPServer()
		if err != nil {
			log.Fatal(err)
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

// Check if the current user has root (administrator) rights
//  and if not, ask and try to run as root
func requireAdminRights() {
	admin, _ := util.HaveAdminRights()
	if //noinspection ALL
	admin || isdelve.Enabled {
		// Don't forget that for this to work you need to add "delve" tag explicitly
		// https://stackoverflow.com/questions/47879070/how-can-i-see-if-the-goland-debugger-is-running-in-the-program
		return
	}

	if runtime.GOOS == "windows" {
		log.Fatal("This is the first launch of AdGuard Home. You must run it as Administrator.")

	} else {
		log.Error("This is the first launch of AdGuard Home. You must run it as root.")

		_, _ = io.WriteString(os.Stdout, "Do you want to start AdGuard Home as root user? [y/n] ")
		stdin := bufio.NewReader(os.Stdin)
		buf, _ := stdin.ReadString('\n')
		buf = strings.TrimSpace(buf)
		if buf != "y" {
			os.Exit(1)
		}

		cmd := exec.Command("sudo", os.Args...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		_ = cmd.Run()
		os.Exit(1)
	}
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
	if args.verbose {
		ls.Verbose = true
	}
	if args.logFile != "" {
		ls.LogFile = args.logFile
	}

	// log.SetLevel(log.INFO) - default
	if ls.Verbose {
		log.SetLevel(log.DEBUG)
	}

	if args.runningAsService && ls.LogFile == "" && runtime.GOOS == "windows" {
		// When running as a Windows service, use eventlog by default if nothing else is configured
		// Otherwise, we'll simply loose the log output
		ls.LogFile = configSyslog
	}

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

		file, err := os.OpenFile(logFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		if err != nil {
			log.Fatalf("cannot create a log file: %s", err)
		}
		log.SetOutput(file)
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
	err = stopDHCPServer()
	if err != nil {
		log.Error("Couldn't stop DHCP server: %s", err)
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

// command-line arguments
type options struct {
	verbose        bool   // is verbose logging enabled
	configFilename string // path to the config file
	workDir        string // path to the working directory where we will store the filters data and the querylog
	bindHost       string // host address to bind HTTP server on
	bindPort       int    // port to serve HTTP pages on
	logFile        string // Path to the log file. If empty, write to stdout. If "syslog", writes to syslog
	pidFile        string // File name to save PID to
	checkConfig    bool   // Check configuration and exit
	disableUpdate  bool   // If set, don't check for updates

	// service control action (see service.ControlAction array + "status" command)
	serviceControlAction string

	// runningAsService flag is set to true when options are passed from the service runner
	runningAsService bool
}

// loadOptions reads command line arguments and initializes configuration
func loadOptions() options {
	o := options{}

	var printHelp func()
	var opts = []struct {
		longName          string
		shortName         string
		description       string
		callbackWithValue func(value string)
		callbackNoValue   func()
	}{
		{"config", "c", "Path to the config file", func(value string) { o.configFilename = value }, nil},
		{"work-dir", "w", "Path to the working directory", func(value string) { o.workDir = value }, nil},
		{"host", "h", "Host address to bind HTTP server on", func(value string) { o.bindHost = value }, nil},
		{"port", "p", "Port to serve HTTP pages on", func(value string) {
			v, err := strconv.Atoi(value)
			if err != nil {
				panic("Got port that is not a number")
			}
			o.bindPort = v
		}, nil},
		{"service", "s", "Service control action: status, install, uninstall, start, stop, restart, reload (configuration)", func(value string) {
			o.serviceControlAction = value
		}, nil},
		{"logfile", "l", "Path to log file. If empty: write to stdout; if 'syslog': write to system log", func(value string) {
			o.logFile = value
		}, nil},
		{"pidfile", "", "Path to a file where PID is stored", func(value string) { o.pidFile = value }, nil},
		{"check-config", "", "Check configuration and exit", nil, func() { o.checkConfig = true }},
		{"no-check-update", "", "Don't check for updates", nil, func() { o.disableUpdate = true }},
		{"verbose", "v", "Enable verbose output", nil, func() { o.verbose = true }},
		{"version", "", "Show the version and exit", nil, func() {
			fmt.Printf("AdGuardHome %s\n", versionString)
			os.Exit(0)
		}},
		{"help", "", "Print this help", nil, func() {
			printHelp()
			os.Exit(64)
		}},
	}
	printHelp = func() {
		fmt.Printf("Usage:\n\n")
		fmt.Printf("%s [options]\n\n", os.Args[0])
		fmt.Printf("Options:\n")
		for _, opt := range opts {
			val := ""
			if opt.callbackWithValue != nil {
				val = " VALUE"
			}
			if opt.shortName != "" {
				fmt.Printf("  -%s, %-30s %s\n", opt.shortName, "--"+opt.longName+val, opt.description)
			} else {
				fmt.Printf("  %-34s %s\n", "--"+opt.longName+val, opt.description)
			}
		}
	}
	for i := 1; i < len(os.Args); i++ {
		v := os.Args[i]
		knownParam := false
		for _, opt := range opts {
			if v == "--"+opt.longName || (opt.shortName != "" && v == "-"+opt.shortName) {
				if opt.callbackWithValue != nil {
					if i+1 >= len(os.Args) {
						log.Error("Got %s without argument\n", v)
						os.Exit(64)
					}
					i++
					opt.callbackWithValue(os.Args[i])
				} else if opt.callbackNoValue != nil {
					opt.callbackNoValue()
				}
				knownParam = true
				break
			}
		}
		if !knownParam {
			log.Error("unknown option %v\n", v)
			printHelp()
			os.Exit(64)
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
	if proto == "https" && tlsConf.ServerName != "" {
		if tlsConf.PortHTTPS == 443 {
			log.Printf("Go to https://%s", tlsConf.ServerName)
		} else {
			log.Printf("Go to https://%s:%d", tlsConf.ServerName, tlsConf.PortHTTPS)
		}
	} else if config.BindHost == "0.0.0.0" {
		log.Println("AdGuard Home is available on the following addresses:")
		ifaces, err := util.GetValidNetInterfacesForWeb()
		if err != nil {
			// That's weird, but we'll ignore it
			address = net.JoinHostPort(config.BindHost, strconv.Itoa(config.BindPort))
			log.Printf("Go to %s://%s", proto, address)
			return
		}

		for _, iface := range ifaces {
			address = net.JoinHostPort(iface.Addresses[0], strconv.Itoa(config.BindPort))
			log.Printf("Go to %s://%s", proto, address)
		}
	} else {
		address = net.JoinHostPort(config.BindHost, strconv.Itoa(config.BindPort))
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
