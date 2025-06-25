package home

import (
	"encoding/binary"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthMiddlewareGLiNet(t *testing.T) {
	t.Parallel()

	const (
		testTTL = 60 * time.Second

		glTokenFileSuffix = "test"
	)

	tempDir := t.TempDir()
	glFilePrefix = tempDir + "/gl_token_"
	glTokenFile := glFilePrefix + glTokenFileSuffix

	glFileData := make([]byte, 4)
	binary.NativeEndian.PutUint32(glFileData, uint32(time.Now().Add(testTTL).Unix()))

	err := os.WriteFile(glTokenFile, glFileData, 0o644)
	require.NoError(t, err)

	mw := newAuthMiddlewareGLiNet(&authMiddlewareGLiNetConfig{
		logger:          testLogger,
		clock:           timeutil.SystemClock{},
		tokenFilePrefix: glFilePrefix,
		maxTokenSize:    MaxFileSize,
		ttl:             testTTL,
	})

	h := &testAuthHandler{}
	wrapped := mw.Wrap(h)

	reqValidCookie := httptest.NewRequest(http.MethodGet, "/", nil)
	reqValidCookie.AddCookie(&http.Cookie{Name: glCookieName, Value: glTokenFileSuffix})

	reqInvalidCookie := httptest.NewRequest(http.MethodGet, "/", nil)
	reqInvalidCookie.AddCookie(&http.Cookie{Name: glCookieName, Value: "invalid_cookie"})

	testCases := []struct {
		req      *http.Request
		name     string
		wantCode int
	}{{
		req:      httptest.NewRequest(http.MethodGet, "/", nil),
		name:     "no_cookie",
		wantCode: http.StatusFound,
	}, {
		req:      reqValidCookie,
		name:     "valid_cookie",
		wantCode: http.StatusOK,
	}, {
		req:      reqInvalidCookie,
		name:     "invalid_cookie",
		wantCode: http.StatusFound,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			w := httptest.NewRecorder()
			wrapped.ServeHTTP(w, tc.req)

			assert.Equal(t, tc.wantCode, w.Code)
		})
	}
}
