package aghtls

import (
	"crypto/tls"
	"testing"

	"github.com/AdguardTeam/golibs/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultManager_HasIPAddrs(t *testing.T) {
	t.Parallel()

	_, noIPChainPEM, noIPKeyPEM := newCertWithoutIP(t)
	ipChainPEM, ipKeyPEM := newCertWithIP(t)

	noIPSettings := &ExtendedTLSConfig{
		Enabled:          true,
		CertificateChain: string(noIPChainPEM),
		PrivateKey:       string(noIPKeyPEM),
	}
	ipSettings := &ExtendedTLSConfig{
		Enabled:          true,
		CertificateChain: string(ipChainPEM),
		PrivateKey:       string(ipKeyPEM),
	}

	testCases := []struct {
		want                 assert.BoolAssertionFunc
		settings             *ExtendedTLSConfig
		name                 string
		certificateChainData []byte
		privateKeyData       []byte
	}{{
		name:     "no_ip_in_cert",
		settings: noIPSettings,
		want:     assert.False,
	}, {
		name:     "has_ip_in_cert",
		settings: ipSettings,
		want:     assert.True,
	}, {
		name:                 "updated_to_ip",
		settings:             noIPSettings,
		certificateChainData: ipChainPEM,
		privateKeyData:       ipKeyPEM,
		want:                 assert.True,
	}, {
		name:                 "updated_to_no_ip",
		settings:             ipSettings,
		certificateChainData: noIPChainPEM,
		privateKeyData:       noIPKeyPEM,
		want:                 assert.False,
	}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Do not run in parallel because the test modifies the TLS
			// manager's state.
			ctx := testutil.ContextWithTimeout(t, testTimeout)

			m, err := NewDefaultManager(ctx, &DefaultManagerConfig{
				Logger:            testLogger,
				ExtendedTLSConfig: tc.settings,
			})
			require.NoError(t, err)

			if tc.certificateChainData == nil && tc.privateKeyData == nil {
				tc.want(t, m.HasIPAddrs())

				return
			}

			func() {
				m.mu.Lock()
				defer m.mu.Unlock()

				var cert tls.Certificate
				cert, err = tls.X509KeyPair(tc.certificateChainData, tc.privateKeyData)
				require.NoError(t, err)

				m.tlsCert = &cert
			}()

			tc.want(t, m.HasIPAddrs())
		})
	}
}
