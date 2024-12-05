package websvc_test

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"net/http"
	"net/netip"
	"net/url"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/next/agh"
	"github.com/AdguardTeam/AdGuardHome/internal/next/websvc"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/netutil/urlutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_HandlePatchSettingsHTTP(t *testing.T) {
	wantWeb := &websvc.HTTPAPIHTTPSettings{
		Addresses:       []netip.AddrPort{netip.MustParseAddrPort("127.0.1.1:80")},
		SecureAddresses: []netip.AddrPort{netip.MustParseAddrPort("127.0.1.1:443")},
		Timeout:         aghhttp.JSONDuration(10 * time.Second),
		ForceHTTPS:      false,
	}

	svc, err := websvc.New(&websvc.Config{
		Logger: slogutil.NewDiscardLogger(),
		Pprof: &websvc.PprofConfig{
			Enabled: false,
		},
		TLS: &tls.Config{
			Certificates: []tls.Certificate{{}},
		},
		Addresses:       []netip.AddrPort{netip.MustParseAddrPort("127.0.0.1:0")},
		SecureAddresses: []netip.AddrPort{netip.MustParseAddrPort("127.0.0.1:0")},
		Timeout:         5 * time.Second,
		ForceHTTPS:      true,
	})
	require.NoError(t, err)

	confMgr := newConfigManager()
	confMgr.onWeb = func() (s agh.ServiceWithConfig[*websvc.Config]) { return svc }
	confMgr.onUpdateWeb = func(ctx context.Context, c *websvc.Config) (err error) { return nil }

	_, addr := newTestServer(t, confMgr)
	u := &url.URL{
		Scheme: urlutil.SchemeHTTP,
		Host:   addr.String(),
		Path:   websvc.PathPatternV1SettingsHTTP,
	}

	req := jobj{
		"addresses":        wantWeb.Addresses,
		"secure_addresses": wantWeb.SecureAddresses,
		"timeout":          wantWeb.Timeout,
		"force_https":      wantWeb.ForceHTTPS,
	}

	respBody := httpPatch(t, u, req, http.StatusOK)
	resp := &websvc.HTTPAPIHTTPSettings{}
	err = json.Unmarshal(respBody, resp)
	require.NoError(t, err)

	assert.Equal(t, wantWeb, resp)
}
