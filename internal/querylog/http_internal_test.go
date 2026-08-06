package querylog

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test domains for querylog search testing.
const (
	testDomainBlocked   = "blocked.org"
	testDomainNotFound  = "notfound.org"
	testDomainRewritten = "rewritten.org"
)

// response is the GET /control/querylog HTTP response structure.
type response struct {
	Data []data `json:"data"`
}

// data is a single querylog entry in the response.
type data struct {
	Question question `json:"question"`
}

// question is the DNS question part of a querylog entry.
type question struct {
	Name string `json:"name"`
}

// parseHostnamesFromEntry is a helper that parses the /control/querylog
// response and extracts the host names from it.
func parseHostnamesFromEntry(tb testing.TB, in io.Reader) (hostnames []string) {
	tb.Helper()

	var resp response
	err := json.NewDecoder(in).Decode(&resp)
	require.NoError(tb, err)

	for _, d := range resp.Data {
		hostnames = append(hostnames, d.Question.Name)
	}

	return hostnames
}

func TestQuerylog_HandleQueryLog_reasonSearchCriterion(t *testing.T) {
	testCases := []struct {
		name       string
		query      url.Values
		wantMsg    string
		wantHosts  []string
		wantStatus int
	}{{
		name:       "no_params",
		query:      url.Values{},
		wantMsg:    "",
		wantStatus: http.StatusOK,
		wantHosts:  []string{testDomainRewritten, testDomainBlocked, testDomainNotFound},
	}, {
		name:       "reason_not_found",
		query:      url.Values{"reason": []string{filtering.NotFilteredNotFound.String()}},
		wantMsg:    "",
		wantStatus: http.StatusOK,
		wantHosts:  []string{testDomainNotFound},
	}, {
		name:       "reason_block_list",
		query:      url.Values{"reason": []string{filtering.FilteredBlockList.String()}},
		wantMsg:    "",
		wantStatus: http.StatusOK,
		wantHosts:  []string{testDomainBlocked},
	}, {
		name:       "reason_rewritten",
		query:      url.Values{"reason": []string{filtering.Rewritten.String()}},
		wantMsg:    "",
		wantStatus: http.StatusOK,
		wantHosts:  []string{testDomainRewritten},
	}, {
		name: "multiple_reasons",
		query: url.Values{"reason": []string{
			filtering.Rewritten.String(),
			filtering.FilteredBlockList.String(),
		}},
		wantMsg:    "",
		wantStatus: http.StatusOK,
		wantHosts:  []string{testDomainRewritten, testDomainBlocked},
	}, {
		name:       "invalid_reason",
		query:      url.Values{"reason": []string{"InvalidReason"}},
		wantMsg:    `parsing params: reason: bad enum value: "InvalidReason"` + "\n",
		wantStatus: http.StatusBadRequest,
		wantHosts:  nil,
	}, {
		name: "reason_and_status_conflict",
		query: url.Values{
			"reason":          []string{filtering.Rewritten.String()},
			"response_status": []string{filteringStatusAll},
		},
		wantMsg: `parsing params: "reason" and "response_status"` +
			` criteria cannot be used together` + "\n",
		wantStatus: http.StatusBadRequest,
		wantHosts:  nil,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			l := newTestQueryLog(t)

			u := (&url.URL{
				Path:     "/control/querylog",
				RawQuery: tc.query.Encode(),
			}).String()

			require.True(t, t.Run("memory", func(t *testing.T) {
				testSearchAPI(t, l, tc.wantStatus, tc.wantHosts, tc.wantMsg, u)
			}))

			require.True(t, t.Run("file", func(t *testing.T) {
				ctx := testutil.ContextWithTimeout(t, testTimeout)
				err := l.flushLogBuffer(ctx)
				require.NoError(t, err)

				testSearchAPI(t, l, tc.wantStatus, tc.wantHosts, tc.wantMsg, u)
			}))
		})
	}
}

// newTestQueryLog is a helper that returns new *queryLog initialized with
// common test values.  It also adds several test entries.
func newTestQueryLog(tb testing.TB) (l *queryLog) {
	tb.Helper()

	l, err := newQueryLog(Config{
		Logger:      testLogger,
		Enabled:     true,
		FileEnabled: true,
		RotationIvl: timeutil.Day,
		MemSize:     100,
		BaseDir:     tb.TempDir(),
		Anonymizer:  aghnet.NewIPMut(nil),
	})
	require.NoError(tb, err)

	addTestEntry(
		l,
		testDomainNotFound,
		testAnswerIPv4,
		testClientIPv4,
		filtering.NotFilteredNotFound,
	)
	addTestEntry(
		l,
		testDomainBlocked,
		testAnswerIPv4,
		testClientIPv4,
		filtering.FilteredBlockList,
	)
	addTestEntry(
		l,
		testDomainRewritten,
		testAnswerIPv4,
		testClientIPv4,
		filtering.Rewritten,
	)

	return l
}

// testSearchAPI is a helper that makes sure that l handles GET
// /control/querylog HTTP API requests correctly.
func testSearchAPI(
	tb testing.TB,
	l *queryLog,
	wantStatus int,
	wantHosts []string,
	wantMsg string,
	u string,
) {
	tb.Helper()

	ctx := testutil.ContextWithTimeout(tb, testTimeout)
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	w := httptest.NewRecorder()

	l.handleQueryLog(w, req)

	assert.Equal(tb, wantStatus, w.Code)
	if wantStatus != http.StatusOK {
		msg, err := io.ReadAll(w.Body)
		require.NoError(tb, err)

		assert.Equal(tb, wantMsg, string(msg))

		return
	}

	gotHosts := parseHostnamesFromEntry(tb, w.Body)
	assert.Equal(tb, wantHosts, gotHosts)
}
