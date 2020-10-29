package home

import (
	"fmt"
	"net"
	"net/http"

	uuid "github.com/satori/go.uuid"
	"howett.net/plist"
)

type DNSSettings struct {
	DNSProtocol string
	ServerURL   string `plist:",omitempty"`
	ServerName  string `plist:",omitempty"`
}

type PayloadContent struct {
	Name               string
	PayloadDescription string
	PayloadDisplayName string
	PayloadIdentifier  string
	PayloadType        string
	PayloadUUID        string
	PayloadVersion     int
	DNSSettings        DNSSettings
}

type MobileConfig struct {
	PayloadContent           []PayloadContent
	PayloadDescription       string
	PayloadDisplayName       string
	PayloadIdentifier        string
	PayloadRemovalDisallowed bool
	PayloadType              string
	PayloadUUID              string
	PayloadVersion           int
}

func genUUIDv4() string {
	return uuid.NewV4().String()
}

const (
	dnsProtoHTTPS = "HTTPS"
	dnsProtoTLS   = "TLS"
)

func getMobileConfig(d DNSSettings) ([]byte, error) {
	var name string
	switch d.DNSProtocol {
	case dnsProtoHTTPS:
		name = fmt.Sprintf("%s DoH", d.ServerName)
	case dnsProtoTLS:
		name = fmt.Sprintf("%s DoT", d.ServerName)
	default:
		return nil, fmt.Errorf("bad dns protocol %q", d.DNSProtocol)
	}

	data := MobileConfig{
		PayloadContent: []PayloadContent{{
			Name:               name,
			PayloadDescription: "Configures device to use AdGuard Home",
			PayloadDisplayName: name,
			PayloadIdentifier:  fmt.Sprintf("com.apple.dnsSettings.managed.%s", genUUIDv4()),
			PayloadType:        "com.apple.dnsSettings.managed",
			PayloadUUID:        genUUIDv4(),
			PayloadVersion:     1,
			DNSSettings:        d,
		}},
		PayloadDescription:       "Adds AdGuard Home to Big Sur and iOS 14 or newer systems",
		PayloadDisplayName:       name,
		PayloadIdentifier:        genUUIDv4(),
		PayloadRemovalDisallowed: false,
		PayloadType:              "Configuration",
		PayloadUUID:              genUUIDv4(),
		PayloadVersion:           1,
	}

	return plist.MarshalIndent(data, plist.XMLFormat, "\t")
}

func handleMobileConfig(w http.ResponseWriter, d DNSSettings) {
	mobileconfig, err := getMobileConfig(d)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "plist.MarshalIndent: %s", err)
	}

	w.Header().Set("Content-Type", "application/xml")
	_, _ = w.Write(mobileconfig)
}

func handleMobileConfigDoh(w http.ResponseWriter, r *http.Request) {
	handleMobileConfig(w, DNSSettings{
		DNSProtocol: dnsProtoHTTPS,
		ServerURL:   fmt.Sprintf("https://%s/dns-query", r.Host),
	})
}

func handleMobileConfigDot(w http.ResponseWriter, r *http.Request) {
	var err error

	var host string
	host, _, err = net.SplitHostPort(r.Host)
	if err != nil {
		httpError(w, http.StatusBadRequest, "getting host: %s", err)
	}

	handleMobileConfig(w, DNSSettings{
		DNSProtocol: dnsProtoTLS,
		ServerName:  host,
	})
}
