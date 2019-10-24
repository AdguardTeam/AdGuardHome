package querylog

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/AdguardTeam/AdGuardHome/dnsfilter"
	"github.com/AdguardTeam/golibs/log"
	"github.com/miekg/dns"
)

const enableGzip = false
const maxEntrySize = 1000

// flushLogBuffer flushes the current buffer to file and resets the current buffer
func (l *queryLog) flushLogBuffer(fullFlush bool) error {
	l.fileFlushLock.Lock()
	defer l.fileFlushLock.Unlock()

	// flush remainder to file
	l.bufferLock.Lock()
	needFlush := len(l.buffer) >= logBufferCap
	if !needFlush && !fullFlush {
		l.bufferLock.Unlock()
		return nil
	}
	flushBuffer := l.buffer
	l.buffer = nil
	l.flushPending = false
	l.bufferLock.Unlock()
	err := l.flushToFile(flushBuffer)
	if err != nil {
		log.Error("Saving querylog to file failed: %s", err)
		return err
	}
	return nil
}

// flushToFile saves the specified log entries to the query log file
func (l *queryLog) flushToFile(buffer []*logEntry) error {
	if len(buffer) == 0 {
		log.Debug("querylog: there's nothing to write to a file")
		return nil
	}
	start := time.Now()

	var b bytes.Buffer
	e := json.NewEncoder(&b)
	for _, entry := range buffer {
		err := e.Encode(entry)
		if err != nil {
			log.Error("Failed to marshal entry: %s", err)
			return err
		}
	}

	elapsed := time.Since(start)
	log.Debug("%d elements serialized via json in %v: %d kB, %v/entry, %v/entry", len(buffer), elapsed, b.Len()/1024, float64(b.Len())/float64(len(buffer)), elapsed/time.Duration(len(buffer)))

	var err error
	var zb bytes.Buffer
	filename := l.logFile

	// gzip enabled?
	if enableGzip {
		filename += ".gz"

		zw := gzip.NewWriter(&zb)
		zw.Name = l.logFile
		zw.ModTime = time.Now()

		_, err = zw.Write(b.Bytes())
		if err != nil {
			log.Error("Couldn't compress to gzip: %s", err)
			zw.Close()
			return err
		}

		if err = zw.Close(); err != nil {
			log.Error("Couldn't close gzip writer: %s", err)
			return err
		}
	} else {
		zb = b
	}

	l.fileWriteLock.Lock()
	defer l.fileWriteLock.Unlock()
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		log.Error("failed to create file \"%s\": %s", filename, err)
		return err
	}
	defer f.Close()

	n, err := f.Write(zb.Bytes())
	if err != nil {
		log.Error("Couldn't write to file: %s", err)
		return err
	}

	log.Debug("ok \"%s\": %v bytes written", filename, n)

	return nil
}

func (l *queryLog) rotate() error {
	from := l.logFile
	to := l.logFile + ".1"

	if enableGzip {
		from = l.logFile + ".gz"
		to = l.logFile + ".gz.1"
	}

	if _, err := os.Stat(from); os.IsNotExist(err) {
		// do nothing, file doesn't exist
		return nil
	}

	err := os.Rename(from, to)
	if err != nil {
		log.Error("Failed to rename querylog: %s", err)
		return err
	}

	log.Debug("Rotated from %s to %s successfully", from, to)

	return nil
}

func (l *queryLog) periodicRotate() {
	for range time.Tick(time.Duration(l.conf.Interval) * 24 * time.Hour) {
		err := l.rotate()
		if err != nil {
			log.Error("Failed to rotate querylog: %s", err)
			// do nothing, continue rotating
		}
	}
}

// Reader is the DB reader context
type Reader struct {
	ql     *queryLog
	search *getDataParams

	f         *os.File
	reader    *bufio.Reader // reads file line by line
	now       time.Time
	validFrom int64 // UNIX time (ns)
	olderThan int64 // UNIX time (ns)
	oldest    time.Time

	files []string
	ifile int

	limit        uint64
	count        uint64 // counter for returned elements
	latest       bool   // return the latest entries
	filePrepared bool

	seeking       bool       // we're seaching for an entry with exact time stamp
	fseeker       fileSeeker // file seeker object
	fpos          uint64     // current file offset
	nSeekRequests uint32     // number of Seek() requests made (finding a new line doesn't count)

	timecnt uint64
}

type fileSeeker struct {
	target uint64 // target value

	pos     uint64 // current offset, may be adjusted by user for increased accuracy
	lastpos uint64 // the last offset returned
	lo      uint64 // low boundary offset
	hi      uint64 // high boundary offset
}

// OpenReader - return reader object
func (l *queryLog) OpenReader() *Reader {
	r := Reader{}
	r.ql = l
	r.now = time.Now()
	r.validFrom = r.now.Unix() - int64(l.conf.Interval*24*60*60)
	r.validFrom *= 1000000000
	r.files = []string{
		r.ql.logFile,
		r.ql.logFile + ".1",
	}
	return &r
}

// Close - close the reader
func (r *Reader) Close() {
	elapsed := time.Since(r.now)
	var perunit time.Duration
	if r.count > 0 {
		perunit = elapsed / time.Duration(r.count)
	}
	log.Debug("querylog: read %d entries in %v, %v/entry, seek-reqs:%d  time:%dus (%d%%)",
		r.count, elapsed, perunit, r.nSeekRequests, r.timecnt/1000, r.timecnt*100/uint64(elapsed.Nanoseconds()))

	if r.f != nil {
		r.f.Close()
	}
}

// BeginRead - start reading
// olderThan: stop returning entries when an entry with this time is reached
// count: minimum number of entries to return
func (r *Reader) BeginRead(olderThan time.Time, count uint64, search *getDataParams) {
	r.olderThan = olderThan.UnixNano()
	r.latest = olderThan.IsZero()
	r.oldest = time.Time{}
	r.search = search
	r.limit = count
	if r.latest {
		r.olderThan = r.now.UnixNano()
	}
	r.filePrepared = false
	r.seeking = false
}

// BeginReadPrev - start reading the previous data chunk
func (r *Reader) BeginReadPrev(count uint64) {
	r.olderThan = r.oldest.UnixNano()
	r.oldest = time.Time{}
	r.latest = false
	r.limit = count
	r.count = 0

	off := r.fpos - maxEntrySize*(r.limit+1)
	if int64(off) < maxEntrySize {
		off = 0
	}
	r.fpos = off
	log.Debug("QueryLog: seek: %x", off)
	_, err := r.f.Seek(int64(off), io.SeekStart)
	if err != nil {
		log.Error("file.Seek: %s: %s", r.files[r.ifile], err)
		return
	}
	r.nSeekRequests++

	r.seekToNewLine()
	r.fseeker.pos = r.fpos

	r.filePrepared = true
	r.seeking = false
}

// Perform binary seek
// Return 0: success;  1: seek reqiured;  -1: error
func (fs *fileSeeker) seekBinary(cur uint64) int32 {
	log.Debug("QueryLog: seek: tgt=%x cur=%x, %x: [%x..%x]", fs.target, cur, fs.pos, fs.lo, fs.hi)

	off := uint64(0)
	if fs.pos >= fs.lo && fs.pos < fs.hi {
		if cur == fs.target {
			return 0
		} else if cur < fs.target {
			fs.lo = fs.pos + 1
		} else {
			fs.hi = fs.pos
		}
		off = fs.lo + (fs.hi-fs.lo)/2
	} else {
		// we didn't find another entry from the last file offset: now return the boundary beginning
		off = fs.lo
	}

	if off == fs.lastpos {
		return -1
	}

	fs.lastpos = off
	fs.pos = off
	return 1
}

// Seek to a new line
func (r *Reader) seekToNewLine() bool {
	r.reader = bufio.NewReader(r.f)
	b, err := r.reader.ReadBytes('\n')
	if err != nil {
		r.reader = nil
		log.Error("QueryLog: file.Read: %s: %s", r.files[r.ifile], err)
		return false
	}

	off := len(b)
	r.fpos += uint64(off)
	log.Debug("QueryLog: seek: %x (+%d)", r.fpos, off)
	return true
}

// Open a file
func (r *Reader) openFile() bool {
	var err error
	fn := r.files[r.ifile]

	r.f, err = os.Open(fn)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Error("QueryLog: Failed to open file \"%s\": %s", fn, err)
		}
		return false
	}
	return true
}

// Seek to the needed position
func (r *Reader) prepareRead() bool {
	fn := r.files[r.ifile]

	fi, err := r.f.Stat()
	if err != nil {
		log.Error("QueryLog: file.Stat: %s: %s", fn, err)
		return false
	}
	fsize := uint64(fi.Size())

	off := uint64(0)
	if r.latest {
		// read data from the end of file
		off = fsize - maxEntrySize*(r.limit+1)
		if int64(off) < maxEntrySize {
			off = 0
		}
		r.fpos = off
		log.Debug("QueryLog: seek: %x", off)
		_, err = r.f.Seek(int64(off), io.SeekStart)
		if err != nil {
			log.Error("QueryLog: file.Seek: %s: %s", fn, err)
			return false
		}
	} else {
		// start searching in file: we'll read the first chunk of data from the middle of file
		r.seeking = true
		r.fseeker = fileSeeker{}
		r.fseeker.target = uint64(r.olderThan)
		r.fseeker.hi = fsize
		rc := r.fseeker.seekBinary(0)
		r.fpos = r.fseeker.pos
		if rc == 1 {
			_, err = r.f.Seek(int64(r.fpos), io.SeekStart)
			if err != nil {
				log.Error("QueryLog: file.Seek: %s: %s", fn, err)
				return false
			}
		}
	}
	r.nSeekRequests++

	if !r.seekToNewLine() {
		return false
	}
	r.fseeker.pos = r.fpos
	return true
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

func (r *Reader) applySearch(str string) bool {
	if r.search.ResponseStatus == responseStatusFiltered {
		boolVal, ok := readJSONBool(str, "IsFiltered")
		if !ok || !boolVal {
			return false
		}
	}

	if len(r.search.Domain) != 0 {
		val := readJSONValue(str, "QH")
		if len(val) == 0 {
			return false
		}

		if (r.search.StrictMatchDomain && val != r.search.Domain) ||
			(!r.search.StrictMatchDomain && strings.Index(val, r.search.Domain) == -1) {
			return false
		}
	}

	if len(r.search.QuestionType) != 0 {
		val := readJSONValue(str, "QT")
		if len(val) == 0 {
			return false
		}
		if val != r.search.QuestionType {
			return false
		}
	}

	if len(r.search.Client) != 0 {
		val := readJSONValue(str, "IP")
		if len(val) == 0 {
			log.Debug("QueryLog: failed to decode")
			return false
		}

		if (r.search.StrictMatchClient && val != r.search.Client) ||
			(!r.search.StrictMatchClient && strings.Index(val, r.search.Client) == -1) {
			return false
		}
	}

	return true
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

// nolint (gocyclo)
func decode(ent *logEntry, str string) {
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
			ent.IP = v
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
			log.Debug("decode err: %s", err)
			break
		}
	}
}

// Next - return the next entry or nil if reading is finished
func (r *Reader) Next() *logEntry { // nolint
	for {
		// open file if needed
		if r.f == nil {
			if r.ifile == len(r.files) {
				return nil
			}
			if !r.openFile() {
				r.ifile++
				continue
			}
		}

		if !r.filePrepared {
			if !r.prepareRead() {
				return nil
			}
			r.filePrepared = true
		}

		b, err := r.reader.ReadBytes('\n')
		if err != nil {
			return nil
		}
		str := string(b)

		val := readJSONValue(str, "T")
		if len(val) == 0 {
			val = readJSONValue(str, "Time")
		}
		if len(val) == 0 {
			log.Debug("QueryLog: failed to decode")
			continue
		}
		tm, err := time.Parse(time.RFC3339, val)
		if err != nil {
			log.Debug("QueryLog: failed to decode")
			continue
		}
		t := tm.UnixNano()

		if r.seeking {

			r.reader = nil
			rr := r.fseeker.seekBinary(uint64(t))
			r.fpos = r.fseeker.pos
			if rr < 0 {
				log.Error("QueryLog: File seek error: can't find the target entry: %s", r.files[r.ifile])
				return nil
			} else if rr == 0 {
				// We found the target entry.
				// We'll start reading the previous chunk of data.
				r.seeking = false

				off := r.fpos - (maxEntrySize * (r.limit + 1))
				if int64(off) < maxEntrySize {
					off = 0
				}
				r.fpos = off
			}

			_, err = r.f.Seek(int64(r.fpos), io.SeekStart)
			if err != nil {
				log.Error("QueryLog: file.Seek: %s: %s", r.files[r.ifile], err)
				return nil
			}
			r.nSeekRequests++

			if !r.seekToNewLine() {
				return nil
			}
			r.fseeker.pos = r.fpos
			continue
		}

		if r.oldest.IsZero() {
			r.oldest = tm
		}

		if t < r.validFrom {
			continue
		}
		if t >= r.olderThan {
			return nil
		}
		r.count++

		if !r.applySearch(str) {
			continue
		}

		st := time.Now()
		var ent logEntry
		decode(&ent, str)
		r.timecnt += uint64(time.Now().Sub(st).Nanoseconds())

		return &ent
	}
}

// Total returns the total number of processed items
func (r *Reader) Total() uint64 {
	return r.count
}

// Oldest returns the time of the oldest processed entry
func (r *Reader) Oldest() time.Time {
	return r.oldest
}
