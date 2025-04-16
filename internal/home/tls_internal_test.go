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
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghalg"
	"github.com/AdguardTeam/AdGuardHome/internal/client"
	"github.com/AdguardTeam/AdGuardHome/internal/dnsforward"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO(s.chzhen):  Consider moving to testdata.
var testCertChainData = []byte(`-----BEGIN CERTIFICATE-----
MIICKzCCAZSgAwIBAgIJAMT9kPVJdM7LMA0GCSqGSIb3DQEBCwUAMC0xFDASBgNV
BAoMC0FkR3VhcmQgTHRkMRUwEwYDVQQDDAxBZEd1YXJkIEhvbWUwHhcNMTkwMjI3
MDkyNDIzWhcNNDYwNzE0MDkyNDIzWjAtMRQwEgYDVQQKDAtBZEd1YXJkIEx0ZDEV
MBMGA1UEAwwMQWRHdWFyZCBIb21lMIGfMA0GCSqGSIb3DQEBAQUAA4GNADCBiQKB
gQCwvwUnPJiOvLcOaWmGu6Y68ksFr13nrXBcsDlhxlXy8PaohVi3XxEmt2OrVjKW
QFw/bdV4fZ9tdWFAVRRkgeGbIZzP7YBD1Ore/O5SQ+DbCCEafvjJCcXQIrTeKFE6
i9G3aSMHs0Pwq2LgV8U5mYotLrvyFiE8QPInJbDDMpaFYwIDAQABo1MwUTAdBgNV
HQ4EFgQUdLUmQpEqrhn4eKO029jYd2AAZEQwHwYDVR0jBBgwFoAUdLUmQpEqrhn4
eKO029jYd2AAZEQwDwYDVR0TAQH/BAUwAwEB/zANBgkqhkiG9w0BAQsFAAOBgQB8
LwlXfbakf7qkVTlCNXgoY7RaJ8rJdPgOZPoCTVToEhT6u/cb1c2qp8QB0dNExDna
b0Z+dnODTZqQOJo6z/wIXlcUrnR4cQVvytXt8lFn+26l6Y6EMI26twC/xWr+1swq
Muj4FeWHVDerquH4yMr1jsYLD3ci+kc5sbIX6TfVxQ==
-----END CERTIFICATE-----`)

var testPrivateKeyData = []byte(`-----BEGIN PRIVATE KEY-----
MIICeAIBADANBgkqhkiG9w0BAQEFAASCAmIwggJeAgEAAoGBALC/BSc8mI68tw5p
aYa7pjrySwWvXeetcFywOWHGVfLw9qiFWLdfESa3Y6tWMpZAXD9t1Xh9n211YUBV
FGSB4ZshnM/tgEPU6t787lJD4NsIIRp++MkJxdAitN4oUTqL0bdpIwezQ/CrYuBX
xTmZii0uu/IWITxA8iclsMMyloVjAgMBAAECgYEAmjzoG1h27UDkIlB9BVWl95TP
QVPLB81D267xNFDnWk1Lgr5zL/pnNjkdYjyjgpkBp1yKyE4gHV4skv5sAFWTcOCU
QCgfPfUn/rDFcxVzAdJVWAa/CpJNaZgjTPR8NTGU+Ztod+wfBESNCP5tbnuw0GbL
MuwdLQJGbzeJYpsNysECQQDfFHYoRNfgxHwMbX24GCoNZIgk12uDmGTA9CS5E+72
9t3V1y4CfXxSkfhqNbd5RWrUBRLEw9BKofBS7L9NMDKDAkEAytQoIueE1vqEAaRg
a3A1YDUekKesU5wKfKfKlXvNgB7Hwh4HuvoQS9RCvVhf/60Dvq8KSu6hSjkFRquj
FQ5roQJBAMwKwyiCD5MfJPeZDmzcbVpiocRQ5Z4wPbffl9dRTDnIA5AciZDthlFg
An/jMjZSMCxNl6UyFcqt5Et1EGVhuFECQQCZLXxaT+qcyHjlHJTMzuMgkz1QFbEp
O5EX70gpeGQMPDK0QSWpaazg956njJSDbNCFM4BccrdQbJu1cW4qOsfBAkAMgZuG
O88slmgTRHX4JGFmy3rrLiHNI2BbJSuJ++Yllz8beVzh6NfvuY+HKRCmPqoBPATU
kXS9jgARhhiWXJrk
-----END PRIVATE KEY-----`)

func TestValidateCertificates(t *testing.T) {
	ctx := testutil.ContextWithTimeout(t, testTimeout)
	logger := slogutil.NewDiscardLogger()

	m, err := newTLSManager(ctx, &tlsManagerConfig{
		logger:         logger,
		configModified: func() {},
		servePlainDNS:  false,
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
}

// storeGlobals is a test helper function that saves global variables and
// restores them once the test is complete.
//
// The global variables are:
//   - [configuration.dns]
//   - [homeContext.clients.storage]
//   - [homeContext.dnsServer]
//   - [homeContext.mux]
//
// TODO(s.chzhen):  Remove this once the TLS manager no longer accesses global
// variables.  Make tests that use this helper concurrent.
func storeGlobals(tb testing.TB) {
	tb.Helper()

	prevConfig := config
	storage := globalContext.clients.storage
	dnsServer := globalContext.dnsServer
	mux := globalContext.mux

	tb.Cleanup(func() {
		config = prevConfig
		globalContext.clients.storage = storage
		globalContext.dnsServer = dnsServer
		globalContext.mux = mux
	})
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
// specified paths.
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

	var (
		logger = slogutil.NewDiscardLogger()
		ctx    = testutil.ContextWithTimeout(t, testTimeout)
		err    error
	)

	globalContext.dnsServer, err = dnsforward.NewServer(dnsforward.DNSCreateParams{
		Logger: logger,
	})
	require.NoError(t, err)

	globalContext.clients.storage, err = client.NewStorage(ctx, &client.StorageConfig{
		Logger: logger,
		Clock:  timeutil.SystemClock{},
	})
	require.NoError(t, err)

	globalContext.mux = http.NewServeMux()

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
		logger:         logger,
		configModified: func() {},
		tlsSettings: tlsConfigSettings{
			Enabled:         true,
			CertificatePath: certPath,
			PrivateKeyPath:  keyPath,
		},
		servePlainDNS: false,
	})
	require.NoError(t, err)

	web, err := initWeb(ctx, options{}, nil, nil, logger, nil, false)
	require.NoError(t, err)

	m.setWebAPI(web)

	conf := m.config()
	assertCertSerialNumber(t, conf, snBefore)

	certDER, key = newCertAndKey(t, snAfter)
	writeCertAndKey(t, certDER, certPath, key, keyPath)

	m.reload(ctx)

	conf = m.config()
	assertCertSerialNumber(t, conf, snAfter)
}

func TestTLSManager_HandleTLSStatus(t *testing.T) {
	var (
		logger = slogutil.NewDiscardLogger()
		ctx    = testutil.ContextWithTimeout(t, testTimeout)
		err    error
	)

	m, err := newTLSManager(ctx, &tlsManagerConfig{
		logger:         logger,
		configModified: func() {},
		tlsSettings: tlsConfigSettings{
			Enabled:          true,
			CertificateChain: string(testCertChainData),
			PrivateKey:       string(testPrivateKeyData),
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

	wantCertificateChain := base64.StdEncoding.EncodeToString(testCertChainData)
	assert.True(t, res.Enabled)
	assert.Equal(t, wantCertificateChain, res.CertificateChain)
	assert.True(t, res.PrivateKeySaved)
}

func TestValidateTLSSettings(t *testing.T) {
	storeGlobals(t)

	globalContext.mux = http.NewServeMux()

	var (
		logger = slogutil.NewDiscardLogger()
		ctx    = testutil.ContextWithTimeout(t, testTimeout)
		err    error
	)

	m, err := newTLSManager(ctx, &tlsManagerConfig{
		logger:         logger,
		configModified: func() {},
		servePlainDNS:  false,
	})
	require.NoError(t, err)

	web, err := initWeb(ctx, options{}, nil, nil, logger, nil, false)
	require.NoError(t, err)

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

	globalContext.mux = http.NewServeMux()

	var (
		logger = slogutil.NewDiscardLogger()
		ctx    = testutil.ContextWithTimeout(t, testTimeout)
		err    error
	)

	m, err := newTLSManager(ctx, &tlsManagerConfig{
		logger:         logger,
		configModified: func() {},
		tlsSettings: tlsConfigSettings{
			Enabled:          true,
			CertificateChain: string(testCertChainData),
			PrivateKey:       string(testPrivateKeyData),
		},
		servePlainDNS: false,
	})
	require.NoError(t, err)

	web, err := initWeb(ctx, options{}, nil, nil, logger, nil, false)
	require.NoError(t, err)

	m.setWebAPI(web)

	setts := &tlsConfigSettingsExt{
		tlsConfigSettings: tlsConfigSettings{
			Enabled:          true,
			CertificateChain: base64.StdEncoding.EncodeToString(testCertChainData),
			PrivateKey:       base64.StdEncoding.EncodeToString(testPrivateKeyData),
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

	cert, err := tls.X509KeyPair(testCertChainData, testPrivateKeyData)
	require.NoError(t, err)

	wantIssuer := cert.Leaf.Issuer.String()
	assert.Equal(t, wantIssuer, res.Issuer)
}

func TestTLSManager_HandleTLSConfigure(t *testing.T) {
	// Store the global state before making any changes.
	storeGlobals(t)

	var (
		logger = slogutil.NewDiscardLogger()
		ctx    = testutil.ContextWithTimeout(t, testTimeout)
		err    error
	)

	globalContext.dnsServer, err = dnsforward.NewServer(dnsforward.DNSCreateParams{
		Logger: logger,
	})
	require.NoError(t, err)

	err = globalContext.dnsServer.Prepare(&dnsforward.ServerConfig{
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
		Logger: logger,
		Clock:  timeutil.SystemClock{},
	})
	require.NoError(t, err)

	globalContext.mux = http.NewServeMux()

	config.DNS.BindHosts = []netip.Addr{netip.MustParseAddr("127.0.0.1")}
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
		logger:         logger,
		configModified: func() {},
		tlsSettings: tlsConfigSettings{
			Enabled:         true,
			CertificatePath: certPath,
			PrivateKeyPath:  keyPath,
		},
		servePlainDNS: true,
	})
	require.NoError(t, err)

	web, err := initWeb(ctx, options{}, nil, nil, logger, nil, false)
	require.NoError(t, err)

	m.setWebAPI(web)

	conf := m.config()
	assertCertSerialNumber(t, conf, wantSerialNumber)

	// Prepare a request with the new TLS configuration.
	setts := &tlsConfigSettingsExt{
		tlsConfigSettings: tlsConfigSettings{
			Enabled:          true,
			PortHTTPS:        4433,
			CertificateChain: base64.StdEncoding.EncodeToString(testCertChainData),
			PrivateKey:       base64.StdEncoding.EncodeToString(testPrivateKeyData),
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
	testutil.CleanupAndRequireSuccess(t, globalContext.dnsServer.Stop)

	res := &tlsConfig{
		tlsConfigStatus: &tlsConfigStatus{},
	}
	err = json.NewDecoder(w.Body).Decode(res)
	require.NoError(t, err)

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
