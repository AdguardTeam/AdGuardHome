package dnsfilter

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/AdguardTeam/AdguardDNS/dnsfilter"
	"github.com/coredns/coredns/plugin/pkg/response"
	"github.com/miekg/dns"
	"github.com/zfjagann/golang-ring"
)

var logBuffer = ring.Ring{}

type logEntry struct {
	R       *dns.Msg
	Result  dnsfilter.Result
	Time    time.Time
	Elapsed time.Duration
}

func init() {
	logBuffer.SetCapacity(1000)
}

func logRequest(r *dns.Msg, result dnsfilter.Result, elapsed time.Duration) {
	entry := logEntry{
		R:       r,
		Result:  result,
		Time:    time.Now(),
		Elapsed: elapsed,
	}
	logBuffer.Enqueue(entry)
}

func handler(w http.ResponseWriter, r *http.Request) {
	values := logBuffer.Values()
	var data = []map[string]interface{}{}
	for _, value := range values {
		entry, ok := value.(logEntry)
		if !ok {
			continue
		}

		jsonentry := map[string]interface{}{
			"reason":     entry.Result.Reason.String(),
			"elapsed_ms": strconv.FormatFloat(entry.Elapsed.Seconds()*1000, 'f', -1, 64),
			"time":       entry.Time.Format(time.RFC3339),
		}
		question := map[string]interface{}{
			"host":  strings.ToLower(strings.TrimSuffix(entry.R.Question[0].Name, ".")),
			"type":  dns.Type(entry.R.Question[0].Qtype).String(),
			"class": dns.Class(entry.R.Question[0].Qclass).String(),
		}
		jsonentry["question"] = question

		status, _ := response.Typify(entry.R, time.Now().UTC())
		jsonentry["status"] = status.String()
		if len(entry.Result.Rule) > 0 {
			jsonentry["rule"] = entry.Result.Rule
		}

		if len(entry.R.Answer) > 0 {
			var answers = []map[string]interface{}{}
			for _, k := range entry.R.Answer {
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

	json, err := json.Marshal(data)
	if err != nil {
		errortext := fmt.Sprintf("Couldn't marshal data into json: %s", err)
		log.Println(errortext)
		http.Error(w, errortext, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(json)
	if err != nil {
		errortext := fmt.Sprintf("Unable to write response json: %s", err)
		log.Println(errortext)
		http.Error(w, errortext, 500)
	}
}

func startQueryLogServer() {
	listenAddr := "127.0.0.1:8618" // sha512sum of "querylog" then each byte summed

	http.HandleFunc("/querylog", handler)
	if err := http.ListenAndServe(listenAddr, nil); err != nil {
		log.Fatalf("error in ListenAndServe: %s", err)
	}
}

func trace(text string) {
	pc := make([]uintptr, 10) // at least 1 entry needed
	runtime.Callers(2, pc)
	f := runtime.FuncForPC(pc[0])
	log.Printf("%s(): %s\n", f.Name(), text)
}
