package stats

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnit_Deserialize(t *testing.T) {
	testCases := []struct {
		db   *unitDB
		name string
		want unit
	}{{
		name: "empty",
		want: unit{
			domains:            map[string]uint64{},
			blockedDomains:     map[string]uint64{},
			clients:            map[string]uint64{},
			nResult:            []uint64{0, 0, 0, 0, 0, 0},
			id:                 0,
			nTotal:             0,
			timeSum:            0,
			upstreamsResponses: map[string]uint64{},
			upstreamsTimeSum:   map[string]uint64{},
		},
		db: &unitDB{
			NResult:            []uint64{0, 0, 0, 0, 0, 0},
			Domains:            []countPair{},
			BlockedDomains:     []countPair{},
			Clients:            []countPair{},
			NTotal:             0,
			TimeAvg:            0,
			UpstreamsResponses: []countPair{},
			UpstreamsTimeSum:   []countPair{},
		},
	}, {
		name: "basic",
		want: unit{
			domains: map[string]uint64{
				"example.com": 1,
			},
			blockedDomains: map[string]uint64{
				"example.net": 1,
			},
			clients: map[string]uint64{
				"127.0.0.1": 2,
			},
			nResult: []uint64{0, 1, 1, 0, 0, 0},
			id:      0,
			nTotal:  2,
			timeSum: 246912,
			upstreamsResponses: map[string]uint64{
				"1.2.3.4": 2,
			},
			upstreamsTimeSum: map[string]uint64{
				"1.2.3.4": 246912,
			},
			upstreamsResponsesTotal: 2,
			upstreamsTimeSumTotal:   246912,
		},
		db: &unitDB{
			NResult: []uint64{0, 1, 1, 0, 0, 0},
			Domains: []countPair{{
				"example.com", 1,
			}},
			BlockedDomains: []countPair{{
				"example.net", 1,
			}},
			Clients: []countPair{{
				"127.0.0.1", 2,
			}},
			NTotal:  2,
			TimeAvg: 123456,
			UpstreamsResponses: []countPair{{
				"1.2.3.4", 2,
			}},
			UpstreamsTimeSum: []countPair{{
				"1.2.3.4", 246912,
			}},
		},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := unit{}
			got.deserialize(tc.db)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestTopUpstreamsPairs(t *testing.T) {
	testCases := []struct {
		db            *unitDB
		name          string
		wantResponses []topAddrs
		wantAvgTime   []topAddrsFloat
	}{{
		name: "empty",
		db: &unitDB{
			NResult:            []uint64{0, 0, 0, 0, 0, 0},
			Domains:            []countPair{},
			BlockedDomains:     []countPair{},
			Clients:            []countPair{},
			NTotal:             0,
			TimeAvg:            0,
			UpstreamsResponses: []countPair{},
			UpstreamsTimeSum:   []countPair{},
		},
		wantResponses: []topAddrs{},
		wantAvgTime:   []topAddrsFloat{},
	}, {
		name: "basic",
		db: &unitDB{
			NResult:        []uint64{0, 0, 0, 0, 0, 0},
			Domains:        []countPair{},
			BlockedDomains: []countPair{},
			Clients:        []countPair{},
			NTotal:         0,
			TimeAvg:        0,
			UpstreamsResponses: []countPair{{
				"1.2.3.4", 2,
			}},
			UpstreamsTimeSum: []countPair{{
				"1.2.3.4", 246912,
			}},
		},
		wantResponses: []topAddrs{{
			"1.2.3.4": 2,
		}},
		wantAvgTime: []topAddrsFloat{{
			"1.2.3.4": 0.123456,
		}},
	}, {
		name: "sorted",
		db: &unitDB{
			NResult:        []uint64{0, 0, 0, 0, 0, 0},
			Domains:        []countPair{},
			BlockedDomains: []countPair{},
			Clients:        []countPair{},
			NTotal:         0,
			TimeAvg:        0,
			UpstreamsResponses: []countPair{
				{"3.3.3.3", 8},
				{"2.2.2.2", 4},
				{"4.4.4.4", 16},
				{"1.1.1.1", 2},
			},
			UpstreamsTimeSum: []countPair{
				{"3.3.3.3", 800_000_000},
				{"2.2.2.2", 40_000_000},
				{"4.4.4.4", 16_000_000_000},
				{"1.1.1.1", 2_000_000},
			},
		},
		wantResponses: []topAddrs{
			{"4.4.4.4": 16},
			{"3.3.3.3": 8},
			{"2.2.2.2": 4},
			{"1.1.1.1": 2},
		},
		wantAvgTime: []topAddrsFloat{
			{"4.4.4.4": 1000},
			{"3.3.3.3": 100},
			{"2.2.2.2": 10},
			{"1.1.1.1": 1},
		},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotResponses, gotAvgTime := topUpstreamsPairs([]*unitDB{tc.db})
			assert.Equal(t, tc.wantResponses, gotResponses)
			assert.Equal(t, tc.wantAvgTime, gotAvgTime)
		})
	}
}

// TestStatsCtx_dataFromUnits_avgProcessingTime makes sure that the average
// processing time is weighted by the number of requests of each unit.
//
// [unitDB.TimeAvg] is the mean within a single unit, so taking the mean of the
// means would give a nearly idle hour the same weight as a busy one.
func TestStatsCtx_dataFromUnits_avgProcessingTime(t *testing.T) {
	t.Parallel()

	// A busy unit with a low mean and a nearly idle one with a high mean.  The
	// weighted mean is dominated by the busy unit:
	//
	//	(1000*1000 + 100000*10) / (1000 + 10) = 1980.198 microseconds
	//
	// The unweighted mean of the means would be (1000 + 100000) / 2 = 50500,
	// which is off by more than an order of magnitude.
	units := []*unitDB{{
		NResult: make([]uint64, resultLast),
		NTotal:  1000,
		TimeAvg: 1000,
	}, {
		NResult: make([]uint64, resultLast),
		NTotal:  10,
		TimeAvg: 100_000,
	}}

	s := &StatsCtx{}
	resp := s.dataFromUnits(units, 0)

	wantAvg := microsecondsToSeconds(2_000_000 / 1010.0)
	assert.InDelta(t, wantAvg, resp.AvgProcessingTime, 1e-12)
	assert.Equal(t, uint64(1010), resp.NumDNSQueries)
}

// TestAvgUpstreamResponseTime_truncation makes sure that the global average is
// not computed from the two bounded per-upstream lists.
//
// [unit.serialize] truncates upstreamsResponses and upstreamsTimeSum to the top
// [maxUpstreams] entries independently of each other, so on a unit that has
// seen more upstreams than that the two need not describe the same set, and
// dividing one total by the other compares two different populations.
func TestAvgUpstreamResponseTime_truncation(t *testing.T) {
	t.Parallel()

	u := newUnit(0)

	// Two groups, each maxUpstreams large, ranked oppositely: the first wins on
	// response count, the second on total duration.  Truncation therefore keeps
	// the counts of one and the durations of the other.
	const (
		countHeavyResponses, countHeavyTime = 100, 1
		timeHeavyResponses, timeHeavyTime   = 99, 99
	)

	for i := range maxUpstreams {
		countHeavy := fmt.Sprintf("count-heavy-%03d", i)
		u.upstreamsResponses[countHeavy] = countHeavyResponses
		u.upstreamsTimeSum[countHeavy] = countHeavyTime

		timeHeavy := fmt.Sprintf("time-heavy-%03d", i)
		u.upstreamsResponses[timeHeavy] = timeHeavyResponses
		u.upstreamsTimeSum[timeHeavy] = timeHeavyTime
	}

	u.upstreamsResponsesTotal = maxUpstreams * (countHeavyResponses + timeHeavyResponses)
	u.upstreamsTimeSumTotal = maxUpstreams * (countHeavyTime + timeHeavyTime)

	udb := u.serialize()

	// Both lists are bounded, and they hold different upstreams.
	require.Len(t, udb.UpstreamsResponses, maxUpstreams)
	require.Len(t, udb.UpstreamsTimeSum, maxUpstreams)

	wantAvg := microsecondsToSeconds(
		float64(u.upstreamsTimeSumTotal) / float64(u.upstreamsResponsesTotal),
	)

	assert.InDelta(t, wantAvg, avgUpstreamResponseTime([]*unitDB{udb}), 1e-15)
}

// TestAvgUpstreamResponseTime_oldRecord makes sure that a record written before
// the exact totals existed still yields an average, from the bounded lists that
// are all it has.
func TestAvgUpstreamResponseTime_oldRecord(t *testing.T) {
	t.Parallel()

	udb := &unitDB{
		UpstreamsResponses: []countPair{{Name: "1.2.3.4", Count: 4}},
		UpstreamsTimeSum:   []countPair{{Name: "1.2.3.4", Count: 800}},
	}

	assert.InDelta(t, microsecondsToSeconds(200), avgUpstreamResponseTime([]*unitDB{udb}), 1e-15)
}
