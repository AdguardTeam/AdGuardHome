package stats

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/dnsproxy/proxy"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"go.etcd.io/bbolt"
)

const (
	// maxDomains is the max number of top domains to return.
	maxDomains = 100

	// maxClients is the max number of top clients to return.
	maxClients = 100

	// maxUpstreams is the max number of top upstreams to return.
	maxUpstreams = 100
)

// UnitIDGenFunc is the signature of a function that generates a unique ID for
// the statistics unit.
type UnitIDGenFunc func() (id uint32)

// Supported values of [StatsResp.TimeUnits].
const (
	timeUnitsHours = "hours"
	timeUnitsDays  = "days"
)

// Result is the resulting code of processing the DNS request.
type Result int

// Supported Result values.
//
// TODO(e.burkov):  Think about better naming.
const (
	RNotFiltered Result = iota + 1
	RFiltered
	RSafeBrowsing
	RSafeSearch
	RParental

	resultLast = RParental + 1
)

// Entry is a statistics data entry.
type Entry struct {
	// Clients is the client's primary ID.
	//
	// TODO(a.garipov): Make this a {net.IP, string} enum?
	Client string

	// Domain is the domain name requested.
	Domain string

	// UpstreamStats contains the DNS query statistics for both the upstream and
	// fallback DNS servers.  Don't modify items in the slice.
	UpstreamStats []*proxy.UpstreamStatistics

	// Result is the result of processing the request.
	Result Result

	// ProcessingTime is the duration of the request processing from the start
	// of the request including timeouts.
	ProcessingTime time.Duration
}

// validate returns an error if entry is not valid.
func (e *Entry) validate() (err error) {
	switch {
	case e.Result == 0:
		return errors.Error("result code is not set")
	case e.Result >= resultLast:
		return fmt.Errorf("unknown result code %d", e.Result)
	case e.Domain == "":
		return errors.Error("domain is empty")
	case e.Client == "":
		return errors.Error("client is empty")
	default:
		return nil
	}
}

// UpstreamEntry is a statistics data entry for a single successful DNS exchange
// with an upstream server.
//
// Note that such an exchange isn't necessarily bound to a client's request, see
// [Interface.UpdateUpstream].
type UpstreamEntry struct {
	// Address is the address of the upstream DNS server that responded.
	Address string

	// Domain is the domain name that has been requested from the upstream.
	Domain string

	// QueryDuration is the duration of the exchange.
	QueryDuration time.Duration
}

// validate returns an error if the upstream entry is not valid.
func (e *UpstreamEntry) validate() (err error) {
	switch {
	case e.Address == "":
		return errors.Error("upstream address is empty")
	case e.Domain == "":
		return errors.Error("domain is empty")
	default:
		return nil
	}
}

// unit collects the statistics data for a specific period of time.
type unit struct {
	// domains stores the number of requests for each domain.
	domains map[string]uint64

	// blockedDomains stores the number of requests for each domain that has
	// been blocked.
	blockedDomains map[string]uint64

	// clients stores the number of requests from each client.
	clients map[string]uint64

	// upstreamsResponses stores the number of responses from each upstream.
	upstreamsResponses map[string]uint64

	// upstreamsTimeSum stores the sum of durations of successful queries in
	// microseconds to each upstream.
	upstreamsTimeSum map[string]uint64

	// nResult stores the number of requests grouped by it's result.
	nResult []uint64

	// id is the unique unit's identifier.  It's set to an absolute hour number
	// since the beginning of UNIX time by the default ID generating function.
	//
	// Must not be rewritten after creating to be accessed concurrently without
	// using mu.
	id uint32

	// nTotal stores the total number of requests.
	nTotal uint64

	// upstreamsResponsesTotal is the number of responses from all the upstream
	// servers, and upstreamsTimeSumTotal is the sum of their durations in
	// microseconds.
	//
	// They are kept apart from upstreamsResponses and upstreamsTimeSum because
	// those are truncated to the top [maxUpstreams] entries independently of
	// each other when serialized, so the two need not describe the same set of
	// upstreams and cannot be summed against one another.
	upstreamsResponsesTotal uint64
	upstreamsTimeSumTotal   uint64

	// timeSum stores the sum of processing time in microseconds of each request
	// written by the unit.
	timeSum uint64
}

// newUnit allocates the new *unit.
func newUnit(id uint32) (u *unit) {
	return &unit{
		domains:            map[string]uint64{},
		blockedDomains:     map[string]uint64{},
		clients:            map[string]uint64{},
		upstreamsResponses: map[string]uint64{},
		upstreamsTimeSum:   map[string]uint64{},
		nResult:            make([]uint64, resultLast),
		id:                 id,
	}
}

// countPair is a single name-number pair for deserializing statistics data into
// the database.
type countPair struct {
	Name  string
	Count uint64
}

// unitDB is the structure for serializing statistics data into the database.
//
// NOTE: Do not change the names or types of fields, as this structure is used
// for GOB encoding.
type unitDB struct {
	// NResult is the number of requests by the result's kind.
	NResult []uint64

	// Domains is the number of requests for each domain name.
	Domains []countPair

	// BlockedDomains is the number of requests blocked for each domain name.
	BlockedDomains []countPair

	// Clients is the number of requests from each client.
	Clients []countPair

	// UpstreamsResponses is the number of responses from each upstream.
	UpstreamsResponses []countPair

	// UpstreamsTimeSum is the sum of processing time in microseconds of
	// responses from each upstream.
	UpstreamsTimeSum []countPair

	// NTotal is the total number of requests.
	NTotal uint64

	// UpstreamsResponsesTotal is the number of responses from all the upstream
	// servers, and UpstreamsTimeSumTotal is the sum of their durations in
	// microseconds, both before UpstreamsResponses and UpstreamsTimeSum are
	// truncated.  They are absent from the records written by the versions that
	// predate them, in which case they decode as zero, see
	// [avgUpstreamResponseTime].
	UpstreamsResponsesTotal uint64
	UpstreamsTimeSumTotal   uint64

	// TimeAvg is the average of processing times in microseconds of all the
	// requests in the unit.
	TimeAvg uint32
}

// newUnitID is the default UnitIDGenFunc that generates the unique id hourly.
func newUnitID() (id uint32) {
	const secsInHour = int64(time.Hour / time.Second)

	return uint32(time.Now().Unix() / secsInHour)
}

func finishTxn(tx *bbolt.Tx, commit bool) (err error) {
	if commit {
		err = errors.Annotate(tx.Commit(), "committing: %w")
	} else {
		err = errors.Annotate(tx.Rollback(), "rolling back: %w")
	}

	return err
}

// bucketNameLen is the length of a bucket, a 64-bit unsigned integer.
//
// TODO(a.garipov): Find out why a 64-bit integer is used when IDs seem to
// always be 32 bits.
const bucketNameLen = 8

// idToUnitName converts a numerical ID into a database unit name.
func idToUnitName(id uint32) (name []byte) {
	n := [bucketNameLen]byte{}
	binary.BigEndian.PutUint64(n[:], uint64(id))

	return n[:]
}

// unitNameToID converts a database unit name into a numerical ID.  ok is false
// if name is not a valid database unit name.
func unitNameToID(name []byte) (id uint32, ok bool) {
	if len(name) < bucketNameLen {
		return 0, false
	}

	return uint32(binary.BigEndian.Uint64(name)), true
}

// compareCount used to sort countPair by Count in descending order.
func (a countPair) compareCount(b countPair) (res int) {
	switch x, y := a.Count, b.Count; {
	case x > y:
		return -1
	case x < y:
		return +1
	default:
		return 0
	}
}

func convertMapToSlice(m map[string]uint64, maxVal int) (s []countPair) {
	s = make([]countPair, 0, len(m))
	for k, v := range m {
		s = append(s, countPair{Name: k, Count: v})
	}

	slices.SortFunc(s, countPair.compareCount)

	return s[:min(maxVal, len(s))]
}

func convertSliceToMap(a []countPair) (m map[string]uint64) {
	m = map[string]uint64{}
	for _, it := range a {
		m[it.Name] = it.Count
	}

	return m
}

// serialize converts u to the *unitDB.  It's safe for concurrent use.  u must
// not be nil.
func (u *unit) serialize() (udb *unitDB) {
	var timeAvg uint32 = 0
	if u.nTotal != 0 {
		timeAvg = uint32(u.timeSum / u.nTotal)
	}

	return &unitDB{
		NTotal:             u.nTotal,
		NResult:            append([]uint64{}, u.nResult...),
		Domains:            convertMapToSlice(u.domains, maxDomains),
		BlockedDomains:     convertMapToSlice(u.blockedDomains, maxDomains),
		Clients:            convertMapToSlice(u.clients, maxClients),
		UpstreamsResponses: convertMapToSlice(u.upstreamsResponses, maxUpstreams),
		UpstreamsTimeSum:   convertMapToSlice(u.upstreamsTimeSum, maxUpstreams),
		TimeAvg:            timeAvg,

		UpstreamsResponsesTotal: u.upstreamsResponsesTotal,
		UpstreamsTimeSumTotal:   u.upstreamsTimeSumTotal,
	}
}

// loadUnitFromDB loads unit by id from the database.
func (s *StatsCtx) loadUnitFromDB(tx *bbolt.Tx, id uint32) (udb *unitDB) {
	bkt := tx.Bucket(idToUnitName(id))
	if bkt == nil {
		return nil
	}

	s.logger.Debug("loading unit", "id", id)

	var buf bytes.Buffer
	buf.Write(bkt.Get([]byte{0}))
	udb = &unitDB{}

	err := gob.NewDecoder(&buf).Decode(udb)
	if err != nil {
		s.logger.Error("gob decode", slogutil.KeyError, err)

		return nil
	}

	return udb
}

// deserialize assigns the appropriate values from udb to u.  u must not be nil.
// It's safe for concurrent use.
func (u *unit) deserialize(udb *unitDB) {
	if udb == nil {
		return
	}

	u.nTotal = udb.NTotal
	u.nResult = make([]uint64, resultLast)
	copy(u.nResult, udb.NResult)
	u.domains = convertSliceToMap(udb.Domains)
	u.blockedDomains = convertSliceToMap(udb.BlockedDomains)
	u.clients = convertSliceToMap(udb.Clients)
	u.upstreamsResponses = convertSliceToMap(udb.UpstreamsResponses)
	u.upstreamsTimeSum = convertSliceToMap(udb.UpstreamsTimeSum)
	u.upstreamsTimeSumTotal, u.upstreamsResponsesTotal = upstreamTotals(udb)
	u.timeSum = uint64(udb.TimeAvg) * udb.NTotal
}

// add adds new data to u.  It's safe for concurrent use.
func (u *unit) add(e *Entry) {
	u.nResult[e.Result]++
	if e.Result == RNotFiltered {
		u.domains[e.Domain]++
	} else {
		u.blockedDomains[e.Domain]++
	}

	u.clients[e.Client]++
	pt := uint64(e.ProcessingTime.Microseconds())
	u.timeSum += pt
	u.nTotal++

	for _, s := range e.UpstreamStats {
		if s.IsCached || s.Error != nil {
			continue
		}

		addr := s.Address
		dur := uint64(s.QueryDuration.Microseconds())

		u.upstreamsResponses[addr]++
		u.upstreamsTimeSum[addr] += dur
		u.upstreamsResponsesTotal++
		u.upstreamsTimeSumTotal += dur
	}
}

// addUpstream adds the data about a single upstream exchange to u.  It's safe
// for concurrent use.
func (u *unit) addUpstream(e *UpstreamEntry) {
	dur := uint64(e.QueryDuration.Microseconds())

	u.upstreamsResponses[e.Address]++
	u.upstreamsTimeSum[e.Address] += dur
	u.upstreamsResponsesTotal++
	u.upstreamsTimeSumTotal += dur
}

// flushUnitToDB puts udb to the database at id.
func (s *StatsCtx) flushUnitToDB(udb *unitDB, tx *bbolt.Tx, id uint32) (err error) {
	s.logger.Debug("flushing unit", "id", id, "req_num", udb.NTotal)

	bkt, err := tx.CreateBucketIfNotExists(idToUnitName(id))
	if err != nil {
		return fmt.Errorf("creating bucket: %w", err)
	}

	buf := &bytes.Buffer{}
	err = gob.NewEncoder(buf).Encode(udb)
	if err != nil {
		return fmt.Errorf("encoding unit: %w", err)
	}

	err = bkt.Put([]byte{0}, buf.Bytes())
	if err != nil {
		return fmt.Errorf("putting unit to database: %w", err)
	}

	return nil
}

func convertTopSlice(a []countPair) (m []map[string]uint64) {
	m = make([]map[string]uint64, 0, len(a))
	for _, it := range a {
		m = append(m, map[string]uint64{it.Name: it.Count})
	}

	return m
}

// pairsGetter is a signature for topsCollector argument.
type pairsGetter func(u *unitDB) (pairs []countPair)

// topsCollector collects statistics about highest values from the given *unitDB
// slice using pg to retrieve data.
func topsCollector(units []*unitDB, max int, ignored *aghnet.IgnoreEngine, pg pairsGetter) []map[string]uint64 {
	m := map[string]uint64{}
	for _, u := range units {
		for _, cp := range pg(u) {
			if !ignored.Has(cp.Name) {
				m[cp.Name] += cp.Count
			}
		}
	}
	a2 := convertMapToSlice(m, max)

	return convertTopSlice(a2)
}

// getData returns the statistics data using the following algorithm:
//
//  1. Prepare a slice of N units, where N is the value of "limit" configuration
//     setting.  Load data for the most recent units from the file.  If a unit
//     with required ID doesn't exist, just add an empty unit.  Get data for the
//     current unit.
//
//  2. Process data from the units and prepare an output map object, including
//     per time unit counters (DNS queries per time-unit, blocked queries per
//     time unit, etc.).  If the time unit is hour, just add values from each
//     unit to the slice; otherwise, the time unit is day, so aggregate per-hour
//     data into days.
//
//     To get the top counters (queries per domain, queries per blocked domain,
//     etc.), first sum up data for all units into a single map.  Then,  get the
//     pairs with the highest numbers.
//
//     The total counters (DNS queries, blocked, etc.) are just the sum of data
//     for all units.
func (s *StatsCtx) getData(limit uint32) (resp *StatsResp, ok bool) {
	if limit == 0 {
		return &StatsResp{
			TimeUnits: "days",

			TopBlocked:            []topAddrs{},
			TopClients:            []topAddrs{},
			TopQueried:            []topAddrs{},
			TopUpstreamsResponses: []topAddrs{},
			TopUpstreamsAvgTime:   []topAddrsFloat{},

			BlockedFiltering:     []uint64{},
			DNSQueries:           []uint64{},
			ReplacedParental:     []uint64{},
			ReplacedSafebrowsing: []uint64{},
		}, true
	}

	units, curID := s.loadUnits(limit)
	if units == nil {
		return &StatsResp{}, false
	}

	return s.dataFromUnits(units, curID), true
}

// dataFromUnits collects and returns the statistics data.
func (s *StatsCtx) dataFromUnits(units []*unitDB, curID uint32) (resp *StatsResp) {
	topUpstreamsResponses, topUpstreamsAvgTime := topUpstreamsPairs(units)

	resp = &StatsResp{
		TopQueried:            topsCollector(units, maxDomains, s.ignored, func(u *unitDB) (pairs []countPair) { return u.Domains }),
		TopBlocked:            topsCollector(units, maxDomains, s.ignored, func(u *unitDB) (pairs []countPair) { return u.BlockedDomains }),
		TopUpstreamsResponses: topUpstreamsResponses,
		TopUpstreamsAvgTime:   topUpstreamsAvgTime,
		TopClients:            topsCollector(units, maxClients, nil, topClientPairs(s)),

		AvgUpstreamResponseTime: avgUpstreamResponseTime(units),
	}

	s.fillCollectedStats(resp, units, curID)

	// Total counters:
	sum := unitDB{
		NResult: make([]uint64, resultLast),
	}

	// timeSum is the total processing time of all the requests within units, in
	// microseconds, and timeN is their number.
	//
	// NOTE:  [unitDB.TimeAvg] is the mean processing time within a single unit,
	// so it must be weighted by the number of requests of that unit.  Taking
	// the mean of the means instead would give an hour with a handful of
	// requests the same weight as an hour with tens of thousands of them, and
	// would also disagree with the upstream response times, which are averaged
	// over the whole period, see [topUpstreamsPairs].
	var timeSum, timeN uint64
	for _, u := range units {
		sum.NTotal += u.NTotal
		timeSum += uint64(u.TimeAvg) * u.NTotal
		timeN += u.NTotal
		sum.NResult[RFiltered] += u.NResult[RFiltered]
		sum.NResult[RSafeBrowsing] += u.NResult[RSafeBrowsing]
		sum.NResult[RSafeSearch] += u.NResult[RSafeSearch]
		sum.NResult[RParental] += u.NResult[RParental]
	}

	resp.NumDNSQueries = sum.NTotal
	resp.NumBlockedFiltering = sum.NResult[RFiltered]
	resp.NumReplacedSafebrowsing = sum.NResult[RSafeBrowsing]
	resp.NumReplacedSafesearch = sum.NResult[RSafeSearch]
	resp.NumReplacedParental = sum.NResult[RParental]

	if timeN != 0 {
		resp.AvgProcessingTime = microsecondsToSeconds(float64(timeSum) / float64(timeN))
	}

	return resp
}

// fillCollectedStats fills data with collected statistics.
func (s *StatsCtx) fillCollectedStats(data *StatsResp, units []*unitDB, curID uint32) {
	size := len(units)
	data.TimeUnits = timeUnitsHours

	daysCount := size / 24
	if daysCount > 7 {
		size = daysCount
		data.TimeUnits = timeUnitsDays
	}

	data.DNSQueries = make([]uint64, size)
	data.BlockedFiltering = make([]uint64, size)
	data.ReplacedSafebrowsing = make([]uint64, size)
	data.ReplacedParental = make([]uint64, size)

	if data.TimeUnits == timeUnitsDays {
		s.fillCollectedStatsDaily(data, units, curID, size)

		return
	}

	for i, u := range units {
		data.DNSQueries[i] += u.NTotal
		data.BlockedFiltering[i] += u.NResult[RFiltered]
		data.ReplacedSafebrowsing[i] += u.NResult[RSafeBrowsing]
		data.ReplacedParental[i] += u.NResult[RParental]
	}
}

// fillCollectedStatsDaily fills data with collected daily statistics.  units
// must contain data for the count of days.
//
// TODO(s.chzhen):  Improve collection of statistics for frontend.  Dashboard
// cards should contain statistics for the whole interval without rounding to
// days.
func (s *StatsCtx) fillCollectedStatsDaily(
	data *StatsResp,
	units []*unitDB,
	curHour uint32,
	days int,
) {
	// Per time unit counters: 720 hours may span 31 days, so we skip data for
	// the first hours in this case.  align_ceil(24)
	hours := countHours(curHour, days)
	units = units[len(units)-hours:]

	for i, u := range units {
		day := i / 24

		data.DNSQueries[day] += u.NTotal
		data.BlockedFiltering[day] += u.NResult[RFiltered]
		data.ReplacedSafebrowsing[day] += u.NResult[RSafeBrowsing]
		data.ReplacedParental[day] += u.NResult[RParental]
	}
}

// countHours returns the number of hours in the last days.
func countHours(curHour uint32, days int) (n int) {
	hoursInCurDay := int(curHour % 24)
	if hoursInCurDay == 0 {
		hoursInCurDay = 24
	}

	hoursInRestDays := (days - 1) * 24

	return hoursInRestDays + hoursInCurDay
}

func topClientPairs(s *StatsCtx) (pg pairsGetter) {
	return func(u *unitDB) (clients []countPair) {
		for _, c := range u.Clients {
			if c.Name != "" && !s.shouldCountClient([]string{c.Name}) {
				continue
			}

			clients = append(clients, c)
		}

		return clients
	}
}

// topUpstreamsPairs returns sorted lists of number of total responses and the
// average of processing time for each upstream.
func topUpstreamsPairs(
	units []*unitDB,
) (topUpstreamsResponses []topAddrs, topUpstreamsAvgTime []topAddrsFloat) {
	upstreamsResponses := topAddrs{}
	upstreamsTimeSum := topAddrsFloat{}

	for _, u := range units {
		for _, cp := range u.UpstreamsResponses {
			upstreamsResponses[cp.Name] += cp.Count
		}

		for _, cp := range u.UpstreamsTimeSum {
			upstreamsTimeSum[cp.Name] += float64(cp.Count)
		}
	}

	upstreamsAvgTime := topAddrsFloat{}

	for u, n := range upstreamsResponses {
		total := upstreamsTimeSum[u]

		if total != 0 {
			upstreamsAvgTime[u] = microsecondsToSeconds(total / float64(n))
		}
	}

	upstreamsPairs := convertMapToSlice(upstreamsResponses, maxUpstreams)
	topUpstreamsResponses = convertTopSlice(upstreamsPairs)

	return topUpstreamsResponses, prepareTopUpstreamsAvgTime(upstreamsAvgTime)
}

// avgUpstreamResponseTime returns the average time that the upstream DNS
// servers took to respond within units, in seconds.
//
// It is averaged over the responses of every upstream server, as opposed to
// [StatsResp.AvgProcessingTime], which is averaged over every request that
// AdGuard Home has processed, including the ones answered from the cache or
// blocked by a filter.  Those take almost no time, so the two values are
// expected to differ, often by an order of magnitude.
func avgUpstreamResponseTime(units []*unitDB) (avg float64) {
	var timeSum, respNum uint64
	for _, u := range units {
		t, n := upstreamTotals(u)
		timeSum += t
		respNum += n
	}

	if respNum == 0 {
		return 0
	}

	return microsecondsToSeconds(float64(timeSum) / float64(respNum))
}

// upstreamTotals returns the total duration of the responses from all the
// upstream servers of udb, in microseconds, and their number.  udb must not be
// nil.
//
// [unitDB.UpstreamsResponses] and [unitDB.UpstreamsTimeSum] are each truncated
// to the top [maxUpstreams] entries independently of each other, so on a unit
// that has seen more upstreams than that they need not describe the same set,
// and summing one against the other gives a ratio of two different populations.
// The exact totals are therefore stored alongside them.
//
// Records written before those totals existed decode them as zero, and the
// bounded slices are all such a record has; summing them is what this used to
// do and remains the best available answer for it.
func upstreamTotals(udb *unitDB) (timeSum, respNum uint64) {
	if udb.UpstreamsResponsesTotal != 0 || udb.UpstreamsTimeSumTotal != 0 {
		return udb.UpstreamsTimeSumTotal, udb.UpstreamsResponsesTotal
	}

	for _, cp := range udb.UpstreamsTimeSum {
		timeSum += cp.Count
	}

	for _, cp := range udb.UpstreamsResponses {
		respNum += cp.Count
	}

	return timeSum, respNum
}

// microsecondsToSeconds converts microseconds to seconds.
//
// NOTE:  Frontend expects time duration in seconds as floating-point number
// with double precision.
func microsecondsToSeconds(n float64) (r float64) {
	const micro = 1e-6

	return n * micro
}

// prepareTopUpstreamsAvgTime returns sorted list of average processing times
// of the DNS requests from each upstream.
func prepareTopUpstreamsAvgTime(
	upstreamsAvgTime topAddrsFloat,
) (topUpstreamsAvgTime []topAddrsFloat) {
	keys := slices.SortedStableFunc(maps.Keys(upstreamsAvgTime), func(a, b string) (res int) {
		switch x, y := upstreamsAvgTime[a], upstreamsAvgTime[b]; {
		case x > y:
			return -1
		case x < y:
			return +1
		default:
			return 0
		}
	})

	topUpstreamsAvgTime = make([]topAddrsFloat, 0, len(upstreamsAvgTime))
	for _, k := range keys {
		topUpstreamsAvgTime = append(topUpstreamsAvgTime, topAddrsFloat{k: upstreamsAvgTime[k]})
	}

	return topUpstreamsAvgTime
}
