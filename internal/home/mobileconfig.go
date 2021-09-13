package home

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"

	"github.com/AdguardTeam/AdGuardHome/internal/dnsforward"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/stringutil"
	uuid "github.com/satori/go.uuid"
	"howett.net/plist"
)

// dnsSettings is the DNSSetting.DNSSettings mobileconfig profile.
//
// See https://developer.apple.com/documentation/devicemanagement/dnssettings/dnssettings.
type dnsSettings struct {
	// DNSProtocol is the required protocol to be used.  The valid values
	// are "HTTPS" and "TLS".
	DNSProtocol string

	// ServerURL is the URI template of the DoH server.  It must be empty if
	// DNSProtocol is not "HTTPS".
	ServerURL string `plist:",omitempty"`

	// ServerName is the hostname of the DoT server.  It must be empty if
	// DNSProtocol is not "TLS".
	ServerName string `plist:",omitempty"`

	// ServerAddresses is a list of plain DNS server IP addresses used to
	// resolve the hostname in ServerURL or ServerName.
	ServerAddresses []string `plist:",omitempty"`
}

// payloadContent is a Device Management Profile payload.
//
// See https://developer.apple.com/documentation/devicemanagement/configuring_multiple_devices_using_profiles#3234127.
type payloadContent struct {
	DNSSettings *dnsSettings

	PayloadType        string
	PayloadIdentifier  string
	PayloadUUID        string
	PayloadDisplayName string
	PayloadDescription string
	PayloadVersion     int
}

// dnsSettingsPayloadType is the payload type for a DNSSettings profile.
const dnsSettingsPayloadType = "com.apple.dnsSettings.managed"

// mobileConfig contains the TopLevel properties for configuring Device
// Management Profiles.
//
// See https://developer.apple.com/documentation/devicemanagement/toplevel.
type mobileConfig struct {
	PayloadDescription       string
	PayloadDisplayName       string
	PayloadIdentifier        string
	PayloadType              string
	PayloadUUID              string
	PayloadContent           []*payloadContent
	PayloadVersion           int
	PayloadRemovalDisallowed bool
}

func genUUIDv4() string {
	return uuid.NewV4().String()
}

const (
	dnsProtoHTTPS = "HTTPS"
	dnsProtoTLS   = "TLS"
)

func encodeMobileConfig(d *dnsSettings, clientID string) ([]byte, error) {
	var dspName string
	switch proto := d.DNSProtocol; proto {
	case dnsProtoHTTPS:
		dspName = fmt.Sprintf("%s DoH", d.ServerName)
		u := &url.URL{
			Scheme: schemeHTTPS,
			Host:   d.ServerName,
			Path:   path.Join("/dns-query", clientID),
		}
		d.ServerURL = u.String()

		// Empty the ServerName field since it is only must be presented
		// in DNS-over-TLS configuration.
		d.ServerName = ""
	case dnsProtoTLS:
		dspName = fmt.Sprintf("%s DoT", d.ServerName)
		if clientID != "" {
			d.ServerName = clientID + "." + d.ServerName
		}
	default:
		return nil, fmt.Errorf("bad dns protocol %q", proto)
	}

	payloadID := fmt.Sprintf("%s.%s", dnsSettingsPayloadType, genUUIDv4())
	data := &mobileConfig{
		PayloadDescription: "Adds AdGuard Home to macOS Big Sur " +
			"and iOS 14 or newer systems",
		PayloadDisplayName: dspName,
		PayloadIdentifier:  genUUIDv4(),
		PayloadType:        "Configuration",
		PayloadUUID:        genUUIDv4(),
		PayloadContent: []*payloadContent{{
			PayloadType:        dnsSettingsPayloadType,
			PayloadIdentifier:  payloadID,
			PayloadUUID:        genUUIDv4(),
			PayloadDisplayName: dspName,
			PayloadDescription: "Configures device to use AdGuard Home",
			PayloadVersion:     1,
			DNSSettings:        d,
		}},
		PayloadVersion:           1,
		PayloadRemovalDisallowed: false,
	}

	return plist.MarshalIndent(data, plist.XMLFormat, "\t")
}

func respondJSONError(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(http.StatusInternalServerError)
	err := json.NewEncoder(w).Encode(&jsonError{
		Message: msg,
	})
	if err != nil {
		log.Debug("writing %d json response: %s", status, err)
	}
}

const errEmptyHost errors.Error = "no host in query parameters and no server_name"

func handleMobileConfig(w http.ResponseWriter, r *http.Request, dnsp string) {
	var err error

	q := r.URL.Query()
	host := q.Get("host")
	if host == "" {
		respondJSONError(w, http.StatusInternalServerError, string(errEmptyHost))

		return
	}

	clientID := q.Get("client_id")
	if clientID != "" {
		err = dnsforward.ValidateClientID(clientID)
		if err != nil {
			respondJSONError(w, http.StatusBadRequest, err.Error())

			return
		}
	}

	d := &dnsSettings{
		DNSProtocol:     dnsp,
		ServerName:      host,
		ServerAddresses: cloneBootstrap(),
	}

	mobileconfig, err := encodeMobileConfig(d, clientID)
	if err != nil {
		respondJSONError(w, http.StatusInternalServerError, err.Error())

		return
	}

	w.Header().Set("Content-Type", "application/xml")

	const (
		dohContDisp = `attachment; filename=doh.mobileconfig`
		dotContDisp = `attachment; filename=dot.mobileconfig`
	)

	contDisp := dohContDisp
	if dnsp == dnsProtoTLS {
		contDisp = dotContDisp
	}

	w.Header().Set("Content-Disposition", contDisp)

	_, _ = w.Write(mobileconfig)
}

// cloneBootstrap returns a clone of the current bootstrap DNS servers.
func cloneBootstrap() (bootstrap []string) {
	config.RLock()
	defer config.RUnlock()

	return stringutil.CloneSlice(config.DNS.BootstrapDNS)
}

func handleMobileConfigDoH(w http.ResponseWriter, r *http.Request) {
	handleMobileConfig(w, r, dnsProtoHTTPS)
}

func handleMobileConfigDoT(w http.ResponseWriter, r *http.Request) {
	handleMobileConfig(w, r, dnsProtoTLS)
}
