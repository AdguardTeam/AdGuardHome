package filtering

import (
	"context"
	"fmt"
	"hash"
	"hash/maphash"
	"io"
	"os"
	"runtime/debug"
	"sync"

	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/syncutil"
	"github.com/AdguardTeam/urlfilter/filterlist"
)

// rebuildGCPercent is the garbage-collection target percentage used while the
// filtering engines are being rebuilt, see [debug.SetGCPercent].
//
// A rebuild keeps the previous engines alive until the new ones are ready, so
// the live heap roughly doubles for its duration.  With the default target of
// 100% the heap is only collected once it has grown to twice the live heap, so
// the transient peak of a rebuild is about three times the memory that a
// single set of engines takes.
//
// Ported from https://github.com/AdguardTeam/AdGuardHome/pull/8498.
const rebuildGCPercent = 20

// gcTarget serializes the changes of the garbage-collection target between the
// rebuilds that are running at the same time.
type gcTarget struct {
	set func(percent int) (prev int)

	mu sync.Mutex

	depth int
	prev  int
}

// rebuildGCTarget is the garbage-collection target shared by every [DNSFilter]
// within the process.
var rebuildGCTarget = &gcTarget{set: debug.SetGCPercent}

// freeOSMemory collects the heap and returns as much of it to the OS as it can.
var freeOSMemory = debug.FreeOSMemory

// tighten lowers the garbage-collection target to percent for the duration of a
// rebuild and returns the function that undoes it, which must be called exactly
// once.  The target is only ever tightened.
func (t *gcTarget) tighten(percent int) (restore func()) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.depth == 0 {
		prev := t.set(percent)
		if prev < percent {
			t.set(prev)

			return func() {}
		}

		t.prev = prev
	}

	t.depth++

	return t.release
}

// release undoes a single [gcTarget.tighten].
func (t *gcTarget) release() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.depth--
	if t.depth == 0 {
		t.set(t.prev)
	}
}

// fingerprint is a fingerprint of the rule lists that a pair of filtering
// engines is built from.
type fingerprint = uint64

// fingerprintSeed is drawn once per process so fingerprints cannot be targeted
// from the outside.
var fingerprintSeed = maphash.MakeSeed()

// filtersFingerprint returns the fingerprint of the given rule lists.
func filtersFingerprint(allowFilters, blockFilters []Filter) (f fingerprint) {
	h := &maphash.Hash{}
	h.SetSeed(fingerprintSeed)

	hashFilters(h, blockFilters)
	_, _ = h.Write([]byte{0})
	hashFilters(h, allowFilters)

	return h.Sum64()
}

const fingerprintBufSize = 32 * 1024

var fingerprintBufPool = syncutil.NewSlicePool[byte](fingerprintBufSize)

// onlyReader hides methods other than [io.Reader.Read].
type onlyReader struct {
	io.Reader
}

// hashFilters writes the data identifying filters into h.
func hashFilters(h hash.Hash, filters []Filter) {
	for _, f := range filters {
		_, _ = fmt.Fprintf(h, "id:%d;path:%q;data:%d;", f.ID, f.FilePath, len(f.Data))
		_, _ = h.Write(f.Data)
		hashFileContents(h, f.FilePath)
	}
}

// hashFileContents writes the current contents of the file at path into h.
func hashFileContents(h hash.Hash, path string) {
	if path == "" {
		return
	}

	f, err := os.Open(path)
	if err != nil {
		_, _ = fmt.Fprintf(h, "err:%v;", err)

		return
	}
	defer func() { _ = f.Close() }()

	bufPtr := fingerprintBufPool.Get()
	defer fingerprintBufPool.Put(bufPtr)

	n, err := io.CopyBuffer(h, onlyReader{Reader: f}, *bufPtr)
	if err != nil {
		_, _ = fmt.Fprintf(h, "readerr:%v;", err)

		return
	}

	_, _ = fmt.Fprintf(h, "size:%d;", n)
}

// closeStorage closes rs and logs the error, if any.  rs may be nil.
func (d *DNSFilter) closeStorage(
	ctx context.Context,
	rs *filterlist.RuleStorage,
	name string,
) {
	if rs == nil {
		return
	}

	err := rs.Close()
	if err != nil {
		d.logger.ErrorContext(ctx, "closing "+name, slogutil.KeyError, err)
	}
}

// closeRuleLists closes each rule list in lists, ignoring errors.
func closeRuleLists(lists []filterlist.Interface) {
	for _, l := range lists {
		_ = l.Close()
	}
}
