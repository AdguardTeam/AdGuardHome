package home

import (
	"slices"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghalg"
	"github.com/AdguardTeam/AdGuardHome/internal/aghtls"
)

// tlsConfigStatus contains the status of a certificate chain and key pair.
type tlsConfigStatus struct {
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

// tlsConfigSettings is the TLS configuration for DNS-over-TLS, DNS-over-QUIC,
// and HTTPS.
type tlsConfigSettings struct {
	// CertificateChain is the PEM-encoded certificate chain.  Must be empty if
	// [tlsConfigSettings.CertificatePath] is provided.
	CertificateChain string `json:"certificate_chain"`

	// CertificatePath is the path to the certificate file.  Must be empty if
	// [tlsConfigSettings.CertificateChain] is provided.
	CertificatePath string `json:"certificate_path"`

	// DNSCryptConfigFile is the path to the DNSCrypt config file.  Must be set
	// if PortDNSCrypt is not zero.
	//
	// See https://github.com/AdguardTeam/dnsproxy and
	// https://github.com/AdguardTeam/dnscrypt.
	DNSCryptConfigFile string `json:"dnscrypt_config_file"`

	// ServerName is the hostname of the HTTPS/TLS server.
	ServerName string `json:"server_name,omitempty"`

	// PrivateKey is the PEM-encoded private key.  Must be empty if
	// [tlsConfigSettings.PrivateKeyPath] is provided.
	PrivateKey string `json:"private_key"`

	// PrivateKeyPath is the path to the private key file.  Must be empty if
	// [tlsConfigSettings.PrivateKey] is provided.
	PrivateKeyPath string `json:"private_key_path"`

	// PortDNSCrypt is the port for DNSCrypt requests.  If it's zero, DNSCrypt
	// is disabled.
	PortDNSCrypt uint16 `json:"port_dnscrypt"`

	// PortDNSOverQUIC is the DNS-over-QUIC port.  If 0, DoQ will be disabled.
	PortDNSOverQUIC uint16 `json:"port_dns_over_quic,omitempty"`

	// PortDNSOverTLS is the DNS-over-TLS port.  If 0, DoT will be disabled.
	PortDNSOverTLS uint16 `json:"port_dns_over_tls,omitempty"`

	// PortHTTPS is the HTTPS port.  If 0, HTTPS will be disabled.
	PortHTTPS uint16 `json:"port_https,omitempty"`

	// Enabled indicates whether encryption (DoT/DoH/HTTPS) is enabled.
	Enabled bool `json:"enabled"`

	// ForceHTTPS, if true, forces an HTTP to HTTPS redirect.
	ForceHTTPS bool `json:"force_https"`
}

// tlsConfig is the TLS configuration and status response.
//
// TODO(d.kolyshev): Consider renaming to JSON DTO.
type tlsConfig struct {
	*tlsConfigStatus     `json:",inline"`
	tlsConfigSettingsExt `json:",inline"`
}

// tlsConfigSettingsExt is used to (un)marshal PrivateKeySaved field and
// ServePlainDNS field.
//
// TODO(d.kolyshev): Consider renaming to JSON DTO.
type tlsConfigSettingsExt struct {
	tlsConfigSettings `json:",inline"`

	// PrivateKeySaved is true if the private key is saved as a string and omit
	// key from answer.  It is used to ensure that clients don't send and
	// receive previously saved private keys.
	PrivateKeySaved bool `json:"private_key_saved"`

	// ServePlainDNS defines if plain DNS is allowed for incoming requests.  It
	// is an [aghalg.NullBool] to be able to tell when it's set without using
	// pointers.
	ServePlainDNS aghalg.NullBool `json:"serve_plain_dns"`
}

// confFromTLSSettings converts the TLS settings to the TLS configuration
// version.  It starts from curr, and overrides only the settings that are
// exposed by the HTTP API.  DNSCryptConfigFile and PortDNSCrypt are
// intentionally kept from curr and are not taken from s.  s and curr must not
// be nil.
func confFromTLSSettings(
	s *tlsConfigSettings,
	curr *aghtls.ExtendedTLSConfig,
) (conf *aghtls.ExtendedTLSConfig) {
	conf = curr.Clone()

	conf.PrivateKeyPath = s.PrivateKeyPath
	conf.PrivateKey = s.PrivateKey
	conf.ServerName = s.ServerName
	conf.CertificatePath = s.CertificatePath
	conf.CertificateChain = s.CertificateChain
	conf.PortDNSOverQUIC = s.PortDNSOverQUIC
	conf.PortDNSOverTLS = s.PortDNSOverTLS
	conf.PortHTTPS = s.PortHTTPS
	conf.Enabled = s.Enabled
	conf.ForceHTTPS = s.ForceHTTPS

	return conf
}

// confToTLSSettings converts the TLS configuration to the TLS settings.  conf
// must not be nil.
func confToTLSSettings(conf *aghtls.ExtendedTLSConfig) (s *tlsConfigSettings) {
	return &tlsConfigSettings{
		CertificateChain:   conf.CertificateChain,
		CertificatePath:    conf.CertificatePath,
		DNSCryptConfigFile: conf.DNSCryptConfigFile,
		ServerName:         conf.ServerName,
		PrivateKey:         conf.PrivateKey,
		PrivateKeyPath:     conf.PrivateKeyPath,
		PortDNSCrypt:       conf.PortDNSCrypt,
		PortDNSOverQUIC:    conf.PortDNSOverQUIC,
		PortDNSOverTLS:     conf.PortDNSOverTLS,
		PortHTTPS:          conf.PortHTTPS,
		Enabled:            conf.Enabled,
		ForceHTTPS:         conf.ForceHTTPS,
	}
}

// tlsConfigStatusFromConf converts the TLS configuration status to the TLS
// settings status.  s must not be nil.
func tlsConfigStatusFromConf(s *aghtls.TLSConfigStatus) (status *tlsConfigStatus) {
	return &tlsConfigStatus{
		Subject:           s.Subject,
		Issuer:            s.Issuer,
		KeyType:           s.KeyType,
		NotBefore:         s.NotBefore,
		NotAfter:          s.NotAfter,
		WarningValidation: s.WarningValidation,
		DNSNames:          slices.Clone(s.DNSNames),
		ValidCert:         s.ValidCert,
		ValidChain:        s.ValidChain,
		ValidKey:          s.ValidKey,
		ValidPair:         s.ValidPair,
	}
}
