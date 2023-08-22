package aghnet_test

import (
	"net/netip"
	"path"
	"path/filepath"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghchan"
	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/hostsfile"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// nl is a newline character.
const nl = "\n"

// Variables mirroring the etc_hosts file from testdata.
var (
	addr1000 = netip.MustParseAddr("1.0.0.0")
	addr1001 = netip.MustParseAddr("1.0.0.1")
	addr1002 = netip.MustParseAddr("1.0.0.2")
	addr1003 = netip.MustParseAddr("1.0.0.3")
	addr1004 = netip.MustParseAddr("1.0.0.4")
	addr1357 = netip.MustParseAddr("1.3.5.7")
	addr4216 = netip.MustParseAddr("4.2.1.6")
	addr7531 = netip.MustParseAddr("7.5.3.1")

	addr0  = netip.MustParseAddr("::")
	addr1  = netip.MustParseAddr("::1")
	addr2  = netip.MustParseAddr("::2")
	addr3  = netip.MustParseAddr("::3")
	addr4  = netip.MustParseAddr("::4")
	addr42 = netip.MustParseAddr("::42")
	addr13 = netip.MustParseAddr("::13")
	addr31 = netip.MustParseAddr("::31")

	hostsSrc = "./" + filepath.Join("./testdata", "etc_hosts")

	testHosts = map[netip.Addr][]*hostsfile.Record{
		addr1000: {{
			Addr:   addr1000,
			Source: hostsSrc,
			Names:  []string{"hello", "hello.world"},
		}, {
			Addr:   addr1000,
			Source: hostsSrc,
			Names:  []string{"hello.world.again"},
		}, {
			Addr:   addr1000,
			Source: hostsSrc,
			Names:  []string{"hello.world"},
		}},
		addr1001: {{
			Addr:   addr1001,
			Source: hostsSrc,
			Names:  []string{"simplehost"},
		}, {
			Addr:   addr1001,
			Source: hostsSrc,
			Names:  []string{"simplehost"},
		}},
		addr1002: {{
			Addr:   addr1002,
			Source: hostsSrc,
			Names:  []string{"a.whole", "lot.of", "aliases", "for.testing"},
		}},
		addr1003: {{
			Addr:   addr1003,
			Source: hostsSrc,
			Names:  []string{"*"},
		}},
		addr1004: {{
			Addr:   addr1004,
			Source: hostsSrc,
			Names:  []string{"*.com"},
		}},
		addr1357: {{
			Addr:   addr1357,
			Source: hostsSrc,
			Names:  []string{"domain4", "domain4.alias"},
		}},
		addr7531: {{
			Addr:   addr7531,
			Source: hostsSrc,
			Names:  []string{"domain4.alias", "domain4"},
		}},
		addr4216: {{
			Addr:   addr4216,
			Source: hostsSrc,
			Names:  []string{"domain", "domain.alias"},
		}},
		addr0: {{
			Addr:   addr0,
			Source: hostsSrc,
			Names:  []string{"hello", "hello.world"},
		}, {
			Addr:   addr0,
			Source: hostsSrc,
			Names:  []string{"hello.world.again"},
		}, {
			Addr:   addr0,
			Source: hostsSrc,
			Names:  []string{"hello.world"},
		}},
		addr1: {{
			Addr:   addr1,
			Source: hostsSrc,
			Names:  []string{"simplehost"},
		}, {
			Addr:   addr1,
			Source: hostsSrc,
			Names:  []string{"simplehost"},
		}},
		addr2: {{
			Addr:   addr2,
			Source: hostsSrc,
			Names:  []string{"a.whole", "lot.of", "aliases", "for.testing"},
		}},
		addr3: {{
			Addr:   addr3,
			Source: hostsSrc,
			Names:  []string{"*"},
		}},
		addr4: {{
			Addr:   addr4,
			Source: hostsSrc,
			Names:  []string{"*.com"},
		}},
		addr42: {{
			Addr:   addr42,
			Source: hostsSrc,
			Names:  []string{"domain.alias", "domain"},
		}},
		addr13: {{
			Addr:   addr13,
			Source: hostsSrc,
			Names:  []string{"domain6", "domain6.alias"},
		}},
		addr31: {{
			Addr:   addr31,
			Source: hostsSrc,
			Names:  []string{"domain6.alias", "domain6"},
		}},
	}
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

	testFS := fstest.MapFS{"dir/file1": &fstest.MapFile{Data: []byte(ipStr + ` hostname` + nl)}}

	// event is a convenient alias for an empty struct{} to emit test events.
	type event = struct{}

	eventsCh := make(chan event, 1)
	t.Cleanup(func() { close(eventsCh) })

	w := &aghtest.FSWatcher{
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

	checkRefresh := func(t *testing.T, want aghnet.Hosts) {
		t.Helper()

		upd, ok := aghchan.MustReceive(hc.Upd(), 1*time.Second)
		require.True(t, ok)

		assert.Equal(t, want, upd)
	}

	t.Run("initial_refresh", func(t *testing.T) {
		checkRefresh(t, aghnet.Hosts{
			ip: {{
				Addr:   ip,
				Source: "file1",
				Names:  []string{"hostname"},
			}},
		})
	})

	t.Run("second_refresh", func(t *testing.T) {
		testFS["dir/file2"] = &fstest.MapFile{Data: []byte(anotherIPStr + ` alias` + nl)}
		eventsCh <- event{}

		checkRefresh(t, aghnet.Hosts{
			ip: {{
				Addr:   ip,
				Source: "file1",
				Names:  []string{"hostname"},
			}},
			anotherIP: {{
				Addr:   anotherIP,
				Source: "file2",
				Names:  []string{"alias"},
			}},
		})
	})

	t.Run("double_refresh", func(t *testing.T) {
		// Make a change once.
		testFS["dir/file1"] = &fstest.MapFile{Data: []byte(ipStr + ` alias` + nl)}
		eventsCh <- event{}

		// Require the changes are written.
		require.Eventually(t, func() bool {
			ips := hc.MatchName("hostname")

			return len(ips) == 0
		}, 5*time.Second, time.Second/2)

		// Make a change again.
		testFS["dir/file2"] = &fstest.MapFile{Data: []byte(ipStr + ` hostname` + nl)}
		eventsCh <- event{}

		// Require the changes are written.
		require.Eventually(t, func() bool {
			ips := hc.MatchName("hostname")

			return len(ips) > 0
		}, 5*time.Second, time.Second/2)

		assert.Len(t, hc.Upd(), 1)
	})
}

func TestHostsContainer_MatchName(t *testing.T) {
	require.NoError(t, fstest.TestFS(testdata, "etc_hosts"))

	stubWatcher := aghtest.FSWatcher{
		OnEvents: func() (e <-chan struct{}) { return nil },
		OnAdd:    func(name string) (err error) { return nil },
		OnClose:  func() (err error) { return nil },
	}

	testCases := []struct {
		req  string
		name string
		want []*hostsfile.Record
	}{{
		req:  "simplehost",
		name: "simple",
		want: append(testHosts[addr1001], testHosts[addr1]...),
	}, {
		req:  "hello.world",
		name: "hello_alias",
		want: []*hostsfile.Record{
			testHosts[addr1000][0],
			testHosts[addr1000][2],
			testHosts[addr0][0],
			testHosts[addr0][2],
		},
	}, {
		req:  "hello.world.again",
		name: "other_line_alias",
		want: []*hostsfile.Record{
			testHosts[addr1000][1],
			testHosts[addr0][1],
		},
	}, {
		req:  "say.hello",
		name: "hello_subdomain",
		want: nil,
	}, {
		req:  "say.hello.world",
		name: "hello_alias_subdomain",
		want: nil,
	}, {
		req:  "for.testing",
		name: "lots_of_aliases",
		want: append(testHosts[addr1002], testHosts[addr2]...),
	}, {
		req:  "nonexistent.example",
		name: "non-existing",
		want: nil,
	}, {
		req:  "domain",
		name: "issue_4216_4_6",
		want: append(testHosts[addr4216], testHosts[addr42]...),
	}, {
		req:  "domain4",
		name: "issue_4216_4",
		want: append(testHosts[addr1357], testHosts[addr7531]...),
	}, {
		req:  "domain6",
		name: "issue_4216_6",
		want: append(testHosts[addr13], testHosts[addr31]...),
	}}

	hc, err := aghnet.NewHostsContainer(testdata, &stubWatcher, "etc_hosts")
	require.NoError(t, err)
	testutil.CleanupAndRequireSuccess(t, hc.Close)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			recs := hc.MatchName(tc.req)
			assert.Equal(t, tc.want, recs)
		})
	}
}

func TestHostsContainer_MatchAddr(t *testing.T) {
	require.NoError(t, fstest.TestFS(testdata, "etc_hosts"))

	stubWatcher := aghtest.FSWatcher{
		OnEvents: func() (e <-chan struct{}) { return nil },
		OnAdd:    func(name string) (err error) { return nil },
		OnClose:  func() (err error) { return nil },
	}

	hc, err := aghnet.NewHostsContainer(testdata, &stubWatcher, "etc_hosts")
	require.NoError(t, err)
	testutil.CleanupAndRequireSuccess(t, hc.Close)

	testCases := []struct {
		req  netip.Addr
		name string
		want []*hostsfile.Record
	}{{
		req:  netip.AddrFrom4([4]byte{1, 0, 0, 1}),
		name: "reverse",
		want: testHosts[addr1001],
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			recs := hc.MatchAddr(tc.req)
			assert.Equal(t, tc.want, recs)
		})
	}
}
