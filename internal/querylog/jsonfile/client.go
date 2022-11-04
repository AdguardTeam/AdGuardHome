package jsonfile

import "github.com/AdguardTeam/AdGuardHome/internal/querylog/logs"

// clientCacheKey is the key by which a cached client information is found.
type clientCacheKey struct {
	clientID string
	ip       string
}

// clientCache is the cache of client information found throughout a request to
// the query log API.  It is used both to speed up the lookup, as well as to
// make sure that changes in client data between two lookups don't create
// discrepancies in our response.
type clientCache map[clientCacheKey]*logs.Client
