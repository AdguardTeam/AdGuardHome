package aghnet

import (
	"io/fs"
	"net"
	"path"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/stringutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/AdguardTeam/urlfilter"
	"github.com/AdguardTeam/urlfilter/rules"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	nl = "\n"
	sp = " "
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
		wantErr: ErrNoHostsPaths,
		name:    "no_files",
		paths:   []string{},
	}, {
		wantErr: ErrNoHostsPaths,
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

			hc, err := NewHostsContainer(0, testFS, &aghtest.FSWatcher{
				OnEvents: onEvents,
				OnAdd:    onAdd,
				OnClose:  func() (err error) { panic("not implemented") },
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
			_, _ = NewHostsContainer(0, nil, &aghtest.FSWatcher{
				// Those shouldn't panic.
				OnEvents: func() (e <-chan struct{}) { return nil },
				OnAdd:    func(name string) (err error) { return nil },
				OnClose:  func() (err error) { return nil },
			}, p)
		})
	})

	t.Run("nil_watcher", func(t *testing.T) {
		require.Panics(t, func() {
			_, _ = NewHostsContainer(0, testFS, nil, p)
		})
	})

	t.Run("err_watcher", func(t *testing.T) {
		const errOnAdd errors.Error = "error"

		errWatcher := &aghtest.FSWatcher{
			OnEvents: func() (e <-chan struct{}) { panic("not implemented") },
			OnAdd:    func(name string) (err error) { return errOnAdd },
			OnClose:  func() (err error) { panic("not implemented") },
		}

		hc, err := NewHostsContainer(0, testFS, errWatcher, p)
		require.ErrorIs(t, err, errOnAdd)

		assert.Nil(t, hc)
	})
}

func TestHostsContainer_refresh(t *testing.T) {
	// TODO(e.burkov):  Test the case with no actual updates.

	ip := net.IP{127, 0, 0, 1}
	ipStr := ip.String()

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
		OnClose: func() (err error) { panic("not implemented") },
	}

	hc, err := NewHostsContainer(0, testFS, w, "dir")
	require.NoError(t, err)
	testutil.CleanupAndRequireSuccess(t, hc.Close)

	checkRefresh := func(t *testing.T, wantHosts *stringutil.Set) {
		upd, ok := <-hc.Upd()
		require.True(t, ok)
		require.NotNil(t, upd)

		assert.Equal(t, 1, upd.Len())

		v, ok := upd.Get(ip)
		require.True(t, ok)

		var set *stringutil.Set
		set, ok = v.(*stringutil.Set)
		require.True(t, ok)

		assert.True(t, set.Equal(wantHosts))
	}

	t.Run("initial_refresh", func(t *testing.T) {
		checkRefresh(t, stringutil.NewSet("hostname"))
	})

	t.Run("second_refresh", func(t *testing.T) {
		testFS["dir/file2"] = &fstest.MapFile{Data: []byte(ipStr + ` alias` + nl)}
		eventsCh <- event{}
		checkRefresh(t, stringutil.NewSet("hostname", "alias"))
	})

	t.Run("double_refresh", func(t *testing.T) {
		// Make a change once.
		testFS["dir/file1"] = &fstest.MapFile{Data: []byte(ipStr + ` alias` + nl)}
		eventsCh <- event{}

		// Require the changes are written.
		require.Eventually(t, func() bool {
			res, ok := hc.MatchRequest(&urlfilter.DNSRequest{
				Hostname: "hostname",
				DNSType:  dns.TypeA,
			})

			return !ok && res.DNSRewrites() == nil
		}, 5*time.Second, time.Second/2)

		// Make a change again.
		testFS["dir/file2"] = &fstest.MapFile{Data: []byte(ipStr + ` hostname` + nl)}
		eventsCh <- event{}

		// Require the changes are written.
		require.Eventually(t, func() bool {
			res, ok := hc.MatchRequest(&urlfilter.DNSRequest{
				Hostname: "hostname",
				DNSType:  dns.TypeA,
			})

			return !ok && res.DNSRewrites() != nil
		}, 5*time.Second, time.Second/2)

		assert.Len(t, hc.Upd(), 1)
	})
}

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

		badFS := &aghtest.StatFS{
			OnStat: func(name string) (fs.FileInfo, error) {
				return nil, errStat
			},
		}

		_, err := pathsToPatterns(badFS, []string{""})
		assert.ErrorIs(t, err, errStat)
	})
}

func TestHostsContainer_Translate(t *testing.T) {
	stubWatcher := aghtest.FSWatcher{
		OnEvents: func() (e <-chan struct{}) { return nil },
		OnAdd:    func(name string) (err error) { return nil },
		OnClose:  func() (err error) { panic("not implemented") },
	}

	require.NoError(t, fstest.TestFS(testdata, "etc_hosts"))

	hc, err := NewHostsContainer(0, testdata, &stubWatcher, "etc_hosts")
	require.NoError(t, err)
	testutil.CleanupAndRequireSuccess(t, hc.Close)

	testCases := []struct {
		name      string
		rule      string
		wantTrans []string
	}{{
		name:      "simplehost",
		rule:      "|simplehost^$dnsrewrite=NOERROR;A;1.0.0.1",
		wantTrans: []string{"1.0.0.1", "simplehost"},
	}, {
		name:      "hello",
		rule:      "|hello^$dnsrewrite=NOERROR;A;1.0.0.0",
		wantTrans: []string{"1.0.0.0", "hello", "hello.world"},
	}, {
		name:      "hello-alias",
		rule:      "|hello.world.again^$dnsrewrite=NOERROR;A;1.0.0.0",
		wantTrans: []string{"1.0.0.0", "hello.world.again"},
	}, {
		name:      "simplehost_v6",
		rule:      "|simplehost^$dnsrewrite=NOERROR;AAAA;::1",
		wantTrans: []string{"::1", "simplehost"},
	}, {
		name:      "hello_v6",
		rule:      "|hello^$dnsrewrite=NOERROR;AAAA;::",
		wantTrans: []string{"::", "hello", "hello.world"},
	}, {
		name:      "hello_v6-alias",
		rule:      "|hello.world.again^$dnsrewrite=NOERROR;AAAA;::",
		wantTrans: []string{"::", "hello.world.again"},
	}, {
		name:      "simplehost_ptr",
		rule:      "|1.0.0.1.in-addr.arpa^$dnsrewrite=NOERROR;PTR;simplehost.",
		wantTrans: []string{"1.0.0.1", "simplehost"},
	}, {
		name:      "hello_ptr",
		rule:      "|0.0.0.1.in-addr.arpa^$dnsrewrite=NOERROR;PTR;hello.",
		wantTrans: []string{"1.0.0.0", "hello", "hello.world"},
	}, {
		name:      "hello_ptr-alias",
		rule:      "|0.0.0.1.in-addr.arpa^$dnsrewrite=NOERROR;PTR;hello.world.again.",
		wantTrans: []string{"1.0.0.0", "hello.world.again"},
	}, {
		name: "simplehost_ptr_v6",
		rule: "|1.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.ip6.arpa" +
			"^$dnsrewrite=NOERROR;PTR;simplehost.",
		wantTrans: []string{"::1", "simplehost"},
	}, {
		name: "hello_ptr_v6",
		rule: "|0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.ip6.arpa" +
			"^$dnsrewrite=NOERROR;PTR;hello.",
		wantTrans: []string{"::", "hello", "hello.world"},
	}, {
		name: "hello_ptr_v6-alias",
		rule: "|0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.0.ip6.arpa" +
			"^$dnsrewrite=NOERROR;PTR;hello.world.again.",
		wantTrans: []string{"::", "hello.world.again"},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := stringutil.NewSet(strings.Fields(hc.Translate(tc.rule))...)
			assert.True(t, stringutil.NewSet(tc.wantTrans...).Equal(got))
		})
	}
}

func TestHostsContainer(t *testing.T) {
	const listID = 1234

	require.NoError(t, fstest.TestFS(testdata, "etc_hosts"))

	testCases := []struct {
		want []*rules.DNSRewrite
		name string
		req  *urlfilter.DNSRequest
	}{{
		want: []*rules.DNSRewrite{{
			RCode:  dns.RcodeSuccess,
			Value:  net.IPv4(1, 0, 0, 1),
			RRType: dns.TypeA,
		}, {
			RCode:  dns.RcodeSuccess,
			Value:  net.ParseIP("::1"),
			RRType: dns.TypeAAAA,
		}},
		name: "simple",
		req: &urlfilter.DNSRequest{
			Hostname: "simplehost",
			DNSType:  dns.TypeA,
		},
	}, {
		want: []*rules.DNSRewrite{{
			RCode:  dns.RcodeSuccess,
			Value:  net.IPv4(1, 0, 0, 0),
			RRType: dns.TypeA,
		}, {
			RCode:  dns.RcodeSuccess,
			Value:  net.ParseIP("::"),
			RRType: dns.TypeAAAA,
		}},
		name: "hello_alias",
		req: &urlfilter.DNSRequest{
			Hostname: "hello.world",
			DNSType:  dns.TypeA,
		},
	}, {
		want: []*rules.DNSRewrite{{
			RCode:  dns.RcodeSuccess,
			Value:  net.IPv4(1, 0, 0, 0),
			RRType: dns.TypeA,
		}, {
			RCode:  dns.RcodeSuccess,
			Value:  net.ParseIP("::"),
			RRType: dns.TypeAAAA,
		}},
		name: "other_line_alias",
		req: &urlfilter.DNSRequest{
			Hostname: "hello.world.again",
			DNSType:  dns.TypeA,
		},
	}, {
		want: []*rules.DNSRewrite{},
		name: "hello_subdomain",
		req: &urlfilter.DNSRequest{
			Hostname: "say.hello",
			DNSType:  dns.TypeA,
		},
	}, {
		want: []*rules.DNSRewrite{},
		name: "hello_alias_subdomain",
		req: &urlfilter.DNSRequest{
			Hostname: "say.hello.world",
			DNSType:  dns.TypeA,
		},
	}, {
		want: []*rules.DNSRewrite{{
			RCode:  dns.RcodeSuccess,
			RRType: dns.TypeA,
			Value:  net.IPv4(1, 0, 0, 2),
		}, {
			RCode:  dns.RcodeSuccess,
			RRType: dns.TypeAAAA,
			Value:  net.ParseIP("::2"),
		}},
		name: "lots_of_aliases",
		req: &urlfilter.DNSRequest{
			Hostname: "for.testing",
			DNSType:  dns.TypeA,
		},
	}, {
		want: []*rules.DNSRewrite{{
			RCode:  dns.RcodeSuccess,
			RRType: dns.TypePTR,
			Value:  "simplehost.",
		}},
		name: "reverse",
		req: &urlfilter.DNSRequest{
			Hostname: "1.0.0.1.in-addr.arpa",
			DNSType:  dns.TypePTR,
		},
	}, {
		want: []*rules.DNSRewrite{},
		name: "non-existing",
		req: &urlfilter.DNSRequest{
			Hostname: "nonexisting",
			DNSType:  dns.TypeA,
		},
	}, {
		want: nil,
		name: "bad_type",
		req: &urlfilter.DNSRequest{
			Hostname: "1.0.0.1.in-addr.arpa",
			DNSType:  dns.TypeSRV,
		},
	}, {
		want: []*rules.DNSRewrite{{
			RCode:  dns.RcodeSuccess,
			RRType: dns.TypeA,
			Value:  net.IPv4(4, 2, 1, 6),
		}, {
			RCode:  dns.RcodeSuccess,
			RRType: dns.TypeAAAA,
			Value:  net.ParseIP("::42"),
		}},
		name: "issue_4216_4_6",
		req: &urlfilter.DNSRequest{
			Hostname: "domain",
			DNSType:  dns.TypeA,
		},
	}, {
		want: []*rules.DNSRewrite{{
			RCode:  dns.RcodeSuccess,
			RRType: dns.TypeA,
			Value:  net.IPv4(7, 5, 3, 1),
		}, {
			RCode:  dns.RcodeSuccess,
			RRType: dns.TypeA,
			Value:  net.IPv4(1, 3, 5, 7),
		}},
		name: "issue_4216_4",
		req: &urlfilter.DNSRequest{
			Hostname: "domain4",
			DNSType:  dns.TypeA,
		},
	}, {
		want: []*rules.DNSRewrite{{
			RCode:  dns.RcodeSuccess,
			RRType: dns.TypeAAAA,
			Value:  net.ParseIP("::13"),
		}, {
			RCode:  dns.RcodeSuccess,
			RRType: dns.TypeAAAA,
			Value:  net.ParseIP("::31"),
		}},
		name: "issue_4216_6",
		req: &urlfilter.DNSRequest{
			Hostname: "domain6",
			DNSType:  dns.TypeAAAA,
		},
	}}

	stubWatcher := aghtest.FSWatcher{
		OnEvents: func() (e <-chan struct{}) { return nil },
		OnAdd:    func(name string) (err error) { return nil },
		OnClose:  func() (err error) { panic("not implemented") },
	}

	hc, err := NewHostsContainer(listID, testdata, &stubWatcher, "etc_hosts")
	require.NoError(t, err)
	testutil.CleanupAndRequireSuccess(t, hc.Close)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res, ok := hc.MatchRequest(tc.req)
			require.False(t, ok)

			if tc.want == nil {
				assert.Nil(t, res)

				return
			}

			require.NotNil(t, res)

			rewrites := res.DNSRewrites()
			require.Len(t, rewrites, len(tc.want))

			for i, rewrite := range rewrites {
				require.Equal(t, listID, rewrite.FilterListID)

				rw := rewrite.DNSRewrite
				require.NotNil(t, rw)

				assert.Equal(t, tc.want[i], rw)
			}
		})
	}
}

func TestUniqueRules_ParseLine(t *testing.T) {
	ip := net.IP{127, 0, 0, 1}
	ipStr := ip.String()

	testCases := []struct {
		name      string
		line      string
		wantIP    net.IP
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
		wantIP:    nil,
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
		wantIP:    nil,
		wantHosts: nil,
	}, {
		name:      "partial_comment",
		line:      ipStr + ` host#name`,
		wantIP:    ip,
		wantHosts: []string{"host"},
	}, {
		name:      "empty",
		line:      ``,
		wantIP:    nil,
		wantHosts: nil,
	}}

	for _, tc := range testCases {
		hp := hostsParser{}
		t.Run(tc.name, func(t *testing.T) {
			got, hosts := hp.parseLine(tc.line)
			assert.True(t, tc.wantIP.Equal(got))
			assert.Equal(t, tc.wantHosts, hosts)
		})
	}
}
