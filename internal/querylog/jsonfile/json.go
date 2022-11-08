package jsonfile

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/AdGuardHome/internal/querylog/logs"
	"github.com/AdguardTeam/golibs/log"
	"github.com/miekg/dns"
	"golang.org/x/exp/slices"
	"golang.org/x/net/idna"
)

// entriesToJSON converts query log entries to LogsPayload.
func (l *queryLog) entriesToJSON(entries []*logEntry, oldest time.Time) *logs.LogsPayload {
	o := &logs.LogsPayload{}
	o.Data = make([]*logs.LogData, 0, len(entries))
	// The elements order is already reversed to be from newer to older.
	for _, entry := range entries {
		jsonEntry := l.entryToJSON(entry, l.anonymizer.Load())
		o.Data = append(o.Data, jsonEntry)
	}
	if !oldest.IsZero() {
		o.Oldest = oldest.Format(time.RFC3339Nano)
	}
	return o
}

// entryToJSON converts a log entry's data into an entry for the JSON API.
func (l *queryLog) entryToJSON(entry *logEntry, anonFunc aghnet.IPMutFunc) (o *logs.LogData) {
	hostname := entry.QHost
	question := logs.Question{
		Type:  entry.QType,
		Class: entry.QClass,
		Name:  hostname,
	}
	if qhost, err := idna.ToUnicode(hostname); err != nil {
		log.Debug("querylog: translating %q into unicode: %s", hostname, err)
	} else if qhost != hostname && qhost != "" {
		question.UnicodeName = qhost
	}

	entIP := slices.Clone(entry.IP)
	anonFunc(entIP)

	o = &logs.LogData{
		Reason:      entry.Result.Reason.String(),
		ElapsedMs:   strconv.FormatFloat(entry.Elapsed.Seconds()*1000, 'f', -1, 64),
		Time:        entry.Time.Format(time.RFC3339Nano),
		Client:      entIP,
		ClientId:    entry.ClientID,
		ClientProto: entry.ClientProto,
		Cached:      entry.Cached,
		ReqECS:      entry.ReqECS,
		Upstream:    entry.Upstream,
		Question:    question,
		Rules:       resultRulesToJSONRules(entry.Result.Rules),
	}

	if entIP.Equal(entry.IP) {
		o.ClientInfo = entry.client
	}

	if len(entry.Result.Rules) > 0 {
		if r := entry.Result.Rules[0]; len(r.Text) > 0 {
			o.Rule = r.Text
			o.FilterId = r.FilterListID
		}
	}

	if len(entry.Result.ServiceName) != 0 {
		o.ServiceName = entry.Result.ServiceName
	}

	if len(entry.Answer) > 0 {
		msg := &dns.Msg{}
		if err := msg.Unpack(entry.Answer); err != nil {
			log.Debug("querylog: failed to unpack dns msg answer: %v: %s", entry.Answer, err)

			return
		}

		o.Status = dns.RcodeToString[msg.Rcode]
		// Old query logs may still keep AD flag value in the message.  Try to get
		// it from there as well.
		o.AnswerDnssec = entry.AuthenticatedData || msg.AuthenticatedData
		if a := formatAnswers(msg); a != nil {
			o.Answer = a
		}
	}

	if len(entry.OrigAnswer) > 0 {
		orig := &dns.Msg{}
		err := orig.Unpack(entry.OrigAnswer)
		if err != nil {
			log.Debug("querylog: orig.Unpack(entry.OrigAnswer): %v: %s", entry.OrigAnswer, err)

			return
		}
		if a := formatAnswers(orig); a != nil {
			o.OriginalAnswer = a
		}
	}

	return
}

func resultRulesToJSONRules(rules []*filtering.ResultRule) (jsonRules []logs.RuleEntry) {
	jsonRules = make([]logs.RuleEntry, len(rules))
	for i, r := range rules {
		jsonRules[i] = logs.RuleEntry{
			FilterListId: r.FilterListID,
			Text:         r.Text,
		}
	}
	return jsonRules
}

func formatAnswers(a *dns.Msg) (answers []logs.Answer) {
	if a == nil || len(a.Answer) == 0 {
		return nil
	}
	answers = make([]logs.Answer, 0, len(a.Answer))
	for _, k := range a.Answer {
		header := k.Header()
		answer := logs.Answer{
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
