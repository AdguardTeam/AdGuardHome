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
	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/AdGuardHome/internal/aghtls"
	"github.com/AdguardTeam/AdGuardHome/internal/client"
	"github.com/AdguardTeam/AdGuardHome/internal/dnsforward"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/AdguardTeam/golibs/timeutil"
	"github.com/quic-go/quic-go/http3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Paths to the test TLS-related data.
const (
	testCertificatePath = "./testdata/cert.pem"
	testPrivateKeyPath  = "./testdata/key.pem"
)

func TestGetBindAddr(t *testing.T) {
	testCases := []struct {
		name        string
		network     string
		addr        netip.Addr
		wantNetwork string
		wantAddr    string
	}{{
		name:        "ipv4_unspecified",
		network:     "tcp",
		addr:        netip.IPv4Unspecified(),
		wantNetwork: "tcp4",
		wantAddr:    "0.0.0.0:443",
	}, {
		name:        "ipv6_unspecified",
		network:     "tcp",
		addr:        netip.IPv6Unspecified(),
		wantNetwork: "tcp",
		wantAddr:    ":443",
	}, {
		name:        "ipv4",
		network:     "tcp",
		addr:        netutil.IPv4Localhost(),
		wantNetwork: "tcp",
		wantAddr:    "127.0.0.1:443",
	}, {
		name:        "ipv6",
		network:     "tcp",
		addr:        netutil.IPv6Localhost(),
		wantNetwork: "tcp",
		wantAddr:    "[::1]:443",
	}, {
		name:        "udp_ipv4_unspecified",
		network:     "udp",
		addr:        netip.IPv4Unspecified(),
		wantNetwork: "udp4",
		wantAddr:    "0.0.0.0:443",
	}, {
		name:        "udp_ipv6_unspecified",
		network:     "udp",
		addr:        netip.IPv6Unspecified(),
		wantNetwork: "udp",
		wantAddr:    ":443",
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			network, addrStr := getBindAddr(tc.network, tc.addr, 443)

			assert.Equal(t, tc.wantNetwork, network)
			assert.Equal(t, tc.wantAddr, addrStr)
		})
	}
}

// canDial returns true if a connection to addr can be established using
// network.
func canDial(t *testing.T, network, addr string) (ok bool) {
	t.Helper()

	conn, err := net.DialTimeout(network, addr, testTimeout)
	if err != nil {
		return false
	}

	require.NoError(t, conn.Close())

	return true
}

// TestGetBindAddr_families is a regression test for the bind-scope
// compatibility of the listeners created using [getBindAddr].  It verifies on
// a real listener that the explicitly configured unspecified IPv4 address
// remains IPv4-only, while the unspecified IPv6 address enables dual-stack
// listening.
func TestGetBindAddr_families(t *testing.T) {
	ln6, err := net.Listen("tcp6", "[::1]:0")
	if err != nil {
		t.Skipf("skipping: IPv6 seems unsupported: %v", err)
	}
	require.NoError(t, ln6.Close())

	testCases := []struct {
		name     string
		addr     netip.Addr
		wantIPv4 bool
		wantIPv6 bool
	}{{
		name:     "ipv4_unspecified",
		addr:     netip.IPv4Unspecified(),
		wantIPv4: true,
		wantIPv6: false,
	}, {
		name:     "ipv6_unspecified",
		addr:     netip.IPv6Unspecified(),
		wantIPv4: true,
		wantIPv6: true,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			network, addrStr := getBindAddr("tcp", tc.addr, 0)

			ln, lErr := net.Listen(network, addrStr)
			require.NoError(t, lErr)
			testutil.CleanupAndRequireSuccess(t, ln.Close)

			tcpAddr := testutil.RequireTypeAssert[*net.TCPAddr](t, ln.Addr())
			port := uint16(tcpAddr.Port)

			assert.Equal(t, tc.wantIPv4, canDial(t, "tcp4", netutil.JoinHostPort("127.0.0.1", port)))
			assert.Equal(t, tc.wantIPv6, canDial(t, "tcp6", netutil.JoinHostPort("::1", port)))
		})
	}
}

// TestServeHTTP3_connClose is a regression test that checks that the packet
// connection owned by [serveHTTP3] is closed when the HTTP/3 server is shut
// down, since [http3.Server.Serve] does not close connections provided by the
// caller, and an unclosed connection would make rebinding the same address
// fail with EADDRINUSE, e.g. on a TLS reconfiguration.
func TestServeHTTP3_connClose(t *testing.T) {
	certDER, key := newCertAndKey(t, 1)
	srv := &http3.Server{
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{{
				Certificate: [][]byte{certDER},
				PrivateKey:  key,
			}},
			MinVersion: tls.VersionTLS12,
		},
		Handler: http.NewServeMux(),
	}

	addrStr, served := startServeHTTP3(t, srv)

	require.NoError(t, srv.Close())

	srvErr, _ := testutil.RequireReceive(t, served, testTimeout)
	assert.ErrorIs(t, srvErr, http.ErrServerClosed)

	// The address must be available again after the server is closed.
	conn, err := net.ListenPacket("udp", addrStr)
	require.NoError(t, err)
	require.NoError(t, conn.Close())
}

// startServeHTTP3 reserves a free UDP address on the IPv4 loopback, starts
// srv on it using [serveHTTP3] in a separate goroutine, and waits until the
// address is bound.  Since a concurrent listener may take the reserved
// address before [serveHTTP3] binds it, the reservation is retried with a
// fresh address in that case.  srv must not be nil.
func startServeHTTP3(t *testing.T, srv *http3.Server) (addrStr string, served chan error) {
	t.Helper()

	const maxAttempts = 5

	ctx := testutil.ContextWithTimeout(t, testTimeout)

	var lastErr error
	for range maxAttempts {
		conn, err := net.ListenPacket("udp", "127.0.0.1:0")
		require.NoError(t, err)

		addrStr = conn.LocalAddr().String()
		require.NoError(t, conn.Close())

		served = make(chan error, 1)
		go func(addr string, ch chan error) {
			ch <- serveHTTP3(ctx, testLogger, srv, "udp", addr)
		}(addrStr, served)

		deadline := time.Now().Add(testTimeout)

	wait:
		for time.Now().Before(deadline) {
			select {
			case lastErr = <-served:
				// The reserved address has been taken by a concurrent
				// listener, so [serveHTTP3] returned early.  Retry with a
				// fresh address.
				break wait
			default:
			}

			c, lErr := net.ListenPacket("udp", addrStr)
			if lErr != nil {
				// The address is bound by the server.
				return addrStr, served
			}

			require.NoError(t, c.Close())

			time.Sleep(testTimeout / 100)
		}
	}

	t.Fatalf("http/3 server did not bind after %d attempts: %v", maxAttempts, lastErr)

	// Generally unreachable.
	return "", nil
}

func TestWebAPI_HandleTLSConfigure(t *testing.T) {
	// Store the global state before making any changes.
	storeGlobals(t)

	var (
		ctx = testutil.ContextWithTimeout(t, testTimeout)
		err error
	)

	globalContext.dnsServer, err = dnsforward.NewServer(dnsforward.DNSCreateParams{
		Logger:     testLogger,
		TLSManager: aghtls.EmptyManager{},
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
	ctx = testutil.ContextWithTimeout(t, testTimeout)
	m, err := aghtls.NewDefaultManager(ctx, &aghtls.DefaultManagerConfig{
		Logger:  testLogger,
		Watcher: aghos.EmptyFSWatcher{},
		ExtendedTLSConfig: &aghtls.ExtendedTLSConfig{
			Enabled:         true,
			CertificatePath: certPath,
			PrivateKeyPath:  keyPath,
			ServePlainDNS:   true,
		},
		ServePlainDNS: true,
	})
	require.NoError(t, err)

	web := newTestWeb(t, &webConfig{tlsManager: m})

	extTLSConf := m.ExtendedTLSConfig()
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
		tlsConfigStatus:      &tlsConfigStatus{},
		tlsConfigSettingsExt: tlsConfigSettingsExt{},
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
		tlsConf := m.TLSConfig()

		var tlsCert *tls.Certificate
		tlsCert, err = tlsConf.GetCertificate(&tls.ClientHelloInfo{})
		if err != nil || tlsCert.Leaf == nil {
			return false
		}

		return tlsCert.Leaf.Issuer.String() == wantIssuer
	}, testTimeout, testTimeout/10)
}

func TestWebAPI_HandleTLSStatus(t *testing.T) {
	testCertChain := requireReadFile(t, testCertificatePath)
	testPrivateKeyData := requireReadFile(t, testPrivateKeyPath)

	ctx := testutil.ContextWithTimeout(t, testTimeout)
	m, err := aghtls.NewDefaultManager(ctx, &aghtls.DefaultManagerConfig{
		Logger:  testLogger,
		Watcher: aghos.EmptyFSWatcher{},
		ExtendedTLSConfig: &aghtls.ExtendedTLSConfig{
			Enabled:          true,
			CertificateChain: string(testCertChain),
			PrivateKey:       string(testPrivateKeyData),
		},
	})
	require.NoError(t, err)

	web := newTestWeb(t, &webConfig{tlsManager: m})

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

	ctx := testutil.ContextWithTimeout(t, testTimeout)
	m, err := aghtls.NewDefaultManager(ctx, &aghtls.DefaultManagerConfig{
		Logger: testLogger,
	})
	require.NoError(t, err)

	web := newTestWeb(t, &webConfig{tlsManager: m})

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
		setts   *tlsConfigSettingsExt
	}{{
		name:    "basic",
		wantErr: "",
		setts:   &tlsConfigSettingsExt{},
	}, {
		name:    "disabled_all",
		wantErr: "plain DNS is required in case encryption protocols are disabled",
		setts: &tlsConfigSettingsExt{
			ServePlainDNS: aghalg.NBFalse,
		},
	}, {
		name:    "busy_https_port",
		wantErr: fmt.Sprintf("port %d for HTTPS is not available", busyTCPPort),
		setts: &tlsConfigSettingsExt{
			tlsConfigSettings: tlsConfigSettings{
				Enabled:   true,
				PortHTTPS: uint16(busyTCPPort),
			},
		},
	}, {
		name:    "busy_dot_port",
		wantErr: fmt.Sprintf("port %d for DNS-over-TLS is not available", busyTCPPort),
		setts: &tlsConfigSettingsExt{
			tlsConfigSettings: tlsConfigSettings{
				Enabled:        true,
				PortDNSOverTLS: uint16(busyTCPPort),
			},
		},
	}, {
		name:    "busy_doq_port",
		wantErr: fmt.Sprintf("port %d for DNS-over-QUIC is not available", busyUDPPort),
		setts: &tlsConfigSettingsExt{
			tlsConfigSettings: tlsConfigSettings{
				Enabled:         true,
				PortDNSOverQUIC: uint16(busyUDPPort),
			},
		},
	}, {
		name:    "duplicate_port",
		wantErr: "validating tcp ports: duplicated values: [4433]",
		setts: &tlsConfigSettingsExt{
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

	ctx := testutil.ContextWithTimeout(t, testTimeout)
	m, err := aghtls.NewDefaultManager(ctx, &aghtls.DefaultManagerConfig{
		Logger:  testLogger,
		Watcher: aghos.EmptyFSWatcher{},
		ExtendedTLSConfig: &aghtls.ExtendedTLSConfig{
			Enabled:         true,
			CertificatePath: testCertificatePath,
			PrivateKeyPath:  testPrivateKeyPath,
		},
	})
	require.NoError(t, err)

	web := newTestWeb(t, &webConfig{tlsManager: m})

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
func assertCertSerialNumber(tb testing.TB, conf *aghtls.ExtendedTLSConfig, wantSN int64) {
	tb.Helper()

	cert, err := tls.X509KeyPair(conf.CertificateChainData, conf.PrivateKeyData)
	require.NoError(tb, err)

	assert.Equal(tb, wantSN, cert.Leaf.SerialNumber.Int64())
}
