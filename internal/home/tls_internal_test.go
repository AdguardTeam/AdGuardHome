package home

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"path/filepath"
	"testing"
	"time"

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

// Paths to the test TLS-related data.
const (
	testCertificatePath = "./testdata/cert.pem"
	testPrivateKeyPath  = "./testdata/key.pem"
)

func TestValidateCertificates(t *testing.T) {
	ctx := testutil.ContextWithTimeout(t, testTimeout)

	m, err := newTLSManager(ctx, &tlsManagerConfig{
		logger:        testLogger,
		confModifier:  agh.EmptyConfigModifier{},
		manager:       aghtls.EmptyManager{},
		servePlainDNS: false,
	})
	require.NoError(t, err)

	t.Run("bad_certificate", func(t *testing.T) {
		status := &tlsConfigStatus{}
		err = m.validateCertificates(ctx, status, []byte("bad cert"), nil, "")
		testutil.AssertErrorMsg(t, "empty certificate", err)
		assert.False(t, status.ValidCert)
		assert.False(t, status.ValidChain)
	})

	t.Run("bad_private_key", func(t *testing.T) {
		status := &tlsConfigStatus{}
		err = m.validateCertificates(ctx, status, nil, []byte("bad priv key"), "")
		testutil.AssertErrorMsg(t, "no valid keys were found", err)
		assert.False(t, status.ValidKey)
	})

	t.Run("valid", func(t *testing.T) {
		status := &tlsConfigStatus{}

		testCertChainData := readFile(t, testCertificatePath)
		testPrivateKeyData := readFile(t, testPrivateKeyPath)

		err = m.validateCertificates(ctx, status, testCertChainData, testPrivateKeyData, "")
		assert.Error(t, err)

		notBefore := time.Date(2019, 2, 27, 9, 24, 23, 0, time.UTC)
		notAfter := time.Date(2046, 7, 14, 9, 24, 23, 0, time.UTC)

		assert.True(t, status.ValidCert)
		assert.False(t, status.ValidChain)
		assert.True(t, status.ValidKey)
		assert.Equal(t, "RSA", status.KeyType)
		assert.Equal(t, "CN=AdGuard Home,O=AdGuard Ltd", status.Subject)
		assert.Equal(t, "CN=AdGuard Home,O=AdGuard Ltd", status.Issuer)
		assert.Equal(t, notBefore, status.NotBefore)
		assert.Equal(t, notAfter, status.NotAfter)
		assert.True(t, status.ValidPair)
	})

	t.Run("no_ip_in_cert", func(t *testing.T) {
		caCert, chainPEM, leafKeyPEM := newCertWithoutIP(t)

		m.rootCerts = x509.NewCertPool()
		m.rootCerts.AddCert(caCert)

		status := &tlsConfigStatus{}
		var ok bool
		ok, err = m.validateCertificate(ctx, status, chainPEM, "")
		assert.True(t, ok)
		assert.ErrorIs(t, err, errNoIPInCert)
		assert.True(t, status.ValidCert)
		assert.True(t, status.ValidChain)

		status = &tlsConfigStatus{}
		err = m.validateCertificates(ctx, status, chainPEM, leafKeyPEM, "")
		assert.ErrorIs(t, err, errNoIPInCert)
		assert.True(t, status.ValidCert)
		assert.True(t, status.ValidChain)
		assert.True(t, status.ValidKey)
		assert.True(t, status.ValidPair)
	})
}

// newCertWithoutIP generates a CA certificate, a leaf certificate without an IP
// address, and the PEM-encoded leaf private key.
func newCertWithoutIP(tb testing.TB) (
	caCert *x509.Certificate,
	chainPEM []byte,
	leafKeyPEM []byte,
) {
	tb.Helper()

	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(tb, err)

	now := time.Now()
	caTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}

	caDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &caKey.PublicKey, caKey)
	require.NoError(tb, err)

	caCert, err = x509.ParseCertificate(caDER)
	require.NoError(tb, err)

	leafKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(tb, err)

	leafTmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}

	leafDER, err := x509.CreateCertificate(
		rand.Reader,
		leafTmpl,
		caTmpl,
		&leafKey.PublicKey,
		caKey,
	)
	require.NoError(tb, err)

	buf := bytes.Buffer{}
	err = pem.Encode(&buf, &pem.Block{Type: "CERTIFICATE", Bytes: leafDER})
	require.NoError(tb, err)

	err = pem.Encode(&buf, &pem.Block{Type: "CERTIFICATE", Bytes: caDER})
	require.NoError(tb, err)

	leafKeyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(leafKey),
	})

	return caCert, buf.Bytes(), leafKeyPEM
}

// newCertAndKey is a helper function that generates certificate and key.
func newCertAndKey(tb testing.TB, n int64) (certDER []byte, key *rsa.PrivateKey) {
	tb.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(tb, err)

	certTmpl := &x509.Certificate{
		SerialNumber: big.NewInt(n),
	}

	certDER, err = x509.CreateCertificate(rand.Reader, certTmpl, certTmpl, &key.PublicKey, key)
	require.NoError(tb, err)

	return certDER, key
}

// writeCertAndKey is a helper function that writes certificate and key to
// specified paths.  key must not be nil.
func writeCertAndKey(
	tb testing.TB,
	certDER []byte,
	certPath string,
	key *rsa.PrivateKey,
	keyPath string,
) {
	tb.Helper()

	certFile, err := os.OpenFile(certPath, os.O_WRONLY|os.O_CREATE, 0o600)
	require.NoError(tb, err)

	defer func() {
		err = certFile.Close()
		require.NoError(tb, err)
	}()

	err = pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	require.NoError(tb, err)

	keyFile, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE, 0o600)
	require.NoError(tb, err)

	defer func() {
		err = keyFile.Close()
		require.NoError(tb, err)
	}()

	err = pem.Encode(keyFile, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})
	require.NoError(tb, err)
}

// assertCertSerialNumber is a helper function that checks serial number of the
// TLS certificate.
func assertCertSerialNumber(tb testing.TB, conf *tlsConfigSettings, wantSN int64) {
	tb.Helper()

	cert, err := tls.X509KeyPair(conf.CertificateChainData, conf.PrivateKeyData)
	require.NoError(tb, err)

	assert.Equal(tb, wantSN, cert.Leaf.SerialNumber.Int64())
}

func TestTLSManager_Reload(t *testing.T) {
	storeGlobals(t)

	config.DNS.Port = 0

	var (
		ctx = testutil.ContextWithTimeout(t, testTimeout)
		err error
	)

	globalContext.dnsServer, err = dnsforward.NewServer(dnsforward.DNSCreateParams{
		Logger: testLogger,
	})
	require.NoError(t, err)

	globalContext.clients.storage, err = client.NewStorage(ctx, &client.StorageConfig{
		BaseLogger: testLogger,
		Logger:     testLogger,
		Clock:      timeutil.SystemClock{},
	})
	require.NoError(t, err)

	const (
		snBefore int64 = 1
		snAfter  int64 = 2
	)

	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "cert.pem")
	keyPath := filepath.Join(tmpDir, "key.pem")

	certDER, key := newCertAndKey(t, snBefore)
	writeCertAndKey(t, certDER, certPath, key, keyPath)

	m, err := newTLSManager(ctx, &tlsManagerConfig{
		logger:       testLogger,
		confModifier: agh.EmptyConfigModifier{},
		manager:      aghtls.EmptyManager{},
		tlsSettings: tlsConfigSettings{
			Enabled:         true,
			CertificatePath: certPath,
			PrivateKeyPath:  keyPath,
		},
		servePlainDNS: false,
	})
	require.NoError(t, err)

	web := newTestWeb(t, &webConfig{})
	m.setWebAPI(web)

	extTLSconf := m.extendedTLSConfig()
	assertCertSerialNumber(t, extTLSconf, snBefore)

	certDER, key = newCertAndKey(t, snAfter)
	writeCertAndKey(t, certDER, certPath, key, keyPath)

	m.reload(ctx)

	// The [tlsManager.reload] method will start the DNS server and it should be
	// stopped after the test ends.
	testutil.CleanupAndRequireSuccess(t, func() (err error) {
		return globalContext.dnsServer.Stop(testutil.ContextWithTimeout(t, testTimeout))
	})

	extTLSconf = m.extendedTLSConfig()
	assertCertSerialNumber(t, extTLSconf, snAfter)
}

func TestTLSManager_HandleTLSStatus(t *testing.T) {
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

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/control/tls/status", nil)
	m.handleTLSStatus(w, r)

	res := &tlsConfigSettingsExt{}
	err = json.NewDecoder(w.Body).Decode(res)
	require.NoError(t, err)

	certChainData := readFile(t, testCertificatePath)
	wantCertificateChain := base64.StdEncoding.EncodeToString(certChainData)

	assert.True(t, res.Enabled)
	assert.True(t, res.PrivateKeySaved)
	assert.Equal(t, wantCertificateChain, res.CertificateChain)
}

func TestValidateTLSSettings(t *testing.T) {
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

	web := newTestWeb(t, &webConfig{})
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
			err = m.validateTLSSettings(tc.setts)
			testutil.AssertErrorMsg(t, tc.wantErr, err)
		})
	}
}

func TestTLSManager_HandleTLSValidate(t *testing.T) {
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

	web := newTestWeb(t, &webConfig{})
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
	m.handleTLSValidate(w, r)

	res := &tlsConfigStatus{}
	err = json.NewDecoder(w.Body).Decode(res)
	require.NoError(t, err)

	testCertChainData := readFile(t, testCertificatePath)
	testPrivateKeyData := readFile(t, testPrivateKeyPath)

	cert, err := tls.X509KeyPair(testCertChainData, testPrivateKeyData)
	require.NoError(t, err)

	wantIssuer := cert.Leaf.Issuer.String()
	assert.Equal(t, wantIssuer, res.Issuer)
}

// readFile reads the file at the specified path and returns its content.
//
// TODO(m.kazantsev):  Move to golibs/testutil.
func readFile(tb testing.TB, path string) (data []byte) {
	tb.Helper()

	file, err := os.Open(path)
	require.NoError(tb, err)
	defer func() {
		require.NoError(tb, file.Close())
	}()

	data, err = io.ReadAll(file)
	require.NoError(tb, err)

	return data
}

func TestTLSManager_HandleTLSConfigure(t *testing.T) {
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
		})
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
		},
		servePlainDNS: true,
	})
	require.NoError(t, err)

	web := newTestWeb(t, &webConfig{})
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
	m.handleTLSConfigure(w, r)

	// The [tlsManager.handleTLSConfigure] method will start the DNS server and
	// it should be stopped after the test ends.
	testutil.CleanupAndRequireSuccess(t, func() (err error) {
		return globalContext.dnsServer.Stop(testutil.ContextWithTimeout(t, testTimeout))
	})

	res := &tlsConfig{
		tlsConfigStatus: &tlsConfigStatus{},
	}

	err = json.NewDecoder(w.Body).Decode(res)
	require.NoError(t, err)

	testCertChainData := readFile(t, testCertificatePath)
	testPrivateKeyData := readFile(t, testPrivateKeyPath)

	cert, err := tls.X509KeyPair(testCertChainData, testPrivateKeyData)
	require.NoError(t, err)

	wantIssuer := cert.Leaf.Issuer.String()
	assert.Equal(t, wantIssuer, res.tlsConfigStatus.Issuer)

	// Assert that the Web API's TLS configuration has been updated.
	//
	// TODO(s.chzhen):  Remove when [httpsServer.cond] is removed.
	assert.Eventually(t, func() bool {
		web.httpsServer.condLock.Lock()
		defer web.httpsServer.condLock.Unlock()

		cert = web.httpsServer.cert
		if cert.Leaf == nil {
			return false
		}

		assert.Equal(t, wantIssuer, cert.Leaf.Issuer.String())

		return true
	}, testTimeout, testTimeout/10)
}
