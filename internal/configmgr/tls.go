package configmgr

import (
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/validate"
)

// TLSConfig is the on-disk TLS configuration for DNS-over-TLS, DNS-over-QUIC,
// and HTTPS.
type TLSConfig struct {
	// CertificateChain is the PEM-encoded certificate chain.  Must be empty if
	// [TLSConfig.CertificatePath] is provided.
	CertificateChain string `yaml:"certificate_chain"`

	// CertificatePath is the path to the certificate file.  Must be empty if
	// [TLSConfig.CertificateChain] is provided.
	CertificatePath string `yaml:"certificate_path"`

	// DNSCryptConfigFile is the path to the DNSCrypt config file.  It must be
	// set if [TLSConfig.PortDNSCrypt] is not zero.
	//
	// See https://github.com/AdguardTeam/dnsproxy and
	// https://github.com/AdguardTeam/dnscrypt.
	DNSCryptConfigFile string `yaml:"dnscrypt_config_file"`

	// ServerName is the hostname of the HTTPS/TLS server.
	ServerName string `yaml:"server_name"`

	// PrivateKey is the PEM-encoded private key.  Must be empty if
	// [TLSConfig.PrivateKeyPath] is provided.
	PrivateKey string `yaml:"private_key"`

	// PrivateKeyPath is the path to the private key file.  Must be empty if
	// [TLSConfig.PrivateKey] is provided.
	PrivateKeyPath string `yaml:"private_key_path"`

	// OverrideTLSCiphers, when set, contains the names of the cipher suites to
	// use.  If the slice is empty, the default safe suites are used.
	OverrideTLSCiphers []string `yaml:"override_tls_ciphers,omitempty"`

	// PortDNSCrypt is the port for DNSCrypt requests.  If it's zero, DNSCrypt
	// is disabled.
	PortDNSCrypt uint16 `yaml:"port_dnscrypt"`

	// PortDNSOverQUIC is the DNS-over-QUIC port.  If 0, DoQ will be disabled.
	PortDNSOverQUIC uint16 `yaml:"port_dns_over_quic"`

	// PortDNSOverTLS is the DNS-over-TLS port.  If 0, DoT will be disabled.
	PortDNSOverTLS uint16 `yaml:"port_dns_over_tls"`

	// PortHTTPS is the HTTPS port.  If 0, HTTPS will be disabled.
	PortHTTPS uint16 `yaml:"port_https"`

	// Enabled indicates whether encryption (DoT/DoH/HTTPS) is enabled.
	Enabled bool `yaml:"enabled"`

	// ForceHTTPS, if true, forces an HTTP to HTTPS redirect.
	ForceHTTPS bool `yaml:"force_https"`

	// StrictSNICheck controls if the connections with SNI mismatching the
	// certificate's ones should be rejected.
	StrictSNICheck bool `yaml:"strict_sni_check"`
}

// type check
var _ validate.Interface = (*TLSConfig)(nil)

// Validate implements the [validate.Interface] interface for *TLSConfig.
func (c *TLSConfig) Validate() (err error) {
	if c == nil {
		return errors.ErrNoValue
	}

	// TODO(d.kolyshev):  Add more validations.

	return nil
}
