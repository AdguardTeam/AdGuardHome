package aghstrings

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSet(t *testing.T) {
	const s = "a"

	t.Run("nil", func(t *testing.T) {
		var set *Set

		assert.NotPanics(t, func() {
			set.Del(s)
		})

		assert.NotPanics(t, func() {
			assert.False(t, set.Has(s))
		})

		assert.NotPanics(t, func() {
			assert.Equal(t, 0, set.Len())
		})

		assert.NotPanics(t, func() {
			assert.Nil(t, set.Values())
		})

		assert.Panics(t, func() {
			set.Add(s)
		})
	})

	t.Run("non_nil", func(t *testing.T) {
		set := NewSet()
		assert.Equal(t, 0, set.Len())

		ok := set.Has(s)
		assert.False(t, ok)

		set.Add(s)
		ok = set.Has(s)
		assert.True(t, ok)

		assert.Equal(t, []string{s}, set.Values())

		set.Del(s)
		ok = set.Has(s)
		assert.False(t, ok)

		set = NewSet(s)
		assert.Equal(t, 1, set.Len())
	})
}
