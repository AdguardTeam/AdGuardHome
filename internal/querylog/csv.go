package querylog

import (
	"strconv"
	"strings"
	"time"

	"github.com/AdguardTeam/golibs/log"
	"github.com/miekg/dns"
)

// csvHeaderRow is a slice of strings with row names for CSV header.  This
// slice should correspond with [logEntry.toCSV] func.
var csvHeaderRow = []string{
	"ans_dnssec",
	"ans_rcode",
	"ans_type",
	"ans_value",
	"cached",
	"client_ip",
	"client_id",
	"ecs",
	"elapsed",
	"filter_id",
	"filter_rule",
	"proto",
	"qclass",
	"qname",
	"qtype",
	"reason",
	"time",
	"upstream",
}

// toCSV returns a slice of strings with entry fields according to the
// csvHeaderRow slice.
func (e *logEntry) toCSV() (out []string) {
	var filterID, filterRule string

	if e.Result.IsFiltered && len(e.Result.Rules) > 0 {
		rule := e.Result.Rules[0]
		filterID = strconv.FormatInt(rule.FilterListID, 10)
		filterRule = rule.Text
	}

	aData := ansData(e)

	return []string{
		strconv.FormatBool(e.AuthenticatedData),
		aData.rCode,
		aData.typ,
		aData.value,
		strconv.FormatBool(e.Cached),
		e.IP.String(),
		e.ClientID,
		e.ReqECS,
		strconv.FormatFloat(e.Elapsed.Seconds()*1000, 'f', -1, 64),
		filterID,
		filterRule,
		string(e.ClientProto),
		e.QClass,
		e.QHost,
		e.QType,
		e.Result.Reason.String(),
		e.Time.Format(time.RFC3339Nano),
		e.Upstream,
	}
}

// csvAnswer is a helper struct for csv row answer fields.
type csvAnswer struct {
	rCode string
	typ   string
	value string
}

// ansData returns a map with message answer data.
func ansData(entry *logEntry) (out csvAnswer) {
	if len(entry.Answer) == 0 {
		return out
	}

	msg := &dns.Msg{}
	if err := msg.Unpack(entry.Answer); err != nil {
		log.Debug("querylog: failed to unpack dns msg answer: %v: %s", entry.Answer, err)

		return out
	}

	out.rCode = dns.RcodeToString[msg.Rcode]

	if len(msg.Answer) == 0 {
		return out
	}

	rr := msg.Answer[0]
	header := rr.Header()

	out.typ = dns.TypeToString[header.Rrtype]

	// Remove the header string from the answer value since it's mostly
	// unnecessary in the log.
	out.value = strings.TrimPrefix(rr.String(), header.String())

	return out
}
