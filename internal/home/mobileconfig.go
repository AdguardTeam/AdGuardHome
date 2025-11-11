package home

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/AdguardTeam/AdGuardHome/internal/client"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/httphdr"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil/urlutil"
	"github.com/google/uuid"
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

	// ServerAddresses is a list IP addresses of the server.
	//
	// TODO(a.garipov): Allow users to set this.
	//
	// See https://github.com/AdguardTeam/AdGuardHome/issues/3607.
	ServerAddresses []net.IP `plist:",omitempty"`
}

// payloadContent is a Device Management Profile payload.
//
// See https://developer.apple.com/documentation/devicemanagement/configuring_multiple_devices_using_profiles#3234127.
type payloadContent struct {
	DNSSettings *dnsSettings

	OnDemandEnabled int
	OnDemandRules   []*onDemandRule

	PayloadType        string
	PayloadIdentifier  string
	PayloadDisplayName string
	PayloadDescription string
	PayloadUUID        string
	PayloadVersion     int
}

// onDemandRule determines which queries use the DNS server.
//
// See https://developer.apple.com/documentation/devicemanagement/dnssettings/ondemandruleselement.
type onDemandRule struct {
	Action string
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
	PayloadType              string
	PayloadContent           []*payloadContent
	PayloadIdentifier        string
	PayloadUUID              string
	PayloadVersion           int
	PayloadRemovalDisallowed bool
	PayloadScope             string
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
			Scheme: urlutil.SchemeHTTPS,
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

	payloadID := fmt.Sprintf("%s.%s", dnsSettingsPayloadType, uuid.New())
	data := &mobileConfig{
		PayloadDescription: "Adds AdGuard Home to macOS Big Sur and iOS 14 or newer systems",
		PayloadDisplayName: dspName,
		PayloadType:        "Configuration",
		PayloadScope:       "System",
		PayloadContent: []*payloadContent{{
			DNSSettings:     d,
			OnDemandEnabled: 1,
			OnDemandRules: []*onDemandRule{{
				Action: "Connect",
			}},
			PayloadType:        dnsSettingsPayloadType,
			PayloadIdentifier:  payloadID,
			PayloadDisplayName: dspName,
			PayloadDescription: "Configures device to use AdGuard Home",
			PayloadUUID:        strings.ToUpper(uuid.New().String()),
			PayloadVersion:     1,
		}},
		PayloadIdentifier:        strings.ToUpper(uuid.New().String()),
		PayloadUUID:              strings.ToUpper(uuid.New().String()),
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
		err = client.ValidateClientID(clientID)
		if err != nil {
			respondJSONError(w, http.StatusBadRequest, err.Error())

			return
		}
	}

	d := &dnsSettings{
		DNSProtocol: dnsp,
		ServerName:  host,
	}

	mobileconfig, err := encodeMobileConfig(d, clientID)
	if err != nil {
		respondJSONError(w, http.StatusInternalServerError, err.Error())

		return
	}

	w.Header().Set(httphdr.ContentType, "application/xml")

	const (
		dohContDisp = `attachment; filename=doh.mobileconfig`
		dotContDisp = `attachment; filename=dot.mobileconfig`
	)

	contDisp := dohContDisp
	if dnsp == dnsProtoTLS {
		contDisp = dotContDisp
	}

	w.Header().Set(httphdr.ContentDisposition, contDisp)

	_, _ = w.Write(mobileconfig)
}

func handleMobileConfigDoH(w http.ResponseWriter, r *http.Request) {
	handleMobileConfig(w, r, dnsProtoHTTPS)
}

func handleMobileConfigDoT(w http.ResponseWriter, r *http.Request) {
	handleMobileConfig(w, r, dnsProtoTLS)
}
