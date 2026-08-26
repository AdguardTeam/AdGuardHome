package aghtls

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testTimeout is the common timeout for tests and contexts.
const testTimeout = 1 * time.Second

// testLogger is a logger used in tests.
var testLogger = slogutil.NewDiscardLogger()

// newCertWithIP generates a self-signed certificate with an IP address in its
// SAN extension and returns the PEM-encoded certificate and private key.
func newCertWithIP(tb testing.TB) (certPEM, keyPEM []byte) {
	tb.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(tb, err)

	now := time.Now()
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		IPAddresses:  []net.IP{net.ParseIP("192.0.2.1")},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(tb, err)

	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM = pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})

	return certPEM, keyPEM
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

// Paths to the test TLS-related data.
const (
	testCertificatePath = "./testdata/cert.pem"
	testPrivateKeyPath  = "./testdata/key.pem"
)

func TestValidateCertificates(t *testing.T) {
	ctx := testutil.ContextWithTimeout(t, testTimeout)

	tlsManager := &DefaultManager{}

	var err error
	t.Run("bad_certificate", func(t *testing.T) {
		status := &TLSConfigStatus{}
		err = validateCertificates(
			ctx,
			testLogger,
			tlsManager,
			status,
			[]byte("bad cert"),
			nil,
			"",
		)
		testutil.AssertErrorMsg(t, "empty certificate", err)
		assert.False(t, status.ValidCert)
		assert.False(t, status.ValidChain)
	})

	t.Run("bad_private_key", func(t *testing.T) {
		status := &TLSConfigStatus{}
		err = validateCertificates(
			ctx,
			testLogger,
			tlsManager,
			status,
			nil,
			[]byte("bad priv key"),
			"",
		)
		testutil.AssertErrorMsg(t, "no valid keys were found", err)
		assert.False(t, status.ValidKey)
	})

	t.Run("valid", func(t *testing.T) {
		status := &TLSConfigStatus{}

		testCertChainData := requireReadFile(t, testCertificatePath)
		testPrivateKeyData := requireReadFile(t, testPrivateKeyPath)

		err = validateCertificates(
			ctx,
			testLogger,
			tlsManager,
			status,
			testCertChainData,
			testPrivateKeyData,
			"",
		)
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

		pool := x509.NewCertPool()
		pool.AddCert(caCert)

		tlsManager = &DefaultManager{rootCerts: pool}

		status := &TLSConfigStatus{}
		var ok bool
		ok, err = validateCertificate(ctx, testLogger, pool, status, chainPEM, "")
		assert.True(t, ok)
		assert.ErrorIs(t, err, errNoIPInCert)
		assert.True(t, status.ValidCert)
		assert.True(t, status.ValidChain)

		status = &TLSConfigStatus{}
		err = validateCertificates(
			ctx,
			testLogger,
			tlsManager,
			status,
			chainPEM,
			leafKeyPEM,
			"",
		)
		assert.ErrorIs(t, err, errNoIPInCert)
		assert.True(t, status.ValidCert)
		assert.True(t, status.ValidChain)
		assert.True(t, status.ValidKey)
		assert.True(t, status.ValidPair)
	})
}

func TestDefaultManager_reload(t *testing.T) {
	ctx := testutil.ContextWithTimeout(t, testTimeout)

	const (
		snBefore int64 = 1
		snAfter  int64 = 2
	)

	tmpDir := t.TempDir()
	certPath := filepath.Join(tmpDir, "cert.pem")
	keyPath := filepath.Join(tmpDir, "key.pem")

	certDER, key := newCertAndKey(t, snBefore)
	writeCertAndKey(t, certDER, certPath, key, keyPath)

	m, err := NewDefaultManager(ctx, &DefaultManagerConfig{
		Logger:  testLogger,
		Watcher: aghos.EmptyFSWatcher{},
		ExtendedTLSConfig: &ExtendedTLSConfig{
			Enabled:         true,
			CertificatePath: certPath,
			PrivateKeyPath:  keyPath,
		},
	})
	require.NoError(t, err)

	extTLSConf := m.ExtendedTLSConfig()
	assertCertSerialNumber(t, extTLSConf, snBefore)

	certDER, key = newCertAndKey(t, snAfter)
	writeCertAndKey(t, certDER, certPath, key, keyPath)

	m.reload(ctx)

	extTLSConf = m.ExtendedTLSConfig()
	assertCertSerialNumber(t, extTLSConf, snAfter)
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
func assertCertSerialNumber(tb testing.TB, conf *ExtendedTLSConfig, wantSN int64) {
	tb.Helper()

	cert, err := tls.X509KeyPair(conf.CertificateChainData, conf.PrivateKeyData)
	require.NoError(tb, err)

	assert.Equal(tb, wantSN, cert.Leaf.SerialNumber.Int64())
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
