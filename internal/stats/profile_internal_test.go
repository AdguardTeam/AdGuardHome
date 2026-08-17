package stats

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const profileCurrentUnitID uint32 = 10_000

func TestStatsCtx_LoadUnitsDoesNotBlockUpdate(t *testing.T) {
	s := newTestStatsCtx(t, Config{
		Enabled: true,
		UnitID:  func() (id uint32) { return profileCurrentUnitID },
	})
	t.Cleanup(func() { require.NoError(t, s.Close()) })

	db := s.db.Load()
	blockingTx, err := db.Begin(true)
	require.NoError(t, err)

	txOpen := true
	t.Cleanup(func() {
		if txOpen {
			require.NoError(t, finishTxn(blockingTx, false))
		}
	})

	var (
		units []*unitDB
		curID uint32
	)
	loadDone := make(chan struct{})
	go func() {
		defer close(loadDone)

		units, curID = s.loadUnits(1)
	}()

	updateDone := make(chan struct{})
	go func() {
		s.Update(&Entry{
			Client: "192.0.2.1",
			Domain: "example.org",
			Result: RNotFiltered,
		})
		close(updateDone)
	}()

	// A read-only transaction neither waits for the active writer nor keeps
	// currMu locked while loading the database.
	testutil.RequireReceive(t, loadDone, time.Second)
	testutil.RequireReceive(t, updateDone, time.Second)
	require.Len(t, units, 1)
	assert.Equal(t, profileCurrentUnitID, curID)

	require.NoError(t, finishTxn(blockingTx, false))
	txOpen = false

	assert.Equal(t, uint64(1), s.curr.nTotal)
}

type profileLoadHandler struct {
	slog.Handler

	loadEntered  chan<- struct{}
	flushEntered chan<- struct{}
	loadRelease  <-chan struct{}
	loadOnce     sync.Once
	flushOnce    sync.Once
}

func (h *profileLoadHandler) Handle(ctx context.Context, r slog.Record) (err error) {
	switch r.Message {
	case "loading unit":
		h.loadOnce.Do(func() {
			close(h.loadEntered)
			<-h.loadRelease
		})
	case "flushing unit":
		h.flushOnce.Do(func() { close(h.flushEntered) })
	}

	return h.Handler.Handle(ctx, r)
}

func TestStatsCtx_LoadUnitsConcurrentFlush(t *testing.T) {
	const limit uint32 = 3

	loadEntered := make(chan struct{})
	loadRelease := make(chan struct{})
	flushEntered := make(chan struct{})

	curID := profileCurrentUnitID
	s := newTestStatsCtx(t, Config{
		Enabled: true,
		Limit:   time.Duration(limit) * time.Hour,
		UnitID:  func() (id uint32) { return curID },
	})
	t.Cleanup(func() { require.NoError(t, s.Close()) })

	db := s.db.Load()
	tx, err := db.Begin(true)
	require.NoError(t, err)

	for i, n := range []uint64{1, 2} {
		udb := &unitDB{
			NResult: make([]uint64, resultLast),
			NTotal:  n,
		}

		err = s.flushUnitToDB(udb, tx, curID-limit+1+uint32(i))
		require.NoError(t, err)
	}
	require.NoError(t, finishTxn(tx, true))

	for range 3 {
		s.Update(&Entry{
			Client: "192.0.2.1",
			Domain: "example.org",
			Result: RNotFiltered,
		})
	}

	s.logger = slog.New(&profileLoadHandler{
		Handler:      slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}),
		loadEntered:  loadEntered,
		flushEntered: flushEntered,
		loadRelease:  loadRelease,
	})

	var (
		units       []*unitDB
		loadedCurID uint32
	)
	loadDone := make(chan struct{})
	go func() {
		units, loadedCurID = s.loadUnits(limit)
		close(loadDone)
	}()

	testutil.RequireReceive(t, loadEntered, time.Second)

	var (
		cont     bool
		sleepFor time.Duration
	)
	flushDone := make(chan struct{})
	curID++
	go func() {
		cont, sleepFor = s.flush()
		close(flushDone)
	}()

	var releaseOnce sync.Once
	releaseLoad := func() { releaseOnce.Do(func() { close(loadRelease) }) }
	t.Cleanup(releaseLoad)

	// The flush must be able to replace the current unit and begin persisting it
	// while loadUnits is decoding its database view.
	testutil.RequireReceive(t, flushEntered, time.Second)
	releaseLoad()
	testutil.RequireReceive(t, loadDone, time.Second)
	testutil.RequireReceive(t, flushDone, time.Second)

	assert.True(t, cont)
	assert.Zero(t, sleepFor)
	assert.Equal(t, profileCurrentUnitID, loadedCurID)
	if assert.Len(t, units, int(limit)) {
		assert.Equal(t, uint64(1), units[0].NTotal)
		assert.Equal(t, uint64(2), units[1].NTotal)
		assert.Equal(t, uint64(3), units[2].NTotal)
	}
}

func TestStatsCtx_LoadUnitsDoesNotBlockFlushAtMapGrowth(t *testing.T) {
	const (
		limit        uint32 = 2
		longNameSize        = 16 * 1024
	)

	loadEntered := make(chan struct{})
	loadRelease := make(chan struct{})
	flushEntered := make(chan struct{})

	curID := profileCurrentUnitID
	s := newTestStatsCtx(t, Config{
		Enabled: true,
		Limit:   time.Duration(limit) * time.Hour,
		UnitID:  func() (id uint32) { return curID },
	})
	t.Cleanup(func() { require.NoError(t, s.Close()) })

	db := s.db.Load()
	tx, err := db.Begin(true)
	require.NoError(t, err)

	err = s.flushUnitToDB(&unitDB{NResult: make([]uint64, resultLast)}, tx, curID-1)
	require.NoError(t, err)
	require.NoError(t, finishTxn(tx, true))

	infoBefore, err := os.Stat(s.filename)
	require.NoError(t, err)

	longName := strings.Repeat("a", longNameSize)
	for i := range maxDomains {
		s.curr.domains[fmt.Sprintf("%03d.%s", i, longName)] = uint64(i + 1)
	}

	s.logger = slog.New(&profileLoadHandler{
		Handler:      slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}),
		loadEntered:  loadEntered,
		flushEntered: flushEntered,
		loadRelease:  loadRelease,
	})

	loadDone := make(chan struct{})
	go func() {
		_, _ = s.loadUnits(limit)
		close(loadDone)
	}()

	testutil.RequireReceive(t, loadEntered, time.Second)

	flushDone := make(chan struct{})
	curID++
	go func() {
		_, _ = s.flush()
		close(flushDone)
	}()

	var releaseOnce sync.Once
	releaseLoad := func() { releaseOnce.Do(func() { close(loadRelease) }) }
	t.Cleanup(releaseLoad)

	// The read transaction must be closed before decoding begins.  Otherwise a
	// map-growing flush holds currMu while waiting for the loader to finish.
	testutil.RequireReceive(t, flushEntered, time.Second)

	updateDone := make(chan struct{})
	go func() {
		s.Update(&Entry{
			Client: "192.0.2.1",
			Domain: "example.org",
			Result: RNotFiltered,
		})
		close(updateDone)
	}()
	testutil.RequireReceive(t, updateDone, time.Second)

	releaseLoad()
	testutil.RequireReceive(t, loadDone, time.Second)
	testutil.RequireReceive(t, flushDone, time.Second)

	infoAfter, err := os.Stat(s.filename)
	require.NoError(t, err)
	assert.Greater(t, infoAfter.Size(), infoBefore.Size())
}

func BenchmarkStatsCtx_LoadUnits(b *testing.B) {
	for _, hours := range profileRetentionHours() {
		b.Run(fmt.Sprintf("%dh", hours), func(b *testing.B) {
			s, dbMiB := newProfileStatsCtx(b, hours)

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				units, _ := s.loadUnits(hours)
				runtime.KeepAlive(units)
			}
			b.ReportMetric(dbMiB, "MiB/db")
		})
	}
}

func BenchmarkStatsCtx_GetData(b *testing.B) {
	for _, hours := range profileRetentionHours() {
		b.Run(fmt.Sprintf("%dh", hours), func(b *testing.B) {
			s, dbMiB := newProfileStatsCtx(b, hours)

			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				resp, ok := s.getData(hours)
				if !ok {
					b.Fatal("getData failed")
				}

				runtime.KeepAlive(resp)
			}
			b.ReportMetric(dbMiB, "MiB/db")
		})
	}
}

func profileRetentionHours() (hours []uint32) {
	return []uint32{24, 168, 720, 2160}
}

func newProfileStatsCtx(b *testing.B, hours uint32) (s *StatsCtx, dbMiB float64) {
	b.Helper()

	s = newTestStatsCtx(b, Config{
		Enabled: true,
		Limit:   time.Duration(hours) * time.Hour,
		UnitID:  func() (id uint32) { return profileCurrentUnitID },
	})
	b.Cleanup(func() { require.NoError(b, s.Close()) })

	udb := newProfileUnitDB()
	db := s.db.Load()
	tx, err := db.Begin(true)
	require.NoError(b, err)

	firstID := profileCurrentUnitID - hours + 1
	for id := firstID; id != profileCurrentUnitID; id++ {
		err = s.flushUnitToDB(udb, tx, id)
		require.NoError(b, err)
	}

	require.NoError(b, finishTxn(tx, true))
	s.curr.deserialize(udb)

	info, err := os.Stat(s.filename)
	require.NoError(b, err)

	return s, float64(info.Size()) / (1024 * 1024)
}

func newProfileUnitDB() (udb *unitDB) {
	udb = &unitDB{
		NResult: make([]uint64, resultLast),
		NTotal:  10_000,
		TimeAvg: 25_000,
	}
	udb.NResult[RNotFiltered] = 8_000
	udb.NResult[RFiltered] = 2_000

	for i := range maxDomains {
		count := uint64(maxDomains - i)
		udb.Domains = append(udb.Domains, countPair{
			Name:  fmt.Sprintf("domain-%03d.example", i),
			Count: count,
		})
		udb.BlockedDomains = append(udb.BlockedDomains, countPair{
			Name:  fmt.Sprintf("blocked-%03d.example", i),
			Count: count,
		})
	}

	for i := range maxClients {
		udb.Clients = append(udb.Clients, countPair{
			Name:  fmt.Sprintf("2001:db8::%x", i+1),
			Count: uint64(maxClients - i),
		})
	}

	for i := range maxUpstreams {
		count := uint64(maxUpstreams - i)
		name := fmt.Sprintf("https://upstream-%03d.example/dns-query", i)
		udb.UpstreamsResponses = append(udb.UpstreamsResponses, countPair{
			Name:  name,
			Count: count,
		})
		udb.UpstreamsTimeSum = append(udb.UpstreamsTimeSum, countPair{
			Name:  name,
			Count: count * 25_000,
		})
	}

	return udb
}
