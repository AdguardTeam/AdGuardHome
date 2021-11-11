package aghnet

import (
	"io/fs"
	"net"
	"path"
	"strings"
	"sync/atomic"
	"testing"
	"testing/fstest"

	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/urlfilter"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	nl = "\n"
	sp = " "
)

const closeCalled errors.Error = "close method called"

// fsWatcherOnCloseStub is a stub implementation of the Close method of
// aghos.FSWatcher.
func fsWatcherOnCloseStub() (err error) {
	return closeCalled
}

func TestNewHostsContainer(t *testing.T) {
	const dirname = "dir"
	const filename = "file1"

	p := path.Join(dirname, filename)

	testFS := fstest.MapFS{
		p: &fstest.MapFile{Data: []byte("127.0.0.1 localhost")},
	}

	testCases := []struct {
		name         string
		paths        []string
		wantErr      error
		wantPatterns []string
	}{{
		name:         "one_file",
		paths:        []string{p},
		wantErr:      nil,
		wantPatterns: []string{p},
	}, {
		name:         "no_files",
		paths:        []string{},
		wantErr:      errNoPaths,
		wantPatterns: nil,
	}, {
		name:         "non-existent_file",
		paths:        []string{path.Join(dirname, filename+"2")},
		wantErr:      fs.ErrNotExist,
		wantPatterns: nil,
	}, {
		name:         "whole_dir",
		paths:        []string{dirname},
		wantErr:      nil,
		wantPatterns: []string{path.Join(dirname, "*")},
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

			hc, err := NewHostsContainer(testFS, &aghtest.FSWatcher{
				OnEvents: onEvents,
				OnAdd:    onAdd,
				OnClose:  fsWatcherOnCloseStub,
			}, tc.paths...)
			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)

				assert.Nil(t, hc)

				return
			}

			require.NoError(t, err)
			t.Cleanup(func() {
				require.ErrorIs(t, hc.Close(), closeCalled)
			})

			require.NotNil(t, hc)

			assert.Equal(t, tc.wantPatterns, hc.patterns)
			assert.NotNil(t, <-hc.Upd())

			eventsCh <- struct{}{}
			assert.Equal(t, uint32(1), atomic.LoadUint32(&eventsCalledCounter))
		})
	}

	t.Run("nil_fs", func(t *testing.T) {
		require.Panics(t, func() {
			_, _ = NewHostsContainer(nil, &aghtest.FSWatcher{
				// Those shouldn't panic.
				OnEvents: func() (e <-chan struct{}) { return nil },
				OnAdd:    func(name string) (err error) { return nil },
				OnClose:  func() (err error) { return nil },
			}, p)
		})
	})

	t.Run("nil_watcher", func(t *testing.T) {
		require.Panics(t, func() {
			_, _ = NewHostsContainer(testFS, nil, p)
		})
	})

	t.Run("err_watcher", func(t *testing.T) {
		const errOnAdd errors.Error = "error"

		errWatcher := &aghtest.FSWatcher{
			OnEvents: func() (e <-chan struct{}) { panic("not implemented") },
			OnAdd:    func(name string) (err error) { return errOnAdd },
			OnClose:  func() (err error) { panic("not implemented") },
		}

		hc, err := NewHostsContainer(testFS, errWatcher, p)
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

	eventsCh := make(chan struct{}, 1)
	t.Cleanup(func() { close(eventsCh) })

	w := &aghtest.FSWatcher{
		OnEvents: func() (e <-chan struct{}) { return eventsCh },
		OnAdd: func(name string) (err error) {
			assert.Equal(t, dirname, name)

			return nil
		},
		OnClose: fsWatcherOnCloseStub,
	}

	hc, err := NewHostsContainer(testFS, w, dirname)
	require.NoError(t, err)
	t.Cleanup(func() { require.ErrorIs(t, hc.Close(), closeCalled) })

	checkRefresh := func(t *testing.T, wantHosts []string) {
		upd, ok := <-hc.Upd()
		require.True(t, ok)
		require.NotNil(t, upd)

		assert.Equal(t, 1, upd.Len())

		v, ok := upd.Get(knownIP)
		require.True(t, ok)

		var hosts []string
		hosts, ok = v.([]string)
		require.True(t, ok)
		require.Len(t, hosts, len(wantHosts))

		assert.Equal(t, wantHosts, hosts)
	}

	t.Run("initial_refresh", func(t *testing.T) {
		checkRefresh(t, []string{knownHost})
	})

	testFS[p2] = &fstest.MapFile{
		Data: []byte(strings.Join([]string{knownIP.String(), knownAlias}, sp) + nl),
	}

	eventsCh <- struct{}{}

	t.Run("second_refresh", func(t *testing.T) {
		checkRefresh(t, []string{knownHost, knownAlias})
	})
}

func TestHostsContainer_MatchRequest(t *testing.T) {
	var (
		ip4 = net.IP{127, 0, 0, 1}
		ip6 = net.IP{
			0, 0, 0, 0,
			0, 0, 0, 0,
			0, 0, 0, 0,
			0, 0, 0, 1,
		}

		hostname4  = "localhost"
		hostname6  = "localhostv6"
		hostname4a = "abcd"

		reversed4, _ = netutil.IPToReversedAddr(ip4)
		reversed6, _ = netutil.IPToReversedAddr(ip6)
	)

	const filename = "file1"

	gsfs := fstest.MapFS{
		filename: &fstest.MapFile{Data: []byte(
			strings.Join([]string{ip4.String(), hostname4, hostname4a}, sp) + nl +
				strings.Join([]string{ip6.String(), hostname6}, sp) + nl +
				strings.Join([]string{"256.256.256.256", "fakebroadcast"}, sp) + nl,
		)},
	}

	hc, err := NewHostsContainer(gsfs, &aghtest.FSWatcher{
		OnEvents: func() (e <-chan struct{}) { panic("not implemented") },
		OnAdd: func(name string) (err error) {
			assert.Equal(t, filename, name)

			return nil
		},
		OnClose: fsWatcherOnCloseStub,
	}, filename)
	require.NoError(t, err)
	t.Cleanup(func() { require.ErrorIs(t, hc.Close(), closeCalled) })

	testCase := []struct {
		name string
		want []interface{}
		req  urlfilter.DNSRequest
	}{{
		name: "a",
		want: []interface{}{ip4.To16()},
		req: urlfilter.DNSRequest{
			Hostname: hostname4,
			DNSType:  dns.TypeA,
		},
	}, {
		name: "aaaa",
		want: []interface{}{ip6},
		req: urlfilter.DNSRequest{
			Hostname: hostname6,
			DNSType:  dns.TypeAAAA,
		},
	}, {
		name: "ptr",
		want: []interface{}{
			dns.Fqdn(hostname4),
			dns.Fqdn(hostname4a),
		},
		req: urlfilter.DNSRequest{
			Hostname: reversed4,
			DNSType:  dns.TypePTR,
		},
	}, {
		name: "ptr_v6",
		want: []interface{}{dns.Fqdn(hostname6)},
		req: urlfilter.DNSRequest{
			Hostname: reversed6,
			DNSType:  dns.TypePTR,
		},
	}, {
		name: "a_alias",
		want: []interface{}{ip4.To16()},
		req: urlfilter.DNSRequest{
			Hostname: hostname4a,
			DNSType:  dns.TypeA,
		},
	}}

	for _, tc := range testCase {
		t.Run(tc.name, func(t *testing.T) {
			res, ok := hc.MatchRequest(tc.req)
			require.False(t, ok)
			require.NotNil(t, res)

			rws := res.DNSRewrites()
			require.Len(t, rws, len(tc.want))

			for i, w := range tc.want {
				require.NotNil(t, rws[i])

				rw := rws[i].DNSRewrite
				require.NotNil(t, rw)

				assert.Equal(t, w, rw.Value)
			}
		})
	}

	t.Run("cname", func(t *testing.T) {
		res, ok := hc.MatchRequest(urlfilter.DNSRequest{
			Hostname: hostname4,
			DNSType:  dns.TypeCNAME,
		})
		require.False(t, ok)

		assert.Nil(t, res)
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
		wantErr: fs.ErrNotExist,
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
}

func TestUniqueRules_AddPair(t *testing.T) {
	knownIP := net.IP{1, 2, 3, 4}

	const knownHost = "host1"

	ipToHost := netutil.NewIPMap(0)
	ipToHost.Set(knownIP, []string{knownHost})

	testCases := []struct {
		name      string
		host      string
		wantRules string
		ip        net.IP
	}{{
		name: "new_one",
		host: "host2",
		wantRules: "||host2^$dnsrewrite=NOERROR;A;1.2.3.4\n" +
			"||4.3.2.1.in-addr.arpa^$dnsrewrite=NOERROR;PTR;host2.\n",
		ip: knownIP,
	}, {
		name:      "existing_one",
		host:      knownHost,
		wantRules: "",
		ip:        knownIP,
	}, {
		name: "new_ip",
		host: knownHost,
		wantRules: "||" + knownHost + "^$dnsrewrite=NOERROR;A;1.2.3.5\n" +
			"||5.3.2.1.in-addr.arpa^$dnsrewrite=NOERROR;PTR;" + knownHost + ".\n",
		ip: net.IP{1, 2, 3, 5},
	}, {
		name:      "bad_ip",
		host:      knownHost,
		wantRules: "",
		ip:        net.IP{1, 2, 3, 4, 5},
	}}

	for _, tc := range testCases {
		hp := hostsParser{
			rules: &strings.Builder{},
			table: ipToHost.ShallowClone(),
		}

		t.Run(tc.name, func(t *testing.T) {
			hp.addPair(tc.ip, tc.host)
			assert.Equal(t, tc.wantRules, hp.rules.String())
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
