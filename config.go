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
	CoreDNS   coreDNSConfig `yaml:"coredns"`
	Filters   []filter      `yaml:"filters"`
	UserRules []string      `yaml:"user_rules"`

	sync.Mutex `yaml:"-"`
}

type coreDNSConfig struct {
	Port                int `yaml:"port"`
	binaryFile          string
	coreFile            string
	FilterFile          string   `yaml:"-"`
	FilteringEnabled    bool     `yaml:"filtering_enabled"`
	SafeBrowsingEnabled bool     `yaml:"safebrowsing_enabled"`
	SafeSearchEnabled   bool     `yaml:"safesearch_enabled"`
	ParentalEnabled     bool     `yaml:"parental_enabled"`
	ParentalSensitivity int      `yaml:"parental_sensitivity"`
	QueryLogEnabled     bool     `yaml:"querylog_enabled"`
	Pprof               string   `yaml:"pprof"`
	UpstreamDNS         []string `yaml:"upstream_dns"`
	Cache               string   `yaml:"cache"`
	Prometheus          string   `yaml:"prometheus"`
}

type filter struct {
	Enabled     bool   `json:"enabled"`
	URL         string `json:"url"`
	RulesCount  int    `json:"rules_count" yaml:"-"`
	Name        string `json:"name" yaml:"-"`
	contents    []byte
	LastUpdated time.Time `json:"last_updated" yaml:"-"`
}

var defaultDNS = []string{"1.1.1.1", "1.0.0.1"}

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
		SafeBrowsingEnabled: true,
		QueryLogEnabled:     true,
		UpstreamDNS:         defaultDNS,
		Cache:               "cache",
		Prometheus:          "prometheus :9153",
	},
	Filters: []filter{
		{Enabled: true, URL: "https://filters.adtidy.org/windows/filters/15.txt"},
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
	err = ioutil.WriteFile(configfile, yamlText, 0644)
	if err != nil {
		log.Printf("Couldn't write YAML config: %s", err)
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
	err = ioutil.WriteFile(corefile, []byte(configtext), 0644)
	if err != nil {
		log.Printf("Couldn't write DNS config: %s", err)
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
    {{if .FilteringEnabled}}dnsfilter {{.FilterFile}} {
        {{if .SafeBrowsingEnabled}}safebrowsing{{end}}
        {{if .ParentalEnabled}}parental {{.ParentalSensitivity}}{{end}}
        {{if .SafeSearchEnabled}}safesearch{{end}}
        {{if .QueryLogEnabled}}querylog{{end}}
    }{{end}}
    {{.Pprof}}
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
