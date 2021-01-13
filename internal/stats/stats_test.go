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
	e.Client = net.IP{127, 0, 0, 1}
	e.Result = RFiltered
	e.Time = 123456
	s.Update(e)

	e.Domain = "domain"
	e.Client = net.IP{127, 0, 0, 1}
	e.Result = RNotFiltered
	e.Time = 123456
	s.Update(e)

	d := s.getData()
	a := []uint64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2}
	assert.True(t, UIntArrayEquals(d["dns_queries"].([]uint64), a))

	a = []uint64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
	assert.True(t, UIntArrayEquals(d["blocked_filtering"].([]uint64), a))

	a = []uint64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	assert.True(t, UIntArrayEquals(d["replaced_safebrowsing"].([]uint64), a))

	a = []uint64{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	assert.True(t, UIntArrayEquals(d["replaced_parental"].([]uint64), a))

	m := d["top_queried_domains"].([]map[string]uint64)
	assert.EqualValues(t, 1, m[0]["domain"])

	m = d["top_blocked_domains"].([]map[string]uint64)
	assert.EqualValues(t, 1, m[0]["domain"])

	m = d["top_clients"].([]map[string]uint64)
	assert.EqualValues(t, 2, m[0]["127.0.0.1"])

	assert.EqualValues(t, 2, d["num_dns_queries"].(uint64))
	assert.EqualValues(t, 1, d["num_blocked_filtering"].(uint64))
	assert.EqualValues(t, 0, d["num_replaced_safebrowsing"].(uint64))
	assert.EqualValues(t, 0, d["num_replaced_safesearch"].(uint64))
	assert.EqualValues(t, 0, d["num_replaced_parental"].(uint64))
	assert.EqualValues(t, 0.123456, d["avg_processing_time"].(float64))

	topClients := s.GetTopClientsIP(2)
	assert.Equal(t, "127.0.0.1", topClients[0])

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
			e.Client = net.IP{127, 0, 0, 1}
			e.Client[2] = byte((i & 0xff00) >> 8)
			e.Client[3] = byte(i & 0xff)
			e.Result = RNotFiltered
			e.Time = 123456
			s.Update(e)
		}
	}

	d := s.getData()
	assert.EqualValues(t, int(hour)*n, d["num_dns_queries"])

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
