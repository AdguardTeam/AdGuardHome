//go:build darwin || freebsd || linux || openbsd

package dhcpd

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/agh"
	"github.com/AdguardTeam/AdGuardHome/internal/aghalg"
	"github.com/AdguardTeam/golibs/osutil/executil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// defaultResponse is a helper that returns the response with default
// configuration.
func defaultResponse() *dhcpStatusResponse {
	conf4 := defaultV4ServerConf()
	conf4.LeaseDuration = 86400

	resp := &dhcpStatusResponse{
		V4:           *conf4,
		V6:           V6ServerConf{PrefixSource: V6PrefixSourceStatic},
		Leases:       []*leaseDynamic{},
		StaticLeases: []*leaseStatic{},
		Enabled:      true,
	}

	return resp
}

func TestV6JSONToServerConf_PrefixSource(t *testing.T) {
	current := V6ServerConf{PrefixSource: V6PrefixSourceInterface}
	rangeStart := netip.MustParseAddr("2001:db8::42")
	leaseDuration := uint32(1800)
	got := v6JSONToServerConf(&v6ServerConfJSON{
		RangeStart:    &rangeStart,
		LeaseDuration: &leaseDuration,
	}, current)

	assert.Equal(t, V6PrefixSourceInterface, got.PrefixSource)
	assert.Equal(t, net.ParseIP("2001:db8::42"), got.RangeStart)
}

func TestServer_HandleDHCPSetConfigV6_InterfaceRASLAACOnly(t *testing.T) {
	s := &server{
		conf: &ServerConfig{
			Logger:             testLogger,
			CommandConstructor: executil.EmptyCommandConstructor{},
			Conf6: V6ServerConf{
				PrefixSource: V6PrefixSourceInterface,
				RASLAACOnly:  true,
			},
		},
	}

	srv6, enabled, err := s.handleDHCPSetConfigV6(&dhcpServerConfigJSON{
		V6:            &v6ServerConfJSON{},
		InterfaceName: "en0",
		Enabled:       aghalg.NBTrue,
	})
	require.NoError(t, err)
	assert.True(t, enabled)

	srv, ok := srv6.(*v6Server)
	require.True(t, ok)
	assert.Equal(t, V6PrefixSourceInterface, srv.conf.PrefixSource)
	assert.True(t, srv.conf.Enabled)
	assert.Nil(t, srv.conf.RangeStart)
}

func TestServer_HandleDHCPSetConfigV6_PreservesLivePrefixSource(t *testing.T) {
	currentSrv, err := v6Create(V6ServerConf{
		Enabled:      true,
		PrefixSource: V6PrefixSourceInterface,
		RangeStart:   net.ParseIP("2001:db8::10"),
		notify:       notify6,
	})
	require.NoError(t, err)

	s := &server{
		srv6: currentSrv,
		conf: &ServerConfig{
			Logger:             testLogger,
			CommandConstructor: executil.EmptyCommandConstructor{},
			Conf6: V6ServerConf{
				PrefixSource: V6PrefixSourceStatic,
				RangeStart:   net.ParseIP("2001:db8::10"),
			},
		},
	}

	rangeStart := netip.MustParseAddr("2001:db8::10")
	leaseDuration := uint32(1800)
	srv6, enabled, err := s.handleDHCPSetConfigV6(&dhcpServerConfigJSON{
		V6: &v6ServerConfJSON{
			RangeStart:    &rangeStart,
			LeaseDuration: &leaseDuration,
		},
		InterfaceName: "en0",
		Enabled:       aghalg.NBTrue,
	})
	require.NoError(t, err)
	assert.True(t, enabled)

	srv, ok := srv6.(*v6Server)
	require.True(t, ok)
	assert.Equal(t, V6PrefixSourceInterface, srv.conf.PrefixSource)
	assert.False(t, srv.conf.skipDeprecatedLeaseRestore)
}

func TestV6JSONToServerConf_PreservesOmittedFields(t *testing.T) {
	current := V6ServerConf{
		RangeStart:    net.ParseIP("2001:db8::10"),
		LeaseDuration: 7200,
		PrefixSource:  V6PrefixSourceStatic,
	}
	prefixSource := V6PrefixSourceInterface

	got := v6JSONToServerConf(&v6ServerConfJSON{
		PrefixSource: &prefixSource,
	}, current)

	assert.Equal(t, net.ParseIP("2001:db8::10"), got.RangeStart)
	assert.Equal(t, uint32(7200), got.LeaseDuration)
	assert.Equal(t, V6PrefixSourceInterface, got.PrefixSource)
}

// handleLease is the helper function that calls handler with provided static
// lease as body and returns modified response recorder.
func handleLease(tb testing.TB, lease *leaseStatic, handler http.HandlerFunc) (w *httptest.ResponseRecorder) {
	tb.Helper()

	w = httptest.NewRecorder()

	b := &bytes.Buffer{}
	err := json.NewEncoder(b).Encode(lease)
	require.NoError(tb, err)

	var r *http.Request
	r, err = http.NewRequest(http.MethodPost, "", b)
	require.NoError(tb, err)

	handler(w, r)

	return w
}

// checkStatus is a helper that asserts the response of
// [*server.handleDHCPStatus].
func checkStatus(t *testing.T, s *server, want *dhcpStatusResponse) {
	w := httptest.NewRecorder()

	b := &bytes.Buffer{}
	err := json.NewEncoder(b).Encode(&want)
	require.NoError(t, err)

	r, err := http.NewRequest(http.MethodPost, "", b)
	require.NoError(t, err)

	s.handleDHCPStatus(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	assert.JSONEq(t, b.String(), w.Body.String())
}

func TestServer_handleDHCPStatus(t *testing.T) {
	const (
		staticName = "static-client"
		staticMAC  = "aa:aa:aa:aa:aa:aa"
	)

	staticIP := netip.MustParseAddr("192.168.10.10")

	staticLease := &leaseStatic{
		HWAddr:   staticMAC,
		IP:       staticIP,
		Hostname: staticName,
	}

	ctx := testutil.ContextWithTimeout(t, testTimeout)
	s, err := Create(ctx, &ServerConfig{
		Logger:       testLogger,
		Enabled:      true,
		Conf4:        *defaultV4ServerConf(),
		DataDir:      t.TempDir(),
		ConfModifier: agh.EmptyConfigModifier{},
	})
	require.NoError(t, err)

	ok := t.Run("status", func(t *testing.T) {
		resp := defaultResponse()

		checkStatus(t, s, resp)
	})
	require.True(t, ok)

	ok = t.Run("add_static_lease", func(t *testing.T) {
		w := handleLease(t, staticLease, s.handleDHCPAddStaticLease)
		assert.Equal(t, http.StatusOK, w.Code)

		resp := defaultResponse()
		resp.StaticLeases = []*leaseStatic{staticLease}

		checkStatus(t, s, resp)
	})
	require.True(t, ok)

	ok = t.Run("add_invalid_lease", func(t *testing.T) {
		w := handleLease(t, staticLease, s.handleDHCPAddStaticLease)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	require.True(t, ok)

	ok = t.Run("remove_static_lease", func(t *testing.T) {
		w := handleLease(t, staticLease, s.handleDHCPRemoveStaticLease)
		assert.Equal(t, http.StatusOK, w.Code)

		resp := defaultResponse()

		checkStatus(t, s, resp)
	})
	require.True(t, ok)

	ok = t.Run("set_config", func(t *testing.T) {
		w := httptest.NewRecorder()

		resp := defaultResponse()
		resp.Enabled = false

		b := &bytes.Buffer{}
		err = json.NewEncoder(b).Encode(&resp)
		require.NoError(t, err)

		var r *http.Request
		r, err = http.NewRequest(http.MethodPost, "", b)
		require.NoError(t, err)

		s.handleDHCPSetConfig(w, r)
		assert.Equal(t, http.StatusOK, w.Code)

		checkStatus(t, s, resp)
	})
	require.True(t, ok)
}

func TestServer_HandleUpdateStaticLease(t *testing.T) {
	const (
		leaseV4Name = "static-client-v4"
		leaseV4MAC  = "44:44:44:44:44:44"

		leaseV6Name = "static-client-v6"
		leaseV6MAC  = "66:66:66:66:66:66"
	)

	leaseV4IP := netip.MustParseAddr("192.168.10.10")
	leaseV6IP := netip.MustParseAddr("2001::6")

	const (
		leaseV4Pos = iota
		leaseV6Pos
	)

	leases := []*leaseStatic{
		leaseV4Pos: {
			HWAddr:   leaseV4MAC,
			IP:       leaseV4IP,
			Hostname: leaseV4Name,
		},
		leaseV6Pos: {
			HWAddr:   leaseV6MAC,
			IP:       leaseV6IP,
			Hostname: leaseV6Name,
		},
	}

	ctx := testutil.ContextWithTimeout(t, testTimeout)
	s, err := Create(ctx, &ServerConfig{
		Logger:       testLogger,
		Enabled:      true,
		Conf4:        *defaultV4ServerConf(),
		Conf6:        V6ServerConf{},
		DataDir:      t.TempDir(),
		ConfModifier: agh.EmptyConfigModifier{},
	})
	require.NoError(t, err)

	for _, l := range leases {
		w := handleLease(t, l, s.handleDHCPAddStaticLease)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	testCases := []struct {
		name  string
		pos   int
		lease *leaseStatic
	}{{
		name: "update_v4_name",
		pos:  leaseV4Pos,
		lease: &leaseStatic{
			HWAddr:   leaseV4MAC,
			IP:       leaseV4IP,
			Hostname: "updated-client-v4",
		},
	}, {
		name: "update_v4_ip",
		pos:  leaseV4Pos,
		lease: &leaseStatic{
			HWAddr:   leaseV4MAC,
			IP:       netip.MustParseAddr("192.168.10.200"),
			Hostname: "updated-client-v4",
		},
	}, {
		name: "update_v6_name",
		pos:  leaseV6Pos,
		lease: &leaseStatic{
			HWAddr:   leaseV6MAC,
			IP:       leaseV6IP,
			Hostname: "updated-client-v6",
		},
	}, {
		name: "update_v6_ip",
		pos:  leaseV6Pos,
		lease: &leaseStatic{
			HWAddr:   leaseV6MAC,
			IP:       netip.MustParseAddr("2001::666"),
			Hostname: "updated-client-v6",
		},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := handleLease(t, tc.lease, s.handleDHCPUpdateStaticLease)
			assert.Equal(t, http.StatusOK, w.Code)

			resp := defaultResponse()
			leases[tc.pos] = tc.lease
			resp.StaticLeases = leases

			checkStatus(t, s, resp)
		})
	}
}

func TestServer_HandleUpdateStaticLease_validation(t *testing.T) {
	const (
		leaseV4Name = "static-client-v4"
		leaseV4MAC  = "44:44:44:44:44:44"

		anotherV4Name = "another-client-v4"
		anotherV4MAC  = "55:55:55:55:55:55"
	)

	leaseV4IP := netip.MustParseAddr("192.168.10.10")
	anotherV4IP := netip.MustParseAddr("192.168.10.20")

	leases := []*leaseStatic{{
		HWAddr:   leaseV4MAC,
		IP:       leaseV4IP,
		Hostname: leaseV4Name,
	}, {
		HWAddr:   anotherV4MAC,
		IP:       anotherV4IP,
		Hostname: anotherV4Name,
	}}

	ctx := testutil.ContextWithTimeout(t, testTimeout)
	s, err := Create(ctx, &ServerConfig{
		Logger:       testLogger,
		Enabled:      true,
		Conf4:        *defaultV4ServerConf(),
		Conf6:        V6ServerConf{},
		DataDir:      t.TempDir(),
		ConfModifier: agh.EmptyConfigModifier{},
	})
	require.NoError(t, err)

	for _, l := range leases {
		w := handleLease(t, l, s.handleDHCPAddStaticLease)
		assert.Equal(t, http.StatusOK, w.Code)
	}

	testCases := []struct {
		lease *leaseStatic
		name  string
		want  string
	}{{
		name: "v4_unknown_mac",
		lease: &leaseStatic{
			HWAddr:   "aa:aa:aa:aa:aa:aa",
			IP:       leaseV4IP,
			Hostname: leaseV4Name,
		},
		want: "dhcpv4: updating static lease: can't find lease aa:aa:aa:aa:aa:aa\n",
	}, {
		name: "update_v4_same_ip",
		lease: &leaseStatic{
			HWAddr:   leaseV4MAC,
			IP:       anotherV4IP,
			Hostname: leaseV4Name,
		},
		want: "dhcpv4: updating static lease: ip address is not unique\n",
	}, {
		name: "update_v4_same_name",
		lease: &leaseStatic{
			HWAddr:   leaseV4MAC,
			IP:       leaseV4IP,
			Hostname: anotherV4Name,
		},
		want: "dhcpv4: updating static lease: hostname is not unique\n",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := handleLease(t, tc.lease, s.handleDHCPUpdateStaticLease)
			assert.Equal(t, http.StatusBadRequest, w.Code)
			assert.Equal(t, tc.want, w.Body.String())
		})
	}
}
