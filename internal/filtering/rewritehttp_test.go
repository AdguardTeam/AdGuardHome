package filtering_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/agh"
	"github.com/AdguardTeam/AdGuardHome/internal/aghalg"
	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO(d.kolyshev): Use [rewrite.Item] instead.
type rewriteJSON struct {
	Domain  string          `json:"domain"`
	Answer  string          `json:"answer"`
	Enabled aghalg.NullBool `json:"enabled"`
}

// newRewriteJSON returns a freshly initialized *rewriteJSON.
func newRewriteJSON(domain, answer string, enabled aghalg.NullBool) (rw *rewriteJSON) {
	return &rewriteJSON{
		Domain:  domain,
		Answer:  answer,
		Enabled: enabled,
	}
}

type rewriteUpdateJSON struct {
	Target rewriteJSON `json:"target"`
	Update rewriteJSON `json:"update"`
}

// URLs for the rewrite HTTP API.
const (
	listURL   = "/control/rewrite/list"
	addURL    = "/control/rewrite/add"
	deleteURL = "/control/rewrite/delete"
	updateURL = "/control/rewrite/update"
)

// Messages for the rewrite HTTP API.
const (
	decodeMsg            = "json.Decode: json: cannot unmarshal string into Go value of type"
	decodeErrorMsg       = decodeMsg + " filtering.rewriteEntryJSON\n"
	decodeUpdateErrorMsg = decodeMsg + " filtering.rewriteUpdateJSON\n"
)

// Test domains.
const (
	exampleDomain  = "example.local"
	exampleAnswer  = "example.rewrite"
	oneDomain      = "one.local"
	oneAnswer      = "one.rewrite"
	disabledDomain = "disabled.local"
	disabledAnswer = "disabled.rewrite"
	addDomain      = "add.local"
	addAnswer      = "add.rewrite"
	updDomain      = "upd.local"
	updAnswer      = "upd.rewrite"
	invDomain      = "inv.local"
	invAnswer      = "inv.rewrite"
	invalidDomain  = "invalid_domain"
)

// testRewrites is a common rewrites list for tests.
var testRewrites = []*rewriteJSON{
	newRewriteJSON(exampleDomain, exampleAnswer, aghalg.NBTrue),
	newRewriteJSON(oneDomain, oneAnswer, aghalg.NBTrue),
	newRewriteJSON(disabledDomain, disabledAnswer, aghalg.NBFalse),
}

func TestDNSFilter_HandleRewriteHTTP(t *testing.T) {
	t.Parallel()

	rewritesJSON, mErr := json.Marshal(testRewrites)
	require.NoError(t, mErr)

	testCases := []struct {
		reqData     any
		name        string
		url         string
		method      string
		wantList    []*rewriteJSON
		wantBody    []byte
		wantConfMod bool
		wantStatus  int
	}{{
		name:        "list",
		url:         listURL,
		method:      http.MethodGet,
		reqData:     nil,
		wantConfMod: false,
		wantStatus:  http.StatusOK,
		wantBody:    append(rewritesJSON, '\n'),
		wantList:    testRewrites,
	}, {
		name:        "add_enabled_null",
		url:         addURL,
		method:      http.MethodPost,
		reqData:     rewriteJSON{Domain: addDomain, Answer: addAnswer},
		wantConfMod: true,
		wantStatus:  http.StatusOK,
		wantBody:    []byte{},
		wantList: append(
			testRewrites,
			newRewriteJSON(addDomain, addAnswer, aghalg.NBTrue),
		),
	}, {
		name:   "add_enabled_false",
		url:    addURL,
		method: http.MethodPost,
		reqData: rewriteJSON{
			Domain:  addDomain,
			Answer:  addAnswer,
			Enabled: aghalg.NBFalse,
		},
		wantConfMod: true,
		wantStatus:  http.StatusOK,
		wantBody:    []byte{},
		wantList: append(
			testRewrites,
			newRewriteJSON(addDomain, addAnswer, aghalg.NBFalse),
		),
	}, {
		name:   "add_enabled_true",
		url:    addURL,
		method: http.MethodPost,
		reqData: rewriteJSON{
			Domain:  addDomain,
			Answer:  addAnswer,
			Enabled: aghalg.NBTrue,
		},
		wantConfMod: true,
		wantStatus:  http.StatusOK,
		wantBody:    []byte{},
		wantList: append(
			testRewrites,
			newRewriteJSON(addDomain, addAnswer, aghalg.NBTrue),
		),
	}, {
		name:        "delete",
		url:         deleteURL,
		method:      http.MethodPost,
		reqData:     rewriteJSON{Domain: oneDomain, Answer: oneAnswer},
		wantConfMod: true,
		wantStatus:  http.StatusOK,
		wantBody:    []byte{},
		wantList: []*rewriteJSON{
			newRewriteJSON(exampleDomain, exampleAnswer, aghalg.NBTrue),
			newRewriteJSON(disabledDomain, disabledAnswer, aghalg.NBFalse),
		},
	}, {
		name:   "update_enabled_null",
		url:    updateURL,
		method: http.MethodPut,
		reqData: rewriteUpdateJSON{
			Target: rewriteJSON{Domain: oneDomain, Answer: oneAnswer},
			Update: rewriteJSON{Domain: updDomain, Answer: updAnswer},
		},
		wantConfMod: true,
		wantStatus:  http.StatusOK,
		wantBody:    []byte{},
		wantList: []*rewriteJSON{
			newRewriteJSON(exampleDomain, exampleAnswer, aghalg.NBTrue),
			newRewriteJSON(updDomain, updAnswer, aghalg.NBTrue),
			newRewriteJSON(disabledDomain, disabledAnswer, aghalg.NBFalse),
		},
	}, {
		name:   "update_enabled_false",
		url:    updateURL,
		method: http.MethodPut,
		reqData: rewriteUpdateJSON{
			Target: rewriteJSON{Domain: oneDomain, Answer: oneAnswer},
			Update: rewriteJSON{
				Domain:  updDomain,
				Answer:  updAnswer,
				Enabled: aghalg.NBFalse,
			},
		},
		wantConfMod: true,
		wantStatus:  http.StatusOK,
		wantBody:    []byte{},
		wantList: []*rewriteJSON{
			newRewriteJSON(exampleDomain, exampleAnswer, aghalg.NBTrue),
			newRewriteJSON(updDomain, updAnswer, aghalg.NBFalse),
			newRewriteJSON(disabledDomain, disabledAnswer, aghalg.NBFalse),
		},
	}, {
		name:   "update_enabled_true",
		url:    updateURL,
		method: http.MethodPut,
		reqData: rewriteUpdateJSON{
			Target: rewriteJSON{Domain: oneDomain, Answer: oneAnswer},
			Update: rewriteJSON{Domain: updDomain, Answer: updAnswer, Enabled: aghalg.NBTrue},
		},
		wantConfMod: true,
		wantStatus:  http.StatusOK,
		wantBody:    []byte{},
		wantList: []*rewriteJSON{
			newRewriteJSON(exampleDomain, exampleAnswer, aghalg.NBTrue),
			newRewriteJSON(updDomain, updAnswer, aghalg.NBTrue),
			newRewriteJSON(disabledDomain, disabledAnswer, aghalg.NBFalse),
		},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			confModCh := make(chan struct{}, 1)
			pt := testutil.NewPanicT(t)
			confModifier := &aghtest.ConfigModifier{}
			confModifier.OnApply = func(_ context.Context) {
				require.Truef(pt, tc.wantConfMod, "config modified has been fired")
				testutil.RequireSend(pt, confModCh, struct{}{}, testTimeout)
			}

			cl, baseURL := setupTestFiltering(t, confModifier)

			respBody, status := executeRequest(t, cl, baseURL, tc.method, tc.url, tc.reqData)

			if tc.wantConfMod {
				testutil.RequireReceive(t, confModCh, testTimeout)
			}

			assert.Equal(t, tc.wantStatus, status)
			assert.Equal(t, tc.wantBody, respBody)
			assertRewritesList(t, cl, baseURL, tc.wantList)
		})
	}
}

func TestDNSFilter_HandleRewriteHTTP_errors(t *testing.T) {
	t.Parallel()

	pt := testutil.NewPanicT(t)
	confModifier := &aghtest.ConfigModifier{}
	confModifier.OnApply = func(_ context.Context) {
		require.Truef(pt, false, "config modified has been fired")
	}

	testCases := []struct {
		reqData  any
		name     string
		url      string
		method   string
		wantBody []byte
	}{{
		name:     "add_error",
		url:      addURL,
		method:   http.MethodPost,
		reqData:  "invalid_json",
		wantBody: []byte(decodeErrorMsg),
	}, {
		name:   "add_error_invalid_cname",
		url:    addURL,
		method: http.MethodPost,
		reqData: rewriteJSON{
			Domain:  addDomain,
			Answer:  invalidDomain,
			Enabled: aghalg.NBTrue,
		},
		wantBody: []byte(`normalizing: invalid CNAME target "` + invalidDomain + `": bad domain name ` +
			`"` + invalidDomain + `": bad top-level domain name label "` + invalidDomain +
			`": bad top-level domain name label rune '_'` + "\n"),
	}, {
		name:     "delete_error",
		url:      deleteURL,
		method:   http.MethodPost,
		reqData:  "invalid_json",
		wantBody: []byte(decodeErrorMsg),
	}, {
		name:     "update_error",
		url:      updateURL,
		method:   http.MethodPut,
		reqData:  "invalid_json",
		wantBody: []byte(decodeUpdateErrorMsg),
	}, {
		name:   "update_error_target",
		url:    updateURL,
		method: http.MethodPut,
		reqData: rewriteUpdateJSON{
			Target: rewriteJSON{Domain: invDomain, Answer: invAnswer},
			Update: rewriteJSON{Domain: updDomain, Answer: updAnswer},
		},
		wantBody: []byte("target rule not found\n"),
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cl, base := setupTestFiltering(t, confModifier)
			respBody, status := executeRequest(t, cl, base, tc.method, tc.url, tc.reqData)

			assert.Equal(t, http.StatusBadRequest, status)
			assert.Equal(t, tc.wantBody, respBody)
			assertRewritesList(t, cl, base, testRewrites)
		})
	}
}

// assertRewritesList checks if rewrites list equals the list received from the
// handler by listURL.
func assertRewritesList(tb testing.TB, cl *http.Client, base *url.URL, wantList []*rewriteJSON) {
	tb.Helper()

	full := base.ResolveReference(&url.URL{Path: listURL})
	ctx := testutil.ContextWithTimeout(tb, testTimeout)
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, full.String(), nil)
	require.NoError(tb, err)

	resp, err := cl.Do(r)
	require.NoError(tb, err)
	defer func() {
		require.NoError(tb, resp.Body.Close())
	}()

	require.Equal(tb, http.StatusOK, resp.StatusCode)

	var actual []*rewriteJSON
	err = json.NewDecoder(resp.Body).Decode(&actual)
	require.NoError(tb, err)

	assert.Equal(tb, wantList, actual)
}

// setupTestFiltering creates the test environment with the filtering DNS filter
// and returns the HTTP client and server URL for making requests.
func setupTestFiltering(
	tb testing.TB,
	confModifier agh.ConfigModifier,
) (cl *http.Client, base *url.URL) {
	tb.Helper()

	mux := http.NewServeMux()

	d, err := filtering.New(&filtering.Config{
		Logger:       testLogger,
		ConfModifier: confModifier,
		HTTPReg: &aghtest.Registrar{
			OnRegister: func(_, urlPath string, handler http.HandlerFunc) {
				mux.Handle(urlPath, handler)
			},
		},
		Rewrites: rewriteEntriesToLegacyRewrites(testRewrites),
	}, nil)
	require.NoError(tb, err)
	tb.Cleanup(d.Close)

	d.RegisterFilteringHandlers()

	cl, base = aghtest.StartHTTPServer(tb, mux)

	return cl, base
}

// executeRequest prepares and executes an HTTP request against the given
// client and base URL, returning the response body and status code.
func executeRequest(
	tb testing.TB,
	client *http.Client,
	base *url.URL,
	method string,
	path string,
	reqData any,
) (respBody []byte, status int) {
	tb.Helper()

	var body io.Reader
	if reqData != nil {
		data, rErr := json.Marshal(reqData)
		require.NoError(tb, rErr)

		body = bytes.NewReader(data)
	}

	fullURL := base.ResolveReference(&url.URL{Path: path})
	ctx := testutil.ContextWithTimeout(tb, testTimeout)
	r, err := http.NewRequestWithContext(ctx, method, fullURL.String(), body)
	require.NoError(tb, err)

	resp, err := client.Do(r)
	require.NoError(tb, err)
	defer func() {
		require.NoError(tb, resp.Body.Close())
	}()

	respBody, err = io.ReadAll(resp.Body)
	require.NoError(tb, err)

	status = resp.StatusCode

	return respBody, status
}

// rewriteEntriesToLegacyRewrites gets legacy rewrites from json entries.
func rewriteEntriesToLegacyRewrites(entries []*rewriteJSON) (rw []*filtering.LegacyRewrite) {
	for _, entry := range entries {
		rw = append(rw, &filtering.LegacyRewrite{
			Domain:  entry.Domain,
			Answer:  entry.Answer,
			Enabled: entry.Enabled == aghalg.NBTrue,
		})
	}

	return rw
}

func TestDNSFilter_HandleRewriteSettings(t *testing.T) {
	const (
		enabled = "enabled"

		path       = "/control/rewrite/settings"
		pathUpdate = path + "/update"
	)

	var (
		wantEnabled  = fmt.Sprintf("{%q:%s}", enabled, "true")
		wantDisabled = fmt.Sprintf("{%q:%s}", enabled, "false")
	)

	confUpdated := false
	confModifier := &aghtest.ConfigModifier{
		OnApply: func(_ context.Context) {
			confUpdated = true
		},
	}
	handlers := make(map[string]http.Handler)

	d, err := filtering.New(&filtering.Config{
		Logger:       testLogger,
		ConfModifier: confModifier,
		HTTPReg: &aghtest.Registrar{
			OnRegister: func(_, url string, handler http.HandlerFunc) {
				handlers[url] = handler
			},
		},
		RewritesEnabled: false,
	}, nil)
	require.NoError(t, err)

	t.Cleanup(d.Close)

	require.True(t, t.Run("register", func(t *testing.T) {
		d.RegisterFilteringHandlers()
		require.NotEmpty(t, handlers)
		require.Contains(t, handlers, path)
		require.Contains(t, handlers, pathUpdate)

		r := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		handlers[path].ServeHTTP(w, r)
		require.Equal(t, http.StatusOK, w.Code)

		assert.JSONEq(t, wantDisabled, w.Body.String())
	}))

	require.True(t, t.Run("update", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPut, path, bytes.NewReader([]byte(wantEnabled)))
		w := httptest.NewRecorder()
		handlers[pathUpdate].ServeHTTP(w, r)
		require.Equal(t, http.StatusOK, w.Code)

		r = httptest.NewRequest(http.MethodGet, path, nil)
		w = httptest.NewRecorder()
		handlers[path].ServeHTTP(w, r)
		require.Equal(t, http.StatusOK, w.Code)

		assert.True(t, confUpdated)
		assert.JSONEq(t, wantEnabled, w.Body.String())
	}))
}
