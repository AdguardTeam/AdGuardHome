package aghtls

import (
	"context"
	"crypto/tls"
	"crypto/x509"

	"github.com/AdguardTeam/AdGuardHome/internal/aghalg"
)

// TLSConfigProvider provides TLS configuration to consumers.  Implementations
// must be safe for concurrent use.
//
// TODO(m.kazantsev):  Merge with the Manager interface.
type TLSConfigProvider interface {
	// TLSConfig returns a clone of the current TLS configuration.  conf
	// provides its certificates via GetConfigForClient method.
	TLSConfig() (conf *tls.Config)

	// RootCAs returns the current root CA pool.
	RootCAs() (root *x509.CertPool)

	// HasIPAddrs returns true if the current TLS configuration has at least one
	// certificate with an IP address in its SAN extension.
	HasIPAddrs() (ok bool)

	// ExtendedTLSConfig returns a clone of the current extended TLS
	// configuration.
	ExtendedTLSConfig() (conf *ExtendedTLSConfig)

	// SetExtendedTLSConfig updates the current extended TLS configuration.  It
	// returns true if the configuration was changed.  servePlainDNS is used to
	// determine whether to serve DNS over plain UDP/TCP.
	SetExtendedTLSConfig(
		ctx context.Context,
		servePlainDNS aghalg.NullBool,
		conf *ExtendedTLSConfig,
	) (changed bool, err error)
}

// type check
var _ TLSConfigProvider = EmptyTLSConfigProvider{}

// EmptyTLSConfigProvider is the implementation of the [TLSConfigProvider]
// interface that does nothing.
type EmptyTLSConfigProvider struct{}

// TLSConfig implements the [TLSConfigProvider] interface for
// EmptyTLSConfigProvider.  It always returns nil.
func (EmptyTLSConfigProvider) TLSConfig() (conf *tls.Config) {
	return nil
}

// RootCAs implements the [TLSConfigProvider] interface for
// EmptyTLSConfigProvider.  It always returns nil.
func (EmptyTLSConfigProvider) RootCAs() (root *x509.CertPool) {
	return nil
}

// HasIPAddrs implements the [TLSConfigProvider] interface for
// EmptyTLSConfigProvider.  It always returns false.
func (EmptyTLSConfigProvider) HasIPAddrs() (ok bool) {
	return false
}

// ExtendedTLSConfig implements the [TLSConfigProvider] interface for
// EmptyTLSConfigProvider.  It always returns nil.
func (EmptyTLSConfigProvider) ExtendedTLSConfig() (conf *ExtendedTLSConfig) {
	return nil
}

// SetExtendedTLSConfig implements the [TLSConfigProvider] interface for
// EmptyTLSConfigProvider.  It always returns false and nil.
func (EmptyTLSConfigProvider) SetExtendedTLSConfig(
	_ context.Context,
	_ aghalg.NullBool,
	_ *ExtendedTLSConfig,
) (changed bool, err error) {
	return false, nil
}
