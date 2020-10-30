package querylog

import (
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/AdguardTeam/golibs/log"
	"github.com/miekg/dns"
)

// Get Client IP address
func (l *queryLog) getClientIP(clientIP string) string {
	if l.conf.AnonymizeClientIP {
		ip := net.ParseIP(clientIP)
		if ip != nil {
			ip4 := ip.To4()
			const AnonymizeClientIP4Mask = 16
			const AnonymizeClientIP6Mask = 112
			if ip4 != nil {
				clientIP = ip4.Mask(net.CIDRMask(AnonymizeClientIP4Mask, 32)).String()
			} else {
				clientIP = ip.Mask(net.CIDRMask(AnonymizeClientIP6Mask, 128)).String()
			}
		}
	}

	return clientIP
}

// entriesToJSON - converts log entries to JSON
func (l *queryLog) entriesToJSON(entries []*logEntry, oldest time.Time) map[string]interface{} {
	// init the response object
	var data = []map[string]interface{}{}

	// the elements order is already reversed (from newer to older)
	for i := 0; i < len(entries); i++ {
		entry := entries[i]
		jsonEntry := l.logEntryToJSONEntry(entry)
		data = append(data, jsonEntry)
	}

	var result = map[string]interface{}{}
	result["oldest"] = ""
	if !oldest.IsZero() {
		result["oldest"] = oldest.Format(time.RFC3339Nano)
	}
	result["data"] = data

	return result
}

func (l *queryLog) logEntryToJSONEntry(entry *logEntry) map[string]interface{} {
	var msg *dns.Msg

	if len(entry.Answer) > 0 {
		msg = new(dns.Msg)
		if err := msg.Unpack(entry.Answer); err != nil {
			log.Debug("Failed to unpack dns message answer: %s: %s", err, string(entry.Answer))
			msg = nil
		}
	}

	jsonEntry := map[string]interface{}{
		"reason":       entry.Result.Reason.String(),
		"elapsedMs":    strconv.FormatFloat(entry.Elapsed.Seconds()*1000, 'f', -1, 64),
		"time":         entry.Time.Format(time.RFC3339Nano),
		"client":       l.getClientIP(entry.IP),
		"client_proto": entry.ClientProto,
	}
	jsonEntry["question"] = map[string]interface{}{
		"host":  entry.QHost,
		"type":  entry.QType,
		"class": entry.QClass,
	}

	if msg != nil {
		jsonEntry["status"] = dns.RcodeToString[msg.Rcode]

		opt := msg.IsEdns0()
		dnssecOk := false
		if opt != nil {
			dnssecOk = opt.Do()
		}
		jsonEntry["answer_dnssec"] = dnssecOk
	}

	if len(entry.Result.Rule) > 0 {
		jsonEntry["rule"] = entry.Result.Rule
		jsonEntry["filterId"] = entry.Result.FilterID
	}

	if len(entry.Result.ServiceName) != 0 {
		jsonEntry["service_name"] = entry.Result.ServiceName
	}

	answers := answerToMap(msg)
	if answers != nil {
		jsonEntry["answer"] = answers
	}

	if len(entry.OrigAnswer) != 0 {
		a := new(dns.Msg)
		err := a.Unpack(entry.OrigAnswer)
		if err == nil {
			answers = answerToMap(a)
			if answers != nil {
				jsonEntry["original_answer"] = answers
			}
		} else {
			log.Debug("Querylog: msg.Unpack(entry.OrigAnswer): %s: %s", err, string(entry.OrigAnswer))
		}
	}

	jsonEntry["upstream"] = entry.Upstream

	return jsonEntry
}

func answerToMap(a *dns.Msg) []map[string]interface{} {
	if a == nil || len(a.Answer) == 0 {
		return nil
	}

	var answers = []map[string]interface{}{}
	for _, k := range a.Answer {
		header := k.Header()
		answer := map[string]interface{}{
			"type": dns.TypeToString[header.Rrtype],
			"ttl":  header.Ttl,
		}
		// try most common record types
		switch v := k.(type) {
		case *dns.A:
			answer["value"] = v.A.String()
		case *dns.AAAA:
			answer["value"] = v.AAAA.String()
		case *dns.MX:
			answer["value"] = fmt.Sprintf("%v %v", v.Preference, v.Mx)
		case *dns.CNAME:
			answer["value"] = v.Target
		case *dns.NS:
			answer["value"] = v.Ns
		case *dns.SPF:
			answer["value"] = v.Txt
		case *dns.TXT:
			answer["value"] = v.Txt
		case *dns.PTR:
			answer["value"] = v.Ptr
		case *dns.SOA:
			answer["value"] = fmt.Sprintf("%v %v %v %v %v %v %v", v.Ns, v.Mbox, v.Serial, v.Refresh, v.Retry, v.Expire, v.Minttl)
		case *dns.CAA:
			answer["value"] = fmt.Sprintf("%v %v \"%v\"", v.Flag, v.Tag, v.Value)
		case *dns.HINFO:
			answer["value"] = fmt.Sprintf("\"%v\" \"%v\"", v.Cpu, v.Os)
		case *dns.RRSIG:
			answer["value"] = fmt.Sprintf("%v %v %v %v %v %v %v %v %v", dns.TypeToString[v.TypeCovered], v.Algorithm, v.Labels, v.OrigTtl, v.Expiration, v.Inception, v.KeyTag, v.SignerName, v.Signature)
		default:
			// type unknown, marshall it as-is
			answer["value"] = v
		}
		answers = append(answers, answer)
	}

	return answers
}
