package home

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"path"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghuser"
	"github.com/AdguardTeam/golibs/netutil/urlutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/net/http2"
)

func TestWebApi_H2CVulnerability(t *testing.T) {
	ctx := testutil.ContextWithTimeout(t, testTimeout)

	passwordHash, err := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	require.NoError(t, err)

	user := webUser{
		Name:         "foo",
		PasswordHash: string(passwordHash),
		UserID:       aghuser.MustNewUserID(),
	}

	auth, err := newAuth(testutil.ContextWithTimeout(t, testTimeout), &authConfig{
		baseLogger:     testLogger,
		rateLimiter:    emptyRateLimiter{},
		trustedProxies: testTrustedProxies,
		dbFilename:     path.Join(t.TempDir(), "sessions.db"),
		users:          []webUser{user},
		sessionTTL:     testTimeout,
		isGLiNet:       false,
	})

	require.NoError(t, err)
	t.Cleanup(func() { auth.close(ctx) })

	web := newTestWeb(t, &webConfig{
		auth: auth,
	})

	host := config.HTTPConfig.Address.String()

	go web.start(ctx)

	u := &url.URL{
		Scheme: urlutil.SchemeHTTP,
		Host:   host,
		Path:   "/health-check",
	}
	require.EventuallyWithT(t, func(c *assert.CollectT) {
		var req *http.Request
		req, err = http.NewRequest(http.MethodGet, u.String(), nil)
		require.NoError(c, err)

		_, err = http.DefaultClient.Do(req)
		require.NoError(c, err)
	}, testTimeout, testTimeout/10)

	t.Cleanup(func() { web.close(ctx) })

	// TODO(f.setrakov): !! Consider implementing a custom H2C client that
	// allows us to make H2C requests using the Upgrade method instead of prior
	// knowledge, and substitute a different path after a successful connection
	// has been established.
	client := &http.Client{
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLS: func(network, addr string, _ *tls.Config) (net.Conn, error) {
				return net.Dial(network, addr)
			},
		},
	}

	u.Path = "/control/profile"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}
