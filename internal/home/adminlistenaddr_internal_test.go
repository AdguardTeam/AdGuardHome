package home

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestAdminListenAddr verifies that [adminListenAddr] returns the configured
// address only when it is valid and has a non-zero port.
func TestAdminListenAddr(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		in   netip.AddrPort
		want netip.AddrPort
	}{{
		name: "zero",
		in:   netip.AddrPort{},
		want: netip.AddrPort{},
	}, {
		name: "zero_port",
		in:   netip.AddrPortFrom(netip.MustParseAddr("127.0.0.1"), 0),
		want: netip.AddrPort{},
	}, {
		name: "loopback_port",
		in:   netip.MustParseAddrPort("127.0.0.1:4443"),
		want: netip.MustParseAddrPort("127.0.0.1:4443"),
	}, {
		name: "unspecified_port",
		in:   netip.MustParseAddrPort("0.0.0.0:8443"),
		want: netip.MustParseAddrPort("0.0.0.0:8443"),
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			conf := &tlsConfigSettings{AdminListenAddr: tc.in}
			assert.Equal(t, tc.want, adminListenAddr(conf))
		})
	}

	t.Run("nil", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, netip.AddrPort{}, adminListenAddr(nil))
	})
}

// TestValidatePorts_adminHTTPS verifies that [validatePorts] includes the
// optional dedicated admin HTTPS port in its uniqueness check.
func TestValidatePorts_adminHTTPS(t *testing.T) {
	t.Parallel()

	// Baseline ports used below.  These are non-overlapping so that the test
	// cases below fail or pass solely based on the admin HTTPS port.
	const (
		bindPort        tcpPort = 3000
		dohPort         tcpPort = 443
		dotPort         tcpPort = 853
		dnscryptTCPPort tcpPort = 5443
		dnsPort         udpPort = 53
		doqPort         udpPort = 784
	)

	testCases := []struct {
		name           string
		adminHTTPSPort tcpPort
		wantErr        bool
	}{{
		name:           "unused",
		adminHTTPSPort: 0,
		wantErr:        false,
	}, {
		name:           "unique",
		adminHTTPSPort: 4443,
		wantErr:        false,
	}, {
		name:           "conflicts_with_doh",
		adminHTTPSPort: dohPort,
		wantErr:        true,
	}, {
		name:           "conflicts_with_webapi",
		adminHTTPSPort: bindPort,
		wantErr:        true,
	}, {
		name:           "conflicts_with_dot",
		adminHTTPSPort: dotPort,
		wantErr:        true,
	}, {
		name:           "conflicts_with_plain_dns",
		adminHTTPSPort: tcpPort(dnsPort),
		wantErr:        true,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := validatePorts(
				bindPort,
				dohPort,
				dotPort,
				dnscryptTCPPort,
				tc.adminHTTPSPort,
				dnsPort,
				doqPort,
			)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestSetPrivateFieldsAndCompare_adminListenAddr verifies that
// [tlsConfigSettings.setPrivateFieldsAndCompare] preserves the server-side
// [tlsConfigSettings.AdminListenAddr] value across updates from the frontend
// and that [cmp.Equal] is able to compare the field.
func TestSetPrivateFieldsAndCompare_adminListenAddr(t *testing.T) {
	t.Parallel()

	const testAddr = "127.0.0.1:4443"

	server := &tlsConfigSettings{
		Enabled:         true,
		PortHTTPS:       443,
		AdminListenAddr: netip.MustParseAddrPort(testAddr),
	}

	// Simulate the frontend sending a new TLS config that does not include
	// AdminListenAddr (since it is not exposed in the UI).  The server-side
	// value must be preserved.
	fromFrontend := &tlsConfigSettings{
		Enabled:   true,
		PortHTTPS: 443,
	}

	equal := server.setPrivateFieldsAndCompare(fromFrontend)
	assert.True(
		t,
		equal,
		"configs should be equal when only AdminListenAddr is missing from the frontend",
	)
	assert.Equal(
		t,
		netip.MustParseAddrPort(testAddr),
		fromFrontend.AdminListenAddr,
		"server-side AdminListenAddr should be copied into frontend payload",
	)

	// If the frontend changes an actual user-editable field, the configs must
	// compare unequal.
	fromFrontend2 := &tlsConfigSettings{
		Enabled:   true,
		PortHTTPS: 8443,
	}

	equal = server.setPrivateFieldsAndCompare(fromFrontend2)
	assert.False(t, equal, "configs should be unequal when PortHTTPS differs")
}

// TestRegisterDoHHandlers_muxSplit verifies that when
// [tlsConfigSettings.AdminListenAddr] is configured, [registerDoHHandlers]
// registers DoH routes on the dedicated DoH mux rather than on the shared
// admin mux.
func TestRegisterDoHHandlers_muxSplit(t *testing.T) {
	// Save and restore the globals touched by this test.  The global
	// [homeContext] cannot be copied by value because it contains mutexes,
	// so save only the single field that is overwritten.
	prevWeb := globalContext.web
	prevConfig := config
	t.Cleanup(func() {
		globalContext.web = prevWeb
		config = prevConfig
	})

	const dohRoute = "/dns-query"

	testCases := []struct {
		name            string
		adminListenAddr netip.AddrPort
		wantOnAdmin     bool
		wantOnDoH       bool
	}{{
		name:            "disabled",
		adminListenAddr: netip.AddrPort{},
		wantOnAdmin:     true,
		wantOnDoH:       false,
	}, {
		name:            "enabled",
		adminListenAddr: netip.MustParseAddrPort("127.0.0.1:4443"),
		wantOnAdmin:     false,
		wantOnDoH:       true,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			config = &configuration{
				TLS: tlsConfigSettings{
					AdminListenAddr: tc.adminListenAddr,
				},
			}

			adminMux := http.NewServeMux()
			dohMux := http.NewServeMux()

			globalContext.web = &webAPI{
				conf: &webAPIConfig{
					mux:    adminMux,
					dohMux: dohMux,
				},
			}

			registerDoHHandlers([]string{dohRoute})

			assert.Equal(t, tc.wantOnAdmin, hasPattern(adminMux, dohRoute))
			assert.Equal(t, tc.wantOnDoH, hasPattern(dohMux, dohRoute))
		})
	}
}

// hasPattern returns true if pattern is registered on mux, as determined by
// [http.ServeMux.Handler].
func hasPattern(mux *http.ServeMux, pattern string) (ok bool) {
	req := httptest.NewRequest(http.MethodGet, pattern, nil)
	_, p := mux.Handler(req)

	return p == pattern
}
