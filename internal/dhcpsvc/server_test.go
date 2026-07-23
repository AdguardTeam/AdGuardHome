package dhcpsvc_test

import (
	"context"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/dhcpsvc"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDHCPServer_AddLease(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		want       *dhcpsvc.Lease
		wantErrMsg string
	}{{
		name: "outside_range",
		want: &dhcpsvc.Lease{
			Hostname: testLease4HostnameUnknown,
			IP:       netip.MustParseAddr("1.2.3.4"),
			HWAddr:   testHWUnknown,
		},
		wantErrMsg: "adding lease: no interface for ip 1.2.3.4",
	}, {
		name: "duplicate_ip",
		want: &dhcpsvc.Lease{
			Hostname: testLease4HostnameUnknown,
			IP:       testIPv4Static,
			HWAddr:   testHWUnknown,
		},
		wantErrMsg: "adding lease: lease for ip " + testIPv4Static.String() +
			" already exists",
	}, {
		name: "duplicate_hostname",
		want: &dhcpsvc.Lease{
			Hostname: testLease4HostnameStatic,
			IP:       testIPv4Unknown,
			HWAddr:   testHWUnknown,
		},
		wantErrMsg: "adding lease: lease for hostname " + testLease4HostnameStatic +
			" already exists",
	}, {
		name: "duplicate_hostname_case",
		want: &dhcpsvc.Lease{
			Hostname: strings.ToUpper(testLease4HostnameStatic),
			IP:       testIPv4Unknown,
			HWAddr:   testHWUnknown,
		},
		wantErrMsg: "adding lease: lease for hostname " +
			strings.ToUpper(testLease4HostnameStatic) + " already exists",
	}, {
		name: "duplicate_mac",
		want: &dhcpsvc.Lease{
			Hostname: testLease4HostnameUnknown,
			IP:       testIPv4Unknown,
			HWAddr:   testHWStatic,
		},
		wantErrMsg: "adding lease: lease for mac " + testHWStatic.String() +
			" already exists",
	}, {
		name: "valid",
		want: &dhcpsvc.Lease{
			Hostname: testLease4HostnameUnknown,
			IP:       testIPv4Unknown,
			HWAddr:   testHWUnknown,
		},
		wantErrMsg: "",
	}, {
		name: "valid_v6",
		want: &dhcpsvc.Lease{
			Hostname: testLease6HostnameUnknown,
			IP:       testIPv6Unknown,
			HWAddr:   testHWUnknown,
		},
		wantErrMsg: "",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			onStore := func(ctx context.Context, leases []*dhcpsvc.Lease) (err error) {
				assert.Contains(t, leases, tc.want)

				return nil
			}

			db := newTestDatabase(t, testLeases)
			if tc.wantErrMsg == "" {
				db.onStore = onStore
			}

			srv := newTestDHCPServer(t, &dhcpsvc.Config{
				Database: db,
				Enabled:  true,
			})

			ctx := testutil.ContextWithTimeout(t, testTimeout)

			err := srv.AddLease(ctx, tc.want)
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)
		})
	}
}

func TestDHCPServer_index(t *testing.T) {
	t.Parallel()

	srv := newTestDHCPServer(t, &dhcpsvc.Config{
		Database: newTestDatabase(t, testLeases),
		Enabled:  true,
	})

	t.Run("ip_idx", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, testIPv4Static, srv.IPByHost(testLease4HostnameStatic))
		assert.Equal(t, testIPv4Dynamic, srv.IPByHost(testLease4HostnameDynamic))
		// TODO(e.burkov):  Consider treating expired leases as non-existent.
		assert.Equal(t, testIPv4Expired, srv.IPByHost(testLease4HostnameExpired))
		assert.Zero(t, srv.IPByHost(testLease4HostnameUnknown))
	})

	t.Run("name_idx", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, testLease4HostnameStatic, srv.HostByIP(testIPv4Static))
		assert.Equal(t, testLease4HostnameDynamic, srv.HostByIP(testIPv4Dynamic))
		assert.Equal(t, testLease4HostnameExpired, srv.HostByIP(testIPv4Expired))
		assert.Zero(t, srv.HostByIP(testIPv4Unknown))
		assert.Zero(t, srv.HostByIP(netip.Addr{}))
	})

	t.Run("mac_idx", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, testHWStatic, srv.MACByIP(testIPv4Static))
		assert.Equal(t, testHWDynamic, srv.MACByIP(testIPv4Dynamic))
		assert.Equal(t, testHWExpired, srv.MACByIP(testIPv4Expired))
		assert.Zero(t, srv.MACByIP(testIPv4Unknown))
		assert.Zero(t, srv.MACByIP(netip.Addr{}))
	})
}

func TestDHCPServer_UpdateStaticLease(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		lease      *dhcpsvc.Lease
		wantErrMsg string
	}{{
		name: "outside_range",
		lease: &dhcpsvc.Lease{
			IP:       testIPv4Conf.RangeEnd.Next(),
			Expiry:   time.Time{},
			Hostname: testLease4HostnameStatic,
			HWAddr:   testHWStatic,
			IsStatic: true,
		},
		wantErrMsg: "updating static lease: no interface for ip " +
			testIPv4Conf.RangeEnd.Next().String(),
	}, {
		name: "not_found",
		lease: &dhcpsvc.Lease{
			IP:       testIPv4Unknown,
			Expiry:   time.Time{},
			Hostname: testLease4HostnameUnknown,
			HWAddr:   testHWUnknown,
			IsStatic: true,
		},
		wantErrMsg: "updating static lease: no lease for mac " + testHWUnknown.String(),
	}, {
		name: "duplicate_ip",
		lease: &dhcpsvc.Lease{
			IP:       testIPv4Dynamic,
			Expiry:   time.Time{},
			Hostname: testLease4HostnameStatic,
			HWAddr:   testHWStatic,
			IsStatic: true,
		},
		wantErrMsg: "updating static lease: lease for ip " + testIPv4Dynamic.String() +
			" already exists",
	}, {
		name: "duplicate_hostname",
		lease: &dhcpsvc.Lease{
			IP:       testIPv4Unknown,
			Expiry:   time.Time{},
			Hostname: testLease4HostnameDynamic,
			HWAddr:   testHWStatic,
			IsStatic: true,
		},
		wantErrMsg: "updating static lease: lease for hostname " + testLease4HostnameDynamic +
			" already exists",
	}, {
		name: "duplicate_hostname_case",
		lease: &dhcpsvc.Lease{
			IP:       testIPv4Unknown,
			Expiry:   time.Time{},
			Hostname: strings.ToUpper(testLease4HostnameDynamic),
			HWAddr:   testHWStatic,
			IsStatic: true,
		},
		wantErrMsg: "updating static lease: lease for hostname " +
			strings.ToUpper(testLease4HostnameDynamic) + " already exists",
	}, {
		name: "valid",
		lease: &dhcpsvc.Lease{
			IP:       testIPv4Unknown,
			Expiry:   time.Time{},
			Hostname: testLease4HostnameStatic,
			HWAddr:   testHWStatic,
			IsStatic: true,
		},
		wantErrMsg: "",
	}, {
		name: "valid_v6",
		lease: &dhcpsvc.Lease{
			IP:       testIPv6Unknown,
			Expiry:   time.Time{},
			Hostname: testLease6HostnameUnknown,
			HWAddr:   testHWStatic,
			IsStatic: true,
		},
		wantErrMsg: "",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			onStore := func(ctx context.Context, leases []*dhcpsvc.Lease) (err error) {
				assert.Contains(t, leases, tc.lease)

				return nil
			}

			db := newTestDatabase(t, testLeases)
			if tc.wantErrMsg == "" {
				db.onStore = onStore
			}

			srv := newTestDHCPServer(t, &dhcpsvc.Config{
				Database: db,
				Enabled:  true,
			})

			ctx := testutil.ContextWithTimeout(t, testTimeout)
			testutil.AssertErrorMsg(t, tc.wantErrMsg, srv.UpdateStaticLease(ctx, tc.lease))
		})
	}
}

func TestDHCPServer_RemoveLease(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		want       *dhcpsvc.Lease
		wantErrMsg string
	}{{
		name: "not_found_mac",
		want: &dhcpsvc.Lease{
			Hostname: testLease4HostnameStatic,
			IP:       testIPv4Static,
			HWAddr:   testHWUnknown,
		},
		wantErrMsg: "removing lease: no lease for mac " + testHWUnknown.String(),
	}, {
		name: "not_found_ip",
		want: &dhcpsvc.Lease{
			Hostname: testLease4HostnameStatic,
			IP:       testIPv4Unknown,
			HWAddr:   testHWStatic,
		},
		wantErrMsg: "removing lease: no lease for ip " + testIPv4Unknown.String(),
	}, {
		name: "not_found_host",
		want: &dhcpsvc.Lease{
			Hostname: testLease4HostnameUnknown,
			IP:       testIPv4Static,
			HWAddr:   testHWStatic,
		},
		wantErrMsg: "removing lease: no lease for hostname " + testLease4HostnameUnknown,
	}, {
		name: "valid",
		want: &dhcpsvc.Lease{
			Hostname: testLease4HostnameStatic,
			IP:       testIPv4Static,
			HWAddr:   testHWStatic,
		},
		wantErrMsg: "",
	}, {
		name: "valid_v6",
		want: &dhcpsvc.Lease{
			Hostname: testLease6HostnameStatic,
			IP:       testIPv6Static,
			HWAddr:   testHWStatic,
		},
		wantErrMsg: "",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			onStore := func(ctx context.Context, leases []*dhcpsvc.Lease) (err error) {
				assert.NotContains(t, leases, tc.want)

				return nil
			}

			db := newTestDatabase(t, testLeases)
			if tc.wantErrMsg == "" {
				db.onStore = onStore
			}

			srv := newTestDHCPServer(t, &dhcpsvc.Config{
				Database: db,
				Enabled:  true,
			})

			ctx := testutil.ContextWithTimeout(t, testTimeout)

			err := srv.RemoveLease(ctx, tc.want)
			testutil.AssertErrorMsg(t, tc.wantErrMsg, err)
		})
	}
}

func TestDHCPServer_Reset(t *testing.T) {
	t.Parallel()

	db := newTestDatabase(t, testLeases)
	db.onStore = func(_ context.Context, leases []*dhcpsvc.Lease) (err error) {
		assert.Empty(t, leases)

		return nil
	}

	srv := newTestDHCPServer(t, &dhcpsvc.Config{
		Database: db,
		Enabled:  true,
	})

	require.ElementsMatch(t, srv.Leases(), testLeases)

	ctx := testutil.ContextWithTimeout(t, testTimeout)
	require.NoError(t, srv.Reset(ctx))

	assert.Empty(t, srv.Leases())
}

func TestServer_Leases(t *testing.T) {
	t.Parallel()

	srv := newTestDHCPServer(t, &dhcpsvc.Config{
		Database: newTestDatabase(t, testLeases),
		Enabled:  true,
	})

	assert.ElementsMatch(t, testLeases, srv.Leases())
}
