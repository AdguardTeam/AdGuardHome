package stats

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghalg"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Common domain values for tests.
const (
	TestDomain1 = "example.com"
	TestDomain2 = "example.org"
)

func TestHandleStatsConfig(t *testing.T) {
	const (
		smallIvl = 1 * time.Minute
		minIvl   = 1 * time.Hour
		maxIvl   = 365 * timeutil.Day
	)

	testCases := []struct {
		name     string
		wantErr  string
		body     getConfigResp
		wantCode int
	}{{
		name: "set_ivl_1_minIvl",
		body: getConfigResp{
			Enabled:        aghalg.NBTrue,
			Interval:       float64(minIvl.Milliseconds()),
			Ignored:        []string{},
			IgnoredEnabled: aghalg.NBFalse,
		},
		wantCode: http.StatusOK,
		wantErr:  "",
	}, {
		name: "small_interval",
		body: getConfigResp{
			Enabled:        aghalg.NBTrue,
			Interval:       float64(smallIvl.Milliseconds()),
			Ignored:        []string{},
			IgnoredEnabled: aghalg.NBFalse,
		},
		wantCode: http.StatusUnprocessableEntity,
		wantErr:  "unsupported interval: less than an hour\n",
	}, {
		name: "big_interval",
		body: getConfigResp{
			Enabled:        aghalg.NBTrue,
			Interval:       float64(maxIvl.Milliseconds() + minIvl.Milliseconds()),
			Ignored:        []string{},
			IgnoredEnabled: aghalg.NBFalse,
		},
		wantCode: http.StatusUnprocessableEntity,
		wantErr:  "unsupported interval: more than a year\n",
	}, {
		name: "set_ignored_ivl_1_maxIvl",
		body: getConfigResp{
			Enabled:  aghalg.NBTrue,
			Interval: float64(maxIvl.Milliseconds()),
			Ignored: []string{
				"ignor.ed",
			},
			IgnoredEnabled: aghalg.NBTrue,
		},
		wantCode: http.StatusOK,
		wantErr:  "",
	}, {
		name: "enabled_is_null",
		body: getConfigResp{
			Enabled:        aghalg.NBNull,
			Interval:       float64(minIvl.Milliseconds()),
			Ignored:        []string{},
			IgnoredEnabled: aghalg.NBFalse,
		},
		wantCode: http.StatusUnprocessableEntity,
		wantErr:  "enabled is null\n",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s := newTestStatsCtx(t, Config{Enabled: true})

			s.Start()
			testutil.CleanupAndRequireSuccess(t, s.Close)

			buf, err := json.Marshal(tc.body)
			require.NoError(t, err)

			const (
				configGet = "/control/stats/config"
				configPut = "/control/stats/config/update"
			)

			req := httptest.NewRequest(http.MethodPut, configPut, bytes.NewReader(buf))
			rw := httptest.NewRecorder()

			s.handlePutStatsConfig(rw, req)
			require.Equal(t, tc.wantCode, rw.Code)

			if tc.wantCode != http.StatusOK {
				assert.Equal(t, tc.wantErr, rw.Body.String())

				return
			}

			resp := httptest.NewRequest(http.MethodGet, configGet, nil)
			rw = httptest.NewRecorder()

			s.handleGetStatsConfig(rw, resp)
			require.Equal(t, http.StatusOK, rw.Code)

			ans := getConfigResp{}
			err = json.Unmarshal(rw.Body.Bytes(), &ans)
			require.NoError(t, err)

			assert.Equal(t, tc.body, ans)
		})
	}
}

// populateTestData is a helper that creates test entries in db.  s must not be
// nil.
func populateTestData(tb testing.TB, s *StatsCtx) {
	tb.Helper()

	oldUnitID := newUnitID() - 1
	oldUnit := &unitDB{
		NResult: make([]uint64, resultLast),
		Domains: []countPair{{Name: TestDomain1, Count: 1}},
		NTotal:  1,
	}

	db := s.db.Load()
	tx, err := db.Begin(true)
	require.NoError(tb, err)

	err = s.flushUnitToDB(oldUnit, tx, uint32(oldUnitID))
	require.NoError(tb, err)

	err = finishTxn(tx, true)
	require.NoError(tb, err)

	s.Update(&Entry{
		Client:         netutil.IPv4Localhost().String(),
		Domain:         TestDomain2,
		ProcessingTime: 3 * time.Minute,
		Result:         RNotFiltered,
	})
}

func TestStatsCtx_handleStats(t *testing.T) {
	testCases := []struct {
		name                  string
		wantErr               string
		wantTopQueriedDomains []topAddrs
		wantDNSQueries        uint64
		wantCode              int
		recent                int64
	}{{
		name:     "short_interval",
		wantErr:  "recent: out of range: must be no less than 3600000, got 240000\n",
		wantCode: http.StatusBadRequest,
		recent:   4 * time.Minute.Milliseconds(),
	}, {
		name:     "long_interval",
		wantErr:  "recent: out of range: must be no greater than 86400000, got 259200000\n",
		wantCode: http.StatusBadRequest,
		recent:   72 * time.Hour.Milliseconds(),
	}, {
		name:     "interval_is_not_multiple_of_hour",
		wantErr:  "recent: must be a multiple of 1 hour\n",
		wantCode: http.StatusBadRequest,
		recent:   time.Hour.Milliseconds() + 1,
	}, {
		name:           "no_interval",
		wantCode:       http.StatusOK,
		wantDNSQueries: 2,
		wantTopQueriedDomains: []topAddrs{{
			TestDomain1: 1,
		}, {
			TestDomain2: 1,
		}},
	}, {
		name:           "valid_interval",
		wantCode:       http.StatusOK,
		wantDNSQueries: 1,
		wantTopQueriedDomains: []topAddrs{{
			TestDomain2: 1,
		}},
		recent: time.Hour.Milliseconds(),
	}}

	s := newTestStatsCtx(t, Config{
		Enabled: true,
	})

	s.Start()
	defer testutil.CleanupAndRequireSuccess(t, s.Close)

	populateTestData(t, s)
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			url := "/control/stats"
			if tc.recent != 0 {
				url += fmt.Sprintf("?recent=%d", tc.recent)
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			rw := httptest.NewRecorder()

			s.handleStats(rw, req)
			require.Equal(t, tc.wantCode, rw.Code)

			if rw.Code != http.StatusOK {
				require.Equal(t, tc.wantErr, rw.Body.String())

				return
			}

			ans := StatsResp{}
			err := json.Unmarshal(rw.Body.Bytes(), &ans)
			require.NoError(t, err)

			assert.Equal(t, tc.wantDNSQueries, ans.NumDNSQueries)
			assert.ElementsMatch(t, tc.wantTopQueriedDomains, ans.TopQueried)
		})
	}
}
