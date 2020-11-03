package querylog

import (
	"encoding/base64"
	"strconv"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/dnsfilter"
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

		case "CP":
			ent.ClientProto, err = NewClientProto(v)

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
		case "ServiceName":
			ent.Result.ServiceName = v

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
