package querylog

import (
	"context"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/filtering"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/miekg/dns"
	"golang.org/x/net/idna"
)

// TODO(a.garipov): Use a proper structured approach here.

// jobject is a JSON object alias.
type jobject = map[string]any

// entriesToJSON converts query log entries to JSON.
func (l *queryLog) entriesToJSON(
	ctx context.Context,
	entries []*logEntry,
	oldest time.Time,
	anonFunc aghnet.IPMutFunc,
) (res jobject) {
	data := make([]jobject, 0, len(entries))

	// The elements order is already reversed to be from newer to older.
	for _, entry := range entries {
		jsonEntry := l.entryToJSON(ctx, entry, anonFunc)
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
func (l *queryLog) entryToJSON(
	ctx context.Context,
	entry *logEntry,
	anonFunc aghnet.IPMutFunc,
) (jsonEntry jobject) {
	hostname := entry.QHost
	question := jobject{
		"type":  entry.QType,
		"class": entry.QClass,
		"name":  hostname,
	}

	if qhost, err := idna.ToUnicode(hostname); err != nil {
		l.logger.DebugContext(
			ctx,
			"translating into unicode",
			"hostname", hostname,
			slogutil.KeyError, err,
		)
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

	l.setMsgData(ctx, entry, jsonEntry)
	l.setOrigAns(ctx, entry, jsonEntry)

	return jsonEntry
}

// setMsgData sets the message data in jsonEntry.
func (l *queryLog) setMsgData(ctx context.Context, entry *logEntry, jsonEntry jobject) {
	if len(entry.Answer) == 0 {
		return
	}

	msg := &dns.Msg{}
	if err := msg.Unpack(entry.Answer); err != nil {
		l.logger.DebugContext(
			ctx,
			"unpacking dns message",
			"answer", entry.Answer,
			slogutil.KeyError, err,
		)

		return
	}

	jsonEntry["status"] = dns.RcodeToString[msg.Rcode]
	// Old query logs may still keep AD flag value in the message.  Try to get
	// it from there as well.
	jsonEntry["answer_dnssec"] = entry.AuthenticatedData || msg.AuthenticatedData

	if a := answerToJSON(msg); a != nil {
		jsonEntry["answer"] = a
	}
}

// setOrigAns sets the original answer data in jsonEntry.
func (l *queryLog) setOrigAns(ctx context.Context, entry *logEntry, jsonEntry jobject) {
	if len(entry.OrigAnswer) == 0 {
		return
	}

	orig := &dns.Msg{}
	err := orig.Unpack(entry.OrigAnswer)
	if err != nil {
		l.logger.DebugContext(
			ctx,
			"setting original answer",
			"answer", entry.OrigAnswer,
			slogutil.KeyError, err,
		)

		return
	}

	if a := answerToJSON(orig); a != nil {
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

// answerToJSON converts the answer records of msg, if any, to their JSON form.
func answerToJSON(msg *dns.Msg) (answers []*dnsAnswer) {
	if msg == nil || len(msg.Answer) == 0 {
		return nil
	}

	answers = make([]*dnsAnswer, 0, len(msg.Answer))
	for _, rr := range msg.Answer {
		header := rr.Header()
		a := &dnsAnswer{
			Type: dns.TypeToString[header.Rrtype],
			// Remove the header string from the answer value since it's mostly
			// unnecessary in the log.
			Value: strings.TrimPrefix(rr.String(), header.String()),
			TTL:   header.Ttl,
		}

		answers = append(answers, a)
	}

	return answers
}
