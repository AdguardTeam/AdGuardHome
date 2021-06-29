package home

import (
	"bytes"
	"encoding/binary"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/aghstrings"
	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/AdguardTeam/golibs/cache"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/miekg/dns"
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
			ipCache:   ipCache,
			exchanger: &rDNSExchanger{},
			clients: &clientsContainer{
				list:    map[string]*Client{},
				idIndex: tc.cliIDIndex,
				ipToRC:  aghnet.NewIPMap(0),
				allTags: aghstrings.NewSet(),
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

// rDNSExchanger is a mock dnsforward.RDNSExchanger implementation for tests.
type rDNSExchanger struct {
	ex         aghtest.Exchanger
	usePrivate bool
}

// Exchange implements dnsforward.RDNSExchanger interface for *RDNSExchanger.
func (e *rDNSExchanger) Exchange(ip net.IP) (host string, err error) {
	req := &dns.Msg{
		Question: []dns.Question{{
			Name:  ip.String(),
			Qtype: dns.TypePTR,
		}},
	}

	resp, err := e.ex.Exchange(req)
	if err != nil {
		return "", err
	}

	if len(resp.Answer) == 0 {
		return "", nil
	}

	return resp.Answer[0].Header().Name, nil
}

// Exchange implements dnsforward.RDNSExchanger interface for *RDNSExchanger.
func (e *rDNSExchanger) ResolvesPrivatePTR() (ok bool) {
	return e.usePrivate
}

func TestRDNS_ensurePrivateCache(t *testing.T) {
	data := []byte{1, 2, 3, 4}

	ipCache := cache.New(cache.Config{
		EnableLRU: true,
		MaxCount:  defaultRDNSCacheSize,
	})

	ex := &rDNSExchanger{}

	rdns := &RDNS{
		ipCache:   ipCache,
		exchanger: ex,
	}

	rdns.ipCache.Set(data, data)
	require.NotZero(t, rdns.ipCache.Stats().Count)

	ex.usePrivate = !ex.usePrivate

	rdns.ensurePrivateCache()
	require.Zero(t, rdns.ipCache.Stats().Count)
}

func TestRDNS_WorkerLoop(t *testing.T) {
	aghtest.ReplaceLogLevel(t, log.DEBUG)
	w := &bytes.Buffer{}
	aghtest.ReplaceLogWriter(t, w)

	locUpstream := &aghtest.TestUpstream{
		Reverse: map[string][]string{
			"192.168.1.1":            {"local.domain"},
			"2a00:1450:400c:c06::93": {"ipv6.domain"},
		},
	}
	errUpstream := &aghtest.TestErrUpstream{
		Err: errors.Error("1234"),
	}

	testCases := []struct {
		ups     upstream.Upstream
		wantLog string
		name    string
		cliIP   net.IP
	}{{
		ups:     locUpstream,
		wantLog: "",
		name:    "all_good",
		cliIP:   net.IP{192, 168, 1, 1},
	}, {
		ups:     errUpstream,
		wantLog: `rdns: resolving "192.168.1.2": errupstream: 1234`,
		name:    "resolve_error",
		cliIP:   net.IP{192, 168, 1, 2},
	}, {
		ups:     locUpstream,
		wantLog: "",
		name:    "ipv6_good",
		cliIP:   net.ParseIP("2a00:1450:400c:c06::93"),
	}}

	for _, tc := range testCases {
		w.Reset()

		cc := &clientsContainer{
			list:    map[string]*Client{},
			idIndex: map[string]*Client{},
			ipToRC:  aghnet.NewIPMap(0),
			allTags: aghstrings.NewSet(),
		}
		ch := make(chan net.IP)
		rdns := &RDNS{
			exchanger: &rDNSExchanger{
				ex: aghtest.Exchanger{
					Ups: tc.ups,
				},
			},
			clients: cc,
			ipCh:    ch,
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

			assert.True(t, cc.Exists(tc.cliIP, ClientSourceRDNS))
		})
	}
}
