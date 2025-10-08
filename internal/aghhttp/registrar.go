package aghhttp

import (
	"net/http"
	"sync"
)

// Registrar registers an HTTP handler for a method and path.
type Registrar interface {
	Register(method, path string, h http.HandlerFunc)
}

// DeferredRegistrar is an implementation of [Registrar] that queues handler
// registrations until Bind is called.
type DeferredRegistrar struct {
	mu         *sync.Mutex
	registerFn RegisterFunc
	queue      []item
}

// item is an entry in the [DeferredRegistrar] queue.
type item struct {
	handlerFn http.HandlerFunc
	method    string
	path      string
}

// NewDeferredRegistrar returns a new properly initialized *DeferredRegistrar.
func NewDeferredRegistrar() (r *DeferredRegistrar) {
	return &DeferredRegistrar{
		mu: &sync.Mutex{},
	}
}

// type check
var _ Registrar = (*DeferredRegistrar)(nil)

// Register implements the [Registrar] interface.
func (r *DeferredRegistrar) Register(method, path string, h http.HandlerFunc) {
	var fn RegisterFunc
	defer func() {
		if fn != nil {
			fn(method, path, h)
		}
	}()

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.registerFn == nil {
		r.queue = append(r.queue, item{
			handlerFn: h,
			method:    method,
			path:      path,
		})

		return
	}

	fn = r.registerFn
}

// Bind registers queued HTTP handlers with fn and uses fn for future
// registrations.
func (r *DeferredRegistrar) Bind(fn RegisterFunc) {
	var q []item

	func() {
		r.mu.Lock()
		defer r.mu.Unlock()

		q = r.queue
		r.queue = nil
		r.registerFn = fn
	}()

	for _, it := range q {
		fn(it.method, it.path, it.handlerFn)
	}
}
