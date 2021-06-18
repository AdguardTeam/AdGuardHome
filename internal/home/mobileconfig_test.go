package home

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"howett.net/plist"
)

func TestHandleMobileConfigDoH(t *testing.T) {
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
		assert.Equal(t, "example.org DoH", mc.PayloadContent[0].Name)
		assert.Equal(t, "example.org DoH", mc.PayloadContent[0].PayloadDisplayName)
		assert.Equal(t, "https://example.org/dns-query", mc.PayloadContent[0].DNSSettings.ServerURL)
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
		assert.Equal(t, "example.org DoH", mc.PayloadContent[0].Name)
		assert.Equal(t, "example.org DoH", mc.PayloadContent[0].PayloadDisplayName)
		assert.Equal(t, "https://example.org/dns-query/cli42", mc.PayloadContent[0].DNSSettings.ServerURL)
	})
}

func TestHandleMobileConfigDoT(t *testing.T) {
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
		assert.Equal(t, "example.org DoT", mc.PayloadContent[0].Name)
		assert.Equal(t, "example.org DoT", mc.PayloadContent[0].PayloadDisplayName)
		assert.Equal(t, "example.org", mc.PayloadContent[0].DNSSettings.ServerName)
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
		assert.Equal(t, "example.org DoT", mc.PayloadContent[0].Name)
		assert.Equal(t, "example.org DoT", mc.PayloadContent[0].PayloadDisplayName)
		assert.Equal(t, "cli42.example.org", mc.PayloadContent[0].DNSSettings.ServerName)
	})
}
