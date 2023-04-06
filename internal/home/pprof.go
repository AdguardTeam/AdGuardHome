package home

import (
	"net/http"
	"net/http/pprof"
	"runtime"

	"github.com/AdguardTeam/golibs/log"
)

// startPprof launches the debug and profiling server on addr.
func startPprof(addr string) {
	runtime.SetBlockProfileRate(1)
	runtime.SetMutexProfileFraction(1)

	mux := http.NewServeMux()

	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	// See profileSupportsDelta in src/net/http/pprof/pprof.go.
	mux.Handle("/debug/pprof/allocs", pprof.Handler("allocs"))
	mux.Handle("/debug/pprof/block", pprof.Handler("block"))
	mux.Handle("/debug/pprof/goroutine", pprof.Handler("goroutine"))
	mux.Handle("/debug/pprof/heap", pprof.Handler("heap"))
	mux.Handle("/debug/pprof/mutex", pprof.Handler("mutex"))
	mux.Handle("/debug/pprof/threadcreate", pprof.Handler("threadcreate"))

	go func() {
		defer log.OnPanic("pprof server")

		log.Info("pprof: listening on %q", addr)
		err := http.ListenAndServe(addr, mux)
		log.Info("pprof server errors: %v", err)
	}()
}
