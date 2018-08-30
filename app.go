package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gobuffalo/packr"
)

// VersionString will be set through ldflags, contains current version
var VersionString = "undefined"

func main() {
	log.Printf("AdGuard DNS web interface backend, version %s\n", VersionString)
	box := packr.NewBox("build/static")
	{
		executable, err := os.Executable()
		if err != nil {
			panic(err)
		}
		config.ourBinaryDir = filepath.Dir(executable)
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
		// parse from config file
		err := parseConfig()
		if err != nil {
			log.Fatal(err)
		}
		if bindHost != nil {
			config.BindHost = *bindHost
		}
		if bindPort != nil {
			config.BindPort = *bindPort
		}
	}

	err := writeConfig()
	if err != nil {
		log.Fatal(err)
	}

	address := net.JoinHostPort(config.BindHost, strconv.Itoa(config.BindPort))

	runStatsCollectors()
	runFilterRefreshers()

	http.Handle("/", http.FileServer(box))
	registerControlHandlers()

	URL := fmt.Sprintf("http://%s", address)
	log.Println("Go to " + URL)
	log.Fatal(http.ListenAndServe(address, nil))
}
