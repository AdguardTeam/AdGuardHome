package home

import (
	"crypto/x509"
	"net/netip"
	"os"
	"path/filepath"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/AdGuardHome/internal/aghtls"
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

// newTestTLSConfigProvider returns a [aghtls.TLSConfigProvider] fake that
// serves the given extended TLS configuration.  extTLSConf must not be nil.
func newTestTLSConfigProvider(extTLSConf *aghtls.ExtendedTLSConfig) (p *aghtest.TLSConfigProvider) {
	return &aghtest.TLSConfigProvider{
		OnExtendedTLSConfig: func() (conf *aghtls.ExtendedTLSConfig) {
			return extTLSConf
		},
		OnRootCAs: func() (pool *x509.CertPool) {
			return nil
		},
	}
}

func TestNewServerConfig_DefaultHosts(t *testing.T) {
	dnsConf := &dnsConfig{
		BindHosts: nil,
		Port:      53,
		PendingRequests: &pendingRequests{
			Enabled: false,
		},
	}
	dohConf := &doHConfig{}

	conf, err := newServerConfig(
		dnsConf,
		&clientSourcesConfig{},
		dohConf,
		newTestTLSConfigProvider(&aghtls.ExtendedTLSConfig{}),
		&aghtest.Registrar{},
		nil, // clientsContainer
		&aghtest.ConfigModifier{},
	)
	require.NoError(t, err)
	require.Len(t, conf.UDPListenAddrs, 2)

	assert.Equal(t, netutil.IPv4Localhost().String(), conf.UDPListenAddrs[0].IP.String())
	assert.Equal(t, netutil.IPv6Localhost().String(), conf.UDPListenAddrs[1].IP.String())
}

func TestNewServerConfig_Issue8363BindHosts(t *testing.T) {
	bindHosts := []netip.Addr{
		netip.IPv4Unspecified(),
		netip.IPv6Unspecified(),
		netutil.IPv4Localhost(),
		netutil.IPv6Localhost(),
	}
	dnsConf := &dnsConfig{
		BindHosts: bindHosts,
		Port:      53,
		PendingRequests: &pendingRequests{
			Enabled: false,
		},
	}
	extTLSConf := &aghtls.ExtendedTLSConfig{
		Enabled:         true,
		PortDNSOverTLS:  853,
		PortDNSOverQUIC: 853,
	}

	conf, err := newServerConfig(
		dnsConf,
		&clientSourcesConfig{},
		&doHConfig{},
		newTestTLSConfigProvider(extTLSConf),
		&aghtest.Registrar{},
		nil, // clientsContainer
		&aghtest.ConfigModifier{},
	)
	require.NoError(t, err)
	require.Len(t, conf.TLSConf.TLSListenAddrs, len(bindHosts))
	require.Len(t, conf.TLSConf.QUICListenAddrs, len(bindHosts))

	for i, host := range bindHosts {
		assert.Equal(t, host.String(), conf.TLSConf.TLSListenAddrs[i].IP.String())
		assert.Equal(t, host.String(), conf.TLSConf.QUICListenAddrs[i].IP.String())
	}
}
