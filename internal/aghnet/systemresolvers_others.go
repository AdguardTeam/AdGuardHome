// +build !windows

package aghnet

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/agherr"
	"github.com/AdguardTeam/AdGuardHome/internal/aghstrings"
)

// defaultHostGen is the default method of generating host for Refresh.
func defaultHostGen() (host string) {
	// TODO(e.burkov): Use strings.Builder.
	return fmt.Sprintf("test%d.org", time.Now().UnixNano())
}

// systemResolvers is a default implementation of SystemResolvers interface.
type systemResolvers struct {
	resolver    *net.Resolver
	hostGenFunc HostGenFunc

	// addrs is the set that contains cached local resolvers' addresses.
	addrs     *aghstrings.Set
	addrsLock sync.RWMutex
}

func (sr *systemResolvers) refresh() (err error) {
	defer agherr.Annotate("systemResolvers: %w", &err)

	_, err = sr.resolver.LookupHost(context.Background(), sr.hostGenFunc())
	dnserr := &net.DNSError{}
	if errors.As(err, &dnserr) && dnserr.Err == fakeDialErr.Error() {
		return nil
	}

	return err
}

func newSystemResolvers(refreshIvl time.Duration, hostGenFunc HostGenFunc) (sr SystemResolvers) {
	if hostGenFunc == nil {
		hostGenFunc = defaultHostGen
	}
	s := &systemResolvers{
		resolver: &net.Resolver{
			PreferGo: true,
		},
		hostGenFunc: hostGenFunc,
		addrs:       aghstrings.NewSet(),
	}
	s.resolver.Dial = s.dialFunc

	return s
}

// dialFunc gets the resolver's address and puts it into internal cache.
func (sr *systemResolvers) dialFunc(_ context.Context, _, address string) (_ net.Conn, err error) {
	// Just validate the passed address is a valid IP.
	var host string
	host, err = SplitHost(address)
	if err != nil {
		// TODO(e.burkov): Maybe use a structured badAddrPassedErr to
		// allow unwrapping of the real error.
		return nil, fmt.Errorf("%s: %w", err, badAddrPassedErr)
	}

	if net.ParseIP(host) == nil {
		return nil, fmt.Errorf("parsing %q: %w", host, badAddrPassedErr)
	}

	sr.addrsLock.Lock()
	defer sr.addrsLock.Unlock()

	sr.addrs.Add(host)

	return nil, fakeDialErr
}

func (sr *systemResolvers) Get() (rs []string) {
	sr.addrsLock.RLock()
	defer sr.addrsLock.RUnlock()

	return sr.addrs.Values()
}
