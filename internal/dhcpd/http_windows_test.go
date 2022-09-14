//go:build windows

package dhcpd

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer_notImplemented(t *testing.T) {
	s := &server{}

	w := httptest.NewRecorder()
	r, err := http.NewRequest(http.MethodGet, "/unsupported", nil)
	require.NoError(t, err)

	s.notImplemented(w, r)
	assert.Equal(t, http.StatusNotImplemented, w.Code)

	wantStr := fmt.Sprintf("{%q:%q}", "message", aghos.Unsupported("dhcp"))
	assert.JSONEq(t, wantStr, w.Body.String())
}
