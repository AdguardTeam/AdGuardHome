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

// proxyCacheStats returns the current state of the general and the subnet
// caches of p.  It returns a zero cacheStats if p has no cache.  p must not be
// nil.
//
// dnsproxy doesn't provide any API to inspect its cache, so the unexported
// fields are accessed using reflection.  The layout is checked by tests, so a
// dnsproxy update that changes it is detected early.  This is the only reason
// this file is excluded from the blocklisted imports check in go-lint.sh.
//
// TODO:  Use the public API when dnsproxy provides one.
func proxyCacheStats(p *proxy.Proxy) (cs cacheStats, err error) {
	cacheField := reflect.ValueOf(p).Elem().FieldByName("cache")
	if !cacheField.IsValid() || cacheField.Kind() != reflect.Pointer {
		return cacheStats{}, errCacheLayout
	} else if cacheField.IsNil() {
		return cacheStats{}, nil
	}

	c := cacheField.Elem()
	for _, names := range [...][2]string{
		{"items", "itemsLock"},
		{"itemsWithSubnet", "itemsWithSubnetLock"},
	} {
		var st glcache.Stats
		st, err = lockedCacheStats(c, names[0], names[1])
		if err != nil {
			return cacheStats{}, err
		}

		cs.entries += uint64(max(st.Count, 0))
		cs.size += uint64(max(st.Size, 0))
	}

	return cs, nil
}

// lockedCacheStats returns the statistics of the [glcache.Cache] stored in the
// field of c named itemsName, which is protected by the [*sync.RWMutex] stored
// in the field named lockName.  It returns zero statistics if the cache is nil.
func lockedCacheStats(c reflect.Value, itemsName, lockName string) (st glcache.Stats, err error) {
	items, ok := exportedField[glcache.Cache](c, itemsName)
	if !ok {
		return glcache.Stats{}, errCacheLayout
	} else if items == nil {
		return glcache.Stats{}, nil
	}

	mu, ok := exportedField[*sync.RWMutex](c, lockName)
	if !ok || mu == nil {
		return glcache.Stats{}, errCacheLayout
	}

	// Use the write lock, since dnsproxy removes expired items under the read
	// lock and [glcache.Cache.Stats] reads the items without synchronization.
	mu.Lock()
	defer mu.Unlock()

	return items.Stats(), nil
}

// exportedField returns the value of the possibly unexported field of the
// addressable struct value v with the given name.  ok is false if there is no
// such field or it has a type other than T.
func exportedField[T any](v reflect.Value, name string) (val T, ok bool) {
	f := v.FieldByName(name)
	if !f.IsValid() || !f.CanAddr() || f.Type() != reflect.TypeFor[T]() {
		return val, false
	}

	// #nosec G103 -- The field is addressable and its type has been checked
	// above, so the pointer is valid and of the right type.
	return *(*T)(unsafe.Pointer(f.UnsafeAddr())), true
}
