package websvc_test

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/v1/websvc"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testTimeout = 1 * time.Second

func TestService_Start_getHealthCheck(t *testing.T) {
	c := &websvc.Config{
		TLS: nil,
		Addresses: []*netutil.IPPort{{
			IP:   net.IP{127, 0, 0, 1},
			Port: 0,
		}},
		SecureAddresses: nil,
		Timeout:         testTimeout,
	}

	svc := websvc.New(c)

	err := svc.Start()
	require.NoError(t, err)
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
		t.Cleanup(cancel)

		err = svc.Shutdown(ctx)
		require.NoError(t, err)
	})

	addrs := svc.Addrs()
	require.Len(t, addrs, 1)

	u := &url.URL{
		Scheme: "http",
		Host:   addrs[0],
		Path:   "/health-check",
	}
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	require.NoError(t, err)

	httpCli := &http.Client{
		Timeout: testTimeout,
	}
	resp, err := httpCli.Do(req)
	require.NoError(t, err)

	testutil.CleanupAndRequireSuccess(t, resp.Body.Close)

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	assert.Equal(t, []byte("OK"), body)
}
