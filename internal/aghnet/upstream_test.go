package aghnet_test

import (
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/stretchr/testify/assert"
)

func TestIsCommentOrEmpty(t *testing.T) {
	for _, tc := range []struct {
		want assert.BoolAssertionFunc
		str  string
	}{{
		want: assert.True,
		str:  "",
	}, {
		want: assert.True,
		str:  "# comment",
	}, {
		want: assert.False,
		str:  "1.2.3.4",
	}} {
		tc.want(t, aghnet.IsCommentOrEmpty(tc.str))
	}
}
