package stats

import (
	"fmt"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/AdguardTeam/golibs/testutil"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO(e.burkov):  Use more realistic data.
func TestStatsCollector(t *testing.T) {
	ng := func(_ *unitDB) uint64 { return 0 }
	units := make([]*unitDB, 720)

	t.Run("hours", func(t *testing.T) {
		statsData := statsCollector(units, 0, Hours, ng)
		assert.Len(t, statsData, 720)
	})

	t.Run("days", func(t *testing.T) {
		for i := 0; i != 25; i++ {
			statsData := statsCollector(units, uint32(i), Days, ng)
			require.Lenf(t, statsData, 30, "i=%d", i)
		}
	})
}

func TestStats_races(t *testing.T) {
	var r uint32
	idGen := func() (id uint32) { return atomic.LoadUint32(&r) }
	conf := Config{
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
		e := Entry{
			Domain: fmt.Sprintf("example-%d.org", i),
			Client: fmt.Sprintf("client_%d", i),
			Result: Result(i)%(resultLast-1) + 1,
			Time:   uint32(time.Since(startTime).Milliseconds()),
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

		for i := 0; i < writersNum; i++ {
			startWG.Add(1)
			finWG.Add(1)
			go writeFunc(startWG, finWG, waitCh, i)
		}

		for i := 0; i < readersNum; i++ {
			startWG.Add(1)
			finWG.Add(1)
			go readFunc(startWG, finWG, waitCh)
		}

		startWG.Wait()
		close(waitCh)
		finWG.Wait()
	}
}
