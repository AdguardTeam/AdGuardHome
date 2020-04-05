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
	"time"

	"github.com/AdguardTeam/golibs/log"
	bolt "go.etcd.io/bbolt"
)

const (
	maxDomains = 100 // max number of top domains to store in file or return via Get()
	maxClients = 100 // max number of top clients to store in file or return via Get()
)

// statsCtx - global context
type statsCtx struct {
	db   *bolt.DB
	conf *Config

	unit     *unit      // the current unit
	unitLock sync.Mutex // protect 'unit'
}

// data for 1 time unit
type unit struct {
	id uint32 // unit ID.  Default: absolute hour since Jan 1, 1970

	nTotal  uint64   // total requests
	nResult []uint64 // number of requests per one result
	timeSum uint64   // sum of processing time of all requests (usec)

	// top:
	domains        map[string]uint64 // number of requests per domain
	blockedDomains map[string]uint64 // number of blocked requests per domain
	clients        map[string]uint64 // number of requests per client
}

// name-count pair
type countPair struct {
	Name  string
	Count uint64
}

// structure for storing data in file
type unitDB struct {
	NTotal  uint64
	NResult []uint64

	Domains        []countPair
	BlockedDomains []countPair
	Clients        []countPair

	TimeAvg uint32 // usec
}

func createObject(conf Config) (*statsCtx, error) {
	s := statsCtx{}
	if !checkInterval(conf.LimitDays) {
		conf.LimitDays = 1
	}
	s.conf = &Config{}
	*s.conf = conf
	s.conf.limit = conf.LimitDays * 24
	if conf.UnitID == nil {
		s.conf.UnitID = newUnitID
	}

	if !s.dbOpen() {
		return nil, fmt.Errorf("open database")
	}

	id := s.conf.UnitID()
	tx := s.beginTxn(true)
	var udb *unitDB
	if tx != nil {
		log.Tracef("Deleting old units...")
		firstID := id - s.conf.limit - 1
		unitDel := 0
		forEachBkt := func(name []byte, b *bolt.Bucket) error {
			id := uint32(btoi(name))
			if id < firstID {
				err := tx.DeleteBucket(name)
				if err != nil {
					log.Debug("tx.DeleteBucket: %s", err)
				}
				log.Debug("Stats: deleted unit %d", id)
				unitDel++
				return nil
			}
			return fmt.Errorf("")
		}
		_ = tx.ForEach(forEachBkt)

		udb = s.loadUnitFromDB(tx, id)

		if unitDel != 0 {
			s.commitTxn(tx)
		} else {
			_ = tx.Rollback()
		}
	}

	u := unit{}
	s.initUnit(&u, id)
	if udb != nil {
		deserialize(&u, udb)
	}
	s.unit = &u

	log.Debug("Stats: initialized")
	return &s, nil
}

func (s *statsCtx) Start() {
	s.initWeb()
	go s.periodicFlush()
}

func checkInterval(days uint32) bool {
	return days == 1 || days == 7 || days == 30 || days == 90
}

func (s *statsCtx) dbOpen() bool {
	var err error
	log.Tracef("db.Open...")
	s.db, err = bolt.Open(s.conf.Filename, 0644, nil)
	if err != nil {
		log.Error("Stats: open DB: %s: %s", s.conf.Filename, err)
		if err.Error() == "invalid argument" {
			log.Error("AdGuard Home cannot be initialized due to an incompatible file system.\nPlease read the explanation here: https://github.com/AdguardTeam/AdGuardHome/wiki/Getting-Started#limitations")
		}
		return false
	}
	log.Tracef("db.Open")
	return true
}

// Atomically swap the currently active unit with a new value
// Return old value
func (s *statsCtx) swapUnit(new *unit) *unit {
	s.unitLock.Lock()
	u := s.unit
	s.unit = new
	s.unitLock.Unlock()
	return u
}

// Get unit ID for the current hour
func newUnitID() uint32 {
	return uint32(time.Now().Unix() / (60 * 60))
}

// Initialize a unit
func (s *statsCtx) initUnit(u *unit, id uint32) {
	u.id = id
	u.nResult = make([]uint64, rLast)
	u.domains = make(map[string]uint64)
	u.blockedDomains = make(map[string]uint64)
	u.clients = make(map[string]uint64)
}

// Open a DB transaction
func (s *statsCtx) beginTxn(wr bool) *bolt.Tx {
	db := s.db
	if db == nil {
		return nil
	}

	log.Tracef("db.Begin...")
	tx, err := db.Begin(wr)
	if err != nil {
		log.Error("db.Begin: %s", err)
		return nil
	}
	log.Tracef("db.Begin")
	return tx
}

func (s *statsCtx) commitTxn(tx *bolt.Tx) {
	err := tx.Commit()
	if err != nil {
		log.Debug("tx.Commit: %s", err)
		return
	}
	log.Tracef("tx.Commit")
}

// Get unit name
func unitName(id uint32) []byte {
	return itob(uint64(id))
}

// Convert integer to 8-byte array (big endian)
func itob(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}

// Convert 8-byte array (big endian) to integer
func btoi(b []byte) uint64 {
	return binary.BigEndian.Uint64(b)
}

// Flush the current unit to DB and delete an old unit when a new hour is started
// If a unit must be flushed:
// . lock DB
// . atomically set a new empty unit as the current one and get the old unit
//   This is important to do it inside DB lock, so the reader won't get inconsistent results.
// . write the unit to DB
// . remove the stale unit from DB
// . unlock DB
func (s *statsCtx) periodicFlush() {
	for {
		s.unitLock.Lock()
		ptr := s.unit
		s.unitLock.Unlock()
		if ptr == nil {
			break
		}

		id := s.conf.UnitID()
		if ptr.id == id {
			time.Sleep(time.Second)
			continue
		}

		tx := s.beginTxn(true)

		nu := unit{}
		s.initUnit(&nu, id)
		u := s.swapUnit(&nu)
		udb := serialize(u)

		if tx == nil {
			continue
		}
		ok1 := s.flushUnitToDB(tx, u.id, udb)
		ok2 := s.deleteUnit(tx, id-s.conf.limit)
		if ok1 || ok2 {
			s.commitTxn(tx)
		} else {
			_ = tx.Rollback()
		}
	}
	log.Tracef("periodicFlush() exited")
}

// Delete unit's data from file
func (s *statsCtx) deleteUnit(tx *bolt.Tx, id uint32) bool {
	err := tx.DeleteBucket(unitName(id))
	if err != nil {
		log.Tracef("bolt DeleteBucket: %s", err)
		return false
	}
	log.Debug("Stats: deleted unit %d", id)
	return true
}

func convertMapToArray(m map[string]uint64, max int) []countPair {
	a := []countPair{}
	for k, v := range m {
		pair := countPair{}
		pair.Name = k
		pair.Count = v
		a = append(a, pair)
	}
	less := func(i, j int) bool {
		if a[i].Count >= a[j].Count {
			return true
		}
		return false
	}
	sort.Slice(a, less)
	if max > len(a) {
		max = len(a)
	}
	return a[:max]
}

func convertArrayToMap(a []countPair) map[string]uint64 {
	m := map[string]uint64{}
	for _, it := range a {
		m[it.Name] = it.Count
	}
	return m
}

func serialize(u *unit) *unitDB {
	udb := unitDB{}
	udb.NTotal = u.nTotal
	for _, it := range u.nResult {
		udb.NResult = append(udb.NResult, it)
	}
	if u.nTotal != 0 {
		udb.TimeAvg = uint32(u.timeSum / u.nTotal)
	}
	udb.Domains = convertMapToArray(u.domains, maxDomains)
	udb.BlockedDomains = convertMapToArray(u.blockedDomains, maxDomains)
	udb.Clients = convertMapToArray(u.clients, maxClients)
	return &udb
}

func deserialize(u *unit, udb *unitDB) {
	u.nTotal = udb.NTotal

	n := len(udb.NResult)
	if n < len(u.nResult) {
		n = len(u.nResult) // n = min(len(udb.NResult), len(u.nResult))
	}
	for i := 1; i < n; i++ {
		u.nResult[i] = udb.NResult[i]
	}

	u.domains = convertArrayToMap(udb.Domains)
	u.blockedDomains = convertArrayToMap(udb.BlockedDomains)
	u.clients = convertArrayToMap(udb.Clients)
	u.timeSum = uint64(udb.TimeAvg) * u.nTotal
}

func (s *statsCtx) flushUnitToDB(tx *bolt.Tx, id uint32, udb *unitDB) bool {
	log.Tracef("Flushing unit %d", id)

	bkt, err := tx.CreateBucketIfNotExists(unitName(id))
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

func (s *statsCtx) loadUnitFromDB(tx *bolt.Tx, id uint32) *unitDB {
	bkt := tx.Bucket(unitName(id))
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

func convertTopArray(a []countPair) []map[string]uint64 {
	m := []map[string]uint64{}
	for _, it := range a {
		ent := map[string]uint64{}
		ent[it.Name] = it.Count
		m = append(m, ent)
	}
	return m
}

func (s *statsCtx) setLimit(limitDays int) {
	conf := *s.conf
	conf.limit = uint32(limitDays) * 24
	s.conf = &conf
	log.Debug("Stats: set limit: %d", limitDays)
}

func (s *statsCtx) WriteDiskConfig(dc *DiskConfig) {
	dc.Interval = s.conf.limit / 24
}

func (s *statsCtx) Close() {
	u := s.swapUnit(nil)
	udb := serialize(u)
	tx := s.beginTxn(true)
	if tx != nil {
		if s.flushUnitToDB(tx, u.id, udb) {
			s.commitTxn(tx)
		} else {
			_ = tx.Rollback()
		}
	}

	if s.db != nil {
		log.Tracef("db.Close...")
		_ = s.db.Close()
		log.Tracef("db.Close")
	}

	log.Debug("Stats: closed")
}

// Reset counters and clear database
func (s *statsCtx) clear() {
	tx := s.beginTxn(true)
	if tx != nil {
		db := s.db
		s.db = nil
		_ = tx.Rollback()
		// the active transactions can continue using database,
		//  but no new transactions will be opened
		_ = db.Close()
		log.Tracef("db.Close")
		// all active transactions are now closed
	}

	u := unit{}
	s.initUnit(&u, s.conf.UnitID())
	_ = s.swapUnit(&u)

	err := os.Remove(s.conf.Filename)
	if err != nil {
		log.Error("os.Remove: %s", err)
	}

	_ = s.dbOpen()

	log.Debug("Stats: cleared")
}

// Get Client IP address
func (s *statsCtx) getClientIP(clientIP string) string {
	if s.conf.AnonymizeClientIP {
		ip := net.ParseIP(clientIP)
		if ip != nil {
			ip4 := ip.To4()
			const AnonymizeClientIP4Mask = 24
			const AnonymizeClientIP6Mask = 112
			if ip4 != nil {
				clientIP = ip4.Mask(net.CIDRMask(AnonymizeClientIP4Mask, 32)).String()
			} else {
				clientIP = ip.Mask(net.CIDRMask(AnonymizeClientIP6Mask, 128)).String()
			}
		}
	}

	return clientIP
}

func (s *statsCtx) Update(e Entry) {
	if e.Result == 0 ||
		e.Result >= rLast ||
		len(e.Domain) == 0 ||
		!(len(e.Client) == 4 || len(e.Client) == 16) {
		return
	}
	client := s.getClientIP(e.Client.String())

	s.unitLock.Lock()
	u := s.unit

	u.nResult[e.Result]++

	if e.Result == RNotFiltered {
		u.domains[e.Domain]++
	} else {
		u.blockedDomains[e.Domain]++
	}

	u.clients[client]++
	u.timeSum += uint64(e.Time)
	u.nTotal++
	s.unitLock.Unlock()
}

func (s *statsCtx) loadUnits(limit uint32) ([]*unitDB, uint32) {
	tx := s.beginTxn(false)
	if tx == nil {
		return nil, 0
	}

	s.unitLock.Lock()
	curUnit := serialize(s.unit)
	curID := s.unit.id
	s.unitLock.Unlock()

	units := []*unitDB{} //per-hour units
	firstID := curID - limit + 1
	for i := firstID; i != curID; i++ {
		u := s.loadUnitFromDB(tx, i)
		if u == nil {
			u = &unitDB{}
			u.NResult = make([]uint64, rLast)
		}
		units = append(units, u)
	}

	_ = tx.Rollback()

	units = append(units, curUnit)

	if len(units) != int(limit) {
		log.Fatalf("len(units) != limit: %d %d", len(units), limit)
	}

	return units, firstID
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
// nolint (gocyclo)
func (s *statsCtx) getData() map[string]interface{} {
	limit := s.conf.limit

	d := map[string]interface{}{}
	timeUnit := Hours
	if limit/24 > 7 {
		timeUnit = Days
	}

	units, firstID := s.loadUnits(limit)
	if units == nil {
		return nil
	}

	// per time unit counters:

	// 720 hours may span 31 days, so we skip data for the first day in this case
	firstDayID := (firstID + 24 - 1) / 24 * 24 // align_ceil(24)

	a := []uint64{}
	if timeUnit == Hours {
		for _, u := range units {
			a = append(a, u.NTotal)
		}
	} else {
		var sum uint64
		id := firstDayID
		nextDayID := firstDayID + 24
		for i := firstDayID - firstID; int(i) != len(units); i++ {
			sum += units[i].NTotal
			if id == nextDayID {
				a = append(a, sum)
				sum = 0
				nextDayID += 24
			}
			id++
		}
		if id <= nextDayID {
			a = append(a, sum)
		}
		if len(a) != int(limit/24) {
			log.Fatalf("len(a) != limit: %d %d", len(a), limit)
		}
	}
	d["dns_queries"] = a

	a = []uint64{}
	if timeUnit == Hours {
		for _, u := range units {
			a = append(a, u.NResult[RFiltered])
		}
	} else {
		var sum uint64
		id := firstDayID
		nextDayID := firstDayID + 24
		for i := firstDayID - firstID; int(i) != len(units); i++ {
			sum += units[i].NResult[RFiltered]
			if id == nextDayID {
				a = append(a, sum)
				sum = 0
				nextDayID += 24
			}
			id++
		}
		if id <= nextDayID {
			a = append(a, sum)
		}
	}
	d["blocked_filtering"] = a

	a = []uint64{}
	if timeUnit == Hours {
		for _, u := range units {
			a = append(a, u.NResult[RSafeBrowsing])
		}
	} else {
		var sum uint64
		id := firstDayID
		nextDayID := firstDayID + 24
		for i := firstDayID - firstID; int(i) != len(units); i++ {
			sum += units[i].NResult[RSafeBrowsing]
			if id == nextDayID {
				a = append(a, sum)
				sum = 0
				nextDayID += 24
			}
			id++
		}
		if id <= nextDayID {
			a = append(a, sum)
		}
	}
	d["replaced_safebrowsing"] = a

	a = []uint64{}
	if timeUnit == Hours {
		for _, u := range units {
			a = append(a, u.NResult[RParental])
		}
	} else {
		var sum uint64
		id := firstDayID
		nextDayID := firstDayID + 24
		for i := firstDayID - firstID; int(i) != len(units); i++ {
			sum += units[i].NResult[RParental]
			if id == nextDayID {
				a = append(a, sum)
				sum = 0
				nextDayID += 24
			}
			id++
		}
		if id <= nextDayID {
			a = append(a, sum)
		}
	}
	d["replaced_parental"] = a

	// top counters:

	m := map[string]uint64{}
	for _, u := range units {
		for _, it := range u.Domains {
			m[it.Name] += it.Count
		}
	}
	a2 := convertMapToArray(m, maxDomains)
	d["top_queried_domains"] = convertTopArray(a2)

	m = map[string]uint64{}
	for _, u := range units {
		for _, it := range u.BlockedDomains {
			m[it.Name] += it.Count
		}
	}
	a2 = convertMapToArray(m, maxDomains)
	d["top_blocked_domains"] = convertTopArray(a2)

	m = map[string]uint64{}
	for _, u := range units {
		for _, it := range u.Clients {
			m[it.Name] += it.Count
		}
	}
	a2 = convertMapToArray(m, maxClients)
	d["top_clients"] = convertTopArray(a2)

	// total counters:

	sum := unitDB{}
	sum.NResult = make([]uint64, rLast)
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

	d["num_dns_queries"] = sum.NTotal
	d["num_blocked_filtering"] = sum.NResult[RFiltered]
	d["num_replaced_safebrowsing"] = sum.NResult[RSafeBrowsing]
	d["num_replaced_safesearch"] = sum.NResult[RSafeSearch]
	d["num_replaced_parental"] = sum.NResult[RParental]

	avgTime := float64(0)
	if timeN != 0 {
		avgTime = float64(sum.TimeAvg/uint32(timeN)) / 1000000
	}
	d["avg_processing_time"] = avgTime

	d["time_units"] = "hours"
	if timeUnit == Days {
		d["time_units"] = "days"
	}

	return d
}

func (s *statsCtx) GetTopClientsIP(maxCount uint) []string {
	units, _ := s.loadUnits(s.conf.limit)
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
	a := convertMapToArray(m, int(maxCount))
	d := []string{}
	for _, it := range a {
		d = append(d, it.Name)
	}
	return d
}
