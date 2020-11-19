package querylog

import (
	"io"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/util"
	"github.com/AdguardTeam/golibs/log"
)

// search - searches log entries in the query log using specified parameters
// returns the list of entries found + time of the oldest entry
func (l *queryLog) search(params *searchParams) ([]*logEntry, time.Time) {
	now := time.Now()

	if params.limit == 0 {
		return []*logEntry{}, time.Time{}
	}

	// add from file
	fileEntries, oldest, total := l.searchFiles(params)

	// add from memory buffer
	l.bufferLock.Lock()
	total += len(l.buffer)
	memoryEntries := make([]*logEntry, 0)

	// go through the buffer in the reverse order
	// from NEWER to OLDER
	for i := len(l.buffer) - 1; i >= 0; i-- {
		entry := l.buffer[i]
		if !params.match(entry) {
			continue
		}
		memoryEntries = append(memoryEntries, entry)
	}
	l.bufferLock.Unlock()

	// limits
	totalLimit := params.offset + params.limit

	// now let's get a unified collection
	entries := append(memoryEntries, fileEntries...)
	if len(entries) > totalLimit {
		// remove extra records
		entries = entries[:totalLimit]
	}

	if params.offset > 0 {
		if len(entries) > params.offset {
			entries = entries[params.offset:]
		} else {
			entries = make([]*logEntry, 0)
			oldest = time.Time{}
		}
	}

	if len(entries) > 0 && len(entries) <= totalLimit {
		// Update oldest after merging in the memory buffer.
		oldest = entries[len(entries)-1].Time
	}

	log.Debug("QueryLog: prepared data (%d/%d) older than %s in %s",
		len(entries), total, params.olderThan, time.Since(now))

	return entries, oldest
}

// searchFiles reads log entries from all log files and applies the specified search criteria.
// IMPORTANT: this method does not scan more than "maxSearchEntries" so you
// may need to call it many times.
//
// it returns:
// * an array of log entries that we have read
// * time of the oldest processed entry (even if it was discarded)
// * total number of processed entries (including discarded).
func (l *queryLog) searchFiles(params *searchParams) ([]*logEntry, time.Time, int) {
	entries := make([]*logEntry, 0)
	oldest := time.Time{}

	r, err := l.openReader()
	if err != nil {
		log.Error("Failed to open qlog reader: %v", err)
		return entries, oldest, 0
	}
	defer r.Close()

	if params.olderThan.IsZero() {
		err = r.SeekStart()
	} else {
		err = r.Seek(params.olderThan.UnixNano())
		if err == nil {
			// Read to the next record right away
			// The one that was specified in the "oldest" param is not needed,
			// we need only the one next to it
			_, err = r.ReadNext()
		}
	}

	if err != nil {
		log.Debug("Cannot Seek() to %v: %v", params.olderThan, err)
		return entries, oldest, 0
	}

	totalLimit := params.offset + params.limit
	total := 0
	oldestNano := int64(0)
	// By default, we do not scan more than "maxFileScanEntries" at once
	// The idea is to make search calls faster so that the UI could handle it and show something
	// This behavior can be overridden if "maxFileScanEntries" is set to 0
	for total < params.maxFileScanEntries || params.maxFileScanEntries <= 0 {
		entry, ts, err := l.readNextEntry(r, params)

		if err == io.EOF {
			// there's nothing to read anymore
			break
		}

		oldestNano = ts
		total++

		if entry != nil {
			entries = append(entries, entry)
			if len(entries) == totalLimit {
				// Do not read more than "totalLimit" records at once
				break
			}
		}
	}

	if oldestNano != 0 {
		oldest = time.Unix(0, oldestNano)
	}
	return entries, oldest, total
}

// readNextEntry - reads the next log entry and checks if it matches the search criteria (getDataParams)
//
// returns:
// * log entry that matches search criteria or null if it was discarded (or if there's nothing to read)
// * timestamp of the processed log entry
// * error if we can't read anymore
func (l *queryLog) readNextEntry(r *QLogReader, params *searchParams) (*logEntry, int64, error) {
	line, err := r.ReadNext()
	if err != nil {
		return nil, 0, err
	}

	// Read the log record timestamp right away
	timestamp := readQLogTimestamp(line)

	// Quick check without deserializing log entry
	if !params.quickMatch(line) {
		return nil, timestamp, nil
	}

	entry := logEntry{}
	decodeLogEntry(&entry, line)

	// Full check of the deserialized log entry
	if !params.match(&entry) {
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
