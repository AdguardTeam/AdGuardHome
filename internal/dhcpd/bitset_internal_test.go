package dhcpd

import (
	"math"
	"testing"
	"testing/quick"

	"github.com/stretchr/testify/assert"
)

func TestBitSet(t *testing.T) {
	t.Run("nil", func(t *testing.T) {
		var s *bitSet

		ok := s.isSet(0)
		assert.False(t, ok)

		assert.NotPanics(t, func() {
			s.set(0, true)
		})

		ok = s.isSet(0)
		assert.False(t, ok)

		assert.NotPanics(t, func() {
			s.set(0, false)
		})

		ok = s.isSet(0)
		assert.False(t, ok)
	})

	t.Run("non_nil", func(t *testing.T) {
		s := newBitSet()

		ok := s.isSet(0)
		assert.False(t, ok)

		s.set(0, true)

		ok = s.isSet(0)
		assert.True(t, ok)

		s.set(0, false)

		ok = s.isSet(0)
		assert.False(t, ok)
	})

	t.Run("non_nil_long", func(t *testing.T) {
		s := newBitSet()

		s.set(0, true)
		s.set(math.MaxUint64, true)
		assert.Len(t, s.words, 2)

		ok := s.isSet(0)
		assert.True(t, ok)

		ok = s.isSet(math.MaxUint64)
		assert.True(t, ok)
	})

	t.Run("compare_to_map", func(t *testing.T) {
		m := map[uint64]struct{}{}
		s := newBitSet()

		mapFunc := func(setNew, checkOld, delOld uint64) (ok bool) {
			m[setNew] = struct{}{}
			delete(m, delOld)
			_, ok = m[checkOld]

			return ok
		}

		setFunc := func(setNew, checkOld, delOld uint64) (ok bool) {
			s.set(setNew, true)
			s.set(delOld, false)
			ok = s.isSet(checkOld)

			return ok
		}

		err := quick.CheckEqual(mapFunc, setFunc, &quick.Config{
			MaxCount:      10_000,
			MaxCountScale: 10,
		})
		assert.NoError(t, err)
	})
}
