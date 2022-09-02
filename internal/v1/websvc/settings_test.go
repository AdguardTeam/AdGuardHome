package websvc_test

import (
	"encoding/json"
	"net/http"
	"net/netip"
	"net/url"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/v1/dnssvc"
	"github.com/AdguardTeam/AdGuardHome/internal/v1/websvc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_HandleGetSettingsAll(t *testing.T) {
	// TODO(a.garipov): Add all currently supported parameters.

	dnsAddrs := []netip.AddrPort{netip.MustParseAddrPort("127.0.0.1:53")}
	upsSrvs := []string{"94.140.14.14", "1.1.1.1"}

	webAddrs := []netip.AddrPort{netip.MustParseAddrPort("127.0.0.1:80")}
	const webTimeout = 5 * time.Second

	confMgr := newConfigManager()
	confMgr.onDNS = func() (c *dnssvc.Service) {
		c, err := dnssvc.New(&dnssvc.Config{
			Addresses:       dnsAddrs,
			UpstreamServers: upsSrvs,
		})
		require.NoError(t, err)

		return c
	}

	confMgr.onWeb = func() (c *websvc.Service) {
		return websvc.New(&websvc.Config{
			Addresses: webAddrs,
			Timeout:   webTimeout,
		})
	}

	_, addr := newTestServer(t, confMgr)
	u := &url.URL{
		Scheme: "http",
		Host:   addr.String(),
		Path:   websvc.PathV1SettingsAll,
	}

	body := httpGet(t, u, http.StatusOK)
	resp := &websvc.RespGetV1SettingsAll{}
	err := json.Unmarshal(body, resp)
	require.NoError(t, err)

	assert.Equal(t, dnsAddrs, resp.DNS.Addresses)
	assert.Equal(t, upsSrvs, resp.DNS.UpstreamServers)

	assert.Equal(t, webAddrs, resp.HTTP.Addresses)
	assert.Equal(t, webTimeout, resp.HTTP.Timeout.Duration)
}
