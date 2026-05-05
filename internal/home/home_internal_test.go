package home

import (
	"cmp"
	"net/http"
	"net/netip"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/agh"
	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/require"
)

// testTimeout is the common timeout for tests and contexts.
const testTimeout = 1 * time.Second

// testLogger is a common logger for tests.
var testLogger = slogutil.NewDiscardLogger()

// testTrustedProxies is a common trusted proxies set for tests.
var testTrustedProxies = netutil.SliceSubnetSet([]netip.Prefix{})

// newTestWeb is a helper that creates new webAPI and fills it's config with
// given values.  If conf is nil, the default configuration will be used.
func newTestWeb(
	tb testing.TB,
	conf *webConfig,
) (web *webAPI) {
	tb.Helper()

	ctx := testutil.ContextWithTimeout(tb, testTimeout)
	conf = cmp.Or(conf, &webConfig{})

	web, err := newWeb(ctx, &webConfig{
		clientBuildFS:  conf.clientBuildFS,
		updater:        conf.updater,
		opts:           conf.opts,
		baseLogger:     testLogger,
		tlsManager:     conf.tlsManager,
		auth:           conf.auth,
		mux:            cmp.Or(conf.mux, http.NewServeMux()),
		configModifier: cmp.Or[agh.ConfigModifier](conf.configModifier, &agh.EmptyConfigModifier{}),
		httpReg:        cmp.Or[aghhttp.Registrar](conf.httpReg, &aghhttp.EmptyRegistrar{}),
		workDir:        conf.workDir,
		confPath:       conf.confPath,
		isCustomUpdURL: conf.isCustomUpdURL,
		isFirstRun:     conf.isFirstRun,
	})

	require.NoError(tb, err)

	return web
}

// storeGlobals is a test helper function that saves global variables and
// restores them once the test is complete.
//
// The global variables are:
//   - [config]
//   - [glFilePrefix]
//   - [globalContext.clients.storage]
//   - [globalContext.dnsServer]
//   - [globalContext.web]
//
// TODO(s.chzhen):  Remove this once the TLS manager no longer accesses global
// variables.  Make tests that use this helper concurrent.
func storeGlobals(tb testing.TB) {
	tb.Helper()

	prevConfig := config
	prefGLFilePrefix := glFilePrefix
	storage := globalContext.clients.storage
	dnsServer := globalContext.dnsServer
	web := globalContext.web

	tb.Cleanup(func() {
		config = prevConfig
		glFilePrefix = prefGLFilePrefix
		globalContext.clients.storage = storage
		globalContext.dnsServer = dnsServer
		globalContext.web = web
	})
}

func TestMain(m *testing.M) {
	initCmdLineOpts()

	testutil.DiscardLogOutput(m)
}
