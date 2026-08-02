package home

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/AdguardTeam/golibs/httphdr"
	"github.com/AdguardTeam/golibs/ioutil"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecoverPanics(t *testing.T) {
	const panicMessage = "sensitive test panic"

	logOutput := &bytes.Buffer{}
	l := slog.New(slog.NewTextHandler(logOutput, &slog.HandlerOptions{
		ReplaceAttr: slogutil.RemoveTime,
	}))
	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		header := w.Header()
		header.Set(httphdr.AccessControlAllowOrigin, "http://example.test")
		header.Set(httphdr.AltSvc, `h3=":443"`)
		header.Set(httphdr.ContentEncoding, "gzip")
		header.Set(httphdr.ContentType, "application/json")
		header.Set(httphdr.SetCookie, "secret=value")
		header.Set(httphdr.StrictTransportSecurity, "max-age=31536000")
		header.Set(httphdr.Trailer, "X-Test-Trailer")
		header.Set(httphdr.Vary, httphdr.Origin)
		header.Set(http.TrailerPrefix+"X-Test-Trailer", "value")

		panic(panicMessage)
	})

	req := httptest.NewRequest(http.MethodGet, "https://example.test/panic", nil)
	res := httptest.NewRecorder()

	require.NotPanics(t, func() {
		recoverPanics(l, h).ServeHTTP(res, req)
	})

	assert.Equal(t, http.StatusInternalServerError, res.Code)
	assert.Equal(t, http.StatusText(http.StatusInternalServerError)+"\n", res.Body.String())
	header := res.Header()
	assert.Equal(t, "http://example.test", header.Get(httphdr.AccessControlAllowOrigin))
	assert.Equal(t, `h3=":443"`, header.Get(httphdr.AltSvc))
	assert.Equal(t, "text/plain; charset=utf-8", header.Get(httphdr.ContentType))
	assert.Equal(t, "no-store", header.Get(httphdr.CacheControl))
	assert.Equal(t, "nosniff", header.Get(httphdr.XContentTypeOptions))
	assert.Equal(t, "max-age=31536000", header.Get(httphdr.StrictTransportSecurity))
	assert.Equal(t, httphdr.Origin, header.Get(httphdr.Vary))
	assert.Empty(t, header.Values(httphdr.SetCookie))
	assert.Empty(t, header.Get(httphdr.ContentEncoding))
	assert.Empty(t, header.Get(httphdr.Trailer))
	assert.Empty(t, header.Get(http.TrailerPrefix+"X-Test-Trailer"))
	assert.NotContains(t, res.Body.String(), panicMessage)

	logString := logOutput.String()
	assert.Contains(t, logString, `level=ERROR msg="recovered from panic" value="`+panicMessage+`"`)
	assert.Contains(t, logString, "level=ERROR msg=stack")
}

func TestRecoverPanics_invalidStatus(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(99)
	})

	req := httptest.NewRequest(http.MethodGet, "https://example.test/panic", nil)
	res := httptest.NewRecorder()

	require.NotPanics(t, func() {
		recoverPanics(testLogger, h).ServeHTTP(res, req)
	})
	assert.Equal(t, http.StatusInternalServerError, res.Code)
	assert.Equal(t, http.StatusText(http.StatusInternalServerError)+"\n", res.Body.String())
}

type informationalResponseWriter struct {
	header http.Header
	body   bytes.Buffer

	finalStatus int
	infos       []int
}

func (w *informationalResponseWriter) Header() (h http.Header) {
	return w.header
}

func (w *informationalResponseWriter) Write(b []byte) (n int, err error) {
	if w.finalStatus == 0 {
		w.finalStatus = http.StatusOK
	}

	return w.body.Write(b)
}

func (w *informationalResponseWriter) WriteHeader(statusCode int) {
	if statusCode >= 100 && statusCode <= 199 && statusCode != http.StatusSwitchingProtocols {
		w.infos = append(w.infos, statusCode)

		return
	}

	if w.finalStatus == 0 {
		w.finalStatus = statusCode
	}
}

func TestRecoverPanics_earlyHints(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusEarlyHints)

		panic("after early hints")
	})

	req := httptest.NewRequest(http.MethodGet, "https://example.test/panic", nil)
	res := &informationalResponseWriter{header: http.Header{}}

	require.NotPanics(t, func() {
		recoverPanics(testLogger, h).ServeHTTP(res, req)
	})
	assert.Equal(t, []int{http.StatusEarlyHints}, res.infos)
	assert.Equal(t, http.StatusInternalServerError, res.finalStatus)
	assert.Equal(t, http.StatusText(http.StatusInternalServerError)+"\n", res.body.String())
}

func TestRecoverPanics_normal(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Test", "value")
		w.WriteHeader(http.StatusCreated)
		_, _ = io.WriteString(w, "response body")
	})

	req := httptest.NewRequest(http.MethodGet, "https://example.test/normal", nil)
	res := httptest.NewRecorder()
	recoverPanics(testLogger, h).ServeHTTP(res, req)

	assert.Equal(t, http.StatusCreated, res.Code)
	assert.Equal(t, "value", res.Header().Get("X-Test"))
	assert.Equal(t, "response body", res.Body.String())
}

func TestRecoverPanics_abort(t *testing.T) {
	logOutput := &bytes.Buffer{}
	l := slog.New(slog.NewTextHandler(logOutput, nil))
	h := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic(http.ErrAbortHandler)
	})

	req := httptest.NewRequest(http.MethodGet, "https://example.test/abort", nil)
	res := httptest.NewRecorder()

	require.PanicsWithValue(t, http.ErrAbortHandler, func() {
		recoverPanics(l, h).ServeHTTP(res, req)
	})
	assert.Empty(t, logOutput.String())
}

func TestRecoverPanics_committed(t *testing.T) {
	testCases := []struct {
		prepare func(w http.ResponseWriter) (err error)
		name    string
	}{
		{
			name: "write",
			prepare: func(w http.ResponseWriter) (err error) {
				_, err = io.WriteString(w, "partial response")

				return err
			},
		}, {
			name: "write_header",
			prepare: func(w http.ResponseWriter) (err error) {
				w.WriteHeader(http.StatusNoContent)

				return nil
			},
		}, {
			name: "flush",
			prepare: func(w http.ResponseWriter) (err error) {
				return http.NewResponseController(w).Flush()
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				require.NoError(t, tc.prepare(w))

				panic("after commit")
			})

			req := httptest.NewRequest(http.MethodGet, "https://example.test/panic", nil)
			res := httptest.NewRecorder()

			require.PanicsWithValue(t, http.ErrAbortHandler, func() {
				recoverPanics(testLogger, h).ServeHTTP(res, req)
			})
		})
	}
}

func TestRecoverPanics_responseController(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		err := http.NewResponseController(w).Flush()
		require.NoError(t, err)
	})

	req := httptest.NewRequest(http.MethodGet, "https://example.test/flush", nil)
	res := httptest.NewRecorder()
	recoverPanics(testLogger, h).ServeHTTP(res, req)

	assert.True(t, res.Flushed)
}

func TestRecoverPanics_committedServer(t *testing.T) {
	const partialBody = "partial response"

	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set(httphdr.ContentLength, "100")
		_, err := io.WriteString(w, partialBody)
		require.NoError(t, err)

		err = http.NewResponseController(w).Flush()
		require.NoError(t, err)

		panic("after flush")
	})

	srv := httptest.NewServer(recoverPanics(testLogger, h))
	t.Cleanup(srv.Close)

	res, err := srv.Client().Get(srv.URL)
	require.NoError(t, err)
	testutil.CleanupAndRequireSuccess(t, res.Body.Close)

	body, err := io.ReadAll(res.Body)
	assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
	assert.Equal(t, http.StatusOK, res.StatusCode)
	assert.Equal(t, partialBody, string(body))
}

func TestLimitRequestBody(t *testing.T) {
	errReqLimitReached := &ioutil.LimitError{
		Limit: defaultReqBodySzLim.Bytes(),
	}

	testCases := []struct {
		wantErr error
		name    string
		body    string
		want    []byte
	}{{
		wantErr: nil,
		name:    "not_so_big",
		body:    "somestr",
		want:    []byte("somestr"),
	}, {
		wantErr: errReqLimitReached,
		name:    "so_big",
		body:    string(make([]byte, defaultReqBodySzLim+1)),
		want:    make([]byte, defaultReqBodySzLim),
	}, {
		wantErr: nil,
		name:    "empty",
		body:    "",
		want:    []byte(nil),
	}}

	makeHandler := func(tb testing.TB, err *error) http.HandlerFunc {
		tb.Helper()

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var b []byte
			b, *err = io.ReadAll(r.Body)
			_, werr := w.Write(b)
			require.NoError(tb, werr)
		})
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var err error
			handler := makeHandler(t, &err)
			lim := limitRequestBody(handler)

			req := httptest.NewRequest(http.MethodPost, "https://www.example.com", strings.NewReader(tc.body))
			res := httptest.NewRecorder()

			lim.ServeHTTP(res, req)

			assert.Equal(t, tc.wantErr, err)
			assert.Equal(t, tc.want, res.Body.Bytes())
		})
	}
}
