package querylog

import (
	"net"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/miekg/dns"
)

// logEntry represents a single entry in the file.
type logEntry struct {
	// client is the found client information, if any.
	client *Client

	Time time.Time `json:"T"`

	QHost  string `json:"QH"`
	QType  string `json:"QT"`
	QClass string `json:"QC"`

	ReqECS string `json:"ECS,omitempty"`

	ClientID    string      `json:"CID,omitempty"`
	ClientProto ClientProto `json:"CP"`

	Upstream string `json:",omitempty"`

	Answer     []byte `json:",omitempty"`
	OrigAnswer []byte `json:",omitempty"`

	IP net.IP `json:"IP"`

	Result filtering.Result

	Elapsed time.Duration

	Cached            bool `json:",omitempty"`
	AuthenticatedData bool `json:"AD,omitempty"`
}

// shallowClone returns a shallow clone of e.
func (e *logEntry) shallowClone() (clone *logEntry) {
	cloneVal := *e

	return &cloneVal
}

// addResponse adds data from resp to e.Answer if resp is not nil.  If isOrig is
// true, addResponse sets the e.OrigAnswer field instead of e.Answer.  Any
// errors are logged.
func (e *logEntry) addResponse(resp *dns.Msg, isOrig bool) {
	if resp == nil {
		return
	}

	var err error
	if isOrig {
		e.OrigAnswer, err = resp.Pack()
		err = errors.Annotate(err, "packing orig answer: %w")
	} else {
		e.Answer, err = resp.Pack()
		err = errors.Annotate(err, "packing answer: %w")
	}
	if err != nil {
		log.Error("querylog: %s", err)
	}
}

// parseDNSRewriteResultIPs fills logEntry's DNSRewriteResult response records
// with the IP addresses parsed from the raw strings.
func (e *logEntry) parseDNSRewriteResultIPs() {
	for rrType, rrValues := range e.Result.DNSRewriteResult.Response {
		switch rrType {
		case dns.TypeA, dns.TypeAAAA:
			for i, v := range rrValues {
				s, _ := v.(string)
				rrValues[i] = net.ParseIP(s)
			}
		default:
			// Go on.
		}
	}
}
