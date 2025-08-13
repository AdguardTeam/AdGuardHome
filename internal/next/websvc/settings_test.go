package websvc_test

import (
	"encoding/json"
	"net/http"
	"net/netip"
	"net/url"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/next/agh"
	"github.com/AdguardTeam/AdGuardHome/internal/next/dnssvc"
	"github.com/AdguardTeam/AdGuardHome/internal/next/websvc"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/netutil/urlutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_HandleGetSettingsAll(t *testing.T) {
	// TODO(a.garipov): Add all currently supported parameters.

	wantDNS := &websvc.HTTPAPIDNSSettings{
		UpstreamMode:        proxy.UpstreamModeParallel,
		Addresses:           []netip.AddrPort{netip.MustParseAddrPort("127.0.0.1:53")},
		BootstrapServers:    []string{"94.140.14.140", "94.140.14.141"},
		UpstreamServers:     []string{"94.140.14.14", "1.1.1.1"},
		UpstreamTimeout:     aghhttp.JSONDuration(1 * time.Second),
		Ratelimit:           100,
		CacheSize:           1048576,
		BootstrapPreferIPv6: true,
		RefuseAny:           true,
		UseDNS64:            true,
	}

	confMgr := newConfigManager()
	confMgr.onDNS = func() (s agh.ServiceWithConfig[*dnssvc.Config]) {
		c, err := dnssvc.New(&dnssvc.Config{
			Logger:              slogutil.NewDiscardLogger(),
			UpstreamMode:        proxy.UpstreamModeParallel,
			Addresses:           wantDNS.Addresses,
			UpstreamServers:     wantDNS.UpstreamServers,
			BootstrapServers:    wantDNS.BootstrapServers,
			UpstreamTimeout:     time.Duration(wantDNS.UpstreamTimeout),
			CacheSize:           1048576,
			Ratelimit:           100,
			BootstrapPreferIPv6: true,
			RefuseAny:           true,
			UseDNS64:            true,
		})
		require.NoError(t, err)

		return c
	}

	svc, addr := newTestServer(t, confMgr)
	u := &url.URL{
		Scheme: urlutil.SchemeHTTP,
		Host:   addr.String(),
		Path:   websvc.PathPatternV1SettingsAll,
	}

	confMgr.onWeb = func() (s agh.ServiceWithConfig[*websvc.Config]) {
		return svc
	}

	wantWeb := &websvc.HTTPAPIHTTPSettings{
		Addresses:       []netip.AddrPort{addr},
		SecureAddresses: nil,
		Timeout:         aghhttp.JSONDuration(testTimeout),
		ForceHTTPS:      false,
	}

	body := httpGet(t, u, http.StatusOK)
	resp := &websvc.RespGetV1SettingsAll{}
	err := json.Unmarshal(body, resp)
	require.NoError(t, err)

	assert.Equal(t, wantDNS, resp.DNS)
	assert.Equal(t, wantWeb, resp.HTTP)
}
