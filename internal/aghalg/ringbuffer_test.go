package aghalg_test

import (
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghalg"
	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slices"
)

// elements is a helper function that returns n elements of the buffer.
func elements(b *aghalg.RingBuffer[int], n int, reverse bool) (es []int) {
	fn := b.Range
	if reverse {
		fn = b.ReverseRange
	}

	i := 0
	fn(func(e int) (cont bool) {
		if i >= n {
			return false
		}

		es = append(es, e)
		i++

		return true
	})

	return es
}

func TestNewRingBuffer(t *testing.T) {
	t.Run("success_and_clear", func(t *testing.T) {
		b := aghalg.NewRingBuffer[int](5)
		for i := 0; i < 10; i++ {
			b.Append(i)
		}
		assert.Equal(t, []int{5, 6, 7, 8, 9}, elements(b, b.Len(), false))

		b.Clear()
		assert.Zero(t, b.Len())
	})

	t.Run("negative_size", func(t *testing.T) {
		assert.PanicsWithError(t, "ring buffer: size must be greater or equal to zero", func() {
			aghalg.NewRingBuffer[int](-5)
		})
	})

	t.Run("zero", func(t *testing.T) {
		b := aghalg.NewRingBuffer[int](0)
		for i := 0; i < 10; i++ {
			b.Append(i)
			assert.Equal(t, 0, b.Len())
			assert.Empty(t, elements(b, b.Len(), false))
			assert.Empty(t, elements(b, b.Len(), true))
		}
	})

	t.Run("single", func(t *testing.T) {
		b := aghalg.NewRingBuffer[int](1)
		for i := 0; i < 10; i++ {
			b.Append(i)
			assert.Equal(t, 1, b.Len())
			assert.Equal(t, []int{i}, elements(b, b.Len(), false))
			assert.Equal(t, []int{i}, elements(b, b.Len(), true))
		}
	})
}

func TestRingBuffer_Range(t *testing.T) {
	const size = 5

	b := aghalg.NewRingBuffer[int](size)

	testCases := []struct {
		name   string
		want   []int
		count  int
		length int
	}{{
		name:   "three",
		count:  3,
		length: 3,
		want:   []int{0, 1, 2},
	}, {
		name:   "ten",
		count:  10,
		length: size,
		want:   []int{5, 6, 7, 8, 9},
	}, {
		name:   "hundred",
		count:  100,
		length: size,
		want:   []int{95, 96, 97, 98, 99},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			for i := 0; i < tc.count; i++ {
				b.Append(i)
			}

			bufLen := b.Len()
			assert.Equal(t, tc.length, bufLen)

			want := tc.want
			assert.Equal(t, want, elements(b, bufLen, false))
			assert.Equal(t, want[:len(want)-1], elements(b, bufLen-1, false))
			assert.Equal(t, want[:len(want)/2], elements(b, bufLen/2, false))

			want = want[:cap(want)]
			slices.Reverse(want)

			assert.Equal(t, want, elements(b, bufLen, true))
			assert.Equal(t, want[:len(want)-1], elements(b, bufLen-1, true))
			assert.Equal(t, want[:len(want)/2], elements(b, bufLen/2, true))
		})
	}
}

func TestRingBuffer_Range_increment(t *testing.T) {
	const size = 5

	b := aghalg.NewRingBuffer[int](size)

	testCases := []struct {
		name string
		want []int
	}{{
		name: "one",
		want: []int{0},
	}, {
		name: "two",
		want: []int{0, 1},
	}, {
		name: "three",
		want: []int{0, 1, 2},
	}, {
		name: "four",
		want: []int{0, 1, 2, 3},
	}, {
		name: "five",
		want: []int{0, 1, 2, 3, 4},
	}, {
		name: "six",
		want: []int{1, 2, 3, 4, 5},
	}, {
		name: "seven",
		want: []int{2, 3, 4, 5, 6},
	}, {
		name: "eight",
		want: []int{3, 4, 5, 6, 7},
	}, {
		name: "nine",
		want: []int{4, 5, 6, 7, 8},
	}, {
		name: "ten",
		want: []int{5, 6, 7, 8, 9},
	}}

	for i, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			b.Append(i)

			assert.Equal(t, tc.want, elements(b, b.Len(), false))

			slices.Reverse(tc.want)
			assert.Equal(t, tc.want, elements(b, b.Len(), true))
		})
	}
}
