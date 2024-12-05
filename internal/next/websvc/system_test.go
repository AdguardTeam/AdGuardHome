package websvc_test

import (
	"encoding/json"
	"net/http"
	"net/url"
	"runtime"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/next/websvc"
	"github.com/AdguardTeam/golibs/netutil/urlutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_handleGetV1SystemInfo(t *testing.T) {
	confMgr := newConfigManager()
	_, addr := newTestServer(t, confMgr)
	u := &url.URL{
		Scheme: urlutil.SchemeHTTP,
		Host:   addr.String(),
		Path:   websvc.PathPatternV1SystemInfo,
	}

	body := httpGet(t, u, http.StatusOK)
	resp := &websvc.RespGetV1SystemInfo{}
	err := json.Unmarshal(body, resp)
	require.NoError(t, err)

	// TODO(a.garipov): Consider making version.Channel and version.Version
	// testable and test these better.
	assert.NotEmpty(t, resp.Channel)

	assert.Equal(t, resp.Arch, runtime.GOARCH)
	assert.Equal(t, resp.OS, runtime.GOOS)
	assert.Equal(t, testStart, time.Time(resp.Start))
}
