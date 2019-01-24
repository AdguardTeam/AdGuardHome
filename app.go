package main

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/gobuffalo/packr"
	"github.com/hmage/golibs/log"
	"golang.org/x/crypto/ssh/terminal"
)

// VersionString will be set through ldflags, contains current version
var VersionString = "undefined"

func main() {
	log.Printf("AdGuard Home web interface backend, version %s\n", VersionString)
	box := packr.NewBox("build/static")
	{
		executable, err := os.Executable()
		if err != nil {
			panic(err)
		}

		executableName := filepath.Base(executable)
		if executableName == "AdGuardHome" {
			// Binary build
			config.ourBinaryDir = filepath.Dir(executable)
		} else {
			// Most likely we're debugging -- using current working directory in this case
			workDir, _ := os.Getwd()
			config.ourBinaryDir = workDir
		}
		log.Printf("Current working directory is %s", config.ourBinaryDir)
	}

	// config can be specified, which reads options from there, but other command line flags have to override config values
	// therefore, we must do it manually instead of using a lib
	loadOptions()

	// Load filters from the disk
	// And if any filter has zero ID, assign a new one
	for i := range config.Filters {
		filter := &config.Filters[i] // otherwise we're operating on a copy
		if filter.ID == 0 {
			filter.ID = assignUniqueFilterID()
		}
		err := filter.load()
		if err != nil {
			// This is okay for the first start, the filter will be loaded later
			log.Printf("Couldn't load filter %d contents due to %s", filter.ID, err)
			// clear LastUpdated so it gets fetched right away
		}
		if len(filter.Rules) == 0 {
			filter.LastUpdated = time.Time{}
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

	signalChannel := make(chan os.Signal)
	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)
	go func() {
		<-signalChannel
		cleanup()
		os.Exit(0)
	}()

	// Save the updated config
	err := config.write()
	if err != nil {
		log.Fatal(err)
	}

	address := net.JoinHostPort(config.BindHost, strconv.Itoa(config.BindPort))

	go periodicallyRefreshFilters()

	http.Handle("/", optionalAuthHandler(http.FileServer(box)))
	registerControlHandlers()

	err = startDNSServer()
	if err != nil {
		log.Fatal(err)
	}

	err = startDHCPServer()
	if err != nil {
		log.Fatal(err)
	}

	URL := fmt.Sprintf("http://%s", address)
	log.Println("Go to " + URL)
	log.Fatal(http.ListenAndServe(address, nil))
}

func cleanup() {
	err := stopDNSServer()
	if err != nil {
		log.Printf("Couldn't stop DNS server: %s", err)
	}
}

func getInput() (string, error) {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	text := scanner.Text()
	err := scanner.Err()
	return text, err
}

// loadOptions reads command line arguments and initializes configuration
func loadOptions() {
	var printHelp func()
	var configFilename *string
	var bindHost *string
	var bindPort *int
	var opts = []struct {
		longName          string
		shortName         string
		description       string
		callbackWithValue func(value string)
		callbackNoValue   func()
	}{
		{"config", "c", "path to config file", func(value string) { configFilename = &value }, nil},
		{"host", "h", "host address to bind HTTP server on", func(value string) { bindHost = &value }, nil},
		{"port", "p", "port to serve HTTP pages on", func(value string) {
			v, err := strconv.Atoi(value)
			if err != nil {
				panic("Got port that is not a number")
			}
			bindPort = &v
		}, nil},
		{"verbose", "v", "enable verbose output", nil, func() { log.Verbose = true }},
		{"help", "h", "print this help", nil, func() { printHelp(); os.Exit(64) }},
	}
	printHelp = func() {
		fmt.Printf("Usage:\n\n")
		fmt.Printf("%s [options]\n\n", os.Args[0])
		fmt.Printf("Options:\n")
		for _, opt := range opts {
			fmt.Printf("  -%s, %-30s %s\n", opt.shortName, "--"+opt.longName, opt.description)
		}
	}
	for i := 1; i < len(os.Args); i++ {
		v := os.Args[i]
		knownParam := false
		for _, opt := range opts {
			if v == "--"+opt.longName || v == "-"+opt.shortName {
				if opt.callbackWithValue != nil {
					if i+1 > len(os.Args) {
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
	if configFilename != nil {
		config.ourConfigFilename = *configFilename
	}

	err := askUsernamePasswordIfPossible()
	if err != nil {
		log.Fatal(err)
	}

	// Do the upgrade if necessary
	err = upgradeConfig()
	if err != nil {
		log.Fatal(err)
	}

	// parse from config file
	err = parseConfig()
	if err != nil {
		log.Fatal(err)
	}

	// override bind host/port from the console
	if bindHost != nil {
		config.BindHost = *bindHost
	}
	if bindPort != nil {
		config.BindPort = *bindPort
	}
}

func promptAndGet(prompt string) (string, error) {
	for {
		fmt.Print(prompt)
		input, err := getInput()
		if err != nil {
			log.Printf("Failed to get input, aborting: %s", err)
			return "", err
		}
		if len(input) != 0 {
			return input, nil
		}
		// try again
	}
}

func promptAndGetPassword(prompt string) (string, error) {
	for {
		fmt.Print(prompt)
		password, err := terminal.ReadPassword(int(os.Stdin.Fd()))
		fmt.Print("\n")
		if err != nil {
			log.Printf("Failed to get input, aborting: %s", err)
			return "", err
		}
		if len(password) != 0 {
			return string(password), nil
		}
		// try again
	}
}

func askUsernamePasswordIfPossible() error {
	configfile := config.ourConfigFilename
	if !filepath.IsAbs(configfile) {
		configfile = filepath.Join(config.ourBinaryDir, config.ourConfigFilename)
	}
	_, err := os.Stat(configfile)
	if !os.IsNotExist(err) {
		// do nothing, file exists
		return nil
	}
	if !terminal.IsTerminal(int(os.Stdin.Fd())) {
		return nil // do nothing
	}
	if !terminal.IsTerminal(int(os.Stdout.Fd())) {
		return nil // do nothing
	}
	fmt.Printf("Would you like to set user/password for the web interface authentication (yes/no)?\n")
	yesno, err := promptAndGet("Please type 'yes' or 'no': ")
	if err != nil {
		return err
	}
	if yesno[0] != 'y' && yesno[0] != 'Y' {
		return nil
	}
	username, err := promptAndGet("Please enter the username: ")
	if err != nil {
		return err
	}

	password, err := promptAndGetPassword("Please enter the password: ")
	if err != nil {
		return err
	}

	password2, err := promptAndGetPassword("Please enter password again: ")
	if err != nil {
		return err
	}
	if password2 != password {
		fmt.Printf("Passwords do not match! Aborting\n")
		os.Exit(1)
	}

	config.AuthName = username
	config.AuthPass = password
	return nil
}
