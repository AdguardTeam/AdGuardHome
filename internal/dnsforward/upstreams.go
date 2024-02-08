package dnsforward

import (
	"fmt"
	"net/netip"
	"os"
	"slices"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/stringutil"
	"golang.org/x/exp/maps"
)

// loadUpstreams parses upstream DNS servers from the configured file or from
// the configuration itself.
func (s *Server) loadUpstreams() (upstreams []string, err error) {
	if s.conf.UpstreamDNSFileName == "" {
		return stringutil.FilterOut(s.conf.UpstreamDNS, IsCommentOrEmpty), nil
	}

	var data []byte
	data, err = os.ReadFile(s.conf.UpstreamDNSFileName)
	if err != nil {
		return nil, fmt.Errorf("reading upstream from file: %w", err)
	}

	upstreams = stringutil.SplitTrimmed(string(data), "\n")

	log.Debug("dnsforward: got %d upstreams in %q", len(upstreams), s.conf.UpstreamDNSFileName)

	return stringutil.FilterOut(upstreams, IsCommentOrEmpty), nil
}

// prepareUpstreamSettings sets upstream DNS server settings.
func (s *Server) prepareUpstreamSettings(boot upstream.Resolver) (err error) {
	// Load upstreams either from the file, or from the settings
	var upstreams []string
	upstreams, err = s.loadUpstreams()
	if err != nil {
		return fmt.Errorf("loading upstreams: %w", err)
	}

	s.conf.UpstreamConfig, err = s.prepareUpstreamConfig(upstreams, defaultDNS, &upstream.Options{
		Bootstrap:    boot,
		Timeout:      s.conf.UpstreamTimeout,
		HTTPVersions: UpstreamHTTPVersions(s.conf.UseHTTP3Upstreams),
		PreferIPv6:   s.conf.BootstrapPreferIPv6,
		// Use a customized set of RootCAs, because Go's default mechanism of
		// loading TLS roots does not always work properly on some routers so we're
		// loading roots manually and pass it here.
		//
		// See [aghtls.SystemRootCAs].
		//
		// TODO(a.garipov): Investigate if that's true.
		RootCAs:      s.conf.TLSv12Roots,
		CipherSuites: s.conf.TLSCiphers,
	})
	if err != nil {
		return fmt.Errorf("preparing upstream config: %w", err)
	}

	return nil
}

// prepareUpstreamConfig returns the upstream configuration based on upstreams
// and configuration of s.
func (s *Server) prepareUpstreamConfig(
	upstreams []string,
	defaultUpstreams []string,
	opts *upstream.Options,
) (uc *proxy.UpstreamConfig, err error) {
	uc, err = proxy.ParseUpstreamsConfig(upstreams, opts)
	if err != nil {
		return nil, fmt.Errorf("parsing upstream config: %w", err)
	}

	if len(uc.Upstreams) == 0 && defaultUpstreams != nil {
		log.Info("dnsforward: warning: no default upstreams specified, using %v", defaultUpstreams)
		var defaultUpstreamConfig *proxy.UpstreamConfig
		defaultUpstreamConfig, err = proxy.ParseUpstreamsConfig(defaultUpstreams, opts)
		if err != nil {
			return nil, fmt.Errorf("parsing default upstreams: %w", err)
		}

		uc.Upstreams = defaultUpstreamConfig.Upstreams
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
		conf.UpstreamMode = proxy.UModeParallel
	case UpstreamModeFastestAddr:
		conf.UpstreamMode = proxy.UModeFastestAddr
		conf.FastestPingTimeout = fastestTimeout
	case UpstreamModeLoadBalance:
		conf.UpstreamMode = proxy.UModeLoadBalance
	default:
		return fmt.Errorf("unexpected value %q", upstreamMode)
	}

	return nil
}

// createBootstrap returns a bootstrap resolver based on the configuration of s.
// boots are the upstream resolvers that should be closed after use.  r is the
// actual bootstrap resolver, which may include the system hosts.
//
// TODO(e.burkov):  This function currently returns a resolver and a slice of
// the upstream resolvers, which are essentially the same.  boots are returned
// for being able to close them afterwards, but it introduces an implicit
// contract that r could only be used before that.  Anyway, this code should
// improve when the [proxy.UpstreamConfig] will become an [upstream.Resolver]
// and be used here.
func (s *Server) createBootstrap(
	addrs []string,
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

	if s.etcHosts != nil {
		r = upstream.ConsequentResolver{s.etcHosts, parallel}
	} else {
		r = parallel
	}

	return r, boots, nil
}

// IsCommentOrEmpty returns true if s starts with a "#" character or is empty.
// This function is useful for filtering out non-upstream lines from upstream
// configs.
func IsCommentOrEmpty(s string) (ok bool) {
	return len(s) == 0 || s[0] == '#'
}

// ValidateUpstreamsPrivate validates each upstream and returns an error if any
// upstream is invalid or if there are no default upstreams specified.  It also
// checks each domain of domain-specific upstreams for being ARPA pointing to
// a locally-served network.  privateNets must not be nil.
func ValidateUpstreamsPrivate(upstreams []string, privateNets netutil.SubnetSet) (err error) {
	conf, err := proxy.ParseUpstreamsConfig(upstreams, &upstream.Options{})
	if err != nil {
		return fmt.Errorf("creating config: %w", err)
	}

	if conf == nil {
		return nil
	}

	keys := maps.Keys(conf.DomainReservedUpstreams)
	slices.Sort(keys)

	var errs []error
	for _, domain := range keys {
		var subnet netip.Prefix
		subnet, err = extractARPASubnet(domain)
		if err != nil {
			errs = append(errs, err)

			continue
		}

		if !privateNets.Contains(subnet.Addr()) {
			errs = append(
				errs,
				fmt.Errorf("arpa domain %q should point to a locally-served network", domain),
			)
		}
	}

	return errors.Annotate(errors.Join(errs...), "checking domain-specific upstreams: %w")
}
