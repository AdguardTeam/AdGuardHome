package aghnet

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIPMap_allocs(t *testing.T) {
	ip4 := net.IP{1, 2, 3, 4}
	m := NewIPMap(0)
	m.Set(ip4, 42)

	t.Run("get", func(t *testing.T) {
		var v interface{}
		var ok bool
		allocs := testing.AllocsPerRun(100, func() {
			v, ok = m.Get(ip4)
		})

		require.True(t, ok)
		require.Equal(t, 42, v)

		assert.Equal(t, float64(0), allocs)
	})

	t.Run("len", func(t *testing.T) {
		var n int
		allocs := testing.AllocsPerRun(100, func() {
			n = m.Len()
		})

		require.Equal(t, 1, n)

		assert.Equal(t, float64(0), allocs)
	})
}

func TestIPMap(t *testing.T) {
	ip4 := net.IP{1, 2, 3, 4}
	ip6 := net.IP{
		0x12, 0x34, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x56, 0x78,
	}

	val := 42

	t.Run("nil", func(t *testing.T) {
		var m *IPMap

		assert.NotPanics(t, func() {
			m.Del(ip4)
			m.Del(ip6)
		})

		assert.NotPanics(t, func() {
			v, ok := m.Get(ip4)
			assert.Nil(t, v)
			assert.False(t, ok)

			v, ok = m.Get(ip6)
			assert.Nil(t, v)
			assert.False(t, ok)
		})

		assert.NotPanics(t, func() {
			assert.Equal(t, 0, m.Len())
		})

		assert.NotPanics(t, func() {
			n := 0
			m.Range(func(_ net.IP, _ interface{}) (cont bool) {
				n++

				return true
			})

			assert.Equal(t, 0, n)
		})

		assert.Panics(t, func() {
			m.Set(ip4, val)
		})

		assert.Panics(t, func() {
			m.Set(ip6, val)
		})

		assert.NotPanics(t, func() {
			sclone := m.ShallowClone()
			assert.Nil(t, sclone)
		})
	})

	testIPMap := func(t *testing.T, ip net.IP, s string) {
		m := NewIPMap(0)
		assert.Equal(t, 0, m.Len())

		v, ok := m.Get(ip)
		assert.Nil(t, v)
		assert.False(t, ok)

		m.Set(ip, val)
		v, ok = m.Get(ip)
		assert.Equal(t, val, v)
		assert.True(t, ok)

		n := 0
		m.Range(func(ipKey net.IP, v interface{}) (cont bool) {
			assert.Equal(t, ip.To16(), ipKey)
			assert.Equal(t, val, v)

			n++

			return false
		})
		assert.Equal(t, 1, n)

		sclone := m.ShallowClone()
		assert.Equal(t, m, sclone)

		assert.Equal(t, s, m.String())

		m.Del(ip)
		v, ok = m.Get(ip)
		assert.Nil(t, v)
		assert.False(t, ok)
		assert.Equal(t, 0, m.Len())
	}

	t.Run("ipv4", func(t *testing.T) {
		testIPMap(t, ip4, "map[1.2.3.4:42]")
	})

	t.Run("ipv6", func(t *testing.T) {
		testIPMap(t, ip6, "map[1234::5678:42]")
	})
}
