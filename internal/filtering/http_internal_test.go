package filtering

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/AdGuardHome/internal/schedule"
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
			confModifier := &aghtest.ConfigModifier{}
			confModifier.OnApply = func(_ context.Context) {
				confModifiedCalled = true
			}
			d, err := New(&Config{
				Logger:           testLogger,
				FilteringEnabled: true,
				Filters:          tc.initial,
				HTTPClient: &http.Client{
					Timeout: 5 * time.Second,
				},
				ConfModifier: confModifier,
				HTTPReg:      aghhttp.EmptyRegistrar{},
				DataDir:      filtersDir,
				MaxHTTPSize:  testFilterSize,
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

func TestDNSFilter_filterSetPropertiesHTTPMetadata(t *testing.T) {
	const (
		oldURL = "https://example.org/old.txt"
		newURL = "https://example.org/new.txt"
	)

	testCases := []struct {
		name             string
		url              string
		wantMetadataKept bool
	}{
		{
			name:             "same_url",
			url:              oldURL,
			wantMetadataKept: true,
		}, {
			name: "changed_url",
			url:  newURL,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			d := newDNSFilter(t)
			d.conf.Filters = []FilterYAML{{
				Enabled:    true,
				URL:        oldURL,
				Name:       "test-filter",
				RulesCount: 1,
				Filter: Filter{
					ID: 1,
				},
			}}

			flt := &d.conf.Filters[0]
			require.NoError(t, os.WriteFile(
				flt.Path(d.conf.DataDir),
				[]byte("||example.org^\n"),
				0o600,
			))
			digest, err := d.currentFilterDigest(flt)
			require.NoError(t, err)
			require.NoError(t, d.storeHTTPMetadata(flt, filterHTTPMetadata{
				ETag:   `"v1"`,
				Digest: digest,
			}))
			metadataPath := flt.httpMetadataPath(d.conf.DataDir)
			require.FileExists(t, metadataPath)

			restart, err := d.filterSetProperties(oldURL, FilterYAML{
				Enabled: false,
				URL:     tc.url,
				Name:    "test-filter",
			}, false)
			require.NoError(t, err)
			assert.True(t, restart)
			assert.False(t, flt.Enabled)
			assert.Equal(t, tc.url, flt.URL)
			if tc.wantMetadataKept {
				require.FileExists(t, metadataPath)
			} else {
				require.NoFileExists(t, metadataPath)
			}
		})
	}
}

func TestDNSFilter_handleFilteringRemoveURLRemovesHTTPMetadata(t *testing.T) {
	const filterURL = "https://example.org/filter.txt"

	confModifier := &aghtest.ConfigModifier{
		OnApply: func(context.Context) {},
	}
	d, err := New(&Config{
		Logger:       testLogger,
		ConfModifier: confModifier,
		HTTPReg:      aghhttp.EmptyRegistrar{},
		DataDir:      t.TempDir(),
		Filters: []FilterYAML{{
			Enabled: false,
			URL:     filterURL,
			Filter: Filter{
				ID: 1,
			},
		}},
	}, nil)
	require.NoError(t, err)
	d.Start()
	t.Cleanup(d.Close)

	flt := &d.conf.Filters[0]
	contentPath := flt.Path(d.conf.DataDir)
	metadataPath := flt.httpMetadataPath(d.conf.DataDir)
	require.NoError(t, os.WriteFile(contentPath, []byte("||example.org^\n"), 0o600))
	digest, err := d.currentFilterDigest(flt)
	require.NoError(t, err)
	require.NoError(t, d.storeHTTPMetadata(flt, filterHTTPMetadata{
		ETag:   `"v1"`,
		Digest: digest,
	}))
	require.FileExists(t, metadataPath)

	body, err := json.Marshal(&filterURLReq{
		URL: filterURL,
	})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "http://example.org", bytes.NewReader(body))
	resp := httptest.NewRecorder()

	d.handleFilteringRemoveURL(resp, req)

	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Equal(t, "OK 0 rules\n", resp.Body.String())
	require.NoFileExists(t, metadataPath)
	require.FileExists(t, contentPath+".old")
}

func TestDNSFilter_handleFilteringAddURLEmptyHTTPMetadata(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("ETag", `"v1"`)
		_, err := w.Write([]byte("! Empty filter\n"))
		require.NoError(testutil.NewPanicT(t), err)
	}))
	t.Cleanup(srv.Close)

	dataDir := t.TempDir()
	d, err := New(&Config{
		Logger:      testLogger,
		DataDir:     dataDir,
		HTTPClient:  srv.Client(),
		MaxHTTPSize: testFilterSize,
	}, nil)
	require.NoError(t, err)
	t.Cleanup(d.Close)

	body, err := json.Marshal(&filterAddJSON{
		Name: "empty-filter",
		URL:  srv.URL,
	})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "http://example.org", bytes.NewReader(body))
	resp := httptest.NewRecorder()

	d.handleFilteringAddURL(resp, req)

	assert.Equal(t, http.StatusBadRequest, resp.Code)
	assert.Empty(t, d.conf.Filters)

	files, err := filepath.Glob(filepath.Join(dataDir, filterDir, "*"))
	require.NoError(t, err)
	assert.Empty(t, files)
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
			pt := testutil.NewPanicT(t)
			handlers := make(map[string]http.Handler)
			confModifier := &aghtest.ConfigModifier{}
			confModifier.OnApply = func(_ context.Context) {
				testutil.RequireSend(pt, confModCh, struct{}{}, testTimeout)
			}

			d, err := New(&Config{
				Logger:       testLogger,
				ConfModifier: confModifier,
				DataDir:      filtersDir,
				HTTPReg: &aghtest.Registrar{
					OnRegister: func(_, url string, handler http.HandlerFunc) {
						handlers[url] = handler
					},
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
			confModifier := &aghtest.ConfigModifier{}
			pt := testutil.NewPanicT(t)
			confModifier.OnApply = func(_ context.Context) {
				testutil.RequireSend(pt, confModCh, struct{}{}, testTimeout)
			}

			d, err := New(&Config{
				Logger:       testLogger,
				ConfModifier: confModifier,
				DataDir:      filtersDir,
				HTTPReg: &aghtest.Registrar{
					OnRegister: func(_, url string, handler http.HandlerFunc) {
						handlers[url] = handler
					},
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

func TestDNSFilter_HandleCheckHost(t *testing.T) {
	const (
		cliName = "client_name"
		cliID   = "client_id"

		notFilteredHost = "not.filterd.example"
		allowedHost     = "allowed.example"
		blockedHost     = "blocked.example"
		cliHost         = "client.example"
		qTypeHost       = "qtype.example"
		cliQTypeHost    = "cli.qtype.example"

		target          = "/control/check_host"
		hostFmt         = target + "?name=%s"
		hostCliFmt      = hostFmt + "&client=%s"
		hostQTypeFmt    = hostFmt + "&qtype=%s"
		hostCliQTypeFmt = hostCliFmt + "&qtype=%s"

		allowedRuleFmt         = "@@||%s^"
		blockedRuleFmt         = "||%s^"
		blockedRuleCliFmt      = blockedRuleFmt + "$client=%s"
		blockedRuleQTypeFmt    = blockedRuleFmt + "$dnstype=%s"
		blockedRuleCliQTypeFmt = blockedRuleCliFmt + ",dnstype=%s"
	)

	var (
		allowedRule            = fmt.Sprintf(allowedRuleFmt, allowedHost)
		blockedRule            = fmt.Sprintf(blockedRuleFmt, blockedHost)
		blockedClientRule      = fmt.Sprintf(blockedRuleCliFmt, cliHost, cliName)
		blockedQTypeRule       = fmt.Sprintf(blockedRuleQTypeFmt, qTypeHost, "CNAME")
		blockedClientQTypeRule = fmt.Sprintf(blockedRuleCliQTypeFmt, cliQTypeHost, cliName, "CNAME")

		notFilteredURL        = fmt.Sprintf(hostFmt, notFilteredHost)
		allowedURL            = fmt.Sprintf(hostFmt, allowedHost)
		blockedURL            = fmt.Sprintf(hostFmt, blockedHost)
		blockedClientURL      = fmt.Sprintf(hostCliFmt, cliHost, cliID)
		allowedQTypeURL       = fmt.Sprintf(hostQTypeFmt, qTypeHost, "AAAA")
		blockedQTypeURL       = fmt.Sprintf(hostQTypeFmt, qTypeHost, "CNAME")
		allowedClientQTypeURL = fmt.Sprintf(hostCliQTypeFmt, cliQTypeHost, cliID, "AAAA")
		blockedClientQTypeURL = fmt.Sprintf(hostCliQTypeFmt, cliQTypeHost, cliID, "CNAME")
	)

	rules := []string{
		allowedRule,
		blockedRule,
		blockedClientRule,
		blockedQTypeRule,
		blockedClientQTypeRule,
	}
	rulesData := strings.Join(rules, "\n")

	filters := []Filter{{
		ID: 0, Data: []byte(rulesData),
	}}

	clientNames := map[string]string{
		cliID: cliName,
	}

	dnsFilter, err := New(&Config{
		Logger: testLogger,
		BlockedServices: &BlockedServices{
			Schedule: schedule.EmptyWeekly(),
		},
		ApplyClientFiltering: func(clientID string, cliAddr netip.Addr, setts *Settings) {
			setts.ClientName = clientNames[clientID]
		},
	}, filters)
	require.NoError(t, err)

	testCases := []struct {
		name string
		url  string
		want *checkHostResp
	}{{
		name: "not_filtered",
		url:  notFilteredURL,
		want: &checkHostResp{
			Reason: reasonNames[NotFilteredNotFound],
			Rule:   "",
			Rules:  []*checkHostRespRule{},
		},
	}, {
		name: "allowed",
		url:  allowedURL,
		want: &checkHostResp{
			Reason: reasonNames[NotFilteredAllowList],
			Rule:   allowedRule,
			Rules: []*checkHostRespRule{{
				Text: allowedRule,
			}},
		},
	}, {
		name: "blocked",
		url:  blockedURL,
		want: &checkHostResp{
			Reason: reasonNames[FilteredBlockList],
			Rule:   blockedRule,
			Rules: []*checkHostRespRule{{
				Text: blockedRule,
			}},
		},
	}, {
		name: "blocked_client",
		url:  blockedClientURL,
		want: &checkHostResp{
			Reason: reasonNames[FilteredBlockList],
			Rule:   blockedClientRule,
			Rules: []*checkHostRespRule{{
				Text: blockedClientRule,
			}},
		},
	}, {
		name: "allowed_qtype",
		url:  allowedQTypeURL,
		want: &checkHostResp{
			Reason: reasonNames[NotFilteredNotFound],
			Rule:   "",
			Rules:  []*checkHostRespRule{},
		},
	}, {
		name: "blocked_qtype",
		url:  blockedQTypeURL,
		want: &checkHostResp{
			Reason: reasonNames[FilteredBlockList],
			Rule:   blockedQTypeRule,
			Rules: []*checkHostRespRule{{
				Text: blockedQTypeRule,
			}},
		},
	}, {
		name: "blocked_client_qtype",
		url:  blockedClientQTypeURL,
		want: &checkHostResp{
			Reason: reasonNames[FilteredBlockList],
			Rule:   blockedClientQTypeRule,
			Rules: []*checkHostRespRule{{
				Text: blockedClientQTypeRule,
			}},
		},
	}, {
		name: "allowed_client_qtype",
		url:  allowedClientQTypeURL,
		want: &checkHostResp{
			Reason: reasonNames[NotFilteredNotFound],
			Rule:   "",
			Rules:  []*checkHostRespRule{},
		},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, tc.url, nil)
			w := httptest.NewRecorder()

			dnsFilter.handleCheckHost(w, r)

			res := &checkHostResp{}
			err = json.NewDecoder(w.Body).Decode(res)
			require.NoError(t, err)

			assert.Equal(t, tc.want, res)
		})
	}
}
