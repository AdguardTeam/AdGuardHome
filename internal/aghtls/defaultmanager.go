package aghtls

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
	"log/slog"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghalg"
	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/c2h5oh/datasize"
	"github.com/google/go-cmp/cmp"
)

// DefaultManagerConfig is the configuration structure for [NewDefaultManager].
type DefaultManagerConfig struct {
	// Logger is used for logging the operation of the manager.  It must not be
	// nil.
	Logger *slog.Logger

	// Watcher is used to watch the TLS certificate and key files.  It must not
	// be nil.
	Watcher aghos.FSWatcher

	// ExtendedTLSConfig is the initial extended TLS configuration.  It may be
	// nil.
	ExtendedTLSConfig *ExtendedTLSConfig

	// ServePlainDNS is used to determine whether to serve DNS over plain
	// UDP/TCP.
	ServePlainDNS bool
}

// DefaultManager is the default implementation of the [Manager] interface.
//
// TODO(e.burkov):  Add tests.
type DefaultManager struct {
	certLastMod time.Time
	watcher     aghos.FSWatcher
	logger      *slog.Logger

	// mu protects tlsConf, extTLSConf, certLastMod, tlsCert, and pair.
	mu              *sync.Mutex
	tlsConf         *tls.Config
	extTLSConf      *ExtendedTLSConfig
	tlsCert         *tls.Certificate
	rootCerts       *x509.CertPool
	updates         chan UpdateSignal
	pair            TLSPair
	customCipherIDs []uint16
}

// NewDefaultManager returns a new properly initialized default manager.  conf
// must be non-nil and valid.  mgr is always non-nil while any returned error
// indicates that the TLS configuration isn't valid.  Thus TLS may be
// initialized later, e.g. via the web UI.
func NewDefaultManager(
	ctx context.Context,
	conf *DefaultManagerConfig,
) (mgr *DefaultManager, err error) {
	mgr = &DefaultManager{
		logger: conf.Logger,
		mu:     &sync.Mutex{},
		pair:   TLSPair{},
		// Buffer the channel to avoid missing updates.
		updates:    make(chan UpdateSignal, 1),
		watcher:    conf.Watcher,
		extTLSConf: &ExtendedTLSConfig{},
	}

	if conf.ExtendedTLSConfig != nil {
		mgr.extTLSConf = conf.ExtendedTLSConfig
	}

	mgr.rootCerts = SystemRootCAs(ctx, conf.Logger)
	mgr.extTLSConf.ServePlainDNS = conf.ServePlainDNS
	mgr.extTLSConf.Status = TLSConfigStatus{}

	mgr.parseCustomCiphers(ctx)

	// There is no need to lock m.mu here, since the manager isn't shared with
	// other goroutines yet.
	if !mgr.extTLSConf.Enabled {
		return mgr, nil
	}

	err = mgr.Set(ctx, TLSPair{
		CertPath: mgr.extTLSConf.CertificatePath,
		KeyPath:  mgr.extTLSConf.PrivateKeyPath,
	})
	if err != nil {
		mgr.logger.ErrorContext(ctx, "setting tls files", slogutil.KeyError, err)
	}

	err = mgr.prepareTLSConfig(ctx)
	if err != nil {
		// Don't wrap the error, because it's informative enough as is.
		return mgr, err
	}

	return mgr, nil
}

// parseCustomCiphers parses the custom TLS ciphers specified in the extended
// TLS configuration and sets them in the TLS manager.  If no custom ciphers are
// specified, it uses the default ciphers.
func (mgr *DefaultManager) parseCustomCiphers(ctx context.Context) {
	if len(mgr.extTLSConf.OverrideTLSCiphers) == 0 {
		mgr.logger.InfoContext(ctx, "using default ciphers")

		return
	}

	var err error
	mgr.customCipherIDs, err = ParseCiphers(mgr.extTLSConf.OverrideTLSCiphers)
	if err != nil {
		// Should not happen because upstreams are already validated.  See
		// [validateTLSCipherIDs].
		panic(err)
	}

	mgr.logger.InfoContext(ctx, "overriding ciphers", "ciphers", mgr.extTLSConf.OverrideTLSCiphers)
}

// prepareTLSConfig prepares the TLS configuration for the mgr.  It returns an
// error if the TLS configuration is invalid.
func (mgr *DefaultManager) prepareTLSConfig(ctx context.Context) (err error) {
	err = LoadTLSConfig(ctx, mgr.logger, mgr, mgr.extTLSConf, &mgr.extTLSConf.Status)
	if err != nil {
		mgr.extTLSConf.Enabled = false

		// Don't wrap the error, because it's informative enough as is.
		return err
	}

	cert, err := tls.X509KeyPair(mgr.extTLSConf.CertificateChainData, mgr.extTLSConf.PrivateKeyData)
	if err != nil {
		// DNSCrypt provides its own certificate, meaning we can ignore TLS
		// certificate parsing errors.
		if mgr.extTLSConf.PortDNSCrypt != 0 && mgr.extTLSConf.DNSCryptConfigFile != "" {
			mgr.logger.InfoContext(ctx, "dnscrypt is configured")

			return nil
		}

		mgr.extTLSConf.Enabled = false

		return fmt.Errorf("parsing tls certificate: %w", err)
	}

	slices.Sort(cert.Leaf.DNSNames)

	mgr.tlsCert = &cert
	mgr.setCertFileTime(ctx)
	mgr.tlsConf = &tls.Config{
		RootCAs:        mgr.rootCerts,
		CipherSuites:   mgr.customCipherIDs,
		GetCertificate: mgr.onGetCertificate,
		MinVersion:     tls.VersionTLS12,
	}

	return nil
}

// type check
var _ Manager = (*DefaultManager)(nil)

// Set implements the [Manager] interface for *DefaultManager.
func (mgr *DefaultManager) Set(ctx context.Context, certKey TLSPair) (err error) {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	return mgr.setLocked(ctx, certKey)
}

// setLocked sets the TLS certificate and key.  mgr.mu must be locked.
func (mgr *DefaultManager) setLocked(ctx context.Context, certKey TLSPair) (err error) {
	mgr.logger.DebugContext(ctx, "setting", "cert", certKey.CertPath, "key", certKey.KeyPath)

	var errs []error

	old := mgr.pair

	errs = mgr.appendUnwatchErr(errs, "old cert", old.CertPath)
	errs = mgr.appendUnwatchErr(errs, "old key", old.KeyPath)
	errs = mgr.appendWatchErr(errs, "new cert", certKey.CertPath)
	errs = mgr.appendWatchErr(errs, "new key", certKey.KeyPath)

	mgr.pair = certKey

	return errors.Join(errs...)
}

// appendUnwatchErr stops watching a file at path p described by what and
// appends an error to the errs slice, if any.  Empty p is ignored.
func (mgr *DefaultManager) appendUnwatchErr(errs []error, what, p string) (result []error) {
	if p == "" {
		return errs
	}

	err := mgr.watcher.Remove(p)
	if err != nil {
		errs = append(errs, fmt.Errorf("unwatching %s %s: %w", what, p, err))
	}

	return errs
}

// appendWatchErr starts watching a file at path p described by what and
// appends an error to the errs slice, if any.  Empty p is ignored.
func (mgr *DefaultManager) appendWatchErr(errs []error, what, p string) (result []error) {
	if p == "" {
		return errs
	}

	err := mgr.watcher.Add(p)
	if err != nil {
		errs = append(errs, fmt.Errorf("watching %s %s: %w", what, p, err))
	}

	return errs
}

// Refresh implements the [service.Refresher] interface for *DefaultManager.
func (mgr *DefaultManager) Refresh(ctx context.Context) (err error) {
	mgr.logger.DebugContext(ctx, "refreshing")

	select {
	case mgr.updates <- UpdateSignal{}:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("refreshing: %w", ctx.Err())
	default:
		return nil
	}
}

// Start implements the [service.Interface] interface for *DefaultManager.
func (mgr *DefaultManager) Start(ctx context.Context) (err error) {
	err = mgr.watcher.Start(ctx)
	if err != nil {
		return fmt.Errorf("starting watcher: %w", err)
	}

	go mgr.handleEvents(ctx)
	go mgr.handleCertFileChange(ctx)

	return nil
}

// Shutdown implements the [service.Interface] interface for *DefaultManager.
func (mgr *DefaultManager) Shutdown(ctx context.Context) (err error) {
	defer close(mgr.updates)

	err = mgr.watcher.Shutdown(ctx)
	if err != nil {
		return fmt.Errorf("shutting down watcher: %w", err)
	}

	return nil
}

// Updates implements the [Manager] interface for *DefaultManager.
func (mgr *DefaultManager) Updates(ctx context.Context) (updates <-chan UpdateSignal) {
	return mgr.updates
}

// handleEvents handles changes of the tracked files.  It is intended to be run
// in a separate goroutine.
func (mgr *DefaultManager) handleEvents(ctx context.Context) {
	defer slogutil.RecoverAndLog(ctx, mgr.logger)

	eventsCh := mgr.watcher.Events()
	if eventsCh == nil {
		mgr.logger.DebugContext(ctx, "watcher does not emit events")

		return
	}

	for range eventsCh {
		err := mgr.Refresh(ctx)
		if err != nil {
			mgr.logger.ErrorContext(ctx, "refreshing", slogutil.KeyError, err)
		}
	}
}

// handleCertFileChange handles changes in the certificate file.  It's intended
// to be run as a goroutine.
func (mgr *DefaultManager) handleCertFileChange(ctx context.Context) {
	defer slogutil.RecoverAndLog(ctx, mgr.logger)

	updates := mgr.Updates(ctx)
	if updates == nil {
		mgr.logger.ErrorContext(ctx, "no updates channel")

		return
	}

	for range updates {
		mgr.logger.DebugContext(ctx, "reloading")

		mgr.reload(ctx)
	}
}

// reload updates the configuration and restarts the TLS manager.  It logs any
// encountered errors.
//
// TODO(s.chzhen):  Consider returning an error.
func (mgr *DefaultManager) reload(ctx context.Context) {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	tlsConfPtr := mgr.extTLSConf

	if !tlsConfPtr.Enabled || len(tlsConfPtr.CertificatePath) == 0 {
		return
	}

	certPath := tlsConfPtr.CertificatePath
	fi, err := os.Stat(certPath)
	if err != nil {
		mgr.logger.ErrorContext(ctx, "checking certificate file", slogutil.KeyError, err)

		return
	}

	if fi.ModTime().UTC().Equal(mgr.certLastMod) {
		mgr.logger.InfoContext(ctx, "certificate file is not modified")

		return
	}

	mgr.logger.InfoContext(ctx, "certificate file is modified")

	extTLSConf := *tlsConfPtr
	status := &TLSConfigStatus{}

	err = LoadTLSConfig(ctx, mgr.logger, mgr, &extTLSConf, status)
	if err != nil {
		mgr.logger.WarnContext(ctx, "reloading interrupted", slogutil.KeyError, err)

		return
	}

	err = mgr.updateTLSCert(&extTLSConf)
	if err != nil {
		mgr.logger.WarnContext(ctx, "failed to update tls certificate", slogutil.KeyError, err)

		return
	}

	extTLSConf.Status = *status

	mgr.extTLSConf = &extTLSConf
	mgr.certLastMod = fi.ModTime().UTC()
}

// CipherSuites implements the [Manager] interface for *DefaultManager.  It
// returns the list of TLS cipher suite IDs.
func (mgr *DefaultManager) CipherSuites() (cs []uint16) {
	return mgr.customCipherIDs
}

// TLSConfig implements the [Manager] interface for *DefaultManager.
func (mgr *DefaultManager) TLSConfig() (conf *tls.Config) {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	return mgr.tlsConf.Clone()
}

// RootCAs implements the [Manager] interface for *DefaultManager.
func (mgr *DefaultManager) RootCAs() (root *x509.CertPool) {
	return mgr.rootCerts
}

// HasIPAddrs implements the [Manager] interface for *DefaultManager.  It
// returns true if the current TLS configuration has at least one certificate
// with an IP address in its SAN extension.
func (mgr *DefaultManager) HasIPAddrs() (ok bool) {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	if mgr.tlsCert == nil || mgr.tlsCert.Leaf == nil {
		return false
	}

	return CertificateHasIP(mgr.tlsCert.Leaf)
}

// ExtendedTLSConfig implements the [Manager] provider interface for
// *DefaultManager.  It returns a deep copy of the stored extended TLS
// configuration.
func (mgr *DefaultManager) ExtendedTLSConfig() (extTLSConf *ExtendedTLSConfig) {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	return mgr.extTLSConf.Clone()
}

// SetExtendedTLSConfig implements the [Manager] interface for *DefaultManager.
// It updates the TLS configuration with the given one.  newConf must not be
// nil.  newConf is always modified.  If restartHTTPS is true, the HTTPS server
// must be restarted.  If error is not nil, restartHTTPS cannot be true.
func (mgr *DefaultManager) SetExtendedTLSConfig(
	ctx context.Context,
	servePlainDNS aghalg.NullBool,
	newConf *ExtendedTLSConfig,
) (restartHTTPS bool, err error) {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	err = mgr.updateTLSCert(newConf)
	if err != nil {
		mgr.logger.ErrorContext(ctx, "updating tls certificate", slogutil.KeyError, err)

		// Don't wrap the error, because it is informative enough as is.
		return false, err
	}

	if servePlainDNS != aghalg.NBNull {
		newConf.ServePlainDNS = servePlainDNS == aghalg.NBTrue
	} else {
		newConf.ServePlainDNS = mgr.extTLSConf.ServePlainDNS
	}

	if !setPrivateFieldsAndCompare(mgr.extTLSConf, newConf) {
		mgr.logger.InfoContext(ctx, "config has changed, restarting https server")
		restartHTTPS = true
	} else {
		mgr.logger.InfoContext(ctx, "config has not changed")
	}

	mgr.extTLSConf = newConf

	certPath, keyPath := "", ""
	if newConf.Enabled {
		certPath, keyPath = newConf.CertificatePath, newConf.PrivateKeyPath
	}

	err = mgr.setLocked(ctx, TLSPair{
		CertPath: certPath,
		KeyPath:  keyPath,
	})
	if err != nil {
		mgr.logger.ErrorContext(ctx, "setting tls files", slogutil.KeyError, err)
	}

	mgr.setCertFileTime(ctx)

	return restartHTTPS, nil
}

// updateTLSCert loads and updates a TLS certificate for m.tlsConf.  If
// m.tlsConf is nil, it will be initialized.  extTLSConf must not be nil.  m.mu
// must be locked.
func (mgr *DefaultManager) updateTLSCert(extTLSConf *ExtendedTLSConfig) (err error) {
	if len(extTLSConf.CertificateChainData) == 0 || len(extTLSConf.PrivateKeyData) == 0 {
		return nil
	}

	cert, err := tls.X509KeyPair(extTLSConf.CertificateChainData, extTLSConf.PrivateKeyData)
	if err != nil {
		return fmt.Errorf("loading tls certificate: %w", err)
	}

	slices.Sort(cert.Leaf.DNSNames)

	if mgr.tlsConf == nil {
		mgr.tlsConf = &tls.Config{
			RootCAs:        mgr.rootCerts,
			CipherSuites:   mgr.customCipherIDs,
			MinVersion:     tls.VersionTLS12,
			GetCertificate: mgr.onGetCertificate,
		}
	}

	mgr.tlsCert = &cert

	return nil
}

// onGetCertificate gets [*tls.Certificate] from [*tls.Config].  If
// [DefaultManager.extTLSConf.Enabled] is false, nil is returned.
//
// TODO(m.kazantsev):  Consider using tls.SupportsCertificate.
func (mgr *DefaultManager) onGetCertificate(
	chi *tls.ClientHelloInfo) (cert *tls.Certificate, err error,
) {
	mgr.mu.Lock()
	defer mgr.mu.Unlock()

	if !mgr.extTLSConf.Enabled || mgr.tlsConf == nil {
		return nil, nil
	}

	tlsCert := *mgr.tlsCert

	return &tlsCert, nil
}

// setPrivateFieldsAndCompare sets any missing properties in conf to match those
// in c and returns true if TLS configurations are equal.  conf must not be nil.
// It sets the following properties because these are not accepted from the
// frontend:
//
//	[ExtendedTLSConfig.DNSCryptConfigFile]
//	[ExtendedTLSConfig.OverrideTLSCiphers]
//	[ExtendedTLSConfig.PortDNSCrypt]
func setPrivateFieldsAndCompare(
	currentTLSConf *ExtendedTLSConfig,
	newTLSConf *ExtendedTLSConfig,
) (equal bool) {
	newTLSConf.OverrideTLSCiphers = slices.Clone(currentTLSConf.OverrideTLSCiphers)

	newTLSConf.DNSCryptConfigFile = currentTLSConf.DNSCryptConfigFile
	newTLSConf.PortDNSCrypt = currentTLSConf.PortDNSCrypt

	// TODO(a.garipov): Define a custom comparer.
	return cmp.Equal(currentTLSConf, newTLSConf)
}

// setCertFileTime sets [tlsManager.certLastMod] from the certificate.  If there
// are errors, setCertFileTime logs them.  m.mu is expected to be locked.
func (mgr *DefaultManager) setCertFileTime(ctx context.Context) {
	if mgr.extTLSConf.CertificatePath == "" {
		return
	}

	fi, err := os.Stat(mgr.extTLSConf.CertificatePath)
	if err != nil {
		mgr.logger.ErrorContext(ctx, "looking up certificate path", slogutil.KeyError, err)

		return
	}

	mgr.certLastMod = fi.ModTime().UTC()
}

// LoadTLSConfig loads and validates the TLS configuration.  It also sets
// [aghtls.ExtendedTLSConfig.CertificateChainData] and
// [aghtls.ExtendedTLSConfig.PrivateKeyData] properties.  The returned error is
// also set in [aghtls.TLSConfigStatus.WarningValidation].  All arguments must
// not be nil.
func LoadTLSConfig(
	ctx context.Context,
	logger *slog.Logger,
	tlsConfProvider Manager,
	extTLSConf *ExtendedTLSConfig,
	status *TLSConfigStatus,
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
	status *TLSConfigStatus,
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
func loadCertificateChainData(extTLSConf *ExtendedTLSConfig) (err error) {
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
func loadPrivateKeyData(extTLSConf *ExtendedTLSConfig) (err error) {
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

	if !CertificateHasIP(parsedCerts[0]) {
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
// logger and tlsManager must not be nil.  Other parameters are optional.
func validateCertificates(
	ctx context.Context,
	logger *slog.Logger,
	tlsManager Manager,
	status *TLSConfigStatus,
	certChain []byte,
	pkey []byte,
	serverName string,
) (err error) {
	// Check only the public certificate separately from the key.
	if len(certChain) > 0 {
		var ok bool
		ok, err = validateCertificate(
			ctx,
			logger,
			tlsManager.RootCAs(),
			status,
			certChain,
			serverName,
		)
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
// it is used to accumulate the validation results.  logger and rootCAs must not
// be nil. Other parameters are optional.  If ok is true, the returned error, if
// any, is not critical.
func validateCertificate(
	ctx context.Context,
	logger *slog.Logger,
	rootCAs *x509.CertPool,
	status *TLSConfigStatus,
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
