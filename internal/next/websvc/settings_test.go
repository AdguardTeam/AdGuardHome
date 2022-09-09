package websvc_test

import (
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/netip"
	"net/url"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/next/dnssvc"
	"github.com/AdguardTeam/AdGuardHome/internal/next/websvc"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_HandleGetSettingsAll(t *testing.T) {
	// TODO(a.garipov): Add all currently supported parameters.

	wantDNS := &websvc.HTTPAPIDNSSettings{
		Addresses:        []netip.AddrPort{netip.MustParseAddrPort("127.0.0.1:53")},
		BootstrapServers: []string{"94.140.14.140", "94.140.14.141"},
		UpstreamServers:  []string{"94.140.14.14", "1.1.1.1"},
		UpstreamTimeout:  timeutil.Duration{Duration: 1 * time.Second},
	}

	wantWeb := &websvc.HTTPAPIHTTPSettings{
		Addresses:       []netip.AddrPort{netip.MustParseAddrPort("127.0.0.1:80")},
		SecureAddresses: []netip.AddrPort{netip.MustParseAddrPort("127.0.0.1:443")},
		Timeout:         timeutil.Duration{Duration: 5 * time.Second},
		ForceHTTPS:      true,
	}

	confMgr := newConfigManager()
	confMgr.onDNS = func() (c *dnssvc.Service) {
		c, err := dnssvc.New(&dnssvc.Config{
			Addresses:        wantDNS.Addresses,
			UpstreamServers:  wantDNS.UpstreamServers,
			BootstrapServers: wantDNS.BootstrapServers,
			UpstreamTimeout:  wantDNS.UpstreamTimeout.Duration,
		})
		require.NoError(t, err)

		return c
	}

	confMgr.onWeb = func() (c *websvc.Service) {
		return websvc.New(&websvc.Config{
			TLS: &tls.Config{
				Certificates: []tls.Certificate{{}},
			},
			Addresses:       wantWeb.Addresses,
			SecureAddresses: wantWeb.SecureAddresses,
			Timeout:         wantWeb.Timeout.Duration,
			ForceHTTPS:      true,
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

	assert.Equal(t, wantDNS, resp.DNS)
	assert.Equal(t, wantWeb, resp.HTTP)
}
