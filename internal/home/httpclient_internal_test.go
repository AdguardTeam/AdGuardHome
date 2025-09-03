package home

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AdguardTeam/golibs/httphdr"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomUserAgentTransport_RoundTrip(t *testing.T) {
	t.Parallel()

	const (
		customUA  = "Custom-user-agent/1.1"
		presentUA = "Present-user-agent/1.1"
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua := r.Header.Get(httphdr.UserAgent)
		_, err := io.WriteString(w, ua)
		require.NoError(testutil.PanicT{}, err)
	}))
	t.Cleanup(srv.Close)

	testCases := []struct {
		client *http.Client
		wantUA []byte
		reqUA  string
		name   string
	}{{
		client: &http.Client{Transport: http.DefaultTransport},
		wantUA: []byte("Go-http-client/1.1"),
		reqUA:  "",
		name:   "default",
	}, {
		client: &http.Client{
			Transport: newCustomUserAgentTransport(http.DefaultTransport, customUA),
		},
		reqUA:  "",
		wantUA: []byte(customUA),
		name:   "custom",
	}, {
		client: &http.Client{
			Transport: newCustomUserAgentTransport(http.DefaultTransport, customUA),
		},
		reqUA:  presentUA,
		wantUA: []byte(presentUA),
		name:   "present",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := testutil.ContextWithTimeout(t, testTimeout)
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
			require.NoError(t, err)

			if tc.reqUA != "" {
				req.Header.Set(httphdr.UserAgent, tc.reqUA)
			}

			resp, err := tc.client.Do(req)
			require.NoError(t, err)

			testutil.CleanupAndRequireSuccess(t, resp.Body.Close)

			got, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			assert.Equal(t, tc.wantUA, got)
		})
	}
}
