package home

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghalg"
	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/dnsforward"
	"github.com/AdguardTeam/golibs/log"
	"github.com/google/go-cmp/cmp"
)

// Encryption Settings HTTP API

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

// tlsConfigResp is the TLS configuration and status response.
type tlsConfigResp struct {
	*tlsConfigStatus
	*tlsConfiguration

	// PrivateKeySaved is true if the private key is saved as a string and omit
	// key from answer.
	PrivateKeySaved bool `yaml:"-" json:"private_key_saved"`
}

// tlsConfigReq is the TLS configuration request.
type tlsConfigReq struct {
	tlsConfiguration

	// PrivateKeySaved is true if the private key is saved as a string and omit
	// key from answer.
	PrivateKeySaved bool `yaml:"-" json:"private_key_saved"`
}

// handleTLSStatus is the handler for the GET /control/tls/status HTTP API.
func (m *tlsManager) handleTLSStatus(w http.ResponseWriter, r *http.Request) {
	var resp *tlsConfigResp
	func() {
		m.mu.RLock()
		defer m.mu.RUnlock()

		resp = &tlsConfigResp{
			tlsConfigStatus:  m.status,
			tlsConfiguration: m.conf.cloneForEncoding(),
		}
	}()

	marshalTLS(w, r, resp)
}

// handleTLSValidate is the handler for the POST /control/tls/validate HTTP API.
func (m *tlsManager) handleTLSValidate(w http.ResponseWriter, r *http.Request) {
	req, err := unmarshalTLS(r)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "Failed to unmarshal TLS config: %s", err)

		return
	}

	if req.PrivateKeySaved {
		req.PrivateKey = m.confForEncoding().PrivateKey
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

	if !webCheckPortAvailable(req.PortHTTPS) {
		aghhttp.Error(
			r,
			w,
			http.StatusBadRequest,
			"port %d is not available, cannot enable HTTPS on it",
			req.PortHTTPS,
		)

		return
	}

	resp := &tlsConfigResp{
		tlsConfigStatus:  &tlsConfigStatus{},
		tlsConfiguration: &req.tlsConfiguration,
	}

	// Skip the error check, since we are only interested in the value of
	// resl.tlsConfigStatus.WarningValidation.
	_ = loadTLSConf(resp.tlsConfiguration, resp.tlsConfigStatus)

	marshalTLS(w, r, resp)
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

// handleTLSConfigure is the handler for the POST /control/tls/configure HTTP
// API.
func (m *tlsManager) handleTLSConfigure(w http.ResponseWriter, r *http.Request) {
	req, err := unmarshalTLS(r)
	if err != nil {
		aghhttp.Error(r, w, http.StatusBadRequest, "Failed to unmarshal TLS config: %s", err)

		return
	}

	if req.PrivateKeySaved {
		req.PrivateKey = m.confForEncoding().PrivateKey
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

	resp := &tlsConfigResp{
		tlsConfigStatus:  &tlsConfigStatus{},
		tlsConfiguration: &req.tlsConfiguration,
	}
	err = loadTLSConf(resp.tlsConfiguration, resp.tlsConfigStatus)
	if err != nil {
		marshalTLS(w, r, resp)

		return
	}

	restartRequired := m.setConf(resp)
	onConfigModified()

	err = reconfigureDNSServer()
	if err != nil {
		aghhttp.Error(r, w, http.StatusInternalServerError, "%s", err)

		return
	}

	resp.tlsConfiguration = m.confForEncoding()
	marshalTLS(w, r, resp)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}

	// The background context is used because the TLSConfigChanged wraps context
	// with timeout on its own and shuts down the server, which handles current
	// request. It is also should be done in a separate goroutine due to the
	// same reason.
	if restartRequired {
		go func() {
			Context.web.TLSConfigChanged(context.Background(), resp.tlsConfiguration)
		}()
	}
}

// setConf sets the necessary values from the new configuration.
func (m *tlsManager) setConf(newConf *tlsConfigResp) (restartRequired bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Reset the DNSCrypt data before comparing, since we currently do not
	// accept these from the frontend.
	//
	// TODO(a.garipov): Define a custom comparer for dnsforward.TLSConfig.
	newConf.DNSCryptConfigFile = m.conf.DNSCryptConfigFile
	newConf.PortDNSCrypt = m.conf.PortDNSCrypt
	if !cmp.Equal(m.conf, newConf, cmp.AllowUnexported(dnsforward.TLSConfig{})) {
		log.Info("tls: config has changed, restarting https server")
		restartRequired = true
	} else {
		log.Info("tls: config has not changed")
	}

	// Do not just write "m.conf = *newConf.tlsConfiguration", because all other
	// members of m.conf must be preserved.
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

	m.setCertFileTime()

	m.status = newConf.tlsConfigStatus

	return restartRequired
}

// marshalTLS handles Base64-encoded certificates transparently.
func marshalTLS(w http.ResponseWriter, r *http.Request, conf *tlsConfigResp) {
	if conf.CertificateChain != "" {
		encoded := base64.StdEncoding.EncodeToString([]byte(conf.CertificateChain))
		conf.CertificateChain = encoded
	}

	if conf.PrivateKey != "" {
		conf.PrivateKeySaved = true
		conf.PrivateKey = ""
	}

	_ = aghhttp.WriteJSONResponse(w, r, conf)
}

// unmarshalTLS handles Base64-encoded certificates transparently.
func unmarshalTLS(r *http.Request) (req *tlsConfigReq, err error) {
	req = &tlsConfigReq{}
	err = json.NewDecoder(r.Body).Decode(req)
	if err != nil {
		return nil, fmt.Errorf("parsing tls config: %w", err)
	}

	if req.CertificateChain != "" {
		var cert []byte
		cert, err = base64.StdEncoding.DecodeString(req.CertificateChain)
		if err != nil {
			return nil, fmt.Errorf("failed to base64-decode certificate chain: %w", err)
		}

		req.CertificateChain = string(cert)
		if req.CertificatePath != "" {
			return nil, fmt.Errorf("certificate data and file can't be set together")
		}
	}

	if req.PrivateKey != "" {
		var key []byte
		key, err = base64.StdEncoding.DecodeString(req.PrivateKey)
		if err != nil {
			return nil, fmt.Errorf("failed to base64-decode private key: %w", err)
		}

		req.PrivateKey = string(key)
		if req.PrivateKeyPath != "" {
			return nil, fmt.Errorf("private key data and file can't be set together")
		}
	}

	return req, nil
}

// registerWebHandlers registers HTTP handlers for TLS configuration.
func (m *tlsManager) registerWebHandlers() {
	httpRegister(http.MethodGet, "/control/tls/status", m.handleTLSStatus)
	httpRegister(http.MethodPost, "/control/tls/configure", m.handleTLSConfigure)
	httpRegister(http.MethodPost, "/control/tls/validate", m.handleTLSValidate)
}
