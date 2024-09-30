package dnsforward

import (
	"fmt"
	"slices"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/stringutil"
)

// newBootstrap returns a bootstrap resolver based on the configuration of s.
// boots are the upstream resolvers that should be closed after use.  r is the
// actual bootstrap resolver, which may include the system hosts.
//
// TODO(e.burkov):  This function currently returns a resolver and a slice of
// the upstream resolvers, which are essentially the same.  boots are returned
// for being able to close them afterwards, but it introduces an implicit
// contract that r could only be used before that.  Anyway, this code should
// improve when the [proxy.UpstreamConfig] will become an [upstream.Resolver]
// and be used here.
func newBootstrap(
	addrs []string,
	etcHosts upstream.Resolver,
	opts *upstream.Options,
) (r upstream.Resolver, boots []*upstream.UpstreamResolver, err error) {
	if len(addrs) == 0 {
		addrs = defaultBootstrap
	}

	boots, err = aghnet.ParseBootstraps(addrs, opts)
	if err != nil {
		// Don't wrap the error, since it's informative enough as is.
		return nil, nil, err
	}

	var parallel upstream.ParallelResolver
	for _, b := range boots {
		parallel = append(parallel, upstream.NewCachingResolver(b))
	}

	if etcHosts != nil {
		r = upstream.ConsequentResolver{etcHosts, parallel}
	} else {
		r = parallel
	}

	return r, boots, nil
}

// newUpstreamConfig returns the upstream configuration based on upstreams.  If
// upstreams slice specifies no default upstreams, defaultUpstreams are used to
// create upstreams with no domain specifications.  opts are used when creating
// upstream configuration.
func newUpstreamConfig(
	upstreams []string,
	defaultUpstreams []string,
	opts *upstream.Options,
) (uc *proxy.UpstreamConfig, err error) {
	uc, err = proxy.ParseUpstreamsConfig(upstreams, opts)
	if err != nil {
		return uc, fmt.Errorf("parsing upstreams: %w", err)
	}

	if len(uc.Upstreams) == 0 && len(defaultUpstreams) > 0 {
		log.Info("dnsforward: warning: no default upstreams specified, using %v", defaultUpstreams)

		var defaultUpstreamConfig *proxy.UpstreamConfig
		defaultUpstreamConfig, err = proxy.ParseUpstreamsConfig(defaultUpstreams, opts)
		if err != nil {
			return uc, fmt.Errorf("parsing default upstreams: %w", err)
		}

		uc.Upstreams = defaultUpstreamConfig.Upstreams
	}

	return uc, nil
}

// newPrivateConfig creates an upstream configuration for resolving PTR records
// for local addresses.  The configuration is built either from the provided
// addresses or from the system resolvers.  unwanted filters the resulting
// upstream configuration.
func newPrivateConfig(
	addrs []string,
	unwanted addrPortSet,
	sysResolvers SystemResolvers,
	privateNets netutil.SubnetSet,
	opts *upstream.Options,
) (uc *proxy.UpstreamConfig, err error) {
	confNeedsFiltering := len(addrs) > 0
	if confNeedsFiltering {
		addrs = stringutil.FilterOut(addrs, IsCommentOrEmpty)
	} else {
		sysResolvers := slices.DeleteFunc(slices.Clone(sysResolvers.Addrs()), unwanted.Has)
		addrs = make([]string, 0, len(sysResolvers))
		for _, r := range sysResolvers {
			addrs = append(addrs, r.String())
		}
	}

	log.Debug("dnsforward: private-use upstreams: %v", addrs)

	uc, err = proxy.ParseUpstreamsConfig(addrs, opts)
	if err != nil {
		return uc, fmt.Errorf("preparing private upstreams: %w", err)
	}

	if confNeedsFiltering {
		err = filterOutAddrs(uc, unwanted)
		if err != nil {
			return uc, fmt.Errorf("filtering private upstreams: %w", err)
		}
	}

	// Prevalidate the config to catch the exact error before creating proxy.
	// See TODO on [PrivateRDNSError].
	err = proxy.ValidatePrivateConfig(uc, privateNets)
	if err != nil {
		return uc, &PrivateRDNSError{err: err}
	}

	return uc, nil
}

// UpstreamHTTPVersions returns the HTTP versions for upstream configuration
// depending on configuration.
func UpstreamHTTPVersions(http3 bool) (v []upstream.HTTPVersion) {
	if !http3 {
		return upstream.DefaultHTTPVersions
	}

	return []upstream.HTTPVersion{
		upstream.HTTPVersion3,
		upstream.HTTPVersion2,
		upstream.HTTPVersion11,
	}
}

// setProxyUpstreamMode sets the upstream mode and related settings in conf
// based on provided parameters.
func setProxyUpstreamMode(
	conf *proxy.Config,
	upstreamMode UpstreamMode,
	fastestTimeout time.Duration,
) (err error) {
	switch upstreamMode {
	case UpstreamModeParallel:
		conf.UpstreamMode = proxy.UpstreamModeParallel
	case UpstreamModeFastestAddr:
		conf.UpstreamMode = proxy.UpstreamModeFastestAddr
		conf.FastestPingTimeout = fastestTimeout
	case UpstreamModeLoadBalance:
		conf.UpstreamMode = proxy.UpstreamModeLoadBalance
	default:
		return fmt.Errorf("unexpected value %q", upstreamMode)
	}

	return nil
}

// IsCommentOrEmpty returns true if s starts with a "#" character or is empty.
// This function is useful for filtering out non-upstream lines from upstream
// configs.
func IsCommentOrEmpty(s string) (ok bool) {
	return len(s) == 0 || s[0] == '#'
}
