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

	// checkStatus is a helper that asserts the response of
	// [*server.handleDHCPStatus].
	checkStatus := func(t *testing.T, want *dhcpStatusResponse) {
		w := httptest.NewRecorder()
		var req *http.Request
		req, err = http.NewRequest(http.MethodGet, "", nil)
		require.NoError(t, err)

		b := &bytes.Buffer{}
		err = json.NewEncoder(b).Encode(&want)
		require.NoError(t, err)

		s.handleDHCPStatus(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		assert.JSONEq(t, b.String(), w.Body.String())
	}

	// defaultResponse is a helper that returs the response with default
	// configuration.
	defaultResponse := func() *dhcpStatusResponse {
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

	ok := t.Run("status", func(t *testing.T) {
		resp := defaultResponse()

		checkStatus(t, resp)
	})
	require.True(t, ok)

	ok = t.Run("add_static_lease", func(t *testing.T) {
		w := httptest.NewRecorder()

		b := &bytes.Buffer{}
		err = json.NewEncoder(b).Encode(staticLease)
		require.NoError(t, err)

		var r *http.Request
		r, err = http.NewRequest(http.MethodPost, "", b)
		require.NoError(t, err)

		s.handleDHCPAddStaticLease(w, r)
		assert.Equal(t, http.StatusOK, w.Code)

		resp := defaultResponse()
		resp.StaticLeases = []*leaseStatic{staticLease}

		checkStatus(t, resp)
	})
	require.True(t, ok)

	ok = t.Run("add_invalid_lease", func(t *testing.T) {
		w := httptest.NewRecorder()

		b := &bytes.Buffer{}

		err = json.NewEncoder(b).Encode(&leaseStatic{})
		require.NoError(t, err)

		var r *http.Request
		r, err = http.NewRequest(http.MethodPost, "", b)
		require.NoError(t, err)

		s.handleDHCPAddStaticLease(w, r)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
	require.True(t, ok)

	ok = t.Run("remove_static_lease", func(t *testing.T) {
		w := httptest.NewRecorder()

		b := &bytes.Buffer{}
		err = json.NewEncoder(b).Encode(staticLease)
		require.NoError(t, err)

		var r *http.Request
		r, err = http.NewRequest(http.MethodPost, "", b)
		require.NoError(t, err)

		s.handleDHCPRemoveStaticLease(w, r)
		assert.Equal(t, http.StatusOK, w.Code)

		resp := defaultResponse()

		checkStatus(t, resp)
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

		checkStatus(t, resp)
	})
	require.True(t, ok)
}
