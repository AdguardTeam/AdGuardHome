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
	const iface interfaceName = `em0`
	const nl = "\n"

	testCases := []struct {
		name       string
		rcconfData string
		wantCont   bool
	}{{
		name:       "simple",
		rcconfData: `ifconfig_em0="inet 127.0.0.253 netmask 0xffffffff"` + nl,
		wantCont:   false,
	}, {
		name:       "case_insensitiveness",
		rcconfData: `ifconfig_em0="InEt 127.0.0.253 NeTmAsK 0xffffffff"` + nl,
		wantCont:   false,
	}, {
		name: "comments_and_trash",
		rcconfData: `# comment 1` + nl +
			`` + nl +
			`# comment 2` + nl +
			`ifconfig_em0="inet 127.0.0.253 netmask 0xffffffff"` + nl,
		wantCont: false,
	}, {
		name: "aliases",
		rcconfData: `ifconfig_em0_alias="inet 127.0.0.1/24"` + nl +
			`ifconfig_em0="inet 127.0.0.253 netmask 0xffffffff"` + nl,
		wantCont: false,
	}, {
		name: "incorrect_config",
		rcconfData: `ifconfig_em0="inet6 127.0.0.253 netmask 0xffffffff"` + nl +
			`ifconfig_em0="inet 256.256.256.256 netmask 0xffffffff"` + nl +
			`ifconfig_em0=""` + nl,
		wantCont: true,
	}}

	for _, tc := range testCases {
		r := strings.NewReader(tc.rcconfData)
		t.Run(tc.name, func(t *testing.T) {
			_, cont, err := iface.rcConfStaticConfig(r)
			require.NoError(t, err)

			assert.Equal(t, tc.wantCont, cont)
		})
	}
}
