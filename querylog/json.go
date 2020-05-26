package querylog

import (
	"encoding/base64"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/dnsfilter"
	"github.com/AdguardTeam/golibs/log"
	"github.com/miekg/dns"
)

// decodeLogEntry - decodes query log entry from a line
// nolint (gocyclo)
func decodeLogEntry(ent *logEntry, str string) {
	var b bool
	var i int
	var err error
	for {
		k, v, t := readJSON(&str)
		if t == jsonTErr {
			break
		}
		switch k {
		case "IP":
			if len(ent.IP) == 0 {
				ent.IP = v
			}
		case "T":
			ent.Time, err = time.Parse(time.RFC3339, v)

		case "QH":
			ent.QHost = v
		case "QT":
			ent.QType = v
		case "QC":
			ent.QClass = v

		case "Answer":
			ent.Answer, err = base64.StdEncoding.DecodeString(v)
		case "OrigAnswer":
			ent.OrigAnswer, err = base64.StdEncoding.DecodeString(v)

		case "IsFiltered":
			b, err = strconv.ParseBool(v)
			ent.Result.IsFiltered = b
		case "Rule":
			ent.Result.Rule = v
		case "FilterID":
			i, err = strconv.Atoi(v)
			ent.Result.FilterID = int64(i)
		case "Reason":
			i, err = strconv.Atoi(v)
			ent.Result.Reason = dnsfilter.Reason(i)

		case "Upstream":
			ent.Upstream = v
		case "Elapsed":
			i, err = strconv.Atoi(v)
			ent.Elapsed = time.Duration(i)

		// pre-v0.99.3 compatibility:
		case "Question":
			var qstr []byte
			qstr, err = base64.StdEncoding.DecodeString(v)
			if err != nil {
				break
			}
			q := new(dns.Msg)
			err = q.Unpack(qstr)
			if err != nil {
				break
			}
			ent.QHost = q.Question[0].Name
			if len(ent.QHost) == 0 {
				break
			}
			ent.QHost = ent.QHost[:len(ent.QHost)-1]
			ent.QType = dns.TypeToString[q.Question[0].Qtype]
			ent.QClass = dns.ClassToString[q.Question[0].Qclass]
		case "Time":
			ent.Time, err = time.Parse(time.RFC3339, v)
		}

		if err != nil {
			log.Debug("decodeLogEntry err: %s", err)
			break
		}
	}
}

// Get value from "key":"value"
func readJSONValue(s, name string) string {
	i := strings.Index(s, "\""+name+"\":\"")
	if i == -1 {
		return ""
	}
	start := i + 1 + len(name) + 3
	i = strings.IndexByte(s[start:], '"')
	if i == -1 {
		return ""
	}
	end := start + i
	return s[start:end]
}

const (
	jsonTErr = iota
	jsonTObj
	jsonTStr
	jsonTNum
	jsonTBool
)

// Parse JSON key-value pair
//  e.g.: "key":VALUE where VALUE is "string", true|false (boolean), or 123.456 (number)
// Note the limitations:
//  . doesn't support whitespace
//  . doesn't support "null"
//  . doesn't validate boolean or number
//  . no proper handling of {} braces
//  . no handling of [] brackets
// Return (key, value, type)
func readJSON(ps *string) (string, string, int32) {
	s := *ps
	k := ""
	v := ""
	t := int32(jsonTErr)

	q1 := strings.IndexByte(s, '"')
	if q1 == -1 {
		return k, v, t
	}
	q2 := strings.IndexByte(s[q1+1:], '"')
	if q2 == -1 {
		return k, v, t
	}
	k = s[q1+1 : q1+1+q2]
	s = s[q1+1+q2+1:]

	if len(s) < 2 || s[0] != ':' {
		return k, v, t
	}

	if s[1] == '"' {
		q2 = strings.IndexByte(s[2:], '"')
		if q2 == -1 {
			return k, v, t
		}
		v = s[2 : 2+q2]
		t = jsonTStr
		s = s[2+q2+1:]

	} else if s[1] == '{' {
		t = jsonTObj
		s = s[1+1:]

	} else {
		sep := strings.IndexAny(s[1:], ",}")
		if sep == -1 {
			return k, v, t
		}
		v = s[1 : 1+sep]
		if s[1] == 't' || s[1] == 'f' {
			t = jsonTBool
		} else if s[1] == '.' || (s[1] >= '0' && s[1] <= '9') {
			t = jsonTNum
		}
		s = s[1+sep+1:]
	}

	*ps = s
	return k, v, t
}

// Get Client IP address
func (l *queryLog) getClientIP(clientIP string) string {
	if l.conf.AnonymizeClientIP {
		ip := net.ParseIP(clientIP)
		if ip != nil {
			ip4 := ip.To4()
			const AnonymizeClientIP4Mask = 24
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
		"reason":    entry.Result.Reason.String(),
		"elapsedMs": strconv.FormatFloat(entry.Elapsed.Seconds()*1000, 'f', -1, 64),
		"time":      entry.Time.Format(time.RFC3339Nano),
		"client":    l.getClientIP(entry.IP),
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
