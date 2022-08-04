package stats

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"net"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"go.etcd.io/bbolt"
)

// TODO(a.garipov): Rewrite all of this.  Add proper error handling and
// inspection.  Improve logging.  Decrease complexity.

const (
	maxDomains = 100 // max number of top domains to store in file or return via Get()
	maxClients = 100 // max number of top clients to store in file or return via Get()
)

// StatsCtx collects the statistics and flushes it to the database.  Its default
// flushing interval is one hour.
//
// TODO(e.burkov):  Use atomic.Pointer for accessing curr and db in go1.19.
type StatsCtx struct {
	// currMu protects the current unit.
	currMu *sync.Mutex
	// curr is the actual statistics collection result.
	curr *unit

	// dbMu protects db.
	dbMu *sync.Mutex
	// db is the opened statistics database, if any.
	db *bbolt.DB

	// unitIDGen is the function that generates an identifier for the current
	// unit.  It's here for only testing purposes.
	unitIDGen UnitIDGenFunc

	// httpRegister is used to set HTTP handlers.
	httpRegister aghhttp.RegisterFunc

	// configModified is called whenever the configuration is modified via web
	// interface.
	configModified func()

	// filename is the name of database file.
	filename string

	// limitHours is the maximum number of hours to collect statistics into the
	// current unit.
	limitHours uint32
}

// unit collects the statistics data for a specific period of time.
type unit struct {
	// mu protects all the fields of a unit.
	mu *sync.RWMutex

	// id is the unique unit's identifier.  It's set to an absolute hour number
	// since the beginning of UNIX time by the default ID generating function.
	id uint32

	// nTotal stores the total number of requests.
	nTotal uint64
	// nResult stores the number of requests grouped by it's result.
	nResult []uint64
	// timeSum stores the sum of processing time in milliseconds of each request
	// written by the unit.
	timeSum uint64

	// domains stores the number of requests for each domain.
	domains map[string]uint64
	// blockedDomains stores the number of requests for each domain that has
	// been blocked.
	blockedDomains map[string]uint64
	// clients stores the number of requests from each client.
	clients map[string]uint64
}

// ongoing returns the current unit.  It's safe for concurrent use.
//
// Note that the unit itself should be locked before accessing.
func (s *StatsCtx) ongoing() (u *unit) {
	s.currMu.Lock()
	defer s.currMu.Unlock()

	return s.curr
}

// swapCurrent swaps the current unit with another and returns it.  It's safe
// for concurrent use.
func (s *StatsCtx) swapCurrent(with *unit) (old *unit) {
	s.currMu.Lock()
	defer s.currMu.Unlock()

	old, s.curr = s.curr, with

	return old
}

// database returns the database if it's opened.  It's safe for concurrent use.
func (s *StatsCtx) database() (db *bbolt.DB) {
	s.dbMu.Lock()
	defer s.dbMu.Unlock()

	return s.db
}

// swapDatabase swaps the database with another one and returns it.  It's safe
// for concurrent use.
func (s *StatsCtx) swapDatabase(with *bbolt.DB) (old *bbolt.DB) {
	s.dbMu.Lock()
	defer s.dbMu.Unlock()

	old, s.db = s.db, with

	return old
}

// countPair is a single name-number pair for deserializing statistics data into
// the database.
type countPair struct {
	Name  string
	Count uint64
}

// unitDB is the structure for deserializing statistics data into the database.
type unitDB struct {
	// NTotal is the total number of requests.
	NTotal uint64
	// NResult is the number of requests by the result's kind.
	NResult []uint64

	// Domains is the number of requests for each domain name.
	Domains []countPair
	// BlockedDomains is the number of requests blocked for each domain name.
	BlockedDomains []countPair
	// Clients is the number of requests from each client.
	Clients []countPair

	// TimeAvg is the average of processing times in milliseconds of all the
	// requests in the unit.
	TimeAvg uint32
}

// withRecovered turns the value recovered from panic if any into an error and
// combines it with the one pointed by orig.  orig must be non-nil.
func withRecovered(orig *error) {
	p := recover()
	if p == nil {
		return
	}

	var err error
	switch p := p.(type) {
	case error:
		err = fmt.Errorf("panic: %w", p)
	default:
		err = fmt.Errorf("panic: recovered value of type %[1]T: %[1]v", p)
	}

	*orig = errors.WithDeferred(*orig, err)
}

// isEnabled is a helper that check if the statistics collecting is enabled.
func (s *StatsCtx) isEnabled() (ok bool) {
	return atomic.LoadUint32(&s.limitHours) != 0
}

// New creates s from conf and properly initializes it.  Don't use s before
// calling it's Start method.
func New(conf Config) (s *StatsCtx, err error) {
	defer withRecovered(&err)

	s = &StatsCtx{
		currMu:         &sync.Mutex{},
		dbMu:           &sync.Mutex{},
		filename:       conf.Filename,
		configModified: conf.ConfigModified,
		httpRegister:   conf.HTTPRegister,
	}
	if s.limitHours = conf.LimitDays * 24; !checkInterval(conf.LimitDays) {
		s.limitHours = 24
	}
	if s.unitIDGen = newUnitID; conf.UnitID != nil {
		s.unitIDGen = conf.UnitID
	}

	if err = s.dbOpen(); err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	id := s.unitIDGen()
	tx := beginTxn(s.db, true)
	var udb *unitDB
	if tx != nil {
		log.Tracef("Deleting old units...")
		firstID := id - s.limitHours - 1
		unitDel := 0

		err = tx.ForEach(newBucketWalker(tx, &unitDel, firstID))
		if err != nil && !errors.Is(err, errStop) {
			log.Debug("stats: deleting units: %s", err)
		}

		udb = s.loadUnitFromDB(tx, id)

		if unitDel != 0 {
			s.commitTxn(tx)
		} else {
			err = tx.Rollback()
			if err != nil {
				log.Debug("rolling back: %s", err)
			}
		}
	}

	u := newUnit(id)
	// This use of deserialize is safe since the accessed unit has just been
	// created.
	u.deserialize(udb)
	s.curr = u

	log.Debug("stats: initialized")

	return s, nil
}

// TODO(a.garipov): See if this is actually necessary.  Looks like a rather
// bizarre solution.
const errStop errors.Error = "stop iteration"

// newBucketWalker returns a new bucket walker that deletes old units.  The
// integer that unitDelPtr points to is incremented for every successful
// deletion.  If the bucket isn't deleted, f returns errStop.
func newBucketWalker(
	tx *bbolt.Tx,
	unitDelPtr *int,
	firstID uint32,
) (f func(name []byte, b *bbolt.Bucket) (err error)) {
	return func(name []byte, _ *bbolt.Bucket) (err error) {
		nameID, ok := unitNameToID(name)
		if !ok || nameID < firstID {
			err = tx.DeleteBucket(name)
			if err != nil {
				log.Debug("stats: tx.DeleteBucket: %s", err)

				return nil
			}

			log.Debug("stats: deleted unit %d (name %x)", nameID, name)

			*unitDelPtr++

			return nil
		}

		return errStop
	}
}

// Start makes s process the incoming data.
func (s *StatsCtx) Start() {
	s.initWeb()
	go s.periodicFlush()
}

// checkInterval returns true if days is valid to be used as statistics
// retention interval.  The valid values are 0, 1, 7, 30 and 90.
func checkInterval(days uint32) (ok bool) {
	return days == 0 || days == 1 || days == 7 || days == 30 || days == 90
}

// dbOpen returns an error if the database can't be opened from the specified
// file.  It's safe for concurrent use.
func (s *StatsCtx) dbOpen() (err error) {
	log.Tracef("db.Open...")

	s.dbMu.Lock()
	defer s.dbMu.Unlock()

	s.db, err = bbolt.Open(s.filename, 0o644, nil)
	if err != nil {
		log.Error("stats: open DB: %s: %s", s.filename, err)
		if err.Error() == "invalid argument" {
			log.Error("AdGuard Home cannot be initialized due to an incompatible file system.\nPlease read the explanation here: https://github.com/AdguardTeam/AdGuardHome/wiki/Getting-Started#limitations")
		}

		return err
	}

	log.Tracef("db.Open")

	return nil
}

// newUnitID is the default UnitIDGenFunc that generates the unique id hourly.
func newUnitID() (id uint32) {
	const secsInHour = int64(time.Hour / time.Second)

	return uint32(time.Now().Unix() / secsInHour)
}

// newUnit allocates the new *unit.
func newUnit(id uint32) (u *unit) {
	return &unit{
		mu:             &sync.RWMutex{},
		id:             id,
		nResult:        make([]uint64, resultLast),
		domains:        make(map[string]uint64),
		blockedDomains: make(map[string]uint64),
		clients:        make(map[string]uint64),
	}
}

// beginTxn opens a new database transaction.  If writable is true, the
// transaction will be opened for writing, and for reading otherwise.  It
// returns nil if the transaction can't be created.
func beginTxn(db *bbolt.DB, writable bool) (tx *bbolt.Tx) {
	if db == nil {
		return nil
	}

	log.Tracef("opening a database transaction")

	tx, err := db.Begin(writable)
	if err != nil {
		log.Error("stats: opening a transaction: %s", err)

		return nil
	}

	log.Tracef("transaction has been opened")

	return tx
}

// commitTxn applies the changes made in tx to the database.
func (s *StatsCtx) commitTxn(tx *bbolt.Tx) {
	err := tx.Commit()
	if err != nil {
		log.Error("stats: committing a transaction: %s", err)

		return
	}

	log.Tracef("transaction has been committed")
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

// Flush the current unit to DB and delete an old unit when a new hour is started
// If a unit must be flushed:
// . lock DB
// . atomically set a new empty unit as the current one and get the old unit
//   This is important to do it inside DB lock, so the reader won't get inconsistent results.
// . write the unit to DB
// . remove the stale unit from DB
// . unlock DB
func (s *StatsCtx) periodicFlush() {
	for ptr := s.ongoing(); ptr != nil; ptr = s.ongoing() {
		id := s.unitIDGen()
		// Access the unit's ID with atomic to avoid locking the whole unit.
		if !s.isEnabled() || atomic.LoadUint32(&ptr.id) == id {
			time.Sleep(time.Second)

			continue
		}

		tx := beginTxn(s.database(), true)

		nu := newUnit(id)
		u := s.swapCurrent(nu)
		udb := u.serialize()

		if tx == nil {
			continue
		}

		flushOK := flushUnitToDB(tx, u.id, udb)
		delOK := s.deleteUnit(tx, id-atomic.LoadUint32(&s.limitHours))
		if flushOK || delOK {
			s.commitTxn(tx)
		} else {
			_ = tx.Rollback()
		}
	}

	log.Tracef("periodicFlush() exited")
}

// deleteUnit removes the unit by it's id from the database the tx belongs to.
func (s *StatsCtx) deleteUnit(tx *bbolt.Tx, id uint32) bool {
	err := tx.DeleteBucket(idToUnitName(id))
	if err != nil {
		log.Tracef("stats: bolt DeleteBucket: %s", err)

		return false
	}

	log.Debug("stats: deleted unit %d", id)

	return true
}

func convertMapToSlice(m map[string]uint64, max int) []countPair {
	a := []countPair{}
	for k, v := range m {
		a = append(a, countPair{Name: k, Count: v})
	}
	less := func(i, j int) bool {
		return a[j].Count < a[i].Count
	}
	sort.Slice(a, less)
	if max > len(a) {
		max = len(a)
	}
	return a[:max]
}

func convertSliceToMap(a []countPair) map[string]uint64 {
	m := map[string]uint64{}
	for _, it := range a {
		m[it.Name] = it.Count
	}
	return m
}

// serialize converts u to the *unitDB.  It's safe for concurrent use.
func (u *unit) serialize() (udb *unitDB) {
	u.mu.RLock()
	defer u.mu.RUnlock()

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

// deserealize assigns the appropriate values from udb to u.  u must not be nil.
// It's safe for concurrent use.
func (u *unit) deserialize(udb *unitDB) {
	if udb == nil {
		return
	}

	u.mu.Lock()
	defer u.mu.Unlock()

	u.nTotal = udb.NTotal
	u.nResult = make([]uint64, resultLast)
	copy(u.nResult, udb.NResult)
	u.domains = convertSliceToMap(udb.Domains)
	u.blockedDomains = convertSliceToMap(udb.BlockedDomains)
	u.clients = convertSliceToMap(udb.Clients)
	u.timeSum = uint64(udb.TimeAvg) * udb.NTotal
}

func flushUnitToDB(tx *bbolt.Tx, id uint32, udb *unitDB) bool {
	log.Tracef("Flushing unit %d", id)

	bkt, err := tx.CreateBucketIfNotExists(idToUnitName(id))
	if err != nil {
		log.Error("tx.CreateBucketIfNotExists: %s", err)
		return false
	}

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err = enc.Encode(udb)
	if err != nil {
		log.Error("gob.Encode: %s", err)
		return false
	}

	err = bkt.Put([]byte{0}, buf.Bytes())
	if err != nil {
		log.Error("bkt.Put: %s", err)
		return false
	}

	return true
}

func (s *StatsCtx) loadUnitFromDB(tx *bbolt.Tx, id uint32) *unitDB {
	bkt := tx.Bucket(idToUnitName(id))
	if bkt == nil {
		return nil
	}

	// log.Tracef("Loading unit %d", id)

	var buf bytes.Buffer
	buf.Write(bkt.Get([]byte{0}))
	dec := gob.NewDecoder(&buf)
	udb := unitDB{}
	err := dec.Decode(&udb)
	if err != nil {
		log.Error("gob Decode: %s", err)
		return nil
	}

	return &udb
}

func convertTopSlice(a []countPair) (m []map[string]uint64) {
	m = make([]map[string]uint64, 0, len(a))
	for _, it := range a {
		m = append(m, map[string]uint64{it.Name: it.Count})
	}

	return m
}

func (s *StatsCtx) setLimit(limitDays int) {
	atomic.StoreUint32(&s.limitHours, uint32(24*limitDays))
	if limitDays == 0 {
		s.clear()
	}

	log.Debug("stats: set limit: %d days", limitDays)
}

func (s *StatsCtx) WriteDiskConfig(dc *DiskConfig) {
	dc.Interval = atomic.LoadUint32(&s.limitHours) / 24
}

func (s *StatsCtx) Close() {
	u := s.swapCurrent(nil)

	db := s.database()
	if tx := beginTxn(db, true); tx != nil {
		udb := u.serialize()
		if flushUnitToDB(tx, u.id, udb) {
			s.commitTxn(tx)
		} else {
			_ = tx.Rollback()
		}
	}

	if db != nil {
		log.Tracef("db.Close...")
		_ = db.Close()
		log.Tracef("db.Close")
	}

	log.Debug("stats: closed")
}

// Reset counters and clear database
func (s *StatsCtx) clear() {
	db := s.database()
	tx := beginTxn(db, true)
	if tx != nil {
		_ = s.swapDatabase(nil)
		_ = tx.Rollback()
		// the active transactions can continue using database,
		//  but no new transactions will be opened
		_ = db.Close()
		log.Tracef("db.Close")
		// all active transactions are now closed
	}

	u := newUnit(s.unitIDGen())
	_ = s.swapCurrent(u)

	err := os.Remove(s.filename)
	if err != nil {
		log.Error("os.Remove: %s", err)
	}

	_ = s.dbOpen()

	log.Debug("stats: cleared")
}

func (s *StatsCtx) Update(e Entry) {
	if !s.isEnabled() {
		return
	}

	if e.Result == 0 ||
		e.Result >= resultLast ||
		e.Domain == "" ||
		e.Client == "" {
		return
	}

	clientID := e.Client
	if ip := net.ParseIP(clientID); ip != nil {
		clientID = ip.String()
	}

	u := s.ongoing()
	if u == nil {
		return
	}

	u.mu.Lock()
	defer u.mu.Unlock()

	u.nResult[e.Result]++
	if e.Result == RNotFiltered {
		u.domains[e.Domain]++
	} else {
		u.blockedDomains[e.Domain]++
	}

	u.clients[clientID]++
	u.timeSum += uint64(e.Time)
	u.nTotal++
}

func (s *StatsCtx) loadUnits(limit uint32) ([]*unitDB, uint32) {
	tx := beginTxn(s.database(), false)
	if tx == nil {
		return nil, 0
	}

	cur := s.ongoing()
	var curID uint32
	if cur != nil {
		curID = atomic.LoadUint32(&cur.id)
	} else {
		curID = s.unitIDGen()
	}

	// Per-hour units.
	units := []*unitDB{}
	firstID := curID - limit + 1
	for i := firstID; i != curID; i++ {
		u := s.loadUnitFromDB(tx, i)
		if u == nil {
			u = &unitDB{}
			u.NResult = make([]uint64, resultLast)
		}
		units = append(units, u)
	}

	_ = tx.Rollback()

	if cur != nil {
		units = append(units, cur.serialize())
	}

	if len(units) != int(limit) {
		log.Fatalf("len(units) != limit: %d %d", len(units), limit)
	}

	return units, firstID
}

// numsGetter is a signature for statsCollector argument.
type numsGetter func(u *unitDB) (num uint64)

// statsCollector collects statisctics for the given *unitDB slice by specified
// timeUnit using ng to retrieve data.
func statsCollector(units []*unitDB, firstID uint32, timeUnit TimeUnit, ng numsGetter) (nums []uint64) {
	if timeUnit == Hours {
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
func topsCollector(units []*unitDB, max int, pg pairsGetter) []map[string]uint64 {
	m := map[string]uint64{}
	for _, u := range units {
		for _, cp := range pg(u) {
			m[cp.Name] += cp.Count
		}
	}
	a2 := convertMapToSlice(m, max)
	return convertTopSlice(a2)
}

/* Algorithm:
. Prepare array of N units, where N is the value of "limit" configuration setting
 . Load data for the most recent units from file
   If a unit with required ID doesn't exist, just add an empty unit
 . Get data for the current unit
. Process data from the units and prepare an output map object:
 * per time unit counters:
  * DNS-queries/time-unit
  * blocked/time-unit
  * safebrowsing-blocked/time-unit
  * parental-blocked/time-unit
  If time-unit is an hour, just add values from each unit to an array.
  If time-unit is a day, aggregate per-hour data into days.
 * top counters:
  * queries/domain
  * queries/blocked-domain
  * queries/client
  To get these values we first sum up data for all units into a single map.
  Then we get the pairs with the highest numbers (the values are sorted in descending order)
 * total counters:
  * DNS-queries
  * blocked
  * safebrowsing-blocked
  * safesearch-blocked
  * parental-blocked
  These values are just the sum of data for all units.
*/
func (s *StatsCtx) getData() (statsResponse, bool) {
	limit := atomic.LoadUint32(&s.limitHours)
	if limit == 0 {
		return statsResponse{
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
		return statsResponse{}, false
	}

	dnsQueries := statsCollector(units, firstID, timeUnit, func(u *unitDB) (num uint64) { return u.NTotal })
	if timeUnit != Hours && len(dnsQueries) != int(limit/24) {
		log.Fatalf("len(dnsQueries) != limit: %d %d", len(dnsQueries), limit)
	}

	data := statsResponse{
		DNSQueries:           dnsQueries,
		BlockedFiltering:     statsCollector(units, firstID, timeUnit, func(u *unitDB) (num uint64) { return u.NResult[RFiltered] }),
		ReplacedSafebrowsing: statsCollector(units, firstID, timeUnit, func(u *unitDB) (num uint64) { return u.NResult[RSafeBrowsing] }),
		ReplacedParental:     statsCollector(units, firstID, timeUnit, func(u *unitDB) (num uint64) { return u.NResult[RParental] }),
		TopQueried:           topsCollector(units, maxDomains, func(u *unitDB) (pairs []countPair) { return u.Domains }),
		TopBlocked:           topsCollector(units, maxDomains, func(u *unitDB) (pairs []countPair) { return u.BlockedDomains }),
		TopClients:           topsCollector(units, maxClients, func(u *unitDB) (pairs []countPair) { return u.Clients }),
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

func (s *StatsCtx) GetTopClientsIP(maxCount uint) []net.IP {
	if !s.isEnabled() {
		return nil
	}

	units, _ := s.loadUnits(atomic.LoadUint32(&s.limitHours))
	if units == nil {
		return nil
	}

	// top clients
	m := map[string]uint64{}
	for _, u := range units {
		for _, it := range u.Clients {
			m[it.Name] += it.Count
		}
	}
	a := convertMapToSlice(m, int(maxCount))
	d := []net.IP{}
	for _, it := range a {
		ip := net.ParseIP(it.Name)
		if ip != nil {
			d = append(d, ip)
		}
	}
	return d
}
