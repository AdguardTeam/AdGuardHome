package filtering

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDNSFilter_handleFilteringSetURL(t *testing.T) {
	filtersDir := t.TempDir()

	var goodRulesEndpoint, anotherGoodRulesEndpoint, badRulesEndpoint string
	for _, rulesSource := range []struct {
		endpoint *string
		content  []byte
	}{{
		endpoint: &goodRulesEndpoint,
		content:  []byte(`||example.org^`),
	}, {
		endpoint: &anotherGoodRulesEndpoint,
		content:  []byte(`||example.com^`),
	}, {
		endpoint: &badRulesEndpoint,
		content:  []byte(`<html></html>`),
	}} {
		*rulesSource.endpoint = serveFiltersLocally(t, rulesSource.content)
	}

	testCases := []struct {
		name     string
		wantBody string
		oldURL   string
		newName  string
		newURL   string
		initial  []FilterYAML
	}{{
		name:     "success",
		wantBody: "",
		oldURL:   goodRulesEndpoint,
		newName:  "default_one",
		newURL:   anotherGoodRulesEndpoint,
		initial: []FilterYAML{{
			Enabled: true,
			URL:     goodRulesEndpoint,
			Name:    "default_one",
			white:   false,
		}},
	}, {
		name:     "non-existing",
		wantBody: "url doesn't exist\n",
		oldURL:   anotherGoodRulesEndpoint,
		newName:  "default_one",
		newURL:   goodRulesEndpoint,
		initial: []FilterYAML{{
			Enabled: true,
			URL:     goodRulesEndpoint,
			Name:    "default_one",
			white:   false,
		}},
	}, {
		name:     "existing",
		wantBody: "url already exists\n",
		oldURL:   goodRulesEndpoint,
		newName:  "default_one",
		newURL:   anotherGoodRulesEndpoint,
		initial: []FilterYAML{{
			Enabled: true,
			URL:     goodRulesEndpoint,
			Name:    "default_one",
			white:   false,
		}, {
			Enabled: true,
			URL:     anotherGoodRulesEndpoint,
			Name:    "another_default_one",
			white:   false,
		}},
	}, {
		name:     "bad_rules",
		wantBody: "data is HTML, not plain text\n",
		oldURL:   goodRulesEndpoint,
		newName:  "default_one",
		newURL:   badRulesEndpoint,
		initial: []FilterYAML{{
			Enabled: true,
			URL:     goodRulesEndpoint,
			Name:    "default_one",
			white:   false,
		}},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			confModifiedCalled := false
			d, err := New(&Config{
				FilteringEnabled: true,
				Filters:          tc.initial,
				HTTPClient: &http.Client{
					Timeout: 5 * time.Second,
				},
				ConfigModified: func() { confModifiedCalled = true },
				DataDir:        filtersDir,
			}, nil)
			require.NoError(t, err)
			t.Cleanup(d.Close)

			d.Start()

			reqData := &filterURLReq{
				Data: &filterURLReqData{
					// Leave the name of an existing list.
					Name:    tc.newName,
					URL:     tc.newURL,
					Enabled: true,
				},
				URL:       tc.oldURL,
				Whitelist: false,
			}
			data, err := json.Marshal(reqData)
			require.NoError(t, err)

			r := httptest.NewRequest(http.MethodPost, "http://example.org", bytes.NewReader(data))
			w := httptest.NewRecorder()

			d.handleFilteringSetURL(w, r)
			assert.Equal(t, tc.wantBody, w.Body.String())

			// For the moment the non-empty response body only contains occurred
			// error, so the configuration shouldn't be written.
			assert.Equal(t, tc.wantBody == "", confModifiedCalled)
		})
	}
}

func TestDNSFilter_handleSafeBrowsingStatus(t *testing.T) {
	const (
		testTimeout = time.Second
		statusURL   = "/control/safebrowsing/status"
	)

	confModCh := make(chan struct{})
	filtersDir := t.TempDir()

	testCases := []struct {
		name       string
		url        string
		enabled    bool
		wantStatus assert.BoolAssertionFunc
	}{{
		name:       "enable_off",
		url:        "/control/safebrowsing/enable",
		enabled:    false,
		wantStatus: assert.True,
	}, {
		name:       "enable_on",
		url:        "/control/safebrowsing/enable",
		enabled:    true,
		wantStatus: assert.True,
	}, {
		name:       "disable_on",
		url:        "/control/safebrowsing/disable",
		enabled:    true,
		wantStatus: assert.False,
	}, {
		name:       "disable_off",
		url:        "/control/safebrowsing/disable",
		enabled:    false,
		wantStatus: assert.False,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handlers := make(map[string]http.Handler)

			d, err := New(&Config{
				ConfigModified: func() {
					testutil.RequireSend(testutil.PanicT{}, confModCh, struct{}{}, testTimeout)
				},
				DataDir: filtersDir,
				HTTPRegister: func(_, url string, handler http.HandlerFunc) {
					handlers[url] = handler
				},
				SafeBrowsingEnabled: tc.enabled,
			}, nil)
			require.NoError(t, err)
			t.Cleanup(d.Close)

			d.RegisterFilteringHandlers()
			require.NotEmpty(t, handlers)
			require.Contains(t, handlers, statusURL)

			r := httptest.NewRequest(http.MethodPost, tc.url, nil)
			w := httptest.NewRecorder()

			go handlers[tc.url].ServeHTTP(w, r)

			testutil.RequireReceive(t, confModCh, testTimeout)

			r = httptest.NewRequest(http.MethodGet, statusURL, nil)
			w = httptest.NewRecorder()

			handlers[statusURL].ServeHTTP(w, r)
			require.Equal(t, http.StatusOK, w.Code)

			status := struct {
				Enabled bool `json:"enabled"`
			}{
				Enabled: false,
			}

			err = json.NewDecoder(w.Body).Decode(&status)
			require.NoError(t, err)

			tc.wantStatus(t, status.Enabled)
		})
	}
}

func TestDNSFilter_handleParentalStatus(t *testing.T) {
	const (
		testTimeout = time.Second
		statusURL   = "/control/parental/status"
	)

	confModCh := make(chan struct{})
	filtersDir := t.TempDir()

	testCases := []struct {
		name       string
		url        string
		enabled    bool
		wantStatus assert.BoolAssertionFunc
	}{{
		name:       "enable_off",
		url:        "/control/parental/enable",
		enabled:    false,
		wantStatus: assert.True,
	}, {
		name:       "enable_on",
		url:        "/control/parental/enable",
		enabled:    true,
		wantStatus: assert.True,
	}, {
		name:       "disable_on",
		url:        "/control/parental/disable",
		enabled:    true,
		wantStatus: assert.False,
	}, {
		name:       "disable_off",
		url:        "/control/parental/disable",
		enabled:    false,
		wantStatus: assert.False,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			handlers := make(map[string]http.Handler)

			d, err := New(&Config{
				ConfigModified: func() {
					testutil.RequireSend(testutil.PanicT{}, confModCh, struct{}{}, testTimeout)
				},
				DataDir: filtersDir,
				HTTPRegister: func(_, url string, handler http.HandlerFunc) {
					handlers[url] = handler
				},
				ParentalEnabled: tc.enabled,
			}, nil)
			require.NoError(t, err)
			t.Cleanup(d.Close)

			d.RegisterFilteringHandlers()
			require.NotEmpty(t, handlers)
			require.Contains(t, handlers, statusURL)

			r := httptest.NewRequest(http.MethodPost, tc.url, nil)
			w := httptest.NewRecorder()

			go handlers[tc.url].ServeHTTP(w, r)

			testutil.RequireReceive(t, confModCh, testTimeout)

			r = httptest.NewRequest(http.MethodGet, statusURL, nil)
			w = httptest.NewRecorder()

			handlers[statusURL].ServeHTTP(w, r)
			require.Equal(t, http.StatusOK, w.Code)

			status := struct {
				Enabled bool `json:"enabled"`
			}{
				Enabled: false,
			}

			err = json.NewDecoder(w.Body).Decode(&status)
			require.NoError(t, err)

			tc.wantStatus(t, status.Enabled)
		})
	}
}
