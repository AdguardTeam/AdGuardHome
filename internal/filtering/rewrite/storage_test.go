package rewrite

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewDefaultStorage(t *testing.T) {
	items := []*Item{{
		Domain: "example.com",
		Answer: "answer.com",
	}}

	s, err := NewDefaultStorage(-1, items)
	require.NoError(t, err)

	require.Len(t, s.List(), 1)
}

func TestDefaultStorage_CRUD(t *testing.T) {
	var items []*Item

	s, err := NewDefaultStorage(-1, items)
	require.NoError(t, err)
	require.Len(t, s.List(), 0)

	item := &Item{Domain: "example.com", Answer: "answer.com"}

	err = s.Add(item)
	require.NoError(t, err)

	list := s.List()
	require.Len(t, list, 1)
	require.True(t, item.equal(list[0]))

	err = s.Remove(item)
	require.NoError(t, err)
	require.Len(t, s.List(), 0)
}
