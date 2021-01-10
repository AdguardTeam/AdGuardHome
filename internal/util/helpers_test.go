package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitNext(t *testing.T) {
	s := " a,b , c "
	assert.True(t, SplitNext(&s, ',') == "a")
	assert.True(t, SplitNext(&s, ',') == "b")
	assert.True(t, SplitNext(&s, ',') == "c" && len(s) == 0)
}
