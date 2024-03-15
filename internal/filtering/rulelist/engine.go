package rulelist

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/urlfilter"
	"github.com/AdguardTeam/urlfilter/filterlist"
	"github.com/c2h5oh/datasize"
)

// Engine is a single DNS filter based on one or more rule lists.  This
// structure contains the filtering engine combining several rule lists.
//
// TODO(a.garipov): Merge with [TextEngine] in some way?
type Engine struct {
	// mu protects engine and storage.
	//
	// TODO(a.garipov): See if anything else should be protected.
	mu *sync.RWMutex

	// engine is the filtering engine.
	engine *urlfilter.DNSEngine

	// storage is the filtering-rule storage.  It is saved here to close it.
	storage *filterlist.RuleStorage

	// name is the human-readable name of the engine, like "allowed", "blocked",
	// or "custom".
	name string

	// filters is the data about rule filters in this engine.
	filters []*Filter
}

// EngineConfig is the configuration for rule-list filtering engines created by
// combining refreshable filters.
type EngineConfig struct {
	// Name is the human-readable name of this engine, like "allowed",
	// "blocked", or "custom".
	Name string

	// Filters is the data about rule lists in this engine.  There must be no
	// other references to the elements of this slice.
	Filters []*Filter
}

// NewEngine returns a new rule-list filtering engine.  The engine is not
// refreshed, so a refresh should be performed before use.
func NewEngine(c *EngineConfig) (e *Engine) {
	return &Engine{
		mu:      &sync.RWMutex{},
		name:    c.Name,
		filters: c.Filters,
	}
}

// Close closes the underlying rule-list engine as well as the rule lists.
func (e *Engine) Close() (err error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.storage == nil {
		return nil
	}

	err = e.storage.Close()
	if err != nil {
		return fmt.Errorf("closing engine %q: %w", e.name, err)
	}

	return nil
}

// FilterRequest returns the result of filtering req using the DNS filtering
// engine.
func (e *Engine) FilterRequest(
	req *urlfilter.DNSRequest,
) (res *urlfilter.DNSResult, hasMatched bool) {
	return e.currentEngine().MatchRequest(req)
}

// currentEngine returns the current filtering engine.
func (e *Engine) currentEngine() (enging *urlfilter.DNSEngine) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.engine
}

// Refresh updates all rule lists in e.  ctx is used for cancellation.
// parseBuf, cli, cacheDir, and maxSize are used for updates of rule-list
// filters; see [Filter.Refresh].
//
// TODO(a.garipov): Unexport and test in an internal test or through enigne
// tests.
func (e *Engine) Refresh(
	ctx context.Context,
	parseBuf []byte,
	cli *http.Client,
	cacheDir string,
	maxSize datasize.ByteSize,
) (err error) {
	defer func() { err = errors.Annotate(err, "updating engine %q: %w", e.name) }()

	var filtersToRefresh []*Filter
	for _, f := range e.filters {
		if f.enabled {
			filtersToRefresh = append(filtersToRefresh, f)
		}
	}

	if len(filtersToRefresh) == 0 {
		log.Info("filtering: updating engine %q: no rule-list filters", e.name)

		return nil
	}

	engRefr := &engineRefresh{
		httpCli:    cli,
		cacheDir:   cacheDir,
		engineName: e.name,
		parseBuf:   parseBuf,
		maxSize:    maxSize,
	}

	ruleLists, errs := engRefr.process(ctx, e.filters)
	if isOneTimeoutError(errs) {
		// Don't wrap the error since it's informative enough as is.
		return err
	}

	storage, err := filterlist.NewRuleStorage(ruleLists)
	if err != nil {
		errs = append(errs, fmt.Errorf("creating rule storage: %w", err))

		return errors.Join(errs...)
	}

	e.resetStorage(storage)

	return errors.Join(errs...)
}

// resetStorage sets e.storage and e.engine and closes the previous storage.
// Errors from closing the previous storage are logged.
func (e *Engine) resetStorage(storage *filterlist.RuleStorage) {
	e.mu.Lock()
	defer e.mu.Unlock()

	prevStorage := e.storage
	e.storage, e.engine = storage, urlfilter.NewDNSEngine(storage)

	if prevStorage == nil {
		return
	}

	err := prevStorage.Close()
	if err != nil {
		log.Error("filtering: engine %q: closing old storage: %s", e.name, err)
	}
}

// isOneTimeoutError returns true if the sole error in errs is either
// [context.Canceled] or [context.DeadlineExceeded].
func isOneTimeoutError(errs []error) (ok bool) {
	if len(errs) != 1 {
		return false
	}

	err := errs[0]

	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

// engineRefresh represents a single ongoing engine refresh.
type engineRefresh struct {
	httpCli    *http.Client
	cacheDir   string
	engineName string
	parseBuf   []byte
	maxSize    datasize.ByteSize
}

// process runs updates of all given rule-list filters.  All errors are logged
// as they appear, since the update can take a significant amount of time.
// errs contains all errors that happened during the update, unless the context
// is canceled or its deadline is reached, in which case errs will only contain
// a single timeout error.
//
// TODO(a.garipov): Think of a better way to communicate the timeout condition?
func (r *engineRefresh) process(
	ctx context.Context,
	filters []*Filter,
) (ruleLists []filterlist.RuleList, errs []error) {
	ruleLists = make([]filterlist.RuleList, 0, len(filters))
	for i, f := range filters {
		select {
		case <-ctx.Done():
			return nil, []error{fmt.Errorf("timeout after updating %d filters: %w", i, ctx.Err())}
		default:
			// Go on.
		}

		err := r.processFilter(ctx, f)
		if err == nil {
			ruleLists = append(ruleLists, f.ruleList)

			continue
		}

		errs = append(errs, err)

		// Also log immediately, since the update can take a lot of time.
		log.Error(
			"filtering: updating engine %q: rule list %s from url %q: %s\n",
			r.engineName,
			f.uid,
			f.url,
			err,
		)
	}

	return ruleLists, errs
}

// processFilter runs an update of a single rule-list filter.
func (r *engineRefresh) processFilter(ctx context.Context, f *Filter) (err error) {
	prevChecksum := f.checksum
	parseRes, err := f.Refresh(ctx, r.parseBuf, r.httpCli, r.cacheDir, r.maxSize)
	if err != nil {
		return fmt.Errorf("updating %s: %w", f.uid, err)
	}

	if prevChecksum == parseRes.Checksum {
		log.Info("filtering: engine %q: filter %q: no change", r.engineName, f.uid)

		return nil
	}

	log.Info(
		"filtering: updated engine %q: filter %q: %d bytes, %d rules",
		r.engineName,
		f.uid,
		parseRes.BytesWritten,
		parseRes.RulesCount,
	)

	return nil
}
