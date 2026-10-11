package dnsforward

import (
	"reflect"
	"sync"
	"unsafe"

	"github.com/AdguardTeam/dnsproxy/proxy"
	glcache "github.com/AdguardTeam/golibs/cache"
	"github.com/AdguardTeam/golibs/errors"
)

// errCacheLayout is returned when the internal layout of the dnsproxy cache
// isn't the one expected by [proxyCacheStats].
const errCacheLayout errors.Error = "unexpected dnsproxy cache layout"

// cacheStats is the snapshot of the current state of the DNS cache.
type cacheStats struct {
	// entries is the number of items currently stored in the cache.
	entries uint64

	// size is the total size of keys and values currently stored in the cache,
	// in bytes.
	size uint64
}

// add adds the statistics of a single cache to cs.  cs must not be nil.
func (cs *cacheStats) add(st glcache.Stats) {
	cs.entries += uint64(max(st.Count, 0))
	cs.size += uint64(max(st.Size, 0))
}

// proxyCacheStats returns the current state of the general and the subnet
// caches of p.  It returns a zero cacheStats if p has no cache.  p must not be
// nil.
//
// Packages reflect and unsafe are used here, because dnsproxy doesn't provide
// any API to inspect its cache, so its unexported fields have to be read
// directly.  The layout is checked by tests, so a dnsproxy update that changes
// it is detected early.  This is also the reason this file is excluded from
// the blocklisted imports check in go-lint.sh.
func proxyCacheStats(p *proxy.Proxy) (cs cacheStats, err error) {
	cacheField := reflect.ValueOf(p).Elem().FieldByName("cache")
	if !cacheField.IsValid() || cacheField.Kind() != reflect.Pointer {
		return cacheStats{}, errCacheLayout
	} else if cacheField.IsNil() {
		return cacheStats{}, nil
	}

	c := cacheField.Elem()

	general, err := lockedCacheStats(c, "items", "itemsLock")
	if err != nil {
		return cacheStats{}, err
	}

	subnet, err := lockedCacheStats(c, "itemsWithSubnet", "itemsWithSubnetLock")
	if err != nil {
		return cacheStats{}, err
	}

	cs.add(general)
	cs.add(subnet)

	return cs, nil
}

// lockedCacheStats returns the statistics of the [glcache.Cache] stored in the
// field of c named itemsName, which is protected by the [*sync.RWMutex] stored
// in the field named lockName.  It returns zero statistics if the cache is nil.
// c must be an addressable struct value.
func lockedCacheStats(
	c reflect.Value,
	itemsName string,
	lockName string,
) (st glcache.Stats, err error) {
	items, ok := fieldValue[glcache.Cache](c, itemsName)
	if !ok {
		return glcache.Stats{}, errCacheLayout
	} else if items == nil {
		return glcache.Stats{}, nil
	}

	mu, ok := fieldValue[*sync.RWMutex](c, lockName)
	if !ok || mu == nil {
		return glcache.Stats{}, errCacheLayout
	}

	// Use the write lock, since dnsproxy removes expired items under the read
	// lock and [glcache.Cache.Stats] reads the items without synchronization.
	mu.Lock()
	defer mu.Unlock()

	return items.Stats(), nil
}

// fieldValue returns the value of the possibly unexported field of the
// addressable struct value v with the given name.  ok is false if there is no
// such field or it has a type other than T.
func fieldValue[T any](v reflect.Value, name string) (val T, ok bool) {
	f := v.FieldByName(name)
	if !f.IsValid() || !f.CanAddr() || f.Type() != reflect.TypeFor[T]() {
		var zero T

		return zero, false
	}

	// #nosec G103 -- The field is addressable and its type has been checked
	// above, so the pointer is valid and of the right type.
	return *(*T)(unsafe.Pointer(f.UnsafeAddr())), true
}
