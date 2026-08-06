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
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/urlfilter/rules"
	"github.com/miekg/dns"
)

// logEntryHandler represents a handler for decoding json token to the logEntry
// struct.  ent must not be nil.
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
// dec and ent must not be nil.
func (l *queryLog) decodeResultRuleKey(
	ctx context.Context,
	key string,
	idx int,
	dec *json.Decoder,
	ent *logEntry,
) {
	var vToken json.Token
	switch key {
	case "FilterListID":
		ent.Result.Rules, vToken = l.decodeVTokenAndAddRule(ctx, key, idx, dec, ent.Result.Rules)
		if n, ok := vToken.(json.Number); ok {
			id, _ := n.Int64()
			ent.Result.Rules[idx].FilterListID = rulelist.APIID(id)
		}
	case "IP":
		ent.Result.Rules, vToken = l.decodeVTokenAndAddRule(ctx, key, idx, dec, ent.Result.Rules)
		if ipStr, ok := vToken.(string); ok {
			ip, err := netip.ParseAddr(ipStr)
			if err != nil {
				l.logger.DebugContext(ctx, "decoding ip", "value", ipStr, slogutil.KeyError, err)

				return
			}

			ent.Result.Rules[idx].IP = ip
		}
	case "Text":
		ent.Result.Rules, vToken = l.decodeVTokenAndAddRule(ctx, key, idx, dec, ent.Result.Rules)
		if s, ok := vToken.(string); ok {
			ent.Result.Rules[idx].Text = s
		}
	default:
		// Go on.
	}
}

// decodeVTokenAndAddRule decodes the "Rules" toke as [filtering.ResultRule] and
// then adds the decoded object to the slice of result rules.  dec must not be
// nil.
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
// as a slice of the result rules.  All arguments must not be nil.
func (l *queryLog) decodeResultRules(ctx context.Context, dec *json.Decoder, ent *logEntry) {
	const msgPrefix = "decoding result rules"

	for {
		delimToken, err := dec.Token()
		switch err {
		case nil:
			// Go on.
		case io.EOF:
			return
		default:
			l.logger.DebugContext(ctx, msgPrefix+"; token", slogutil.KeyError, err)

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
		switch {
		case err == nil:
			continue
		case
			err == io.EOF,
			errors.Is(err, ErrEndOfToken):
			return
		default:
			l.logger.DebugContext(ctx, msgPrefix+"; rule token", slogutil.KeyError, err)

			return
		}
	}
}

// decodeResultRuleToken decodes the tokens of "Rules" type to the logEntry ent.
// All arguments must not be nil.
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
// pipeline.  All arguments must not be nil.
func (l *queryLog) decodeResultReverseHosts(ctx context.Context, dec *json.Decoder, ent *logEntry) {
	const msgPrefix = "decoding result reverse hosts"

	for {
		itemToken, err := dec.Token()
		switch err {
		case nil:
			// Go on.
		case io.EOF:
			return
		default:
			l.logger.DebugContext(ctx, msgPrefix+"; token", slogutil.KeyError, err)

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
			setPTRRewriteResult(v, ent)
		default:
			continue
		}
	}
}

// setPTRRewriteResult sets ent.Result.DNSRewriteResult.  ent must not be nil.
func setPTRRewriteResult(v string, ent *logEntry) {
	v = dns.Fqdn(v)
	res := &ent.Result

	if res.DNSRewriteResult == nil {
		res.DNSRewriteResult = &filtering.DNSRewriteResult{
			RCode: dns.RcodeSuccess,
			Response: filtering.DNSRewriteResultResponse{
				dns.TypePTR: []rules.RRValue{v},
			},
		}

		return
	}

	res.DNSRewriteResult.RCode = dns.RcodeSuccess

	if rres := ent.Result.DNSRewriteResult; rres.Response == nil {
		rres.Response = filtering.DNSRewriteResultResponse{dns.TypePTR: []rules.RRValue{v}}
	} else {
		rres.Response[dns.TypePTR] = append(rres.Response[dns.TypePTR], v)
	}
}

// decodeResultIPList parses the dec's tokens into logEntry ent interpreting it
// as the result IP addresses list.  All arguments must not be nil.
func (l *queryLog) decodeResultIPList(ctx context.Context, dec *json.Decoder, ent *logEntry) {
	const msgPrefix = "decoding result ip list"

	for {
		itemToken, err := dec.Token()
		switch err {
		case nil:
			// Go on.
		case io.EOF:
			return
		default:
			l.logger.DebugContext(ctx, msgPrefix+"; token", slogutil.KeyError, err)

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
			ent.Result.IPList = appendIfValidIP(ent.Result.IPList, v)
		default:
			continue
		}
	}
}

// appendIfValidIP appends a valid netip.Addr from s, if there is one, to orig
// and returns it.
func appendIfValidIP(orig []netip.Addr, s string) (res []netip.Addr) {
	res = orig

	if !netutil.IsValidIPString(s) {
		return res
	}

	return append(res, netip.MustParseAddr(s))
}

// decodeResultDNSRewriteResultKey decodes the token of "DNSRewriteResult" type
// to the logEntry struct.  dec and ent must not be nil.
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
		switch err {
		case nil:
			// Go on.
		case io.EOF:
			return
		default:
			l.logger.DebugContext(ctx, msgPrefix+"; token", slogutil.KeyError, err)

			return
		}

		ent.Result.DNSRewriteResult = ensureNonNil(ent.Result.DNSRewriteResult)

		if n, ok := vToken.(json.Number); ok {
			rcode64, _ := n.Int64()
			ent.Result.DNSRewriteResult.RCode = rules.RCode(rcode64)
		}
	case "Response":
		ent.Result.DNSRewriteResult = ensureNonNil(ent.Result.DNSRewriteResult)
		ent.Result.DNSRewriteResult.Response = ensureNonNilMap(ent.Result.DNSRewriteResult.Response)

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

// ensureNonNil returns a new non-nil pointer if ptr is nil; otherwise, it
// returns ptr.
func ensureNonNil[T any](ptr *T) (res *T) {
	if ptr == nil {
		return new(T)
	}

	return ptr
}

// ensureNonNilMap returns a new non-nil map if m is nil; otherwise, it returns
// m.
func ensureNonNilMap[K comparable, V any](m map[K]V) (res map[K]V) {
	if m == nil {
		return map[K]V{}
	}

	return m
}

// decodeResultDNSRewriteResult parses the dec's tokens into logEntry ent
// interpreting it as the result DNSRewriteResult.  All arguments must not be
// nil.
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
// consistent with current implementation.  ent must not be nil.
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

// parseKeyToken parses the dec's token key.  dec must not be nil.
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

// decodeResult decodes a token of "Result" type to logEntry struct.  All
// arguments must not be nil.
func (l *queryLog) decodeResult(ctx context.Context, dec *json.Decoder, ent *logEntry) {
	defer translateResult(ent)

	for l.decodeResultKeyValue(ctx, dec, ent) {
	}
}

// decodeResultKeyValue decodes a single entry key-value pair.  If ok is true,
// the decoding was successful.  All arguments must not be nil.
func (l *queryLog) decodeResultKeyValue(
	ctx context.Context,
	dec *json.Decoder,
	ent *logEntry,
) (ok bool) {
	const msgPrefix = "decoding result"

	key, err := parseKeyToken(dec)
	if err != nil {
		if err != io.EOF && !errors.Is(err, ErrEndOfToken) {
			l.logger.DebugContext(ctx, msgPrefix+"; token", slogutil.KeyError, err)
		}

		return false
	}

	if key == "" {
		return true
	}

	ok = l.resultDecHandler(ctx, key, dec, ent)
	if ok {
		return true
	}

	handler, ok := resultHandlers[key]
	if !ok {
		return true
	}

	val, err := dec.Token()
	if err != nil {
		l.logger.DebugContext(ctx, msgPrefix+"; token", slogutil.KeyError, err)

		return false
	}

	if err = handler(val, ent); err != nil {
		l.logger.DebugContext(ctx, msgPrefix+"; handler", slogutil.KeyError, err)

		return false
	}

	return true
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

		ent.Result.Rules[l-1].FilterListID = rulelist.APIID(id)

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

// resultDecHandlers calls a decode handler for key if there is one.  dec and
// ent must not be nil.
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

// decodeLogEntry decodes string str to logEntry ent.  ent must not be nil.
func (l *queryLog) decodeLogEntry(ctx context.Context, ent *logEntry, str string) {
	dec := json.NewDecoder(strings.NewReader(str))
	dec.UseNumber()

	for l.decodeLogEntryKeyValue(ctx, dec, ent) {
	}
}

// decodeLogEntryKeyValue decodes a single entry key-value pair.  If ok is true,
// the decoding was successful.  All arguments must not be nil.
func (l *queryLog) decodeLogEntryKeyValue(
	ctx context.Context,
	dec *json.Decoder,
	ent *logEntry,
) (ok bool) {
	const msgPrefix = "decoding log entry"

	keyToken, err := dec.Token()
	if err != nil {
		if err != io.EOF {
			l.logger.DebugContext(ctx, msgPrefix+"; token", slogutil.KeyError, err)
		}

		return false
	}

	_, ok = keyToken.(json.Delim)
	if ok {
		return true
	}

	key, ok := keyToken.(string)
	if !ok {
		err = fmt.Errorf("%s: keyToken is %T (%[2]v) and not string", msgPrefix, keyToken)
		l.logger.DebugContext(ctx, msgPrefix, slogutil.KeyError, err)

		return false
	}

	if key == "Result" {
		l.decodeResult(ctx, dec, ent)

		return true
	}

	handler, ok := logEntryHandlers[key]
	if !ok {
		return true
	}

	val, err := dec.Token()
	if err != nil {
		l.logger.DebugContext(ctx, msgPrefix+"; token", slogutil.KeyError, err)

		return false
	}

	if err = handler(val, ent); err != nil {
		l.logger.DebugContext(ctx, msgPrefix+"; handler", slogutil.KeyError, err)

		return false
	}

	return true
}

// newUnexpectedDelimiterError is a helper for creating informative errors.
func newUnexpectedDelimiterError(d json.Delim) (err error) {
	return fmt.Errorf("unexpected delimiter: %q", d)
}
