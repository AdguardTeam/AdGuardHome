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

	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/AdGuardHome/internal/next/agh"
	"github.com/AdguardTeam/AdGuardHome/internal/next/dnssvc"
	"github.com/AdguardTeam/AdGuardHome/internal/next/websvc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_HandlePatchSettingsDNS(t *testing.T) {
	wantDNS := &websvc.HTTPAPIDNSSettings{
		Addresses:        []netip.AddrPort{netip.MustParseAddrPort("127.0.1.1:53")},
		BootstrapServers: []string{"1.0.0.1"},
		UpstreamServers:  []string{"1.1.1.1"},
		UpstreamTimeout:  websvc.JSONDuration(2 * time.Second),
	}

	var started atomic.Bool
	confMgr := newConfigManager()
	confMgr.onDNS = func() (s agh.ServiceWithConfig[*dnssvc.Config]) {
		return &aghtest.ServiceWithConfig[*dnssvc.Config]{
			OnStart: func() (err error) {
				started.Store(true)

				return nil
			},
			OnShutdown: func(_ context.Context) (err error) { panic("not implemented") },
			OnConfig:   func() (c *dnssvc.Config) { panic("not implemented") },
		}
	}
	confMgr.onUpdateDNS = func(ctx context.Context, c *dnssvc.Config) (err error) {
		return nil
	}

	_, addr := newTestServer(t, confMgr)
	u := &url.URL{
		Scheme: "http",
		Host:   addr.String(),
		Path:   websvc.PathV1SettingsDNS,
	}

	req := jobj{
		"addresses":         wantDNS.Addresses,
		"bootstrap_servers": wantDNS.BootstrapServers,
		"upstream_servers":  wantDNS.UpstreamServers,
		"upstream_timeout":  wantDNS.UpstreamTimeout,
	}

	respBody := httpPatch(t, u, req, http.StatusOK)
	resp := &websvc.HTTPAPIDNSSettings{}
	err := json.Unmarshal(respBody, resp)
	require.NoError(t, err)

	assert.True(t, started.Load())
	assert.Equal(t, wantDNS, resp)
	assert.Equal(t, wantDNS, resp)
}
