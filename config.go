package main

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"text/template"
	"time"

	"gopkg.in/yaml.v2"
)

// configuration is loaded from YAML
type configuration struct {
	ourConfigFilename string
	ourBinaryDir      string

	BindHost  string        `yaml:"bind_host"`
	BindPort  int           `yaml:"bind_port"`
	AuthName  string        `yaml:"auth_name"`
	AuthPass  string        `yaml:"auth_pass"`
	CoreDNS   coreDNSConfig `yaml:"coredns"`
	Filters   []filter      `yaml:"filters"`
	UserRules []string      `yaml:"user_rules"`

	sync.RWMutex `yaml:"-"`
}

type coreDNSConfig struct {
	binaryFile          string
	coreFile            string
	FilterFile          string   `yaml:"-"`
	Port                int      `yaml:"port"`
	FilteringEnabled    bool     `yaml:"filtering_enabled"`
	SafeBrowsingEnabled bool     `yaml:"safebrowsing_enabled"`
	SafeSearchEnabled   bool     `yaml:"safesearch_enabled"`
	ParentalEnabled     bool     `yaml:"parental_enabled"`
	ParentalSensitivity int      `yaml:"parental_sensitivity"`
	BlockedResponseTTL  int      `yaml:"blocked_response_ttl"`
	QueryLogEnabled     bool     `yaml:"querylog_enabled"`
	Pprof               string   `yaml:"-"`
	Cache               string   `yaml:"-"`
	Prometheus          string   `yaml:"-"`
	UpstreamDNS         []string `yaml:"upstream_dns"`
}

type filter struct {
	URL         string `json:"url"`
	Name        string `json:"name" yaml:"name"`
	Enabled     bool   `json:"enabled"`
	RulesCount  int    `json:"rules_count" yaml:"-"`
	contents    []byte
	LastUpdated time.Time `json:"last_updated" yaml:"-"`
}

var defaultDNS = []string{"tls://1.1.1.1", "tls://1.0.0.1"}

// initialize to default values, will be changed later when reading config or parsing command line
var config = configuration{
	ourConfigFilename: "AdguardDNS.yaml",
	BindPort:          3000,
	BindHost:          "127.0.0.1",
	CoreDNS: coreDNSConfig{
		Port:                53,
		binaryFile:          "coredns",       // only filename, no path
		coreFile:            "Corefile",      // only filename, no path
		FilterFile:          "dnsfilter.txt", // only filename, no path
		FilteringEnabled:    true,
		SafeBrowsingEnabled: false,
		BlockedResponseTTL:  10, // in seconds
		QueryLogEnabled:     true,
		UpstreamDNS:         defaultDNS,
		Cache:               "cache",
		Prometheus:          "prometheus :9153",
	},
	Filters: []filter{
		{Enabled: true, URL: "https://adguardteam.github.io/AdGuardSDNSFilter/Filters/filter.txt"},
		{Enabled: false, URL: "https://adaway.org/hosts.txt", Name: "AdAway"},
		{Enabled: false, URL: "https://hosts-file.net/ad_servers.txt", Name: "hpHosts - Ad and Tracking servers only"},
		{Enabled: false, URL: "http://www.malwaredomainlist.com/hostslist/hosts.txt", Name: "MalwareDomainList.com Hosts List"},
	},
}

func parseConfig() error {
	configfile := filepath.Join(config.ourBinaryDir, config.ourConfigFilename)
	log.Printf("Reading YAML file: %s", configfile)
	if _, err := os.Stat(configfile); os.IsNotExist(err) {
		// do nothing, file doesn't exist
		log.Printf("YAML file doesn't exist, skipping: %s", configfile)
		return nil
	}
	yamlFile, err := ioutil.ReadFile(configfile)
	if err != nil {
		log.Printf("Couldn't read config file: %s", err)
		return err
	}
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		log.Printf("Couldn't parse config file: %s", err)
		return err
	}

	return nil
}

func writeConfig() error {
	configfile := filepath.Join(config.ourBinaryDir, config.ourConfigFilename)
	log.Printf("Writing YAML file: %s", configfile)
	yamlText, err := yaml.Marshal(&config)
	if err != nil {
		log.Printf("Couldn't generate YAML file: %s", err)
		return err
	}
	err = ioutil.WriteFile(configfile+".tmp", yamlText, 0644)
	if err != nil {
		log.Printf("Couldn't write YAML config: %s", err)
		return err
	}
	err = os.Rename(configfile+".tmp", configfile)
	if err != nil {
		log.Printf("Couldn't rename YAML config: %s", err)
		return err
	}
	return nil
}

// --------------
// coredns config
// --------------
func writeCoreDNSConfig() error {
	corefile := filepath.Join(config.ourBinaryDir, config.CoreDNS.coreFile)
	log.Printf("Writing DNS config: %s", corefile)
	configtext, err := generateCoreDNSConfigText()
	if err != nil {
		log.Printf("Couldn't generate DNS config: %s", err)
		return err
	}
	err = ioutil.WriteFile(corefile+".tmp", []byte(configtext), 0644)
	if err != nil {
		log.Printf("Couldn't write DNS config: %s", err)
	}
	err = os.Rename(corefile+".tmp", corefile)
	if err != nil {
		log.Printf("Couldn't rename DNS config: %s", err)
	}
	return err
}

func writeAllConfigs() error {
	err := writeConfig()
	if err != nil {
		log.Printf("Couldn't write our config: %s", err)
		return err
	}
	err = writeCoreDNSConfig()
	if err != nil {
		log.Printf("Couldn't write DNS config: %s", err)
		return err
	}
	return nil
}

const coreDNSConfigTemplate = `. {
    dnsfilter {{if .FilteringEnabled}}{{.FilterFile}}{{end}} {
        {{if .SafeBrowsingEnabled}}safebrowsing{{end}}
        {{if .ParentalEnabled}}parental {{.ParentalSensitivity}}{{end}}
        {{if .SafeSearchEnabled}}safesearch{{end}}
        {{if .QueryLogEnabled}}querylog{{end}}
        blocked_ttl {{.BlockedResponseTTL}}
    }
    {{.Pprof}}
    hosts {
        fallthrough
    }
    {{if .UpstreamDNS}}forward . {{range .UpstreamDNS}}{{.}} {{end}}{{end}}
    {{.Cache}}
    {{.Prometheus}}
}
`

var removeEmptyLines = regexp.MustCompile("([\t ]*\n)+")

// generate config text
func generateCoreDNSConfigText() (string, error) {
	t, err := template.New("config").Parse(coreDNSConfigTemplate)
	if err != nil {
		log.Printf("Couldn't generate DNS config: %s", err)
		return "", err
	}

	var configBytes bytes.Buffer
	// run the template
	err = t.Execute(&configBytes, config.CoreDNS)
	if err != nil {
		log.Printf("Couldn't generate DNS config: %s", err)
		return "", err
	}
	configtext := configBytes.String()

	// remove empty lines from generated config
	configtext = removeEmptyLines.ReplaceAllString(configtext, "\n")
	return configtext, nil
}
