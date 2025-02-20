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
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/stats"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
		Logger:            slogutil.NewDiscardLogger(),
		ShouldCountClient: func([]string) bool { return true },
		Filename:          filepath.Join(t.TempDir(), "stats.db"),
		Limit:             timeutil.Day,
		Enabled:           true,
		UnitID:            constUnitID,
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
		const respUpstream = "upstream"

		entries := []*stats.Entry{{
			Domain:         reqDomain,
			Client:         cliIPStr,
			Result:         stats.RFiltered,
			ProcessingTime: time.Microsecond * 123456,
			UpstreamStats: []*proxy.UpstreamStatistics{{
				Address:       respUpstream,
				QueryDuration: time.Microsecond * 222222,
			}},
		}, {
			Domain:         reqDomain,
			Client:         cliIPStr,
			Result:         stats.RNotFiltered,
			ProcessingTime: time.Microsecond * 123456,
			UpstreamStats: []*proxy.UpstreamStatistics{{
				Address:       respUpstream,
				QueryDuration: time.Microsecond * 222222,
			}},
		}}

		wantData := &stats.StatsResp{
			TimeUnits:             "hours",
			TopQueried:            []map[string]uint64{0: {reqDomain: 1}},
			TopClients:            []map[string]uint64{0: {cliIPStr: 2}},
			TopBlocked:            []map[string]uint64{0: {reqDomain: 1}},
			TopUpstreamsResponses: []map[string]uint64{0: {respUpstream: 2}},
			TopUpstreamsAvgTime:   []map[string]float64{0: {respUpstream: 0.222222}},
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
			TimeUnits:             "hours",
			TopQueried:            []map[string]uint64{},
			TopClients:            []map[string]uint64{},
			TopBlocked:            []map[string]uint64{},
			TopUpstreamsResponses: []map[string]uint64{},
			TopUpstreamsAvgTime:   []map[string]float64{},
			DNSQueries:            _24zeroes[:],
			BlockedFiltering:      _24zeroes[:],
			ReplacedSafebrowsing:  _24zeroes[:],
			ReplacedParental:      _24zeroes[:],
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
		Logger:            slogutil.NewDiscardLogger(),
		ShouldCountClient: func([]string) bool { return true },
		Filename:          filepath.Join(t.TempDir(), "stats.db"),
		Limit:             timeutil.Day,
		Enabled:           true,
		UnitID:            func() (id uint32) { return atomic.LoadUint32(&curHour) },
		HTTPRegister:      func(_, url string, handler http.HandlerFunc) { handlers[url] = handler },
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

		for i := range cliNumPerHour {
			ip := net.IP{127, 0, byte((i & 0xff00) >> 8), byte(i & 0xff)}
			e := &stats.Entry{
				Domain:         fmt.Sprintf("domain%d.hour%d", i, h),
				Client:         ip.String(),
				Result:         stats.RNotFiltered,
				ProcessingTime: 123456,
			}
			s.Update(e)
		}
	}

	data := &stats.StatsResp{}
	assertSuccessAndUnmarshal(t, data, handlers["/control/stats"], req)
	assert.Equal(t, hoursNum*cliNumPerHour, int(data.NumDNSQueries))
}

func TestShouldCount(t *testing.T) {
	const (
		ignored1 = "ignor.ed"
		ignored2 = "ignored.to"
	)
	ignored := []string{ignored1, ignored2}
	engine, err := aghnet.NewIgnoreEngine(ignored)
	require.NoError(t, err)

	s, err := stats.New(stats.Config{
		Logger:   slogutil.NewDiscardLogger(),
		Enabled:  true,
		Filename: filepath.Join(t.TempDir(), "stats.db"),
		Limit:    timeutil.Day,
		Ignored:  engine,
		ShouldCountClient: func(ids []string) (a bool) {
			return ids[0] != "no_count"
		},
	})
	require.NoError(t, err)

	s.Start()
	testutil.CleanupAndRequireSuccess(t, s.Close)

	testCases := []struct {
		wantCount assert.BoolAssertionFunc
		name      string
		host      string
		ids       []string
	}{{
		name:      "count",
		host:      "example.com",
		ids:       []string{"whatever"},
		wantCount: assert.True,
	}, {
		name:      "no_count_ignored_1",
		host:      ignored1,
		ids:       []string{"whatever"},
		wantCount: assert.False,
	}, {
		name:      "no_count_ignored_2",
		host:      ignored2,
		ids:       []string{"whatever"},
		wantCount: assert.False,
	}, {
		name:      "no_count_client_ignore",
		host:      "example.com",
		ids:       []string{"no_count"},
		wantCount: assert.False,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res := s.ShouldCount(tc.host, dns.TypeA, dns.ClassINET, tc.ids)

			tc.wantCount(t, res)
		})
	}
}
