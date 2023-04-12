package querylog

// Client is the information required by the query log to match against clients
// during searches.
type Client struct {
	WHOIS          *ClientWHOIS `json:"whois,omitempty"`
	Name           string       `json:"name"`
	DisallowedRule string       `json:"disallowed_rule"`
	Disallowed     bool         `json:"disallowed"`
	IgnoreQueryLog bool         `json:"-"`
}

// ClientWHOIS is the filtered WHOIS data for the client.
//
// TODO(a.garipov): Merge with home.RuntimeClientWHOISInfo after the
// refactoring is done.
type ClientWHOIS struct {
	City    string `json:"city,omitempty"`
	Country string `json:"country,omitempty"`
	Orgname string `json:"orgname,omitempty"`
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
