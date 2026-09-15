package aghhttp_test

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/golibs/httphdr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStaticFileServer(t *testing.T) {
	const (
		fingerprint = "0123456789abcdef0123"
		immutable   = "public, max-age=31536000, immutable"
		noCache     = "no-cache"
	)

	assetModTime := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	files := fstest.MapFS{
		"index.html":        {Data: []byte("index")},
		"install.html":      {Data: []byte("install")},
		"manifest.json":     {Data: []byte("manifest")},
		"service-worker.js": {Data: []byte("worker")},
		"main." + fingerprint + ".js": {
			Data:    []byte("javascript"),
			ModTime: assetModTime,
		},
		"main." + fingerprint + ".css":        {Data: []byte("stylesheet")},
		"main.deadbeef.js":                    {Data: []byte("short hash")},
		"main.0123456789abcdefg123.js":        {Data: []byte("non-hex hash")},
		"archive." + fingerprint + ".txt":     {Data: []byte("unsupported")},
		"assets/chunk." + fingerprint + ".js": {Data: []byte("chunk")},
		"assets/icon." + fingerprint + ".svg": {Data: []byte("icon")},
		"assets/font." + fingerprint + ".woff2": {
			Data: []byte("font"),
		},
		"assets/favicon.svg": {
			Data:    []byte("favicon"),
			ModTime: assetModTime,
		},
		"directory." + fingerprint + ".js": {
			Mode: fs.ModeDir,
		},
	}

	h := aghhttp.NewStaticFileServer(files)
	testCases := []struct {
		name       string
		method     string
		path       string
		rangeHdr   string
		ifModSince string
		wantCache  string
		wantCode   int
	}{
		{
			name:      "root_html",
			method:    http.MethodGet,
			path:      "/",
			wantCache: noCache,
			wantCode:  http.StatusOK,
		},
		{
			name:      "install_html",
			method:    http.MethodGet,
			path:      "/install.html",
			wantCache: noCache,
			wantCode:  http.StatusOK,
		},
		{
			name:      "index_redirect",
			method:    http.MethodGet,
			path:      "/index.html",
			wantCache: noCache,
			wantCode:  http.StatusMovedPermanently,
		},
		{
			name:      "manifest",
			method:    http.MethodGet,
			path:      "/manifest.json",
			wantCache: noCache,
			wantCode:  http.StatusOK,
		},
		{
			name:      "service_worker",
			method:    http.MethodGet,
			path:      "/service-worker.js",
			wantCache: noCache,
			wantCode:  http.StatusOK,
		},
		{
			name:      "javascript",
			method:    http.MethodGet,
			path:      "/main." + fingerprint + ".js",
			wantCache: immutable,
			wantCode:  http.StatusOK,
		},
		{
			name:      "stylesheet",
			method:    http.MethodGet,
			path:      "/main." + fingerprint + ".css",
			wantCache: immutable,
			wantCode:  http.StatusOK,
		},
		{
			name:      "nested_chunk",
			method:    http.MethodGet,
			path:      "/assets/chunk." + fingerprint + ".js",
			wantCache: immutable,
			wantCode:  http.StatusOK,
		},
		{
			name:      "image",
			method:    http.MethodGet,
			path:      "/assets/icon." + fingerprint + ".svg",
			wantCache: immutable,
			wantCode:  http.StatusOK,
		},
		{
			name:      "font",
			method:    http.MethodGet,
			path:      "/assets/font." + fingerprint + ".woff2",
			wantCache: immutable,
			wantCode:  http.StatusOK,
		},
		{
			name:      "head",
			method:    http.MethodHead,
			path:      "/main." + fingerprint + ".js",
			wantCache: immutable,
			wantCode:  http.StatusOK,
		},
		{
			name:      "range",
			method:    http.MethodGet,
			path:      "/main." + fingerprint + ".js",
			rangeHdr:  "bytes=0-2",
			wantCache: immutable,
			wantCode:  http.StatusPartialContent,
		},
		{
			name:      "query",
			method:    http.MethodGet,
			path:      "/main." + fingerprint + ".js?v=1",
			wantCache: immutable,
			wantCode:  http.StatusOK,
		},
		{
			name:       "not_modified_fingerprinted",
			method:     http.MethodGet,
			path:       "/main." + fingerprint + ".js",
			ifModSince: assetModTime.Format(http.TimeFormat),
			wantCache:  immutable,
			wantCode:   http.StatusNotModified,
		},
		{
			name:       "not_modified_unversioned",
			method:     http.MethodGet,
			path:       "/assets/favicon.svg",
			ifModSince: assetModTime.Format(http.TimeFormat),
			wantCache:  noCache,
			wantCode:   http.StatusNotModified,
		},
		{
			name:      "unversioned_asset",
			method:    http.MethodGet,
			path:      "/assets/favicon.svg",
			wantCache: noCache,
			wantCode:  http.StatusOK,
		},
		{
			name:      "short_hash",
			method:    http.MethodGet,
			path:      "/main.deadbeef.js",
			wantCache: noCache,
			wantCode:  http.StatusOK,
		},
		{
			name:      "non_hex_hash",
			method:    http.MethodGet,
			path:      "/main.0123456789abcdefg123.js",
			wantCache: noCache,
			wantCode:  http.StatusOK,
		},
		{
			name:      "unsupported_extension",
			method:    http.MethodGet,
			path:      "/archive." + fingerprint + ".txt",
			wantCache: noCache,
			wantCode:  http.StatusOK,
		},
		{
			name:      "noncanonical_path",
			method:    http.MethodGet,
			path:      "/assets/../main." + fingerprint + ".js",
			wantCache: noCache,
			wantCode:  http.StatusOK,
		},
		{
			name:      "directory",
			method:    http.MethodGet,
			path:      "/directory." + fingerprint + ".js/",
			wantCache: noCache,
			wantCode:  http.StatusOK,
		},
		{
			name:      "missing",
			method:    http.MethodGet,
			path:      "/missing." + fingerprint + ".js",
			wantCache: noCache,
			wantCode:  http.StatusNotFound,
		},
		{
			name:      "missing_post",
			method:    http.MethodPost,
			path:      "/missing." + fingerprint + ".js",
			wantCache: noCache,
			wantCode:  http.StatusNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			if tc.rangeHdr != "" {
				req.Header.Set("Range", tc.rangeHdr)
			}
			if tc.ifModSince != "" {
				req.Header.Set(httphdr.IfModifiedSince, tc.ifModSince)
			}

			resp := httptest.NewRecorder()
			h.ServeHTTP(resp, req)

			require.Equal(t, tc.wantCode, resp.Code)
			assert.Equal(t, tc.wantCache, resp.Header().Get(httphdr.CacheControl))
		})
	}
}
