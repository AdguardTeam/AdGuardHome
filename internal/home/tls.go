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

// tlsConfig is the TLS configuration and status response.
type tlsConfig struct {
	*tlsConfigStatus     `json:",inline"`
	tlsConfigSettingsExt `json:",inline"`
}

// tlsConfigSettingsExt is used to (un)marshal PrivateKeySaved field and
// ServePlainDNS field.
type tlsConfigSettingsExt struct {
	tlsConfigSettings `json:",inline"`

	// PrivateKeySaved is true if the private key is saved as a string and omit
	// key from answer.  It is used to ensure that clients don't send and
	// receive previously saved private keys.
	PrivateKeySaved bool `yaml:"-" json:"private_key_saved"`

	// ServePlainDNS defines if plain DNS is allowed for incoming requests.  It
	// is an [aghalg.NullBool] to be able to tell when it's set without using
	// pointers.
	ServePlainDNS aghalg.NullBool `yaml:"-" json:"serve_plain_dns"`
}

// confFromTLSSettings converts the TLS settings to the TLS configuration
// version.  s must not be nil.
func confFromTLSSettings(s *tlsConfigSettings) (conf *aghtls.ExtendedTLSConfig) {
	conf = &aghtls.ExtendedTLSConfig{
		PrivateKeyPath:       s.PrivateKeyPath,
		PrivateKey:           s.PrivateKey,
		ServerName:           s.ServerName,
		DNSCryptConfigFile:   s.DNSCryptConfigFile,
		CertificatePath:      s.CertificatePath,
		CertificateChain:     s.CertificateChain,
		OverrideTLSCiphers:   slices.Clone(s.OverrideTLSCiphers),
		CertificateChainData: slices.Clone(s.CertificateChainData),
		PrivateKeyData:       slices.Clone(s.PrivateKeyData),
		PortDNSCrypt:         s.PortDNSCrypt,
		PortDNSOverQUIC:      s.PortDNSOverQUIC,
		PortDNSOverTLS:       s.PortDNSOverTLS,
		PortHTTPS:            s.PortHTTPS,
		Enabled:              s.Enabled,
		ForceHTTPS:           s.ForceHTTPS,
		StrictSNICheck:       s.StrictSNICheck,
		ServePlainDNS:        s.ServePlainDNS,
	}

	conf.Status = aghtls.TLSConfigStatus{
		Subject:           s.Status.Subject,
		Issuer:            s.Status.Issuer,
		KeyType:           s.Status.KeyType,
		NotBefore:         s.Status.NotBefore,
		NotAfter:          s.Status.NotAfter,
		WarningValidation: s.Status.WarningValidation,
		DNSNames:          slices.Clone(s.Status.DNSNames),
		ValidCert:         s.Status.ValidCert,
		ValidChain:        s.Status.ValidChain,
		ValidKey:          s.Status.ValidKey,
		ValidPair:         s.Status.ValidPair,
	}

	return conf
}

// confToTLSSettings converts the TLS configuration to the TLS settings.  conf
// must not be nil.
func confToTLSSettings(conf *aghtls.ExtendedTLSConfig) (s tlsConfigSettings) {
	return tlsConfigSettings{
		PrivateKeyPath:       conf.PrivateKeyPath,
		PrivateKey:           conf.PrivateKey,
		ServerName:           conf.ServerName,
		DNSCryptConfigFile:   conf.DNSCryptConfigFile,
		CertificatePath:      conf.CertificatePath,
		CertificateChain:     conf.CertificateChain,
		OverrideTLSCiphers:   slices.Clone(conf.OverrideTLSCiphers),
		CertificateChainData: slices.Clone(conf.CertificateChainData),
		PrivateKeyData:       slices.Clone(conf.PrivateKeyData),
		Status:               *tlsConfigStatusFromConf(&conf.Status),
		PortDNSCrypt:         conf.PortDNSCrypt,
		PortDNSOverQUIC:      conf.PortDNSOverQUIC,
		PortDNSOverTLS:       conf.PortDNSOverTLS,
		PortHTTPS:            conf.PortHTTPS,
		Enabled:              conf.Enabled,
		ForceHTTPS:           conf.ForceHTTPS,
		StrictSNICheck:       conf.StrictSNICheck,
		ServePlainDNS:        conf.ServePlainDNS,
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
