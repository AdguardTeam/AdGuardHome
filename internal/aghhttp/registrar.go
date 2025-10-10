package aghhttp

import (
	"net/http"
)

// Registrar registers an HTTP handler for a method and path.
//
// TODO(s.chzhen):  Implement [httputil.Router].
type Registrar interface {
	Register(method, path string, h http.HandlerFunc)
}

// EmptyRegistrar is an implementation of [Registrar] that does nothing.
type EmptyRegistrar struct{}

// type check
var _ Registrar = EmptyRegistrar{}

// Register implements the [Registrar] interface.
func (EmptyRegistrar) Register(_, _ string, _ http.HandlerFunc) {}

// wrapFunc composes an HTTP handler for a route.
type wrapFunc func(method, path string, h http.HandlerFunc) (wrapped http.Handler)

// DefaultRegistrar is an implementation of [Registrar] that registers handlers
// after applying a user-provided wrap function.
type DefaultRegistrar struct {
	mux  *http.ServeMux
	wrap wrapFunc
}

// NewDefaultRegistrar returns a new properly initialized *DefaultRegistrar.
// mux and wrap must not be nil.
func NewDefaultRegistrar(mux *http.ServeMux, wrap wrapFunc) (r *DefaultRegistrar) {
	return &DefaultRegistrar{
		mux:  mux,
		wrap: wrap,
	}
}

// type check
var _ Registrar = (*DefaultRegistrar)(nil)

// Register implements the [Registrar] interface.
func (r *DefaultRegistrar) Register(method, path string, h http.HandlerFunc) {
	wrapped := r.wrap(method, path, h)
	r.mux.Handle(path, wrapped)
}
