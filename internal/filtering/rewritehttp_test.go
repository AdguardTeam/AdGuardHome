package filtering_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

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

const (
	listURL   = "/control/rewrite/list"
	addURL    = "/control/rewrite/add"
	deleteURL = "/control/rewrite/delete"
	updateURL = "/control/rewrite/update"

	decodeMsg            = "json.Decode: json: cannot unmarshal string into Go value of type"
	decodeErrorMsg       = decodeMsg + " filtering.rewriteEntryJSON\n"
	decodeUpdateErrorMsg = decodeMsg + " filtering.rewriteUpdateJSON\n"
)

func TestDNSFilter_HandleRewriteHTTP(t *testing.T) {
	t.Parallel()

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
	)

	testRewrites := []*rewriteJSON{
		newRewriteJSON(exampleDomain, exampleAnswer, aghalg.NBTrue),
		newRewriteJSON(oneDomain, oneAnswer, aghalg.NBTrue),
		newRewriteJSON(disabledDomain, disabledAnswer, aghalg.NBFalse),
	}

	testRewritesJSON, mErr := json.Marshal(testRewrites)
	require.NoError(t, mErr)

	testCases := []struct {
		reqData     any
		name        string
		url         string
		method      string
		wantList    []*rewriteJSON
		wantBody    string
		wantConfMod bool
		wantStatus  int
	}{{
		name:        "list",
		url:         listURL,
		method:      http.MethodGet,
		reqData:     nil,
		wantConfMod: false,
		wantStatus:  http.StatusOK,
		wantBody:    string(testRewritesJSON) + "\n",
		wantList:    testRewrites,
	}, {
		name:        "add_enabled_null",
		url:         addURL,
		method:      http.MethodPost,
		reqData:     rewriteJSON{Domain: addDomain, Answer: addAnswer},
		wantConfMod: true,
		wantStatus:  http.StatusOK,
		wantBody:    "",
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
		wantBody:    "",
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
		wantBody:    "",
		wantList: append(
			testRewrites,
			newRewriteJSON(addDomain, addAnswer, aghalg.NBTrue),
		),
	}, {
		name:        "add_error",
		url:         addURL,
		method:      http.MethodPost,
		reqData:     "invalid_json",
		wantConfMod: false,
		wantStatus:  http.StatusBadRequest,
		wantBody:    decodeErrorMsg,
		wantList:    testRewrites,
	}, {
		name:        "delete",
		url:         deleteURL,
		method:      http.MethodPost,
		reqData:     rewriteJSON{Domain: oneDomain, Answer: oneAnswer},
		wantConfMod: true,
		wantStatus:  http.StatusOK,
		wantBody:    "",
		wantList: []*rewriteJSON{
			newRewriteJSON(exampleDomain, exampleAnswer, aghalg.NBTrue),
			newRewriteJSON(disabledDomain, disabledAnswer, aghalg.NBFalse),
		},
	}, {
		name:        "delete_error",
		url:         deleteURL,
		method:      http.MethodPost,
		reqData:     "invalid_json",
		wantConfMod: false,
		wantStatus:  http.StatusBadRequest,
		wantBody:    decodeErrorMsg,
		wantList:    testRewrites,
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
		wantBody:    "",
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
		wantBody:    "",
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
		wantBody:    "",
		wantList: []*rewriteJSON{
			newRewriteJSON(exampleDomain, exampleAnswer, aghalg.NBTrue),
			newRewriteJSON(updDomain, updAnswer, aghalg.NBTrue),
			newRewriteJSON(disabledDomain, disabledAnswer, aghalg.NBFalse),
		},
	}, {
		name:        "update_error",
		url:         updateURL,
		method:      http.MethodPut,
		reqData:     "invalid_json",
		wantConfMod: false,
		wantStatus:  http.StatusBadRequest,
		wantBody:    decodeUpdateErrorMsg,
		wantList:    testRewrites,
	}, {
		name:   "update_error_target",
		url:    updateURL,
		method: http.MethodPut,
		reqData: rewriteUpdateJSON{
			Target: rewriteJSON{Domain: invDomain, Answer: invAnswer},
			Update: rewriteJSON{Domain: updDomain, Answer: updAnswer},
		},
		wantConfMod: false,
		wantStatus:  http.StatusBadRequest,
		wantBody:    "target rule not found\n",
		wantList:    testRewrites,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			confModCh := make(chan struct{})
			reqCh := make(chan struct{})

			handlers := make(map[string]http.Handler)
			confModifier := &aghtest.ConfigModifier{}
			confModifier.OnApply = func(_ context.Context) {
				require.Truef(t, tc.wantConfMod, "config modified has been fired")
				testutil.RequireSend(testutil.PanicT{}, confModCh, struct{}{}, testTimeout)
			}

			d, err := filtering.New(&filtering.Config{
				Logger:       testLogger,
				ConfModifier: confModifier,
				HTTPReg: &aghtest.Registrar{
					OnRegister: func(_, url string, handler http.HandlerFunc) {
						handlers[url] = handler
					},
				},
				Rewrites: rewriteEntriesToLegacyRewrites(testRewrites),
			}, nil)
			require.NoError(t, err)
			t.Cleanup(d.Close)

			d.RegisterFilteringHandlers()
			require.NotEmpty(t, handlers)
			require.Contains(t, handlers, listURL)
			require.Contains(t, handlers, tc.url)

			var body io.Reader
			if tc.reqData != nil {
				data, rErr := json.Marshal(tc.reqData)
				require.NoError(t, rErr)

				body = bytes.NewReader(data)
			}

			r := httptest.NewRequest(tc.method, tc.url, body)
			w := httptest.NewRecorder()

			go func() {
				handlers[tc.url].ServeHTTP(w, r)

				testutil.RequireSend(testutil.PanicT{}, reqCh, struct{}{}, testTimeout)
			}()

			if tc.wantConfMod {
				testutil.RequireReceive(t, confModCh, testTimeout)
			}

			testutil.RequireReceive(t, reqCh, testTimeout)
			assert.Equal(t, tc.wantStatus, w.Code)

			respBody, err := io.ReadAll(w.Body)
			require.NoError(t, err)
			assert.Equal(t, []byte(tc.wantBody), respBody)

			assertRewritesList(t, handlers[listURL], tc.wantList)
		})
	}
}

// assertRewritesList checks if rewrites list equals the list received from the
// handler by listURL.
func assertRewritesList(tb testing.TB, handler http.Handler, wantList []*rewriteJSON) {
	tb.Helper()

	r := httptest.NewRequest(http.MethodGet, listURL, nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)
	require.Equal(tb, http.StatusOK, w.Code)

	var actual []*rewriteJSON
	err := json.NewDecoder(w.Body).Decode(&actual)
	require.NoError(tb, err)

	assert.Equal(tb, wantList, actual)
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
