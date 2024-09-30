package stats

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghalg"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleStatsConfig(t *testing.T) {
	const (
		smallIvl = 1 * time.Minute
		minIvl   = 1 * time.Hour
		maxIvl   = 365 * timeutil.Day
	)

	conf := Config{
		Logger:            slogutil.NewDiscardLogger(),
		UnitID:            func() (id uint32) { return 0 },
		ConfigModified:    func() {},
		ShouldCountClient: func([]string) bool { return true },
		Filename:          filepath.Join(t.TempDir(), "stats.db"),
		Limit:             time.Hour * 24,
		Enabled:           true,
	}

	testCases := []struct {
		name     string
		wantErr  string
		body     getConfigResp
		wantCode int
	}{{
		name: "set_ivl_1_minIvl",
		body: getConfigResp{
			Enabled:  aghalg.NBTrue,
			Interval: float64(minIvl.Milliseconds()),
			Ignored:  []string{},
		},
		wantCode: http.StatusOK,
		wantErr:  "",
	}, {
		name: "small_interval",
		body: getConfigResp{
			Enabled:  aghalg.NBTrue,
			Interval: float64(smallIvl.Milliseconds()),
			Ignored:  []string{},
		},
		wantCode: http.StatusUnprocessableEntity,
		wantErr:  "unsupported interval: less than an hour\n",
	}, {
		name: "big_interval",
		body: getConfigResp{
			Enabled:  aghalg.NBTrue,
			Interval: float64(maxIvl.Milliseconds() + minIvl.Milliseconds()),
			Ignored:  []string{},
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
		},
		wantCode: http.StatusOK,
		wantErr:  "",
	}, {
		name: "enabled_is_null",
		body: getConfigResp{
			Enabled:  aghalg.NBNull,
			Interval: float64(minIvl.Milliseconds()),
			Ignored:  []string{},
		},
		wantCode: http.StatusUnprocessableEntity,
		wantErr:  "enabled is null\n",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			s, err := New(conf)
			require.NoError(t, err)

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
