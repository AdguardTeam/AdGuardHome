//go:build freebsd
// +build freebsd

package aghnet

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRcConfStaticConfig(t *testing.T) {
	const ifaceName = `em0`
	const nl = "\n"

	testCases := []struct {
		name       string
		rcconfData string
		wantHas    bool
	}{{
		name:       "simple",
		rcconfData: `ifconfig_em0="inet 127.0.0.253 netmask 0xffffffff"` + nl,
		wantHas:    true,
	}, {
		name:       "case_insensitiveness",
		rcconfData: `ifconfig_em0="InEt 127.0.0.253 NeTmAsK 0xffffffff"` + nl,
		wantHas:    true,
	}, {
		name: "comments_and_trash",
		rcconfData: `# comment 1` + nl +
			`` + nl +
			`# comment 2` + nl +
			`ifconfig_em0="inet 127.0.0.253 netmask 0xffffffff"` + nl,
		wantHas: true,
	}, {
		name: "aliases",
		rcconfData: `ifconfig_em0_alias="inet 127.0.0.1/24"` + nl +
			`ifconfig_em0="inet 127.0.0.253 netmask 0xffffffff"` + nl,
		wantHas: true,
	}, {
		name: "incorrect_config",
		rcconfData: `ifconfig_em0="inet6 127.0.0.253 netmask 0xffffffff"` + nl +
			`ifconfig_em0="inet 127.0.0.253 net-mask 0xffffffff"` + nl +
			`ifconfig_em0="inet 256.256.256.256 netmask 0xffffffff"` + nl +
			`ifconfig_em0=""` + nl,
		wantHas: false,
	}}

	for _, tc := range testCases {
		r := strings.NewReader(tc.rcconfData)
		t.Run(tc.name, func(t *testing.T) {
			has, err := rcConfStaticConfig(r, ifaceName)
			require.NoError(t, err)

			assert.Equal(t, tc.wantHas, has)
		})
	}
}
