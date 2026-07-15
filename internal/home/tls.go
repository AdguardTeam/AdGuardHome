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
	"log/slog"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/agh"
	"github.com/AdguardTeam/AdGuardHome/internal/aghalg"
	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/aghtls"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/c2h5oh/datasize"
)

// tlsManager contains the current configuration and state of AdGuard Home TLS
// encryption.
type tlsManager struct {
	// logger is used for logging the operation of the TLS Manager.
	logger *slog.Logger

	// mu protects certLastMod, tlsConf, extTLSConf.
	mu *sync.Mutex

	// certLastMod is the last modification time of the certificate file.
	certLastMod time.Time

	// tlsConf is a current TLS configuration.  It may be nil.
	tlsConf *tls.Config

	// extTLSConf contains extended TLS configuration settings.  It must not be
	// nil.
	extTLSConf *tlsConfigSettings

	// rootCerts is a pool of root CAs for TLSv1.2.
	rootCerts *x509.CertPool

	// web is the web UI and API server.  It must not be nil.
	//
	// TODO(s.chzhen):  Temporary cyclic dependency due to ongoing refactoring.
	// Resolve it.
	web *webAPI

	// confModifier is used to update the global configuration.
	confModifier agh.ConfigModifier

	// httpReg registers HTTP handlers.  It must not be nil.
	httpReg aghhttp.Registrar

	// manager is used to manage the TLS certificate and key files.  It must not
	// be nil.
	manager aghtls.Manager

	// customCipherIDs are the IDs of the cipher suites that AdGuard Home must
	// use.
	customCipherIDs []uint16
}

// tlsManagerConfig contains the settings for initializing the TLS manager.
type tlsManagerConfig struct {
	// logger is used for logging the operation of the TLS Manager.  It must not
	// be nil.
	logger *slog.Logger

	// confModifier is used to update the global configuration.  It must not be
	// nil.
	confModifier agh.ConfigModifier

	// manager is used to manage the TLS certificate and key files.  It must not
	// be nil.
	manager aghtls.Manager

	httpReg aghhttp.Registrar

	// tlsSettings contains the TLS configuration settings.
	tlsSettings tlsConfigSettings

	// servePlainDNS defines if plain DNS is allowed for incoming requests.
	servePlainDNS bool
}

// newTLSManager initializes the manager of TLS configuration.  m is always
// non-nil while any returned error indicates that the TLS configuration isn't
// valid.  Thus TLS may be initialized later, e.g. via the web UI.  conf must
// not be nil.  Note that [tlsManager.web] must be initialized later on by using
// [tlsManager.setWebAPI].
func newTLSManager(ctx context.Context, conf *tlsManagerConfig) (m *tlsManager, err error) {
	m = &tlsManager{
		logger:       conf.logger,
		mu:           &sync.Mutex{},
		confModifier: conf.confModifier,
		httpReg:      conf.httpReg,
		manager:      conf.manager,
		extTLSConf:   &conf.tlsSettings,
	}

	m.rootCerts = aghtls.SystemRootCAs(ctx, conf.logger)

	m.extTLSConf.ServePlainDNS = conf.servePlainDNS
	m.extTLSConf.Status = tlsConfigStatus{}

	if len(conf.tlsSettings.OverrideTLSCiphers) > 0 {
		m.customCipherIDs, err = aghtls.ParseCiphers(conf.tlsSettings.OverrideTLSCiphers)
		if err != nil {
			// Should not happen because upstreams are already validated.  See
			// [validateTLSCipherIDs].
			panic(err)
		}

		m.logger.InfoContext(ctx, "overriding ciphers", "ciphers", conf.tlsSettings.OverrideTLSCiphers)
	} else {
		m.logger.InfoContext(ctx, "using default ciphers")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.extTLSConf.Enabled {
		return m, nil
	}

	err = m.manager.Set(ctx, aghtls.TLSPair{
		CertPath: m.extTLSConf.CertificatePath,
		KeyPath:  m.extTLSConf.PrivateKeyPath,
	})
	if err != nil {
		m.logger.ErrorContext(ctx, "setting tls files", slogutil.KeyError, err)
	}

	err = m.loadTLSConfig(ctx, m.extTLSConf, &m.extTLSConf.Status)
	if err != nil {
		m.extTLSConf.Enabled = false

		// Don't wrap the error, because it's informative enough as is.
		return m, err
	}

	cert, err := tls.X509KeyPair(m.extTLSConf.CertificateChainData, m.extTLSConf.PrivateKeyData)
	if err != nil {
		m.extTLSConf.Enabled = false

		return m, fmt.Errorf("parsing tls certificate: %w", err)
	}

	slices.Sort(cert.Leaf.DNSNames)

	m.tlsConf = &tls.Config{
		RootCAs:        m.rootCerts,
		CipherSuites:   m.customCipherIDs,
		MinVersion:     tls.VersionTLS12,
		GetCertificate: m.onGetCertificate,
		Certificates:   []tls.Certificate{cert},
	}

	m.setCertFileTime(ctx)

	return m, nil
}

// checkIfValidStatus checks if status is valid.  If it is valid, certErr is set
// to nil.  Otherwise, certErr is returned as is.  status must not be nil.
func (m *tlsManager) checkIfValidStatus(
	ctx context.Context,
	status *tlsConfigStatus,
	certErr error,
) (err error) {
	if certErr == nil {
		return nil
	}

	status.WarningValidation = certErr.Error()
	if status.ValidCert && status.ValidKey && status.ValidPair {
		// Do not return warnings since those aren't critical, just log.
		m.logger.WarnContext(
			ctx,
			"error while loading tls configuration",
			slogutil.KeyError, certErr,
		)

		certErr = nil
	}

	return certErr
}

// setWebAPI stores the provided web API.  It must be called before
// [tlsManager.start], [tlsManager.reload], [webAPI.handleTLSConfigure], or
// [webAPI.validateTLSSettings].
//
// TODO(s.chzhen):  Remove it once cyclic dependency is resolved.
func (m *tlsManager) setWebAPI(webAPI *webAPI) {
	m.web = webAPI
}

// extendedTLSConfig returns a deep copy of the stored extended TLS
// configuration.
func (m *tlsManager) extendedTLSConfig() (extTLSConf *tlsConfigSettings) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.extTLSConf.clone()
}

// setCertFileTime sets [tlsManager.certLastMod] from the certificate.  If there
// are errors, setCertFileTime logs them.  m.mu is expected to be locked.
func (m *tlsManager) setCertFileTime(ctx context.Context) {
	if m.extTLSConf.CertificatePath == "" {
		return
	}

	fi, err := os.Stat(m.extTLSConf.CertificatePath)
	if err != nil {
		m.logger.ErrorContext(ctx, "looking up certificate path", slogutil.KeyError, err)

		return
	}

	m.certLastMod = fi.ModTime().UTC()
}

// start updates the configuration of t and starts it.
//
// TODO(s.chzhen):  Use context.
func (m *tlsManager) start(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// The background context is used because the TLSConfigChanged wraps context
	// with timeout on its own and shuts down the server, which handles current
	// request.
	m.web.tlsConfigChanged(context.Background(), m.extTLSConf)

	go m.handleCertFileChange(ctx)
}

// handleCertFileChange handles changes in the certificate file.  It's intended
// to be run as a goroutine.
func (m *tlsManager) handleCertFileChange(ctx context.Context) {
	defer slogutil.RecoverAndLog(ctx, m.logger)

	updates := m.manager.Updates(ctx)
	if updates == nil {
		m.logger.ErrorContext(ctx, "no updates channel")

		return
	}

	for range updates {
		m.logger.DebugContext(ctx, "reloading")

		m.reload(ctx)
	}
}

// reload updates the configuration and restarts the TLS manager.  It logs any
// encountered errors.
//
// TODO(s.chzhen):  Consider returning an error.
func (m *tlsManager) reload(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()

	tlsConfPtr := m.extTLSConf

	if !tlsConfPtr.Enabled || len(tlsConfPtr.CertificatePath) == 0 {
		return
	}

	certPath := tlsConfPtr.CertificatePath
	fi, err := os.Stat(certPath)
	if err != nil {
		m.logger.ErrorContext(ctx, "checking certificate file", slogutil.KeyError, err)

		return
	}

	if fi.ModTime().UTC().Equal(m.certLastMod) {
		m.logger.InfoContext(ctx, "certificate file is not modified")

		return
	}

	m.logger.InfoContext(ctx, "certificate file is modified")

	extTLSConf := *tlsConfPtr
	status := &tlsConfigStatus{}

	err = m.loadTLSConfig(ctx, &extTLSConf, status)
	if err != nil {
		m.logger.WarnContext(ctx, "reloading interrupted", slogutil.KeyError, err)

		return
	}

	err = m.updateTLSCert(&extTLSConf)
	if err != nil {
		m.logger.WarnContext(ctx, "failed to update tls certificate", slogutil.KeyError, err)

		return
	}

	extTLSConf.Status = *status

	m.extTLSConf = &extTLSConf
	m.certLastMod = fi.ModTime().UTC()

	// The background context is used because the TLSConfigChanged wraps context
	// with timeout on its own and shuts down the server, which handles current
	// request.
	m.web.tlsConfigChanged(context.Background(), m.extTLSConf)
}

// loadTLSConfig loads and validates the TLS configuration.  It also sets
// [tlsConfigSettings.CertificateChainData] and
// [tlsConfigSettings.PrivateKeyData] properties.  The returned error is also
// set in status.WarningValidation.  All arguments must not be nil.  m.mu is
// expected to be locked.
func (m *tlsManager) loadTLSConfig(
	ctx context.Context,
	extTLSConf *tlsConfigSettings,
	status *tlsConfigStatus,
) (err error) {
	defer func() {
		err = m.checkIfValidStatus(ctx, status, err)
	}()

	err = loadCertificateChainData(extTLSConf)
	if err != nil {
		// Don't wrap the error, because it's informative enough as is.
		return err
	}

	err = loadPrivateKeyData(extTLSConf)
	if err != nil {
		// Don't wrap the error, because it's informative enough as is.
		return err
	}

	err = m.validateCertificates(
		ctx,
		status,
		extTLSConf.CertificateChainData,
		extTLSConf.PrivateKeyData,
		extTLSConf.ServerName,
	)

	return errors.Annotate(err, "validating certificate pair: %w")
}

// loadCertificateChainData loads PEM-encoded certificates chain data to the
// TLS configuration. tlsConf must be not nil. tlsConf.CertificateChainData
// struct field will be modified in case tlsConfig.CertificatePath is not an
// empty string.  extTLSConf must not be nil.
func loadCertificateChainData(extTLSConf *tlsConfigSettings) (err error) {
	extTLSConf.CertificateChainData = []byte(extTLSConf.CertificateChain)
	if extTLSConf.CertificatePath != "" {
		if extTLSConf.CertificateChain != "" {
			return errors.Error("certificate data and file can't be set together")
		}

		extTLSConf.CertificateChainData, err = os.ReadFile(extTLSConf.CertificatePath)
		if err != nil {
			return fmt.Errorf("reading cert file: %w", err)
		}
	}

	return nil
}

// loadPrivateKeyData loads PEM-encoded private key data to the TLS
// configuration. tlsConf must be not nil. tlsConf.PrivateKeyData struct field
// will be modified in case tlsConfig.PrivateKeyPath is not an empty string.
// extTLSConf must not be nil.
func loadPrivateKeyData(extTLSConf *tlsConfigSettings) (err error) {
	extTLSConf.PrivateKeyData = []byte(extTLSConf.PrivateKey)
	if extTLSConf.PrivateKeyPath != "" {
		if extTLSConf.PrivateKey != "" {
			return errors.Error("private key data and file can't be set together")
		}

		extTLSConf.PrivateKeyData, err = os.ReadFile(extTLSConf.PrivateKeyPath)
		if err != nil {
			return fmt.Errorf("reading key file: %w", err)
		}
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

// setConfig updates manager TLS configuration with the given one.  newConf must
// not be nil.
func (m *tlsManager) setConfig(
	ctx context.Context,
	newConf *tlsConfigSettings,
	servePlain aghalg.NullBool,
) (restartHTTPS bool, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	err = m.updateTLSCert(newConf)
	if err != nil {
		m.logger.ErrorContext(ctx, "updating tls certificate", slogutil.KeyError, err)

		// Don't wrap the error, because it is informative enough as is.
		return false, err
	}

	m.extTLSConf.updatePlainDNS(newConf, servePlain)

	if !m.extTLSConf.setPrivateFieldsAndCompare(newConf) {
		m.logger.InfoContext(ctx, "config has changed, restarting https server")
		restartHTTPS = true
	} else {
		m.logger.InfoContext(ctx, "config has not changed")
	}

	m.extTLSConf = newConf

	certPath, keyPath := "", ""
	if newConf.Enabled {
		certPath, keyPath = newConf.CertificatePath, newConf.PrivateKeyPath
	}

	err = m.manager.Set(ctx, aghtls.TLSPair{
		CertPath: certPath,
		KeyPath:  keyPath,
	})
	if err != nil {
		m.logger.ErrorContext(ctx, "setting tls files", slogutil.KeyError, err)
	}

	m.setCertFileTime(ctx)

	return restartHTTPS, nil
}

// updatePlainDNS checks the old value of [tlsConfigSettings.ServePlainDNS] in
// c and if it differs from servePlain, sets the value of servePlain in
// newTLSConf.ServePlainDNS.  newTLSConf must not be nil.
func (c *tlsConfigSettings) updatePlainDNS(
	newTLSConf *tlsConfigSettings,
	servePlain aghalg.NullBool,
) {
	if servePlain != aghalg.NBNull {
		func() {
			config.Lock()
			defer config.Unlock()

			config.DNS.ServePlainDNS = servePlain == aghalg.NBTrue
		}()

		newTLSConf.ServePlainDNS = servePlain == aghalg.NBTrue
	} else {
		newTLSConf.ServePlainDNS = c.ServePlainDNS
	}
}

// validateCertChain verifies certs using the first as the main one and others
// as intermediate.  srvName stands for the expected DNS name.  certs must not
// be empty.
//
// TODO(e.burkov):  Pass logger and rootCerts through arguments and remove
// dependency on tlsManager.
func (m *tlsManager) validateCertChain(
	ctx context.Context,
	certs []*x509.Certificate,
	srvName string,
) (err error) {
	main, others := certs[0], certs[1:]

	pool := x509.NewCertPool()
	for _, cert := range others {
		pool.AddCert(cert)
	}

	othersLen := len(others)
	if othersLen > 0 {
		m.logger.InfoContext(
			ctx,
			"verifying certificate chain: got an intermediate cert",
			"num", othersLen,
		)
	}

	opts := x509.VerifyOptions{
		DNSName:       srvName,
		Roots:         m.rootCerts,
		Intermediates: pool,
	}
	_, err = main.Verify(opts)
	if err != nil {
		return fmt.Errorf("certificate does not verify: %w", err)
	}

	return nil
}

// errNoIPInCert is the error that is returned from [tlsManager.parseCertChain]
// if the leaf certificate doesn't contain IPs.
const errNoIPInCert errors.Error = `certificates has no IP addresses; ` +
	`DNS-over-TLS won't be advertised via DDR`

// parseCertChain parses the certificate chain from raw data, and returns it.
// If ok is true, the returned error, if any, is not critical.
//
// TODO(e.burkov):  Pass logger through arguments and remove dependency on
// tlsManager.
func (m *tlsManager) parseCertChain(
	ctx context.Context,
	chain []byte,
) (parsedCerts []*x509.Certificate, ok bool, err error) {
	m.logger.DebugContext(ctx, "parsing certificate chain", "size", datasize.ByteSize(len(chain)))

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

	m.logger.InfoContext(ctx, "parsing multiple pem certificates", "num", len(parsedCerts))

	if !aghtls.CertificateHasIP(parsedCerts[0]) {
		err = errNoIPInCert
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
func (m *tlsManager) validateCertificates(
	ctx context.Context,
	status *tlsConfigStatus,
	certChain []byte,
	pkey []byte,
	serverName string,
) (err error) {
	// Check only the public certificate separately from the key.
	if len(certChain) > 0 {
		var ok bool
		ok, err = m.validateCertificate(ctx, status, certChain, serverName)
		if !ok {
			// Don't wrap the error, since it's informative enough as is.
			return err
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

		// Set status.ValidKey to true to signal the frontend that the
		// key is valid.
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

// validateCertificate processes certificate data.  status must not be nil, as
// it is used to accumulate the validation results.  Other parameters are
// optional.  If ok is true, the returned error, if any, is not critical.
func (m *tlsManager) validateCertificate(
	ctx context.Context,
	status *tlsConfigStatus,
	certChain []byte,
	serverName string,
) (ok bool, err error) {
	// parseErr is a non-critical parse warning.
	var parseErr error
	var certs []*x509.Certificate

	// Set status.ValidCert to true to signal the frontend that the
	// certificate opens successfully and certificate chain is valid.
	certs, status.ValidCert, parseErr = m.parseCertChain(ctx, certChain)
	if !status.ValidCert {
		// Don't wrap the error, since it's informative enough as is.
		return false, parseErr
	}

	mainCert := certs[0]
	status.Subject = mainCert.Subject.String()
	status.Issuer = mainCert.Issuer.String()
	status.NotAfter = mainCert.NotAfter
	status.NotBefore = mainCert.NotBefore
	status.DNSNames = mainCert.DNSNames

	err = m.validateCertChain(ctx, certs, serverName)
	if err != nil {
		// Let self-signed certs through and don't return this error to set
		// its message into the status.WarningValidation afterwards.
		return true, err
	}

	status.ValidChain = true

	// Propagate the non-critical parse warning.
	return true, parseErr
}

// Key types.
const (
	keyTypeECDSA   = "ECDSA"
	keyTypeED25519 = "ED25519"
	keyTypeRSA     = "RSA"
)

// Attempt to parse the given private key DER block.  OpenSSL 0.9.8 generates
// PKCS#1 private keys by default, while OpenSSL 1.0.0 generates PKCS#8 keys.
// OpenSSL ecparam generates SEC1 EC private keys for ECDSA.  We try all three.
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

// unmarshalTLS handles base64-encoded certificates transparently.
func unmarshalTLS(r *http.Request) (data tlsConfigSettingsExt, err error) {
	data = tlsConfigSettingsExt{}
	err = json.NewDecoder(r.Body).Decode(&data)
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

	if data.PrivateKey == "" {
		return data, nil
	}

	key, err := base64.StdEncoding.DecodeString(data.PrivateKey)
	if err != nil {
		return data, fmt.Errorf("failed to base64-decode private key: %w", err)
	}

	data.PrivateKey = string(key)
	if data.PrivateKeyPath != "" {
		return data, fmt.Errorf("private key data and file can't be set together")
	}

	return data, nil
}

// marshalTLS encodes sensitive fields and writes data as JSON.  All arguments
// must not be nil.
func (m *tlsManager) marshalTLS(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	data *tlsConfig,
) {
	if data.CertificateChain != "" {
		encoded := base64.StdEncoding.EncodeToString([]byte(data.CertificateChain))
		data.CertificateChain = encoded
	}

	if data.PrivateKey != "" {
		data.PrivateKeySaved = true
		data.PrivateKey = ""
	}

	aghhttp.WriteJSONResponseOK(ctx, m.logger, w, r, *data)
}

// TLSConfig implements the [aghtls.TLSConfigProvider] interface for
// *tlsManager.
func (m *tlsManager) TLSConfig() (conf *tls.Config) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.tlsConf.Clone()
}

// RootCAs implements the [aghtls.TLSConfigProvider] interface for *tlsManager.
func (m *tlsManager) RootCAs() (root *x509.CertPool) {
	return m.rootCerts
}

// onGetCertificate gets [*tls.Certificate] from [*tls.Config].  If
// [tlsManager.extTLSConf.Enabled] is false, nil is returned.
func (m *tlsManager) onGetCertificate(chi *tls.ClientHelloInfo) (cert *tls.Certificate, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.extTLSConf.Enabled || m.tlsConf == nil {
		return nil, nil
	}

	if len(m.tlsConf.Certificates) == 0 {
		return nil, nil
	}

	tlsCert := m.tlsConf.Certificates[0]

	return &tlsCert, nil
}

// updateTLSCert loads and updates a TLS certificate for m.tlsConf.  If
// m.tlsConf is nil, it will be initialized.  extTLSConf must not be nil.  m.mu
// is expected to be locked.
func (m *tlsManager) updateTLSCert(extTLSConf *tlsConfigSettings) (err error) {
	if len(extTLSConf.CertificateChainData) == 0 || len(extTLSConf.PrivateKeyData) == 0 {
		return nil
	}

	cert, err := tls.X509KeyPair(extTLSConf.CertificateChainData, extTLSConf.PrivateKeyData)
	if err != nil {
		return fmt.Errorf("loading tls certificate: %w", err)
	}

	slices.Sort(cert.Leaf.DNSNames)

	if m.tlsConf == nil {
		m.tlsConf = &tls.Config{
			RootCAs:        m.rootCerts,
			CipherSuites:   m.customCipherIDs,
			MinVersion:     tls.VersionTLS12,
			GetCertificate: m.onGetCertificate,
		}
	}

	m.tlsConf.Certificates = []tls.Certificate{cert}

	return nil
}
