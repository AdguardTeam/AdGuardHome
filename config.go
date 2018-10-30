package main

import (
	"bytes"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"text/template"
	"time"
)

// Current schema version. We compare it with the value from
// the configuration file and perform necessary upgrade operations if needed
const SchemaVersion = 1

// Directory where we'll store all downloaded filters contents
const FiltersDir = "filters"

// configuration is loaded from YAML
type configuration struct {
	ourConfigFilename string
	ourBinaryDir      string
	// Directory to store data (i.e. filters contents)
	ourDataDir string

	// Schema version of the config file. This value is used when performing the app updates.
	SchemaVersion int           `yaml:"schema_version"`
	BindHost      string        `yaml:"bind_host"`
	BindPort      int           `yaml:"bind_port"`
	AuthName      string        `yaml:"auth_name"`
	AuthPass      string        `yaml:"auth_pass"`
	CoreDNS       coreDNSConfig `yaml:"coredns"`
	Filters       []filter      `yaml:"filters"`
	UserRules     []string      `yaml:"user_rules"`

	sync.RWMutex `yaml:"-"`
}

type coreDnsFilter struct {
	ID   int    `yaml:"-"`
	Path string `yaml:"-"`
}

type coreDNSConfig struct {
	binaryFile          string
	coreFile            string
	Filters             []coreDnsFilter `yaml:"-"`
	Port                int             `yaml:"port"`
	ProtectionEnabled   bool            `yaml:"protection_enabled"`
	FilteringEnabled    bool            `yaml:"filtering_enabled"`
	SafeBrowsingEnabled bool            `yaml:"safebrowsing_enabled"`
	SafeSearchEnabled   bool            `yaml:"safesearch_enabled"`
	ParentalEnabled     bool            `yaml:"parental_enabled"`
	ParentalSensitivity int             `yaml:"parental_sensitivity"`
	BlockedResponseTTL  int             `yaml:"blocked_response_ttl"`
	QueryLogEnabled     bool            `yaml:"querylog_enabled"`
	Pprof               string          `yaml:"-"`
	Cache               string          `yaml:"-"`
	Prometheus          string          `yaml:"-"`
	UpstreamDNS         []string        `yaml:"upstream_dns"`
}

type filter struct {
	ID          int    `json:"ID"` // auto-assigned when filter is added
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
	ourConfigFilename: "AdGuardHome.yaml",
	ourDataDir:        "data",
	BindPort:          3000,
	BindHost:          "127.0.0.1",
	CoreDNS: coreDNSConfig{
		Port:                53,
		binaryFile:          "coredns",  // only filename, no path
		coreFile:            "Corefile", // only filename, no path
		ProtectionEnabled:   true,
		FilteringEnabled:    true,
		SafeBrowsingEnabled: false,
		BlockedResponseTTL:  10, // in seconds
		QueryLogEnabled:     true,
		UpstreamDNS:         defaultDNS,
		Cache:               "cache",
		Prometheus:          "prometheus :9153",
	},
	Filters: []filter{
		{ID: 1, Enabled: true, URL: "https://adguardteam.github.io/AdGuardSDNSFilter/Filters/filter.txt", Name: "AdGuard Simplified Domain Names filter"},
		{ID: 2, Enabled: false, URL: "https://adaway.org/hosts.txt", Name: "AdAway"},
		{ID: 3, Enabled: false, URL: "https://hosts-file.net/ad_servers.txt", Name: "hpHosts - Ad and Tracking servers only"},
		{ID: 4, Enabled: false, URL: "http://www.malwaredomainlist.com/hostslist/hosts.txt", Name: "MalwareDomainList.com Hosts List"},
	},
}

// Creates a helper object for working with the user rules
func getUserFilter() filter {

	// TODO: This should be calculated when UserRules are set
	contents := []byte{}
	for _, rule := range config.UserRules {
		contents = append(contents, []byte(rule)...)
		contents = append(contents, '\n')
	}

	userFilter := filter{
		// User filter always has ID=0
		ID:       0,
		contents: contents,
		Enabled:  true,
	}

	return userFilter
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
	err = writeFileSafe(configfile, yamlText)
	if err != nil {
		log.Printf("Couldn't save YAML config: %s", err)
		return err
	}

	userFilter := getUserFilter()
	err = userFilter.save()
	if err != nil {
		log.Printf("Couldn't save the user filter: %s", err)
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
	err = writeFileSafe(corefile, []byte(configtext))
	if err != nil {
		log.Printf("Couldn't save DNS config: %s", err)
		return err
	}
	return nil
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

const coreDNSConfigTemplate = `.:{{.Port}} {
    {{if .ProtectionEnabled}}dnsfilter {
        {{if .SafeBrowsingEnabled}}safebrowsing{{end}}
        {{if .ParentalEnabled}}parental {{.ParentalSensitivity}}{{end}}
        {{if .SafeSearchEnabled}}safesearch{{end}}
        {{if .QueryLogEnabled}}querylog{{end}}
        blocked_ttl {{.BlockedResponseTTL}}
		{{if .FilteringEnabled}}
		{{range .Filters}}
		filter {{.ID}} "{{.Path}}"
		{{end}}
		{{end}}
    }{{end}}
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
	temporaryConfig := config.CoreDNS

	// fill the list of filters
	filters := make([]coreDnsFilter, 0)

	// first of all, append the user filter
	userFilter := getUserFilter()

	if len(userFilter.contents) > 0 {
		filters = append(filters, coreDnsFilter{ID: userFilter.ID, Path: userFilter.getFilterFilePath()})
	}

	// then go through other filters
	for i := range config.Filters {
		filter := &config.Filters[i]

		if filter.Enabled && len(filter.contents) > 0 {
			filters = append(filters, coreDnsFilter{ID: filter.ID, Path: filter.getFilterFilePath()})
		}
	}
	temporaryConfig.Filters = filters

	// run the template
	err = t.Execute(&configBytes, &temporaryConfig)
	if err != nil {
		log.Printf("Couldn't generate DNS config: %s", err)
		return "", err
	}
	configtext := configBytes.String()

	// remove empty lines from generated config
	configtext = removeEmptyLines.ReplaceAllString(configtext, "\n")
	return configtext, nil
}
