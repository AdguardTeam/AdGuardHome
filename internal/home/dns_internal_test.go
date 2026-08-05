package home

import (
	"bytes"
	"crypto/ed25519"
	"net/netip"
	"os"
	"path/filepath"
	"testing"

	"github.com/AdguardTeam/AdGuardHome/internal/agh"
	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/aghtls"
	"github.com/AdguardTeam/AdGuardHome/internal/dnsforward"
	"github.com/AdguardTeam/dnscrypt"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "go.yaml.in/yaml/v4"
)

func TestNewServerConfigCertlessEncryption(t *testing.T) {
	privateKey := ed25519.NewKeyFromSeed(bytes.Repeat([]byte{1}, ed25519.SeedSize))
	dnsCryptConf := &dnscrypt.ResolverConfig{
		ProviderName: dnscrypt.DNSCryptV2Prefix + "example.org",
		PublicKey:    dnscrypt.HexEncodeKey(privateKey.Public().(ed25519.PublicKey)),
		PrivateKey:   dnscrypt.HexEncodeKey(privateKey),
		ResolverSk:   dnscrypt.HexEncodeKey(bytes.Repeat([]byte{2}, dnscrypt.KeySize)),
		ResolverPk:   dnscrypt.HexEncodeKey(bytes.Repeat([]byte{3}, dnscrypt.KeySize)),
		ESVersion:    dnscrypt.XSalsa20Poly1305,
	}

	dnsCryptData, err := yaml.Marshal(dnsCryptConf)
	require.NoError(t, err)

	dnsCryptPath := filepath.Join(t.TempDir(), "dnscrypt.yaml")
	require.NoError(t, os.WriteFile(dnsCryptPath, dnsCryptData, 0o600))

	ctx := testutil.ContextWithTimeout(t, testTimeout)
	tlsMgr, err := newTLSManager(ctx, &tlsManagerConfig{
		logger:       testLogger,
		confModifier: agh.EmptyConfigModifier{},
		manager:      aghtls.EmptyManager{},
		tlsSettings: tlsConfigSettings{
			Enabled:            true,
			PortHTTPS:          5443,
			PortDNSCrypt:       5444,
			PortDNSOverTLS:     5445,
			PortDNSOverQUIC:    5446,
			DNSCryptConfigFile: dnsCryptPath,
		},
	})
	require.NoError(t, err)
	assert.Nil(t, tlsMgr.TLSConfig())

	serverConf, err := newServerConfig(
		&dnsConfig{
			BindHosts: []netip.Addr{netutil.IPv4Localhost()},
			Config: dnsforward.Config{
				UpstreamMode:     dnsforward.UpstreamModeLoadBalance,
				EDNSClientSubnet: &dnsforward.EDNSClientSubnet{},
			},
			ServePlainDNS:   true,
			PendingRequests: &pendingRequests{},
		},
		&clientSourcesConfig{},
		tlsMgr.extendedTLSConfig(),
		&doHConfig{InsecureEnabled: true},
		tlsMgr,
		aghhttp.EmptyRegistrar{},
		dnsforward.EmptyClientsContainer{},
		agh.EmptyConfigModifier{},
	)
	require.NoError(t, err)
	require.NotNil(t, serverConf.TLSConf.DNSCryptConf)
	assert.NotEmpty(t, serverConf.TLSConf.HTTPSListenAddrs)
	assert.NotEmpty(t, serverConf.TLSConf.TLSListenAddrs)
	assert.NotEmpty(t, serverConf.TLSConf.QUICListenAddrs)
	assert.True(t, serverConf.TLSAllowUnencryptedDoH)
	assert.Nil(t, tlsMgr.TLSConfig())

	serverConf.AddrProcConf = nil
	dnsSrv, err := dnsforward.NewServer(dnsforward.DNSCreateParams{
		Logger:            testLogger,
		TLSConfigProvider: tlsMgr,
	})
	require.NoError(t, err)
	require.NoError(t, dnsSrv.Prepare(ctx, serverConf))
	dnsSrv.Close(ctx)

	serverConf, err = newServerConfig(
		&dnsConfig{
			BindHosts: []netip.Addr{netutil.IPv4Localhost()},
			Config: dnsforward.Config{
				UpstreamMode:     dnsforward.UpstreamModeLoadBalance,
				EDNSClientSubnet: &dnsforward.EDNSClientSubnet{},
			},
			ServePlainDNS:   true,
			PendingRequests: &pendingRequests{},
		},
		&clientSourcesConfig{},
		tlsMgr.extendedTLSConfig(),
		&doHConfig{},
		tlsMgr,
		aghhttp.EmptyRegistrar{},
		dnsforward.EmptyClientsContainer{},
		agh.EmptyConfigModifier{},
	)
	require.NoError(t, err)
	assert.Empty(t, serverConf.TLSConf.HTTPSListenAddrs)
}

func TestGetDNSEncryptionCertless(t *testing.T) {
	ctx := testutil.ContextWithTimeout(t, testTimeout)
	tlsMgr, err := newTLSManager(ctx, &tlsManagerConfig{
		logger:       testLogger,
		confModifier: agh.EmptyConfigModifier{},
		manager:      aghtls.EmptyManager{},
		tlsSettings: tlsConfigSettings{
			Enabled:         true,
			ServerName:      "dns.example",
			PortHTTPS:       defaultPortHTTPS,
			PortDNSOverTLS:  853,
			PortDNSOverQUIC: 853,
		},
	})
	require.NoError(t, err)
	require.Nil(t, tlsMgr.TLSConfig())

	de := getDNSEncryption(tlsMgr, false)
	assert.Empty(t, de.https)
	assert.Empty(t, de.tls)
	assert.Empty(t, de.quic)

	de = getDNSEncryption(tlsMgr, true)
	assert.Equal(t, "https://dns.example/dns-query", de.https)
	assert.Empty(t, de.tls)
	assert.Empty(t, de.quic)
}
