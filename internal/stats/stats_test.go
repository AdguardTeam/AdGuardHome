package stats

import (
	"fmt"
	"net"
	"os"
	"sync/atomic"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	testutil.DiscardLogOutput(m)
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
	s, _ := createObject(conf)

	e := Entry{}

	e.Domain = "domain"
	e.Client = "127.0.0.1"
	e.Result = RFiltered
	e.Time = 123456
	s.Update(e)

	e.Domain = "domain"
	e.Client = "127.0.0.1"
	e.Result = RNotFiltered
	e.Time = 123456
	s.Update(e)

	d, ok := s.getData()
	assert.True(t, ok)

	a := []uint64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2}
	assert.True(t, UIntArrayEquals(d.DNSQueries, a))

	a = []uint64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
	assert.True(t, UIntArrayEquals(d.BlockedFiltering, a))

	a = []uint64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	assert.True(t, UIntArrayEquals(d.ReplacedSafebrowsing, a))

	a = []uint64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	assert.True(t, UIntArrayEquals(d.ReplacedParental, a))

	m := d.TopQueried
	assert.EqualValues(t, 1, m[0]["domain"])

	m = d.TopBlocked
	assert.EqualValues(t, 1, m[0]["domain"])

	m = d.TopClients
	assert.EqualValues(t, 2, m[0]["127.0.0.1"])

	assert.EqualValues(t, 2, d.NumDNSQueries)
	assert.EqualValues(t, 1, d.NumBlockedFiltering)
	assert.EqualValues(t, 0, d.NumReplacedSafebrowsing)
	assert.EqualValues(t, 0, d.NumReplacedSafesearch)
	assert.EqualValues(t, 0, d.NumReplacedParental)
	assert.EqualValues(t, 0.123456, d.AvgProcessingTime)

	topClients := s.GetTopClientsIP(2)
	assert.True(t, net.IP{127, 0, 0, 1}.Equal(topClients[0]))

	s.clear()
	s.Close()
	os.Remove(conf.Filename)
}

func TestLargeNumbers(t *testing.T) {
	var hour int32 = 1
	newID := func() uint32 {
		// use "atomic" to make Go race detector happy
		return uint32(atomic.LoadInt32(&hour))
	}

	// log.SetLevel(log.DEBUG)
	conf := Config{
		Filename:  "./stats.db",
		LimitDays: 1,
		UnitID:    newID,
	}
	os.Remove(conf.Filename)
	s, _ := createObject(conf)
	e := Entry{}

	n := 1000 // number of distinct clients and domains every hour
	for h := 0; h != 12; h++ {
		if h != 0 {
			atomic.AddInt32(&hour, 1)
		}
		for i := 0; i != n; i++ {
			e.Domain = fmt.Sprintf("domain%d", i)
			ip := net.IP{127, 0, 0, 1}
			ip[2] = byte((i & 0xff00) >> 8)
			ip[3] = byte(i & 0xff)
			e.Client = ip.String()
			e.Result = RNotFiltered
			e.Time = 123456
			s.Update(e)
		}
	}

	d, ok := s.getData()
	assert.True(t, ok)
	assert.EqualValues(t, int(hour)*n, d.NumDNSQueries)

	s.Close()
	os.Remove(conf.Filename)
}

// this code is a chunk copied from getData() that generates aggregate data per day
func aggregateDataPerDay(firstID uint32) int {
	firstDayID := (firstID + 24 - 1) / 24 * 24 // align_ceil(24)
	a := []uint64{}
	var sum uint64
	id := firstDayID
	nextDayID := firstDayID + 24
	for i := firstDayID - firstID; int(i) != 720; i++ {
		sum++
		if id == nextDayID {
			a = append(a, sum)
			sum = 0
			nextDayID += 24
		}
		id++
	}
	if id <= nextDayID {
		a = append(a, sum)
	}
	return len(a)
}

func TestAggregateDataPerTimeUnit(t *testing.T) {
	for i := 0; i != 25; i++ {
		alen := aggregateDataPerDay(uint32(i))
		assert.Equalf(t, 30, alen, "i=%d", i)
	}
}
