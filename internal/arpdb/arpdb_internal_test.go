package arpdb

import (
	"context"
	"io/fs"
	"net"
	"net/netip"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/agh"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testTimeout is a common timeout for tests.
const testTimeout = 1 * time.Second

// testdata is the filesystem containing data for testing the package.
var testdata fs.FS = os.DirFS("./testdata")

// RunCmdFunc is the signature of aghos.RunCommand function.
type RunCmdFunc func(cmd string, args ...string) (code int, out []byte, err error)

func Test_New(t *testing.T) {
	var a Interface
	require.NotPanics(t, func() { a = New(slogutil.NewDiscardLogger()) })

	assert.NotNil(t, a)
}

// TODO(s.chzhen):  Consider moving mocks into aghtest.

// TestARPDB is the mock implementation of [Interface] to use in tests.
type TestARPDB struct {
	OnRefresh   func(ctx context.Context) (err error)
	OnNeighbors func() (ns []Neighbor)
}

// type check
var _ Interface = (*TestARPDB)(nil)

// Refresh implements the [Interface] interface for *TestARPDB.
func (arp *TestARPDB) Refresh(ctx context.Context) (err error) {
	return arp.OnRefresh(ctx)
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
		OnRefresh: func(_ context.Context) (err error) { succRefrCount++; return nil },
		OnNeighbors: func() (ns []Neighbor) {
			return []Neighbor{{Name: "abc", IP: knownIP, MAC: knownMAC}}
		},
	}
	failDB := &TestARPDB{
		OnRefresh: func(_ context.Context) (err error) {
			failRefrCount++

			return errors.Error("refresh failed")
		},
		OnNeighbors: func() (ns []Neighbor) { return nil },
	}

	t.Run("begin_with_success", func(t *testing.T) {
		t.Cleanup(clnp)

		a := newARPDBs(succDB, failDB)
		err := a.Refresh(testutil.ContextWithTimeout(t, testTimeout))
		require.NoError(t, err)

		assert.Equal(t, 1, succRefrCount)
		assert.Zero(t, failRefrCount)
		assert.NotEmpty(t, a.Neighbors())
	})

	t.Run("begin_with_fail", func(t *testing.T) {
		t.Cleanup(clnp)

		a := newARPDBs(failDB, succDB)
		err := a.Refresh(testutil.ContextWithTimeout(t, testTimeout))
		require.NoError(t, err)

		assert.Equal(t, 1, succRefrCount)
		assert.Equal(t, 1, failRefrCount)
		assert.NotEmpty(t, a.Neighbors())
	})

	t.Run("fail_only", func(t *testing.T) {
		t.Cleanup(clnp)

		wantMsg := "each arpdb failed: refresh failed\nrefresh failed"

		a := newARPDBs(failDB, failDB)
		err := a.Refresh(testutil.ContextWithTimeout(t, testTimeout))
		require.Error(t, err)

		testutil.AssertErrorMsg(t, wantMsg, err)

		assert.Equal(t, 2, failRefrCount)
		assert.Empty(t, a.Neighbors())
	})

	t.Run("fail_after_success", func(t *testing.T) {
		t.Cleanup(clnp)

		shouldFail := false
		unstableDB := &TestARPDB{
			OnRefresh: func(_ context.Context) (err error) {
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
		err := a.Refresh(testutil.ContextWithTimeout(t, testTimeout))
		require.NoError(t, err)

		assert.Zero(t, succRefrCount)
		assert.NotEmpty(t, a.Neighbors())

		// Unstable ARPDB should fail and the succDB should be used.
		err = a.Refresh(testutil.ContextWithTimeout(t, testTimeout))
		require.NoError(t, err)

		assert.Equal(t, 1, succRefrCount)
		assert.NotEmpty(t, a.Neighbors())

		// Unstable ARPDB should refresh successfully again.
		err = a.Refresh(testutil.ContextWithTimeout(t, testTimeout))
		require.NoError(t, err)

		assert.Equal(t, 1, succRefrCount)
		assert.NotEmpty(t, a.Neighbors())
	})

	t.Run("empty", func(t *testing.T) {
		a := newARPDBs()
		require.NoError(t, a.Refresh(testutil.ContextWithTimeout(t, testTimeout)))

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
		a.cmdCons = agh.NewCommandConstructor("cmd", 0, arpAOutput, nil)

		err := a.Refresh(testutil.ContextWithTimeout(t, testTimeout))
		require.NoError(t, err)

		assert.Equal(t, wantNeighs, a.Neighbors())
	})

	t.Run("runcmd_error", func(t *testing.T) {
		a.cmdCons = agh.NewCommandConstructor("cmd", 0, "", errors.Error("can't run"))

		err := a.Refresh(testutil.ContextWithTimeout(t, testTimeout))
		testutil.AssertErrorMsg(t, "cmd arpdb: running command: running: can't run", err)
	})

	t.Run("bad_code", func(t *testing.T) {
		a.cmdCons = agh.NewCommandConstructor("cmd", 1, "", nil)

		err := a.Refresh(testutil.ContextWithTimeout(t, testTimeout))
		testutil.AssertErrorMsg(
			t,
			"cmd arpdb: running command: unexpected exit code 1",
			err,
		)
	})

	t.Run("empty", func(t *testing.T) {
		a.cmdCons = agh.NewCommandConstructor("cmd", 0, "", nil)

		err := a.Refresh(testutil.ContextWithTimeout(t, testTimeout))
		require.NoError(t, err)

		assert.Empty(t, a.Neighbors())
	})
}

func TestEmptyARPDB(t *testing.T) {
	a := Empty{}

	t.Run("refresh", func(t *testing.T) {
		var err error
		require.NotPanics(t, func() {
			err = a.Refresh(testutil.ContextWithTimeout(t, testTimeout))
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
