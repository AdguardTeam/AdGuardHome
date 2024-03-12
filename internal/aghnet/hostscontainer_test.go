package aghnet_test

import (
	"net/netip"
	"path"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/hostsfile"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHostsContainer(t *testing.T) {
	const dirname = "dir"
	const filename = "file1"

	p := path.Join(dirname, filename)

	testFS := fstest.MapFS{
		p: &fstest.MapFile{Data: []byte("127.0.0.1 localhost")},
	}

	testCases := []struct {
		wantErr error
		name    string
		paths   []string
	}{{
		wantErr: nil,
		name:    "one_file",
		paths:   []string{p},
	}, {
		wantErr: aghnet.ErrNoHostsPaths,
		name:    "no_files",
		paths:   []string{},
	}, {
		wantErr: aghnet.ErrNoHostsPaths,
		name:    "non-existent_file",
		paths:   []string{path.Join(dirname, filename+"2")},
	}, {
		wantErr: nil,
		name:    "whole_dir",
		paths:   []string{dirname},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			onAdd := func(name string) (err error) {
				assert.Contains(t, tc.paths, name)

				return nil
			}

			var eventsCalledCounter uint32
			eventsCh := make(chan struct{})
			onEvents := func() (e <-chan struct{}) {
				assert.Equal(t, uint32(1), atomic.AddUint32(&eventsCalledCounter, 1))

				return eventsCh
			}

			hc, err := aghnet.NewHostsContainer(testFS, &aghtest.FSWatcher{
				OnStart:  func() (_ error) { panic("not implemented") },
				OnEvents: onEvents,
				OnAdd:    onAdd,
				OnClose:  func() (err error) { return nil },
			}, tc.paths...)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)

				assert.Nil(t, hc)

				return
			}
			testutil.CleanupAndRequireSuccess(t, hc.Close)

			require.NoError(t, err)
			require.NotNil(t, hc)

			assert.NotNil(t, <-hc.Upd())

			eventsCh <- struct{}{}
			assert.Equal(t, uint32(1), atomic.LoadUint32(&eventsCalledCounter))
		})
	}

	t.Run("nil_fs", func(t *testing.T) {
		require.Panics(t, func() {
			_, _ = aghnet.NewHostsContainer(nil, &aghtest.FSWatcher{
				OnStart: func() (_ error) { panic("not implemented") },
				// Those shouldn't panic.
				OnEvents: func() (e <-chan struct{}) { return nil },
				OnAdd:    func(name string) (err error) { return nil },
				OnClose:  func() (err error) { return nil },
			}, p)
		})
	})

	t.Run("nil_watcher", func(t *testing.T) {
		require.Panics(t, func() {
			_, _ = aghnet.NewHostsContainer(testFS, nil, p)
		})
	})

	t.Run("err_watcher", func(t *testing.T) {
		const errOnAdd errors.Error = "error"

		errWatcher := &aghtest.FSWatcher{
			OnStart:  func() (_ error) { panic("not implemented") },
			OnEvents: func() (e <-chan struct{}) { panic("not implemented") },
			OnAdd:    func(name string) (err error) { return errOnAdd },
			OnClose:  func() (err error) { return nil },
		}

		hc, err := aghnet.NewHostsContainer(testFS, errWatcher, p)
		require.ErrorIs(t, err, errOnAdd)

		assert.Nil(t, hc)
	})
}

func TestHostsContainer_refresh(t *testing.T) {
	// TODO(e.burkov):  Test the case with no actual updates.

	ip := netutil.IPv4Localhost()
	ipStr := ip.String()

	anotherIPStr := "1.2.3.4"
	anotherIP := netip.MustParseAddr(anotherIPStr)

	r1 := &hostsfile.Record{
		Addr:   ip,
		Source: "file1",
		Names:  []string{"hostname"},
	}
	r2 := &hostsfile.Record{
		Addr:   anotherIP,
		Source: "file2",
		Names:  []string{"alias"},
	}

	r1Data, _ := r1.MarshalText()
	r2Data, _ := r2.MarshalText()

	testFS := fstest.MapFS{"dir/file1": &fstest.MapFile{Data: r1Data}}

	// event is a convenient alias for an empty struct{} to emit test events.
	type event = struct{}

	eventsCh := make(chan event, 1)
	t.Cleanup(func() { close(eventsCh) })

	w := &aghtest.FSWatcher{
		OnStart:  func() (_ error) { panic("not implemented") },
		OnEvents: func() (e <-chan event) { return eventsCh },
		OnAdd: func(name string) (err error) {
			assert.Equal(t, "dir", name)

			return nil
		},
		OnClose: func() (err error) { return nil },
	}

	hc, err := aghnet.NewHostsContainer(testFS, w, "dir")
	require.NoError(t, err)
	testutil.CleanupAndRequireSuccess(t, hc.Close)

	strg, _ := hostsfile.NewDefaultStorage()
	strg.Add(r1)

	t.Run("initial_refresh", func(t *testing.T) {
		upd, ok := testutil.RequireReceive(t, hc.Upd(), 1*time.Second)
		require.True(t, ok)

		assert.True(t, strg.Equal(upd))
	})

	strg.Add(r2)

	t.Run("second_refresh", func(t *testing.T) {
		testFS["dir/file2"] = &fstest.MapFile{Data: r2Data}
		eventsCh <- event{}

		upd, ok := testutil.RequireReceive(t, hc.Upd(), 1*time.Second)
		require.True(t, ok)

		assert.True(t, strg.Equal(upd))
	})

	t.Run("double_refresh", func(t *testing.T) {
		// Make a change once.
		testFS["dir/file1"] = &fstest.MapFile{Data: []byte(ipStr + " alias\n")}
		eventsCh <- event{}

		// Require the changes are written.
		current, ok := testutil.RequireReceive(t, hc.Upd(), 1*time.Second)
		require.True(t, ok)

		require.Empty(t, current.ByName("hostname"))

		// Make a change again.
		testFS["dir/file2"] = &fstest.MapFile{Data: []byte(ipStr + " hostname\n")}
		eventsCh <- event{}

		// Require the changes are written.
		current, ok = testutil.RequireReceive(t, hc.Upd(), 1*time.Second)
		require.True(t, ok)

		require.NotEmpty(t, current.ByName("hostname"))
	})
}
