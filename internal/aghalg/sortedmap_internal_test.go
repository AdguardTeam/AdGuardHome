package aghalg

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSortedMap(t *testing.T) {
	var m SortedMap[string, int]

	letters := []string{}
	for i := range 10 {
		r := string('a' + rune(i))
		letters = append(letters, r)
	}

	t.Run("create_and_fill", func(t *testing.T) {
		m = NewSortedMap[string, int](strings.Compare)

		nums := []int{}
		for i, r := range letters {
			m.Set(r, i)
			nums = append(nums, i)
		}

		gotLetters := []string{}
		gotNums := []int{}
		m.Range(func(k string, v int) bool {
			gotLetters = append(gotLetters, k)
			gotNums = append(gotNums, v)

			return true
		})

		assert.Equal(t, letters, gotLetters)
		assert.Equal(t, nums, gotNums)

		n, ok := m.Get(letters[0])
		assert.True(t, ok)
		assert.Equal(t, nums[0], n)
	})

	t.Run("clear", func(t *testing.T) {
		lastLetter := letters[len(letters)-1]
		m.Del(lastLetter)

		_, ok := m.Get(lastLetter)
		assert.False(t, ok)

		m.Clear()

		gotLetters := []string{}
		m.Range(func(k string, _ int) bool {
			gotLetters = append(gotLetters, k)

			return true
		})

		assert.Len(t, gotLetters, 0)
	})
}

func TestNewSortedMap_nil(t *testing.T) {
	const (
		key = "key"
		val = "val"
	)

	var m SortedMap[string, string]

	assert.Panics(t, func() {
		m.Set(key, val)
	})

	assert.NotPanics(t, func() {
		_, ok := m.Get(key)
		assert.False(t, ok)
	})

	assert.NotPanics(t, func() {
		m.Range(func(_, _ string) (cont bool) {
			return true
		})
	})

	assert.NotPanics(t, func() {
		m.Del(key)
	})

	assert.NotPanics(t, func() {
		m.Clear()
	})
}
