package querylog

import (
	"io"
	"sort"
	"time"

	"github.com/AdguardTeam/golibs/log"
)

// client finds the client info, if any, by its client ID and IP address,
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
func (l *queryLog) searchMemory(params *searchParams, cache clientCache) (entries []*logEntry, total int) {
	l.bufferLock.Lock()
	defer l.bufferLock.Unlock()

	// Go through the buffer in the reverse order, from newer to older.
	var err error
	for i := len(l.buffer) - 1; i >= 0; i-- {
		e := l.buffer[i]

		e.client, err = l.client(e.ClientID, e.IP.String(), cache)
		if err != nil {
			msg := "querylog: enriching memory record at time %s" +
				" for client %q (client id %q): %s"
			log.Error(msg, e.Time, e.IP, e.ClientID, err)

			// Go on and try to match anyway.
		}

		if params.match(e) {
			entries = append(entries, e)
		}
	}

	return entries, len(l.buffer)
}

// search - searches log entries in the query log using specified parameters
// returns the list of entries found + time of the oldest entry
func (l *queryLog) search(params *searchParams) ([]*logEntry, time.Time) {
	now := time.Now()

	if params.limit == 0 {
		return []*logEntry{}, time.Time{}
	}

	cache := clientCache{}
	fileEntries, oldest, total := l.searchFiles(params, cache)
	memoryEntries, bufLen := l.searchMemory(params, cache)
	total += bufLen

	totalLimit := params.offset + params.limit

	// now let's get a unified collection
	entries := append(memoryEntries, fileEntries...)
	if len(entries) > totalLimit {
		// remove extra records
		entries = entries[:totalLimit]
	}

	// Resort entries on start time to partially mitigate query log looking
	// weird on the frontend.
	//
	// See https://github.com/AdguardTeam/AdGuardHome/issues/2293.
	sort.SliceStable(entries, func(i, j int) (less bool) {
		return entries[i].Time.After(entries[j].Time)
	})

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

// searchFiles looks up log records from all log files.  It optionally uses the
// client cache, if provided.  searchFiles does not scan more than
// maxFileScanEntries so callers may need to call it several times to get all
// results.  oldset and total are the time of the oldest processed entry and the
// total number of processed entries, including discarded ones, correspondingly.
func (l *queryLog) searchFiles(
	params *searchParams,
	cache clientCache,
) (entries []*logEntry, oldest time.Time, total int) {
	files := []string{
		l.logFile + ".1",
		l.logFile,
	}

	r, err := NewQLogReader(files)
	if err != nil {
		log.Error("querylog: failed to open qlog reader: %s", err)

		return entries, oldest, 0
	}
	defer func() {
		derr := r.Close()
		if derr != nil {
			log.Error("querylog: closing file: %s", err)
		}
	}()

	if params.olderThan.IsZero() {
		err = r.SeekStart()
	} else {
		err = r.seekTS(params.olderThan.UnixNano())
		if err == nil {
			// Read to the next record, because we only need the one
			// that goes after it.
			_, err = r.ReadNext()
		}
	}

	if err != nil {
		log.Debug("querylog: cannot seek to %s: %s", params.olderThan, err)

		return entries, oldest, 0
	}

	totalLimit := params.offset + params.limit
	oldestNano := int64(0)

	// By default, we do not scan more than maxFileScanEntries at once.
	// The idea is to make search calls faster so that the UI could handle
	// it and show something quicker.  This behavior can be overridden if
	// maxFileScanEntries is set to 0.
	for total < params.maxFileScanEntries || params.maxFileScanEntries <= 0 {
		var e *logEntry
		var ts int64

		e, ts, err = l.readNextEntry(r, params, cache)
		if err != nil {
			if err == io.EOF {
				break
			}

			log.Error("querylog: reading next entry: %s", err)
		}

		oldestNano = ts
		total++

		if e != nil {
			entries = append(entries, e)
			if len(entries) == totalLimit {
				break
			}
		}
	}

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
		log.Error("querylog: enriching file record for quick search:"+
			" for client %q (client id %q): %s",
			ip,
			clientID,
			err,
		)
	}

	return c
}

// readNextEntry reads the next log entry and checks if it matches the search
// criteria.  It optionally uses the client cache, if provided.  e is nil if the
// entry doesn't match the search criteria.  ts is the timestamp of the
// processed entry.
func (l *queryLog) readNextEntry(
	r *QLogReader,
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

	e.client, err = l.client(e.ClientID, e.IP.String(), cache)
	if err != nil {
		log.Error(
			"querylog: enriching file record at time %s"+
				" for client %q (client id %q): %s",
			e.Time,
			e.IP,
			e.ClientID,
			err,
		)

		// Go on and try to match anyway.
	}

	ts = e.Time.UnixNano()
	if !params.match(e) {
		return nil, ts, nil
	}

	return e, ts, nil
}
