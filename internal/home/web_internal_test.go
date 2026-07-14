package home

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"path/filepath"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/agh"
	"github.com/AdguardTeam/AdGuardHome/internal/aghalg"
	"github.com/AdguardTeam/AdGuardHome/internal/aghtls"
	"github.com/AdguardTeam/AdGuardHome/internal/client"
	"github.com/AdguardTeam/AdGuardHome/internal/dnsforward"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWebAPI_HandleTLSConfigure(t *testing.T) {
	// Store the global state before making any changes.
	storeGlobals(t)

	var (
		ctx = testutil.ContextWithTimeout(t, testTimeout)
		err error
	)

	globalContext.dnsServer, err = dnsforward.NewServer(dnsforward.DNSCreateParams{
		Logger: testLogger,
	})
	require.NoError(t, err)

	err = globalContext.dnsServer.Prepare(
		testutil.ContextWithTimeout(t, testTimeout),
		&dnsforward.ServerConfig{
			TLSConf: &dnsforward.TLSConfig{},
			Config: dnsforward.Config{
				UpstreamMode:     dnsforward.UpstreamModeLoadBalance,
				EDNSClientSubnet: &dnsforward.EDNSClientSubnet{Enabled: false},
				ClientsContainer: dnsforward.EmptyClientsContainer{},
			},
			ServePlainDNS: true,
		},
	)
	require.NoError(t, err)

	globalContext.clients.storage, err = client.NewStorage(ctx, &client.StorageConfig{
		BaseLogger: testLogger,
		Logger:     testLogger,
		Clock:      timeutil.SystemClock{},
	})
	require.NoError(t, err)

	config.DNS.BindHosts = []netip.Addr{netutil.IPv4Localhost()}
	config.DNS.Port = 0

	const wantSerialNumber int64 = 1

	// Prepare the TLS manager configuration.
	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "cert.pem")
	keyPath := filepath.Join(tmpDir, "key.pem")

	certDER, key := newCertAndKey(t, wantSerialNumber)
	writeCertAndKey(t, certDER, certPath, key, keyPath)

	// Initialize the TLS manager and assert its configuration.
	m, err := newTLSManager(ctx, &tlsManagerConfig{
		logger:       testLogger,
		confModifier: agh.EmptyConfigModifier{},
		manager:      aghtls.EmptyManager{},
		tlsSettings: tlsConfigSettings{
			Enabled:         true,
			CertificatePath: certPath,
			PrivateKeyPath:  keyPath,
			ServePlainDNS:   true,
		},
		servePlainDNS: true,
	})
	require.NoError(t, err)

	web := newTestWeb(t, &webConfig{tlsManager: m})
	m.setWebAPI(web)

	extTLSConf := m.extendedTLSConfig()
	assertCertSerialNumber(t, extTLSConf, wantSerialNumber)

	// Prepare a request with the new TLS configuration.
	setts := &tlsConfigSettingsExt{
		tlsConfigSettings: tlsConfigSettings{
			Enabled:         true,
			PortHTTPS:       4433,
			CertificatePath: testCertificatePath,
			PrivateKeyPath:  testPrivateKeyPath,
		},
	}

	req, err := json.Marshal(setts)
	require.NoError(t, err)

	r := httptest.NewRequest(http.MethodPost, "/control/tls/configure", bytes.NewReader(req))
	w := httptest.NewRecorder()

	// Reconfigure the TLS manager.
	web.handleTLSConfigure(w, r)

	// The [webAPI.handleTLSConfigure] method will start the DNS server and
	// it should be stopped after the test ends.
	testutil.CleanupAndRequireSuccess(t, func() (err error) {
		return globalContext.dnsServer.Stop(testutil.ContextWithTimeout(t, testTimeout))
	})

	res := &tlsConfig{
		tlsConfigStatus: &tlsConfigStatus{},
	}

	err = json.NewDecoder(w.Body).Decode(res)
	require.NoError(t, err)

	testCertChainData := requireReadFile(t, testCertificatePath)
	testPrivateKeyData := requireReadFile(t, testPrivateKeyPath)

	cert, err := tls.X509KeyPair(testCertChainData, testPrivateKeyData)
	require.NoError(t, err)

	wantIssuer := cert.Leaf.Issuer.String()
	assert.Equal(t, wantIssuer, res.tlsConfigStatus.Issuer)

	// Assert that the Web API's TLS configuration has been updated.
	assert.Eventually(t, func() bool {
		cert = web.httpsServer.certificate()
		if cert.Leaf == nil {
			return false
		}

		assert.Equal(t, wantIssuer, cert.Leaf.Issuer.String())

		return true
	}, testTimeout, testTimeout/10)
}

func TestWebAPI_HandleTLSStatus(t *testing.T) {
	var (
		ctx = testutil.ContextWithTimeout(t, testTimeout)
		err error
	)

	testCertChain := requireReadFile(t, testCertificatePath)
	testPrivateKeyData := requireReadFile(t, testPrivateKeyPath)

	m, err := newTLSManager(ctx, &tlsManagerConfig{
		logger:       testLogger,
		confModifier: agh.EmptyConfigModifier{},
		manager:      aghtls.EmptyManager{},
		tlsSettings: tlsConfigSettings{
			Enabled:          true,
			CertificateChain: string(testCertChain),
			PrivateKey:       string(testPrivateKeyData),
		},
		servePlainDNS: false,
	})
	require.NoError(t, err)

	web := newTestWeb(t, &webConfig{tlsManager: m})
	m.setWebAPI(web)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/control/tls/status", nil)
	web.handleTLSStatus(w, r)

	res := &tlsConfigSettingsExt{}
	err = json.NewDecoder(w.Body).Decode(res)
	require.NoError(t, err)

	wantCertificateChain := base64.StdEncoding.EncodeToString(testCertChain)
	assert.True(t, res.Enabled)
	assert.Equal(t, wantCertificateChain, res.CertificateChain)
	assert.True(t, res.PrivateKeySaved)
}

func TestWebAPI_ValidateTLSSettings(t *testing.T) {
	storeGlobals(t)

	var (
		ctx = testutil.ContextWithTimeout(t, testTimeout)
		err error
	)

	m, err := newTLSManager(ctx, &tlsManagerConfig{
		logger:        testLogger,
		confModifier:  agh.EmptyConfigModifier{},
		manager:       aghtls.EmptyManager{},
		servePlainDNS: false,
	})
	require.NoError(t, err)

	web := newTestWeb(t, &webConfig{tlsManager: m})
	m.setWebAPI(web)

	tcpLn, err := net.Listen("tcp", ":0")
	require.NoError(t, err)

	testutil.CleanupAndRequireSuccess(t, tcpLn.Close)

	tcpAddr := testutil.RequireTypeAssert[*net.TCPAddr](t, tcpLn.Addr())
	busyTCPPort := tcpAddr.Port

	udpLn, err := net.ListenPacket("udp", ":0")
	require.NoError(t, err)

	testutil.CleanupAndRequireSuccess(t, udpLn.Close)

	udpAddr := testutil.RequireTypeAssert[*net.UDPAddr](t, udpLn.LocalAddr())
	busyUDPPort := udpAddr.Port

	testCases := []struct {
		name    string
		wantErr string
		setts   tlsConfigSettingsExt
	}{{
		name:    "basic",
		wantErr: "",
		setts:   tlsConfigSettingsExt{},
	}, {
		name:    "disabled_all",
		wantErr: "plain DNS is required in case encryption protocols are disabled",
		setts: tlsConfigSettingsExt{
			ServePlainDNS: aghalg.NBFalse,
		},
	}, {
		name:    "busy_https_port",
		wantErr: fmt.Sprintf("port %d for HTTPS is not available", busyTCPPort),
		setts: tlsConfigSettingsExt{
			tlsConfigSettings: tlsConfigSettings{
				Enabled:   true,
				PortHTTPS: uint16(busyTCPPort),
			},
		},
	}, {
		name:    "busy_dot_port",
		wantErr: fmt.Sprintf("port %d for DNS-over-TLS is not available", busyTCPPort),
		setts: tlsConfigSettingsExt{
			tlsConfigSettings: tlsConfigSettings{
				Enabled:        true,
				PortDNSOverTLS: uint16(busyTCPPort),
			},
		},
	}, {
		name:    "busy_doq_port",
		wantErr: fmt.Sprintf("port %d for DNS-over-QUIC is not available", busyUDPPort),
		setts: tlsConfigSettingsExt{
			tlsConfigSettings: tlsConfigSettings{
				Enabled:         true,
				PortDNSOverQUIC: uint16(busyUDPPort),
			},
		},
	}, {
		name:    "duplicate_port",
		wantErr: "validating tcp ports: duplicated values: [4433]",
		setts: tlsConfigSettingsExt{
			tlsConfigSettings: tlsConfigSettings{
				Enabled:        true,
				PortHTTPS:      4433,
				PortDNSOverTLS: 4433,
			},
		},
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err = web.validateTLSSettings(tc.setts)
			testutil.AssertErrorMsg(t, tc.wantErr, err)
		})
	}
}

func TestWebAPI_HandleTLSValidate(t *testing.T) {
	storeGlobals(t)

	var (
		ctx = testutil.ContextWithTimeout(t, testTimeout)
		err error
	)

	m, err := newTLSManager(ctx, &tlsManagerConfig{
		logger:       testLogger,
		confModifier: agh.EmptyConfigModifier{},
		manager:      aghtls.EmptyManager{},
		tlsSettings: tlsConfigSettings{
			Enabled:         true,
			CertificatePath: testCertificatePath,
			PrivateKeyPath:  testPrivateKeyPath,
		},
		servePlainDNS: false,
	})
	require.NoError(t, err)

	web := newTestWeb(t, &webConfig{tlsManager: m})
	m.setWebAPI(web)

	setts := &tlsConfigSettingsExt{
		tlsConfigSettings: tlsConfigSettings{
			Enabled:         true,
			CertificatePath: testCertificatePath,
			PrivateKeyPath:  testPrivateKeyPath,
		},
	}

	req, err := json.Marshal(setts)
	require.NoError(t, err)

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/control/tls/validate", bytes.NewReader(req))
	web.handleTLSValidate(w, r)

	res := &tlsConfigStatus{}
	err = json.NewDecoder(w.Body).Decode(res)
	require.NoError(t, err)

	testCertChainData := requireReadFile(t, testCertificatePath)
	testPrivateKeyData := requireReadFile(t, testPrivateKeyPath)

	cert, err := tls.X509KeyPair(testCertChainData, testPrivateKeyData)
	require.NoError(t, err)

	wantIssuer := cert.Leaf.Issuer.String()
	assert.Equal(t, wantIssuer, res.Issuer)
}

// requireReadFile reads the file at the specified path and returns its content.
//
// TODO(m.kazantsev):  Move to golibs/testutil.
func requireReadFile(tb testing.TB, path string) (data []byte) {
	tb.Helper()

	data, err := os.ReadFile(path)
	require.NoError(tb, err)

	return data
}
