package filtering

import "github.com/miekg/dns"

// SafeSearch interface describes a service for search engines hosts rewrites.
type SafeSearch interface {
	// CheckHost checks host with safe search filter.  CheckHost must be safe
	// for concurrent use.  qtype must be either [dns.TypeA] or [dns.TypeAAAA].
	CheckHost(host string, qtype uint16) (res Result, err error)

	// Update updates the configuration of the safe search filter.  Update must
	// be safe for concurrent use.  An implementation of Update may ignore some
	// fields, but it must document which.
	Update(conf SafeSearchConfig) (err error)
}

// SafeSearchConfig is a struct with safe search related settings.
type SafeSearchConfig struct {
	// CustomResolver is the resolver used by safe search.
	CustomResolver Resolver `yaml:"-" json:"-"`

	// Enabled indicates if safe search is enabled entirely.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// Services flags.  Each flag indicates if the corresponding service is
	// enabled or disabled.

	Bing       bool `yaml:"bing" json:"bing"`
	DuckDuckGo bool `yaml:"duckduckgo" json:"duckduckgo"`
	Google     bool `yaml:"google" json:"google"`
	Pixabay    bool `yaml:"pixabay" json:"pixabay"`
	Yandex     bool `yaml:"yandex" json:"yandex"`
	YouTube    bool `yaml:"youtube" json:"youtube"`
}

// checkSafeSearch checks host with safe search engine.  Matches
// [hostChecker.check].
func (d *DNSFilter) checkSafeSearch(
	host string,
	qtype uint16,
	setts *Settings,
) (res Result, err error) {
	if !setts.ProtectionEnabled ||
		!setts.SafeSearchEnabled ||
		(qtype != dns.TypeA && qtype != dns.TypeAAAA) {
		return Result{}, nil
	}

	if d.safeSearch == nil {
		return Result{}, nil
	}

	clientSafeSearch := setts.ClientSafeSearch
	if clientSafeSearch != nil {
		return clientSafeSearch.CheckHost(host, qtype)
	}

	return d.safeSearch.CheckHost(host, qtype)
}
