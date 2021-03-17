package home

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"

	"github.com/AdguardTeam/AdGuardHome/internal/dnsforward"
	"github.com/AdguardTeam/golibs/log"
	uuid "github.com/satori/go.uuid"
	"howett.net/plist"
)

type dnsSettings struct {
	DNSProtocol string
	ServerURL   string `plist:",omitempty"`
	ServerName  string `plist:",omitempty"`
	clientID    string
}

type payloadContent struct {
	Name               string
	PayloadDescription string
	PayloadDisplayName string
	PayloadIdentifier  string
	PayloadType        string
	PayloadUUID        string
	DNSSettings        dnsSettings
	PayloadVersion     int
}

type mobileConfig struct {
	PayloadDescription       string
	PayloadDisplayName       string
	PayloadIdentifier        string
	PayloadType              string
	PayloadUUID              string
	PayloadContent           []payloadContent
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

func getMobileConfig(d dnsSettings) ([]byte, error) {
	var dspName string
	switch d.DNSProtocol {
	case dnsProtoHTTPS:
		dspName = fmt.Sprintf("%s DoH", d.ServerName)

		u := &url.URL{
			Scheme: schemeHTTPS,
			Host:   d.ServerName,
			Path:   "/dns-query",
		}
		if d.clientID != "" {
			u.Path = path.Join(u.Path, d.clientID)
		}

		d.ServerURL = u.String()
	case dnsProtoTLS:
		dspName = fmt.Sprintf("%s DoT", d.ServerName)
		if d.clientID != "" {
			d.ServerName = d.clientID + "." + d.ServerName
		}
	default:
		return nil, fmt.Errorf("bad dns protocol %q", d.DNSProtocol)
	}

	data := mobileConfig{
		PayloadContent: []payloadContent{{
			Name:               dspName,
			PayloadDescription: "Configures device to use AdGuard Home",
			PayloadDisplayName: dspName,
			PayloadIdentifier:  fmt.Sprintf("com.apple.dnsSettings.managed.%s", genUUIDv4()),
			PayloadType:        "com.apple.dnsSettings.managed",
			PayloadUUID:        genUUIDv4(),
			PayloadVersion:     1,
			DNSSettings:        d,
		}},
		PayloadDescription:       "Adds AdGuard Home to Big Sur and iOS 14 or newer systems",
		PayloadDisplayName:       dspName,
		PayloadIdentifier:        genUUIDv4(),
		PayloadRemovalDisallowed: false,
		PayloadType:              "Configuration",
		PayloadUUID:              genUUIDv4(),
		PayloadVersion:           1,
	}

	return plist.MarshalIndent(data, plist.XMLFormat, "\t")
}

func handleMobileConfig(w http.ResponseWriter, r *http.Request, dnsp string) {
	var err error

	q := r.URL.Query()
	host := q.Get("host")
	if host == "" {
		host = Context.tls.conf.ServerName
	}

	if host == "" {
		w.WriteHeader(http.StatusInternalServerError)

		const msg = "no host in query parameters and no server_name"
		err = json.NewEncoder(w).Encode(&jsonError{
			Message: msg,
		})
		if err != nil {
			log.Debug("writing 500 json response: %s", err)
		}

		return
	}

	clientID := q.Get("client_id")
	err = dnsforward.ValidateClientID(clientID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)

		err = json.NewEncoder(w).Encode(&jsonError{
			Message: err.Error(),
		})
		if err != nil {
			log.Debug("writing 400 json response: %s", err)
		}

		return
	}

	d := dnsSettings{
		DNSProtocol: dnsp,
		ServerName:  host,
		clientID:    clientID,
	}

	mobileconfig, err := getMobileConfig(d)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "plist.MarshalIndent: %s", err)

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

func handleMobileConfigDOH(w http.ResponseWriter, r *http.Request) {
	handleMobileConfig(w, r, dnsProtoHTTPS)
}

func handleMobileConfigDOT(w http.ResponseWriter, r *http.Request) {
	handleMobileConfig(w, r, dnsProtoTLS)
}
