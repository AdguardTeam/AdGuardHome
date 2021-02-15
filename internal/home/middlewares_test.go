package home

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLimitRequestBody(t *testing.T) {
	errReqLimitReached := &aghio.LimitReachedError{
		Limit: defaultReqBodySzLim,
	}

	testCases := []struct {
		name    string
		body    string
		want    []byte
		wantErr error
	}{{
		name:    "not_so_big",
		body:    "somestr",
		want:    []byte("somestr"),
		wantErr: nil,
	}, {
		name:    "so_big",
		body:    string(make([]byte, defaultReqBodySzLim+1)),
		want:    make([]byte, defaultReqBodySzLim),
		wantErr: errReqLimitReached,
	}, {
		name:    "empty",
		body:    "",
		want:    []byte(nil),
		wantErr: nil,
	}}

	makeHandler := func(err *error) http.HandlerFunc {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var b []byte
			b, *err = ioutil.ReadAll(r.Body)
			_, werr := w.Write(b)
			if werr != nil {
				panic(werr)
			}
		})
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var err error
			handler := makeHandler(&err)
			lim := limitRequestBody(handler)

			req := httptest.NewRequest(http.MethodPost, "https://www.example.com", strings.NewReader(tc.body))
			res := httptest.NewRecorder()

			lim.ServeHTTP(res, req)

			require.Equal(t, tc.wantErr, err)
			assert.Equal(t, tc.want, res.Body.Bytes())
		})
	}
}
