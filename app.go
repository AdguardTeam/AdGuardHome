package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gobuffalo/packr"
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
	{
		var configFilename *string
		var bindHost *string
		var bindPort *int
		var opts = []struct {
			longName    string
			shortName   string
			description string
			callback    func(value string)
		}{
			{"config", "c", "path to config file", func(value string) { configFilename = &value }},
			{"host", "h", "host address to bind HTTP server on", func(value string) { bindHost = &value }},
			{"port", "p", "port to serve HTTP pages on", func(value string) {
				v, err := strconv.Atoi(value)
				if err != nil {
					panic("Got port that is not a number")
				}
				bindPort = &v
			}},
			{"help", "h", "print this help", nil},
		}
		printHelp := func() {
			fmt.Printf("Usage:\n\n")
			fmt.Printf("%s [options]\n\n", os.Args[0])
			fmt.Printf("Options:\n")
			for _, opt := range opts {
				fmt.Printf("  -%s, %-30s %s\n", opt.shortName, "--"+opt.longName, opt.description)
			}
		}
		for i := 1; i < len(os.Args); i++ {
			v := os.Args[i]
			// short-circuit for help
			if v == "--help" || v == "-h" {
				printHelp()
				os.Exit(64)
			}
			knownParam := false
			for _, opt := range opts {
				if v == "--"+opt.longName {
					if i+1 > len(os.Args) {
						log.Printf("ERROR: Got %s without argument\n", v)
						os.Exit(64)
					}
					i++
					opt.callback(os.Args[i])
					knownParam = true
					break
				}
				if v == "-"+opt.shortName {
					if i+1 > len(os.Args) {
						log.Printf("ERROR: Got %s without argument\n", v)
						os.Exit(64)
					}
					i++
					opt.callback(os.Args[i])
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

	// Eat all args so that coredns can start happily
	if len(os.Args) > 1 {
		os.Args = os.Args[:1]
	}

	// Do the upgrade if necessary
	err := upgradeConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Save the updated config
	err = writeConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Load filters from the disk
	for i := range config.Filters {
		filter := &config.Filters[i]
		err = filter.load()
		if err != nil {
			// This is okay for the first start, the filter will be loaded later
			log.Printf("Couldn't load filter %d contents due to %s", filter.ID, err)
		}
	}

	address := net.JoinHostPort(config.BindHost, strconv.Itoa(config.BindPort))

	runFiltersUpdatesTimer()

	http.Handle("/", optionalAuthHandler(http.FileServer(box)))
	registerControlHandlers()

	err = startDNSServer()
	if err != nil {
		log.Fatal(err)
	}

	URL := fmt.Sprintf("http://%s", address)
	log.Println("Go to " + URL)
	log.Fatal(http.ListenAndServe(address, nil))
}

func getInput() (string, error) {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	text := scanner.Text()
	err := scanner.Err()
	return text, err
}

func promptAndGet(prompt string) (string, error) {
	for {
		fmt.Printf(prompt)
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
		fmt.Printf(prompt)
		password, err := terminal.ReadPassword(int(os.Stdin.Fd()))
		fmt.Printf("\n")
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
		trace("File %s exists, won't ask for password", configfile)
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

// Performs necessary upgrade operations if needed
func upgradeConfig() error {
	if config.SchemaVersion == SchemaVersion {
		// No upgrade, do nothing
		return nil
	}

	if config.SchemaVersion > SchemaVersion {
		// Unexpected -- the config file is newer than we expect
		return fmt.Errorf("configuration file is supposed to be used with a newer version of AdGuard Home, schema=%d", config.SchemaVersion)
	}

	// Perform upgrade operations for each consecutive version upgrade
	for oldVersion, newVersion := config.SchemaVersion, config.SchemaVersion+1; newVersion <= SchemaVersion; {
		err := upgradeConfigSchema(oldVersion, newVersion)
		if err != nil {
			log.Fatal(err)
		}

		// Increment old and new versions
		oldVersion++
		newVersion++
	}

	// Save the current schema version
	config.SchemaVersion = SchemaVersion

	return nil
}

// Upgrade from oldVersion to newVersion
func upgradeConfigSchema(oldVersion int, newVersion int) error {
	if oldVersion == 0 && newVersion == 1 {
		log.Printf("Updating schema from %d to %d", oldVersion, newVersion)

		// The first schema upgrade:
		// Added "ID" field to "filter" -- we need to populate this field now
		// Added "config.ourDataDir" -- where we will now store filters contents
		for i := range config.Filters {
			filter := &config.Filters[i] // otherwise we will be operating on a copy

			// Set the filter ID
			log.Printf("Seting ID=%d for filter %s", NextFilterId, filter.URL)
			filter.ID = NextFilterId
			NextFilterId++

			// Forcibly update the filter
			_, err := filter.update(true)
			if err != nil {
				log.Fatal(err)
			}

			// Saving it to the filters dir now
			err = filter.save()
			if err != nil {
				log.Fatal(err)
			}
		}

		// No more "dnsfilter.txt", filters are now loaded from config.ourDataDir/filters/
		dnsFilterPath := filepath.Join(config.ourBinaryDir, "dnsfilter.txt")
		_, err := os.Stat(dnsFilterPath)
		if !os.IsNotExist(err) {
			log.Printf("Deleting %s as we don't need it anymore", dnsFilterPath)
			err = os.Remove(dnsFilterPath)
			if err != nil {
				log.Printf("Cannot remove %s due to %s", dnsFilterPath, err)
			}
		}
	}

	return nil
}
