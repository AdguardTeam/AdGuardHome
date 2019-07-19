// Control: TLS configuring handlers

package home

import (
	"context"
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
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/AdguardTeam/golibs/log"
	"github.com/joomcode/errorx"
)

// RegisterTLSHandlers registers HTTP handlers for TLS configuration
func RegisterTLSHandlers() {
	http.HandleFunc("/control/tls/status", postInstall(optionalAuth(ensureGET(handleTLSStatus))))
	http.HandleFunc("/control/tls/configure", postInstall(optionalAuth(ensurePOST(handleTLSConfigure))))
	http.HandleFunc("/control/tls/validate", postInstall(optionalAuth(ensurePOST(handleTLSValidate))))
}

func handleTLSStatus(w http.ResponseWriter, r *http.Request) {
	log.Tracef("%s %v", r.Method, r.URL)
	marshalTLS(w, config.TLS)
}

func handleTLSValidate(w http.ResponseWriter, r *http.Request) {
	log.Tracef("%s %v", r.Method, r.URL)
	data, err := unmarshalTLS(r)
	if err != nil {
		httpError(w, http.StatusBadRequest, "Failed to unmarshal TLS config: %s", err)
		return
	}

	// check if port is available
	// BUT: if we are already using this port, no need
	alreadyRunning := false
	if config.httpsServer.server != nil {
		alreadyRunning = true
	}
	if !alreadyRunning {
		err = checkPortAvailable(config.BindHost, data.PortHTTPS)
		if err != nil {
			httpError(w, http.StatusBadRequest, "port %d is not available, cannot enable HTTPS on it", data.PortHTTPS)
			return
		}
	}

	data.tlsConfigStatus = validateCertificates(data.CertificateChain, data.PrivateKey, data.ServerName)
	marshalTLS(w, data)
}

func handleTLSConfigure(w http.ResponseWriter, r *http.Request) {
	log.Tracef("%s %v", r.Method, r.URL)
	data, err := unmarshalTLS(r)
	if err != nil {
		httpError(w, http.StatusBadRequest, "Failed to unmarshal TLS config: %s", err)
		return
	}

	// check if port is available
	// BUT: if we are already using this port, no need
	alreadyRunning := false
	if config.httpsServer.server != nil {
		alreadyRunning = true
	}
	if !alreadyRunning {
		err = checkPortAvailable(config.BindHost, data.PortHTTPS)
		if err != nil {
			httpError(w, http.StatusBadRequest, "port %d is not available, cannot enable HTTPS on it", data.PortHTTPS)
			return
		}
	}

	restartHTTPS := false
	data.tlsConfigStatus = validateCertificates(data.CertificateChain, data.PrivateKey, data.ServerName)
	if !reflect.DeepEqual(config.TLS.tlsConfigSettings, data.tlsConfigSettings) {
		log.Printf("tls config settings have changed, will restart HTTPS server")
		restartHTTPS = true
	}
	config.TLS = data
	err = writeAllConfigsAndReloadDNS()
	if err != nil {
		httpError(w, http.StatusInternalServerError, "Couldn't write config file: %s", err)
		return
	}
	marshalTLS(w, data)
	// this needs to be done in a goroutine because Shutdown() is a blocking call, and it will block
	// until all requests are finished, and _we_ are inside a request right now, so it will block indefinitely
	if restartHTTPS {
		go func() {
			time.Sleep(time.Second) // TODO: could not find a way to reliably know that data was fully sent to client by https server, so we wait a bit to let response through before closing the server
			config.httpsServer.cond.L.Lock()
			config.httpsServer.cond.Broadcast()
			if config.httpsServer.server != nil {
				config.httpsServer.server.Shutdown(context.TODO())
			}
			config.httpsServer.cond.L.Unlock()
		}()
	}
}

func verifyCertChain(data *tlsConfigStatus, certChain string, serverName string) error {
	log.Tracef("got certificate: %s", certChain)

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
func unmarshalTLS(r *http.Request) (tlsConfig, error) {
	data := tlsConfig{}
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
	}

	if data.PrivateKey != "" {
		keyPEM, err := base64.StdEncoding.DecodeString(data.PrivateKey)
		if err != nil {
			return data, errorx.Decorate(err, "Failed to base64-decode private key")
		}

		data.PrivateKey = string(keyPEM)
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
