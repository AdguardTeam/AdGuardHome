package stats

import (
	"fmt"
	"net"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func UIntArrayEquals(a []uint, b []uint) bool {
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
	s := New("./stats.db", 1, nil)

	e := Entry{}

	e.Domain = "domain"
	e.Client = net.ParseIP("127.0.0.1")
	e.Result = RFiltered
	e.Time = 123456
	s.Update(e)

	e.Domain = "domain"
	e.Client = net.ParseIP("127.0.0.1")
	e.Result = RNotFiltered
	e.Time = 123456
	s.Update(e)

	d := s.GetData(Hours)
	a := []uint{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 2}
	assert.True(t, UIntArrayEquals(d["dns_queries"].([]uint), a))

	a = []uint{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
	assert.True(t, UIntArrayEquals(d["blocked_filtering"].([]uint), a))

	a = []uint{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	assert.True(t, UIntArrayEquals(d["replaced_safebrowsing"].([]uint), a))

	a = []uint{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	assert.True(t, UIntArrayEquals(d["replaced_parental"].([]uint), a))

	m := d["top_queried_domains"].([]map[string]uint)
	assert.True(t, m[0]["domain"] == 1)

	m = d["top_blocked_domains"].([]map[string]uint)
	assert.True(t, m[0]["domain"] == 1)

	m = d["top_clients"].([]map[string]uint)
	assert.True(t, m[0]["127.0.0.1"] == 2)

	assert.True(t, d["num_dns_queries"].(uint) == 2)
	assert.True(t, d["num_blocked_filtering"].(uint) == 1)
	assert.True(t, d["num_replaced_safebrowsing"].(uint) == 0)
	assert.True(t, d["num_replaced_safesearch"].(uint) == 0)
	assert.True(t, d["num_replaced_parental"].(uint) == 0)
	assert.True(t, d["avg_processing_time"].(float64) == 0.123456)

	s.Clear()
	s.Close()
	os.Remove("./stats.db")
}

func TestLargeNumbers(t *testing.T) {
	var hour int32
	hour = 1
	newID := func() int {
		// use "atomic" to make Go race detector happy
		return int(atomic.LoadInt32(&hour))
	}

	// log.SetLevel(log.DEBUG)
	fn := "./stats.db"
	os.Remove(fn)
	s := New(fn, 1, newID)
	e := Entry{}

	n := 1000 // number of distinct clients and domains every hour
	for h := 0; h != 12; h++ {
		if h != 0 {
			atomic.AddInt32(&hour, 1)
			time.Sleep(1500 * time.Millisecond)
		}
		for i := 0; i != n; i++ {
			e.Domain = fmt.Sprintf("domain%d", i)
			e.Client = net.ParseIP("127.0.0.1")
			e.Client[2] = byte((i & 0xff00) >> 8)
			e.Client[3] = byte(i & 0xff)
			e.Result = RNotFiltered
			e.Time = 123456
			s.Update(e)
		}
	}

	d := s.GetData(Hours)
	assert.True(t, d["num_dns_queries"].(uint) == uint(int(hour)*n))

	s.Close()
	os.Remove(fn)
}
