//go:build darwin || freebsd || linux || openbsd

package dhcpd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

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
		V6:           V6ServerConf{},
		Leases:       []*leaseDynamic{},
		StaticLeases: []*leaseStatic{},
		Enabled:      true,
	}

	return resp
}

// handleLease is the helper function that calls handler with provided static
// lease as body and returns modified response recorder.
func handleLease(t *testing.T, lease *leaseStatic, handler http.HandlerFunc) (w *httptest.ResponseRecorder) {
	t.Helper()

	w = httptest.NewRecorder()

	b := &bytes.Buffer{}
	err := json.NewEncoder(b).Encode(lease)
	require.NoError(t, err)

	var r *http.Request
	r, err = http.NewRequest(http.MethodPost, "", b)
	require.NoError(t, err)

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

	s, err := Create(&ServerConfig{
		Enabled:        true,
		Conf4:          *defaultV4ServerConf(),
		DataDir:        t.TempDir(),
		ConfigModified: func() {},
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

	s, err := Create(&ServerConfig{
		Enabled:        true,
		Conf4:          *defaultV4ServerConf(),
		Conf6:          V6ServerConf{},
		DataDir:        t.TempDir(),
		ConfigModified: func() {},
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

	s, err := Create(&ServerConfig{
		Enabled:        true,
		Conf4:          *defaultV4ServerConf(),
		Conf6:          V6ServerConf{},
		DataDir:        t.TempDir(),
		ConfigModified: func() {},
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
