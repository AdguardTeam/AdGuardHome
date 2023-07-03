package filtering_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO(d.kolyshev): Use [rewrite.Item] instead.
type rewriteJSON struct {
	Domain string `json:"domain"`
	Answer string `json:"answer"`
}

type rewriteUpdateJSON struct {
	Target rewriteJSON `json:"target"`
	Update rewriteJSON `json:"update"`
}

const (
	// testTimeout is the common timeout for tests.
	testTimeout = 100 * time.Millisecond

	listURL   = "/control/rewrite/list"
	addURL    = "/control/rewrite/add"
	deleteURL = "/control/rewrite/delete"
	updateURL = "/control/rewrite/update"

	decodeErrorMsg = "json.Decode: json: cannot unmarshal string into Go value of type" +
		" filtering.rewriteEntryJSON\n"
)

func TestDNSFilter_handleRewriteHTTP(t *testing.T) {
	confModCh := make(chan struct{})
	reqCh := make(chan struct{})
	testRewrites := []*rewriteJSON{
		{Domain: "example.local", Answer: "example.rewrite"},
		{Domain: "one.local", Answer: "one.rewrite"},
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
		name:        "add",
		url:         addURL,
		method:      http.MethodPost,
		reqData:     rewriteJSON{Domain: "add.local", Answer: "add.rewrite"},
		wantConfMod: true,
		wantStatus:  http.StatusOK,
		wantBody:    "",
		wantList: append(
			testRewrites,
			&rewriteJSON{Domain: "add.local", Answer: "add.rewrite"},
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
		reqData:     rewriteJSON{Domain: "one.local", Answer: "one.rewrite"},
		wantConfMod: true,
		wantStatus:  http.StatusOK,
		wantBody:    "",
		wantList:    []*rewriteJSON{{Domain: "example.local", Answer: "example.rewrite"}},
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
		name:   "update",
		url:    updateURL,
		method: http.MethodPut,
		reqData: rewriteUpdateJSON{
			Target: rewriteJSON{Domain: "one.local", Answer: "one.rewrite"},
			Update: rewriteJSON{Domain: "upd.local", Answer: "upd.rewrite"},
		},
		wantConfMod: true,
		wantStatus:  http.StatusOK,
		wantBody:    "",
		wantList: []*rewriteJSON{
			{Domain: "example.local", Answer: "example.rewrite"},
			{Domain: "upd.local", Answer: "upd.rewrite"},
		},
	}, {
		name:        "update_error",
		url:         updateURL,
		method:      http.MethodPut,
		reqData:     "invalid_json",
		wantConfMod: false,
		wantStatus:  http.StatusBadRequest,
		wantBody: "json.Decode: json: cannot unmarshal string into Go value of type" +
			" filtering.rewriteUpdateJSON\n",
		wantList: testRewrites,
	}, {
		name:   "update_error_target",
		url:    updateURL,
		method: http.MethodPut,
		reqData: rewriteUpdateJSON{
			Target: rewriteJSON{Domain: "inv.local", Answer: "inv.rewrite"},
			Update: rewriteJSON{Domain: "upd.local", Answer: "upd.rewrite"},
		},
		wantConfMod: false,
		wantStatus:  http.StatusBadRequest,
		wantBody:    "target rule not found\n",
		wantList:    testRewrites,
	}}

	for _, tc := range testCases {
		onConfModified := func() {
			if !tc.wantConfMod {
				panic("config modified has been fired")
			}

			testutil.RequireSend(testutil.PanicT{}, confModCh, struct{}{}, testTimeout)
		}

		t.Run(tc.name, func(t *testing.T) {
			handlers := make(map[string]http.Handler)

			d, err := filtering.New(&filtering.Config{
				ConfigModified: onConfModified,
				HTTPRegister: func(_, url string, handler http.HandlerFunc) {
					handlers[url] = handler
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
func assertRewritesList(t *testing.T, handler http.Handler, wantList []*rewriteJSON) {
	t.Helper()

	r := httptest.NewRequest(http.MethodGet, listURL, nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)
	require.Equal(t, http.StatusOK, w.Code)

	var actual []*rewriteJSON
	err := json.NewDecoder(w.Body).Decode(&actual)
	require.NoError(t, err)

	assert.Equal(t, wantList, actual)
}

// rewriteEntriesToLegacyRewrites gets legacy rewrites from json entries.
func rewriteEntriesToLegacyRewrites(entries []*rewriteJSON) (rw []*filtering.LegacyRewrite) {
	for _, entry := range entries {
		rw = append(rw, &filtering.LegacyRewrite{
			Domain: entry.Domain,
			Answer: entry.Answer,
		})
	}

	return rw
}
