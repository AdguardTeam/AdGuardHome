package home

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"howett.net/plist"
)

func TestHandleMobileConfigDot(t *testing.T) {
	var err error

	var r *http.Request
	r, err = http.NewRequest(http.MethodGet, "https://example.com:12345/apple/dot.mobileconfig", nil)
	assert.Nil(t, err)

	w := httptest.NewRecorder()

	handleMobileConfigDot(w, r)
	assert.Equal(t, http.StatusOK, w.Code)

	var mc MobileConfig
	_, err = plist.Unmarshal(w.Body.Bytes(), &mc)
	assert.Nil(t, err)

	if assert.Equal(t, 1, len(mc.PayloadContent)) {
		assert.Equal(t, "example.com DoT", mc.PayloadContent[0].Name)
		assert.Equal(t, "example.com DoT", mc.PayloadContent[0].PayloadDisplayName)
		assert.Equal(t, "example.com", mc.PayloadContent[0].DNSSettings.ServerName)
	}
}
