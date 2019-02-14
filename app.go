package main

import (
	"crypto/tls"
	"fmt"
	stdlog "log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/gobuffalo/packr"

	"github.com/hmage/golibs/log"
)

// VersionString will be set through ldflags, contains current version
var VersionString = "undefined"
var httpServer *http.Server
var httpsServer struct {
	server     *http.Server
	cond       *sync.Cond // reacts to config.TLS.Enabled, PortHTTPS, CertificateChain and PrivateKey
	sync.Mutex            // protects config.TLS
}

const (
	// Used in config to indicate that syslog or eventlog (win) should be used for logger output
	configSyslog = "syslog"
)

// main is the entry point
func main() {
	// config can be specified, which reads options from there, but other command line flags have to override config values
	// therefore, we must do it manually instead of using a lib
	args := loadOptions()

	if args.serviceControlAction != "" {
		handleServiceControlAction(args.serviceControlAction)
		return
	}

	signalChannel := make(chan os.Signal)
	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)
	go func() {
		<-signalChannel
		cleanup()
		os.Exit(0)
	}()

	// run the protection
	run(args)
}

// run initializes configuration and runs the AdGuard Home
// run is a blocking method and it won't exit until the service is stopped!
func run(args options) {
	// config file path can be overridden by command-line arguments:
	if args.configFilename != "" {
		config.ourConfigFilename = args.configFilename
	}

	// configure working dir and config path
	initWorkingDir(args)

	// configure log level and output
	configureLogger(args)

	// print the first message after logger is configured
	log.Printf("AdGuard Home, version %s\n", VersionString)
	log.Tracef("Current working directory is %s", config.ourWorkingDir)
	if args.runningAsService {
		log.Printf("AdGuard Home is running as a service")
	}

	config.firstRun = detectFirstRun()

	// Do the upgrade if necessary
	err := upgradeConfig()
	if err != nil {
		log.Fatal(err)
	}

	// parse from config file
	err = parseConfig()
	if err != nil {
		log.Fatal(err)
	}

	// override bind host/port from the console
	if args.bindHost != "" {
		config.BindHost = args.bindHost
	}
	if args.bindPort != 0 {
		config.BindPort = args.bindPort
	}

	// Load filters from the disk
	// And if any filter has zero ID, assign a new one
	for i := range config.Filters {
		filter := &config.Filters[i] // otherwise we're operating on a copy
		if filter.ID == 0 {
			filter.ID = assignUniqueFilterID()
		}
		err = filter.load()
		if err != nil {
			// This is okay for the first start, the filter will be loaded later
			log.Printf("Couldn't load filter %d contents due to %s", filter.ID, err)
			// clear LastUpdated so it gets fetched right away
		}

		if len(filter.Rules) == 0 {
			filter.LastUpdated = time.Time{}
		}
	}

	// Save the updated config
	err = config.write()
	if err != nil {
		log.Fatal(err)
	}

	// Init the DNS server instance before registering HTTP handlers
	dnsBaseDir := filepath.Join(config.ourWorkingDir, dataDir)
	initDNSServer(dnsBaseDir)

	if !config.firstRun {
		err = startDNSServer()
		if err != nil {
			log.Fatal(err)
		}

		err = startDHCPServer()
		if err != nil {
			log.Fatal(err)
		}
	}

	// Update filters we've just loaded right away, don't wait for periodic update timer
	go func() {
		refreshFiltersIfNecessary(false)
		// Save the updated config
		err := config.write()
		if err != nil {
			log.Fatal(err)
		}
	}()
	// Schedule automatic filters updates
	go periodicallyRefreshFilters()

	// Initialize and run the admin Web interface
	box := packr.NewBox("build/static")
	// if not configured, redirect / to /install.html, otherwise redirect /install.html to /
	http.Handle("/", postInstallHandler(optionalAuthHandler(http.FileServer(box))))
	registerControlHandlers()

	// add handlers for /install paths, we only need them when we're not configured yet
	if config.firstRun {
		log.Printf("This is the first launch of AdGuard Home, redirecting everything to /install.html ")
		http.Handle("/install.html", preInstallHandler(http.FileServer(box)))
		registerInstallHandlers()
	}

	httpsServer.cond = sync.NewCond(&httpsServer.Mutex)

	// for https, we have a separate goroutine loop
	go func() {
		for { // this is an endless loop
			httpsServer.cond.L.Lock()
			// this mechanism doesn't let us through until all conditions are ment
			for config.TLS.Enabled == false || config.TLS.PortHTTPS == 0 || config.TLS.PrivateKey == "" || config.TLS.CertificateChain == "" { // sleep until neccessary data is supplied
				httpsServer.cond.Wait()
			}
			address := net.JoinHostPort(config.BindHost, strconv.Itoa(config.TLS.PortHTTPS))
			// validate current TLS config and update warnings (it could have been loaded from file)
			data, err := validateCertificates(config.TLS)
			if err != nil {
				log.Fatal(err)
				os.Exit(1)
			}
			config.TLS = data // update warnings

			// prepare cert for HTTPS server
			cert, err := tls.X509KeyPair([]byte(config.TLS.CertificateChain), []byte(config.TLS.PrivateKey))
			if err != nil {
				log.Fatal(err)
				os.Exit(1)
			}
			httpsServer.cond.L.Unlock()

			// prepare HTTPS server
			httpsServer.server = &http.Server{
				Addr: address,
				TLSConfig: &tls.Config{
					Certificates: []tls.Certificate{cert},
				},
			}

			URL := fmt.Sprintf("https://%s", address)
			log.Println("Go to " + URL)
			err = httpsServer.server.ListenAndServeTLS("", "")
			if err != http.ErrServerClosed {
				log.Fatal(err)
				os.Exit(1)
			}
		}
	}()

	// this loop is used as an ability to change listening host and/or port
	for {
		address := net.JoinHostPort(config.BindHost, strconv.Itoa(config.BindPort))
		URL := fmt.Sprintf("http://%s", address)
		log.Println("Go to " + URL)
		// we need to have new instance, because after Shutdown() the Server is not usable
		httpServer = &http.Server{
			Addr: address,
		}
		err := httpServer.ListenAndServe()
		if err != http.ErrServerClosed {
			log.Fatal(err)
			os.Exit(1)
		}
		// We use ErrServerClosed as a sign that we need to rebind on new address, so go back to the start of the loop
	}
}

// initWorkingDir initializes the ourWorkingDir
// if no command-line arguments specified, we use the directory where our binary file is located
func initWorkingDir(args options) {
	exec, err := os.Executable()
	if err != nil {
		panic(err)
	}

	if args.workDir != "" {
		// If there is a custom config file, use it's directory as our working dir
		config.ourWorkingDir = args.workDir
	} else {
		config.ourWorkingDir = filepath.Dir(exec)
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

	log.Verbose = ls.Verbose

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
		err := configureSyslog()
		if err != nil {
			log.Fatalf("cannot initialize syslog: %s", err)
		}
	} else {
		logFilePath := filepath.Join(config.ourWorkingDir, ls.LogFile)
		file, err := os.OpenFile(logFilePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0755)
		if err != nil {
			log.Fatalf("cannot create a log file: %s", err)
		}
		stdlog.SetOutput(file)
	}
}

func cleanup() {
	log.Printf("Stopping AdGuard Home")

	err := stopDNSServer()
	if err != nil {
		log.Printf("Couldn't stop DNS server: %s", err)
	}
	err = stopDHCPServer()
	if err != nil {
		log.Printf("Couldn't stop DHCP server: %s", err)
	}
}

// command-line arguments
type options struct {
	verbose        bool   // is verbose logging enabled
	configFilename string // path to the config file
	workDir        string // path to the working directory where we will store the filters data and the querylog
	bindHost       string // host address to bind HTTP server on
	bindPort       int    // port to serve HTTP pages on
	logFile        string // Path to the log file. If empty, write to stdout. If "syslog", writes to syslog

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
		{"config", "c", "path to the config file", func(value string) { o.configFilename = value }, nil},
		{"work-dir", "w", "path to the working directory", func(value string) { o.workDir = value }, nil},
		{"host", "h", "host address to bind HTTP server on", func(value string) { o.bindHost = value }, nil},
		{"port", "p", "port to serve HTTP pages on", func(value string) {
			v, err := strconv.Atoi(value)
			if err != nil {
				panic("Got port that is not a number")
			}
			o.bindPort = v
		}, nil},
		{"service", "s", "service control action: status, install, uninstall, start, stop, restart", func(value string) {
			o.serviceControlAction = value
		}, nil},
		{"logfile", "l", "path to the log file. If empty, writes to stdout, if 'syslog' -- system log", func(value string) {
			o.logFile = value
		}, nil},
		{"verbose", "v", "enable verbose output", nil, func() { o.verbose = true }},
		{"help", "", "print this help", nil, func() {
			printHelp()
			os.Exit(64)
		}},
	}
	printHelp = func() {
		fmt.Printf("Usage:\n\n")
		fmt.Printf("%s [options]\n\n", os.Args[0])
		fmt.Printf("Options:\n")
		for _, opt := range opts {
			if opt.shortName != "" {
				fmt.Printf("  -%s, %-30s %s\n", opt.shortName, "--"+opt.longName, opt.description)
			} else {
				fmt.Printf("  %-34s %s\n", "--"+opt.longName, opt.description)
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
						log.Printf("ERROR: Got %s without argument\n", v)
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
			log.Printf("ERROR: unknown option %v\n", v)
			printHelp()
			os.Exit(64)
		}
	}

	return o
}
