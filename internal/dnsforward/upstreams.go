package dnsforward

import (
	"bytes"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/stringutil"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"
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
func (s *Server) prepareUpstreamSettings() (err error) {
	// Load upstreams either from the file, or from the settings
	var upstreams []string
	upstreams, err = s.loadUpstreams()
	if err != nil {
		return fmt.Errorf("loading upstreams: %w", err)
	}

	s.conf.UpstreamConfig, err = s.prepareUpstreamConfig(upstreams, defaultDNS, &upstream.Options{
		Bootstrap:    s.conf.BootstrapDNS,
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

	// dnsFilter can be nil during application update.
	if s.dnsFilter != nil {
		err = s.replaceUpstreamsWithHosts(uc, opts)
		if err != nil {
			return nil, fmt.Errorf("resolving upstreams with hosts: %w", err)
		}
	}

	return uc, nil
}

// replaceUpstreamsWithHosts replaces unique upstreams with their resolved
// versions based on the system hosts file.
//
// TODO(e.burkov):  This should be performed inside dnsproxy, which should
// actually consider /etc/hosts.  See TODO on [aghnet.HostsContainer].
func (s *Server) replaceUpstreamsWithHosts(
	upsConf *proxy.UpstreamConfig,
	opts *upstream.Options,
) (err error) {
	resolved := map[string]*upstream.Options{}

	err = s.resolveUpstreamsWithHosts(resolved, upsConf.Upstreams, opts)
	if err != nil {
		return fmt.Errorf("resolving upstreams: %w", err)
	}

	hosts := maps.Keys(upsConf.DomainReservedUpstreams)
	// TODO(e.burkov):  Think of extracting sorted range into an util function.
	slices.Sort(hosts)
	for _, host := range hosts {
		err = s.resolveUpstreamsWithHosts(resolved, upsConf.DomainReservedUpstreams[host], opts)
		if err != nil {
			return fmt.Errorf("resolving upstreams reserved for %s: %w", host, err)
		}
	}

	hosts = maps.Keys(upsConf.SpecifiedDomainUpstreams)
	slices.Sort(hosts)
	for _, host := range hosts {
		err = s.resolveUpstreamsWithHosts(resolved, upsConf.SpecifiedDomainUpstreams[host], opts)
		if err != nil {
			return fmt.Errorf("resolving upstreams specific for %s: %w", host, err)
		}
	}

	return nil
}

// resolveUpstreamsWithHosts resolves the IP addresses of each of the upstreams
// and replaces those both in upstreams and resolved.  Upstreams that failed to
// resolve are placed to resolved as-is.  This function only returns error of
// upstreams closing.
func (s *Server) resolveUpstreamsWithHosts(
	resolved map[string]*upstream.Options,
	upstreams []upstream.Upstream,
	opts *upstream.Options,
) (err error) {
	for i := range upstreams {
		u := upstreams[i]
		addr := u.Address()
		host := extractUpstreamHost(addr)

		withIPs, ok := resolved[host]
		if !ok {
			recs := s.dnsFilter.EtcHostsRecords(host)
			if len(recs) == 0 {
				resolved[host] = nil

				return nil
			}

			withIPs = opts.Clone()
			withIPs.ServerIPAddrs = make([]net.IP, 0, len(recs))
			for _, rec := range recs {
				withIPs.ServerIPAddrs = append(withIPs.ServerIPAddrs, rec.Addr.AsSlice())
			}

			sortNetIPAddrs(withIPs.ServerIPAddrs, opts.PreferIPv6)
			resolved[host] = withIPs
		} else if withIPs == nil {
			continue
		}

		if err = u.Close(); err != nil {
			return fmt.Errorf("closing upstream %s: %w", addr, err)
		}

		upstreams[i], err = upstream.AddressToUpstream(addr, withIPs)
		if err != nil {
			return fmt.Errorf("replacing upstream %s with resolved %s: %w", addr, host, err)
		}

		log.Debug("dnsforward: using %s for %s", withIPs.ServerIPAddrs, upstreams[i].Address())
	}

	return nil
}

// extractUpstreamHost returns the hostname of addr without port with an
// assumption that any address passed here has already been successfully parsed
// by [upstream.AddressToUpstream].  This function essentially mirrors the logic
// of [upstream.AddressToUpstream], see TODO on [replaceUpstreamsWithHosts].
func extractUpstreamHost(addr string) (host string) {
	var err error
	if strings.Contains(addr, "://") {
		var u *url.URL
		u, err = url.Parse(addr)
		if err != nil {
			log.Debug("dnsforward: parsing upstream %s: %s", addr, err)

			return addr
		}

		return u.Hostname()
	}

	// Probably, plain UDP upstream defined by address or address:port.
	host, err = netutil.SplitHost(addr)
	if err != nil {
		return addr
	}

	return host
}

// sortNetIPAddrs sorts addrs in accordance with the protocol preferences.
// Invalid addresses are sorted near the end.
//
// TODO(e.burkov):  This function taken from dnsproxy, which also already
// contains a few similar functions.  Think of moving to golibs.
func sortNetIPAddrs(addrs []net.IP, preferIPv6 bool) {
	l := len(addrs)
	if l <= 1 {
		return
	}

	slices.SortStableFunc(addrs, func(addrA, addrB net.IP) (res int) {
		switch len(addrA) {
		case net.IPv4len, net.IPv6len:
			switch len(addrB) {
			case net.IPv4len, net.IPv6len:
				// Go on.
			default:
				return -1
			}
		default:
			return 1
		}

		// Treat IPv6-mapped IPv4 addresses as IPv6 addresses.
		aIs4, bIs4 := addrA.To4() != nil, addrB.To4() != nil
		if aIs4 == bIs4 {
			return bytes.Compare(addrA, addrB)
		}

		if aIs4 {
			if preferIPv6 {
				return 1
			}

			return -1
		}

		if preferIPv6 {
			return -1
		}

		return 1
	})
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
	allServers bool,
	fastestAddr bool,
	fastestTimeout time.Duration,
) {
	if allServers {
		conf.UpstreamMode = proxy.UModeParallel
	} else if fastestAddr {
		conf.UpstreamMode = proxy.UModeFastestAddr
		conf.FastestPingTimeout = fastestTimeout
	} else {
		conf.UpstreamMode = proxy.UModeLoadBalance
	}
}
