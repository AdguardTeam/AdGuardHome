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

// Current schema version. We compare it with the value from
// the configuration file and perform necessary upgrade operations if needed
const CurrentSchemaVersion = 1

// Directory where we'll store all downloaded filters contents
const FiltersDir = "filters"

// User filter ID is always 0
const UserFilterId = 0

// Just a counter that we use for incrementing the filter ID
var NextFilterId = time.Now().Unix()

// configuration is loaded from YAML
type configuration struct {
	// Config filename (can be overriden via the command line arguments)
	ourConfigFilename string
	// Basically, this is our working directory
	ourBinaryDir string
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
	Language      string        `yaml:"language"` // two-letter ISO 639-1 language code

	sync.RWMutex `yaml:"-"`
}

type coreDNSConfig struct {
	binaryFile          string
	coreFile            string
	Filters             []filter `yaml:"-"`
	Port                int      `yaml:"port"`
	ProtectionEnabled   bool     `yaml:"protection_enabled"`
	FilteringEnabled    bool     `yaml:"filtering_enabled"`
	SafeBrowsingEnabled bool     `yaml:"safebrowsing_enabled"`
	SafeSearchEnabled   bool     `yaml:"safesearch_enabled"`
	ParentalEnabled     bool     `yaml:"parental_enabled"`
	ParentalSensitivity int      `yaml:"parental_sensitivity"`
	BlockedResponseTTL  int      `yaml:"blocked_response_ttl"`
	QueryLogEnabled     bool     `yaml:"querylog_enabled"`
	Ratelimit           int      `yaml:"ratelimit"`
	RefuseAny           bool     `yaml:"refuse_any"`
	Pprof               string   `yaml:"-"`
	Cache               string   `yaml:"-"`
	Prometheus          string   `yaml:"-"`
	BootstrapDNS        string   `yaml:"bootstrap_dns"`
	UpstreamDNS         []string `yaml:"upstream_dns"`
}

type filter struct {
	ID          int64     `json:"id" yaml:"id"` // auto-assigned when filter is added (see NextFilterId)
	URL         string    `json:"url"`
	Name        string    `json:"name" yaml:"name"`
	Enabled     bool      `json:"enabled"`
	RulesCount  int       `json:"rulesCount" yaml:"-"`
	Contents    []byte    `json:"-" yaml:"-"`
	LastUpdated time.Time `json:"lastUpdated" yaml:"last_updated"`
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
		Ratelimit:           20,
		RefuseAny:           true,
		BootstrapDNS:        "8.8.8.8:53",
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
func userFilter() filter {
	// TODO: This should be calculated when UserRules are set
	var contents []byte
	for _, rule := range config.UserRules {
		contents = append(contents, []byte(rule)...)
		contents = append(contents, '\n')
	}

	userFilter := filter{
		// User filter always has constant ID=0
		ID:       UserFilterId,
		Contents: contents,
		Enabled:  true,
	}

	return userFilter
}

// Loads configuration from the YAML file
func parseConfig() error {
	configFile := filepath.Join(config.ourBinaryDir, config.ourConfigFilename)
	log.Printf("Reading YAML file: %s", configFile)
	if _, err := os.Stat(configFile); os.IsNotExist(err) {
		// do nothing, file doesn't exist
		log.Printf("YAML file doesn't exist, skipping: %s", configFile)
		return nil
	}
	yamlFile, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Printf("Couldn't read config file: %s", err)
		return err
	}
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		log.Printf("Couldn't parse config file: %s", err)
		return err
	}

	// Deduplicate filters
	{
		i := 0 // output index, used for deletion later
		urls := map[string]bool{}
		for _, filter := range config.Filters {
			if _, ok := urls[filter.URL]; !ok {
				// we didn't see it before, keep it
				urls[filter.URL] = true // remember the URL
				config.Filters[i] = filter
				i++
			}
		}
		// all entries we want to keep are at front, delete the rest
		config.Filters = config.Filters[:i]
	}

	// Set the next filter ID to max(filter.ID) + 1
	for i := range config.Filters {
		if NextFilterId < config.Filters[i].ID {
			NextFilterId = config.Filters[i].ID + 1
		}
	}

	return nil
}

// Saves configuration to the YAML file and also saves the user filter contents to a file
func writeConfig() error {
	configFile := filepath.Join(config.ourBinaryDir, config.ourConfigFilename)
	log.Printf("Writing YAML file: %s", configFile)
	yamlText, err := yaml.Marshal(&config)
	if err != nil {
		log.Printf("Couldn't generate YAML file: %s", err)
		return err
	}
	err = safeWriteFile(configFile, yamlText)
	if err != nil {
		log.Printf("Couldn't save YAML config: %s", err)
		return err
	}

	userFilter := userFilter()
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
	coreFile := filepath.Join(config.ourBinaryDir, config.CoreDNS.coreFile)
	log.Printf("Writing DNS config: %s", coreFile)
	configText, err := generateCoreDNSConfigText()
	if err != nil {
		log.Printf("Couldn't generate DNS config: %s", err)
		return err
	}
	err = safeWriteFile(coreFile, []byte(configText))
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
		{{if .FilteringEnabled}}{{range .Filters}}{{if and .Enabled .Contents}}
		filter {{.ID}} "{{.Path}}"
		{{end}}{{end}}{{end}}
	}{{end}}
	{{.Pprof}}
	{{if .RefuseAny}}refuseany{{end}}
	{{if gt .Ratelimit 0}}ratelimit {{.Ratelimit}}{{end}}
	hosts {
		fallthrough
	}
	{{if .UpstreamDNS}}upstream {{range .UpstreamDNS}}{{.}} {{end}} { bootstrap {{.BootstrapDNS}} }{{end}}
	{{.Cache}}
	{{.Prometheus}}
}
`

var removeEmptyLines = regexp.MustCompile("([\t ]*\n)+")

// generate CoreDNS config text
func generateCoreDNSConfigText() (string, error) {
	t, err := template.New("config").Parse(coreDNSConfigTemplate)
	if err != nil {
		log.Printf("Couldn't generate DNS config: %s", err)
		return "", err
	}

	var configBytes bytes.Buffer
	temporaryConfig := config.CoreDNS

	// generate temporary filter list, needed to put userfilter in coredns config
	filters := []filter{}

	// first of all, append the user filter
	userFilter := userFilter()

	filters = append(filters, userFilter)

	// then go through other filters
	filters = append(filters, config.Filters...)
	temporaryConfig.Filters = filters

	// run the template
	err = t.Execute(&configBytes, &temporaryConfig)
	if err != nil {
		log.Printf("Couldn't generate DNS config: %s", err)
		return "", err
	}
	configText := configBytes.String()

	// remove empty lines from generated config
	configText = removeEmptyLines.ReplaceAllString(configText, "\n")
	return configText, nil
}
