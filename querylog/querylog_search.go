package querylog

import (
	"encoding/base64"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/dnsfilter"
	"github.com/AdguardTeam/AdGuardHome/util"
	"github.com/AdguardTeam/golibs/log"
	"github.com/miekg/dns"
)

// searchFiles reads log entries from all log files and applies the specified search criteria.
// IMPORTANT: this method does not scan more than "maxSearchEntries" so you
// may need to call it many times.
//
// it returns:
// * an array of log entries that we have read
// * time of the oldest processed entry (even if it was discarded)
// * total number of processed entries (including discarded).
func (l *queryLog) searchFiles(params getDataParams) ([]*logEntry, time.Time, int) {
	entries := make([]*logEntry, 0)
	oldest := time.Time{}

	r, err := l.openReader()
	if err != nil {
		log.Error("Failed to open qlog reader: %v", err)
		return entries, oldest, 0
	}
	defer r.Close()

	if params.OlderThan.IsZero() {
		err = r.SeekStart()
	} else {
		err = r.Seek(params.OlderThan.UnixNano())
		if err == nil {
			// Read to the next record right away
			// The one that was specified in the "oldest" param is not needed,
			// we need only the one next to it
			_, err = r.ReadNext()
		}
	}

	if err != nil {
		log.Debug("Cannot Seek() to %v: %v", params.OlderThan, err)
		return entries, oldest, 0
	}

	total := 0
	oldestNano := int64(0)
	// Do not scan more than 50k at once
	for total <= maxSearchEntries {
		entry, ts, err := l.readNextEntry(r, params)

		if err == io.EOF {
			// there's nothing to read anymore
			break
		}

		oldestNano = ts
		total++

		if entry != nil {
			entries = append(entries, entry)
			if len(entries) == getDataLimit {
				// Do not read more than "getDataLimit" records at once
				break
			}
		}
	}

	oldest = time.Unix(0, oldestNano)
	return entries, oldest, total
}

// readNextEntry - reads the next log entry and checks if it matches the search criteria (getDataParams)
//
// returns:
// * log entry that matches search criteria or null if it was discarded (or if there's nothing to read)
// * timestamp of the processed log entry
// * error if we can't read anymore
func (l *queryLog) readNextEntry(r *QLogReader, params getDataParams) (*logEntry, int64, error) {
	line, err := r.ReadNext()
	if err != nil {
		return nil, 0, err
	}

	// Read the log record timestamp right away
	timestamp := readQLogTimestamp(line)

	// Quick check without deserializing log entry
	if !quickMatchesGetDataParams(line, params) {
		return nil, timestamp, nil
	}

	entry := logEntry{}
	decodeLogEntry(&entry, line)

	// Full check of the deserialized log entry
	if !matchesGetDataParams(&entry, params) {
		return nil, timestamp, nil
	}

	return &entry, timestamp, nil
}

// openReader - opens QLogReader instance
func (l *queryLog) openReader() (*QLogReader, error) {
	files := make([]string, 0)

	if util.FileExists(l.logFile + ".1") {
		files = append(files, l.logFile+".1")
	}
	if util.FileExists(l.logFile) {
		files = append(files, l.logFile)
	}

	return NewQLogReader(files)
}

// quickMatchesGetDataParams - quickly checks if the line matches getDataParams
// this method does not guarantee anything and the reason is to do a quick check
// without deserializing anything
func quickMatchesGetDataParams(line string, params getDataParams) bool {
	if params.ResponseStatus == responseStatusFiltered {
		boolVal, ok := readJSONBool(line, "IsFiltered")
		if !ok || !boolVal {
			return false
		}
	}

	if len(params.Domain) != 0 {
		val := readJSONValue(line, "QH")
		if len(val) == 0 {
			return false
		}

		if (params.StrictMatchDomain && val != params.Domain) ||
			(!params.StrictMatchDomain && strings.Index(val, params.Domain) == -1) {
			return false
		}
	}

	if len(params.QuestionType) != 0 {
		val := readJSONValue(line, "QT")
		if val != params.QuestionType {
			return false
		}
	}

	if len(params.Client) != 0 {
		val := readJSONValue(line, "IP")
		if len(val) == 0 {
			log.Debug("QueryLog: failed to decodeLogEntry")
			return false
		}

		if (params.StrictMatchClient && val != params.Client) ||
			(!params.StrictMatchClient && strings.Index(val, params.Client) == -1) {
			return false
		}
	}

	return true
}

// matchesGetDataParams - returns true if the entry matches the search parameters
func matchesGetDataParams(entry *logEntry, params getDataParams) bool {
	if params.ResponseStatus == responseStatusFiltered && !entry.Result.IsFiltered {
		return false
	}

	if len(params.QuestionType) != 0 {
		if entry.QType != params.QuestionType {
			return false
		}
	}

	if len(params.Domain) != 0 {
		if (params.StrictMatchDomain && entry.QHost != params.Domain) ||
			(!params.StrictMatchDomain && strings.Index(entry.QHost, params.Domain) == -1) {
			return false
		}
	}

	if len(params.Client) != 0 {
		if (params.StrictMatchClient && entry.IP != params.Client) ||
			(!params.StrictMatchClient && strings.Index(entry.IP, params.Client) == -1) {
			return false
		}
	}

	return true
}

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

// Get bool value from "key":bool
func readJSONBool(s, name string) (bool, bool) {
	i := strings.Index(s, "\""+name+"\":")
	if i == -1 {
		return false, false
	}
	start := i + 1 + len(name) + 2
	b := false
	if strings.HasPrefix(s[start:], "true") {
		b = true
	} else if !strings.HasPrefix(s[start:], "false") {
		return false, false
	}
	return b, true
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
