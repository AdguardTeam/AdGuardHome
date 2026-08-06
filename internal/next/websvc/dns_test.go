package websvc_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/netip"
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/AdGuardHome/internal/next/agh"
	"github.com/AdguardTeam/AdGuardHome/internal/next/dnssvc"
	"github.com/AdguardTeam/AdGuardHome/internal/next/websvc"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/golibs/netutil/urlutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_HandlePatchSettingsDNS(t *testing.T) {
	wantDNS := &websvc.HTTPAPIDNSSettings{
		UpstreamMode:        proxy.UpstreamModeParallel,
		Addresses:           []netip.AddrPort{netip.MustParseAddrPort("127.0.1.1:53")},
		BootstrapServers:    []string{"1.0.0.1"},
		UpstreamServers:     []string{"1.1.1.1"},
		DNS64Prefixes:       []netip.Prefix{netip.MustParsePrefix("1234::/64")},
		UpstreamTimeout:     aghhttp.JSONDuration(2 * time.Second),
		Ratelimit:           100,
		CacheSize:           1048576,
		BootstrapPreferIPv6: true,
		RefuseAny:           true,
		UseDNS64:            true,
	}

	var started atomic.Bool
	confMgr := newConfigManager()
	confMgr.onDNS = func() (s agh.ServiceWithConfig[*dnssvc.Config]) {
		return &aghtest.ServiceWithConfig[*dnssvc.Config]{
			OnStart: func(_ context.Context) (err error) {
				started.Store(true)

				return nil
			},
			OnShutdown: func(ctx context.Context) (_ error) { panic(testutil.UnexpectedCall(ctx)) },
			OnConfig:   func() (c *dnssvc.Config) { return &dnssvc.Config{} },
		}
	}
	confMgr.onUpdateDNS = func(ctx context.Context, c *dnssvc.Config) (err error) {
		return nil
	}

	_, addr := newTestServer(t, confMgr)
	u := &url.URL{
		Scheme: urlutil.SchemeHTTP,
		Host:   addr.String(),
		Path:   websvc.PathPatternV1SettingsDNS,
	}

	req := jobj{
		"upstream_mode":         wantDNS.UpstreamMode,
		"addresses":             wantDNS.Addresses,
		"bootstrap_servers":     wantDNS.BootstrapServers,
		"upstream_servers":      wantDNS.UpstreamServers,
		"dns64_prefixes":        wantDNS.DNS64Prefixes,
		"upstream_timeout":      wantDNS.UpstreamTimeout,
		"cache_size":            wantDNS.CacheSize,
		"ratelimit":             wantDNS.Ratelimit,
		"bootstrap_prefer_ipv6": wantDNS.BootstrapPreferIPv6,
		"refuse_any":            wantDNS.RefuseAny,
		"use_dns64":             wantDNS.UseDNS64,
	}

	respBody := httpPatch(t, u, req, http.StatusOK)
	resp := &websvc.HTTPAPIDNSSettings{}
	err := json.Unmarshal(respBody, resp)
	require.NoError(t, err)

	assert.True(t, started.Load())
	assert.Equal(t, wantDNS, resp)
	assert.Equal(t, wantDNS, resp)
}
