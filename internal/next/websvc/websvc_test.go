package websvc_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"net/netip"
	"net/url"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/next/agh"
	"github.com/AdguardTeam/AdGuardHome/internal/next/dnssvc"
	"github.com/AdguardTeam/AdGuardHome/internal/next/websvc"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/netutil/httputil"
	"github.com/AdguardTeam/golibs/netutil/urlutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/AdguardTeam/golibs/testutil/fakeio/fakefs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testTimeout is the common timeout for tests.
const testTimeout = 1 * time.Second

// testStart is the server start value for tests.
var testStart = time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)

// type check
var _ websvc.ConfigManager = (*configManager)(nil)

// configManager is a [websvc.ConfigManager] for tests.
type configManager struct {
	onDNS func() (svc agh.ServiceWithConfig[*dnssvc.Config])
	onWeb func() (svc agh.ServiceWithConfig[*websvc.Config])

	onUpdateDNS func(ctx context.Context, c *dnssvc.Config) (err error)
	onUpdateWeb func(ctx context.Context, c *websvc.Config) (err error)
}

// DNS implements the [websvc.ConfigManager] interface for *configManager.
func (m *configManager) DNS() (svc agh.ServiceWithConfig[*dnssvc.Config]) {
	return m.onDNS()
}

// Web implements the [websvc.ConfigManager] interface for *configManager.
func (m *configManager) Web() (svc agh.ServiceWithConfig[*websvc.Config]) {
	return m.onWeb()
}

// UpdateDNS implements the [websvc.ConfigManager] interface for *configManager.
func (m *configManager) UpdateDNS(ctx context.Context, c *dnssvc.Config) (err error) {
	return m.onUpdateDNS(ctx, c)
}

// UpdateWeb implements the [websvc.ConfigManager] interface for *configManager.
func (m *configManager) UpdateWeb(ctx context.Context, c *websvc.Config) (err error) {
	return m.onUpdateWeb(ctx, c)
}

// newConfigManager returns a *configManager all methods of which panic.
func newConfigManager() (m *configManager) {
	return &configManager{
		onDNS: func() (_ agh.ServiceWithConfig[*dnssvc.Config]) {
			panic(testutil.UnexpectedCall())
		},
		onWeb: func() (_ agh.ServiceWithConfig[*websvc.Config]) {
			panic(testutil.UnexpectedCall())
		},
		onUpdateDNS: func(ctx context.Context, c *dnssvc.Config) (_ error) {
			panic(testutil.UnexpectedCall(ctx, c))
		},
		onUpdateWeb: func(ctx context.Context, c *websvc.Config) (_ error) {
			panic(testutil.UnexpectedCall(ctx, c))
		},
	}
}

// newTestServer creates and starts a new web service instance as well as its
// sole address.  It also registers a cleanup procedure, which shuts the
// instance down.
func newTestServer(
	tb testing.TB,
	confMgr websvc.ConfigManager,
) (svc *websvc.Service, addr netip.AddrPort) {
	tb.Helper()

	c := &websvc.Config{
		Logger: slogutil.NewDiscardLogger(),
		Pprof: &websvc.PprofConfig{
			Enabled: false,
		},
		ConfigManager: confMgr,
		Frontend: &fakefs.FS{
			OnOpen: func(_ string) (_ fs.File, _ error) { return nil, fs.ErrNotExist },
		},
		TLS:             nil,
		Addresses:       []netip.AddrPort{netip.MustParseAddrPort("127.0.0.1:0")},
		SecureAddresses: nil,
		Timeout:         testTimeout,
		Start:           testStart,
		ForceHTTPS:      false,
	}

	svc, err := websvc.New(c)
	require.NoError(tb, err)

	err = svc.Start(testutil.ContextWithTimeout(tb, testTimeout))
	require.NoError(tb, err)
	testutil.CleanupAndRequireSuccess(tb, func() (err error) {
		return svc.Shutdown(testutil.ContextWithTimeout(tb, testTimeout))
	})

	c = svc.Config()
	require.NotNil(tb, c)
	require.Len(tb, c.Addresses, 1)

	return svc, c.Addresses[0]
}

// jobj is a utility alias for JSON objects.
type jobj map[string]any

// httpGet is a helper that performs an HTTP GET request and returns the body of
// the response as well as checks that the status code is correct.
//
// TODO(a.garipov): Add helpers for other methods.
func httpGet(tb testing.TB, u *url.URL, wantCode int) (body []byte) {
	tb.Helper()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	require.NoErrorf(tb, err, "creating req")

	httpCli := &http.Client{
		Timeout: testTimeout,
	}
	resp, err := httpCli.Do(req)
	require.NoErrorf(tb, err, "performing req")
	require.Equal(tb, wantCode, resp.StatusCode)

	testutil.CleanupAndRequireSuccess(tb, resp.Body.Close)

	body, err = io.ReadAll(resp.Body)
	require.NoErrorf(tb, err, "reading body")

	return body
}

// httpPatch is a helper that performs an HTTP PATCH request with JSON-encoded
// reqBody as the request body and returns the body of the response as well as
// checks that the status code is correct.
//
// TODO(a.garipov): Add helpers for other methods.
func httpPatch(tb testing.TB, u *url.URL, reqBody any, wantCode int) (body []byte) {
	tb.Helper()

	b, err := json.Marshal(reqBody)
	require.NoErrorf(tb, err, "marshaling reqBody")

	req, err := http.NewRequest(http.MethodPatch, u.String(), bytes.NewReader(b))
	require.NoErrorf(tb, err, "creating req")

	httpCli := &http.Client{
		Timeout: testTimeout,
	}
	resp, err := httpCli.Do(req)
	require.NoErrorf(tb, err, "performing req")
	require.Equal(tb, wantCode, resp.StatusCode)

	testutil.CleanupAndRequireSuccess(tb, resp.Body.Close)

	body, err = io.ReadAll(resp.Body)
	require.NoErrorf(tb, err, "reading body")

	return body
}

func TestService_Start_getHealthCheck(t *testing.T) {
	confMgr := newConfigManager()
	_, addr := newTestServer(t, confMgr)
	u := &url.URL{
		Scheme: urlutil.SchemeHTTP,
		Host:   addr.String(),
		Path:   websvc.PathPatternHealthCheck,
	}

	body := httpGet(t, u, http.StatusOK)

	assert.Equal(t, []byte(httputil.HealthCheckHandler), body)
}
