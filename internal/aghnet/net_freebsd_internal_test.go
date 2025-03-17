//go:build freebsd

package aghnet

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIfaceHasStaticIP(t *testing.T) {
	const (
		ifaceName = `em0`
		rcConf    = "etc/rc.conf"
	)

	testCases := []struct {
		name     string
		rootFsys fs.FS
		wantHas  assert.BoolAssertionFunc
	}{{
		name: "simple",
		rootFsys: fstest.MapFS{rcConf: &fstest.MapFile{
			Data: []byte(`ifconfig_` + ifaceName + `="inet 127.0.0.253 netmask 0xffffffff"` + nl),
		}},
		wantHas: assert.True,
	}, {
		name: "case_insensitiveness",
		rootFsys: fstest.MapFS{rcConf: &fstest.MapFile{
			Data: []byte(`ifconfig_` + ifaceName + `="InEt 127.0.0.253 NeTmAsK 0xffffffff"` + nl),
		}},
		wantHas: assert.True,
	}, {
		name: "comments_and_trash",
		rootFsys: fstest.MapFS{rcConf: &fstest.MapFile{
			Data: []byte(`# comment 1` + nl +
				`` + nl +
				`# comment 2` + nl +
				`ifconfig_` + ifaceName + `="inet 127.0.0.253 netmask 0xffffffff"` + nl,
			),
		}},
		wantHas: assert.True,
	}, {
		name: "aliases",
		rootFsys: fstest.MapFS{rcConf: &fstest.MapFile{
			Data: []byte(`ifconfig_` + ifaceName + `_alias="inet 127.0.0.1/24"` + nl +
				`ifconfig_` + ifaceName + `="inet 127.0.0.253 netmask 0xffffffff"` + nl,
			),
		}},
		wantHas: assert.True,
	}, {
		name: "incorrect_config",
		rootFsys: fstest.MapFS{rcConf: &fstest.MapFile{
			Data: []byte(
				`ifconfig_` + ifaceName + `="inet6 127.0.0.253 netmask 0xffffffff"` + nl +
					`ifconfig_` + ifaceName + `="inet 256.256.256.256 netmask 0xffffffff"` + nl +
					`ifconfig_` + ifaceName + `=""` + nl,
			),
		}},
		wantHas: assert.False,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			substRootDirFS(t, tc.rootFsys)

			has, err := IfaceHasStaticIP(ifaceName)
			require.NoError(t, err)

			tc.wantHas(t, has)
		})
	}
}
