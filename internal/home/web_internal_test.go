package home

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/agh"
	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/aghuser"
	"github.com/AdguardTeam/AdGuardHome/internal/querylog"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/netutil/urlutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
)

const (
	// clientPreface is the message sent to the server as a final confirmation
	// of HTTP2 usage.
	clientPreface = "PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"

	// testSettings is a common value of the HTTP2-Settings header for tests.
	testSettings = "AAEAABAAAAIAAAABAAQAAP__AAUAAEAAAAgAAAAAAAMAAABkAAYAAQAA"
)

// performH2CUpgradeAttack establishes a TCP connection to the specified host,
// performs an HTTP2 protocol upgrade, and attempts to access a protected
// endpoint without proper authentication, verifying that the server responds
// with [http.StatusUnauthorized].
func performH2CUpgradeAttack(tb testing.TB, host string) {
	tb.Helper()

	dialer := &net.Dialer{}
	ctx := testutil.ContextWithTimeout(tb, testTimeout)

	conn, err := dialer.DialContext(ctx, "tcp", host)
	require.NoError(tb, err)
	testutil.CleanupAndRequireSuccess(tb, conn.Close)

	writer := bufio.NewWriter(conn)
	reader := bufio.NewReader(conn)

	u := &url.URL{
		Scheme: urlutil.SchemeHTTP,
		Host:   host,
		Path:   "/control/login",
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	require.NoError(tb, err)

	req.Header.Set("Connection", "Upgrade, HTTP2-Settings")
	req.Header.Set("Upgrade", "h2c")
	req.Header.Set("HTTP2-Settings", testSettings)

	err = req.Write(writer)
	require.NoError(tb, err)
	require.NoError(tb, writer.Flush())

	resp, err := http.ReadResponse(reader, req)
	require.NoError(tb, err)
	testutil.CleanupAndRequireSuccess(tb, resp.Body.Close)

	_, err = writer.Write([]byte(clientPreface))
	require.NoError(tb, err)

	framer := http2.NewFramer(writer, reader)
	performH2CSettingsExchange(tb, framer, writer)
	sendAttackH2CRequest(tb, framer, writer, host)
}

// performH2CSettingsExchange performs the HTTP2 settings exchange handshake. It
// sends empty client settings, waits for acknowledgement, and then receives
// the server settings and responds with acknowledgement.
func performH2CSettingsExchange(tb testing.TB, framer *http2.Framer, writer *bufio.Writer) {
	tb.Helper()

	err := framer.WriteSettings()
	require.NoError(tb, err)
	require.NoError(tb, writer.Flush())

	var (
		gotServerSettings bool
		gotSettingsAck    bool
	)
	for !gotServerSettings || !gotSettingsAck {
		var frame http2.Frame
		frame, err = framer.ReadFrame()
		require.NoError(tb, err)

		settings, ok := frame.(*http2.SettingsFrame)
		if !ok {
			continue
		}

		if settings.IsAck() {
			gotSettingsAck = true

			continue
		}

		err = framer.WriteSettingsAck()
		require.NoError(tb, err)
		require.NoError(tb, writer.Flush())

		gotServerSettings = true
	}
}

// sendAttackH2CRequest sends a request to a protected endpoint via an
// established H2C connection, asserting that the server will respond with
// [http.StatusUnauthorized].
func sendAttackH2CRequest(tb testing.TB, framer *http2.Framer, writer *bufio.Writer, host string) {
	tb.Helper()

	var headerBlockFragment bytes.Buffer
	enc := hpack.NewEncoder(&headerBlockFragment)
	headers := []hpack.HeaderField{
		{Name: ":method", Value: http.MethodGet},
		{Name: ":path", Value: "/control/querylog"},
		{Name: ":scheme", Value: urlutil.SchemeHTTP},
		{Name: ":authority", Value: host},
	}

	for _, h := range headers {
		require.NoError(tb, enc.WriteField(h))
	}

	dec := hpack.NewDecoder(4096, func(f hpack.HeaderField) {
		if f.Name != ":status" {
			return
		}

		gotStatus, err := strconv.ParseInt(f.Value, 10, 64)
		require.NoError(tb, err)

		assert.Equal(tb, http.StatusUnauthorized, int(gotStatus))
	})

	// NOTE: Stream ID 1 is implicitly used by the upgrade request, so we
	// continue sending requests starting from the first client-side valid ID.
	targetStreamID := uint32(3)
	err := framer.WriteHeaders(http2.HeadersFrameParam{
		StreamID:      targetStreamID,
		BlockFragment: headerBlockFragment.Bytes(),
		EndHeaders:    true,
		EndStream:     true,
	})
	require.NoError(tb, err)
	require.NoError(tb, writer.Flush())

	for {
		var frame http2.Frame
		frame, err = framer.ReadFrame()
		require.NoError(tb, err)

		if frame.Header().StreamID != targetStreamID {
			continue
		}

		headerFrame := testutil.RequireTypeAssert[*http2.HeadersFrame](tb, frame)
		_, err = dec.Write(headerFrame.HeaderBlockFragment())
		require.NoError(tb, err)
		require.True(tb, headerFrame.StreamEnded())

		break
	}
}

func TestWebAPI_H2CVulnerability(t *testing.T) {
	password := "password"
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
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

	ctx := testutil.ContextWithTimeout(t, testTimeout)
	t.Cleanup(func() { auth.close(ctx) })

	mux := http.NewServeMux()
	mw := &webMw{}
	registrar := aghhttp.NewDefaultRegistrar(mux, mw.wrap)
	web := newTestWeb(t, &webConfig{
		auth: auth,
	})

	queryLog, err := querylog.New(querylog.Config{
		Logger:         testLogger,
		ConfigModifier: agh.EmptyConfigModifier{},
		HTTPReg:        registrar,
		RotationIvl:    24 * time.Hour,
		Enabled:        false,
	})
	require.NoError(t, err)

	mw.set(web)
	globalContext.queryLog = queryLog
	globalContext.web = web

	err = queryLog.Start(ctx)
	require.NoError(t, err)

	testutil.CleanupAndRequireSuccess(t, func() (err error) {
		return queryLog.Shutdown(ctx)
	})

	port := config.HTTPConfig.Address.Port()
	host := fmt.Sprintf("%s:%d", netutil.IPv4Localhost(), port)

	go web.start(ctx)
	t.Cleanup(func() { web.close(ctx) })

	waitForWebAPIReady(t, user.Name, password, host)
	performH2CUpgradeAttack(t, host)
}

// waitForWebAPIReady waits until the [webAPI] server has started and is ready
// to accept connections.
func waitForWebAPIReady(tb testing.TB, username, password, host string) {
	tb.Helper()

	u := &url.URL{
		Scheme: urlutil.SchemeHTTP,
		Host:   host,
		Path:   "/control/login",
	}

	body := bytes.NewBuffer(nil)
	loginReq := loginJSON{
		Name:     username,
		Password: password,
	}

	err := json.NewEncoder(body).Encode(loginReq)
	require.NoError(tb, err)

	require.EventuallyWithT(tb, func(c *assert.CollectT) {
		ctx := testutil.ContextWithTimeout(tb, testTimeout)
		var req *http.Request
		req, err = http.NewRequestWithContext(ctx, http.MethodPost, u.String(), body)
		require.NoError(c, err)

		var resp *http.Response
		resp, err = http.DefaultClient.Do(req)
		require.NoError(c, err)
		require.Equal(c, http.StatusOK, resp.StatusCode)
	}, testTimeout, testTimeout/10)
}
