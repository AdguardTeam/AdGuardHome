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
	"github.com/AdguardTeam/golibs/service"
	"github.com/c2h5oh/datasize"
	"github.com/google/go-cmp/cmp"
)

// tlsManager contains the current configuration and state of AdGuard Home TLS
// encryption.
type tlsManager struct {
	// logger is used for logging the operation of the TLS Manager.
	logger *slog.Logger

	// mu protects certLastMod, tlsCert, tlsConf, extTLSConf.
	mu *sync.Mutex

	// certLastMod is the last modification time of the certificate file.
	certLastMod time.Time

	// tlsCert is the current TLS certificate.  tlsCert must not be stored in
	// [tls.Config.Certificates], as it violates its documentation.
	//
	// TODO(m.kazantsev):  Consider a better approach to store the certificate.
	tlsCert *tls.Certificate

	// tlsConf is a current TLS configuration.  It may be nil.
	tlsConf *tls.Config

	// extTLSConf contains extended TLS configuration settings.  It must not be
	// nil.
	extTLSConf *aghtls.ExtendedTLSConfig

	// rootCerts is a pool of root CAs for TLSv1.2.
	rootCerts *x509.CertPool

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

	// extTLSConf contains the extended TLS configuration.
	extTLSConf *aghtls.ExtendedTLSConfig

	// servePlainDNS defines if plain DNS is allowed for incoming requests.
	servePlainDNS bool
}

// newTLSManager initializes the manager of TLS configuration.  m is always
// non-nil while any returned error indicates that the TLS configuration isn't
// valid.  Thus TLS may be initialized later, e.g. via the web UI.  conf must
// not be nil.
func newTLSManager(ctx context.Context, conf *tlsManagerConfig) (m *tlsManager, err error) {
	m = &tlsManager{
		logger:       conf.logger,
		mu:           &sync.Mutex{},
		confModifier: conf.confModifier,
		httpReg:      conf.httpReg,
		manager:      conf.manager,
		extTLSConf:   &aghtls.ExtendedTLSConfig{},
	}

	if conf.extTLSConf != nil {
		m.extTLSConf = conf.extTLSConf
	}

	m.rootCerts = aghtls.SystemRootCAs(ctx, conf.logger)

	m.extTLSConf.ServePlainDNS = conf.servePlainDNS
	m.extTLSConf.Status = aghtls.TLSConfigStatus{}

	if len(m.extTLSConf.OverrideTLSCiphers) > 0 {
		m.customCipherIDs, err = aghtls.ParseCiphers(m.extTLSConf.OverrideTLSCiphers)
		if err != nil {
			// Should not happen because upstreams are already validated.  See
			// [validateTLSCipherIDs].
			panic(err)
		}

		m.logger.InfoContext(ctx, "overriding ciphers", "ciphers", conf.extTLSConf.OverrideTLSCiphers)
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

	err = loadTLSConfig(ctx, m.logger, m, m.extTLSConf, &m.extTLSConf.Status)
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
	}

	m.tlsCert = &cert
	m.setCertFileTime(ctx)

	return m, nil
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
	status := &aghtls.TLSConfigStatus{}

	err = loadTLSConfig(ctx, m.logger, m, &extTLSConf, status)
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
}

// loadTLSConfig loads and validates the TLS configuration.  It also sets
// [aghtls.ExtendedTLSConfig.CertificateChainData] and
// [aghtls.ExtendedTLSConfig.PrivateKeyData] properties.  The returned error is
// also set in [aghtls.TLSConfigStatus.WarningValidation].  All arguments must
// not be nil.
func loadTLSConfig(
	ctx context.Context,
	logger *slog.Logger,
	tlsConfProvider aghtls.TLSConfigProvider,
	extTLSConf *aghtls.ExtendedTLSConfig,
	status *aghtls.TLSConfigStatus,
) (err error) {
	defer func() {
		err = checkIfValidStatus(ctx, logger, status, err)
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

	err = validateCertificates(
		ctx,
		logger,
		tlsConfProvider,
		status,
		extTLSConf.CertificateChainData,
		extTLSConf.PrivateKeyData,
		extTLSConf.ServerName,
	)

	return errors.Annotate(err, "validating certificate pair: %w")
}

// checkIfValidStatus checks if status is valid.  If it is valid, certErr is set
// to nil.  Otherwise, certErr is returned as is.  logger and status must not be
// nil.
func checkIfValidStatus(
	ctx context.Context,
	logger *slog.Logger,
	status *aghtls.TLSConfigStatus,
	certErr error,
) (err error) {
	if certErr == nil {
		return nil
	}

	status.WarningValidation = certErr.Error()
	if status.ValidCert && status.ValidKey && status.ValidPair {
		// Do not return warnings since those aren't critical, just log.
		logger.WarnContext(
			ctx,
			"error while loading tls configuration",
			slogutil.KeyError, certErr,
		)

		certErr = nil
	}

	return certErr
}

// loadCertificateChainData loads PEM-encoded certificates chain data to the
// TLS configuration. tlsConf must be not nil. tlsConf.CertificateChainData
// struct field will be modified in case tlsConfig.CertificatePath is not an
// empty string.  extTLSConf must not be nil.
func loadCertificateChainData(extTLSConf *aghtls.ExtendedTLSConfig) (err error) {
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
func loadPrivateKeyData(extTLSConf *aghtls.ExtendedTLSConfig) (err error) {
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

// setPrivateFieldsAndCompare sets any missing properties in conf to match those
// in c and returns true if TLS configurations are equal.  conf must not be nil.
// It sets the following properties because these are not accepted from the
// frontend:
//
//	[ExtendedTLSConfig.DNSCryptConfigFile]
//	[ExtendedTLSConfig.OverrideTLSCiphers]
//	[ExtendedTLSConfig.PortDNSCrypt]
//
// The following properties are skipped as they are set by
// [tlsManager.loadTLSConfig]:
//
//	[ExtendedTLSConfig.CertificateChainData]
//	[ExtendedTLSConfig.PrivateKeyData]
func setPrivateFieldsAndCompare(
	currentTLSConf *aghtls.ExtendedTLSConfig,
	newTLSConf *aghtls.ExtendedTLSConfig,
) (equal bool) {
	newTLSConf.OverrideTLSCiphers = slices.Clone(currentTLSConf.OverrideTLSCiphers)

	newTLSConf.DNSCryptConfigFile = currentTLSConf.DNSCryptConfigFile
	newTLSConf.PortDNSCrypt = currentTLSConf.PortDNSCrypt

	// TODO(a.garipov): Define a custom comparer.
	return cmp.Equal(currentTLSConf, newTLSConf)
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

// updatePlainDNS checks the old value of
// [aghtls.ExtendedTLSConfig.ServePlainDNS] in currentTLSConf and if it differs
// from servePlain, sets the value of servePlain in newTLSConf.ServePlainDNS.
// currentTLSConf and newTLSConf must not be nil.
func updatePlainDNS(
	currentTLSConf *aghtls.ExtendedTLSConfig,
	newTLSConf *aghtls.ExtendedTLSConfig,
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
		newTLSConf.ServePlainDNS = currentTLSConf.ServePlainDNS
	}
}

// validateCertChain verifies certs using the first as the main one and others
// as intermediate.  srvName stands for the expected DNS name.  certs must not
// be empty.  logger must not be nil.
func validateCertChain(
	ctx context.Context,
	logger *slog.Logger,
	rootCAs *x509.CertPool,
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
		logger.InfoContext(
			ctx,
			"verifying certificate chain: got an intermediate cert",
			"num", othersLen,
		)
	}

	opts := x509.VerifyOptions{
		DNSName:       srvName,
		Roots:         rootCAs,
		Intermediates: pool,
	}
	_, err = main.Verify(opts)
	if err != nil {
		return fmt.Errorf("certificate does not verify: %w", err)
	}

	return nil
}

// errNoIPInCert is the error that is returned from [parseCertChain]
// if the leaf certificate doesn't contain IPs.
const errNoIPInCert errors.Error = `certificates has no IP addresses; ` +
	`DNS-over-TLS won't be advertised via DDR`

// parseCertChain parses the certificate chain from raw data, and returns it.
// If ok is true, the returned error, if any, is not critical.  logger must not
// be nil.
func parseCertChain(
	ctx context.Context,
	logger *slog.Logger,
	chain []byte,
) (parsedCerts []*x509.Certificate, ok bool, err error) {
	logger.DebugContext(ctx, "parsing certificate chain", "size", datasize.ByteSize(len(chain)))

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

	logger.InfoContext(ctx, "parsing multiple pem certificates", "num", len(parsedCerts))

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
// must not be nil, since it's used to accumulate the validation results.
// logger and tlsConfProvider must not be nil.  Other parameters are optional.
func validateCertificates(
	ctx context.Context,
	logger *slog.Logger,
	tlsConfProvider aghtls.TLSConfigProvider,
	status *aghtls.TLSConfigStatus,
	certChain []byte,
	pkey []byte,
	serverName string,
) (err error) {
	// Check only the public certificate separately from the key.
	if len(certChain) > 0 {
		var ok bool
		ok, err = validateCertificate(ctx, logger, tlsConfProvider.RootCAs(), status, certChain, serverName)
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
// it is used to accumulate the validation results.  logger and tlsConfProvider
// must not be nil. Other parameters are optional.  If ok is true, the returned
// error, if any, is not critical.
func validateCertificate(
	ctx context.Context,
	logger *slog.Logger,
	rootCAs *x509.CertPool,
	status *aghtls.TLSConfigStatus,
	certChain []byte,
	serverName string,
) (ok bool, err error) {
	// parseErr is a non-critical parse warning.
	var parseErr error
	var certs []*x509.Certificate

	// Set status.ValidCert to true to signal the frontend that the
	// certificate opens successfully and certificate chain is valid.
	certs, status.ValidCert, parseErr = parseCertChain(ctx, logger, certChain)
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

	err = validateCertChain(ctx, logger, rootCAs, certs, serverName)
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

	if data.tlsConfigSettings.CertificateChain != "" {
		var cert []byte
		cert, err = base64.StdEncoding.DecodeString(data.tlsConfigSettings.CertificateChain)
		if err != nil {
			return data, fmt.Errorf("failed to base64-decode certificate chain: %w", err)
		}

		data.tlsConfigSettings.CertificateChain = string(cert)
		if data.tlsConfigSettings.CertificatePath != "" {
			return data, fmt.Errorf("certificate data and file can't be set together")
		}
	}

	if data.tlsConfigSettings.PrivateKey == "" {
		return data, nil
	}

	key, err := base64.StdEncoding.DecodeString(data.tlsConfigSettings.PrivateKey)
	if err != nil {
		return data, fmt.Errorf("failed to base64-decode private key: %w", err)
	}

	data.tlsConfigSettings.PrivateKey = string(key)
	if data.tlsConfigSettings.PrivateKeyPath != "" {
		return data, fmt.Errorf("private key data and file can't be set together")
	}

	return data, nil
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

// HasIPAddrs implements the [aghtls.TLSConfigProvider] interface for
// *tlsManager.  It returns true if the current TLS configuration has at least
// one certificate with an IP address in its SAN extension.
func (m *tlsManager) HasIPAddrs() (ok bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.tlsCert == nil || m.tlsCert.Leaf == nil {
		return false
	}

	return aghtls.CertificateHasIP(m.tlsCert.Leaf)
}

// ExtendedTLSConfig implements the [aghtls.TLSConfigProvider] provider
// interface for *tlsManager.  It returns a deep copy of the stored extended TLS
// configuration.
func (m *tlsManager) ExtendedTLSConfig() (extTLSConf *aghtls.ExtendedTLSConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.extTLSConf.Clone()
}

// SetExtendedTLSConfig implements the [aghtls.TLSConfigProvider] interface for
// *tlsManager.  It updates the TLS configuration with the given one.  newConf
// must not be nil.  newConf is always modified. If restartsHTTPS is true,
// the HTTPS server must be restarted.  If error is not nil, restartHTTPS cannot
// be true.
func (m *tlsManager) SetExtendedTLSConfig(
	ctx context.Context,
	servePlainDNS aghalg.NullBool,
	newConf *aghtls.ExtendedTLSConfig,
) (restartHTTPS bool, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	err = m.updateTLSCert(newConf)
	if err != nil {
		m.logger.ErrorContext(ctx, "updating tls certificate", slogutil.KeyError, err)

		// Don't wrap the error, because it is informative enough as is.
		return false, err
	}

	updatePlainDNS(m.extTLSConf, newConf, servePlainDNS)

	if !setPrivateFieldsAndCompare(m.extTLSConf, newConf) {
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

// onGetCertificate gets [*tls.Certificate] from [*tls.Config].  If
// [tlsManager.extTLSConf.Enabled] is false, nil is returned.
//
// TODO(m.kazantsev):  Consider using tls.SupportsCertificate.
func (m *tlsManager) onGetCertificate(chi *tls.ClientHelloInfo) (cert *tls.Certificate, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.extTLSConf.Enabled || m.tlsConf == nil {
		return nil, nil
	}

	tlsCert := *m.tlsCert

	return &tlsCert, nil
}

// updateTLSCert loads and updates a TLS certificate for m.tlsConf.  If
// m.tlsConf is nil, it will be initialized.  extTLSConf must not be nil.  m.mu
// must be locked.
func (m *tlsManager) updateTLSCert(extTLSConf *aghtls.ExtendedTLSConfig) (err error) {
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

	m.tlsCert = &cert

	return nil
}

// type check
var _ service.Interface = (*tlsManager)(nil)

// Start implements the [service.Interface] interface for *tlsManager.  It
// starts the TLS manager.
func (m *tlsManager) Start(ctx context.Context) (err error) {
	go m.handleCertFileChange(ctx)

	return nil
}

// Shutdown implements the [service.Interface] interface for *tlsManager.  It
// shuts down the TLS manager and logs any errors.
//
// TODO(m.kazantsev):  Remove the method once [aghtls.TLSConfigProvider] is
// merged with [aghtls.Manager].
func (m *tlsManager) Shutdown(ctx context.Context) (err error) {
	err = m.manager.Shutdown(ctx)
	if err != nil {
		m.logger.ErrorContext(ctx, "shutting down tls manager", slogutil.KeyError, err)

		// Don't wrap the error, because it is informative enough as is.
		return err
	}

	return nil
}
