package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitNext(t *testing.T) {
	s := " a,b , c "

	assert.Equal(t, "a", SplitNext(&s, ','))
	assert.Equal(t, "b", SplitNext(&s, ','))
	assert.Equal(t, "c", SplitNext(&s, ','))
	require.Empty(t, s)
}
