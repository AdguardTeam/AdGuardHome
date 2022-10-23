package home

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghalg"
	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/dnsforward"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/google/go-cmp/cmp"
)

// tlsManager contains the current configuration and state of AdGuard Home TLS
// encryption.
type tlsManager struct {
	// status is the current status of the configuration.  It is never nil.
	status *tlsConfigStatus

	// certLastMod is the last modification time of the certificate file.
	certLastMod time.Time

	confLock sync.Mutex
	conf     tlsConfigSettings
}

// newTLSManager initializes the TLS configuration.
func newTLSManager(conf tlsConfigSettings) (m *tlsManager, err error) {
	m = &tlsManager{
		status: &tlsConfigStatus{},
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

// load reloads the TLS configuration from files or data from the config file.
func (m *tlsManager) load() (err error) {
	err = loadTLSConf(&m.conf, m.status)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	return nil
}

// WriteDiskConfig - write config
func (m *tlsManager) WriteDiskConfig(conf *tlsConfigSettings) {
	m.confLock.Lock()
	*conf = m.conf
	m.confLock.Unlock()
}

// setCertFileTime sets t.certLastMod from the certificate.  If there are
// errors, setCertFileTime logs them.
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

	m.confLock.Lock()
	tlsConf := m.conf
	m.confLock.Unlock()

	// The background context is used because the TLSConfigChanged wraps context
	// with timeout on its own and shuts down the server, which handles current
	// request.
	Context.web.TLSConfigChanged(context.Background(), tlsConf)
}

// reload updates the configuration and restarts t.
func (m *tlsManager) reload() {
	m.confLock.Lock()
	tlsConf := m.conf
	m.confLock.Unlock()

	if !tlsConf.Enabled || len(tlsConf.CertificatePath) == 0 {
		return
	}

	fi, err := os.Stat(tlsConf.CertificatePath)
	if err != nil {
		log.Error("tls: %s", err)

		return
	}

	if fi.ModTime().UTC().Equal(m.certLastMod) {
		log.Debug("tls: certificate file isn't modified")

		return
	}

	log.Debug("tls: certificate file is modified")

	m.confLock.Lock()
	err = m.load()
	m.confLock.Unlock()
	if err != nil {
		log.Error("tls: reloading: %s", err)

		return
	}

	m.certLastMod = fi.ModTime().UTC()

	_ = reconfigureDNSServer()

	m.confLock.Lock()
	tlsConf = m.conf
	m.confLock.Unlock()

	// The background context is used because the TLSConfigChanged wraps context
	// with timeout on its own and shuts down the server, which handles current
	// request.
	Context.web.TLSConfigChanged(context.Background(), tlsConf)
}

// loadTLSConf loads and validates the TLS configuration.  The returned error is
// also set in status.WarningValidation.
func loadTLSConf(tlsConf *tlsConfigSettings, status *tlsConfigStatus) (err error) {
	defer func() {
		if err != nil {
			status.WarningValidation = err.Error()
		}
	}()

	tlsConf.CertificateChainData = []byte(tlsConf.CertificateChain)
	tlsConf.PrivateKeyData = []byte(tlsConf.PrivateKey)

	if tlsConf.CertificatePath != "" {
		if tlsConf.CertificateChain != "" {
			return errors.Error("certificate data and file can't be set together")
		}

		tlsConf.CertificateChainData, err = os.ReadFile(tlsConf.CertificatePath)
		if err != nil {
			return fmt.Errorf("reading cert file: %w", err)
		}

		status.ValidCert = true
	}

	if tlsConf.PrivateKeyPath != "" {
		if tlsConf.PrivateKey != "" {
			return errors.Error("private key data and file can't be set together")
		}

		tlsConf.PrivateKeyData, err = os.ReadFile(tlsConf.PrivateKeyPath)
		if err != nil {
			return fmt.Errorf("reading key file: %w", err)
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

// tlsConfigStatus contains the status of a certificate chain and key pair.
type tlsConfigStatus struct {
	// Subject is the subject of the first certificate in the chain.
	Subject string `json:"subject,omitempty"`

	// Issuer is the issuer of the first certificate in the chain.
	Issuer string `json:"issuer,omitempty"`

	// KeyType is the type of the private key.
	KeyType string `json:"key_type,omitempty"`

	// NotBefore is the NotBefore field of the first certificate in the chain.
	NotBefore time.Time `json:"not_before,omitempty"`

	// NotAfter is the NotAfter field of the first certificate in the chain.
	NotAfter time.Time `json:"not_after,omitempty"`

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

// tlsConfigSettingsExt is used to (un)marshal the PrivateKeySaved field to
// ensure that clients don't send and receive previously saved private keys.
type tlsConfigSettingsExt struct {
	tlsConfigSettings `json:",inline"`

	// PrivateKeySaved is true if the private key is saved as a string and omit
	// key from answer.
	PrivateKeySaved bool `yaml:"-" json:"private_key_saved,inline"`
}

func (m *tlsManager) handleTLSStatus(w http.ResponseWriter, r *http.Request) {
	m.confLock.Lock()
	data := tlsConfig{
		tlsConfigSettingsExt: tlsConfigSettingsExt{
			tlsConfigSettings: m.conf,
		},
		tlsConfigStatus: m.status,
	}
	m.confLock.Unlock()

	marshalTLS(w, r, data)
}

func (m *tlsManager) handleTLSValidate(w http.ResponseWriter, r *http.Request) {
	setts, err := unmarshalTLS(r)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "Failed to unmarshal TLS config: %s", err)

		return
	}

	if setts.PrivateKeySaved {
		setts.PrivateKey = m.conf.PrivateKey
	}

	if setts.Enabled {
		err = validatePorts(
			tcpPort(config.BindPort),
			tcpPort(config.BetaBindPort),
			tcpPort(setts.PortHTTPS),
			tcpPort(setts.PortDNSOverTLS),
			tcpPort(setts.PortDNSCrypt),
			udpPort(config.DNS.Port),
			udpPort(setts.PortDNSOverQUIC),
		)
		if err != nil {
			aghhttp.Error(r, w, http.StatusBadRequest, "%s", err)

			return
		}
	}

	if !webCheckPortAvailable(setts.PortHTTPS) {
		aghhttp.Error(
			r,
			w,
			http.StatusBadRequest,
			"port %d is not available, cannot enable HTTPS on it",
			setts.PortHTTPS,
		)

		return
	}

	// Skip the error check, since we are only interested in the value of
	// status.WarningValidation.
	status := &tlsConfigStatus{}
	_ = loadTLSConf(&setts.tlsConfigSettings, status)
	resp := tlsConfig{
		tlsConfigSettingsExt: setts,
		tlsConfigStatus:      status,
	}

	marshalTLS(w, r, resp)
}

func (m *tlsManager) setConfig(newConf tlsConfigSettings, status *tlsConfigStatus) (restartHTTPS bool) {
	m.confLock.Lock()
	defer m.confLock.Unlock()

	// Reset the DNSCrypt data before comparing, since we currently do not
	// accept these from the frontend.
	//
	// TODO(a.garipov): Define a custom comparer for dnsforward.TLSConfig.
	newConf.DNSCryptConfigFile = m.conf.DNSCryptConfigFile
	newConf.PortDNSCrypt = m.conf.PortDNSCrypt
	if !cmp.Equal(m.conf, newConf, cmp.AllowUnexported(dnsforward.TLSConfig{})) {
		log.Info("tls config has changed, restarting https server")
		restartHTTPS = true
	} else {
		log.Info("tls: config has not changed")
	}

	// Note: don't do just `t.conf = data` because we must preserve all other members of t.conf
	m.conf.Enabled = newConf.Enabled
	m.conf.ServerName = newConf.ServerName
	m.conf.ForceHTTPS = newConf.ForceHTTPS
	m.conf.PortHTTPS = newConf.PortHTTPS
	m.conf.PortDNSOverTLS = newConf.PortDNSOverTLS
	m.conf.PortDNSOverQUIC = newConf.PortDNSOverQUIC
	m.conf.CertificateChain = newConf.CertificateChain
	m.conf.CertificatePath = newConf.CertificatePath
	m.conf.CertificateChainData = newConf.CertificateChainData
	m.conf.PrivateKey = newConf.PrivateKey
	m.conf.PrivateKeyPath = newConf.PrivateKeyPath
	m.conf.PrivateKeyData = newConf.PrivateKeyData
	m.status = status

	return restartHTTPS
}

func (m *tlsManager) handleTLSConfigure(w http.ResponseWriter, r *http.Request) {
	req, err := unmarshalTLS(r)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "Failed to unmarshal TLS config: %s", err)

		return
	}

	if req.PrivateKeySaved {
		req.PrivateKey = m.conf.PrivateKey
	}

	if req.Enabled {
		err = validatePorts(
			tcpPort(config.BindPort),
			tcpPort(config.BetaBindPort),
			tcpPort(req.PortHTTPS),
			tcpPort(req.PortDNSOverTLS),
			tcpPort(req.PortDNSCrypt),
			udpPort(config.DNS.Port),
			udpPort(req.PortDNSOverQUIC),
		)
		if err != nil {
			aghhttp.Error(r, w, http.StatusBadRequest, "%s", err)

			return
		}
	}

	// TODO(e.burkov):  Investigate and perhaps check other ports.
	if !webCheckPortAvailable(req.PortHTTPS) {
		aghhttp.Error(
			r,
			w,
			http.StatusBadRequest,
			"port %d is not available, cannot enable https on it",
			req.PortHTTPS,
		)

		return
	}

	status := &tlsConfigStatus{}
	err = loadTLSConf(&req.tlsConfigSettings, status)
	if err != nil {
		resp := tlsConfig{
			tlsConfigSettingsExt: req,
			tlsConfigStatus:      status,
		}

		marshalTLS(w, r, resp)

		return
	}

	restartHTTPS := m.setConfig(req.tlsConfigSettings, status)
	m.setCertFileTime()
	onConfigModified()

	err = reconfigureDNSServer()
	if err != nil {
		aghhttp.Error(r, w, http.StatusInternalServerError, "%s", err)

		return
	}

	resp := tlsConfig{
		tlsConfigSettingsExt: req,
		tlsConfigStatus:      m.status,
	}

	marshalTLS(w, r, resp)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	// The background context is used because the TLSConfigChanged wraps context
	// with timeout on its own and shuts down the server, which handles current
	// request. It is also should be done in a separate goroutine due to the
	// same reason.
	if restartHTTPS {
		go func() {
			Context.web.TLSConfigChanged(context.Background(), req.tlsConfigSettings)
		}()
	}
}

// validatePorts validates the uniqueness of TCP and UDP ports for AdGuard Home
// DNS protocols.
func validatePorts(
	bindPort, betaBindPort, dohPort, dotPort, dnscryptTCPPort tcpPort,
	dnsPort, doqPort udpPort,
) (err error) {
	tcpPorts := aghalg.UniqChecker[tcpPort]{}
	addPorts(
		tcpPorts,
		tcpPort(bindPort),
		tcpPort(betaBindPort),
		tcpPort(dohPort),
		tcpPort(dotPort),
		tcpPort(dnscryptTCPPort),
	)

	err = tcpPorts.Validate()
	if err != nil {
		return fmt.Errorf("validating tcp ports: %w", err)
	}

	udpPorts := aghalg.UniqChecker[udpPort]{}
	addPorts(udpPorts, udpPort(dnsPort), udpPort(doqPort))

	err = udpPorts.Validate()
	if err != nil {
		return fmt.Errorf("validating udp ports: %w", err)
	}

	return nil
}

// validateCertChain validates the certificate chain and sets data in status.
// The returned error is also set in status.WarningValidation.
func validateCertChain(status *tlsConfigStatus, certChain []byte, serverName string) (err error) {
	defer func() {
		if err != nil {
			status.WarningValidation = err.Error()
		}
	}()

	log.Debug("tls: got certificate chain: %d bytes", len(certChain))

	var certs []*pem.Block
	pemblock := certChain
	for {
		var decoded *pem.Block
		decoded, pemblock = pem.Decode(pemblock)
		if decoded == nil {
			break
		}

		if decoded.Type == "CERTIFICATE" {
			certs = append(certs, decoded)
		}
	}

	parsedCerts, err := parsePEMCerts(certs)
	if err != nil {
		return err
	}

	status.ValidCert = true

	opts := x509.VerifyOptions{
		DNSName: serverName,
		Roots:   Context.tlsRoots,
	}

	log.Info("tls: number of certs: %d", len(parsedCerts))

	pool := x509.NewCertPool()
	for _, cert := range parsedCerts[1:] {
		log.Info("tls: got an intermediate cert")
		pool.AddCert(cert)
	}

	opts.Intermediates = pool

	mainCert := parsedCerts[0]
	_, err = mainCert.Verify(opts)
	if err != nil {
		// Let self-signed certs through and don't return this error.
		status.WarningValidation = fmt.Sprintf("certificate does not verify: %s", err)
	} else {
		status.ValidChain = true
	}

	if mainCert != nil {
		status.Subject = mainCert.Subject.String()
		status.Issuer = mainCert.Issuer.String()
		status.NotAfter = mainCert.NotAfter
		status.NotBefore = mainCert.NotBefore
		status.DNSNames = mainCert.DNSNames
	}

	return nil
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

// validatePKey validates the private key and sets data in status.  The returned
// error is also set in status.WarningValidation.
func validatePKey(status *tlsConfigStatus, pkey []byte) (err error) {
	defer func() {
		if err != nil {
			status.WarningValidation = err.Error()
		}
	}()

	var key *pem.Block

	// Go through all pem blocks, but take first valid pem block and drop the
	// rest.
	pemblock := []byte(pkey)
	for {
		var decoded *pem.Block
		decoded, pemblock = pem.Decode(pemblock)
		if decoded == nil {
			break
		}

		if decoded.Type == "PRIVATE KEY" || strings.HasSuffix(decoded.Type, " PRIVATE KEY") {
			key = decoded

			break
		}
	}

	if key == nil {
		return errors.Error("no valid keys were found")
	}

	_, keyType, err := parsePrivateKey(key.Bytes)
	if err != nil {
		return fmt.Errorf("parsing private key: %w", err)
	}

	if keyType == keyTypeED25519 {
		return errors.Error(
			"ED25519 keys are not supported by browsers; " +
				"did you mean to use X25519 for key exchange?",
		)
	}

	status.ValidKey = true
	status.KeyType = keyType

	return nil
}

// validateCertificates processes certificate data and its private key.  All
// parameters are optional.  status must not be nil.  The returned error is also
// set in status.WarningValidation.
func validateCertificates(
	status *tlsConfigStatus,
	certChain []byte,
	pkey []byte,
	serverName string,
) (err error) {
	defer func() {
		// Capitalize the warning for the UI.  Assume that warnings are all
		// ASCII-only.
		//
		// TODO(a.garipov): Figure out a better way to do this.  Perhaps a
		// custom string or error type.
		if w := status.WarningValidation; w != "" {
			status.WarningValidation = strings.ToUpper(w[:1]) + w[1:]
		}
	}()

	// Check only the public certificate separately from the key.
	if len(certChain) > 0 {
		err = validateCertChain(status, certChain, serverName)
		if err != nil {
			return err
		}
	}

	// Validate the private key by parsing it.
	if len(pkey) > 0 {
		err = validatePKey(status, pkey)
		if err != nil {
			return err
		}
	}

	// If both are set, validate together.
	if len(certChain) > 0 && len(pkey) > 0 {
		_, err = tls.X509KeyPair(certChain, pkey)
		if err != nil {
			err = fmt.Errorf("certificate-key pair: %w", err)
			status.WarningValidation = err.Error()

			return err
		}

		status.ValidPair = true
	}

	return nil
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

// unmarshalTLS handles base64-encoded certificates transparently
func unmarshalTLS(r *http.Request) (tlsConfigSettingsExt, error) {
	data := tlsConfigSettingsExt{}
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		return data, fmt.Errorf("failed to parse new TLS config json: %w", err)
	}

	if data.CertificateChain != "" {
		var cert []byte
		cert, err = base64.StdEncoding.DecodeString(data.CertificateChain)
		if err != nil {
			return data, fmt.Errorf("failed to base64-decode certificate chain: %w", err)
		}

		data.CertificateChain = string(cert)
		if data.CertificatePath != "" {
			return data, fmt.Errorf("certificate data and file can't be set together")
		}
	}

	if data.PrivateKey != "" {
		var key []byte
		key, err = base64.StdEncoding.DecodeString(data.PrivateKey)
		if err != nil {
			return data, fmt.Errorf("failed to base64-decode private key: %w", err)
		}

		data.PrivateKey = string(key)
		if data.PrivateKeyPath != "" {
			return data, fmt.Errorf("private key data and file can't be set together")
		}
	}

	return data, nil
}

func marshalTLS(w http.ResponseWriter, r *http.Request, data tlsConfig) {
	if data.CertificateChain != "" {
		encoded := base64.StdEncoding.EncodeToString([]byte(data.CertificateChain))
		data.CertificateChain = encoded
	}

	if data.PrivateKey != "" {
		data.PrivateKeySaved = true
		data.PrivateKey = ""
	}

	_ = aghhttp.WriteJSONResponse(w, r, data)
}

// registerWebHandlers registers HTTP handlers for TLS configuration.
func (m *tlsManager) registerWebHandlers() {
	httpRegister(http.MethodGet, "/control/tls/status", m.handleTLSStatus)
	httpRegister(http.MethodPost, "/control/tls/configure", m.handleTLSConfigure)
	httpRegister(http.MethodPost, "/control/tls/validate", m.handleTLSValidate)
}
