package querylog

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/golibs/log"
	"github.com/miekg/dns"
	"golang.org/x/exp/slices"
	"golang.org/x/net/idna"
)

// TODO(a.garipov): Use a proper structured approach here.

// jobject is a JSON object alias.
type jobject = map[string]any

// entriesToJSON converts query log entries to JSON.
func (l *queryLog) entriesToJSON(entries []*logEntry, oldest time.Time) (res jobject) {
	data := make([]jobject, 0, len(entries))

	// The elements order is already reversed to be from newer to older.
	for _, entry := range entries {
		jsonEntry := l.entryToJSON(entry, l.anonymizer.Load())
		data = append(data, jsonEntry)
	}

	res = jobject{
		"data":   data,
		"oldest": "",
	}
	if !oldest.IsZero() {
		res["oldest"] = oldest.Format(time.RFC3339Nano)
	}

	return res
}

// entryToJSON converts a log entry's data into an entry for the JSON API.
func (l *queryLog) entryToJSON(entry *logEntry, anonFunc aghnet.IPMutFunc) (jsonEntry jobject) {
	hostname := entry.QHost
	question := jobject{
		"type":  entry.QType,
		"class": entry.QClass,
		"name":  hostname,
	}

	if qhost, err := idna.ToUnicode(hostname); err != nil {
		log.Debug("querylog: translating %q into unicode: %s", hostname, err)
	} else if qhost != hostname && qhost != "" {
		question["unicode_name"] = qhost
	}

	entIP := slices.Clone(entry.IP)
	anonFunc(entIP)

	jsonEntry = jobject{
		"reason":       entry.Result.Reason.String(),
		"elapsedMs":    strconv.FormatFloat(entry.Elapsed.Seconds()*1000, 'f', -1, 64),
		"time":         entry.Time.Format(time.RFC3339Nano),
		"client":       entIP,
		"client_proto": entry.ClientProto,
		"cached":       entry.Cached,
		"upstream":     entry.Upstream,
		"question":     question,
		"rules":        resultRulesToJSONRules(entry.Result.Rules),
	}

	if entIP.Equal(entry.IP) {
		jsonEntry["client_info"] = entry.client
	}

	if entry.ClientID != "" {
		jsonEntry["client_id"] = entry.ClientID
	}

	if entry.ReqECS != "" {
		jsonEntry["ecs"] = entry.ReqECS
	}

	if len(entry.Result.Rules) > 0 {
		if r := entry.Result.Rules[0]; len(r.Text) > 0 {
			jsonEntry["rule"] = r.Text
			jsonEntry["filterId"] = r.FilterListID
		}
	}

	if len(entry.Result.ServiceName) != 0 {
		jsonEntry["service_name"] = entry.Result.ServiceName
	}

	l.setMsgData(entry, jsonEntry)
	l.setOrigAns(entry, jsonEntry)

	return jsonEntry
}

// setMsgData sets the message data in jsonEntry.
func (l *queryLog) setMsgData(entry *logEntry, jsonEntry jobject) {
	if len(entry.Answer) == 0 {
		return
	}

	msg := &dns.Msg{}
	if err := msg.Unpack(entry.Answer); err != nil {
		log.Debug("querylog: failed to unpack dns msg answer: %v: %s", entry.Answer, err)

		return
	}

	jsonEntry["status"] = dns.RcodeToString[msg.Rcode]
	// Old query logs may still keep AD flag value in the message.  Try to get
	// it from there as well.
	jsonEntry["answer_dnssec"] = entry.AuthenticatedData || msg.AuthenticatedData

	if a := answerToMap(msg); a != nil {
		jsonEntry["answer"] = a
	}
}

// setOrigAns sets the original answer data in jsonEntry.
func (l *queryLog) setOrigAns(entry *logEntry, jsonEntry jobject) {
	if len(entry.OrigAnswer) == 0 {
		return
	}

	orig := &dns.Msg{}
	err := orig.Unpack(entry.OrigAnswer)
	if err != nil {
		log.Debug("querylog: orig.Unpack(entry.OrigAnswer): %v: %s", entry.OrigAnswer, err)

		return
	}

	if a := answerToMap(orig); a != nil {
		jsonEntry["original_answer"] = a
	}
}

func resultRulesToJSONRules(rules []*filtering.ResultRule) (jsonRules []jobject) {
	jsonRules = make([]jobject, len(rules))
	for i, r := range rules {
		jsonRules[i] = jobject{
			"filter_list_id": r.FilterListID,
			"text":           r.Text,
		}
	}

	return jsonRules
}

type dnsAnswer struct {
	Type  string `json:"type"`
	Value string `json:"value"`
	TTL   uint32 `json:"ttl"`
}

func answerToMap(a *dns.Msg) (answers []*dnsAnswer) {
	if a == nil || len(a.Answer) == 0 {
		return nil
	}

	answers = make([]*dnsAnswer, 0, len(a.Answer))
	for _, k := range a.Answer {
		header := k.Header()
		answer := &dnsAnswer{
			Type: dns.TypeToString[header.Rrtype],
			TTL:  header.Ttl,
		}

		// Some special treatment for some well-known types.
		//
		// TODO(a.garipov): Consider just calling String() for everyone
		// instead.
		switch v := k.(type) {
		case nil:
			// Probably unlikely, but go on.
		case *dns.A:
			answer.Value = v.A.String()
		case *dns.AAAA:
			answer.Value = v.AAAA.String()
		case *dns.MX:
			answer.Value = fmt.Sprintf("%v %v", v.Preference, v.Mx)
		case *dns.CNAME:
			answer.Value = v.Target
		case *dns.NS:
			answer.Value = v.Ns
		case *dns.SPF:
			answer.Value = strings.Join(v.Txt, "\n")
		case *dns.TXT:
			answer.Value = strings.Join(v.Txt, "\n")
		case *dns.PTR:
			answer.Value = v.Ptr
		case *dns.SOA:
			answer.Value = fmt.Sprintf("%v %v %v %v %v %v %v", v.Ns, v.Mbox, v.Serial, v.Refresh, v.Retry, v.Expire, v.Minttl)
		case *dns.CAA:
			answer.Value = fmt.Sprintf("%v %v \"%v\"", v.Flag, v.Tag, v.Value)
		case *dns.HINFO:
			answer.Value = fmt.Sprintf("\"%v\" \"%v\"", v.Cpu, v.Os)
		case *dns.RRSIG:
			answer.Value = fmt.Sprintf("%v %v %v %v %v %v %v %v %v", dns.TypeToString[v.TypeCovered], v.Algorithm, v.Labels, v.OrigTtl, v.Expiration, v.Inception, v.KeyTag, v.SignerName, v.Signature)
		default:
			answer.Value = v.String()
		}

		answers = append(answers, answer)
	}

	return answers
}
