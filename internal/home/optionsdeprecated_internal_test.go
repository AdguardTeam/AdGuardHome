package home

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilterNoOpOpts(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name           string
		in             []string
		wantOpts       []string
		wantNoEtcHosts assert.BoolAssertionFunc
	}{{
		name:           "empty",
		in:             []string{},
		wantOpts:       []string{},
		wantNoEtcHosts: assert.False,
	}, {
		name:           "valid_only",
		in:             []string{"-v", "--web-addr", "localhost:8080"},
		wantOpts:       []string{"-v", "--web-addr", "localhost:8080"},
		wantNoEtcHosts: assert.False,
	}, {
		name:           "valid_only_one_dash",
		in:             []string{"-v", "-web-addr", "localhost:8080"},
		wantOpts:       []string{"-v", "-web-addr", "localhost:8080"},
		wantNoEtcHosts: assert.False,
	}, {
		name:           "port",
		in:             []string{"-p", "8080"},
		wantOpts:       []string{},
		wantNoEtcHosts: assert.False,
	}, {
		name: "invalid_only",
		in: []string{
			"--no-mem-optimization",
			"--no-etc-hosts",
			"-h", "localhost",
			"-p", "8080",
		},
		wantOpts:       []string{},
		wantNoEtcHosts: assert.True,
	}, {
		name: "invalid_only_one_dash",
		in: []string{
			"-no-mem-optimization",
			"-no-etc-hosts",
			"-host", "localhost",
			"-port", "8080",
		},
		wantOpts:       []string{},
		wantNoEtcHosts: assert.True,
	}, {
		name:           "no_etc_hosts",
		in:             []string{"--no-etc-hosts"},
		wantOpts:       []string{},
		wantNoEtcHosts: assert.True,
	}, {
		name:           "help_only",
		in:             []string{"-h"},
		wantOpts:       []string{"-h"},
		wantNoEtcHosts: assert.False,
	}, {
		name:           "help_at_end",
		in:             []string{"--verbose", "-h"},
		wantOpts:       []string{"--verbose", "-h"},
		wantNoEtcHosts: assert.False,
	}, {
		name:           "help_at_start",
		in:             []string{"-h", "--verbose"},
		wantOpts:       []string{"-h", "--verbose"},
		wantNoEtcHosts: assert.False,
	}, {
		name:           "host_only",
		in:             []string{"-h", "localhost"},
		wantOpts:       []string{},
		wantNoEtcHosts: assert.False,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Simulate work with [os.Args].
			in := append([]string{"AdGuardHome"}, tc.in...)
			gotOpts, gotNoEtcHosts := filterNoOpOpts(in, io.Discard, io.Discard)
			require.GreaterOrEqual(t, len(gotOpts), 1)

			gotOpts = gotOpts[1:]
			assert.Equal(t, tc.wantOpts, gotOpts)
			tc.wantNoEtcHosts(t, gotNoEtcHosts)
		})
	}
}
