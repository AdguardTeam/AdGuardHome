package home

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/AdguardTeam/golibs/log"
	"github.com/joomcode/errorx"
)

var tlsWebHandlersRegistered = false

// TLSMod - TLS module object
type TLSMod struct {
	certLastMod time.Time // last modification time of the certificate file
	conf        tlsConfigSettings
	confLock    sync.Mutex
	status      tlsConfigStatus
}

// Create TLS module
func tlsCreate(conf tlsConfigSettings) *TLSMod {
	t := &TLSMod{}
	t.conf = conf
	if t.conf.Enabled {
		if !t.load() {
			return nil
		}
		t.setCertFileTime()
	}
	return t
}

func (t *TLSMod) load() bool {
	if !tlsLoadConfig(&t.conf, &t.status) {
		return false
	}

	// validate current TLS config and update warnings (it could have been loaded from file)
	data := validateCertificates(string(t.conf.CertificateChainData), string(t.conf.PrivateKeyData), t.conf.ServerName)
	if !data.ValidPair {
		log.Error(data.WarningValidation)
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

// Start - start the module
func (t *TLSMod) Start() {
	if !tlsWebHandlersRegistered {
		tlsWebHandlersRegistered = true
		t.registerWebHandlers()
	}

	t.confLock.Lock()
	tlsConf := t.conf
	t.confLock.Unlock()
	Context.web.TLSConfigChanged(tlsConf)
}

// Reload - reload certificate file
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
	Context.web.TLSConfigChanged(tlsConf)
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
		tls.CertificateChainData, err = ioutil.ReadFile(tls.CertificatePath)
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
		tls.PrivateKeyData, err = ioutil.ReadFile(tls.PrivateKeyPath)
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
	tlsConfigSettings `json:",inline"`
	tlsConfigStatus   `json:",inline"`
}

func (t *TLSMod) handleTLSStatus(w http.ResponseWriter, r *http.Request) {
	t.confLock.Lock()
	data := tlsConfig{
		tlsConfigSettings: t.conf,
		tlsConfigStatus:   t.status,
	}
	t.confLock.Unlock()
	marshalTLS(w, data)
}

func (t *TLSMod) handleTLSValidate(w http.ResponseWriter, r *http.Request) {
	setts, err := unmarshalTLS(r)
	if err != nil {
		httpError(w, http.StatusBadRequest, "Failed to unmarshal TLS config: %s", err)
		return
	}

	if !WebCheckPortAvailable(setts.PortHTTPS) {
		httpError(w, http.StatusBadRequest, "port %d is not available, cannot enable HTTPS on it", setts.PortHTTPS)
		return
	}

	status := tlsConfigStatus{}
	if tlsLoadConfig(&setts, &status) {
		status = validateCertificates(string(setts.CertificateChainData), string(setts.PrivateKeyData), setts.ServerName)
	}

	data := tlsConfig{
		tlsConfigSettings: setts,
		tlsConfigStatus:   status,
	}
	marshalTLS(w, data)
}

func (t *TLSMod) handleTLSConfigure(w http.ResponseWriter, r *http.Request) {
	data, err := unmarshalTLS(r)
	if err != nil {
		httpError(w, http.StatusBadRequest, "Failed to unmarshal TLS config: %s", err)
		return
	}

	if !WebCheckPortAvailable(data.PortHTTPS) {
		httpError(w, http.StatusBadRequest, "port %d is not available, cannot enable HTTPS on it", data.PortHTTPS)
		return
	}

	status := tlsConfigStatus{}
	if !tlsLoadConfig(&data, &status) {
		data2 := tlsConfig{
			tlsConfigSettings: data,
			tlsConfigStatus:   t.status,
		}
		marshalTLS(w, data2)
		return
	}
	status = validateCertificates(string(data.CertificateChainData), string(data.PrivateKeyData), data.ServerName)
	restartHTTPS := false
	t.confLock.Lock()
	if !reflect.DeepEqual(t.conf, data) {
		log.Printf("tls config settings have changed, will restart HTTPS server")
		restartHTTPS = true
	}
	// Note: don't do just `t.conf = data` because we must preserve all other members of t.conf
	t.conf.Enabled = data.Enabled
	t.conf.ServerName = data.ServerName
	t.conf.ForceHTTPS = data.ForceHTTPS
	t.conf.PortHTTPS = data.PortHTTPS
	t.conf.PortDNSOverTLS = data.PortDNSOverTLS
	t.conf.CertificateChain = data.CertificateChain
	t.conf.CertificatePath = data.CertificatePath
	t.conf.CertificateChainData = data.CertificateChainData
	t.conf.PrivateKey = data.PrivateKey
	t.conf.PrivateKeyPath = data.PrivateKeyPath
	t.conf.PrivateKeyData = data.PrivateKeyData
	t.status = status
	t.confLock.Unlock()
	t.setCertFileTime()
	onConfigModified()
	err = reconfigureDNSServer()
	if err != nil {
		httpError(w, http.StatusInternalServerError, "%s", err)
		return
	}
	data2 := tlsConfig{
		tlsConfigSettings: data,
		tlsConfigStatus:   t.status,
	}
	marshalTLS(w, data2)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	// this needs to be done in a goroutine because Shutdown() is a blocking call, and it will block
	// until all requests are finished, and _we_ are inside a request right now, so it will block indefinitely
	if restartHTTPS {
		go func() {
			Context.web.TLSConfigChanged(data)
		}()
	}
}

func verifyCertChain(data *tlsConfigStatus, certChain string, serverName string) error {
	log.Tracef("TLS: got certificate: %d bytes", len(certChain))

	// now do a more extended validation
	var certs []*pem.Block    // PEM-encoded certificates
	var skippedBytes []string // skipped bytes

	pemblock := []byte(certChain)
	for {
		var decoded *pem.Block
		decoded, pemblock = pem.Decode(pemblock)
		if decoded == nil {
			break
		}
		if decoded.Type == "CERTIFICATE" {
			certs = append(certs, decoded)
		} else {
			// ignore "this result of append is never used" warning
			// nolint
			skippedBytes = append(skippedBytes, decoded.Type)
		}
	}

	var parsedCerts []*x509.Certificate

	for _, cert := range certs {
		parsed, err := x509.ParseCertificate(cert.Bytes)
		if err != nil {
			data.WarningValidation = fmt.Sprintf("Failed to parse certificate: %s", err)
			return errors.New(data.WarningValidation)
		}
		parsedCerts = append(parsedCerts, parsed)
	}

	if len(parsedCerts) == 0 {
		data.WarningValidation = fmt.Sprintf("You have specified an empty certificate")
		return errors.New(data.WarningValidation)
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
	var key *pem.Block        // PEM-encoded certificates
	var skippedBytes []string // skipped bytes

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
		} else {
			// ignore "this result of append is never used"
			// nolint
			skippedBytes = append(skippedBytes, decoded.Type)
		}
	}

	if key == nil {
		data.WarningValidation = "No valid keys were found"
		return errors.New(data.WarningValidation)
	}

	// parse the decoded key
	_, keytype, err := parsePrivateKey(key.Bytes)
	if err != nil {
		data.WarningValidation = fmt.Sprintf("Failed to parse private key: %s", err)
		return errors.New(data.WarningValidation)
	}

	data.ValidKey = true
	data.KeyType = keytype
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

// Attempt to parse the given private key DER block. OpenSSL 0.9.8 generates
// PKCS#1 private keys by default, while OpenSSL 1.0.0 generates PKCS#8 keys.
// OpenSSL ecparam generates SEC1 EC private keys for ECDSA. We try all three.
func parsePrivateKey(der []byte) (crypto.PrivateKey, string, error) {
	if key, err := x509.ParsePKCS1PrivateKey(der); err == nil {
		return key, "RSA", nil
	}

	if key, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		switch key := key.(type) {
		case *rsa.PrivateKey:
			return key, "RSA", nil
		case *ecdsa.PrivateKey:
			return key, "ECDSA", nil
		default:
			return nil, "", errors.New("tls: found unknown private key type in PKCS#8 wrapping")
		}
	}

	if key, err := x509.ParseECPrivateKey(der); err == nil {
		return key, "ECDSA", nil
	}

	return nil, "", errors.New("tls: failed to parse private key")
}

// unmarshalTLS handles base64-encoded certificates transparently
func unmarshalTLS(r *http.Request) (tlsConfigSettings, error) {
	data := tlsConfigSettings{}
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		return data, errorx.Decorate(err, "Failed to parse new TLS config json")
	}

	if data.CertificateChain != "" {
		certPEM, err := base64.StdEncoding.DecodeString(data.CertificateChain)
		if err != nil {
			return data, errorx.Decorate(err, "Failed to base64-decode certificate chain")
		}
		data.CertificateChain = string(certPEM)
		if data.CertificatePath != "" {
			return data, fmt.Errorf("certificate data and file can't be set together")
		}
	}

	if data.PrivateKey != "" {
		keyPEM, err := base64.StdEncoding.DecodeString(data.PrivateKey)
		if err != nil {
			return data, errorx.Decorate(err, "Failed to base64-decode private key")
		}

		data.PrivateKey = string(keyPEM)
		if data.PrivateKeyPath != "" {
			return data, fmt.Errorf("private key data and file can't be set together")
		}
	}

	return data, nil
}

func marshalTLS(w http.ResponseWriter, data tlsConfig) {
	w.Header().Set("Content-Type", "application/json")

	if data.CertificateChain != "" {
		encoded := base64.StdEncoding.EncodeToString([]byte(data.CertificateChain))
		data.CertificateChain = encoded
	}

	if data.PrivateKey != "" {
		encoded := base64.StdEncoding.EncodeToString([]byte(data.PrivateKey))
		data.PrivateKey = encoded
	}

	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Failed to marshal json with TLS status: %s", err)
		return
	}
}

// registerWebHandlers registers HTTP handlers for TLS configuration
func (t *TLSMod) registerWebHandlers() {
	httpRegister("GET", "/control/tls/status", t.handleTLSStatus)
	httpRegister("POST", "/control/tls/configure", t.handleTLSConfigure)
	httpRegister("POST", "/control/tls/validate", t.handleTLSValidate)
}
