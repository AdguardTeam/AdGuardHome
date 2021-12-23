package aghnet

import (
	"io/fs"
	"net"
	"os"
	"path"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"

	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/stringutil"
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

func TestHostsContainer_Refresh(t *testing.T) {
	knownIP := net.IP{127, 0, 0, 1}

	const knownHost = "localhost"
	const knownAlias = "hocallost"

	const dirname = "dir"
	const filename1 = "file1"
	const filename2 = "file2"

	p1 := path.Join(dirname, filename1)
	p2 := path.Join(dirname, filename2)

	testFS := fstest.MapFS{
		p1: &fstest.MapFile{
			Data: []byte(strings.Join([]string{knownIP.String(), knownHost}, sp) + nl),
		},
	}

	// event is a convenient alias for an empty struct{} to emit test events.
	type event = struct{}

	eventsCh := make(chan event, 1)
	t.Cleanup(func() { close(eventsCh) })

	w := &aghtest.FSWatcher{
		OnEvents: func() (e <-chan event) { return eventsCh },
		OnAdd: func(name string) (err error) {
			assert.Equal(t, dirname, name)

			return nil
		},
		OnClose: func() (err error) { panic("not implemented") },
	}

	hc, err := NewHostsContainer(0, testFS, w, dirname)
	require.NoError(t, err)

	checkRefresh := func(t *testing.T, wantHosts *stringutil.Set) {
		upd, ok := <-hc.Upd()
		require.True(t, ok)
		require.NotNil(t, upd)

		assert.Equal(t, 1, upd.Len())

		v, ok := upd.Get(knownIP)
		require.True(t, ok)

		var hosts *stringutil.Set
		hosts, ok = v.(*stringutil.Set)
		require.True(t, ok)

		assert.True(t, hosts.Equal(wantHosts))
	}

	t.Run("initial_refresh", func(t *testing.T) {
		checkRefresh(t, stringutil.NewSet(knownHost))
	})

	testFS[p2] = &fstest.MapFile{
		Data: []byte(strings.Join([]string{knownIP.String(), knownAlias}, sp) + nl),
	}
	eventsCh <- event{}

	t.Run("second_refresh", func(t *testing.T) {
		checkRefresh(t, stringutil.NewSet(knownHost, knownAlias))
	})

	eventsCh <- event{}

	t.Run("no_changes_refresh", func(t *testing.T) {
		assert.Empty(t, hc.Upd())
	})
}

func TestHostsContainer_PathsToPatterns(t *testing.T) {
	const (
		dir0 = "dir"
		dir1 = "dir_1"
		fn1  = "file_1"
		fn2  = "file_2"
		fn3  = "file_3"
		fn4  = "file_4"
	)

	fp1 := path.Join(dir0, fn1)
	fp2 := path.Join(dir0, fn2)
	fp3 := path.Join(dir0, dir1, fn3)

	gsfs := fstest.MapFS{
		fp1: &fstest.MapFile{Data: []byte{1}},
		fp2: &fstest.MapFile{Data: []byte{2}},
		fp3: &fstest.MapFile{Data: []byte{3}},
	}

	testCases := []struct {
		name    string
		wantErr error
		want    []string
		paths   []string
	}{{
		name:    "no_paths",
		wantErr: nil,
		want:    nil,
		paths:   nil,
	}, {
		name:    "single_file",
		wantErr: nil,
		want:    []string{fp1},
		paths:   []string{fp1},
	}, {
		name:    "several_files",
		wantErr: nil,
		want:    []string{fp1, fp2},
		paths:   []string{fp1, fp2},
	}, {
		name:    "whole_dir",
		wantErr: nil,
		want:    []string{path.Join(dir0, "*")},
		paths:   []string{dir0},
	}, {
		name:    "file_and_dir",
		wantErr: nil,
		want:    []string{fp1, path.Join(dir0, dir1, "*")},
		paths:   []string{fp1, path.Join(dir0, dir1)},
	}, {
		name:    "non-existing",
		wantErr: nil,
		want:    nil,
		paths:   []string{path.Join(dir0, "file_3")},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			patterns, err := pathsToPatterns(gsfs, tc.paths)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr)

				return
			}

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

func TestHostsContainer(t *testing.T) {
	const listID = 1234

	testdata := os.DirFS("./testdata")

	nRewrites := func(t *testing.T, res *urlfilter.DNSResult, n int) (rws []*rules.DNSRewrite) {
		t.Helper()

		rewrites := res.DNSRewrites()
		assert.Len(t, rewrites, n)

		for _, rewrite := range rewrites {
			require.Equal(t, listID, rewrite.FilterListID)

			rw := rewrite.DNSRewrite
			require.NotNil(t, rw)

			rws = append(rws, rw)
		}

		return rws
	}

	testCases := []struct {
		testTail func(t *testing.T, res *urlfilter.DNSResult)
		name     string
		req      urlfilter.DNSRequest
	}{{
		name: "simple",
		req: urlfilter.DNSRequest{
			Hostname: "simplehost",
			DNSType:  dns.TypeA,
		},
		testTail: func(t *testing.T, res *urlfilter.DNSResult) {
			rws := nRewrites(t, res, 2)

			v, ok := rws[0].Value.(net.IP)
			require.True(t, ok)

			assert.True(t, net.IP{1, 0, 0, 1}.Equal(v))

			v, ok = rws[1].Value.(net.IP)
			require.True(t, ok)

			// It's ::1.
			assert.True(t, net.IP(append((&[15]byte{})[:], byte(1))).Equal(v))
		},
	}, {
		name: "hello_alias",
		req: urlfilter.DNSRequest{
			Hostname: "hello.world",
			DNSType:  dns.TypeA,
		},
		testTail: func(t *testing.T, res *urlfilter.DNSResult) {
			assert.Equal(t, "hello", nRewrites(t, res, 1)[0].NewCNAME)
		},
	}, {
		name: "hello_subdomain",
		req: urlfilter.DNSRequest{
			Hostname: "say.hello",
			DNSType:  dns.TypeA,
		},
		testTail: func(t *testing.T, res *urlfilter.DNSResult) {
			assert.Empty(t, res.DNSRewrites())
		},
	}, {
		name: "hello_alias_subdomain",
		req: urlfilter.DNSRequest{
			Hostname: "say.hello.world",
			DNSType:  dns.TypeA,
		},
		testTail: func(t *testing.T, res *urlfilter.DNSResult) {
			assert.Empty(t, res.DNSRewrites())
		},
	}, {
		name: "lots_of_aliases",
		req: urlfilter.DNSRequest{
			Hostname: "for.testing",
			DNSType:  dns.TypeA,
		},
		testTail: func(t *testing.T, res *urlfilter.DNSResult) {
			assert.Equal(t, "a.whole", nRewrites(t, res, 1)[0].NewCNAME)
		},
	}, {
		name: "reverse",
		req: urlfilter.DNSRequest{
			Hostname: "1.0.0.1.in-addr.arpa",
			DNSType:  dns.TypePTR,
		},
		testTail: func(t *testing.T, res *urlfilter.DNSResult) {
			rws := nRewrites(t, res, 1)

			assert.Equal(t, dns.TypePTR, rws[0].RRType)
			assert.Equal(t, "simplehost.", rws[0].Value)
		},
	}, {
		name: "non-existing",
		req: urlfilter.DNSRequest{
			Hostname: "nonexisting",
			DNSType:  dns.TypeA,
		},
		testTail: func(t *testing.T, res *urlfilter.DNSResult) {
			require.NotNil(t, res)

			assert.Nil(t, res.DNSRewrites())
		},
	}}

	stubWatcher := aghtest.FSWatcher{
		OnEvents: func() (e <-chan struct{}) { return nil },
		OnAdd:    func(name string) (err error) { return nil },
		OnClose:  func() (err error) { panic("not implemented") },
	}

	hc, err := NewHostsContainer(listID, testdata, &stubWatcher, "etc_hosts")
	require.NoError(t, err)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res, ok := hc.MatchRequest(tc.req)
			require.False(t, ok)
			require.NotNil(t, res)

			tc.testTail(t, res)
		})
	}
}

func TestUniqueRules_ParseLine(t *testing.T) {
	const (
		hostname = "localhost"
		alias    = "hocallost"
	)

	knownIP := net.IP{127, 0, 0, 1}

	testCases := []struct {
		name      string
		line      string
		wantIP    net.IP
		wantHosts []string
	}{{
		name:      "simple",
		line:      strings.Join([]string{knownIP.String(), hostname}, sp),
		wantIP:    knownIP,
		wantHosts: []string{"localhost"},
	}, {
		name:      "aliases",
		line:      strings.Join([]string{knownIP.String(), hostname, alias}, sp),
		wantIP:    knownIP,
		wantHosts: []string{"localhost", "hocallost"},
	}, {
		name:      "invalid_line",
		line:      knownIP.String(),
		wantIP:    nil,
		wantHosts: nil,
	}, {
		name:      "invalid_line_hostname",
		line:      strings.Join([]string{knownIP.String(), "#" + hostname}, sp),
		wantIP:    knownIP,
		wantHosts: nil,
	}, {
		name:      "commented_aliases",
		line:      strings.Join([]string{knownIP.String(), hostname, "#" + alias}, sp),
		wantIP:    knownIP,
		wantHosts: []string{"localhost"},
	}, {
		name:      "whole_comment",
		line:      strings.Join([]string{"#", knownIP.String(), hostname}, sp),
		wantIP:    nil,
		wantHosts: nil,
	}, {
		name:      "partial_comment",
		line:      strings.Join([]string{knownIP.String(), hostname[:4] + "#" + hostname[4:]}, sp),
		wantIP:    knownIP,
		wantHosts: []string{hostname[:4]},
	}, {
		name:      "empty",
		line:      ``,
		wantIP:    nil,
		wantHosts: nil,
	}}

	for _, tc := range testCases {
		hp := hostsParser{}
		t.Run(tc.name, func(t *testing.T) {
			ip, hosts := hp.parseLine(tc.line)
			assert.True(t, tc.wantIP.Equal(ip))
			assert.Equal(t, tc.wantHosts, hosts)
		})
	}
}
