//go:build openbsd
// +build openbsd

package aghnet

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHostnameIfStaticConfig(t *testing.T) {
	const nl = "\n"

	testCases := []struct {
		name       string
		rcconfData string
		wantHas    bool
	}{{
		name:       "simple",
		rcconfData: `inet 127.0.0.253` + nl,
		wantHas:    true,
	}, {
		name:       "case_sensitiveness",
		rcconfData: `InEt 127.0.0.253` + nl,
		wantHas:    false,
	}, {
		name: "comments_and_trash",
		rcconfData: `# comment 1` + nl +
			`` + nl +
			`# inet 127.0.0.253` + nl +
			`inet` + nl,
		wantHas: false,
	}, {
		name: "incorrect_config",
		rcconfData: `inet6 127.0.0.253` + nl +
			`inet 256.256.256.256` + nl,
		wantHas: false,
	}}

	for _, tc := range testCases {
		r := strings.NewReader(tc.rcconfData)
		t.Run(tc.name, func(t *testing.T) {
			has, err := hostnameIfStaticConfig(r)
			require.NoError(t, err)

			assert.Equal(t, tc.wantHas, has)
		})
	}
}
