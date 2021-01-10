package home

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"howett.net/plist"
)

func TestHandleMobileConfigDOH(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r, err := http.NewRequest(http.MethodGet, "https://example.com:12345/apple/doh.mobileconfig?host=example.org", nil)
		assert.Nil(t, err)

		w := httptest.NewRecorder()

		handleMobileConfigDOH(w, r)
		assert.Equal(t, http.StatusOK, w.Code)

		var mc mobileConfig
		_, err = plist.Unmarshal(w.Body.Bytes(), &mc)
		assert.Nil(t, err)

		if assert.Equal(t, 1, len(mc.PayloadContent)) {
			assert.Equal(t, "example.org DoH", mc.PayloadContent[0].Name)
			assert.Equal(t, "example.org DoH", mc.PayloadContent[0].PayloadDisplayName)
			assert.Equal(t, "example.org", mc.PayloadContent[0].DNSSettings.ServerName)
			assert.Equal(t, "https://example.org/dns-query", mc.PayloadContent[0].DNSSettings.ServerURL)
		}
	})

	t.Run("success_no_host", func(t *testing.T) {
		oldTLSConf := Context.tls
		t.Cleanup(func() { Context.tls = oldTLSConf })

		Context.tls = &TLSMod{
			conf: tlsConfigSettings{ServerName: "example.org"},
		}

		r, err := http.NewRequest(http.MethodGet, "https://example.com:12345/apple/doh.mobileconfig", nil)
		assert.Nil(t, err)

		w := httptest.NewRecorder()

		handleMobileConfigDOH(w, r)
		assert.Equal(t, http.StatusOK, w.Code)

		var mc mobileConfig
		_, err = plist.Unmarshal(w.Body.Bytes(), &mc)
		assert.Nil(t, err)

		if assert.Equal(t, 1, len(mc.PayloadContent)) {
			assert.Equal(t, "example.org DoH", mc.PayloadContent[0].Name)
			assert.Equal(t, "example.org DoH", mc.PayloadContent[0].PayloadDisplayName)
			assert.Equal(t, "example.org", mc.PayloadContent[0].DNSSettings.ServerName)
			assert.Equal(t, "https://example.org/dns-query", mc.PayloadContent[0].DNSSettings.ServerURL)
		}
	})

	t.Run("error_no_host", func(t *testing.T) {
		oldTLSConf := Context.tls
		t.Cleanup(func() { Context.tls = oldTLSConf })

		Context.tls = &TLSMod{conf: tlsConfigSettings{}}

		r, err := http.NewRequest(http.MethodGet, "https://example.com:12345/apple/doh.mobileconfig", nil)
		assert.Nil(t, err)

		w := httptest.NewRecorder()

		handleMobileConfigDOH(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

func TestHandleMobileConfigDOT(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		r, err := http.NewRequest(http.MethodGet, "https://example.com:12345/apple/dot.mobileconfig?host=example.org", nil)
		assert.Nil(t, err)

		w := httptest.NewRecorder()

		handleMobileConfigDOT(w, r)
		assert.Equal(t, http.StatusOK, w.Code)

		var mc mobileConfig
		_, err = plist.Unmarshal(w.Body.Bytes(), &mc)
		assert.Nil(t, err)

		if assert.Equal(t, 1, len(mc.PayloadContent)) {
			assert.Equal(t, "example.org DoT", mc.PayloadContent[0].Name)
			assert.Equal(t, "example.org DoT", mc.PayloadContent[0].PayloadDisplayName)
			assert.Equal(t, "example.org", mc.PayloadContent[0].DNSSettings.ServerName)
		}
	})

	t.Run("success_no_host", func(t *testing.T) {
		oldTLSConf := Context.tls
		t.Cleanup(func() { Context.tls = oldTLSConf })

		Context.tls = &TLSMod{
			conf: tlsConfigSettings{ServerName: "example.org"},
		}

		r, err := http.NewRequest(http.MethodGet, "https://example.com:12345/apple/dot.mobileconfig", nil)
		assert.Nil(t, err)

		w := httptest.NewRecorder()

		handleMobileConfigDOT(w, r)
		assert.Equal(t, http.StatusOK, w.Code)

		var mc mobileConfig
		_, err = plist.Unmarshal(w.Body.Bytes(), &mc)
		assert.Nil(t, err)

		if assert.Equal(t, 1, len(mc.PayloadContent)) {
			assert.Equal(t, "example.org DoT", mc.PayloadContent[0].Name)
			assert.Equal(t, "example.org DoT", mc.PayloadContent[0].PayloadDisplayName)
			assert.Equal(t, "example.org", mc.PayloadContent[0].DNSSettings.ServerName)
		}
	})

	t.Run("error_no_host", func(t *testing.T) {
		oldTLSConf := Context.tls
		t.Cleanup(func() { Context.tls = oldTLSConf })

		Context.tls = &TLSMod{conf: tlsConfigSettings{}}

		r, err := http.NewRequest(http.MethodGet, "https://example.com:12345/apple/dot.mobileconfig", nil)
		assert.Nil(t, err)

		w := httptest.NewRecorder()

		handleMobileConfigDOT(w, r)
		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}
