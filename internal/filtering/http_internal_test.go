package filtering

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
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
			confModifier := &aghtest.ConfigModifier{}
			confModifier.OnApply = func(_ context.Context) {
				testutil.RequireSend(testutil.PanicT{}, confModCh, struct{}{}, testTimeout)
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
			confModifier.OnApply = func(_ context.Context) {
				testutil.RequireSend(testutil.PanicT{}, confModCh, struct{}{}, testTimeout)
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

func TestDNSFilter_validateFilterURLForFetch(t *testing.T) {
	dnsFilter, err := New(&Config{
		FilteringEnabled:           true,
		FiltersUpdateIntervalHours: 24,
	}, nil)
	require.NoError(t, err)
	t.Cleanup(dnsFilter.Close)

	testCases := []struct {
		name    string
		url     string
		wantErr string
	}{{
		name:    "valid_http",
		url:     "http://example.com/list",
		wantErr: "",
	}, {
		name:    "valid_https",
		url:     "https://example.com/blocklist",
		wantErr: "",
	}, {
		name:    "valid_https_no_extension",
		url:     "https://raw.githubusercontent.com/user/repo/main/hosts",
		wantErr: "",
	}, {
		name:    "valid_https_with_extension",
		url:     "https://example.com/filter.txt",
		wantErr: "",
	}, {
		name:    "invalid_scheme_ftp",
		url:     "ftp://example.com/list",
		wantErr: "invalid url scheme",
	}, {
		name:    "invalid_scheme_file",
		url:     "file:///path/to/file",
		wantErr: "invalid url scheme",
	}, {
		name:    "empty_host",
		url:     "https:///path",
		wantErr: "empty url host",
	}, {
		name:    "invalid_url",
		url:     "not a url",
		wantErr: "invalid URI for request",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := dnsFilter.validateFilterURLForFetch(tc.url)

			if tc.wantErr == "" {
				assert.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
			}
		})
	}
}

func TestDNSFilter_handleFetchTitle(t *testing.T) {
	const titleEndpoint = "/control/filtering/fetch_title"

	filtersDir := t.TempDir()

	// Set up test HTTP servers with different filter content.
	var (
		filterWithTitle    string
		filterWithoutTitle string
		filterHTML         string
	)

	filterWithTitle = serveFiltersLocally(t, []byte(`! Title: Test Filter
||example.org^
||example.com^`))

	filterWithoutTitle = serveFiltersLocally(t, []byte(`||example.org^
||example.com^`))

	filterHTML = serveFiltersLocally(t, []byte(`<html><body>Not a filter</body></html>`))

	handlers := map[string]http.HandlerFunc{}

	confModifier := &aghtest.ConfigModifier{}
	dnsFilter, err := New(&Config{
		Logger:                     testLogger,
		FilteringEnabled:           true,
		FiltersUpdateIntervalHours: 24,
		ConfModifier:               confModifier,
		DataDir:                    filtersDir,
		HTTPClient:                 http.DefaultClient,
		HTTPReg: &aghtest.Registrar{
			OnRegister: func(_, url string, handler http.HandlerFunc) {
				handlers[url] = handler
			},
		},
	}, nil)
	require.NoError(t, err)
	t.Cleanup(dnsFilter.Close)

	dnsFilter.RegisterFilteringHandlers()
	require.Contains(t, handlers, titleEndpoint)

	testCases := []struct {
		name      string
		url       string
		wantTitle string
		wantCode  int
	}{{
		name:      "with_title",
		url:       filterWithTitle,
		wantTitle: "Test Filter",
		wantCode:  http.StatusOK,
	}, {
		name:      "without_title",
		url:       filterWithoutTitle,
		wantTitle: "",
		wantCode:  http.StatusOK,
	}, {
		name:      "html_content",
		url:       filterHTML,
		wantTitle: "",
		wantCode:  http.StatusBadRequest,
	}, {
		name:      "invalid_url",
		url:       "not-a-url",
		wantTitle: "",
		wantCode:  http.StatusBadRequest,
	}, {
		name:      "invalid_scheme",
		url:       "ftp://example.com/list",
		wantTitle: "",
		wantCode:  http.StatusBadRequest,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqBody := filterTitleReq{URL: tc.url}
			body, err := json.Marshal(reqBody)
			require.NoError(t, err)

			r := httptest.NewRequest(http.MethodPost, titleEndpoint, bytes.NewReader(body))
			w := httptest.NewRecorder()

			handlers[titleEndpoint].ServeHTTP(w, r)
			assert.Equal(t, tc.wantCode, w.Code)

			if tc.wantCode == http.StatusOK {
				resp := &filterTitleResp{}
				err = json.NewDecoder(w.Body).Decode(resp)
				require.NoError(t, err)
				assert.Equal(t, tc.wantTitle, resp.Title)
			}
		})
	}
}
