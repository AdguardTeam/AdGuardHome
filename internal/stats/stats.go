// Package stats provides units for managing statistics of the filtering DNS
// server.
package stats

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/netip"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/aghos"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/timeutil"
	"go.etcd.io/bbolt"
)

// checkInterval returns true if days is valid to be used as statistics
// retention interval.  The valid values are 0, 1, 7, 30 and 90.
func checkInterval(days uint32) (ok bool) {
	return days == 0 || days == 1 || days == 7 || days == 30 || days == 90
}

// validateIvl returns an error if ivl is less than an hour or more than a
// year.
func validateIvl(ivl time.Duration) (err error) {
	if ivl < time.Hour {
		return errors.Error("less than an hour")
	}

	if ivl > timeutil.Day*365 {
		return errors.Error("more than a year")
	}

	return nil
}

// Config is the configuration structure for the statistics collecting.
//
// Do not alter any fields of this structure after using it.
type Config struct {
	// Logger is used for logging the operation of the statistics management.
	// It must not be nil.
	Logger *slog.Logger

	// UnitID is the function to generate the identifier for current unit.  If
	// nil, the default function is used, see newUnitID.
	UnitID UnitIDGenFunc

	// ConfigModified will be called each time the configuration changed via web
	// interface.
	ConfigModified func()

	// ShouldCountClient returns client's ignore setting.
	ShouldCountClient func([]string) bool

	// HTTPRegister is the function that registers handlers for the stats
	// endpoints.
	HTTPRegister aghhttp.RegisterFunc

	// Ignored contains the list of host names, which should not be counted,
	// and matches them.
	Ignored *aghnet.IgnoreEngine

	// Filename is the name of the database file.
	Filename string

	// Limit is an upper limit for collecting statistics.
	Limit time.Duration

	// Enabled tells if the statistics are enabled.
	Enabled bool
}

// Interface is the statistics interface to be used by other packages.
type Interface interface {
	// Start begins the statistics collecting.
	Start()

	io.Closer

	// Update collects the incoming statistics data.
	Update(e *Entry)

	// GetTopClientIP returns at most limit IP addresses corresponding to the
	// clients with the most number of requests.
	TopClientsIP(limit uint) []netip.Addr

	// WriteDiskConfig puts the Interface's configuration to the dc.
	WriteDiskConfig(dc *Config)

	// ShouldCount returns true if request for the host should be counted.
	ShouldCount(host string, qType, qClass uint16, ids []string) bool
}

// StatsCtx collects the statistics and flushes it to the database.  Its default
// flushing interval is one hour.
type StatsCtx struct {
	// logger is used for logging the operation of the statistics management.
	// It must not be nil.
	logger *slog.Logger

	// currMu protects curr.
	currMu *sync.RWMutex
	// curr is the actual statistics collection result.
	curr *unit

	// db is the opened statistics database, if any.
	db atomic.Pointer[bbolt.DB]

	// unitIDGen is the function that generates an identifier for the current
	// unit.  It's here for only testing purposes.
	unitIDGen UnitIDGenFunc

	// httpRegister is used to set HTTP handlers.
	httpRegister aghhttp.RegisterFunc

	// configModified is called whenever the configuration is modified via web
	// interface.
	configModified func()

	// confMu protects ignored, limit, and enabled.
	confMu *sync.RWMutex

	// ignored contains the list of host names, which should not be counted,
	// and matches them.
	ignored *aghnet.IgnoreEngine

	// shouldCountClient returns client's ignore setting.
	shouldCountClient func([]string) bool

	// filename is the name of database file.
	filename string

	// limit is an upper limit for collecting statistics.
	limit time.Duration

	// enabled tells if the statistics are enabled.
	enabled bool
}

// New creates s from conf and properly initializes it.  Don't use s before
// calling it's Start method.
func New(conf Config) (s *StatsCtx, err error) {
	defer withRecovered(&err)

	err = validateIvl(conf.Limit)
	if err != nil {
		return nil, fmt.Errorf("unsupported interval: %w", err)
	}

	if conf.ShouldCountClient == nil {
		return nil, errors.Error("should count client is unspecified")
	}

	s = &StatsCtx{
		logger:         conf.Logger,
		currMu:         &sync.RWMutex{},
		httpRegister:   conf.HTTPRegister,
		configModified: conf.ConfigModified,
		filename:       conf.Filename,

		confMu:            &sync.RWMutex{},
		ignored:           conf.Ignored,
		shouldCountClient: conf.ShouldCountClient,
		limit:             conf.Limit,
		enabled:           conf.Enabled,
	}

	if s.unitIDGen = newUnitID; conf.UnitID != nil {
		s.unitIDGen = conf.UnitID
	}

	// TODO(e.burkov):  Move the code below to the Start method.

	err = s.openDB()
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	var udb *unitDB
	id := s.unitIDGen()

	tx, err := s.db.Load().Begin(true)
	if err != nil {
		return nil, fmt.Errorf("opening a transaction: %w", err)
	}

	deleted := s.deleteOldUnits(tx, id-uint32(s.limit.Hours())-1)
	udb = s.loadUnitFromDB(tx, id)

	err = finishTxn(tx, deleted > 0)
	if err != nil {
		s.logger.Error("finishing transacation", slogutil.KeyError, err)
	}

	s.curr = newUnit(id)
	s.curr.deserialize(udb)

	s.logger.Debug("initialized")

	return s, nil
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

// type check
var _ Interface = (*StatsCtx)(nil)

// Start implements the [Interface] interface for *StatsCtx.
func (s *StatsCtx) Start() {
	s.initWeb()

	go s.periodicFlush()
}

// Close implements the [io.Closer] interface for *StatsCtx.
func (s *StatsCtx) Close() (err error) {
	db := s.db.Swap(nil)
	if db == nil {
		return nil
	}
	defer func() {
		cerr := db.Close()
		if cerr == nil {
			s.logger.Debug("database closed")
		}

		err = errors.WithDeferred(err, cerr)
	}()

	tx, err := db.Begin(true)
	if err != nil {
		return fmt.Errorf("opening transaction: %w", err)
	}
	defer func() { err = errors.WithDeferred(err, finishTxn(tx, err == nil)) }()

	s.currMu.RLock()
	defer s.currMu.RUnlock()

	udb := s.curr.serialize()

	return s.flushUnitToDB(udb, tx, s.curr.id)
}

// Update implements the [Interface] interface for *StatsCtx.  e must not be
// nil.
func (s *StatsCtx) Update(e *Entry) {
	s.confMu.Lock()
	defer s.confMu.Unlock()

	if !s.enabled || s.limit == 0 {
		return
	}

	err := e.validate()
	if err != nil {
		s.logger.Debug("validating entry", slogutil.KeyError, err)

		return
	}

	s.currMu.Lock()
	defer s.currMu.Unlock()

	if s.curr == nil {
		s.logger.Error("current unit is nil")

		return
	}

	s.curr.add(e)
}

// WriteDiskConfig implements the [Interface] interface for *StatsCtx.
func (s *StatsCtx) WriteDiskConfig(dc *Config) {
	s.confMu.RLock()
	defer s.confMu.RUnlock()

	dc.Ignored = s.ignored
	dc.Limit = s.limit
	dc.Enabled = s.enabled
}

// TopClientsIP implements the [Interface] interface for *StatsCtx.
func (s *StatsCtx) TopClientsIP(maxCount uint) (ips []netip.Addr) {
	s.confMu.RLock()
	defer s.confMu.RUnlock()

	limit := uint32(s.limit.Hours())
	if !s.enabled || limit == 0 {
		return nil
	}

	units, _ := s.loadUnits(limit)
	if units == nil {
		return nil
	}

	// Collect data for all the clients to sort and crop it afterwards.
	m := map[string]uint64{}
	for _, u := range units {
		for _, it := range u.Clients {
			m[it.Name] += it.Count
		}
	}

	a := convertMapToSlice(m, int(maxCount))
	ips = []netip.Addr{}
	for _, it := range a {
		ip, err := netip.ParseAddr(it.Name)
		if err == nil {
			ips = append(ips, ip)
		}
	}

	return ips
}

// deleteOldUnits walks the buckets available to tx and deletes old units.  It
// returns the number of deletions performed.
func (s *StatsCtx) deleteOldUnits(tx *bbolt.Tx, firstID uint32) (deleted int) {
	s.logger.Debug("deleting old units up to", "unit", firstID)

	// TODO(a.garipov): See if this is actually necessary.  Looks like a rather
	// bizarre solution.
	const errStop errors.Error = "stop iteration"

	walk := func(name []byte, _ *bbolt.Bucket) (err error) {
		nameID, ok := unitNameToID(name)
		if ok && nameID >= firstID {
			return errStop
		}

		err = tx.DeleteBucket(name)
		if err != nil {
			s.logger.Debug("deleting bucket", slogutil.KeyError, err)

			return nil
		}

		s.logger.Debug("deleted unit", "name_id", nameID, "name", fmt.Sprintf("%x", name))

		deleted++

		return nil
	}

	err := tx.ForEach(walk)
	if err != nil && !errors.Is(err, errStop) {
		s.logger.Debug("deleting units", slogutil.KeyError, err)
	}

	return deleted
}

// openDB returns an error if the database can't be opened from the specified
// file.  It's safe for concurrent use.
func (s *StatsCtx) openDB() (err error) {
	s.logger.Debug("opening database")

	var db *bbolt.DB

	db, err = bbolt.Open(s.filename, aghos.DefaultPermFile, nil)
	if err != nil {
		if err.Error() == "invalid argument" {
			const lines = `AdGuard Home cannot be initialized due to an incompatible file system.
Please read the explanation here: https://github.com/AdguardTeam/AdGuardHome/wiki/Getting-Started#limitations`

			// TODO(s.chzhen):  Use passed context.
			slogutil.PrintLines(
				context.TODO(),
				s.logger,
				slog.LevelError,
				"opening database",
				lines,
			)
		}

		return err
	}

	defer s.logger.Debug("database opened")

	s.db.Store(db)

	return nil
}

func (s *StatsCtx) flush() (cont bool, sleepFor time.Duration) {
	id := s.unitIDGen()

	s.confMu.Lock()
	defer s.confMu.Unlock()

	s.currMu.Lock()
	defer s.currMu.Unlock()

	ptr := s.curr
	if ptr == nil {
		return false, 0
	}

	limit := uint32(s.limit.Hours())
	if limit == 0 || ptr.id == id {
		return true, time.Second
	}

	return s.flushDB(id, limit, ptr)
}

// flushDB flushes the unit to the database.  confMu and currMu are expected to
// be locked.
func (s *StatsCtx) flushDB(id, limit uint32, ptr *unit) (cont bool, sleepFor time.Duration) {
	db := s.db.Load()
	if db == nil {
		return true, 0
	}

	isCommitable := true
	tx, err := db.Begin(true)
	if err != nil {
		s.logger.Error("opening transaction", slogutil.KeyError, err)

		return true, 0
	}
	defer func() {
		if err = finishTxn(tx, isCommitable); err != nil {
			s.logger.Error("finishing transaction", slogutil.KeyError, err)
		}
	}()

	s.curr = newUnit(id)

	udb := ptr.serialize()
	flushErr := s.flushUnitToDB(udb, tx, ptr.id)
	if flushErr != nil {
		s.logger.Error("flushing unit", slogutil.KeyError, flushErr)
		isCommitable = false
	}

	delErr := tx.DeleteBucket(idToUnitName(id - limit))

	if delErr != nil {
		// TODO(e.burkov):  Improve the algorithm of deleting the oldest bucket
		// to avoid the error.
		lvl := slog.LevelDebug
		if !errors.Is(delErr, bbolt.ErrBucketNotFound) {
			isCommitable = false
			lvl = slog.LevelError
		}

		s.logger.Log(context.TODO(), lvl, "deleting bucket", slogutil.KeyError, delErr)
	}

	return true, 0
}

// periodicFlush checks and flushes the unit to the database if the freshly
// generated unit ID differs from the current's ID.  Flushing process includes:
//   - swapping the current unit with the new empty one;
//   - writing the current unit to the database;
//   - removing the stale unit from the database.
func (s *StatsCtx) periodicFlush() {
	for cont, sleepFor := true, time.Duration(0); cont; time.Sleep(sleepFor) {
		cont, sleepFor = s.flush()
	}

	s.logger.Debug("periodic flushing finished")
}

// setLimit sets the limit.  s.lock is expected to be locked.
//
// TODO(s.chzhen):  Remove it when migration to the new API is over.
func (s *StatsCtx) setLimit(limit time.Duration) {
	if limit != 0 {
		s.enabled = true
		s.limit = limit
		s.logger.Debug("setting limit in days", "num", limit/timeutil.Day)

		return
	}

	s.enabled = false
	s.logger.Debug("disabled")

	if err := s.clear(); err != nil {
		s.logger.Error("clearing", slogutil.KeyError, err)
	}
}

// Reset counters and clear database
func (s *StatsCtx) clear() (err error) {
	defer func() { err = errors.Annotate(err, "clearing: %w") }()

	db := s.db.Swap(nil)
	if db != nil {
		var tx *bbolt.Tx
		tx, err = db.Begin(true)
		if err != nil {
			s.logger.Error("opening transaction", slogutil.KeyError, err)
		} else if err = finishTxn(tx, false); err != nil {
			// Don't wrap the error since it's informative enough as is.
			return err
		}

		// Active transactions will continue using database, but new ones won't
		// be created.
		err = db.Close()
		if err != nil {
			return fmt.Errorf("closing database: %w", err)
		}

		// All active transactions are now closed.
		s.logger.Debug("database closed")
	}

	err = os.Remove(s.filename)
	if err != nil {
		s.logger.Error("removing", slogutil.KeyError, err)
	}

	err = s.openDB()
	if err != nil {
		s.logger.Error("opening database", slogutil.KeyError, err)
	}

	// Use defer to unlock the mutex as soon as possible.
	defer s.logger.Debug("cleared")

	s.currMu.Lock()
	defer s.currMu.Unlock()

	s.curr = newUnit(s.unitIDGen())

	return nil
}

// loadUnits returns stored units from the database and current unit ID.
func (s *StatsCtx) loadUnits(limit uint32) (units []*unitDB, curID uint32) {
	db := s.db.Load()
	if db == nil {
		return nil, 0
	}

	// Use writable transaction to ensure any ongoing writable transaction is
	// taken into account.
	tx, err := db.Begin(true)
	if err != nil {
		s.logger.Error("opening transaction", slogutil.KeyError, err)

		return nil, 0
	}

	s.currMu.RLock()
	defer s.currMu.RUnlock()

	cur := s.curr

	if cur != nil {
		curID = cur.id
	} else {
		curID = s.unitIDGen()
	}

	// Per-hour units.
	units = make([]*unitDB, 0, limit)
	firstID := curID - limit + 1
	for i := firstID; i != curID; i++ {
		u := s.loadUnitFromDB(tx, i)
		if u == nil {
			u = &unitDB{NResult: make([]uint64, resultLast)}
		}
		units = append(units, u)
	}

	err = finishTxn(tx, false)
	if err != nil {
		s.logger.Error("finishing transaction", slogutil.KeyError, err)
	}

	if cur != nil {
		units = append(units, cur.serialize())
	}

	if unitsLen := len(units); unitsLen != int(limit) {
		// Should not happen.
		panic(fmt.Errorf("loaded %d units when the desired number is %d", unitsLen, limit))
	}

	return units, curID
}

// ShouldCount returns true if request for the host should be counted.
func (s *StatsCtx) ShouldCount(host string, _, _ uint16, ids []string) bool {
	s.confMu.RLock()
	defer s.confMu.RUnlock()

	if !s.shouldCountClient(ids) {
		return false
	}

	return !s.isIgnored(host)
}

// isIgnored returns true if the host is in the ignored domains list.  It
// assumes that s.confMu is locked for reading.
func (s *StatsCtx) isIgnored(host string) bool {
	return s.ignored.Has(host)
}
