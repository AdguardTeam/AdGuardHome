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
	"path/filepath"
	"runtime"
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

var tlsWebHandlersRegistered = false

// TLSMod - TLS module object
type TLSMod struct {
	certLastMod time.Time // last modification time of the certificate file
	status      tlsConfigStatus
	confLock    sync.Mutex
	conf        tlsConfigSettings
}

// Create TLS module
func tlsCreate(conf tlsConfigSettings) *TLSMod {
	t := &TLSMod{}
	t.conf = conf
	if t.conf.Enabled {
		if !t.load() {
			// Something is not valid - return an empty TLS config
			return &TLSMod{conf: tlsConfigSettings{
				Enabled:             conf.Enabled,
				ServerName:          conf.ServerName,
				PortHTTPS:           conf.PortHTTPS,
				PortDNSOverTLS:      conf.PortDNSOverTLS,
				PortDNSOverQUIC:     conf.PortDNSOverQUIC,
				AllowUnencryptedDoH: conf.AllowUnencryptedDoH,
			}}
		}
		t.setCertFileTime()
	}
	return t
}

func (t *TLSMod) load() bool {
	if !tlsLoadConfig(&t.conf, &t.status) {
		log.Error("failed to load TLS config: %s", t.status.WarningValidation)
		return false
	}

	// validate current TLS config and update warnings (it could have been loaded from file)
	data := validateCertificates(string(t.conf.CertificateChainData), string(t.conf.PrivateKeyData), t.conf.ServerName)
	if !data.ValidPair {
		log.Error("failed to validate certificate: %s", data.WarningValidation)
		return false
	}
	t.status = data
	return true
}

// Close - close module
func (t *TLSMod) Close() {
}

// WriteDiskConfig - write config
func (t *TLSMod) WriteDiskConfig(conf *tlsConfigSettings) {
	t.confLock.Lock()
	*conf = t.conf
	t.confLock.Unlock()
}

func (t *TLSMod) setCertFileTime() {
	if len(t.conf.CertificatePath) == 0 {
		return
	}
	fi, err := os.Stat(t.conf.CertificatePath)
	if err != nil {
		log.Error("TLS: %s", err)
		return
	}
	t.certLastMod = fi.ModTime().UTC()
}

// Start updates the configuration of TLSMod and starts it.
func (t *TLSMod) Start() {
	if !tlsWebHandlersRegistered {
		tlsWebHandlersRegistered = true
		t.registerWebHandlers()
	}

	t.confLock.Lock()
	tlsConf := t.conf
	t.confLock.Unlock()

	// The background context is used because the TLSConfigChanged wraps
	// context with timeout on its own and shuts down the server, which
	// handles current request.
	Context.web.TLSConfigChanged(context.Background(), tlsConf)
}

// Reload updates the configuration of TLSMod and restarts it.
func (t *TLSMod) Reload() {
	t.confLock.Lock()
	tlsConf := t.conf
	t.confLock.Unlock()

	if !tlsConf.Enabled || len(tlsConf.CertificatePath) == 0 {
		return
	}
	fi, err := os.Stat(tlsConf.CertificatePath)
	if err != nil {
		log.Error("TLS: %s", err)
		return
	}
	if fi.ModTime().UTC().Equal(t.certLastMod) {
		log.Debug("TLS: certificate file isn't modified")
		return
	}
	log.Debug("TLS: certificate file is modified")

	t.confLock.Lock()
	r := t.load()
	t.confLock.Unlock()
	if !r {
		return
	}

	t.certLastMod = fi.ModTime().UTC()

	_ = reconfigureDNSServer()

	t.confLock.Lock()
	tlsConf = t.conf
	t.confLock.Unlock()
	// The background context is used because the TLSConfigChanged wraps
	// context with timeout on its own and shuts down the server, which
	// handles current request.
	Context.web.TLSConfigChanged(context.Background(), tlsConf)
}

// Set certificate and private key data
func tlsLoadConfig(tls *tlsConfigSettings, status *tlsConfigStatus) bool {
	tls.CertificateChainData = []byte(tls.CertificateChain)
	tls.PrivateKeyData = []byte(tls.PrivateKey)

	var err error
	if tls.CertificatePath != "" {
		if tls.CertificateChain != "" {
			status.WarningValidation = "certificate data and file can't be set together"
			return false
		}
		tls.CertificateChainData, err = os.ReadFile(tls.CertificatePath)
		if err != nil {
			status.WarningValidation = err.Error()
			return false
		}
		status.ValidCert = true
	}

	if tls.PrivateKeyPath != "" {
		if tls.PrivateKey != "" {
			status.WarningValidation = "private key data and file can't be set together"
			return false
		}
		tls.PrivateKeyData, err = os.ReadFile(tls.PrivateKeyPath)
		if err != nil {
			status.WarningValidation = err.Error()
			return false
		}
		status.ValidKey = true
	}

	return true
}

type tlsConfigStatus struct {
	ValidCert  bool      `json:"valid_cert"`           // ValidCert is true if the specified certificates chain is a valid chain of X509 certificates
	ValidChain bool      `json:"valid_chain"`          // ValidChain is true if the specified certificates chain is verified and issued by a known CA
	Subject    string    `json:"subject,omitempty"`    // Subject is the subject of the first certificate in the chain
	Issuer     string    `json:"issuer,omitempty"`     // Issuer is the issuer of the first certificate in the chain
	NotBefore  time.Time `json:"not_before,omitempty"` // NotBefore is the NotBefore field of the first certificate in the chain
	NotAfter   time.Time `json:"not_after,omitempty"`  // NotAfter is the NotAfter field of the first certificate in the chain
	DNSNames   []string  `json:"dns_names"`            // DNSNames is the value of SubjectAltNames field of the first certificate in the chain

	// key status
	ValidKey bool   `json:"valid_key"`          // ValidKey is true if the key is a valid private key
	KeyType  string `json:"key_type,omitempty"` // KeyType is one of RSA or ECDSA

	// is usable? set by validator
	ValidPair bool `json:"valid_pair"` // ValidPair is true if both certificate and private key are correct

	// warnings
	WarningValidation string `json:"warning_validation,omitempty"` // WarningValidation is a validation warning message with the issue description
}

// field ordering is important -- yaml fields will mirror ordering from here
type tlsConfig struct {
	tlsConfigStatus      `json:",inline"`
	tlsConfigSettingsExt `json:",inline"`
}

// tlsConfigSettingsExt is used to (un)marshal PrivateKeySaved to ensure that
// clients don't send and receive previously saved private keys.
type tlsConfigSettingsExt struct {
	tlsConfigSettings `json:",inline"`
	// If private key saved as a string, we set this flag to true
	// and omit key from answer.
	PrivateKeySaved bool `yaml:"-" json:"private_key_saved,inline"`
}

func (t *TLSMod) handleTLSStatus(w http.ResponseWriter, r *http.Request) {
	t.confLock.Lock()
	data := tlsConfig{
		tlsConfigSettingsExt: tlsConfigSettingsExt{
			tlsConfigSettings: t.conf,
		},
		tlsConfigStatus: t.status,
	}
	t.confLock.Unlock()
	marshalTLS(w, r, data)
}

func (t *TLSMod) handleTLSValidate(w http.ResponseWriter, r *http.Request) {
	setts, err := unmarshalTLS(r)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "Failed to unmarshal TLS config: %s", err)

		return
	}

	if setts.PrivateKeySaved {
		setts.PrivateKey = t.conf.PrivateKey
	}

	if setts.Enabled {
		uc := aghalg.UniqChecker{}
		addPorts(
			uc,
			tcpPort(config.BindPort),
			tcpPort(config.BetaBindPort),
			udpPort(config.DNS.Port),
			tcpPort(setts.PortHTTPS),
			tcpPort(setts.PortDNSOverTLS),
			udpPort(setts.PortDNSOverQUIC),
			tcpPort(setts.PortDNSCrypt),
		)

		err = uc.Validate(aghalg.IntIsBefore)
		if err != nil {
			aghhttp.Error(r, w, http.StatusBadRequest, "validating ports: %s", err)

			return
		}
	}

	if !WebCheckPortAvailable(setts.PortHTTPS) {
		aghhttp.Error(
			r,
			w,
			http.StatusBadRequest,
			"port %d is not available, cannot enable HTTPS on it",
			setts.PortHTTPS,
		)

		return
	}

	status := tlsConfigStatus{}
	if tlsLoadConfig(&setts.tlsConfigSettings, &status) {
		status = validateCertificates(string(setts.CertificateChainData), string(setts.PrivateKeyData), setts.ServerName)
	}

	data := tlsConfig{
		tlsConfigSettingsExt: setts,
		tlsConfigStatus:      status,
	}

	marshalTLS(w, r, data)
}

func (t *TLSMod) setConfig(newConf tlsConfigSettings, status tlsConfigStatus) (restartHTTPS bool) {
	t.confLock.Lock()
	defer t.confLock.Unlock()

	// Reset the DNSCrypt data before comparing, since we currently do not
	// accept these from the frontend.
	//
	// TODO(a.garipov): Define a custom comparer for dnsforward.TLSConfig.
	newConf.DNSCryptConfigFile = t.conf.DNSCryptConfigFile
	newConf.PortDNSCrypt = t.conf.PortDNSCrypt
	if !cmp.Equal(t.conf, newConf, cmp.AllowUnexported(dnsforward.TLSConfig{})) {
		log.Info("tls config has changed, restarting https server")
		restartHTTPS = true
	} else {
		log.Info("tls config has not changed")
	}

	// Note: don't do just `t.conf = data` because we must preserve all other members of t.conf
	t.conf.Enabled = newConf.Enabled
	t.conf.ServerName = newConf.ServerName
	t.conf.ForceHTTPS = newConf.ForceHTTPS
	t.conf.PortHTTPS = newConf.PortHTTPS
	t.conf.PortDNSOverTLS = newConf.PortDNSOverTLS
	t.conf.PortDNSOverQUIC = newConf.PortDNSOverQUIC
	t.conf.CertificateChain = newConf.CertificateChain
	t.conf.CertificatePath = newConf.CertificatePath
	t.conf.CertificateChainData = newConf.CertificateChainData
	t.conf.PrivateKey = newConf.PrivateKey
	t.conf.PrivateKeyPath = newConf.PrivateKeyPath
	t.conf.PrivateKeyData = newConf.PrivateKeyData
	t.status = status

	return restartHTTPS
}

func (t *TLSMod) handleTLSConfigure(w http.ResponseWriter, r *http.Request) {
	data, err := unmarshalTLS(r)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "Failed to unmarshal TLS config: %s", err)

		return
	}

	if data.PrivateKeySaved {
		data.PrivateKey = t.conf.PrivateKey
	}

	if data.Enabled {
		uc := aghalg.UniqChecker{}
		addPorts(
			uc,
			tcpPort(config.BindPort),
			tcpPort(config.BetaBindPort),
			udpPort(config.DNS.Port),
			tcpPort(data.PortHTTPS),
			tcpPort(data.PortDNSOverTLS),
			udpPort(data.PortDNSOverQUIC),
			tcpPort(data.PortDNSCrypt),
		)

		err = uc.Validate(aghalg.IntIsBefore)
		if err != nil {
			aghhttp.Error(r, w, http.StatusBadRequest, "%s", err)

			return
		}
	}

	// TODO(e.burkov):  Investigate and perhaps check other ports.
	if !WebCheckPortAvailable(data.PortHTTPS) {
		aghhttp.Error(
			r,
			w,
			http.StatusBadRequest,
			"port %d is not available, cannot enable HTTPS on it",
			data.PortHTTPS,
		)

		return
	}

	status := tlsConfigStatus{}
	if !tlsLoadConfig(&data.tlsConfigSettings, &status) {
		data2 := tlsConfig{
			tlsConfigSettingsExt: data,
			tlsConfigStatus:      t.status,
		}
		marshalTLS(w, r, data2)

		return
	}

	status = validateCertificates(string(data.CertificateChainData), string(data.PrivateKeyData), data.ServerName)

	restartHTTPS := t.setConfig(data.tlsConfigSettings, status)
	t.setCertFileTime()
	onConfigModified()

	err = reconfigureDNSServer()
	if err != nil {
		aghhttp.Error(r, w, http.StatusInternalServerError, "%s", err)

		return
	}

	data2 := tlsConfig{
		tlsConfigSettingsExt: data,
		tlsConfigStatus:      t.status,
	}

	marshalTLS(w, r, data2)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	// The background context is used because the TLSConfigChanged wraps context
	// with timeout on its own and shuts down the server, which handles current
	// request. It is also should be done in a separate goroutine due to the
	// same reason.
	if restartHTTPS {
		go func() {
			Context.web.TLSConfigChanged(context.Background(), data.tlsConfigSettings)
		}()
	}
}

func verifyCertChain(data *tlsConfigStatus, certChain, serverName string) error {
	log.Tracef("TLS: got certificate: %d bytes", len(certChain))

	// now do a more extended validation
	var certs []*pem.Block // PEM-encoded certificates

	pemblock := []byte(certChain)
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

	var parsedCerts []*x509.Certificate

	for _, cert := range certs {
		parsed, err := x509.ParseCertificate(cert.Bytes)
		if err != nil {
			data.WarningValidation = fmt.Sprintf("Failed to parse certificate: %s", err)
			return errors.Error(data.WarningValidation)
		}
		parsedCerts = append(parsedCerts, parsed)
	}

	if len(parsedCerts) == 0 {
		data.WarningValidation = "You have specified an empty certificate"
		return errors.Error(data.WarningValidation)
	}

	data.ValidCert = true

	// spew.Dump(parsedCerts)

	opts := x509.VerifyOptions{
		DNSName: serverName,
		Roots:   Context.tlsRoots,
	}

	log.Printf("number of certs - %d", len(parsedCerts))
	if len(parsedCerts) > 1 {
		// set up an intermediate
		pool := x509.NewCertPool()
		for _, cert := range parsedCerts[1:] {
			log.Printf("got an intermediate cert")
			pool.AddCert(cert)
		}
		opts.Intermediates = pool
	}

	// TODO: save it as a warning rather than error it out -- shouldn't be a big problem
	mainCert := parsedCerts[0]
	_, err := mainCert.Verify(opts)
	if err != nil {
		// let self-signed certs through
		data.WarningValidation = fmt.Sprintf("Your certificate does not verify: %s", err)
	} else {
		data.ValidChain = true
	}
	// spew.Dump(chains)

	// update status
	if mainCert != nil {
		notAfter := mainCert.NotAfter
		data.Subject = mainCert.Subject.String()
		data.Issuer = mainCert.Issuer.String()
		data.NotAfter = notAfter
		data.NotBefore = mainCert.NotBefore
		data.DNSNames = mainCert.DNSNames
	}

	return nil
}

func validatePkey(data *tlsConfigStatus, pkey string) error {
	// now do a more extended validation
	var key *pem.Block // PEM-encoded certificates

	// go through all pem blocks, but take first valid pem block and drop the rest
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
		data.WarningValidation = "No valid keys were found"

		return errors.Error(data.WarningValidation)
	}

	// parse the decoded key
	_, keyType, err := parsePrivateKey(key.Bytes)
	if err != nil {
		data.WarningValidation = fmt.Sprintf("Failed to parse private key: %s", err)

		return errors.Error(data.WarningValidation)
	} else if keyType == keyTypeED25519 {
		data.WarningValidation = "ED25519 keys are not supported by browsers; " +
			"did you mean to use X25519 for key exchange?"

		return errors.Error(data.WarningValidation)
	}

	data.ValidKey = true
	data.KeyType = keyType

	return nil
}

// Process certificate data and its private key.
// All parameters are optional.
// On error, return partially set object
//  with 'WarningValidation' field containing error description.
func validateCertificates(certChain, pkey, serverName string) tlsConfigStatus {
	var data tlsConfigStatus

	// check only public certificate separately from the key
	if certChain != "" {
		if verifyCertChain(&data, certChain, serverName) != nil {
			return data
		}
	}

	// validate private key (right now the only validation possible is just parsing it)
	if pkey != "" {
		if validatePkey(&data, pkey) != nil {
			return data
		}
	}

	// if both are set, validate both in unison
	if pkey != "" && certChain != "" {
		_, err := tls.X509KeyPair([]byte(certChain), []byte(pkey))
		if err != nil {
			data.WarningValidation = fmt.Sprintf("Invalid certificate or key: %s", err)
			return data
		}
		data.ValidPair = true
	}

	return data
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
	w.Header().Set("Content-Type", "application/json")

	if data.CertificateChain != "" {
		encoded := base64.StdEncoding.EncodeToString([]byte(data.CertificateChain))
		data.CertificateChain = encoded
	}

	if data.PrivateKey != "" {
		data.PrivateKeySaved = true
		data.PrivateKey = ""
	}

	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		aghhttp.Error(
			r,
			w,
			http.StatusInternalServerError,
			"Failed to marshal json with TLS status: %s",
			err,
		)
	}
}

// registerWebHandlers registers HTTP handlers for TLS configuration
func (t *TLSMod) registerWebHandlers() {
	httpRegister(http.MethodGet, "/control/tls/status", t.handleTLSStatus)
	httpRegister(http.MethodPost, "/control/tls/configure", t.handleTLSConfigure)
	httpRegister(http.MethodPost, "/control/tls/validate", t.handleTLSValidate)
}

// LoadSystemRootCAs tries to load root certificates from the operating system.
// It returns nil in case nothing is found so that that Go.crypto will use it's
// default algorithm to find system root CA list.
//
// See https://github.com/AdguardTeam/AdGuardHome/internal/issues/1311.
func LoadSystemRootCAs() (roots *x509.CertPool) {
	// TODO(e.burkov): Use build tags instead.
	if runtime.GOOS != "linux" {
		return nil
	}

	// Directories with the system root certificates, which aren't supported
	// by Go.crypto.
	dirs := []string{
		// Entware.
		"/opt/etc/ssl/certs",
	}
	roots = x509.NewCertPool()
	for _, dir := range dirs {
		dirEnts, err := os.ReadDir(dir)
		if errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			log.Error("opening directory: %q: %s", dir, err)
		}

		var rootsAdded bool
		for _, de := range dirEnts {
			var certData []byte
			certData, err = os.ReadFile(filepath.Join(dir, de.Name()))
			if err == nil && roots.AppendCertsFromPEM(certData) {
				rootsAdded = true
			}
		}

		if rootsAdded {
			return roots
		}
	}

	return nil
}
