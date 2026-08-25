package aghtls

import (
	"slices"
	"time"
)

// ExtendedTLSConfig is the TLS configuration for DNS-over-TLS, DNS-over-QUIC,
// and HTTPS.  When adding new properties, update the [ExtendedTLSConfig.Clone]
// and [setPrivateFieldsAndCompare] methods as necessary.
//
// TODO(m.kazantsev):  Add documentation for each field.
type ExtendedTLSConfig struct {
	PrivateKeyPath       string
	PrivateKey           string
	ServerName           string
	DNSCryptConfigFile   string
	CertificatePath      string
	CertificateChain     string
	OverrideTLSCiphers   []string
	CertificateChainData []byte
	PrivateKeyData       []byte
	Status               TLSConfigStatus
	PortDNSCrypt         uint16
	PortDNSOverQUIC      uint16
	PortDNSOverTLS       uint16
	PortHTTPS            uint16

	// Enabled indicates whether encrypted protocols are enabled. The encrypted
	// protocols are: DNS-over-TLS, DNS-over-QUIC, HTTPS and DNSCrypt.  When set
	// to false, none of the encrypted protocols are started, the HTTPS redirect
	// is disabled and certificate reloading is skipped.  When true, the
	// configured encrypted protocols are attempted to be started, but this does
	// not guarantee that the TLS configuration is valid or complete.
	Enabled bool

	ForceHTTPS     bool
	StrictSNICheck bool
	ServePlainDNS  bool
}

// TLSConfigStatus contains the status of a certificate chain and key pair.
type TLSConfigStatus struct {
	// Subject is the subject of the first certificate in the chain.
	Subject string

	// Issuer is the issuer of the first certificate in the chain.
	Issuer string

	// KeyType is the type of the private key.
	KeyType string

	// NotBefore is the NotBefore field of the first certificate in the chain.
	NotBefore time.Time

	// NotAfter is the NotAfter field of the first certificate in the chain.
	NotAfter time.Time

	// WarningValidation is a validation warning message with the issue
	// description.
	WarningValidation string

	// DNSNames is the value of SubjectAltNames field of the first certificate
	// in the chain.
	DNSNames []string

	// ValidCert is true if the specified certificate chain is a valid chain of
	// X509 certificates.
	ValidCert bool

	// ValidChain is true if the specified certificate chain is verified and
	// issued by a known CA.
	ValidChain bool

	// ValidKey is true if the key is a valid private key.
	ValidKey bool

	// ValidPair is true if both certificate and private key are correct for
	// each other.
	ValidPair bool
}

// Clone returns a deep copy of the [ExtendedTLSConfig].  It is safe to modify
// the returned value without affecting the original.
func (c *ExtendedTLSConfig) Clone() (clone *ExtendedTLSConfig) {
	clone = &ExtendedTLSConfig{}
	*clone = *c

	clone.OverrideTLSCiphers = slices.Clone(c.OverrideTLSCiphers)
	clone.CertificateChainData = slices.Clone(c.CertificateChainData)
	clone.PrivateKeyData = slices.Clone(c.PrivateKeyData)

	clone.Status.DNSNames = slices.Clone(c.Status.DNSNames)

	return clone
}

// UsesPrivilegedPorts returns true if the extended TLS configuration use
// privileged ports.  c must not be nil.
func (c *ExtendedTLSConfig) UsesPrivilegedPorts(maxPrivilegedPort uint16) (ok bool) {
	return c.Enabled && (c.PortHTTPS < maxPrivilegedPort ||
		c.PortDNSOverTLS < maxPrivilegedPort ||
		c.PortDNSOverQUIC < maxPrivilegedPort)
}
