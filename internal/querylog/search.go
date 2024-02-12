package querylog

import (
	"fmt"
	"io"
	"slices"
	"time"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
)

// client finds the client info, if any, by its ClientID and IP address,
// optionally checking the provided cache.  It will use the IP address
// regardless of if the IP anonymization is enabled now, because the
// anonymization could have been disabled in the past, and client will try to
// find those records as well.
func (l *queryLog) client(clientID, ip string, cache clientCache) (c *Client, err error) {
	cck := clientCacheKey{clientID: clientID, ip: ip}

	var ok bool
	if c, ok = cache[cck]; ok {
		return c, nil
	}

	var ids []string
	if clientID != "" {
		ids = append(ids, clientID)
	}

	if ip != "" {
		ids = append(ids, ip)
	}

	c, err = l.findClient(ids)
	if err != nil {
		return nil, err
	}

	// Cache all results, including negative ones, to prevent excessive and
	// expensive client searching.
	cache[cck] = c

	return c, nil
}

// searchMemory looks up log records which are currently in the in-memory
// buffer.  It optionally uses the client cache, if provided.  It also returns
// the total amount of records in the buffer at the moment of searching.
// l.confMu is expected to be locked.
func (l *queryLog) searchMemory(params *searchParams, cache clientCache) (entries []*logEntry, total int) {
	// Check memory size, as the buffer can contain a single log record.  See
	// [newQueryLog].
	if l.conf.MemSize == 0 {
		return nil, 0
	}

	l.bufferLock.Lock()
	defer l.bufferLock.Unlock()

	l.buffer.ReverseRange(func(entry *logEntry) (cont bool) {
		// A shallow clone is enough, since the only thing that this loop
		// modifies is the client field.
		e := entry.shallowClone()

		var err error
		e.client, err = l.client(e.ClientID, e.IP.String(), cache)
		if err != nil {
			msg := "querylog: enriching memory record at time %s" +
				" for client %q (clientid %q): %s"
			log.Error(msg, e.Time, e.IP, e.ClientID, err)

			// Go on and try to match anyway.
		}

		if params.match(e) {
			entries = append(entries, e)
		}

		return true
	})

	return entries, int(l.buffer.Len())
}

// search searches log entries in memory buffer and log file using specified
// parameters and returns the list of entries found and the time of the oldest
// entry.  l.confMu is expected to be locked.
func (l *queryLog) search(params *searchParams) (entries []*logEntry, oldest time.Time) {
	start := time.Now()

	if params.limit == 0 {
		return []*logEntry{}, time.Time{}
	}

	cache := clientCache{}

	memoryEntries, bufLen := l.searchMemory(params, cache)
	log.Debug("querylog: got %d entries from memory", len(memoryEntries))

	fileEntries, oldest, total := l.searchFiles(params, cache)
	log.Debug("querylog: got %d entries from files", len(fileEntries))

	total += bufLen

	totalLimit := params.offset + params.limit

	// now let's get a unified collection
	entries = append(memoryEntries, fileEntries...)
	if len(entries) > totalLimit {
		// remove extra records
		entries = entries[:totalLimit]
	}

	// Resort entries on start time to partially mitigate query log looking
	// weird on the frontend.
	//
	// See https://github.com/AdguardTeam/AdGuardHome/issues/2293.
	slices.SortStableFunc(entries, func(a, b *logEntry) (res int) {
		return -a.Time.Compare(b.Time)
	})

	if params.offset > 0 {
		if len(entries) > params.offset {
			entries = entries[params.offset:]
		} else {
			entries = make([]*logEntry, 0)
			oldest = time.Time{}
		}
	}

	if len(entries) > 0 {
		// Update oldest after merging in the memory buffer.
		oldest = entries[len(entries)-1].Time
	}

	log.Debug(
		"querylog: prepared data (%d/%d) older than %s in %s",
		len(entries),
		total,
		params.olderThan,
		time.Since(start),
	)

	return entries, oldest
}

// seekRecord changes the current position to the next record older than the
// provided parameter.
func (r *qLogReader) seekRecord(olderThan time.Time) (err error) {
	if olderThan.IsZero() {
		return r.SeekStart()
	}

	err = r.seekTS(olderThan.UnixNano())
	if err == nil {
		// Read to the next record, because we only need the one that goes
		// after it.
		_, err = r.ReadNext()
	}

	return err
}

// setQLogReader creates a reader with the specified files and sets the
// position to the next record older than the provided parameter.
func (l *queryLog) setQLogReader(olderThan time.Time) (qr *qLogReader, err error) {
	files := []string{
		l.logFile + ".1",
		l.logFile,
	}

	r, err := newQLogReader(files)
	if err != nil {
		return nil, fmt.Errorf("opening qlog reader: %s", err)
	}

	err = r.seekRecord(olderThan)
	if err != nil {
		defer func() { err = errors.WithDeferred(err, r.Close()) }()
		log.Debug("querylog: cannot seek to %s: %s", olderThan, err)

		return nil, nil
	}

	return r, nil
}

// readEntries reads entries from the reader to totalLimit.  By default, we do
// not scan more than maxFileScanEntries at once.  The idea is to make search
// calls faster so that the UI could handle it and show something quicker.
// This behavior can be overridden if maxFileScanEntries is set to 0.
func (l *queryLog) readEntries(
	r *qLogReader,
	params *searchParams,
	cache clientCache,
	totalLimit int,
) (entries []*logEntry, oldestNano int64, total int) {
	for total < params.maxFileScanEntries || params.maxFileScanEntries <= 0 {
		ent, ts, rErr := l.readNextEntry(r, params, cache)
		if rErr != nil {
			if rErr == io.EOF {
				oldestNano = 0

				break
			}

			log.Error("querylog: reading next entry: %s", rErr)
		}

		oldestNano = ts
		total++

		if ent == nil {
			continue
		}

		entries = append(entries, ent)
		if len(entries) == totalLimit {
			break
		}
	}

	return entries, oldestNano, total
}

// searchFiles looks up log records from all log files.  It optionally uses the
// client cache, if provided.  searchFiles does not scan more than
// maxFileScanEntries so callers may need to call it several times to get all
// the results.  oldest and total are the time of the oldest processed entry
// and the total number of processed entries, including discarded ones,
// correspondingly.
func (l *queryLog) searchFiles(
	params *searchParams,
	cache clientCache,
) (entries []*logEntry, oldest time.Time, total int) {
	r, err := l.setQLogReader(params.olderThan)
	if err != nil {
		log.Error("querylog: %s", err)
	}

	if r == nil {
		return entries, oldest, 0
	}

	defer func() {
		if closeErr := r.Close(); closeErr != nil {
			log.Error("querylog: closing file: %s", closeErr)
		}
	}()

	totalLimit := params.offset + params.limit
	entries, oldestNano, total := l.readEntries(r, params, cache, totalLimit)
	if oldestNano != 0 {
		oldest = time.Unix(0, oldestNano)
	}

	return entries, oldest, total
}

// quickMatchClientFinder is a wrapper around the usual client finding function
// to make it easier to use with quick matches.
type quickMatchClientFinder struct {
	client func(clientID, ip string, cache clientCache) (c *Client, err error)
	cache  clientCache
}

// findClient is a method that can be used as a quickMatchClientFinder.
func (f quickMatchClientFinder) findClient(clientID, ip string) (c *Client) {
	var err error
	c, err = f.client(clientID, ip, f.cache)
	if err != nil {
		log.Error(
			"querylog: enriching file record for quick search: for client %q (clientid %q): %s",
			ip,
			clientID,
			err,
		)
	}

	return c
}

// readNextEntry reads the next log entry and checks if it matches the search
// criteria.  It optionally uses the client cache, if provided.  e is nil if
// the entry doesn't match the search criteria.  ts is the timestamp of the
// processed entry.
func (l *queryLog) readNextEntry(
	r *qLogReader,
	params *searchParams,
	cache clientCache,
) (e *logEntry, ts int64, err error) {
	var line string
	line, err = r.ReadNext()
	if err != nil {
		return nil, 0, err
	}

	clientFinder := quickMatchClientFinder{
		client: l.client,
		cache:  cache,
	}

	if !params.quickMatch(line, clientFinder.findClient) {
		ts = readQLogTimestamp(line)

		return nil, ts, nil
	}

	e = &logEntry{}
	decodeLogEntry(e, line)

	if l.isIgnored(e.QHost) {
		return nil, ts, nil
	}

	e.client, err = l.client(e.ClientID, e.IP.String(), cache)
	if err != nil {
		log.Error(
			"querylog: enriching file record at time %s for client %q (clientid %q): %s",
			e.Time,
			e.IP,
			e.ClientID,
			err,
		)

		// Go on and try to match anyway.
	}

	if e.client != nil && e.client.IgnoreQueryLog {
		return nil, ts, nil
	}

	ts = e.Time.UnixNano()
	if !params.match(e) {
		return nil, ts, nil
	}

	return e, ts, nil
}
