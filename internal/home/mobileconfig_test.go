package home

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/dnsforward"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"howett.net/plist"
)

// testBootstrapDNS are the bootstrap plain DNS server addresses for tests.
var testBootstrapDNS = []string{
	"94.140.14.14",
	"94.140.15.15",
}

// setupBootstraps is a helper that sets up the bootstrap plain DNS server
// configuration for tests and also tears it down in a cleanup function.
func setupBootstraps(t testing.TB) {
	t.Helper()

	prevConfig := config
	t.Cleanup(func() {
		config = prevConfig
	})
	config = &configuration{
		DNS: dnsConfig{
			FilteringConfig: dnsforward.FilteringConfig{
				BootstrapDNS: testBootstrapDNS,
			},
		},
	}
}

func TestHandleMobileConfigDoH(t *testing.T) {
	setupBootstraps(t)

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

		assert.Equal(t, testBootstrapDNS, s.ServerAddresses)
		assert.Empty(t, s.ServerName)
		assert.Equal(t, "https://example.org/dns-query", s.ServerURL)
	})

	t.Run("error_no_host", func(t *testing.T) {
		oldTLSConf := Context.tls
		t.Cleanup(func() { Context.tls = oldTLSConf })

		Context.tls = &TLSMod{conf: tlsConfigSettings{}}

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

		assert.Equal(t, testBootstrapDNS, s.ServerAddresses)
		assert.Empty(t, s.ServerName)
		assert.Equal(t, "https://example.org/dns-query/cli42", s.ServerURL)
	})
}

func TestHandleMobileConfigDoT(t *testing.T) {
	setupBootstraps(t)

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

		assert.Equal(t, testBootstrapDNS, s.ServerAddresses)
		assert.Equal(t, "example.org", s.ServerName)
		assert.Empty(t, s.ServerURL)
	})

	t.Run("error_no_host", func(t *testing.T) {
		oldTLSConf := Context.tls
		t.Cleanup(func() { Context.tls = oldTLSConf })

		Context.tls = &TLSMod{conf: tlsConfigSettings{}}

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

		assert.Equal(t, testBootstrapDNS, s.ServerAddresses)
		assert.Equal(t, "cli42.example.org", s.ServerName)
		assert.Empty(t, s.ServerURL)
	})
}
