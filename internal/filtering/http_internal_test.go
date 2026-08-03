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

	"github.com/AdguardTeam/AdGuardHome/internal/agh"
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

func TestDNSFilter_handleFilteringSetURLWaitsForEngine(t *testing.T) {
	const filterURL = "https://filters.example.org/filter.txt"

	filtersDir := t.TempDir()
	confApplied := make(chan struct{})
	confModifier := &aghtest.ConfigModifier{
		OnApply: func(_ context.Context) {
			close(confApplied)
		},
	}

	d, err := New(&Config{
		Logger:           testLogger,
		FilteringEnabled: true,
		Filters: []FilterYAML{{
			Enabled: true,
			URL:     filterURL,
			Name:    "example",
		}},
		ConfModifier: confModifier,
		HTTPReg:      aghhttp.EmptyRegistrar{},
		DataDir:      filtersDir,
		MaxHTTPSize:  testFilterSize,
	}, nil)
	require.NoError(t, err)

	// Initialize the asynchronous queue without starting its consumer.  This
	// makes the previous fire-and-forget behavior return immediately while the
	// synchronous behavior below is blocked at the engine swap.
	d.filtersInitializerChan = make(chan filtersInitializerParams, 1)

	d.engineLock.RLock()
	engineLocked := true
	t.Cleanup(func() {
		if engineLocked {
			d.engineLock.RUnlock()
		}

		d.Close()
	})

	reqData := &filterURLReq{
		Data: &filterURLReqData{
			Name:    "example",
			URL:     filterURL,
			Enabled: false,
		},
		URL: filterURL,
	}
	data, err := json.Marshal(reqData)
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodPost, "http://example.org", bytes.NewReader(data))
	w := httptest.NewRecorder()
	handlerDone := make(chan struct{})
	go func() {
		d.handleFilteringSetURL(w, r)
		close(handlerDone)
	}()

	testutil.RequireReceive(t, confApplied, testTimeout)

	deadline := time.NewTimer(testTimeout)
	defer deadline.Stop()

	for d.engineLock.TryRLock() {
		d.engineLock.RUnlock()

		select {
		case <-handlerDone:
			require.FailNow(t, "handler returned before the filtering engine was rebuilt")
		case <-deadline.C:
			require.FailNow(t, "filtering engine rebuild did not reach the engine swap")
		default:
			time.Sleep(time.Millisecond)
		}
	}

	select {
	case <-handlerDone:
		require.FailNow(t, "handler returned while the filtering engine rebuild was blocked")
	default:
	}

	d.engineLock.RUnlock()
	engineLocked = false
	testutil.RequireReceive(t, handlerDone, testTimeout)
	assert.Empty(t, w.Body.String())
}

func TestDNSFilter_handleFilteringSetURLSupersedesQueuedEngine(t *testing.T) {
	const hostname = "example.org"

	filterURL := serveFiltersLocally(t, []byte("||"+hostname+"^"))
	d, err := New(&Config{
		Logger:           testLogger,
		FilteringEnabled: true,
		Filters: []FilterYAML{{
			Enabled: true,
			URL:     filterURL,
			Name:    "example",
			Filter: Filter{
				ID: 1,
			},
		}},
		ConfModifier: agh.EmptyConfigModifier{},
		HTTPReg:      aghhttp.EmptyRegistrar{},
		HTTPClient:   http.DefaultClient,
		DataDir:      t.TempDir(),
		MaxHTTPSize:  testFilterSize,
	}, nil)
	require.NoError(t, err)
	t.Cleanup(d.Close)

	updated, err := d.update(&d.conf.Filters[0])
	require.NoError(t, err)
	require.True(t, updated)

	setts := &Settings{
		ProtectionEnabled: true,
		FilteringEnabled:  true,
	}
	d.EnableFilters(false)
	d.checkMatch(t, hostname, setts)

	// Simulate an asynchronous initializer that has already received an older
	// enabled snapshot but has not rebuilt the engine yet.
	d.filtersInitializerChan = make(chan filtersInitializerParams, 1)
	d.EnableFilters(true)
	staleParams, ok := testutil.RequireReceive(t, d.filtersInitializerChan, testTimeout)
	require.True(t, ok)
	require.Len(t, staleParams.blockFilters, 2)

	// Also leave an older snapshot queued, to verify that the synchronous
	// update discards snapshots that the worker has not received yet.
	d.EnableFilters(true)

	reqData := &filterURLReq{
		Data: &filterURLReqData{
			Name:    "example",
			URL:     filterURL,
			Enabled: false,
		},
		URL: filterURL,
	}
	data, err := json.Marshal(reqData)
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodPost, "http://example.org", bytes.NewReader(data))
	w := httptest.NewRecorder()
	d.handleFilteringSetURL(w, r)
	require.Empty(t, w.Body.String())
	d.checkMatchEmpty(t, hostname, setts)

	select {
	case <-d.filtersInitializerChan:
		require.FailNow(t, "synchronous rebuild left an older snapshot queued")
	default:
	}

	// A worker that received the older snapshot before the synchronous update
	// must not restore that stale engine afterwards.
	require.NoError(t, d.initFilteringAsync(t.Context(), staleParams))
	d.checkMatchEmpty(t, hostname, setts)
}

func TestDNSFilter_handleFilteringSetURLReportsEngineError(t *testing.T) {
	const (
		oldHostname = "old.example"
		newHostname = "new.example"
	)

	disabledURL := serveFiltersLocally(t, []byte("||"+newHostname+"^"))
	d, err := New(&Config{
		Logger:           testLogger,
		FilteringEnabled: true,
		Filters: []FilterYAML{{
			Enabled: false,
			URL:     disabledURL,
			Name:    "disabled",
		}, {
			Enabled: true,
			URL:     "https://filters.example.org/enabled.txt",
			Name:    "enabled",
		}},
		ConfModifier: agh.EmptyConfigModifier{},
		HTTPReg:      aghhttp.EmptyRegistrar{},
		HTTPClient:   http.DefaultClient,
		DataDir:      t.TempDir(),
		MaxHTTPSize:  testFilterSize,
	}, nil)
	require.NoError(t, err)
	t.Cleanup(d.Close)

	// Install a known-good engine, then make the configured snapshot invalid so
	// that enabling the list through the handler fails during engine creation.
	require.NoError(t, d.setFilters(t.Context(), []Filter{{
		ID:   d.idGen.next(),
		Data: []byte("||" + oldHostname + "^"),
	}}, nil, false))
	d.conf.Filters[0].ID = d.conf.Filters[1].ID

	setts := &Settings{
		ProtectionEnabled: true,
		FilteringEnabled:  true,
	}
	d.checkMatch(t, oldHostname, setts)

	reqData := &filterURLReq{
		Data: &filterURLReqData{
			Name:    "disabled",
			URL:     disabledURL,
			Enabled: true,
		},
		URL: disabledURL,
	}
	data, err := json.Marshal(reqData)
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodPost, "http://example.org", bytes.NewReader(data))
	w := httptest.NewRecorder()
	d.handleFilteringSetURL(w, r)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "enabling filters")
	d.checkMatch(t, oldHostname, setts)
	d.checkMatchEmpty(t, newHostname, setts)
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
