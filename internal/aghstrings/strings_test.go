package aghstrings

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCloneSlice_family(t *testing.T) {
	a := []string{"1", "2", "3"}

	t.Run("cloneslice_simple", func(t *testing.T) {
		assert.Equal(t, a, CloneSlice(a))
	})

	t.Run("cloneslice_nil", func(t *testing.T) {
		assert.Nil(t, CloneSlice(nil))
	})

	t.Run("cloneslice_empty", func(t *testing.T) {
		assert.Equal(t, []string{}, CloneSlice([]string{}))
	})

	t.Run("clonesliceorempty_nil", func(t *testing.T) {
		assert.Equal(t, []string{}, CloneSliceOrEmpty(nil))
	})

	t.Run("clonesliceorempty_empty", func(t *testing.T) {
		assert.Equal(t, []string{}, CloneSliceOrEmpty([]string{}))
	})

	t.Run("clonesliceorempty_sameness", func(t *testing.T) {
		assert.Equal(t, CloneSlice(a), CloneSliceOrEmpty(a))
	})
}

func TestCoalesce(t *testing.T) {
	assert.Equal(t, "", Coalesce())
	assert.Equal(t, "a", Coalesce("a"))
	assert.Equal(t, "a", Coalesce("", "a"))
	assert.Equal(t, "a", Coalesce("a", ""))
	assert.Equal(t, "a", Coalesce("a", "b"))
}

func TestFilterOut(t *testing.T) {
	strs := []string{
		"1.2.3.4",
		"",
		"# 5.6.7.8",
	}

	want := []string{
		"1.2.3.4",
	}

	got := FilterOut(strs, IsCommentOrEmpty)
	assert.Equal(t, want, got)
}

func TestInSlice(t *testing.T) {
	simpleStrs := []string{"1", "2", "3"}

	testCases := []struct {
		name string
		str  string
		strs []string
		want bool
	}{{
		name: "yes",
		str:  "2",
		strs: simpleStrs,
		want: true,
	}, {
		name: "no",
		str:  "4",
		strs: simpleStrs,
		want: false,
	}, {
		name: "nil",
		str:  "any",
		strs: nil,
		want: false,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, InSlice(tc.strs, tc.str))
		})
	}
}

func TestSplitNext(t *testing.T) {
	t.Run("ordinary", func(t *testing.T) {
		s := " a,b , c "
		require.Equal(t, "a", SplitNext(&s, ','))
		require.Equal(t, "b", SplitNext(&s, ','))
		require.Equal(t, "c", SplitNext(&s, ','))

		assert.Empty(t, s)
	})

	t.Run("nil_source", func(t *testing.T) {
		assert.Equal(t, "", SplitNext(nil, 's'))
	})
}

func TestWriteToBuilder(t *testing.T) {
	b := &strings.Builder{}

	t.Run("single", func(t *testing.T) {
		assert.NotPanics(t, func() { WriteToBuilder(b, t.Name()) })
		assert.Equal(t, t.Name(), b.String())
	})

	b.Reset()
	t.Run("several", func(t *testing.T) {
		const (
			_1   = "one"
			_2   = "two"
			_123 = _1 + _2
		)
		assert.NotPanics(t, func() { WriteToBuilder(b, _1, _2) })
		assert.Equal(t, _123, b.String())
	})

	b.Reset()
	t.Run("nothing", func(t *testing.T) {
		assert.NotPanics(t, func() { WriteToBuilder(b) })
		assert.Equal(t, "", b.String())
	})

	t.Run("nil_builder", func(t *testing.T) {
		assert.Panics(t, func() { WriteToBuilder(nil, "a") })
	})
}
