package home

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
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

	"github.com/AdguardTeam/golibs/log"
	"github.com/NYTimes/gziphandler"
	"github.com/gobuffalo/packr"
)

const (
	// Used in config to indicate that syslog or eventlog (win) should be used for logger output
	configSyslog = "syslog"
)

// Update-related variables
var (
	versionString   string
	updateChannel   string
	versionCheckURL string
)

const versionCheckPeriod = time.Hour * 8

// Main is the entry point
func Main(version string, channel string) {
	// Init update-related global variables
	versionString = version
	updateChannel = channel
	versionCheckURL = "https://static.adguard.com/adguardhome/" + updateChannel + "/version.json"

	// config can be specified, which reads options from there, but other command line flags have to override config values
	// therefore, we must do it manually instead of using a lib
	args := loadOptions()

	if args.serviceControlAction != "" {
		handleServiceControlAction(args.serviceControlAction)
		return
	}

	// run the protection
	run(args)
}

// run initializes configuration and runs the AdGuard Home
// run is a blocking method and it won't exit until the service is stopped!
// nolint
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
	log.Printf("AdGuard Home, version %s, channel %s\n", versionString, updateChannel)
	log.Debug("Current working directory is %s", config.ourWorkingDir)
	if args.runningAsService {
		log.Info("AdGuard Home is running as a service")
	}
	config.runningAsService = args.runningAsService
	config.disableUpdate = args.disableUpdate

	config.firstRun = detectFirstRun()
	if config.firstRun {
		requireAdminRights()
	}

	config.appSignalChannel = make(chan os.Signal)
	signal.Notify(config.appSignalChannel, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)
	go func() {
		<-config.appSignalChannel
		cleanup()
		cleanupAlways()
		os.Exit(0)
	}()

	initConfig()
	config.clients.Init()
	initServices()

	if !config.firstRun {
		// Do the upgrade if necessary
		err := upgradeConfig()
		if err != nil {
			log.Fatal(err)
		}

		err = parseConfig()
		if err != nil {
			os.Exit(1)
		}

		if args.checkConfig {
			log.Info("Configuration file is OK")
			os.Exit(0)
		}
	}

	if (runtime.GOOS == "linux" || runtime.GOOS == "darwin") &&
		config.RlimitNoFile != 0 {
		setRlimit(config.RlimitNoFile)
	}

	// override bind host/port from the console
	if args.bindHost != "" {
		config.BindHost = args.bindHost
	}
	if args.bindPort != 0 {
		config.BindPort = args.bindPort
	}

	if !config.firstRun {
		// Save the updated config
		err := config.write()
		if err != nil {
			log.Fatal(err)
		}

		initDNSServer()
		go func() {
			err = startDNSServer()
			if err != nil {
				log.Fatal(err)
			}
		}()

		err = startDHCPServer()
		if err != nil {
			log.Fatal(err)
		}
	}

	if len(args.pidFile) != 0 && writePIDFile(args.pidFile) {
		config.pidFileName = args.pidFile
	}

	// Initialize and run the admin Web interface
	box := packr.NewBox("../build/static")

	// if not configured, redirect / to /install.html, otherwise redirect /install.html to /
	http.Handle("/", postInstallHandler(optionalAuthHandler(gziphandler.GzipHandler(http.FileServer(box)))))
	registerControlHandlers()

	// add handlers for /install paths, we only need them when we're not configured yet
	if config.firstRun {
		log.Info("This is the first launch of AdGuard Home, redirecting everything to /install.html ")
		http.Handle("/install.html", preInstallHandler(http.FileServer(box)))
		registerInstallHandlers()
	}

	config.httpsServer.cond = sync.NewCond(&config.httpsServer.Mutex)

	// for https, we have a separate goroutine loop
	go httpServerLoop()

	// this loop is used as an ability to change listening host and/or port
	for !config.httpsServer.shutdown {
		printHTTPAddresses("http")

		// we need to have new instance, because after Shutdown() the Server is not usable
		address := net.JoinHostPort(config.BindHost, strconv.Itoa(config.BindPort))
		config.httpServer = &http.Server{
			Addr: address,
		}
		err := config.httpServer.ListenAndServe()
		if err != http.ErrServerClosed {
			cleanupAlways()
			log.Fatal(err)
		}
		// We use ErrServerClosed as a sign that we need to rebind on new address, so go back to the start of the loop
	}

	// wait indefinitely for other go-routines to complete their job
	select {}
}

func httpServerLoop() {
	for !config.httpsServer.shutdown {
		config.httpsServer.cond.L.Lock()
		// this mechanism doesn't let us through until all conditions are met
		for config.TLS.Enabled == false ||
			config.TLS.PortHTTPS == 0 ||
			len(config.TLS.PrivateKeyData) == 0 ||
			len(config.TLS.CertificateChainData) == 0 { // sleep until necessary data is supplied
			config.httpsServer.cond.Wait()
		}
		address := net.JoinHostPort(config.BindHost, strconv.Itoa(config.TLS.PortHTTPS))
		// validate current TLS config and update warnings (it could have been loaded from file)
		data := validateCertificates(string(config.TLS.CertificateChainData), string(config.TLS.PrivateKeyData), config.TLS.ServerName)
		if !data.ValidPair {
			cleanupAlways()
			log.Fatal(data.WarningValidation)
		}
		config.Lock()
		config.TLS.tlsConfigStatus = data // update warnings
		config.Unlock()

		// prepare certs for HTTPS server
		// important -- they have to be copies, otherwise changing the contents in config.TLS will break encryption for in-flight requests
		certchain := make([]byte, len(config.TLS.CertificateChainData))
		copy(certchain, config.TLS.CertificateChainData)
		privatekey := make([]byte, len(config.TLS.PrivateKeyData))
		copy(privatekey, config.TLS.PrivateKeyData)
		cert, err := tls.X509KeyPair(certchain, privatekey)
		if err != nil {
			cleanupAlways()
			log.Fatal(err)
		}
		config.httpsServer.cond.L.Unlock()

		// prepare HTTPS server
		config.httpsServer.server = &http.Server{
			Addr: address,
			TLSConfig: &tls.Config{
				Certificates: []tls.Certificate{cert},
				MinVersion:   tls.VersionTLS12,
			},
		}

		printHTTPAddresses("https")
		err = config.httpsServer.server.ListenAndServeTLS("", "")
		if err != http.ErrServerClosed {
			cleanupAlways()
			log.Fatal(err)
		}
	}
}

// Check if the current user has root (administrator) rights
//  and if not, ask and try to run as root
func requireAdminRights() {
	admin, _ := haveAdminRights()
	if admin {
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

	level := log.INFO
	if ls.Verbose {
		level = log.DEBUG
	}
	log.SetLevel(level)

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

	err := stopDNSServer()
	if err != nil {
		log.Error("Couldn't stop DNS server: %s", err)
	}
	err = stopDHCPServer()
	if err != nil {
		log.Error("Couldn't stop DHCP server: %s", err)
	}
}

// Stop HTTP server, possibly waiting for all active connections to be closed
func stopHTTPServer() {
	config.httpsServer.shutdown = true
	if config.httpsServer.server != nil {
		config.httpsServer.server.Shutdown(context.TODO())
	}
	config.httpServer.Shutdown(context.TODO())
}

// This function is called before application exits
func cleanupAlways() {
	if len(config.pidFileName) != 0 {
		os.Remove(config.pidFileName)
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
		{"service", "s", "Service control action: status, install, uninstall, start, stop, restart", func(value string) {
			o.serviceControlAction = value
		}, nil},
		{"logfile", "l", "Path to log file. If empty: write to stdout; if 'syslog': write to system log", func(value string) {
			o.logFile = value
		}, nil},
		{"pidfile", "", "Path to a file where PID is stored", func(value string) { o.pidFile = value }, nil},
		{"check-config", "", "Check configuration and exit", nil, func() { o.checkConfig = true }},
		{"no-check-update", "", "Don't check for updates", nil, func() { o.disableUpdate = true }},
		{"verbose", "v", "Enable verbose output", nil, func() { o.verbose = true }},
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

	if proto == "https" && config.TLS.ServerName != "" {
		if config.TLS.PortHTTPS == 443 {
			log.Printf("Go to https://%s", config.TLS.ServerName)
		} else {
			log.Printf("Go to https://%s:%d", config.TLS.ServerName, config.TLS.PortHTTPS)
		}
	} else if config.BindHost == "0.0.0.0" {
		log.Println("AdGuard Home is available on the following addresses:")
		ifaces, err := getValidNetInterfacesForWeb()
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
