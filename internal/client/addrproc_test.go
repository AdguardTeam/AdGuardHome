package client_test

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/netip"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/AdGuardHome/internal/client"
	"github.com/AdguardTeam/AdGuardHome/internal/whois"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/AdguardTeam/golibs/testutil/fakenet"
	"github.com/stretchr/testify/assert"
)

func TestEmptyAddrProc(t *testing.T) {
	t.Parallel()

	p := client.EmptyAddrProc{}

	assert.NotPanics(t, func() {
		ctx := testutil.ContextWithTimeout(t, testTimeout)
		p.Process(ctx, testIP)
	})

	assert.NotPanics(t, func() {
		err := p.Close()
		assert.NoError(t, err)
	})
}

func TestDefaultAddrProc_Process_rDNS(t *testing.T) {
	t.Parallel()

	privateIP := netip.MustParseAddr("192.168.0.1")

	testCases := []struct {
		rdnsErr    error
		ip         netip.Addr
		name       string
		host       string
		usePrivate bool
		wantUpd    bool
	}{{
		rdnsErr:    nil,
		ip:         testIP,
		name:       "success",
		host:       testHost,
		usePrivate: false,
		wantUpd:    true,
	}, {
		rdnsErr:    nil,
		ip:         testIP,
		name:       "no_host",
		host:       "",
		usePrivate: false,
		wantUpd:    false,
	}, {
		rdnsErr:    nil,
		ip:         netip.MustParseAddr("127.0.0.1"),
		name:       "localhost",
		host:       "",
		usePrivate: false,
		wantUpd:    false,
	}, {
		rdnsErr:    nil,
		ip:         privateIP,
		name:       "private_ignored",
		host:       "",
		usePrivate: false,
		wantUpd:    false,
	}, {
		rdnsErr:    nil,
		ip:         privateIP,
		name:       "private_processed",
		host:       "private.example",
		usePrivate: true,
		wantUpd:    true,
	}, {
		rdnsErr:    errors.Error("rdns error"),
		ip:         testIP,
		name:       "rdns_error",
		host:       "",
		usePrivate: false,
		wantUpd:    false,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			updIPCh := make(chan netip.Addr, 1)
			updHostCh := make(chan string, 1)
			updInfoCh := make(chan *whois.Info, 1)

			p := client.NewDefaultAddrProc(&client.DefaultAddrProcConfig{
				BaseLogger: slogutil.NewDiscardLogger(),
				DialContext: func(_ context.Context, _, _ string) (conn net.Conn, err error) {
					panic("not implemented")
				},
				Exchanger: &aghtest.Exchanger{
					OnExchange: func(ip netip.Addr) (host string, ttl time.Duration, err error) {
						return tc.host, 0, tc.rdnsErr
					},
				},
				PrivateSubnets: netutil.SubnetSetFunc(netutil.IsLocallyServed),
				AddressUpdater: &aghtest.AddressUpdater{
					OnUpdateAddress: newOnUpdateAddress(tc.wantUpd, updIPCh, updHostCh, updInfoCh),
				},
				CatchPanics:    false,
				UseRDNS:        true,
				UsePrivateRDNS: tc.usePrivate,
				UseWHOIS:       false,
			})
			testutil.CleanupAndRequireSuccess(t, p.Close)

			ctx := testutil.ContextWithTimeout(t, testTimeout)
			p.Process(ctx, tc.ip)

			if !tc.wantUpd {
				return
			}

			gotIP, _ := testutil.RequireReceive(t, updIPCh, testTimeout)
			assert.Equal(t, tc.ip, gotIP)

			gotHost, _ := testutil.RequireReceive(t, updHostCh, testTimeout)
			assert.Equal(t, tc.host, gotHost)

			gotInfo, _ := testutil.RequireReceive(t, updInfoCh, testTimeout)
			assert.Nil(t, gotInfo)
		})
	}
}

// newOnUpdateAddress is a test helper that returns a new OnUpdateAddress
// callback using the provided channels if an update is expected and panicking
// otherwise.
func newOnUpdateAddress(
	want bool,
	ips chan<- netip.Addr,
	hosts chan<- string,
	infos chan<- *whois.Info,
) (f func(ctx context.Context, ip netip.Addr, host string, info *whois.Info)) {
	return func(ctx context.Context, ip netip.Addr, host string, info *whois.Info) {
		if !want && (host != "" || info != nil) {
			panic(fmt.Errorf("got unexpected update for %v with %q and %v", ip, host, info))
		}

		ips <- ip
		hosts <- host
		infos <- info
	}
}

func TestDefaultAddrProc_Process_WHOIS(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		wantInfo *whois.Info
		exchErr  error
		name     string
		wantUpd  bool
	}{{
		wantInfo: &whois.Info{
			City: testWHOISCity,
		},
		exchErr: nil,
		name:    "success",
		wantUpd: true,
	}, {
		wantInfo: nil,
		exchErr:  nil,
		name:     "no_info",
		wantUpd:  false,
	}, {
		wantInfo: nil,
		exchErr:  errors.Error("whois error"),
		name:     "whois_error",
		wantUpd:  false,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			whoisConn := &fakenet.Conn{
				OnClose: func() (err error) { return nil },
				OnRead: func(b []byte) (n int, err error) {
					if tc.wantInfo == nil {
						return 0, tc.exchErr
					}

					data := "city: " + tc.wantInfo.City + "\n"
					copy(b, data)

					return len(data), io.EOF
				},
				OnSetDeadline: func(_ time.Time) (err error) { return nil },
				OnWrite:       func(b []byte) (n int, err error) { return len(b), nil },
			}

			updIPCh := make(chan netip.Addr, 1)
			updHostCh := make(chan string, 1)
			updInfoCh := make(chan *whois.Info, 1)

			p := client.NewDefaultAddrProc(&client.DefaultAddrProcConfig{
				BaseLogger: slogutil.NewDiscardLogger(),
				DialContext: func(_ context.Context, _, _ string) (conn net.Conn, err error) {
					return whoisConn, nil
				},
				Exchanger: &aghtest.Exchanger{
					OnExchange: func(_ netip.Addr) (_ string, _ time.Duration, _ error) {
						panic("not implemented")
					},
				},
				PrivateSubnets: netutil.SubnetSetFunc(netutil.IsLocallyServed),
				AddressUpdater: &aghtest.AddressUpdater{
					OnUpdateAddress: newOnUpdateAddress(tc.wantUpd, updIPCh, updHostCh, updInfoCh),
				},
				CatchPanics:    false,
				UseRDNS:        false,
				UsePrivateRDNS: false,
				UseWHOIS:       true,
			})
			testutil.CleanupAndRequireSuccess(t, p.Close)

			ctx := testutil.ContextWithTimeout(t, testTimeout)
			p.Process(ctx, testIP)

			if !tc.wantUpd {
				return
			}

			gotIP, _ := testutil.RequireReceive(t, updIPCh, testTimeout)
			assert.Equal(t, testIP, gotIP)

			gotHost, _ := testutil.RequireReceive(t, updHostCh, testTimeout)
			assert.Empty(t, gotHost)

			gotInfo, _ := testutil.RequireReceive(t, updInfoCh, testTimeout)
			assert.Equal(t, tc.wantInfo, gotInfo)
		})
	}
}

func TestDefaultAddrProc_Close(t *testing.T) {
	t.Parallel()

	p := client.NewDefaultAddrProc(&client.DefaultAddrProcConfig{
		BaseLogger: slogutil.NewDiscardLogger(),
	})

	err := p.Close()
	assert.NoError(t, err)

	err = p.Close()
	assert.ErrorIs(t, err, client.ErrClosed)
}
