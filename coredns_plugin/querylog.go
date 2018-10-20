package dnsfilter

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/dnsfilter"
	"github.com/coredns/coredns/plugin/pkg/response"
	"github.com/miekg/dns"
)

const (
	logBufferCap           = 5000            // maximum capacity of logBuffer before it's flushed to disk
	queryLogTimeLimit      = time.Hour * 24  // how far in the past we care about querylogs
	queryLogRotationPeriod = time.Hour * 24  // rotate the log every 24 hours
	queryLogFileName       = "querylog.json" // .gz added during compression
	queryLogSize           = 5000            // maximum API response for /querylog
	queryLogTopSize        = 500             // Keep in memory only top N values
	queryLogAPIPort        = "8618"          // 8618 is sha512sum of "querylog" then each byte summed
)

var (
	logBufferLock sync.RWMutex
	logBuffer     []*logEntry

	queryLogCache []*logEntry
	queryLogLock  sync.RWMutex
	queryLogTime  time.Time
)

type logEntry struct {
	Question []byte
	Answer   []byte `json:",omitempty"` // sometimes empty answers happen like binerdunt.top or rev2.globalrootservers.net
	Result   dnsfilter.Result
	Time     time.Time
	Elapsed  time.Duration
	IP       string
}

func logRequest(question *dns.Msg, answer *dns.Msg, result dnsfilter.Result, elapsed time.Duration, ip string) {
	var q []byte
	var a []byte
	var err error

	if question != nil {
		q, err = question.Pack()
		if err != nil {
			log.Printf("failed to pack question for querylog: %s", err)
			return
		}
	}
	if answer != nil {
		a, err = answer.Pack()
		if err != nil {
			log.Printf("failed to pack answer for querylog: %s", err)
			return
		}
	}

	now := time.Now()
	entry := logEntry{
		Question: q,
		Answer:   a,
		Result:   result,
		Time:     now,
		Elapsed:  elapsed,
		IP:       ip,
	}
	var flushBuffer []*logEntry

	logBufferLock.Lock()
	logBuffer = append(logBuffer, &entry)
	if len(logBuffer) >= logBufferCap {
		flushBuffer = logBuffer
		logBuffer = nil
	}
	logBufferLock.Unlock()
	queryLogLock.Lock()
	queryLogCache = append(queryLogCache, &entry)
	if len(queryLogCache) > queryLogSize {
		toremove := len(queryLogCache) - queryLogSize
		queryLogCache = queryLogCache[toremove:]
	}
	queryLogLock.Unlock()

	// add it to running top
	err = runningTop.addEntry(&entry, question, now)
	if err != nil {
		log.Printf("Failed to add entry to running top: %s", err)
		// don't do failure, just log
	}

	// if buffer needs to be flushed to disk, do it now
	if len(flushBuffer) > 0 {
		// write to file
		// do it in separate goroutine -- we are stalling DNS response this whole time
		go flushToFile(flushBuffer)
	}
}

func HandleQueryLog(w http.ResponseWriter, r *http.Request) {
	queryLogLock.RLock()
	values := make([]*logEntry, len(queryLogCache))
	copy(values, queryLogCache)
	queryLogLock.RUnlock()

	// reverse it so that newest is first
	for left, right := 0, len(values)-1; left < right; left, right = left+1, right-1 {
		values[left], values[right] = values[right], values[left]
	}

	var data = []map[string]interface{}{}
	for _, entry := range values {
		var q *dns.Msg
		var a *dns.Msg

		if len(entry.Question) > 0 {
			q = new(dns.Msg)
			if err := q.Unpack(entry.Question); err != nil {
				// ignore, log and move on
				log.Printf("Failed to unpack dns message question: %s", err)
				q = nil
			}
		}
		if len(entry.Answer) > 0 {
			a = new(dns.Msg)
			if err := a.Unpack(entry.Answer); err != nil {
				// ignore, log and move on
				log.Printf("Failed to unpack dns message question: %s", err)
				a = nil
			}
		}

		jsonentry := map[string]interface{}{
			"reason":     entry.Result.Reason.String(),
			"elapsed_ms": strconv.FormatFloat(entry.Elapsed.Seconds()*1000, 'f', -1, 64),
			"time":       entry.Time.Format(time.RFC3339),
			"client":     entry.IP,
		}
		if q != nil {
			jsonentry["question"] = map[string]interface{}{
				"host":  strings.ToLower(strings.TrimSuffix(q.Question[0].Name, ".")),
				"type":  dns.Type(q.Question[0].Qtype).String(),
				"class": dns.Class(q.Question[0].Qclass).String(),
			}
		}

		if a != nil {
			status, _ := response.Typify(a, time.Now().UTC())
			jsonentry["status"] = status.String()
		}
		if len(entry.Result.Rule) > 0 {
			jsonentry["rule"] = entry.Result.Rule
		}

		if a != nil && len(a.Answer) > 0 {
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
					answer["value"] = v.A
				case *dns.AAAA:
					answer["value"] = v.AAAA
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
			jsonentry["answer"] = answers
		}

		data = append(data, jsonentry)
	}

	jsonVal, err := json.Marshal(data)
	if err != nil {
		errortext := fmt.Sprintf("Couldn't marshal data into json: %s", err)
		log.Println(errortext)
		http.Error(w, errortext, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(jsonVal)
	if err != nil {
		errortext := fmt.Sprintf("Unable to write response json: %s", err)
		log.Println(errortext)
		http.Error(w, errortext, http.StatusInternalServerError)
	}
}

func trace(format string, args ...interface{}) {
	pc := make([]uintptr, 10) // at least 1 entry needed
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("%s(): ", path.Base(f.Name())))
	text := fmt.Sprintf(format, args...)
	buf.WriteString(text)
	if len(text) == 0 || text[len(text)-1] != '\n' {
		buf.WriteRune('\n')
	}
	fmt.Fprint(os.Stderr, buf.String())
}
