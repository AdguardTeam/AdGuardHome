package querylog

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/netip"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering/rulelist"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/urlfilter/rules"
	"github.com/miekg/dns"
)

// logEntryHandler represents a handler for decoding json token to the logEntry
// struct.
type logEntryHandler func(t json.Token, ent *logEntry) error

// logEntryHandlers is the map of log entry decode handlers for various keys.
var logEntryHandlers = map[string]logEntryHandler{
	"CID": func(t json.Token, ent *logEntry) error {
		v, ok := t.(string)
		if !ok {
			return nil
		}

		ent.ClientID = v

		return nil
	},
	"IP": func(t json.Token, ent *logEntry) error {
		v, ok := t.(string)
		if !ok {
			return nil
		}

		if ent.IP == nil {
			ent.IP = net.ParseIP(v)
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
	"ECS": func(t json.Token, ent *logEntry) error {
		v, ok := t.(string)
		if !ok {
			return nil
		}

		ent.ReqECS = v

		return nil
	},
	"Cached": func(t json.Token, ent *logEntry) error {
		v, ok := t.(bool)
		if !ok {
			return nil
		}

		ent.Cached = v

		return nil
	},
	"AD": func(t json.Token, ent *logEntry) error {
		v, ok := t.(bool)
		if !ok {
			return nil
		}

		ent.AuthenticatedData = v

		return nil
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

// decodeResultRuleKey decodes the token of "Rules" type to logEntry struct.
func decodeResultRuleKey(key string, i int, dec *json.Decoder, ent *logEntry) {
	var vToken json.Token
	switch key {
	case "FilterListID":
		ent.Result.Rules, vToken = decodeVTokenAndAddRule(key, i, dec, ent.Result.Rules)
		if n, ok := vToken.(json.Number); ok {
			id, _ := n.Int64()
			ent.Result.Rules[i].FilterListID = rulelist.URLFilterID(id)
		}
	case "IP":
		ent.Result.Rules, vToken = decodeVTokenAndAddRule(key, i, dec, ent.Result.Rules)
		if ipStr, ok := vToken.(string); ok {
			if ip, err := netip.ParseAddr(ipStr); err == nil {
				ent.Result.Rules[i].IP = ip
			} else {
				log.Debug("querylog: decoding ipStr value: %s", err)
			}
		}
	case "Text":
		ent.Result.Rules, vToken = decodeVTokenAndAddRule(key, i, dec, ent.Result.Rules)
		if s, ok := vToken.(string); ok {
			ent.Result.Rules[i].Text = s
		}
	default:
		// Go on.
	}
}

// decodeVTokenAndAddRule decodes the "Rules" toke as [filtering.ResultRule]
// and then adds the decoded object to the slice of result rules.
func decodeVTokenAndAddRule(
	key string,
	i int,
	dec *json.Decoder,
	rules []*filtering.ResultRule,
) (newRules []*filtering.ResultRule, vToken json.Token) {
	newRules = rules

	vToken, err := dec.Token()
	if err != nil {
		if err != io.EOF {
			log.Debug("decodeResultRuleKey %s err: %s", key, err)
		}

		return newRules, nil
	}

	if len(rules) < i+1 {
		newRules = append(newRules, &filtering.ResultRule{})
	}

	return newRules, vToken
}

// decodeResultRules parses the dec's tokens into logEntry ent interpreting it
// as a slice of the result rules.
func decodeResultRules(dec *json.Decoder, ent *logEntry) {
	for {
		delimToken, err := dec.Token()
		if err != nil {
			if err != io.EOF {
				log.Debug("decodeResultRules err: %s", err)
			}

			return
		}

		if d, ok := delimToken.(json.Delim); !ok {
			return
		} else if d != '[' {
			log.Debug("decodeResultRules: unexpected delim %q", d)
		}

		err = decodeResultRuleToken(dec, ent)
		if err != nil {
			if err != io.EOF && !errors.Is(err, ErrEndOfToken) {
				log.Debug("decodeResultRules err: %s", err)
			}

			return
		}
	}
}

// decodeResultRuleToken decodes the tokens of "Rules" type to the logEntry ent.
func decodeResultRuleToken(dec *json.Decoder, ent *logEntry) (err error) {
	i := 0
	for {
		var keyToken json.Token
		keyToken, err = dec.Token()
		if err != nil {
			// Don't wrap the error, because it's informative enough as is.
			return err
		}

		if d, ok := keyToken.(json.Delim); ok {
			switch d {
			case '}':
				i++
			case ']':
				return ErrEndOfToken
			default:
				// Go on.
			}

			continue
		}

		key, ok := keyToken.(string)
		if !ok {
			return fmt.Errorf("keyToken is %T (%[1]v) and not string", keyToken)
		}

		decodeResultRuleKey(key, i, dec, ent)
	}
}

// decodeResultReverseHosts parses the dec's tokens into ent interpreting it as
// the result of hosts container's $dnsrewrite rule.  It assumes there are no
// other occurrences of DNSRewriteResult in the entry since hosts container's
// rewrites currently has the highest priority along the entire filtering
// pipeline.
func decodeResultReverseHosts(dec *json.Decoder, ent *logEntry) {
	for {
		itemToken, err := dec.Token()
		if err != nil {
			if err != io.EOF {
				log.Debug("decodeResultReverseHosts err: %s", err)
			}

			return
		}

		switch v := itemToken.(type) {
		case json.Delim:
			if v == '[' {
				continue
			} else if v == ']' {
				return
			}

			log.Debug("decodeResultReverseHosts: unexpected delim %q", v)

			return
		case string:
			v = dns.Fqdn(v)
			if res := &ent.Result; res.DNSRewriteResult == nil {
				res.DNSRewriteResult = &filtering.DNSRewriteResult{
					RCode: dns.RcodeSuccess,
					Response: filtering.DNSRewriteResultResponse{
						dns.TypePTR: []rules.RRValue{v},
					},
				}

				continue
			} else {
				res.DNSRewriteResult.RCode = dns.RcodeSuccess
			}

			if rres := ent.Result.DNSRewriteResult; rres.Response == nil {
				rres.Response = filtering.DNSRewriteResultResponse{dns.TypePTR: []rules.RRValue{v}}
			} else {
				rres.Response[dns.TypePTR] = append(rres.Response[dns.TypePTR], v)
			}
		default:
			continue
		}
	}
}

// decodeResultIPList parses the dec's tokens into logEntry ent interpreting it
// as the result IP addresses list.
func decodeResultIPList(dec *json.Decoder, ent *logEntry) {
	for {
		itemToken, err := dec.Token()
		if err != nil {
			if err != io.EOF {
				log.Debug("decodeResultIPList err: %s", err)
			}

			return
		}

		switch v := itemToken.(type) {
		case json.Delim:
			if v == '[' {
				continue
			} else if v == ']' {
				return
			}

			log.Debug("decodeResultIPList: unexpected delim %q", v)

			return
		case string:
			var ip netip.Addr
			ip, err = netip.ParseAddr(v)
			if err == nil {
				ent.Result.IPList = append(ent.Result.IPList, ip)
			}
		default:
			continue
		}
	}
}

// decodeResultDNSRewriteResultKey decodes the token of "DNSRewriteResult" type
// to the logEntry struct.
func decodeResultDNSRewriteResultKey(key string, dec *json.Decoder, ent *logEntry) {
	var err error

	switch key {
	case "RCode":
		var vToken json.Token
		vToken, err = dec.Token()
		if err != nil {
			if err != io.EOF {
				log.Debug("decodeResultDNSRewriteResultKey err: %s", err)
			}

			return
		}

		if ent.Result.DNSRewriteResult == nil {
			ent.Result.DNSRewriteResult = &filtering.DNSRewriteResult{}
		}

		if n, ok := vToken.(json.Number); ok {
			rcode64, _ := n.Int64()
			ent.Result.DNSRewriteResult.RCode = rules.RCode(rcode64)
		}
	case "Response":
		if ent.Result.DNSRewriteResult == nil {
			ent.Result.DNSRewriteResult = &filtering.DNSRewriteResult{}
		}

		if ent.Result.DNSRewriteResult.Response == nil {
			ent.Result.DNSRewriteResult.Response = filtering.DNSRewriteResultResponse{}
		}

		// TODO(a.garipov): I give up.  This whole file is a mess.  Luckily, we
		// can assume that this field is relatively rare and just use the normal
		// decoding and correct the values.
		err = dec.Decode(&ent.Result.DNSRewriteResult.Response)
		if err != nil {
			log.Debug("decodeResultDNSRewriteResultKey response err: %s", err)
		}

		ent.parseDNSRewriteResultIPs()
	default:
		// Go on.
	}
}

// decodeResultDNSRewriteResult parses the dec's tokens into logEntry ent
// interpreting it as the result DNSRewriteResult.
func decodeResultDNSRewriteResult(dec *json.Decoder, ent *logEntry) {
	for {
		key, err := parseKeyToken(dec)
		if err != nil {
			if err != io.EOF && !errors.Is(err, ErrEndOfToken) {
				log.Debug("decodeResultDNSRewriteResult: %s", err)
			}

			return
		}

		if key == "" {
			continue
		}

		decodeResultDNSRewriteResultKey(key, dec, ent)
	}
}

// translateResult converts some fields of the ent.Result to the format
// consistent with current implementation.
func translateResult(ent *logEntry) {
	res := &ent.Result
	if res.Reason != filtering.RewrittenAutoHosts || len(res.IPList) == 0 {
		return
	}

	if res.DNSRewriteResult == nil {
		res.DNSRewriteResult = &filtering.DNSRewriteResult{
			RCode: dns.RcodeSuccess,
		}
	}

	if res.DNSRewriteResult.Response == nil {
		res.DNSRewriteResult.Response = filtering.DNSRewriteResultResponse{}
	}

	resp := res.DNSRewriteResult.Response
	for _, ip := range res.IPList {
		qType := dns.TypeAAAA
		if ip.Is4() {
			qType = dns.TypeA
		}

		resp[qType] = append(resp[qType], ip)
	}

	res.IPList = nil
}

// ErrEndOfToken is an error returned by parse key token when the closing
// bracket is found.
const ErrEndOfToken errors.Error = "end of token"

// parseKeyToken parses the dec's token key.
func parseKeyToken(dec *json.Decoder) (key string, err error) {
	keyToken, err := dec.Token()
	if err != nil {
		return "", err
	}

	if d, ok := keyToken.(json.Delim); ok {
		if d == '}' {
			return "", ErrEndOfToken
		}

		return "", nil
	}

	key, ok := keyToken.(string)
	if !ok {
		return "", fmt.Errorf("keyToken is %T (%[1]v) and not string", keyToken)
	}

	return key, nil
}

// decodeResult decodes a token of "Result" type to logEntry struct.
func decodeResult(dec *json.Decoder, ent *logEntry) {
	defer translateResult(ent)

	for {
		key, err := parseKeyToken(dec)
		if err != nil {
			if err != io.EOF && !errors.Is(err, ErrEndOfToken) {
				log.Debug("decodeResult: %s", err)
			}

			return
		}

		if key == "" {
			continue
		}

		decHandler, ok := resultDecHandlers[key]
		if ok {
			decHandler(dec, ent)

			continue
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

// resultHandlers is the map of log entry decode handlers for various keys.
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
			ent.Result.Rules = []*filtering.ResultRule{{}}
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

		id, err := n.Int64()
		if err != nil {
			return err
		}

		l := len(ent.Result.Rules)
		if l == 0 {
			ent.Result.Rules = []*filtering.ResultRule{{}}
			l++
		}

		ent.Result.Rules[l-1].FilterListID = rulelist.URLFilterID(id)

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

		ent.Result.Reason = filtering.Reason(i)

		return nil
	},
	"ServiceName": func(t json.Token, ent *logEntry) error {
		s, ok := t.(string)
		if !ok {
			return nil
		}

		ent.Result.ServiceName = s

		return nil
	},
	"CanonName": func(t json.Token, ent *logEntry) error {
		s, ok := t.(string)
		if !ok {
			return nil
		}

		ent.Result.CanonName = s

		return nil
	},
}

// resultDecHandlers is the map of decode handlers for various keys.
var resultDecHandlers = map[string]func(dec *json.Decoder, ent *logEntry){
	"ReverseHosts":     decodeResultReverseHosts,
	"IPList":           decodeResultIPList,
	"Rules":            decodeResultRules,
	"DNSRewriteResult": decodeResultDNSRewriteResult,
}

// decodeLogEntry decodes string str to logEntry ent.
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
