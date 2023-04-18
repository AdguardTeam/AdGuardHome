package stats

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"time"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/stringutil"
	"go.etcd.io/bbolt"
	"golang.org/x/exp/slices"
)

// TODO(a.garipov): Rewrite all of this.  Add proper error handling and
// inspection.  Improve logging.  Decrease complexity.

const (
	// maxDomains is the max number of top domains to return.
	maxDomains = 100
	// maxClients is the max number of top clients to return.
	maxClients = 100
)

// UnitIDGenFunc is the signature of a function that generates a unique ID for
// the statistics unit.
type UnitIDGenFunc func() (id uint32)

// TimeUnit is the unit of measuring time while aggregating the statistics.
type TimeUnit int

// Supported TimeUnit values.
const (
	Hours TimeUnit = iota
	Days
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

	// Result is the result of processing the request.
	Result Result

	// Time is the duration of the request processing in milliseconds.
	Time uint32
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

	// timeSum stores the sum of processing time in milliseconds of each request
	// written by the unit.
	timeSum uint64
}

// newUnit allocates the new *unit.
func newUnit(id uint32) (u *unit) {
	return &unit{
		domains:        map[string]uint64{},
		blockedDomains: map[string]uint64{},
		clients:        map[string]uint64{},
		nResult:        make([]uint64, resultLast),
		id:             id,
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

	// NTotal is the total number of requests.
	NTotal uint64

	// TimeAvg is the average of processing times in milliseconds of all the
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

func convertMapToSlice(m map[string]uint64, max int) (s []countPair) {
	s = make([]countPair, 0, len(m))
	for k, v := range m {
		s = append(s, countPair{Name: k, Count: v})
	}

	slices.SortFunc(s, func(a, b countPair) (sortsBefore bool) {
		return a.Count > b.Count
	})
	if max > len(s) {
		max = len(s)
	}

	return s[:max]
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
		NTotal:         u.nTotal,
		NResult:        append([]uint64{}, u.nResult...),
		Domains:        convertMapToSlice(u.domains, maxDomains),
		BlockedDomains: convertMapToSlice(u.blockedDomains, maxDomains),
		Clients:        convertMapToSlice(u.clients, maxClients),
		TimeAvg:        timeAvg,
	}
}

func loadUnitFromDB(tx *bbolt.Tx, id uint32) (udb *unitDB) {
	bkt := tx.Bucket(idToUnitName(id))
	if bkt == nil {
		return nil
	}

	log.Tracef("Loading unit %d", id)

	var buf bytes.Buffer
	buf.Write(bkt.Get([]byte{0}))
	udb = &unitDB{}

	err := gob.NewDecoder(&buf).Decode(udb)
	if err != nil {
		log.Error("gob Decode: %s", err)

		return nil
	}

	return udb
}

// deserealize assigns the appropriate values from udb to u.  u must not be nil.
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
	u.timeSum = uint64(udb.TimeAvg) * udb.NTotal
}

// add adds new data to u.  It's safe for concurrent use.
func (u *unit) add(res Result, domain, cli string, dur uint64) {
	u.nResult[res]++
	if res == RNotFiltered {
		u.domains[domain]++
	} else {
		u.blockedDomains[domain]++
	}

	u.clients[cli]++
	u.timeSum += dur
	u.nTotal++
}

// flushUnitToDB puts udb to the database at id.
func (udb *unitDB) flushUnitToDB(tx *bbolt.Tx, id uint32) (err error) {
	log.Debug("stats: flushing unit with id %d and total of %d", id, udb.NTotal)

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

// numsGetter is a signature for statsCollector argument.
type numsGetter func(u *unitDB) (num uint64)

// statsCollector collects statisctics for the given *unitDB slice by specified
// timeUnit using ng to retrieve data.
func statsCollector(units []*unitDB, firstID uint32, timeUnit TimeUnit, ng numsGetter) (nums []uint64) {
	if timeUnit == Hours {
		nums = make([]uint64, 0, len(units))
		for _, u := range units {
			nums = append(nums, ng(u))
		}
	} else {
		// Per time unit counters: 720 hours may span 31 days, so we
		// skip data for the first day in this case.
		// align_ceil(24)
		firstDayID := (firstID + 24 - 1) / 24 * 24

		var sum uint64
		id := firstDayID
		nextDayID := firstDayID + 24
		for i := int(firstDayID - firstID); i != len(units); i++ {
			sum += ng(units[i])
			if id == nextDayID {
				nums = append(nums, sum)
				sum = 0
				nextDayID += 24
			}
			id++
		}
		if id <= nextDayID {
			nums = append(nums, sum)
		}
	}
	return nums
}

// pairsGetter is a signature for topsCollector argument.
type pairsGetter func(u *unitDB) (pairs []countPair)

// topsCollector collects statistics about highest values from the given *unitDB
// slice using pg to retrieve data.
func topsCollector(units []*unitDB, max int, ignored *stringutil.Set, pg pairsGetter) []map[string]uint64 {
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
func (s *StatsCtx) getData(limit uint32) (StatsResp, bool) {
	if limit == 0 {
		return StatsResp{
			TimeUnits: "days",

			TopBlocked: []topAddrs{},
			TopClients: []topAddrs{},
			TopQueried: []topAddrs{},

			BlockedFiltering:     []uint64{},
			DNSQueries:           []uint64{},
			ReplacedParental:     []uint64{},
			ReplacedSafebrowsing: []uint64{},
		}, true
	}

	timeUnit := Hours
	if limit/24 > 7 {
		timeUnit = Days
	}

	units, firstID := s.loadUnits(limit)
	if units == nil {
		return StatsResp{}, false
	}

	dnsQueries := statsCollector(units, firstID, timeUnit, func(u *unitDB) (num uint64) { return u.NTotal })
	if timeUnit != Hours && len(dnsQueries) != int(limit/24) {
		log.Fatalf("len(dnsQueries) != limit: %d %d", len(dnsQueries), limit)
	}

	data := StatsResp{
		DNSQueries:           dnsQueries,
		BlockedFiltering:     statsCollector(units, firstID, timeUnit, func(u *unitDB) (num uint64) { return u.NResult[RFiltered] }),
		ReplacedSafebrowsing: statsCollector(units, firstID, timeUnit, func(u *unitDB) (num uint64) { return u.NResult[RSafeBrowsing] }),
		ReplacedParental:     statsCollector(units, firstID, timeUnit, func(u *unitDB) (num uint64) { return u.NResult[RParental] }),
		TopQueried:           topsCollector(units, maxDomains, s.ignored, func(u *unitDB) (pairs []countPair) { return u.Domains }),
		TopBlocked:           topsCollector(units, maxDomains, s.ignored, func(u *unitDB) (pairs []countPair) { return u.BlockedDomains }),
		TopClients:           topsCollector(units, maxClients, nil, topClientPairs(s)),
	}

	// Total counters:
	sum := unitDB{
		NResult: make([]uint64, resultLast),
	}
	timeN := 0
	for _, u := range units {
		sum.NTotal += u.NTotal
		sum.TimeAvg += u.TimeAvg
		if u.TimeAvg != 0 {
			timeN++
		}
		sum.NResult[RFiltered] += u.NResult[RFiltered]
		sum.NResult[RSafeBrowsing] += u.NResult[RSafeBrowsing]
		sum.NResult[RSafeSearch] += u.NResult[RSafeSearch]
		sum.NResult[RParental] += u.NResult[RParental]
	}

	data.NumDNSQueries = sum.NTotal
	data.NumBlockedFiltering = sum.NResult[RFiltered]
	data.NumReplacedSafebrowsing = sum.NResult[RSafeBrowsing]
	data.NumReplacedSafesearch = sum.NResult[RSafeSearch]
	data.NumReplacedParental = sum.NResult[RParental]

	if timeN != 0 {
		data.AvgProcessingTime = float64(sum.TimeAvg/uint32(timeN)) / 1000000
	}

	data.TimeUnits = "hours"
	if timeUnit == Days {
		data.TimeUnits = "days"
	}

	return data, true
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
