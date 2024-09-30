package stats

import (
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStats_races(t *testing.T) {
	var r uint32
	idGen := func() (id uint32) { return atomic.LoadUint32(&r) }
	conf := Config{
		Logger:            slogutil.NewDiscardLogger(),
		ShouldCountClient: func([]string) bool { return true },
		UnitID:            idGen,
		Filename:          filepath.Join(t.TempDir(), "./stats.db"),
		Limit:             timeutil.Day,
	}

	s, err := New(conf)
	require.NoError(t, err)

	s.Start()
	startTime := time.Now()
	testutil.CleanupAndRequireSuccess(t, s.Close)

	writeFunc := func(start, fin *sync.WaitGroup, waitCh <-chan unit, i int) {
		e := &Entry{
			Domain:         fmt.Sprintf("example-%d.org", i),
			Client:         fmt.Sprintf("client_%d", i),
			Result:         Result(i)%(resultLast-1) + 1,
			ProcessingTime: time.Since(startTime),
		}

		start.Done()
		defer fin.Done()

		<-waitCh

		s.Update(e)
	}
	readFunc := func(start, fin *sync.WaitGroup, waitCh <-chan unit) {
		start.Done()
		defer fin.Done()

		<-waitCh

		_, _ = s.getData(24)
	}

	const (
		roundsNum = 3

		writersNum = 10
		readersNum = 5
	)

	for round := 0; round < roundsNum; round++ {
		atomic.StoreUint32(&r, uint32(round))

		startWG, finWG := &sync.WaitGroup{}, &sync.WaitGroup{}
		waitCh := make(chan unit)

		for i := range writersNum {
			startWG.Add(1)
			finWG.Add(1)
			go writeFunc(startWG, finWG, waitCh, i)
		}

		for range readersNum {
			startWG.Add(1)
			finWG.Add(1)
			go readFunc(startWG, finWG, waitCh)
		}

		startWG.Wait()
		close(waitCh)
		finWG.Wait()
	}
}

func TestStatsCtx_FillCollectedStats_daily(t *testing.T) {
	const (
		daysCount = 10

		timeUnits = "days"
	)

	s, err := New(Config{
		Logger:            slogutil.NewDiscardLogger(),
		ShouldCountClient: func([]string) bool { return true },
		Filename:          filepath.Join(t.TempDir(), "./stats.db"),
		Limit:             time.Hour,
	})
	require.NoError(t, err)

	testutil.CleanupAndRequireSuccess(t, s.Close)

	sum := make([][]uint64, resultLast)
	sum[RFiltered] = make([]uint64, daysCount)
	sum[RSafeBrowsing] = make([]uint64, daysCount)
	sum[RParental] = make([]uint64, daysCount)

	total := make([]uint64, daysCount)

	dailyData := []*unitDB{}

	for i := range daysCount * 24 {
		n := uint64(i)
		nResult := make([]uint64, resultLast)
		nResult[RFiltered] = n
		nResult[RSafeBrowsing] = n
		nResult[RParental] = n

		day := i / 24
		sum[RFiltered][day] += n
		sum[RSafeBrowsing][day] += n
		sum[RParental][day] += n

		t := n * 3

		total[day] += t

		dailyData = append(dailyData, &unitDB{
			NTotal:  t,
			NResult: nResult,
		})
	}

	data := &StatsResp{}

	// In this way we will not skip first hours.
	curID := uint32(daysCount * 24)

	s.fillCollectedStats(data, dailyData, curID)

	assert.Equal(t, timeUnits, data.TimeUnits)
	assert.Equal(t, sum[RFiltered], data.BlockedFiltering)
	assert.Equal(t, sum[RSafeBrowsing], data.ReplacedSafebrowsing)
	assert.Equal(t, sum[RParental], data.ReplacedParental)
	assert.Equal(t, total, data.DNSQueries)
}

func TestStatsCtx_DataFromUnits_month(t *testing.T) {
	const hoursInMonth = 720

	s, err := New(Config{
		Logger:            slogutil.NewDiscardLogger(),
		ShouldCountClient: func([]string) bool { return true },
		Filename:          filepath.Join(t.TempDir(), "./stats.db"),
		Limit:             time.Hour,
	})
	require.NoError(t, err)

	testutil.CleanupAndRequireSuccess(t, s.Close)

	units, curID := s.loadUnits(hoursInMonth)
	require.Len(t, units, hoursInMonth)

	var h uint32
	for h = 1; h <= hoursInMonth; h++ {
		data := s.dataFromUnits(units[:h], curID)
		require.NotNil(t, data)
	}
}
