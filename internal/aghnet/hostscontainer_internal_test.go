package aghnet

import (
	"io/fs"
	"net/netip"
	"path"
	"testing"
	"testing/fstest"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/testutil/fakefs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const nl = "\n"

func TestHostsContainer_PathsToPatterns(t *testing.T) {
	gsfs := fstest.MapFS{
		"dir_0/file_1":       &fstest.MapFile{Data: []byte{1}},
		"dir_0/file_2":       &fstest.MapFile{Data: []byte{2}},
		"dir_0/dir_1/file_3": &fstest.MapFile{Data: []byte{3}},
	}

	testCases := []struct {
		name  string
		paths []string
		want  []string
	}{{
		name:  "no_paths",
		paths: nil,
		want:  nil,
	}, {
		name:  "single_file",
		paths: []string{"dir_0/file_1"},
		want:  []string{"dir_0/file_1"},
	}, {
		name:  "several_files",
		paths: []string{"dir_0/file_1", "dir_0/file_2"},
		want:  []string{"dir_0/file_1", "dir_0/file_2"},
	}, {
		name:  "whole_dir",
		paths: []string{"dir_0"},
		want:  []string{"dir_0/*"},
	}, {
		name:  "file_and_dir",
		paths: []string{"dir_0/file_1", "dir_0/dir_1"},
		want:  []string{"dir_0/file_1", "dir_0/dir_1/*"},
	}, {
		name:  "non-existing",
		paths: []string{path.Join("dir_0", "file_3")},
		want:  nil,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			patterns, err := pathsToPatterns(gsfs, tc.paths)
			require.NoError(t, err)

			assert.Equal(t, tc.want, patterns)
		})
	}

	t.Run("bad_file", func(t *testing.T) {
		const errStat errors.Error = "bad file"

		badFS := &fakefs.StatFS{
			OnOpen: func(_ string) (f fs.File, err error) { panic("not implemented") },
			OnStat: func(name string) (fi fs.FileInfo, err error) {
				return nil, errStat
			},
		}

		_, err := pathsToPatterns(badFS, []string{""})
		assert.ErrorIs(t, err, errStat)
	})
}

func TestUniqueRules_ParseLine(t *testing.T) {
	ip := netutil.IPv4Localhost()
	ipStr := ip.String()

	testCases := []struct {
		name      string
		line      string
		wantIP    netip.Addr
		wantHosts []string
	}{{
		name:      "simple",
		line:      ipStr + ` hostname`,
		wantIP:    ip,
		wantHosts: []string{"hostname"},
	}, {
		name:      "aliases",
		line:      ipStr + ` hostname alias`,
		wantIP:    ip,
		wantHosts: []string{"hostname", "alias"},
	}, {
		name:      "invalid_line",
		line:      ipStr,
		wantIP:    netip.Addr{},
		wantHosts: nil,
	}, {
		name:      "invalid_line_hostname",
		line:      ipStr + ` # hostname`,
		wantIP:    ip,
		wantHosts: nil,
	}, {
		name:      "commented_aliases",
		line:      ipStr + ` hostname # alias`,
		wantIP:    ip,
		wantHosts: []string{"hostname"},
	}, {
		name:      "whole_comment",
		line:      `# ` + ipStr + ` hostname`,
		wantIP:    netip.Addr{},
		wantHosts: nil,
	}, {
		name:      "partial_comment",
		line:      ipStr + ` host#name`,
		wantIP:    ip,
		wantHosts: []string{"host"},
	}, {
		name:      "empty",
		line:      ``,
		wantIP:    netip.Addr{},
		wantHosts: nil,
	}, {
		name:      "bad_hosts",
		line:      ipStr + ` bad..host bad._tld empty.tld. ok.host`,
		wantIP:    ip,
		wantHosts: []string{"ok.host"},
	}}

	for _, tc := range testCases {
		hp := hostsParser{}
		t.Run(tc.name, func(t *testing.T) {
			got, hosts := hp.parseLine(tc.line)
			assert.Equal(t, tc.wantIP, got)
			assert.Equal(t, tc.wantHosts, hosts)
		})
	}
}
