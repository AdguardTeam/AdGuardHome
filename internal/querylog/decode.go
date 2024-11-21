package querylog

import (
	"context"
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
	"github.com/AdguardTeam/golibs/logutil/slogutil"
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
func (l *queryLog) decodeResultRuleKey(
	ctx context.Context,
	key string,
	i int,
	dec *json.Decoder,
	ent *logEntry,
) {
	var vToken json.Token
	switch key {
	case "FilterListID":
		ent.Result.Rules, vToken = l.decodeVTokenAndAddRule(ctx, key, i, dec, ent.Result.Rules)
		if n, ok := vToken.(json.Number); ok {
			id, _ := n.Int64()
			ent.Result.Rules[i].FilterListID = rulelist.URLFilterID(id)
		}
	case "IP":
		ent.Result.Rules, vToken = l.decodeVTokenAndAddRule(ctx, key, i, dec, ent.Result.Rules)
		if ipStr, ok := vToken.(string); ok {
			if ip, err := netip.ParseAddr(ipStr); err == nil {
				ent.Result.Rules[i].IP = ip
			} else {
				l.logger.DebugContext(ctx, "decoding ip", "value", ipStr, slogutil.KeyError, err)
			}
		}
	case "Text":
		ent.Result.Rules, vToken = l.decodeVTokenAndAddRule(ctx, key, i, dec, ent.Result.Rules)
		if s, ok := vToken.(string); ok {
			ent.Result.Rules[i].Text = s
		}
	default:
		// Go on.
	}
}

// decodeVTokenAndAddRule decodes the "Rules" toke as [filtering.ResultRule]
// and then adds the decoded object to the slice of result rules.
func (l *queryLog) decodeVTokenAndAddRule(
	ctx context.Context,
	key string,
	i int,
	dec *json.Decoder,
	rules []*filtering.ResultRule,
) (newRules []*filtering.ResultRule, vToken json.Token) {
	newRules = rules

	vToken, err := dec.Token()
	if err != nil {
		if err != io.EOF {
			l.logger.DebugContext(
				ctx,
				"decoding result rule key",
				"key", key,
				slogutil.KeyError, err,
			)
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
func (l *queryLog) decodeResultRules(ctx context.Context, dec *json.Decoder, ent *logEntry) {
	const msgPrefix = "decoding result rules"

	for {
		delimToken, err := dec.Token()
		if err != nil {
			if err != io.EOF {
				l.logger.DebugContext(ctx, msgPrefix+"; token", slogutil.KeyError, err)
			}

			return
		}

		if d, ok := delimToken.(json.Delim); !ok {
			return
		} else if d != '[' {
			l.logger.DebugContext(
				ctx,
				msgPrefix,
				slogutil.KeyError, newUnexpectedDelimiterError(d),
			)
		}

		err = l.decodeResultRuleToken(ctx, dec, ent)
		if err != nil {
			if err != io.EOF && !errors.Is(err, ErrEndOfToken) {
				l.logger.DebugContext(ctx, msgPrefix+"; rule token", slogutil.KeyError, err)
			}

			return
		}
	}
}

// decodeResultRuleToken decodes the tokens of "Rules" type to the logEntry ent.
func (l *queryLog) decodeResultRuleToken(
	ctx context.Context,
	dec *json.Decoder,
	ent *logEntry,
) (err error) {
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

		l.decodeResultRuleKey(ctx, key, i, dec, ent)
	}
}

// decodeResultReverseHosts parses the dec's tokens into ent interpreting it as
// the result of hosts container's $dnsrewrite rule.  It assumes there are no
// other occurrences of DNSRewriteResult in the entry since hosts container's
// rewrites currently has the highest priority along the entire filtering
// pipeline.
func (l *queryLog) decodeResultReverseHosts(ctx context.Context, dec *json.Decoder, ent *logEntry) {
	const msgPrefix = "decoding result reverse hosts"

	for {
		itemToken, err := dec.Token()
		if err != nil {
			if err != io.EOF {
				l.logger.DebugContext(ctx, msgPrefix+"; token", slogutil.KeyError, err)
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

			l.logger.DebugContext(
				ctx,
				msgPrefix,
				slogutil.KeyError, newUnexpectedDelimiterError(v),
			)

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
func (l *queryLog) decodeResultIPList(ctx context.Context, dec *json.Decoder, ent *logEntry) {
	const msgPrefix = "decoding result ip list"

	for {
		itemToken, err := dec.Token()
		if err != nil {
			if err != io.EOF {
				l.logger.DebugContext(ctx, msgPrefix+"; token", slogutil.KeyError, err)
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

			l.logger.DebugContext(
				ctx,
				msgPrefix,
				slogutil.KeyError, newUnexpectedDelimiterError(v),
			)

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
func (l *queryLog) decodeResultDNSRewriteResultKey(
	ctx context.Context,
	key string,
	dec *json.Decoder,
	ent *logEntry,
) {
	const msgPrefix = "decoding result dns rewrite result key"

	var err error

	switch key {
	case "RCode":
		var vToken json.Token
		vToken, err = dec.Token()
		if err != nil {
			if err != io.EOF {
				l.logger.DebugContext(ctx, msgPrefix+"; token", slogutil.KeyError, err)
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
			l.logger.DebugContext(ctx, msgPrefix+"; response", slogutil.KeyError, err)
		}

		ent.parseDNSRewriteResultIPs()
	default:
		// Go on.
	}
}

// decodeResultDNSRewriteResult parses the dec's tokens into logEntry ent
// interpreting it as the result DNSRewriteResult.
func (l *queryLog) decodeResultDNSRewriteResult(
	ctx context.Context,
	dec *json.Decoder,
	ent *logEntry,
) {
	const msgPrefix = "decoding result dns rewrite result"

	for {
		key, err := parseKeyToken(dec)
		if err != nil {
			if err != io.EOF && !errors.Is(err, ErrEndOfToken) {
				l.logger.DebugContext(ctx, msgPrefix+"; token", slogutil.KeyError, err)
			}

			return
		}

		if key == "" {
			continue
		}

		l.decodeResultDNSRewriteResultKey(ctx, key, dec, ent)
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
func (l *queryLog) decodeResult(ctx context.Context, dec *json.Decoder, ent *logEntry) {
	const msgPrefix = "decoding result"

	defer translateResult(ent)

	for {
		key, err := parseKeyToken(dec)
		if err != nil {
			if err != io.EOF && !errors.Is(err, ErrEndOfToken) {
				l.logger.DebugContext(ctx, msgPrefix+"; token", slogutil.KeyError, err)
			}

			return
		}

		if key == "" {
			continue
		}

		ok := l.resultDecHandler(ctx, key, dec, ent)
		if ok {
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
			l.logger.DebugContext(ctx, msgPrefix+"; handler", slogutil.KeyError, err)

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

// resultDecHandlers calls a decode handler for key if there is one.
func (l *queryLog) resultDecHandler(
	ctx context.Context,
	name string,
	dec *json.Decoder,
	ent *logEntry,
) (ok bool) {
	ok = true
	switch name {
	case "ReverseHosts":
		l.decodeResultReverseHosts(ctx, dec, ent)
	case "IPList":
		l.decodeResultIPList(ctx, dec, ent)
	case "Rules":
		l.decodeResultRules(ctx, dec, ent)
	case "DNSRewriteResult":
		l.decodeResultDNSRewriteResult(ctx, dec, ent)
	default:
		ok = false
	}

	return ok
}

// decodeLogEntry decodes string str to logEntry ent.
func (l *queryLog) decodeLogEntry(ctx context.Context, ent *logEntry, str string) {
	const msgPrefix = "decoding log entry"

	dec := json.NewDecoder(strings.NewReader(str))
	dec.UseNumber()

	for {
		keyToken, err := dec.Token()
		if err != nil {
			if err != io.EOF {
				l.logger.DebugContext(ctx, msgPrefix+"; token", slogutil.KeyError, err)
			}

			return
		}

		if _, ok := keyToken.(json.Delim); ok {
			continue
		}

		key, ok := keyToken.(string)
		if !ok {
			err = fmt.Errorf("%s: keyToken is %T (%[2]v) and not string", msgPrefix, keyToken)
			l.logger.DebugContext(ctx, msgPrefix, slogutil.KeyError, err)

			return
		}

		if key == "Result" {
			l.decodeResult(ctx, dec, ent)

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
			l.logger.DebugContext(ctx, msgPrefix+"; handler", slogutil.KeyError, err)

			return
		}
	}
}

// newUnexpectedDelimiterError is a helper for creating informative errors.
func newUnexpectedDelimiterError(d json.Delim) (err error) {
	return fmt.Errorf("unexpected delimiter: %q", d)
}
