package jsonfile

import (
	"io"
	"sort"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/querylog/logs"
	"github.com/AdguardTeam/golibs/log"
)

// client finds the client info, if any, by its ClientID and IP address,
// optionally checking the provided cache.  It will use the IP address
// regardless of if the IP anonymization is enabled now, because the
// anonymization could have been disabled in the past, and client will try to
// find those records as well.
func (l *queryLog) client(clientID, ip string, cache clientCache) (c *logs.Client, err error) {
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
func (l *queryLog) searchMemory(params *logs.SearchParams, cache clientCache) (entries []*logEntry, total int) {
	l.bufferLock.Lock()
	defer l.bufferLock.Unlock()

	// Go through the buffer in the reverse order, from newer to older.
	var err error
	for i := len(l.buffer) - 1; i >= 0; i-- {
		e := l.buffer[i]

		e.client, err = l.client(e.ClientID, e.IP.String(), cache)
		if err != nil {
			msg := "querylog: enriching memory record at time %s" +
				" for client %q (clientid %q): %s"
			log.Error(msg, e.Time, e.IP, e.ClientID, err)

			// Go on and try to match anyway.
		}

		if matchParam(params, e) {
			entries = append(entries, e)
		}
	}

	return entries, len(l.buffer)
}

// search - searches log entries in the query log using specified parameters
// returns the list of entries found + time of the oldest entry
func (l *queryLog) search(params *logs.SearchParams) (entries []*logEntry, oldest time.Time) {
	now := time.Now()

	if params.Limit == 0 {
		return []*logEntry{}, time.Time{}
	}

	cache := clientCache{}
	fileEntries, oldest, total := l.searchFiles(params, cache)
	memoryEntries, bufLen := l.searchMemory(params, cache)
	total += bufLen

	totalLimit := params.Offset + params.Limit

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
	sort.SliceStable(entries, func(i, j int) (less bool) {
		return entries[i].Time.After(entries[j].Time)
	})

	if params.Offset > 0 {
		if len(entries) > params.Offset {
			entries = entries[params.Offset:]
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
		params.OlderThan,
		time.Since(now),
	)

	return entries, oldest
}

// searchFiles looks up log records from all log files.  It optionally uses the
// client cache, if provided.  searchFiles does not scan more than
// maxFileScanEntries so callers may need to call it several times to get all
// results.  oldset and total are the time of the oldest processed entry and the
// total number of processed entries, including discarded ones, correspondingly.
func (l *queryLog) searchFiles(
	params *logs.SearchParams,
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

	if params.OlderThan.IsZero() {
		err = r.SeekStart()
	} else {
		err = r.seekTS(params.OlderThan.UnixNano())
		if err == nil {
			// Read to the next record, because we only need the one
			// that goes after it.
			_, err = r.ReadNext()
		}
	}

	if err != nil {
		log.Debug("querylog: cannot seek to %s: %s", params.OlderThan, err)

		return entries, oldest, 0
	}

	totalLimit := params.Offset + params.Limit
	oldestNano := int64(0)

	// By default, we do not scan more than maxFileScanEntries at once.
	// The idea is to make search calls faster so that the UI could handle
	// it and show something quicker.  This behavior can be overridden if
	// maxFileScanEntries is set to 0.
	for total < params.MaxFileScanEntries || params.MaxFileScanEntries <= 0 {
		var e *logEntry
		var ts int64

		e, ts, err = l.readNextEntry(r, params, cache)
		if err != nil {
			if err == io.EOF {
				oldestNano = 0

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
	client func(clientID, ip string, cache clientCache) (c *logs.Client, err error)
	cache  clientCache
}

// findClient is a method that can be used as a quickMatchClientFinder.
func (f quickMatchClientFinder) findClient(clientID, ip string) (c *logs.Client) {
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
// criteria.  It optionally uses the client cache, if provided.  e is nil if the
// entry doesn't match the search criteria.  ts is the timestamp of the
// processed entry.
func (l *queryLog) readNextEntry(
	r *QLogReader,
	params *logs.SearchParams,
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

	if !quickMatchParam(params, line, clientFinder.findClient) {
		ts = readQLogTimestamp(line)

		return nil, ts, nil
	}

	e = &logEntry{}
	decodeLogEntry(e, line)

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

	ts = e.Time.UnixNano()
	if !matchParam(params, e) {
		return nil, ts, nil
	}

	return e, ts, nil
}

// quickMatchClientFunc is a simplified client finder for quick matches.
type quickMatchClientFunc = func(clientID, ip string) (c *logs.Client)

// quickMatch quickly checks if the line matches the given search parameters.
// It returns false if the line doesn't match.  This method is only here for
// optimization purposes.
func quickMatchParam(s *logs.SearchParams, line string, findClient quickMatchClientFunc) (ok bool) {
	for _, c := range s.SearchCriteria {
		if !quickMatchCrit(&c, line, findClient) {
			return false
		}
	}

	return true
}

// match - checks if the logEntry matches the searchParams
func matchParam(s *logs.SearchParams, entry *logEntry) bool {
	if !s.OlderThan.IsZero() && !entry.Time.Before(s.OlderThan) {
		// Ignore entries newer than what was requested
		return false
	}

	for _, c := range s.SearchCriteria {
		if !matchCrit(&c, entry) {
			return false
		}
	}

	return true
}

// quickMatch quickly checks if the line matches the given search criterion.
// It returns false if the like doesn't match.  This method is only here for
// optimization purposes.
func quickMatchCrit(c *logs.SearchCriterion, line string, findClient quickMatchClientFunc) (ok bool) {
	switch c.CriterionType {
	case logs.CtTerm:
		host := readJSONValue(line, `"QH":"`)
		ip := readJSONValue(line, `"IP":"`)
		clientID := readJSONValue(line, `"CID":"`)

		var name string
		if cli := findClient(clientID, ip); cli != nil {
			name = cli.Name
		}

		if c.Strict {
			return logs.CtDomainOrClientCaseStrict(c.Value, c.AsciiVal, clientID, name, host, ip)
		}

		return logs.CtDomainOrClientCaseNonStrict(c.Value, c.AsciiVal, clientID, name, host, ip)
	case logs.CtFilteringStatus:
		// Go on, as we currently don't do quick matches against
		// filtering statuses.
		return true
	default:
		return true
	}
}

// match checks if the log entry matches this search criterion.
func matchCrit(c *logs.SearchCriterion, entry *logEntry) bool {
	switch c.CriterionType {
	case logs.CtTerm:
		return ctDomainOrClientCase(c, entry)
	case logs.CtFilteringStatus:
		return logs.CtFilteringStatusCase(c, entry.Result)
	}

	return false
}

func ctDomainOrClientCase(c *logs.SearchCriterion, e *logEntry) bool {
	clientID := e.ClientID
	host := e.QHost

	var name string
	if e.client != nil {
		name = e.client.Name
	}

	ip := e.IP.String()
	if c.Strict {
		return logs.CtDomainOrClientCaseStrict(c.Value, c.AsciiVal, clientID, name, host, ip)
	}

	return logs.CtDomainOrClientCaseNonStrict(c.Value, c.AsciiVal, clientID, name, host, ip)
}
