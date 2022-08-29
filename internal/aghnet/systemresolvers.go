package aghnet

// DefaultRefreshIvl is the default period of time between refreshing cached
// addresses.
// const DefaultRefreshIvl = 5 * time.Minute

// HostGenFunc is the signature for functions generating fake hostnames.  The
// implementation must be safe for concurrent use.
type HostGenFunc func() (host string)

// SystemResolvers helps to work with local resolvers' addresses provided by OS.
type SystemResolvers interface {
	// Get returns the slice of local resolvers' addresses.  It must be safe for
	// concurrent use.
	Get() (rs []string)
	// refresh refreshes the local resolvers' addresses cache.  It must be safe
	// for concurrent use.
	refresh() (err error)
}

// NewSystemResolvers returns a SystemResolvers with the cache refresh rate
// defined by refreshIvl. It disables auto-refreshing if refreshIvl is 0.  If
// nil is passed for hostGenFunc, the default generator will be used.
func NewSystemResolvers(
	hostGenFunc HostGenFunc,
) (sr SystemResolvers, err error) {
	sr = newSystemResolvers(hostGenFunc)

	// Fill cache.
	err = sr.refresh()
	if err != nil {
		return nil, err
	}

	return sr, nil
}
