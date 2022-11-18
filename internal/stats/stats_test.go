package stats_test

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/stats"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	testutil.DiscardLogOutput(m)
}

// constUnitID is the UnitIDGenFunc which always return 0.
func constUnitID() (id uint32) { return 0 }

func assertSuccessAndUnmarshal(t *testing.T, to any, handler http.Handler, req *http.Request) {
	t.Helper()

	require.NotNil(t, handler)

	rw := httptest.NewRecorder()

	handler.ServeHTTP(rw, req)
	require.Equal(t, http.StatusOK, rw.Code)

	data := rw.Body.Bytes()
	if to == nil {
		assert.Empty(t, data)

		return
	}

	err := json.Unmarshal(data, to)
	require.NoError(t, err)
}

func TestStats(t *testing.T) {
	cliIP := netutil.IPv4Localhost()
	cliIPStr := cliIP.String()

	handlers := map[string]http.Handler{}
	conf := stats.Config{
		Filename:  filepath.Join(t.TempDir(), "stats.db"),
		LimitDays: 1,
		UnitID:    constUnitID,
		HTTPRegister: func(_, url string, handler http.HandlerFunc) {
			handlers[url] = handler
		},
	}

	s, err := stats.New(conf)
	require.NoError(t, err)

	s.Start()
	testutil.CleanupAndRequireSuccess(t, s.Close)

	t.Run("data", func(t *testing.T) {
		const reqDomain = "domain"

		entries := []stats.Entry{{
			Domain: reqDomain,
			Client: cliIPStr,
			Result: stats.RFiltered,
			Time:   123456,
		}, {
			Domain: reqDomain,
			Client: cliIPStr,
			Result: stats.RNotFiltered,
			Time:   123456,
		}}

		wantData := &stats.StatsResp{
			TimeUnits:  "hours",
			TopQueried: []map[string]uint64{0: {reqDomain: 1}},
			TopClients: []map[string]uint64{0: {cliIPStr: 2}},
			TopBlocked: []map[string]uint64{0: {reqDomain: 1}},
			DNSQueries: []uint64{
				0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2,
			},
			BlockedFiltering: []uint64{
				0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1,
			},
			ReplacedSafebrowsing: []uint64{
				0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
			},
			ReplacedParental: []uint64{
				0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
				0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
			},
			NumDNSQueries:           2,
			NumBlockedFiltering:     1,
			NumReplacedSafebrowsing: 0,
			NumReplacedSafesearch:   0,
			NumReplacedParental:     0,
			AvgProcessingTime:       0.123456,
		}

		for _, e := range entries {
			s.Update(e)
		}

		data := &stats.StatsResp{}
		req := httptest.NewRequest(http.MethodGet, "/control/stats", nil)
		assertSuccessAndUnmarshal(t, data, handlers["/control/stats"], req)

		assert.Equal(t, wantData, data)
	})

	t.Run("tops", func(t *testing.T) {
		topClients := s.TopClientsIP(2)
		require.NotEmpty(t, topClients)

		assert.Equal(t, cliIP, topClients[0])
	})

	t.Run("reset", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/control/stats_reset", nil)
		assertSuccessAndUnmarshal(t, nil, handlers["/control/stats_reset"], req)

		_24zeroes := [24]uint64{}
		emptyData := &stats.StatsResp{
			TimeUnits:            "hours",
			TopQueried:           []map[string]uint64{},
			TopClients:           []map[string]uint64{},
			TopBlocked:           []map[string]uint64{},
			DNSQueries:           _24zeroes[:],
			BlockedFiltering:     _24zeroes[:],
			ReplacedSafebrowsing: _24zeroes[:],
			ReplacedParental:     _24zeroes[:],
		}

		req = httptest.NewRequest(http.MethodGet, "/control/stats", nil)
		data := &stats.StatsResp{}

		assertSuccessAndUnmarshal(t, data, handlers["/control/stats"], req)
		assert.Equal(t, emptyData, data)
	})
}

func TestLargeNumbers(t *testing.T) {
	var curHour uint32 = 1
	handlers := map[string]http.Handler{}

	conf := stats.Config{
		Filename:     filepath.Join(t.TempDir(), "stats.db"),
		LimitDays:    1,
		UnitID:       func() (id uint32) { return atomic.LoadUint32(&curHour) },
		HTTPRegister: func(_, url string, handler http.HandlerFunc) { handlers[url] = handler },
	}

	s, err := stats.New(conf)
	require.NoError(t, err)

	s.Start()
	testutil.CleanupAndRequireSuccess(t, s.Close)

	const (
		hoursNum      = 12
		cliNumPerHour = 1000
	)

	req := httptest.NewRequest(http.MethodGet, "/control/stats", nil)

	for h := 0; h < hoursNum; h++ {
		atomic.AddUint32(&curHour, 1)

		for i := 0; i < cliNumPerHour; i++ {
			ip := net.IP{127, 0, byte((i & 0xff00) >> 8), byte(i & 0xff)}
			e := stats.Entry{
				Domain: fmt.Sprintf("domain%d.hour%d", i, h),
				Client: ip.String(),
				Result: stats.RNotFiltered,
				Time:   123456,
			}
			s.Update(e)
		}
	}

	data := &stats.StatsResp{}
	assertSuccessAndUnmarshal(t, data, handlers["/control/stats"], req)
	assert.Equal(t, hoursNum*cliNumPerHour, int(data.NumDNSQueries))
}
