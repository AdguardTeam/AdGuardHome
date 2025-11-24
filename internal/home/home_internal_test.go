package home

import (
	"cmp"
	"net/http"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/agh"
	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/require"
)

// testLogger is a common logger for tests.
var testLogger = slogutil.NewDiscardLogger()

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

func TestMain(m *testing.M) {
	initCmdLineOpts()
	testutil.DiscardLogOutput(m)
}
