package stats

import (
	"fmt"
	"net"
	"os"
	"sync/atomic"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	aghtest.DiscardLogOutput(m)
}

func UIntArrayEquals(a, b []uint64) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

func TestStats(t *testing.T) {
	conf := Config{
		Filename:  "./stats.db",
		LimitDays: 1,
	}

	s, err := createObject(conf)
	require.Nil(t, err)
	t.Cleanup(func() {
		s.clear()
		s.Close()
		assert.Nil(t, os.Remove(conf.Filename))
	})

	s.Update(Entry{
		Domain: "domain",
		Client: "127.0.0.1",
		Result: RFiltered,
		Time:   123456,
	})
	s.Update(Entry{
		Domain: "domain",
		Client: "127.0.0.1",
		Result: RNotFiltered,
		Time:   123456,
	})

	d, ok := s.getData()
	require.True(t, ok)

	a := []uint64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2}
	assert.True(t, UIntArrayEquals(d.DNSQueries, a))

	a = []uint64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
	assert.True(t, UIntArrayEquals(d.BlockedFiltering, a))

	a = []uint64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	assert.True(t, UIntArrayEquals(d.ReplacedSafebrowsing, a))

	a = []uint64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	assert.True(t, UIntArrayEquals(d.ReplacedParental, a))

	m := d.TopQueried
	require.NotEmpty(t, m)
	assert.EqualValues(t, 1, m[0]["domain"])

	m = d.TopBlocked
	require.NotEmpty(t, m)
	assert.EqualValues(t, 1, m[0]["domain"])

	m = d.TopClients
	require.NotEmpty(t, m)
	assert.EqualValues(t, 2, m[0]["127.0.0.1"])

	assert.EqualValues(t, 2, d.NumDNSQueries)
	assert.EqualValues(t, 1, d.NumBlockedFiltering)
	assert.EqualValues(t, 0, d.NumReplacedSafebrowsing)
	assert.EqualValues(t, 0, d.NumReplacedSafesearch)
	assert.EqualValues(t, 0, d.NumReplacedParental)
	assert.EqualValues(t, 0.123456, d.AvgProcessingTime)

	topClients := s.GetTopClientsIP(2)
	require.NotEmpty(t, topClients)
	assert.True(t, net.IP{127, 0, 0, 1}.Equal(topClients[0]))
}

func TestLargeNumbers(t *testing.T) {
	var hour int32 = 0
	newID := func() uint32 {
		// Use "atomic" to make go race detector happy.
		return uint32(atomic.LoadInt32(&hour))
	}

	conf := Config{
		Filename:  "./stats.db",
		LimitDays: 1,
		UnitID:    newID,
	}
	s, err := createObject(conf)
	require.Nil(t, err)
	t.Cleanup(func() {
		s.Close()
		assert.Nil(t, os.Remove(conf.Filename))
	})

	// Number of distinct clients and domains every hour.
	const n = 1000

	for h := 0; h < 12; h++ {
		atomic.AddInt32(&hour, 1)
		for i := 0; i < n; i++ {
			s.Update(Entry{
				Domain: fmt.Sprintf("domain%d", i),
				Client: net.IP{
					127,
					0,
					byte((i & 0xff00) >> 8),
					byte(i & 0xff),
				}.String(),
				Result: RNotFiltered,
				Time:   123456,
			})
		}
	}

	d, ok := s.getData()
	require.True(t, ok)
	assert.EqualValues(t, hour*n, d.NumDNSQueries)
}

func TestStatsCollector(t *testing.T) {
	ng := func(_ *unitDB) uint64 {
		return 0
	}
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
