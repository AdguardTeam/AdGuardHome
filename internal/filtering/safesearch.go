package filtering

import "context"

// SafeSearch interface describes a service for search engines hosts rewrites.
type SafeSearch interface {
	// CheckHost checks host with safe search filter.  CheckHost must be safe
	// for concurrent use.  qtype must be either [dns.TypeA] or [dns.TypeAAAA].
	CheckHost(ctx context.Context, host string, qtype uint16) (res Result, err error)

	// Update updates the configuration of the safe search filter.  Update must
	// be safe for concurrent use.  An implementation of Update may ignore some
	// fields, but it must document which.
	Update(ctx context.Context, conf SafeSearchConfig) (err error)
}

// SafeSearchConfig is a struct with safe search related settings.
type SafeSearchConfig struct {
	// Enabled indicates if safe search is enabled entirely.
	Enabled bool `yaml:"enabled" json:"enabled"`

	// Services flags.  Each flag indicates if the corresponding service is
	// enabled or disabled.

	Bing       bool `yaml:"bing" json:"bing"`
	DuckDuckGo bool `yaml:"duckduckgo" json:"duckduckgo"`
	Ecosia     bool `yaml:"ecosia" json:"ecosia"`
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
	if d.safeSearch == nil || !setts.ProtectionEnabled || !setts.SafeSearchEnabled {
		return Result{}, nil
	}

	// TODO(s.chzhen):  Pass context.
	ctx := context.TODO()

	clientSafeSearch := setts.ClientSafeSearch
	if clientSafeSearch != nil {
		return clientSafeSearch.CheckHost(ctx, host, qtype)
	}

	return d.safeSearch.CheckHost(ctx, host, qtype)
}
