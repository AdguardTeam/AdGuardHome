package home

import (
	"encoding/binary"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AdguardTeam/golibs/testutil"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthMiddlewareGLiNet(t *testing.T) {
	t.Parallel()

	const (
		testTTL = 60 * time.Second

		glTokenFileSuffix = "test"

		testPerm fs.FileMode = 0o644
	)

	tempDir := t.TempDir()
	glTokenFolder := filepath.Join(tempDir, "foo")
	err := os.MkdirAll(glTokenFolder, 0o755)
	require.NoError(t, err)

	tokenFileRoot, err := os.OpenRoot(glTokenFolder)
	require.NoError(t, err)
	testutil.CleanupAndRequireSuccess(t, tokenFileRoot.Close)

	err = os.MkdirAll(filepath.Join(glTokenFolder, glFilePrefix), testPerm)
	require.NoError(t, err)

	glTokenFile := filepath.Join(glTokenFolder, glFilePrefix+glTokenFileSuffix)

	glFileData := make([]byte, 4)
	binary.NativeEndian.PutUint32(glFileData, uint32(time.Now().Add(testTTL).Unix()))

	err = os.WriteFile(glTokenFile, glFileData, testPerm)
	require.NoError(t, err)

	// Mock token file for testing path traversal vulnerability.  See AG-54304.
	passwdFile := filepath.Join(tempDir, "path_traversal_token")
	err = os.WriteFile(passwdFile, glFileData, testPerm)
	require.NoError(t, err)

	mw := newAuthMiddlewareGLiNet(&authMiddlewareGLiNetConfig{
		logger:        testLogger,
		mux:           http.NewServeMux(),
		clock:         timeutil.SystemClock{},
		tokenFileRoot: tokenFileRoot,
		maxTokenSize:  MaxFileSize,
		ttl:           testTTL,
	})

	h := &testAuthHandler{}
	wrapped := mw.Wrap(h)

	reqValidCookie := httptest.NewRequest(http.MethodGet, "/", nil)
	reqValidCookie.AddCookie(&http.Cookie{Name: glCookieName, Value: glTokenFileSuffix})

	reqInvalidCookie := httptest.NewRequest(http.MethodGet, "/", nil)
	reqInvalidCookie.AddCookie(&http.Cookie{Name: glCookieName, Value: "invalid_cookie"})

	reqPathTraversalToken := httptest.NewRequest(http.MethodGet, "/", nil)
	reqPathTraversalToken.AddCookie(&http.Cookie{
		Name:  glCookieName,
		Value: "/../../path_traversal_token",
	})

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
	}, {
		req:      reqPathTraversalToken,
		name:     "path_traversal_token",
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
