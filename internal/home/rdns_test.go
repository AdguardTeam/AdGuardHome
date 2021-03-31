package home

import (
	"bytes"
	"encoding/binary"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/AdGuardHome/internal/dnsforward"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/cache"
	"github.com/AdguardTeam/golibs/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRDNS_Begin(t *testing.T) {
	aghtest.ReplaceLogLevel(t, log.DEBUG)
	w := &bytes.Buffer{}
	aghtest.ReplaceLogWriter(t, w)

	ip1234, ip1235 := net.IP{1, 2, 3, 4}, net.IP{1, 2, 3, 5}

	testCases := []struct {
		cliIDIndex    map[string]*Client
		customChan    chan net.IP
		name          string
		wantLog       string
		req           net.IP
		wantCacheHit  int
		wantCacheMiss int
	}{{
		cliIDIndex:    map[string]*Client{},
		customChan:    nil,
		name:          "cached",
		wantLog:       "",
		req:           ip1234,
		wantCacheHit:  1,
		wantCacheMiss: 0,
	}, {
		cliIDIndex:    map[string]*Client{},
		customChan:    nil,
		name:          "not_cached",
		wantLog:       "rdns: queue is full",
		req:           ip1235,
		wantCacheHit:  0,
		wantCacheMiss: 1,
	}, {
		cliIDIndex:    map[string]*Client{"1.2.3.5": {}},
		customChan:    nil,
		name:          "already_in_clients",
		wantLog:       "",
		req:           ip1235,
		wantCacheHit:  0,
		wantCacheMiss: 1,
	}, {
		cliIDIndex:    map[string]*Client{},
		customChan:    make(chan net.IP, 1),
		name:          "add_to_queue",
		wantLog:       `rdns: "1.2.3.5" added to queue`,
		req:           ip1235,
		wantCacheHit:  0,
		wantCacheMiss: 1,
	}}

	for _, tc := range testCases {
		w.Reset()

		ipCache := cache.New(cache.Config{
			EnableLRU: true,
			MaxCount:  defaultRDNSCacheSize,
		})
		ttl := make([]byte, binary.Size(uint64(0)))
		binary.BigEndian.PutUint64(ttl, uint64(time.Now().Add(100*time.Hour).Unix()))

		rdns := &RDNS{
			ipCache: ipCache,
			clients: &clientsContainer{
				list:    map[string]*Client{},
				idIndex: tc.cliIDIndex,
				ipHost:  map[string]*ClientHost{},
				allTags: map[string]bool{},
			},
		}
		ipCache.Clear()
		ipCache.Set(net.IP{1, 2, 3, 4}, ttl)

		if tc.customChan != nil {
			rdns.ipCh = tc.customChan
			defer close(tc.customChan)
		}

		t.Run(tc.name, func(t *testing.T) {
			rdns.Begin(tc.req)
			assert.Equal(t, tc.wantCacheHit, ipCache.Stats().Hit)
			assert.Equal(t, tc.wantCacheMiss, ipCache.Stats().Miss)
			assert.Contains(t, w.String(), tc.wantLog)
		})
	}
}

func TestRDNS_Resolve(t *testing.T) {
	extUpstream := &aghtest.TestUpstream{
		Reverse: map[string][]string{
			"1.1.1.1.in-addr.arpa.": {"one.one.one.one"},
		},
	}
	locUpstream := &aghtest.TestUpstream{
		Reverse: map[string][]string{
			"1.1.168.192.in-addr.arpa.": {"local.domain"},
			"2.1.168.192.in-addr.arpa.": {},
		},
	}
	upstreamErr := errors.New("upstream error")
	errUpstream := &aghtest.TestErrUpstream{
		Err: upstreamErr,
	}
	nonPtrUpstream := &aghtest.TestBlockUpstream{
		Hostname: "some-host",
		Block:    true,
	}

	dns := dnsforward.NewCustomServer(&proxy.Proxy{
		Config: proxy.Config{
			UpstreamConfig: &proxy.UpstreamConfig{
				Upstreams: []upstream.Upstream{extUpstream},
			},
		},
	})

	cc := &clientsContainer{}

	snd, err := aghnet.NewSubnetDetector()
	require.NoError(t, err)

	localIP := net.IP{192, 168, 1, 1}
	testCases := []struct {
		name        string
		want        string
		wantErr     error
		locUpstream upstream.Upstream
		req         net.IP
	}{{
		name:        "external_good",
		want:        "one.one.one.one",
		wantErr:     nil,
		locUpstream: nil,
		req:         net.IP{1, 1, 1, 1},
	}, {
		name:        "local_good",
		want:        "local.domain",
		wantErr:     nil,
		locUpstream: locUpstream,
		req:         localIP,
	}, {
		name:        "upstream_error",
		want:        "",
		wantErr:     upstreamErr,
		locUpstream: errUpstream,
		req:         localIP,
	}, {
		name:        "empty_answer_error",
		want:        "",
		wantErr:     rDNSEmptyAnswerErr,
		locUpstream: locUpstream,
		req:         net.IP{192, 168, 1, 2},
	}, {
		name:        "not_ptr_error",
		want:        "",
		wantErr:     rDNSNotPTRErr,
		locUpstream: nonPtrUpstream,
		req:         localIP,
	}}

	for _, tc := range testCases {
		rdns := NewRDNS(dns, cc, snd, &aghtest.Exchanger{
			Ups: tc.locUpstream,
		})

		t.Run(tc.name, func(t *testing.T) {
			r, rerr := rdns.resolve(tc.req)
			require.ErrorIs(t, rerr, tc.wantErr)
			assert.Equal(t, tc.want, r)
		})
	}
}

func TestRDNS_WorkerLoop(t *testing.T) {
	aghtest.ReplaceLogLevel(t, log.DEBUG)
	w := &bytes.Buffer{}
	aghtest.ReplaceLogWriter(t, w)

	locUpstream := &aghtest.TestUpstream{
		Reverse: map[string][]string{
			"1.1.168.192.in-addr.arpa.": {"local.domain"},
		},
	}

	snd, err := aghnet.NewSubnetDetector()
	require.NoError(t, err)

	testCases := []struct {
		wantLog string
		name    string
		cliIP   net.IP
	}{{
		wantLog: "",
		name:    "all_good",
		cliIP:   net.IP{192, 168, 1, 1},
	}, {
		wantLog: `rdns: resolving "192.168.1.2": lookup for "2.1.168.192.in-addr.arpa.": ` +
			string(rDNSEmptyAnswerErr),
		name:  "resolve_error",
		cliIP: net.IP{192, 168, 1, 2},
	}}

	for _, tc := range testCases {
		w.Reset()

		lr := &aghtest.Exchanger{
			Ups: locUpstream,
		}
		cc := &clientsContainer{
			list:    map[string]*Client{},
			idIndex: map[string]*Client{},
			ipHost:  map[string]*ClientHost{},
			allTags: map[string]bool{},
		}
		ch := make(chan net.IP)
		rdns := &RDNS{
			dnsServer:      nil,
			clients:        cc,
			subnetDetector: snd,
			localResolvers: lr,
			ipCh:           ch,
		}

		t.Run(tc.name, func(t *testing.T) {
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				rdns.workerLoop()
				wg.Done()
			}()

			ch <- tc.cliIP
			close(ch)
			wg.Wait()

			if tc.wantLog != "" {
				assert.Contains(t, w.String(), tc.wantLog)

				return
			}

			assert.True(t, cc.Exists(tc.cliIP.String(), ClientSourceRDNS))
		})
	}
}
