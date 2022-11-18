//go:build openbsd

package aghnet

import (
	"fmt"
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIfaceHasStaticIP(t *testing.T) {
	const ifaceName = "em0"

	confFile := fmt.Sprintf("etc/hostname.%s", ifaceName)

	testCases := []struct {
		name     string
		rootFsys fs.FS
		wantHas  assert.BoolAssertionFunc
	}{{
		name: "simple",
		rootFsys: fstest.MapFS{
			confFile: &fstest.MapFile{
				Data: []byte(`inet 127.0.0.253` + nl),
			},
		},
		wantHas: assert.True,
	}, {
		name: "case_sensitiveness",
		rootFsys: fstest.MapFS{
			confFile: &fstest.MapFile{
				Data: []byte(`InEt 127.0.0.253` + nl),
			},
		},
		wantHas: assert.False,
	}, {
		name: "comments_and_trash",
		rootFsys: fstest.MapFS{
			confFile: &fstest.MapFile{
				Data: []byte(`# comment 1` + nl + nl +
					`# inet 127.0.0.253` + nl +
					`inet` + nl,
				),
			},
		},
		wantHas: assert.False,
	}, {
		name: "incorrect_config",
		rootFsys: fstest.MapFS{
			confFile: &fstest.MapFile{
				Data: []byte(`inet6 127.0.0.253` + nl + `inet 256.256.256.256` + nl),
			},
		},
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
