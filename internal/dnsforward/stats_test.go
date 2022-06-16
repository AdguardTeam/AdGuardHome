package dnsforward

import (
	"net"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/AdGuardHome/internal/querylog"
	"github.com/AdguardTeam/AdGuardHome/internal/stats"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/dnsproxy/upstream"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testQueryLog is a simple querylog.QueryLog implementation for tests.
type testQueryLog struct {
	// QueryLog is embedded here simply to make testQueryLog
	// a querylog.QueryLog without actually implementing all methods.
	querylog.QueryLog

	lastParams *querylog.AddParams
}

// Add implements the querylog.QueryLog interface for *testQueryLog.
func (l *testQueryLog) Add(p *querylog.AddParams) {
	l.lastParams = p
}

// testStats is a simple stats.Stats implementation for tests.
type testStats struct {
	// Stats is embedded here simply to make testStats a stats.Stats without
	// actually implementing all methods.
	stats.Stats

	lastEntry stats.Entry
}

// Update implements the stats.Stats interface for *testStats.
func (l *testStats) Update(e stats.Entry) {
	l.lastEntry = e
}

func TestProcessQueryLogsAndStats(t *testing.T) {
	testCases := []struct {
		name           string
		proto          proxy.Proto
		addr           net.Addr
		clientID       string
		wantLogProto   querylog.ClientProto
		wantStatClient string
		wantCode       resultCode
		reason         filtering.Reason
		wantStatResult stats.Result
	}{{
		name:           "success_udp",
		proto:          proxy.ProtoUDP,
		addr:           &net.UDPAddr{IP: net.IP{1, 2, 3, 4}, Port: 1234},
		clientID:       "",
		wantLogProto:   "",
		wantStatClient: "1.2.3.4",
		wantCode:       resultCodeSuccess,
		reason:         filtering.NotFilteredNotFound,
		wantStatResult: stats.RNotFiltered,
	}, {
		name:           "success_tls_clientid",
		proto:          proxy.ProtoTLS,
		addr:           &net.TCPAddr{IP: net.IP{1, 2, 3, 4}, Port: 1234},
		clientID:       "cli42",
		wantLogProto:   querylog.ClientProtoDoT,
		wantStatClient: "cli42",
		wantCode:       resultCodeSuccess,
		reason:         filtering.NotFilteredNotFound,
		wantStatResult: stats.RNotFiltered,
	}, {
		name:           "success_tls",
		proto:          proxy.ProtoTLS,
		addr:           &net.TCPAddr{IP: net.IP{1, 2, 3, 4}, Port: 1234},
		clientID:       "",
		wantLogProto:   querylog.ClientProtoDoT,
		wantStatClient: "1.2.3.4",
		wantCode:       resultCodeSuccess,
		reason:         filtering.NotFilteredNotFound,
		wantStatResult: stats.RNotFiltered,
	}, {
		name:           "success_quic",
		proto:          proxy.ProtoQUIC,
		addr:           &net.UDPAddr{IP: net.IP{1, 2, 3, 4}, Port: 1234},
		clientID:       "",
		wantLogProto:   querylog.ClientProtoDoQ,
		wantStatClient: "1.2.3.4",
		wantCode:       resultCodeSuccess,
		reason:         filtering.NotFilteredNotFound,
		wantStatResult: stats.RNotFiltered,
	}, {
		name:           "success_https",
		proto:          proxy.ProtoHTTPS,
		addr:           &net.TCPAddr{IP: net.IP{1, 2, 3, 4}, Port: 1234},
		clientID:       "",
		wantLogProto:   querylog.ClientProtoDoH,
		wantStatClient: "1.2.3.4",
		wantCode:       resultCodeSuccess,
		reason:         filtering.NotFilteredNotFound,
		wantStatResult: stats.RNotFiltered,
	}, {
		name:           "success_dnscrypt",
		proto:          proxy.ProtoDNSCrypt,
		addr:           &net.TCPAddr{IP: net.IP{1, 2, 3, 4}, Port: 1234},
		clientID:       "",
		wantLogProto:   querylog.ClientProtoDNSCrypt,
		wantStatClient: "1.2.3.4",
		wantCode:       resultCodeSuccess,
		reason:         filtering.NotFilteredNotFound,
		wantStatResult: stats.RNotFiltered,
	}, {
		name:           "success_udp_filtered",
		proto:          proxy.ProtoUDP,
		addr:           &net.UDPAddr{IP: net.IP{1, 2, 3, 4}, Port: 1234},
		clientID:       "",
		wantLogProto:   "",
		wantStatClient: "1.2.3.4",
		wantCode:       resultCodeSuccess,
		reason:         filtering.FilteredBlockList,
		wantStatResult: stats.RFiltered,
	}, {
		name:           "success_udp_sb",
		proto:          proxy.ProtoUDP,
		addr:           &net.UDPAddr{IP: net.IP{1, 2, 3, 4}, Port: 1234},
		clientID:       "",
		wantLogProto:   "",
		wantStatClient: "1.2.3.4",
		wantCode:       resultCodeSuccess,
		reason:         filtering.FilteredSafeBrowsing,
		wantStatResult: stats.RSafeBrowsing,
	}, {
		name:           "success_udp_ss",
		proto:          proxy.ProtoUDP,
		addr:           &net.UDPAddr{IP: net.IP{1, 2, 3, 4}, Port: 1234},
		clientID:       "",
		wantLogProto:   "",
		wantStatClient: "1.2.3.4",
		wantCode:       resultCodeSuccess,
		reason:         filtering.FilteredSafeSearch,
		wantStatResult: stats.RSafeSearch,
	}, {
		name:           "success_udp_pc",
		proto:          proxy.ProtoUDP,
		addr:           &net.UDPAddr{IP: net.IP{1, 2, 3, 4}, Port: 1234},
		clientID:       "",
		wantLogProto:   "",
		wantStatClient: "1.2.3.4",
		wantCode:       resultCodeSuccess,
		reason:         filtering.FilteredParental,
		wantStatResult: stats.RParental,
	}}

	ups, err := upstream.AddressToUpstream("1.1.1.1", nil)
	require.NoError(t, err)

	for _, tc := range testCases {
		ql := &testQueryLog{}
		st := &testStats{}
		srv := &Server{
			queryLog:   ql,
			stats:      st,
			anonymizer: aghnet.NewIPMut(nil),
		}
		t.Run(tc.name, func(t *testing.T) {
			req := &dns.Msg{
				Question: []dns.Question{{
					Name: "example.com.",
				}},
			}
			pctx := &proxy.DNSContext{
				Proto:    tc.proto,
				Req:      req,
				Res:      &dns.Msg{},
				Addr:     tc.addr,
				Upstream: ups,
			}
			dctx := &dnsContext{
				proxyCtx:  pctx,
				startTime: time.Now(),
				result: &filtering.Result{
					Reason: tc.reason,
				},
				clientID: tc.clientID,
			}

			code := srv.processQueryLogsAndStats(dctx)
			assert.Equal(t, tc.wantCode, code)
			assert.Equal(t, tc.wantLogProto, ql.lastParams.ClientProto)
			assert.Equal(t, tc.wantStatClient, st.lastEntry.Client)
			assert.Equal(t, tc.wantStatResult, st.lastEntry.Result)
		})
	}
}
