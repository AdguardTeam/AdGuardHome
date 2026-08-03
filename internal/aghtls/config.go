package aghtls

import (
	"slices"
	"time"
)

// ExtendedTLSConfig is the TLS configuration for DNS-over-TLS, DNS-over-QUIC,
// and HTTPS.  When adding new properties, update the [ExtendedTLSConfig.clone]
// and [ExtendedTLSConfig.setPrivateFieldsAndCompare] methods as necessary.
type ExtendedTLSConfig struct {
	PrivateKeyPath       string          `yaml:"private_key_path" json:"private_key_path"`
	PrivateKey           string          `yaml:"private_key" json:"private_key"`
	ServerName           string          `yaml:"server_name" json:"server_name,omitempty"`
	DNSCryptConfigFile   string          `yaml:"dnscrypt_config_file" json:"dnscrypt_config_file"`
	CertificatePath      string          `yaml:"certificate_path" json:"certificate_path"`
	CertificateChain     string          `yaml:"certificate_chain" json:"certificate_chain"`
	OverrideTLSCiphers   []string        `yaml:"override_tls_ciphers,omitempty" json:"-"`
	CertificateChainData []byte          `yaml:"-" json:"-"`
	PrivateKeyData       []byte          `yaml:"-" json:"-"`
	Status               TLSConfigStatus `yaml:"-" json:"-"`
	PortDNSCrypt         uint16          `yaml:"port_dnscrypt" json:"port_dnscrypt"`
	PortDNSOverQUIC      uint16          `yaml:"port_dns_over_quic" json:"port_dns_over_quic,omitempty"`
	PortDNSOverTLS       uint16          `yaml:"port_dns_over_tls" json:"port_dns_over_tls,omitempty"`
	PortHTTPS            uint16          `yaml:"port_https" json:"port_https,omitempty"`
	Enabled              bool            `yaml:"enabled" json:"enabled"`
	ForceHTTPS           bool            `yaml:"force_https" json:"force_https"`
	StrictSNICheck       bool            `yaml:"strict_sni_check" json:"-"`
	ServePlainDNS        bool            `yaml:"-" json:"-"`
}

// TLSConfigStatus contains the status of a certificate chain and key pair.
type TLSConfigStatus struct {
	// Subject is the subject of the first certificate in the chain.
	Subject string `json:"subject,omitempty"`

	// Issuer is the issuer of the first certificate in the chain.
	Issuer string `json:"issuer,omitempty"`

	// KeyType is the type of the private key.
	KeyType string `json:"key_type,omitempty"`

	// NotBefore is the NotBefore field of the first certificate in the chain.
	NotBefore time.Time `json:"not_before"`

	// NotAfter is the NotAfter field of the first certificate in the chain.
	NotAfter time.Time `json:"not_after"`

	// WarningValidation is a validation warning message with the issue
	// description.
	WarningValidation string `json:"warning_validation,omitempty"`

	// DNSNames is the value of SubjectAltNames field of the first certificate
	// in the chain.
	DNSNames []string `json:"dns_names"`

	// ValidCert is true if the specified certificate chain is a valid chain of
	// X509 certificates.
	ValidCert bool `json:"valid_cert"`

	// ValidChain is true if the specified certificate chain is verified and
	// issued by a known CA.
	ValidChain bool `json:"valid_chain"`

	// ValidKey is true if the key is a valid private key.
	ValidKey bool `json:"valid_key"`

	// ValidPair is true if both certificate and private key are correct for
	// each other.
	ValidPair bool `json:"valid_pair"`
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
