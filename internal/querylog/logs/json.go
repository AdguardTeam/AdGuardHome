package logs

import (
	"fmt"
	"net"

	"github.com/AdguardTeam/AdGuardHome/internal/aghalg"
)

type ConfigPayload struct {
	// Interval is the querylog rotation interval.  Use float64 here to support
	// fractional numbers and not mess the API users by changing the units.
	Interval float64 `json:"interval"`

	// Enabled shows if the querylog is enabled.  It is an [aghalg.NullBool]
	// to be able to tell when it's set without using pointers.
	Enabled aghalg.NullBool `json:"enabled"`

	// AnonymizeClientIP shows if the clients' IP addresses must be anonymized.
	// It is an [aghalg.NullBool] to be able to tell when it's set without using
	// pointers.
	AnonymizeClientIP aghalg.NullBool `json:"anonymize_client_ip"`
}

type LogsPayload struct {
	Data   []*LogData `json:"data"`
	Oldest string     `json:"oldest,omitempty"`
}

type LogData struct {
	Answer         []Answer    `json:"answer,omitempty,omitempty"`
	AnswerDnssec   bool        `json:"answer_dnssec"`
	Cached         bool        `json:"cached"`
	Client         net.IP      `json:"client"`
	ClientId       string      `json:"client_id,omitempty"`
	ClientInfo     *Client     `json:"client_info,omitempty"`
	ClientProto    ClientProto `json:"client_proto"`
	ElapsedMs      string      `json:"elapsedMs"`
	OriginalAnswer []Answer    `json:"original_answer,omitempty"`
	Question       Question    `json:"question"`
	Reason         string      `json:"reason"`
	ReqECS         string      `json:"ecs,omitempty"`
	Rule           string      `json:"rule,omitempty"`
	FilterId       int64       `json:"filterId,omitempty"`
	Rules          []RuleEntry `json:"rules"`
	ServiceName    string      `json:"service_name,omitempty"`
	Status         string      `json:"status,omitempty"`
	Time           string      `json:"time"`
	Upstream       string      `json:"upstream"`
}

type Answer struct {
	Type  string `json:"type"`
	Value string `json:"value"`
	TTL   uint32 `json:"ttl"`
}
type Question struct {
	Class       string `json:"class"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	UnicodeName string `json:"unicode_name,omitempty"`
}
type RuleEntry struct {
	Text         string `json:"text"`
	FilterListId int64  `json:"filter_list_id"`
}

// ClientProto values are names of the client protocols.
type ClientProto string

// Client protocol names.
const (
	ClientProtoDoH      ClientProto = "doh"
	ClientProtoDoQ      ClientProto = "doq"
	ClientProtoDoT      ClientProto = "dot"
	ClientProtoDNSCrypt ClientProto = "dnscrypt"
	ClientProtoPlain    ClientProto = ""
)

// NewClientProto validates that the client protocol name is valid and returns
// the name as a ClientProto.
func NewClientProto(s string) (cp ClientProto, err error) {
	switch cp = ClientProto(s); cp {
	case
		ClientProtoDoH,
		ClientProtoDoQ,
		ClientProtoDoT,
		ClientProtoDNSCrypt,
		ClientProtoPlain:

		return cp, nil
	default:
		return "", fmt.Errorf("invalid client proto: %q", s)
	}
}

// Client is the information required by the query log to match against clients
// during searches.
type Client struct {
	WHOIS          *ClientWHOIS `json:"whois,omitempty"`
	Name           string       `json:"name"`
	DisallowedRule string       `json:"disallowed_rule"`
	Disallowed     bool         `json:"disallowed"`
}

// ClientWHOIS is the filtered WHOIS data for the client.
//
// TODO(a.garipov): Merge with home.RuntimeClientWHOISInfo after the
// refactoring is done.
type ClientWHOIS struct {
	City    string `json:"city,omitempty"`
	Country string `json:"country,omitempty"`
	Orgname string `json:"orgname,omitempty"`
}
