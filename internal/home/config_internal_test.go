package home

import (
	"net/netip"
	"os"
	"path/filepath"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigFilePath(t *testing.T) {
	const (
		realConf       = "real.yaml"
		linkConf       = "conf.link"
		missingConf    = "missing.yaml"
		brokenLinkConf = "broken.link"
	)

	workDir := t.TempDir()
	targetPath := filepath.Join(workDir, realConf)
	linkPath := filepath.Join(workDir, linkConf)
	missingPath := filepath.Join(workDir, missingConf)
	brokenLinkPath := filepath.Join(workDir, brokenLinkConf)

	err := os.Symlink(targetPath, linkPath)
	require.NoError(t, err)

	err = os.Symlink(missingPath, brokenLinkPath)
	require.NoError(t, err)

	f, err := os.Create(targetPath)
	require.NoError(t, err)

	testutil.CleanupAndRequireSuccess(t, f.Close)

	otherDir := t.TempDir()

	// Canonicalize the absolute path (e.g., on macOS: /var -> /private/var; on
	// Windows: RUNNER~1 -> runneradmin).
	wantAbs := targetPath
	p, err := filepath.EvalSymlinks(wantAbs)
	if err == nil {
		wantAbs = p
	}

	testCases := []struct {
		name     string
		chDir    string
		confPath string
		want     string
	}{{
		name:     "absolute_path",
		chDir:    "",
		confPath: targetPath,
		want:     wantAbs,
	}, {
		name:     "relative_path",
		chDir:    "",
		confPath: realConf,
		want:     targetPath,
	}, {
		name:     "symlink",
		chDir:    "",
		confPath: linkConf,
		want:     linkPath,
	}, {
		name:     "symlink_broken",
		chDir:    "",
		confPath: brokenLinkConf,
		want:     brokenLinkPath,
	}, {
		name:     "symlink_before_join",
		chDir:    otherDir,
		confPath: linkConf,
		want:     linkPath,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.chDir != "" {
				t.Chdir(tc.chDir)
			}

			ctx := testutil.ContextWithTimeout(t, testTimeout)
			got := configFilePath(ctx, testLogger, workDir, tc.confPath)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestDefaultConfig_DualStack(t *testing.T) {
	// config is a global variable in internal/home/config.go
	expectedBindHosts := []netip.Addr{netip.IPv4Unspecified(), netip.IPv6Unspecified()}
	assert.Equal(t, expectedBindHosts, config.DNS.BindHosts)
	assert.Equal(t, netip.AddrPortFrom(netip.IPv6Unspecified(), 3000), config.HTTPConfig.Address)
}

func TestNewServerConfig_DualStackFallback(t *testing.T) {
	dnsConf := &dnsConfig{
		BindHosts: nil,
		Port:      53,
		PendingRequests: &pendingRequests{
			Enabled: false,
		},
	}
	tlsConf := &tlsConfigSettings{}
	dohConf := &doHConfig{}

	conf, err := newServerConfig(
		dnsConf,
		&clientSourcesConfig{},
		tlsConf,
		dohConf,
		&tlsManager{},
		&aghtest.Registrar{},
		nil, // clientsContainer
		&aghtest.ConfigModifier{},
	)
	assert.NoError(t, err)

	assert.Len(t, conf.UDPListenAddrs, 2)
	assert.Equal(t, netutil.IPv4Localhost().String(), conf.UDPListenAddrs[0].IP.String())
	assert.Equal(t, netutil.IPv6Localhost().String(), conf.UDPListenAddrs[1].IP.String())
}

