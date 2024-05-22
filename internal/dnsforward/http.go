package dnsforward

import (
	"cmp"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"slices"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/stringutil"
)

// jsonDNSConfig is the JSON representation of the DNS server configuration.
//
// TODO(s.chzhen):  Split it into smaller pieces.  Use aghalg.NullBool instead
// of *bool.
type jsonDNSConfig struct {
	// Upstreams is the list of upstream DNS servers.
	Upstreams *[]string `json:"upstream_dns"`

	// UpstreamsFile is the file containing upstream DNS servers.
	UpstreamsFile *string `json:"upstream_dns_file"`

	// Bootstraps is the list of DNS servers resolving IP addresses of the
	// upstream DoH/DoT resolvers.
	Bootstraps *[]string `json:"bootstrap_dns"`

	// Fallbacks is the list of fallback DNS servers used when upstream DNS
	// servers are not responding.
	Fallbacks *[]string `json:"fallback_dns"`

	// ProtectionEnabled defines if protection is enabled.
	ProtectionEnabled *bool `json:"protection_enabled"`

	// Ratelimit is the number of requests per second allowed per client.
	Ratelimit *uint32 `json:"ratelimit"`

	// RatelimitSubnetLenIPv4 is a subnet length for IPv4 addresses used for
	// rate limiting requests.
	RatelimitSubnetLenIPv4 *int `json:"ratelimit_subnet_len_ipv4"`

	// RatelimitSubnetLenIPv6 is a subnet length for IPv6 addresses used for
	// rate limiting requests.
	RatelimitSubnetLenIPv6 *int `json:"ratelimit_subnet_len_ipv6"`

	// RatelimitWhitelist is a list of IP addresses excluded from rate limiting.
	RatelimitWhitelist *[]netip.Addr `json:"ratelimit_whitelist"`

	// BlockingMode defines the way blocked responses are constructed.
	BlockingMode *filtering.BlockingMode `json:"blocking_mode"`

	// EDNSCSEnabled defines if EDNS Client Subnet is enabled.
	EDNSCSEnabled *bool `json:"edns_cs_enabled"`

	// EDNSCSUseCustom defines if EDNSCSCustomIP should be used.
	EDNSCSUseCustom *bool `json:"edns_cs_use_custom"`

	// DNSSECEnabled defines if DNSSEC is enabled.
	DNSSECEnabled *bool `json:"dnssec_enabled"`

	// DisableIPv6 defines if IPv6 addresses should be dropped.
	DisableIPv6 *bool `json:"disable_ipv6"`

	// UpstreamMode defines the way DNS requests are constructed.
	UpstreamMode *jsonUpstreamMode `json:"upstream_mode"`

	// BlockedResponseTTL is the TTL for blocked responses.
	BlockedResponseTTL *uint32 `json:"blocked_response_ttl"`

	// CacheSize in bytes.
	CacheSize *uint32 `json:"cache_size"`

	// CacheMinTTL is custom minimum TTL for cached DNS responses.
	CacheMinTTL *uint32 `json:"cache_ttl_min"`

	// CacheMaxTTL is custom maximum TTL for cached DNS responses.
	CacheMaxTTL *uint32 `json:"cache_ttl_max"`

	// CacheOptimistic defines if expired entries should be served.
	CacheOptimistic *bool `json:"cache_optimistic"`

	// ResolveClients defines if clients IPs should be resolved into hostnames.
	ResolveClients *bool `json:"resolve_clients"`

	// UsePrivateRDNS defines if privates DNS resolvers should be used.
	UsePrivateRDNS *bool `json:"use_private_ptr_resolvers"`

	// LocalPTRUpstreams is the list of local private DNS resolvers.
	LocalPTRUpstreams *[]string `json:"local_ptr_upstreams"`

	// BlockingIPv4 is custom IPv4 address for blocked A requests.
	BlockingIPv4 netip.Addr `json:"blocking_ipv4"`

	// BlockingIPv6 is custom IPv6 address for blocked AAAA requests.
	BlockingIPv6 netip.Addr `json:"blocking_ipv6"`

	// DisabledUntil is a timestamp until when the protection is disabled.
	DisabledUntil *time.Time `json:"protection_disabled_until"`

	// EDNSCSCustomIP is custom IP for EDNS Client Subnet.
	EDNSCSCustomIP netip.Addr `json:"edns_cs_custom_ip"`

	// DefaultLocalPTRUpstreams is used to pass the addresses from
	// systemResolvers to the front-end.  It's not a pointer to the slice since
	// there is no need to omit it while decoding from JSON.
	DefaultLocalPTRUpstreams []string `json:"default_local_ptr_upstreams,omitempty"`
}

// jsonUpstreamMode is a enumeration of upstream modes.
type jsonUpstreamMode string

const (
	// jsonUpstreamModeEmpty is the default value on frontend, it is used as
	// jsonUpstreamModeLoadBalance mode.
	//
	// Deprecated: Use jsonUpstreamModeLoadBalance instead.
	jsonUpstreamModeEmpty jsonUpstreamMode = ""

	jsonUpstreamModeLoadBalance jsonUpstreamMode = "load_balance"
	jsonUpstreamModeParallel    jsonUpstreamMode = "parallel"
	jsonUpstreamModeFastestAddr jsonUpstreamMode = "fastest_addr"
)

func (s *Server) getDNSConfig() (c *jsonDNSConfig) {
	protectionEnabled, protectionDisabledUntil := s.UpdatedProtectionStatus()

	s.serverLock.RLock()
	defer s.serverLock.RUnlock()

	upstreams := stringutil.CloneSliceOrEmpty(s.conf.UpstreamDNS)
	upstreamFile := s.conf.UpstreamDNSFileName
	bootstraps := stringutil.CloneSliceOrEmpty(s.conf.BootstrapDNS)
	fallbacks := stringutil.CloneSliceOrEmpty(s.conf.FallbackDNS)
	blockingMode, blockingIPv4, blockingIPv6 := s.dnsFilter.BlockingMode()
	blockedResponseTTL := s.dnsFilter.BlockedResponseTTL()
	ratelimit := s.conf.Ratelimit
	ratelimitSubnetLenIPv4 := s.conf.RatelimitSubnetLenIPv4
	ratelimitSubnetLenIPv6 := s.conf.RatelimitSubnetLenIPv6
	ratelimitWhitelist := append([]netip.Addr{}, s.conf.RatelimitWhitelist...)

	customIP := s.conf.EDNSClientSubnet.CustomIP
	enableEDNSClientSubnet := s.conf.EDNSClientSubnet.Enabled
	useCustom := s.conf.EDNSClientSubnet.UseCustom

	enableDNSSEC := s.conf.EnableDNSSEC
	aaaaDisabled := s.conf.AAAADisabled
	cacheSize := s.conf.CacheSize
	cacheMinTTL := s.conf.CacheMinTTL
	cacheMaxTTL := s.conf.CacheMaxTTL
	cacheOptimistic := s.conf.CacheOptimistic
	resolveClients := s.conf.AddrProcConf.UseRDNS
	usePrivateRDNS := s.conf.UsePrivateRDNS
	localPTRUpstreams := stringutil.CloneSliceOrEmpty(s.conf.LocalPTRResolvers)

	var upstreamMode jsonUpstreamMode
	switch s.conf.UpstreamMode {
	case UpstreamModeLoadBalance:
		// TODO(d.kolyshev): Support jsonUpstreamModeLoadBalance on frontend instead
		// of jsonUpstreamModeEmpty.
		upstreamMode = jsonUpstreamModeEmpty
	case UpstreamModeParallel:
		upstreamMode = jsonUpstreamModeParallel
	case UpstreamModeFastestAddr:
		upstreamMode = jsonUpstreamModeFastestAddr
	}

	defPTRUps, err := s.defaultLocalPTRUpstreams()
	if err != nil {
		log.Error("dnsforward: %s", err)
	}

	return &jsonDNSConfig{
		Upstreams:                &upstreams,
		UpstreamsFile:            &upstreamFile,
		Bootstraps:               &bootstraps,
		Fallbacks:                &fallbacks,
		ProtectionEnabled:        &protectionEnabled,
		BlockingMode:             &blockingMode,
		BlockingIPv4:             blockingIPv4,
		BlockingIPv6:             blockingIPv6,
		Ratelimit:                &ratelimit,
		RatelimitSubnetLenIPv4:   &ratelimitSubnetLenIPv4,
		RatelimitSubnetLenIPv6:   &ratelimitSubnetLenIPv6,
		RatelimitWhitelist:       &ratelimitWhitelist,
		EDNSCSCustomIP:           customIP,
		EDNSCSEnabled:            &enableEDNSClientSubnet,
		EDNSCSUseCustom:          &useCustom,
		DNSSECEnabled:            &enableDNSSEC,
		DisableIPv6:              &aaaaDisabled,
		BlockedResponseTTL:       &blockedResponseTTL,
		CacheSize:                &cacheSize,
		CacheMinTTL:              &cacheMinTTL,
		CacheMaxTTL:              &cacheMaxTTL,
		CacheOptimistic:          &cacheOptimistic,
		UpstreamMode:             &upstreamMode,
		ResolveClients:           &resolveClients,
		UsePrivateRDNS:           &usePrivateRDNS,
		LocalPTRUpstreams:        &localPTRUpstreams,
		DefaultLocalPTRUpstreams: defPTRUps,
		DisabledUntil:            protectionDisabledUntil,
	}
}

// defaultLocalPTRUpstreams returns the list of default local PTR resolvers
// filtered of AdGuard Home's own DNS server addresses.  It may appear empty.
func (s *Server) defaultLocalPTRUpstreams() (ups []string, err error) {
	matcher, err := s.conf.ourAddrsSet()
	if err != nil {
		// Don't wrap the error because it's informative enough as is.
		return nil, err
	}

	sysResolvers := slices.DeleteFunc(slices.Clone(s.sysResolvers.Addrs()), matcher.Has)
	ups = make([]string, 0, len(sysResolvers))
	for _, r := range sysResolvers {
		ups = append(ups, r.String())
	}

	return ups, nil
}

// handleGetConfig handles requests to the GET /control/dns_info endpoint.
func (s *Server) handleGetConfig(w http.ResponseWriter, r *http.Request) {
	resp := s.getDNSConfig()
	aghhttp.WriteJSONResponseOK(w, r, resp)
}

// checkBlockingMode returns an error if blocking mode is invalid.
func (req *jsonDNSConfig) checkBlockingMode() (err error) {
	if req.BlockingMode == nil {
		return nil
	}

	return validateBlockingMode(*req.BlockingMode, req.BlockingIPv4, req.BlockingIPv6)
}

// checkUpstreamMode returns an error if the upstream mode is invalid.
func (req *jsonDNSConfig) checkUpstreamMode() (err error) {
	if req.UpstreamMode == nil {
		return nil
	}

	switch um := *req.UpstreamMode; um {
	case
		jsonUpstreamModeEmpty,
		jsonUpstreamModeLoadBalance,
		jsonUpstreamModeParallel,
		jsonUpstreamModeFastestAddr:
		return nil
	default:
		return fmt.Errorf("upstream_mode: incorrect value %q", um)
	}
}

// validate returns an error if any field of req is invalid.
//
// TODO(s.chzhen):  Parse, don't validate.
func (req *jsonDNSConfig) validate(
	ownAddrs addrPortSet,
	sysResolvers SystemResolvers,
	privateNets netutil.SubnetSet,
) (err error) {
	defer func() { err = errors.Annotate(err, "validating dns config: %w") }()

	err = req.validateUpstreamDNSServers(ownAddrs, sysResolvers, privateNets)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return err
	}

	err = req.checkRatelimitSubnetMaskLen()
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return err
	}

	err = req.checkBlockingMode()
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return err
	}

	err = req.checkUpstreamMode()
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return err
	}

	err = req.checkCacheTTL()
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return err
	}

	return nil
}

// checkBootstrap returns an error if any bootstrap address is invalid.
func (req *jsonDNSConfig) checkBootstrap() (err error) {
	if req.Bootstraps == nil {
		return nil
	}

	var b string
	defer func() { err = errors.Annotate(err, "checking bootstrap %s: %w", b) }()

	for _, b = range *req.Bootstraps {
		if b == "" {
			return errors.Error("empty")
		}

		var resolver *upstream.UpstreamResolver
		if resolver, err = upstream.NewUpstreamResolver(b, nil); err != nil {
			// Don't wrap the error because it's informative enough as is.
			return err
		}

		if err = resolver.Close(); err != nil {
			return fmt.Errorf("closing %s: %w", b, err)
		}
	}

	return nil
}

// containsPrivateRDNS returns true if req contains private RDNS settings and
// should be validated.
func (req *jsonDNSConfig) containsPrivateRDNS() (ok bool) {
	return (req.UsePrivateRDNS != nil && *req.UsePrivateRDNS) ||
		(req.LocalPTRUpstreams != nil && len(*req.LocalPTRUpstreams) > 0)
}

// checkPrivateRDNS returns an error if the configuration of the private RDNS is
// not valid.
func (req *jsonDNSConfig) checkPrivateRDNS(
	ownAddrs addrPortSet,
	sysResolvers SystemResolvers,
	privateNets netutil.SubnetSet,
) (err error) {
	if !req.containsPrivateRDNS() {
		return nil
	}

	addrs := cmp.Or(req.LocalPTRUpstreams, &[]string{})

	uc, err := newPrivateConfig(*addrs, ownAddrs, sysResolvers, privateNets, &upstream.Options{})
	err = errors.WithDeferred(err, uc.Close())
	if err != nil {
		return fmt.Errorf("private upstream servers: %w", err)
	}

	return nil
}

// validateUpstreamDNSServers returns an error if any field of req is invalid.
func (req *jsonDNSConfig) validateUpstreamDNSServers(
	ownAddrs addrPortSet,
	sysResolvers SystemResolvers,
	privateNets netutil.SubnetSet,
) (err error) {
	var uc *proxy.UpstreamConfig
	opts := &upstream.Options{}

	if req.Upstreams != nil {
		uc, err = proxy.ParseUpstreamsConfig(*req.Upstreams, opts)
		err = errors.WithDeferred(err, uc.Close())
		if err != nil {
			return fmt.Errorf("upstream servers: %w", err)
		}
	}

	err = req.checkPrivateRDNS(ownAddrs, sysResolvers, privateNets)
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return err
	}

	err = req.checkBootstrap()
	if err != nil {
		// Don't wrap the error since it's informative enough as is.
		return err
	}

	if req.Fallbacks != nil {
		uc, err = proxy.ParseUpstreamsConfig(*req.Fallbacks, opts)
		err = errors.WithDeferred(err, uc.Close())
		if err != nil {
			return fmt.Errorf("fallback servers: %w", err)
		}
	}

	return nil
}

// checkCacheTTL returns an error if the configuration of the cache TTL is
// invalid.
func (req *jsonDNSConfig) checkCacheTTL() (err error) {
	if req.CacheMinTTL == nil && req.CacheMaxTTL == nil {
		return nil
	}

	var minTTL, maxTTL uint32
	if req.CacheMinTTL != nil {
		minTTL = *req.CacheMinTTL
	}

	if req.CacheMaxTTL != nil {
		maxTTL = *req.CacheMaxTTL
	}

	return validateCacheTTL(minTTL, maxTTL)
}

// checkRatelimitSubnetMaskLen returns an error if the length of the subnet mask
// for IPv4 or IPv6 addresses is invalid.
func (req *jsonDNSConfig) checkRatelimitSubnetMaskLen() (err error) {
	err = checkInclusion(req.RatelimitSubnetLenIPv4, 0, netutil.IPv4BitLen)
	if err != nil {
		return fmt.Errorf("ratelimit_subnet_len_ipv4 is invalid: %w", err)
	}

	err = checkInclusion(req.RatelimitSubnetLenIPv6, 0, netutil.IPv6BitLen)
	if err != nil {
		return fmt.Errorf("ratelimit_subnet_len_ipv6 is invalid: %w", err)
	}

	return nil
}

// checkInclusion returns an error if a ptr is not nil and points to value,
// that not in the inclusive range between minN and maxN.
func checkInclusion(ptr *int, minN, maxN int) (err error) {
	if ptr == nil {
		return nil
	}

	n := *ptr
	switch {
	case n < minN:
		return fmt.Errorf("value %d less than min %d", n, minN)
	case n > maxN:
		return fmt.Errorf("value %d greater than max %d", n, maxN)
	}

	return nil
}

// handleSetConfig handles requests to the POST /control/dns_config endpoint.
func (s *Server) handleSetConfig(w http.ResponseWriter, r *http.Request) {
	req := &jsonDNSConfig{}
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "decoding request: %s", err)

		return
	}

	// TODO(e.burkov):  Consider prebuilding this set on startup.
	ourAddrs, err := s.conf.ourAddrsSet()
	if err != nil {
		// TODO(e.burkov):  Put into openapi.
		aghhttp.Error(r, w, http.StatusInternalServerError, "getting our addresses: %s", err)

		return
	}

	err = req.validate(ourAddrs, s.sysResolvers, s.privateNets)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "%s", err)

		return
	}

	restart := s.setConfig(req)
	s.conf.ConfigModified()

	if restart {
		err = s.Reconfigure(nil)
		if err != nil {
			aghhttp.Error(r, w, http.StatusInternalServerError, "%s", err)
		}
	}
}

// setConfig sets the server parameters.  shouldRestart is true if the server
// should be restarted to apply changes.
func (s *Server) setConfig(dc *jsonDNSConfig) (shouldRestart bool) {
	s.serverLock.Lock()
	defer s.serverLock.Unlock()

	if dc.BlockingMode != nil {
		s.dnsFilter.SetBlockingMode(*dc.BlockingMode, dc.BlockingIPv4, dc.BlockingIPv6)
	}

	if dc.BlockedResponseTTL != nil {
		s.dnsFilter.SetBlockedResponseTTL(*dc.BlockedResponseTTL)
	}

	if dc.ProtectionEnabled != nil {
		s.dnsFilter.SetProtectionEnabled(*dc.ProtectionEnabled)
	}

	if dc.UpstreamMode != nil {
		s.conf.UpstreamMode = mustParseUpstreamMode(*dc.UpstreamMode)
	}

	if dc.EDNSCSUseCustom != nil && *dc.EDNSCSUseCustom {
		s.conf.EDNSClientSubnet.CustomIP = dc.EDNSCSCustomIP
	}

	setIfNotNil(&s.conf.EnableDNSSEC, dc.DNSSECEnabled)
	setIfNotNil(&s.conf.AAAADisabled, dc.DisableIPv6)

	return s.setConfigRestartable(dc)
}

// mustParseUpstreamMode returns an upstream mode parsed from jsonUpstreamMode.
// Panics in case of invalid value.
func mustParseUpstreamMode(mode jsonUpstreamMode) (um UpstreamMode) {
	switch mode {
	case jsonUpstreamModeEmpty, jsonUpstreamModeLoadBalance:
		return UpstreamModeLoadBalance
	case jsonUpstreamModeParallel:
		return UpstreamModeParallel
	case jsonUpstreamModeFastestAddr:
		return UpstreamModeFastestAddr
	default:
		// Should never happen, since the value should be validated.
		panic(fmt.Errorf("unexpected upstream mode: %q", mode))
	}
}

// setIfNotNil sets the value pointed at by currentPtr to the value pointed at
// by newPtr if newPtr is not nil.  currentPtr must not be nil.
func setIfNotNil[T any](currentPtr, newPtr *T) (hasSet bool) {
	if newPtr == nil {
		return false
	}

	*currentPtr = *newPtr

	return true
}

// setConfigRestartable sets the parameters which trigger a restart.
// shouldRestart is true if the server should be restarted to apply changes.
// s.serverLock is expected to be locked.
//
// TODO(a.garipov): Some of these could probably be updated without a restart.
// Inspect and consider refactoring.
func (s *Server) setConfigRestartable(dc *jsonDNSConfig) (shouldRestart bool) {
	for _, hasSet := range []bool{
		setIfNotNil(&s.conf.UpstreamDNS, dc.Upstreams),
		setIfNotNil(&s.conf.LocalPTRResolvers, dc.LocalPTRUpstreams),
		setIfNotNil(&s.conf.UpstreamDNSFileName, dc.UpstreamsFile),
		setIfNotNil(&s.conf.BootstrapDNS, dc.Bootstraps),
		setIfNotNil(&s.conf.FallbackDNS, dc.Fallbacks),
		setIfNotNil(&s.conf.EDNSClientSubnet.Enabled, dc.EDNSCSEnabled),
		setIfNotNil(&s.conf.EDNSClientSubnet.UseCustom, dc.EDNSCSUseCustom),
		setIfNotNil(&s.conf.CacheSize, dc.CacheSize),
		setIfNotNil(&s.conf.CacheMinTTL, dc.CacheMinTTL),
		setIfNotNil(&s.conf.CacheMaxTTL, dc.CacheMaxTTL),
		setIfNotNil(&s.conf.CacheOptimistic, dc.CacheOptimistic),
		setIfNotNil(&s.conf.AddrProcConf.UseRDNS, dc.ResolveClients),
		setIfNotNil(&s.conf.UsePrivateRDNS, dc.UsePrivateRDNS),
		setIfNotNil(&s.conf.RatelimitSubnetLenIPv4, dc.RatelimitSubnetLenIPv4),
		setIfNotNil(&s.conf.RatelimitSubnetLenIPv6, dc.RatelimitSubnetLenIPv6),
		setIfNotNil(&s.conf.RatelimitWhitelist, dc.RatelimitWhitelist),
	} {
		shouldRestart = shouldRestart || hasSet
		if shouldRestart {
			break
		}
	}

	if dc.Ratelimit != nil && s.conf.Ratelimit != *dc.Ratelimit {
		s.conf.Ratelimit = *dc.Ratelimit
		shouldRestart = true
	}

	return shouldRestart
}

// upstreamJSON is a request body for handleTestUpstreamDNS endpoint.
type upstreamJSON struct {
	Upstreams        []string `json:"upstream_dns"`
	BootstrapDNS     []string `json:"bootstrap_dns"`
	FallbackDNS      []string `json:"fallback_dns"`
	PrivateUpstreams []string `json:"private_upstream"`
}

// closeBoots closes all the provided bootstrap servers and logs errors if any.
func closeBoots(boots []*upstream.UpstreamResolver) {
	for _, c := range boots {
		logCloserErr(c, "dnsforward: closing bootstrap %s: %s", c.Address())
	}
}

// handleTestUpstreamDNS handles requests to the POST /control/test_upstream_dns
// endpoint.
func (s *Server) handleTestUpstreamDNS(w http.ResponseWriter, r *http.Request) {
	req := &upstreamJSON{}
	err := json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "Failed to read request body: %s", err)

		return
	}

	req.BootstrapDNS = stringutil.FilterOut(req.BootstrapDNS, IsCommentOrEmpty)

	opts := &upstream.Options{
		Timeout:    s.conf.UpstreamTimeout,
		PreferIPv6: s.conf.BootstrapPreferIPv6,
	}

	var boots []*upstream.UpstreamResolver
	opts.Bootstrap, boots, err = newBootstrap(req.BootstrapDNS, s.etcHosts, opts)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "Failed to parse bootstrap servers: %s", err)

		return
	}
	defer closeBoots(boots)

	cv := newUpstreamConfigValidator(req.Upstreams, req.FallbackDNS, req.PrivateUpstreams, opts)
	cv.check()
	cv.close()

	aghhttp.WriteJSONResponseOK(w, r, cv.status())
}

// handleCacheClear is the handler for the POST /control/cache_clear HTTP API.
func (s *Server) handleCacheClear(w http.ResponseWriter, _ *http.Request) {
	s.dnsProxy.ClearCache()
	_, _ = io.WriteString(w, "OK")
}

// protectionJSON is an object for /control/protection endpoint.
type protectionJSON struct {
	Enabled  bool `json:"enabled"`
	Duration uint `json:"duration"`
}

// handleSetProtection is a handler for the POST /control/protection HTTP API.
func (s *Server) handleSetProtection(w http.ResponseWriter, r *http.Request) {
	protectionReq := &protectionJSON{}
	err := json.NewDecoder(r.Body).Decode(protectionReq)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "reading req: %s", err)

		return
	}

	var disabledUntil *time.Time
	if protectionReq.Duration > 0 {
		if protectionReq.Enabled {
			aghhttp.Error(
				r,
				w,
				http.StatusBadRequest,
				"Setting a duration is only allowed with protection disabling",
			)

			return
		}

		calcTime := time.Now().Add(time.Duration(protectionReq.Duration) * time.Millisecond)
		disabledUntil = &calcTime
	}

	func() {
		s.serverLock.Lock()
		defer s.serverLock.Unlock()

		s.dnsFilter.SetProtectionStatus(protectionReq.Enabled, disabledUntil)
	}()

	s.conf.ConfigModified()

	aghhttp.OK(w)
}

// handleDoH is the DNS-over-HTTPs handler.
//
// Control flow:
//
//	HTTP server
//	-> dnsforward.handleDoH
//	-> dnsforward.ServeHTTP
//	-> proxy.ServeHTTP
//	-> proxy.handleDNSRequest
//	-> dnsforward.handleDNSRequest
func (s *Server) handleDoH(w http.ResponseWriter, r *http.Request) {
	if !s.conf.TLSAllowUnencryptedDoH && r.TLS == nil {
		aghhttp.Error(r, w, http.StatusNotFound, "Not Found")

		return
	}

	if !s.IsRunning() {
		aghhttp.Error(r, w, http.StatusInternalServerError, "dns server is not running")

		return
	}

	s.ServeHTTP(w, r)
}

func (s *Server) registerHandlers() {
	if webRegistered || s.conf.HTTPRegister == nil {
		return
	}

	s.conf.HTTPRegister(http.MethodGet, "/control/dns_info", s.handleGetConfig)
	s.conf.HTTPRegister(http.MethodPost, "/control/dns_config", s.handleSetConfig)
	s.conf.HTTPRegister(http.MethodPost, "/control/test_upstream_dns", s.handleTestUpstreamDNS)
	s.conf.HTTPRegister(http.MethodPost, "/control/protection", s.handleSetProtection)

	s.conf.HTTPRegister(http.MethodGet, "/control/access/list", s.handleAccessList)
	s.conf.HTTPRegister(http.MethodPost, "/control/access/set", s.handleAccessSet)

	s.conf.HTTPRegister(http.MethodPost, "/control/cache_clear", s.handleCacheClear)

	// Register both versions, with and without the trailing slash, to
	// prevent a 301 Moved Permanently redirect when clients request the
	// path without the trailing slash.  Those redirects break some clients.
	//
	// See go doc net/http.ServeMux.
	//
	// See also https://github.com/AdguardTeam/AdGuardHome/issues/2628.
	s.conf.HTTPRegister("", "/dns-query", s.handleDoH)
	s.conf.HTTPRegister("", "/dns-query/", s.handleDoH)

	webRegistered = true
}
