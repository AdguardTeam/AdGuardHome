package home

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghtls"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
)

// tlsManager contains the current configuration and state of AdGuard Home TLS
// encryption.
type tlsManager struct {
	// mu protects all fields.
	mu *sync.RWMutex

	// certLastMod is the last modification time of the certificate file.
	certLastMod time.Time

	// status is the current status of the configuration.  It is never nil.
	status *tlsConfigStatus

	// conf is the current TLS configuration.
	conf *tlsConfiguration
}

// newTLSManager initializes the TLS configuration.
func newTLSManager(conf *tlsConfiguration) (m *tlsManager, err error) {
	m = &tlsManager{
		status: &tlsConfigStatus{},
		mu:     &sync.RWMutex{},
		conf:   conf,
	}

	if m.conf.Enabled {
		err = m.load()
		if err != nil {
			return nil, err
		}

		m.setCertFileTime()
	}

	return m, nil
}

// confForEncoding returns a partial clone of the current TLS configuration.  It
// is safe for concurrent use.
func (m *tlsManager) confForEncoding() (conf *tlsConfiguration) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.conf.cloneForEncoding()
}

// load reloads the TLS configuration from files or data from the config file.
// m.mu is expected to be locked for writing.
func (m *tlsManager) load() (err error) {
	err = loadTLSConf(m.conf, m.status)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	return nil
}

// WriteDiskConfig - write config
func (m *tlsManager) WriteDiskConfig(conf *tlsConfiguration) {
	*conf = *m.confForEncoding()
}

// setCertFileTime sets t.certLastMod from the certificate.  If there are
// errors, setCertFileTime logs them.  mu is expected to be locked for writing.
func (m *tlsManager) setCertFileTime() {
	if len(m.conf.CertificatePath) == 0 {
		return
	}

	fi, err := os.Stat(m.conf.CertificatePath)
	if err != nil {
		log.Error("tls: looking up certificate path: %s", err)

		return
	}

	m.certLastMod = fi.ModTime().UTC()
}

// start updates the configuration of t and starts it.
func (m *tlsManager) start() {
	m.registerWebHandlers()

	// The background context is used because the TLSConfigChanged wraps context
	// with timeout on its own and shuts down the server, which handles current
	// request.
	Context.web.TLSConfigChanged(context.Background(), m.confForEncoding())
}

// reload updates the configuration and restarts m.
func (m *tlsManager) reload() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.conf.Enabled || len(m.conf.CertificatePath) == 0 {
		return
	}

	fi, err := os.Stat(m.conf.CertificatePath)
	if err != nil {
		log.Error("tls: %s", err)

		return
	}

	if fi.ModTime().UTC().Equal(m.certLastMod) {
		log.Debug("tls: certificate file isn't modified")

		return
	}

	log.Debug("tls: certificate file is modified")

	err = m.load()
	if err != nil {
		log.Error("tls: reloading: %s", err)

		return
	}

	m.certLastMod = fi.ModTime().UTC()

	_ = reconfigureDNSServer()

	// The background context is used because the TLSConfigChanged wraps context
	// with timeout on its own and shuts down the server, which handles current
	// request.
	Context.web.TLSConfigChanged(context.Background(), m.conf)
}

// loadTLSConf loads and validates the TLS configuration.  The returned error is
// also set in status.WarningValidation.
func loadTLSConf(tlsConf *tlsConfiguration, status *tlsConfigStatus) (err error) {
	defer func() {
		if err != nil {
			status.WarningValidation = err.Error()
			if status.ValidCert && status.ValidKey && status.ValidPair {
				// Do not return warnings since those aren't critical.
				err = nil
			}
		}
	}()

	tlsConf.CertificateChainData = []byte(tlsConf.CertificateChain)
	tlsConf.PrivateKeyData = []byte(tlsConf.PrivateKey)

	if tlsConf.CertificatePath != "" {
		err = loadCert(tlsConf)
		if err != nil {
			// Don't wrap the error, since it's informative enough as is.
			return err
		}

		// Set status.ValidCert to true to signal the frontend that the
		// certificate opens successfully while the private key can't be opened.
		status.ValidCert = true
	}

	if tlsConf.PrivateKeyPath != "" {
		err = loadPKey(tlsConf)
		if err != nil {
			// Don't wrap the error, since it's informative enough as is.
			return err
		}

		status.ValidKey = true
	}

	err = validateCertificates(
		status,
		tlsConf.CertificateChainData,
		tlsConf.PrivateKeyData,
		tlsConf.ServerName,
	)
	if err != nil {
		return fmt.Errorf("validating certificate pair: %w", err)
	}

	return nil
}

// loadCert loads the certificate from file, if necessary.
func loadCert(tlsConf *tlsConfiguration) (err error) {
	if tlsConf.CertificateChain != "" {
		return errors.Error("certificate data and file can't be set together")
	}

	tlsConf.CertificateChainData, err = os.ReadFile(tlsConf.CertificatePath)
	if err != nil {
		return fmt.Errorf("reading cert file: %w", err)
	}

	return nil
}

// loadPKey loads the private key from file, if necessary.
func loadPKey(tlsConf *tlsConfiguration) (err error) {
	if tlsConf.PrivateKey != "" {
		return errors.Error("private key data and file cannot be set together")
	}

	tlsConf.PrivateKeyData, err = os.ReadFile(tlsConf.PrivateKeyPath)
	if err != nil {
		return fmt.Errorf("reading key file: %w", err)
	}

	return nil
}

// validateCertChain verifies certs using the first as the main one and others
// as intermediate.  srvName stands for the expected DNS name.
func validateCertChain(certs []*x509.Certificate, srvName string) (err error) {
	main, others := certs[0], certs[1:]

	pool := x509.NewCertPool()
	for _, cert := range others {
		log.Info("tls: got an intermediate cert")
		pool.AddCert(cert)
	}

	opts := x509.VerifyOptions{
		DNSName:       srvName,
		Roots:         Context.tlsRoots,
		Intermediates: pool,
	}
	_, err = main.Verify(opts)
	if err != nil {
		return fmt.Errorf("certificate does not verify: %w", err)
	}

	return nil
}

// parseCertChain parses the certificate chain from raw data, and returns it.
// If ok is true, the returned error, if any, is not critical.
func parseCertChain(chain []byte) (parsedCerts []*x509.Certificate, ok bool, err error) {
	log.Debug("tls: got certificate chain: %d bytes", len(chain))

	var certs []*pem.Block
	for decoded, pemblock := pem.Decode(chain); decoded != nil; {
		if decoded.Type == "CERTIFICATE" {
			certs = append(certs, decoded)
		}

		decoded, pemblock = pem.Decode(pemblock)
	}

	parsedCerts, err = parsePEMCerts(certs)
	if err != nil {
		return nil, false, err
	}

	log.Info("tls: number of certs: %d", len(parsedCerts))

	if !aghtls.CertificateHasIP(parsedCerts[0]) {
		err = errors.Error(`certificate has no IP addresses` +
			`, this may cause issues with DNS-over-TLS clients`)
	}

	return parsedCerts, true, err
}

// parsePEMCerts parses multiple PEM-encoded certificates.
func parsePEMCerts(certs []*pem.Block) (parsedCerts []*x509.Certificate, err error) {
	for i, cert := range certs {
		var parsed *x509.Certificate
		parsed, err = x509.ParseCertificate(cert.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parsing certificate at index %d: %w", i, err)
		}

		parsedCerts = append(parsedCerts, parsed)
	}

	if len(parsedCerts) == 0 {
		return nil, errors.Error("empty certificate")
	}

	return parsedCerts, nil
}

// validatePKey validates the private key, returning its type.  It returns an
// empty string if error occurs.
func validatePKey(pkey []byte) (keyType string, err error) {
	var key *pem.Block

	// Go through all pem blocks, but take first valid pem block and drop the
	// rest.
	for decoded, pemblock := pem.Decode([]byte(pkey)); decoded != nil; {
		if decoded.Type == "PRIVATE KEY" || strings.HasSuffix(decoded.Type, " PRIVATE KEY") {
			key = decoded

			break
		}

		decoded, pemblock = pem.Decode(pemblock)
	}

	if key == nil {
		return "", errors.Error("no valid keys were found")
	}

	_, keyType, err = parsePrivateKey(key.Bytes)
	if err != nil {
		return "", fmt.Errorf("parsing private key: %w", err)
	}

	if keyType == keyTypeED25519 {
		return "", errors.Error(
			"ED25519 keys are not supported by browsers; " +
				"did you mean to use X25519 for key exchange?",
		)
	}

	return keyType, nil
}

// validateCertificates processes certificate data and its private key.  status
// must not be nil, since it's used to accumulate the validation results.  Other
// parameters are optional.
func validateCertificates(
	status *tlsConfigStatus,
	certChain []byte,
	pkey []byte,
	serverName string,
) (err error) {
	// Check only the public certificate separately from the key.
	if len(certChain) > 0 {
		var certs []*x509.Certificate
		certs, status.ValidCert, err = parseCertChain(certChain)
		if !status.ValidCert {
			// Don't wrap the error, since it's informative enough as is.
			return err
		}

		mainCert := certs[0]
		status.Subject = mainCert.Subject.String()
		status.Issuer = mainCert.Issuer.String()
		status.NotAfter = mainCert.NotAfter
		status.NotBefore = mainCert.NotBefore
		status.DNSNames = mainCert.DNSNames

		if chainErr := validateCertChain(certs, serverName); chainErr != nil {
			// Let self-signed certs through and don't return this error to set
			// its message into the status.WarningValidation afterwards.
			err = chainErr
		} else {
			status.ValidChain = true
		}
	}

	// Validate the private key by parsing it.
	if len(pkey) > 0 {
		var keyErr error
		status.KeyType, keyErr = validatePKey(pkey)
		if keyErr != nil {
			// Don't wrap the error, since it's informative enough as is.
			return keyErr
		}

		status.ValidKey = true
	}

	// If both are set, validate together.
	if len(certChain) > 0 && len(pkey) > 0 {
		_, pairErr := tls.X509KeyPair(certChain, pkey)
		if pairErr != nil {
			return fmt.Errorf("certificate-key pair: %w", pairErr)
		}

		status.ValidPair = true
	}

	return err
}

// Key types.
const (
	keyTypeECDSA   = "ECDSA"
	keyTypeED25519 = "ED25519"
	keyTypeRSA     = "RSA"
)

// Attempt to parse the given private key DER block. OpenSSL 0.9.8 generates
// PKCS#1 private keys by default, while OpenSSL 1.0.0 generates PKCS#8 keys.
// OpenSSL ecparam generates SEC1 EC private keys for ECDSA. We try all three.
//
// TODO(a.garipov): Find out if this version of parsePrivateKey from the stdlib
// is actually necessary.
func parsePrivateKey(der []byte) (key crypto.PrivateKey, typ string, err error) {
	if key, err = x509.ParsePKCS1PrivateKey(der); err == nil {
		return key, keyTypeRSA, nil
	}

	if key, err = x509.ParsePKCS8PrivateKey(der); err == nil {
		switch key := key.(type) {
		case *rsa.PrivateKey:
			return key, keyTypeRSA, nil
		case *ecdsa.PrivateKey:
			return key, keyTypeECDSA, nil
		case ed25519.PrivateKey:
			return key, keyTypeED25519, nil
		default:
			return nil, "", fmt.Errorf(
				"tls: found unknown private key type %T in PKCS#8 wrapping",
				key,
			)
		}
	}

	if key, err = x509.ParseECPrivateKey(der); err == nil {
		return key, keyTypeECDSA, nil
	}

	return nil, "", errors.Error("tls: failed to parse private key")
}
