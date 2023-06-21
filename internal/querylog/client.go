package querylog

import "github.com/AdguardTeam/AdGuardHome/internal/whois"

// Client is the information required by the query log to match against clients
// during searches.
type Client struct {
	WHOIS          *whois.Info `json:"whois,omitempty"`
	Name           string      `json:"name"`
	DisallowedRule string      `json:"disallowed_rule"`
	Disallowed     bool        `json:"disallowed"`
	IgnoreQueryLog bool        `json:"-"`
}

// clientCacheKey is the key by which a cached client information is found.
type clientCacheKey struct {
	clientID string
	ip       string
}

// clientCache is the cache of client information found throughout a request to
// the query log API.  It is used both to speed up the lookup, as well as to
// make sure that changes in client data between two lookups don't create
// discrepancies in our response.
type clientCache map[clientCacheKey]*Client
