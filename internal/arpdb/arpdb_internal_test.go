package arpdb

import (
	"fmt"
	"io/fs"
	"net"
	"net/netip"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testdata is the filesystem containing data for testing the package.
var testdata fs.FS = os.DirFS("./testdata")

// RunCmdFunc is the signature of aghos.RunCommand function.
type RunCmdFunc func(cmd string, args ...string) (code int, out []byte, err error)

// substShell replaces the the aghos.RunCommand function used throughout the
// package with rc for tests ran under t.
func substShell(t testing.TB, rc RunCmdFunc) {
	t.Helper()

	prev := aghosRunCommand
	t.Cleanup(func() { aghosRunCommand = prev })
	aghosRunCommand = rc
}

// mapShell is a substitution of aghos.RunCommand that maps the command to it's
// execution result.  It's only needed to simplify testing.
//
// TODO(e.burkov):  Perhaps put all the shell interactions behind an interface.
type mapShell map[string]struct {
	err  error
	out  string
	code int
}

// theOnlyCmd returns mapShell that only handles a single command and arguments
// combination from cmd.
func theOnlyCmd(cmd string, code int, out string, err error) (s mapShell) {
	return mapShell{cmd: {code: code, out: out, err: err}}
}

// RunCmd is a RunCmdFunc handled by s.
func (s mapShell) RunCmd(cmd string, args ...string) (code int, out []byte, err error) {
	key := strings.Join(append([]string{cmd}, args...), " ")
	ret, ok := s[key]
	if !ok {
		return 0, nil, fmt.Errorf("unexpected shell command %q", key)
	}

	return ret.code, []byte(ret.out), ret.err
}

func Test_New(t *testing.T) {
	var a Interface
	require.NotPanics(t, func() { a = New(slogutil.NewDiscardLogger()) })

	assert.NotNil(t, a)
}

// TODO(s.chzhen):  Consider moving mocks into aghtest.

// TestARPDB is the mock implementation of [Interface] to use in tests.
type TestARPDB struct {
	OnRefresh   func() (err error)
	OnNeighbors func() (ns []Neighbor)
}

// type check
var _ Interface = (*TestARPDB)(nil)

// Refresh implements the [Interface] interface for *TestARPDB.
func (arp *TestARPDB) Refresh() (err error) {
	return arp.OnRefresh()
}

// Neighbors implements the [Interface] interface for *TestARPDB.
func (arp *TestARPDB) Neighbors() (ns []Neighbor) {
	return arp.OnNeighbors()
}

func Test_NewARPDBs(t *testing.T) {
	knownIP := netip.MustParseAddr("1.2.3.4")
	knownMAC := net.HardwareAddr{0xAB, 0xCD, 0xEF, 0xAB, 0xCD, 0xEF}

	succRefrCount, failRefrCount := 0, 0
	clnp := func() {
		succRefrCount, failRefrCount = 0, 0
	}

	succDB := &TestARPDB{
		OnRefresh: func() (err error) { succRefrCount++; return nil },
		OnNeighbors: func() (ns []Neighbor) {
			return []Neighbor{{Name: "abc", IP: knownIP, MAC: knownMAC}}
		},
	}
	failDB := &TestARPDB{
		OnRefresh:   func() (err error) { failRefrCount++; return errors.Error("refresh failed") },
		OnNeighbors: func() (ns []Neighbor) { return nil },
	}

	t.Run("begin_with_success", func(t *testing.T) {
		t.Cleanup(clnp)

		a := newARPDBs(succDB, failDB)
		err := a.Refresh()
		require.NoError(t, err)

		assert.Equal(t, 1, succRefrCount)
		assert.Zero(t, failRefrCount)
		assert.NotEmpty(t, a.Neighbors())
	})

	t.Run("begin_with_fail", func(t *testing.T) {
		t.Cleanup(clnp)

		a := newARPDBs(failDB, succDB)
		err := a.Refresh()
		require.NoError(t, err)

		assert.Equal(t, 1, succRefrCount)
		assert.Equal(t, 1, failRefrCount)
		assert.NotEmpty(t, a.Neighbors())
	})

	t.Run("fail_only", func(t *testing.T) {
		t.Cleanup(clnp)

		wantMsg := "each arpdb failed: refresh failed\nrefresh failed"

		a := newARPDBs(failDB, failDB)
		err := a.Refresh()
		require.Error(t, err)

		testutil.AssertErrorMsg(t, wantMsg, err)

		assert.Equal(t, 2, failRefrCount)
		assert.Empty(t, a.Neighbors())
	})

	t.Run("fail_after_success", func(t *testing.T) {
		t.Cleanup(clnp)

		shouldFail := false
		unstableDB := &TestARPDB{
			OnRefresh: func() (err error) {
				if shouldFail {
					err = errors.Error("unstable failed")
				}
				shouldFail = !shouldFail

				return err
			},
			OnNeighbors: func() (ns []Neighbor) {
				if !shouldFail {
					return failDB.OnNeighbors()
				}

				return succDB.OnNeighbors()
			},
		}
		a := newARPDBs(unstableDB, succDB)

		// Unstable ARPDB should refresh successfully.
		err := a.Refresh()
		require.NoError(t, err)

		assert.Zero(t, succRefrCount)
		assert.NotEmpty(t, a.Neighbors())

		// Unstable ARPDB should fail and the succDB should be used.
		err = a.Refresh()
		require.NoError(t, err)

		assert.Equal(t, 1, succRefrCount)
		assert.NotEmpty(t, a.Neighbors())

		// Unstable ARPDB should refresh successfully again.
		err = a.Refresh()
		require.NoError(t, err)

		assert.Equal(t, 1, succRefrCount)
		assert.NotEmpty(t, a.Neighbors())
	})

	t.Run("empty", func(t *testing.T) {
		a := newARPDBs()
		require.NoError(t, a.Refresh())

		assert.Empty(t, a.Neighbors())
	})
}

func TestCmdARPDB_arpa(t *testing.T) {
	a := &cmdARPDB{
		logger: slogutil.NewDiscardLogger(),
		cmd:    "cmd",
		parse:  parseArpA,
		ns: &neighs{
			mu: &sync.RWMutex{},
			ns: make([]Neighbor, 0),
		},
	}

	t.Run("arp_a", func(t *testing.T) {
		sh := theOnlyCmd("cmd", 0, arpAOutput, nil)
		substShell(t, sh.RunCmd)

		err := a.Refresh()
		require.NoError(t, err)

		assert.Equal(t, wantNeighs, a.Neighbors())
	})

	t.Run("runcmd_error", func(t *testing.T) {
		sh := theOnlyCmd("cmd", 0, "", errors.Error("can't run"))
		substShell(t, sh.RunCmd)

		err := a.Refresh()
		testutil.AssertErrorMsg(t, "cmd arpdb: running command: can't run", err)
	})

	t.Run("bad_code", func(t *testing.T) {
		sh := theOnlyCmd("cmd", 1, "", nil)
		substShell(t, sh.RunCmd)

		err := a.Refresh()
		testutil.AssertErrorMsg(t, "cmd arpdb: running command: unexpected exit code 1", err)
	})

	t.Run("empty", func(t *testing.T) {
		sh := theOnlyCmd("cmd", 0, "", nil)
		substShell(t, sh.RunCmd)

		err := a.Refresh()
		require.NoError(t, err)

		assert.Empty(t, a.Neighbors())
	})
}

func TestEmptyARPDB(t *testing.T) {
	a := Empty{}

	t.Run("refresh", func(t *testing.T) {
		var err error
		require.NotPanics(t, func() {
			err = a.Refresh()
		})

		assert.NoError(t, err)
	})

	t.Run("neighbors", func(t *testing.T) {
		var ns []Neighbor
		require.NotPanics(t, func() {
			ns = a.Neighbors()
		})

		assert.Empty(t, ns)
	})
}
