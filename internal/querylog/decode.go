package querylog

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/dnsfilter"
	"github.com/AdguardTeam/golibs/log"
)

type logEntryHandler (func(t json.Token, ent *logEntry) error)

var logEntryHandlers = map[string]logEntryHandler{
	"IP": func(t json.Token, ent *logEntry) error {
		v, ok := t.(string)
		if !ok {
			return nil
		}
		if len(ent.IP) == 0 {
			ent.IP = v
		}
		return nil
	},
	"T": func(t json.Token, ent *logEntry) error {
		v, ok := t.(string)
		if !ok {
			return nil
		}
		var err error
		ent.Time, err = time.Parse(time.RFC3339, v)
		return err
	},
	"QH": func(t json.Token, ent *logEntry) error {
		v, ok := t.(string)
		if !ok {
			return nil
		}
		ent.QHost = v
		return nil
	},
	"QT": func(t json.Token, ent *logEntry) error {
		v, ok := t.(string)
		if !ok {
			return nil
		}
		ent.QType = v
		return nil
	},
	"QC": func(t json.Token, ent *logEntry) error {
		v, ok := t.(string)
		if !ok {
			return nil
		}
		ent.QClass = v
		return nil
	},
	"CP": func(t json.Token, ent *logEntry) error {
		v, ok := t.(string)
		if !ok {
			return nil
		}
		var err error
		ent.ClientProto, err = NewClientProto(v)
		return err
	},
	"Answer": func(t json.Token, ent *logEntry) error {
		v, ok := t.(string)
		if !ok {
			return nil
		}
		var err error
		ent.Answer, err = base64.StdEncoding.DecodeString(v)
		return err
	},
	"OrigAnswer": func(t json.Token, ent *logEntry) error {
		v, ok := t.(string)
		if !ok {
			return nil
		}
		var err error
		ent.OrigAnswer, err = base64.StdEncoding.DecodeString(v)
		return err
	},
	"Upstream": func(t json.Token, ent *logEntry) error {
		v, ok := t.(string)
		if !ok {
			return nil
		}
		ent.Upstream = v
		return nil
	},
	"Elapsed": func(t json.Token, ent *logEntry) error {
		v, ok := t.(json.Number)
		if !ok {
			return nil
		}
		i, err := v.Int64()
		if err != nil {
			return err
		}
		ent.Elapsed = time.Duration(i)
		return nil
	},
}

var resultHandlers = map[string]logEntryHandler{
	"IsFiltered": func(t json.Token, ent *logEntry) error {
		v, ok := t.(bool)
		if !ok {
			return nil
		}
		ent.Result.IsFiltered = v
		return nil
	},
	"Rule": func(t json.Token, ent *logEntry) error {
		s, ok := t.(string)
		if !ok {
			return nil
		}

		l := len(ent.Result.Rules)
		if l == 0 {
			ent.Result.Rules = []*dnsfilter.ResultRule{{}}
			l++
		}

		ent.Result.Rules[l-1].Text = s

		return nil
	},
	"FilterID": func(t json.Token, ent *logEntry) error {
		n, ok := t.(json.Number)
		if !ok {
			return nil
		}

		i, err := n.Int64()
		if err != nil {
			return err
		}

		l := len(ent.Result.Rules)
		if l == 0 {
			ent.Result.Rules = []*dnsfilter.ResultRule{{}}
			l++
		}

		ent.Result.Rules[l-1].FilterListID = i

		return nil
	},
	"Reason": func(t json.Token, ent *logEntry) error {
		v, ok := t.(json.Number)
		if !ok {
			return nil
		}
		i, err := v.Int64()
		if err != nil {
			return err
		}
		ent.Result.Reason = dnsfilter.Reason(i)
		return nil
	},
	"ServiceName": func(t json.Token, ent *logEntry) error {
		v, ok := t.(string)
		if !ok {
			return nil
		}
		ent.Result.ServiceName = v
		return nil
	},
}

func decodeResult(dec *json.Decoder, ent *logEntry) {
	for {
		keyToken, err := dec.Token()
		if err != nil {
			if err != io.EOF {
				log.Debug("decodeResult err: %s", err)
			}

			return
		}

		if d, ok := keyToken.(json.Delim); ok {
			if d == '}' {
				return
			}

			continue
		}

		key, ok := keyToken.(string)
		if !ok {
			log.Debug("decodeResult: keyToken is %T (%[1]v) and not string", keyToken)

			return
		}

		handler, ok := resultHandlers[key]
		if !ok {
			continue
		}

		val, err := dec.Token()
		if err != nil {
			return
		}

		if err = handler(val, ent); err != nil {
			log.Debug("decodeResult handler err: %s", err)

			return
		}
	}
}

func decodeLogEntry(ent *logEntry, str string) {
	dec := json.NewDecoder(strings.NewReader(str))
	dec.UseNumber()
	for {
		keyToken, err := dec.Token()
		if err != nil {
			if err != io.EOF {
				log.Debug("decodeLogEntry err: %s", err)
			}

			return
		}

		if _, ok := keyToken.(json.Delim); ok {
			continue
		}

		key, ok := keyToken.(string)
		if !ok {
			log.Debug("decodeLogEntry: keyToken is %T (%[1]v) and not string", keyToken)

			return
		}

		if key == "Result" {
			decodeResult(dec, ent)

			continue
		}

		handler, ok := logEntryHandlers[key]
		if !ok {
			continue
		}

		val, err := dec.Token()
		if err != nil {
			return
		}

		if err = handler(val, ent); err != nil {
			log.Debug("decodeLogEntry handler err: %s", err)

			return
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
