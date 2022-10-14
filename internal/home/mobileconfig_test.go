package home

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"howett.net/plist"
)

// setupDNSIPs is a helper that sets up the server IP address configuration for
// tests and also tears it down in a cleanup function.
func setupDNSIPs(t testing.TB) {
	t.Helper()

	prevConfig := config
	prevTLS := Context.tls
	t.Cleanup(func() {
		config = prevConfig
		Context.tls = prevTLS
	})

	config = &configuration{
		DNS: dnsConfig{
			BindHosts: []netip.Addr{netip.IPv4Unspecified()},
			Port:      defaultPortDNS,
		},
	}

	Context.tls = &tlsManager{}
}

func TestHandleMobileConfigDoH(t *testing.T) {
	setupDNSIPs(t)

	t.Run("success", func(t *testing.T) {
		r, err := http.NewRequest(http.MethodGet, "https://example.com:12345/apple/doh.mobileconfig?host=example.org", nil)
		require.NoError(t, err)

		w := httptest.NewRecorder()

		handleMobileConfigDoH(w, r)
		require.Equal(t, http.StatusOK, w.Code)

		var mc mobileConfig
		_, err = plist.Unmarshal(w.Body.Bytes(), &mc)
		require.NoError(t, err)
		require.Len(t, mc.PayloadContent, 1)

		assert.Equal(t, "example.org DoH", mc.PayloadContent[0].PayloadDisplayName)

		s := mc.PayloadContent[0].DNSSettings
		require.NotNil(t, s)

		assert.Empty(t, s.ServerName)
		assert.Equal(t, "https://example.org/dns-query", s.ServerURL)
	})

	t.Run("error_no_host", func(t *testing.T) {
		oldTLSConf := Context.tls
		t.Cleanup(func() { Context.tls = oldTLSConf })

		Context.tls = &tlsManager{conf: tlsConfigSettings{}}

		r, err := http.NewRequest(http.MethodGet, "https://example.com:12345/apple/doh.mobileconfig", nil)
		require.NoError(t, err)

		b := &bytes.Buffer{}
		err = json.NewEncoder(b).Encode(&jsonError{
			Message: errEmptyHost.Error(),
		})
		require.NoError(t, err)

		w := httptest.NewRecorder()

		handleMobileConfigDoH(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.JSONEq(t, w.Body.String(), b.String())
	})

	t.Run("client_id", func(t *testing.T) {
		r, err := http.NewRequest(http.MethodGet, "https://example.com:12345/apple/doh.mobileconfig?host=example.org&client_id=cli42", nil)
		require.NoError(t, err)

		w := httptest.NewRecorder()

		handleMobileConfigDoH(w, r)
		require.Equal(t, http.StatusOK, w.Code)

		var mc mobileConfig
		_, err = plist.Unmarshal(w.Body.Bytes(), &mc)
		require.NoError(t, err)
		require.Len(t, mc.PayloadContent, 1)

		assert.Equal(t, "example.org DoH", mc.PayloadContent[0].PayloadDisplayName)

		s := mc.PayloadContent[0].DNSSettings
		require.NotNil(t, s)

		assert.Empty(t, s.ServerName)
		assert.Equal(t, "https://example.org/dns-query/cli42", s.ServerURL)
	})
}

func TestHandleMobileConfigDoT(t *testing.T) {
	setupDNSIPs(t)

	t.Run("success", func(t *testing.T) {
		r, err := http.NewRequest(http.MethodGet, "https://example.com:12345/apple/dot.mobileconfig?host=example.org", nil)
		require.NoError(t, err)

		w := httptest.NewRecorder()

		handleMobileConfigDoT(w, r)
		require.Equal(t, http.StatusOK, w.Code)

		var mc mobileConfig
		_, err = plist.Unmarshal(w.Body.Bytes(), &mc)
		require.NoError(t, err)
		require.Len(t, mc.PayloadContent, 1)

		assert.Equal(t, "example.org DoT", mc.PayloadContent[0].PayloadDisplayName)

		s := mc.PayloadContent[0].DNSSettings
		require.NotNil(t, s)

		assert.Equal(t, "example.org", s.ServerName)
		assert.Empty(t, s.ServerURL)
	})

	t.Run("error_no_host", func(t *testing.T) {
		oldTLSConf := Context.tls
		t.Cleanup(func() { Context.tls = oldTLSConf })

		Context.tls = &tlsManager{conf: tlsConfigSettings{}}

		r, err := http.NewRequest(http.MethodGet, "https://example.com:12345/apple/dot.mobileconfig", nil)
		require.NoError(t, err)

		b := &bytes.Buffer{}
		err = json.NewEncoder(b).Encode(&jsonError{
			Message: errEmptyHost.Error(),
		})
		require.NoError(t, err)

		w := httptest.NewRecorder()

		handleMobileConfigDoT(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.JSONEq(t, w.Body.String(), b.String())
	})

	t.Run("client_id", func(t *testing.T) {
		r, err := http.NewRequest(http.MethodGet, "https://example.com:12345/apple/dot.mobileconfig?host=example.org&client_id=cli42", nil)
		require.NoError(t, err)

		w := httptest.NewRecorder()

		handleMobileConfigDoT(w, r)
		require.Equal(t, http.StatusOK, w.Code)

		var mc mobileConfig
		_, err = plist.Unmarshal(w.Body.Bytes(), &mc)
		require.NoError(t, err)
		require.Len(t, mc.PayloadContent, 1)

		assert.Equal(t, "example.org DoT", mc.PayloadContent[0].PayloadDisplayName)

		s := mc.PayloadContent[0].DNSSettings
		require.NotNil(t, s)

		assert.Equal(t, "cli42.example.org", s.ServerName)
		assert.Empty(t, s.ServerURL)
	})
}
