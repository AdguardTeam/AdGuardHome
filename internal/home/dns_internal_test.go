package home

import (
	"crypto/tls"
	"net/netip"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/aghtest"
	"github.com/AdguardTeam/AdGuardHome/internal/aghtls"
	"github.com/AdguardTeam/AdGuardHome/internal/configmgr"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Encrypted DNS ports used in tests.
const (
	portDoH = 443
	portDoT = 853
	portDoQ = 854
)

func TestNewDNSTLSConfig_DoHRoutes(t *testing.T) {
	t.Parallel()

	hosts := []netip.Addr{netip.MustParseAddr("192.168.1.1")}
	dohAddr := netutil.IPv4Localhost()
	dohAddrPort := netip.AddrPortFrom(dohAddr, portDoH)

	testExtTLSConf := newTestExtTLSConf(t, portDoH)

	testCases := []struct {
		name      string
		ext       *aghtls.ExtendedTLSConfig
		routes    []string
		wantHTTPS []netip.AddrPort
	}{{
		name: "default_routes",
		ext:  testExtTLSConf,
		routes: []string{
			"GET /dns-query",
			"POST /dns-query",
			"GET /dns-query/{ClientID}",
			"POST /dns-query/{ClientID}",
		},
		wantHTTPS: []netip.AddrPort{dohAddrPort},
	}, {
		name:      "empty_routes",
		ext:       testExtTLSConf,
		routes:    []string{},
		wantHTTPS: nil,
	}, {
		name: "custom_routes",
		ext:  testExtTLSConf,
		routes: []string{
			"GET /doh",
			"POST /doh",
		},
		wantHTTPS: []netip.AddrPort{dohAddrPort},
	}, {
		name:      "port_https_zero",
		ext:       newTestExtTLSConf(t, 0),
		routes:    []string{"GET /dns-query"},
		wantHTTPS: nil,
	}, {
		name: "tls_disabled",
		ext: &aghtls.ExtendedTLSConfig{
			PortHTTPS:       portDoH,
			PortDNSOverTLS:  portDoT,
			PortDNSOverQUIC: portDoQ,
		},
		routes:    []string{"GET /dns-query"},
		wantHTTPS: nil,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mgr := &aghtest.Manager{
				OnTLSConfig: func() (conf *tls.Config) {
					return &tls.Config{}
				},
				OnExtendedTLSConfig: func() (conf *aghtls.ExtendedTLSConfig) {
					return tc.ext
				},
			}

			conf, err := newDNSTLSConfig(
				mgr,
				&configmgr.DoHConfig{Routes: tc.routes},
				dohAddr,
				hosts,
			)
			require.NoError(t, err)

			assert.Equal(t, tc.wantHTTPS, conf.HTTPSListenAddrs)

			if tc.ext.Enabled && tc.ext.PortHTTPS == portDoH {
				// DoT and DoQ are unaffected by the DoH routes.
				require.Len(t, conf.TLSListenAddrs, len(hosts))
				assert.Equal(t, portDoT, conf.TLSListenAddrs[0].Port)

				require.Len(t, conf.QUICListenAddrs, len(hosts))
				assert.Equal(t, portDoQ, conf.QUICListenAddrs[0].Port)
			}
		})
	}
}

// newTestExtTLSConf creates a new *ExtendedTLSConfig with the specified HTTPS
// port for testing purposes.
func newTestExtTLSConf(tb testing.TB, portHTTPS uint16) (c *aghtls.ExtendedTLSConfig) {
	tb.Helper()

	return &aghtls.ExtendedTLSConfig{
		Enabled:         true,
		ServerName:      "doh.example",
		PortHTTPS:       portHTTPS,
		PortDNSOverTLS:  portDoT,
		PortDNSOverQUIC: portDoQ,
	}
}
