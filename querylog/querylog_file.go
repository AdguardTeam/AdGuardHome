package querylog

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/AdguardTeam/golibs/log"
	"github.com/go-test/deep"
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

	err := checkBuffer(buffer, b)
	if err != nil {
		log.Error("failed to check buffer: %s", err)
		return err
	}

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

func checkBuffer(buffer []*logEntry, b bytes.Buffer) error {
	l := len(buffer)
	d := json.NewDecoder(&b)

	i := 0
	for d.More() {
		entry := &logEntry{}
		err := d.Decode(entry)
		if err != nil {
			log.Error("Failed to decode: %s", err)
			return err
		}
		if diff := deep.Equal(entry, buffer[i]); diff != nil {
			log.Error("decoded buffer differs: %s", diff)
			return fmt.Errorf("decoded buffer differs: %s", diff)
		}
		i++
	}
	if i != l {
		err := fmt.Errorf("check fail: %d vs %d entries", l, i)
		log.Error("%v", err)
		return err
	}
	log.Debug("check ok: %d entries", i)

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
	ql *queryLog

	f         *os.File
	jd        *json.Decoder
	now       time.Time
	validFrom int64 // UNIX time (ns)
	olderThan int64 // UNIX time (ns)

	files []string
	ifile int

	limit        uint64
	count        uint64 // counter for returned elements
	latest       bool   // return the latest entries
	filePrepared bool

	searching     bool       // we're seaching for an entry with exact time stamp
	fseeker       fileSeeker // file seeker object
	fpos          uint64     // current file offset
	nSeekRequests uint32     // number of Seek() requests made (finding a new line doesn't count)
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
	log.Debug("querylog: read %d entries in %v, %v/entry, seek-reqs:%d",
		r.count, elapsed, perunit, r.nSeekRequests)

	if r.f != nil {
		r.f.Close()
	}
}

// BeginRead - start reading
// olderThan: stop returning entries when an entry with this time is reached
// count: minimum number of entries to return
func (r *Reader) BeginRead(olderThan time.Time, count uint64) {
	r.olderThan = olderThan.UnixNano()
	r.latest = olderThan.IsZero()
	r.limit = count
	if r.latest {
		r.olderThan = r.now.UnixNano()
	}
	r.filePrepared = false
	r.searching = false
	r.jd = nil
}

// BeginReadPrev - start reading the previous data chunk
func (r *Reader) BeginReadPrev(olderThan time.Time, count uint64) {
	r.olderThan = olderThan.UnixNano()
	r.latest = olderThan.IsZero()
	r.limit = count
	if r.latest {
		r.olderThan = r.now.UnixNano()
	}

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
	r.searching = false
	r.jd = nil
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
	b := make([]byte, maxEntrySize*2)

	_, err := r.f.Read(b)
	if err != nil {
		log.Error("QueryLog: file.Read: %s: %s", r.files[r.ifile], err)
		return false
	}

	off := bytes.IndexByte(b, '\n') + 1
	if off == 0 {
		log.Error("QueryLog: Can't find a new line: %s", r.files[r.ifile])
		return false
	}

	r.fpos += uint64(off)
	log.Debug("QueryLog: seek: %x (+%d)", r.fpos, off)
	_, err = r.f.Seek(int64(r.fpos), io.SeekStart)
	if err != nil {
		log.Error("QueryLog: file.Seek: %s: %s", r.files[r.ifile], err)
		return false
	}
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
		r.searching = true
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

// Next - return the next entry or nil if reading is finished
func (r *Reader) Next() *logEntry { // nolint
	var err error
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

		// open decoder if needed
		if r.jd == nil {
			r.jd = json.NewDecoder(r.f)
		}

		// check if there's data
		if !r.jd.More() {
			r.jd = nil
			return nil
		}

		// read data
		var entry logEntry
		err = r.jd.Decode(&entry)
		if err != nil {
			log.Error("QueryLog: Failed to decode: %s", err)
			r.jd = nil
			return nil
		}

		t := entry.Time.UnixNano()
		if r.searching {
			r.jd = nil

			rr := r.fseeker.seekBinary(uint64(t))
			r.fpos = r.fseeker.pos
			if rr < 0 {
				log.Error("QueryLog: File seek error: can't find the target entry: %s", r.files[r.ifile])
				return nil
			} else if rr == 0 {
				// We found the target entry.
				// We'll start reading the previous chunk of data.
				r.searching = false

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

		if t < r.validFrom {
			continue
		}
		if t >= r.olderThan {
			return nil
		}

		r.count++
		return &entry
	}
}

// Total returns the total number of items
func (r *Reader) Total() int {
	return 0
}
