// Package websvc contains the AdGuard Home HTTP API service.
//
// NOTE: Packages other than cmd must not import this package, as it imports
// most other packages.
//
// TODO(a.garipov): Add tests.
package websvc

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/netip"
	"runtime"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/next/agh"
	"github.com/AdguardTeam/AdGuardHome/internal/next/dnssvc"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/mathutil"
	"github.com/AdguardTeam/golibs/pprofutil"
	httptreemux "github.com/dimfeld/httptreemux/v5"
)

// ConfigManager is the configuration manager interface.
type ConfigManager interface {
	DNS() (svc agh.ServiceWithConfig[*dnssvc.Config])
	Web() (svc agh.ServiceWithConfig[*Config])

	UpdateDNS(ctx context.Context, c *dnssvc.Config) (err error)
	UpdateWeb(ctx context.Context, c *Config) (err error)
}

// Service is the AdGuard Home web service.  A nil *Service is a valid
// [agh.Service] that does nothing.
type Service struct {
	confMgr      ConfigManager
	frontend     fs.FS
	tls          *tls.Config
	pprof        *http.Server
	start        time.Time
	overrideAddr netip.AddrPort
	servers      []*http.Server
	timeout      time.Duration
	pprofPort    uint16
	forceHTTPS   bool
}

// New returns a new properly initialized *Service.  If c is nil, svc is a nil
// *Service that does nothing.  The fields of c must not be modified after
// calling New.
//
// TODO(a.garipov): Get rid of this special handling of nil or explain it
// better.
func New(c *Config) (svc *Service, err error) {
	if c == nil {
		return nil, nil
	}

	svc = &Service{
		confMgr:      c.ConfigManager,
		frontend:     c.Frontend,
		tls:          c.TLS,
		start:        c.Start,
		overrideAddr: c.OverrideAddress,
		timeout:      c.Timeout,
		forceHTTPS:   c.ForceHTTPS,
	}

	mux := newMux(svc)

	if svc.overrideAddr != (netip.AddrPort{}) {
		svc.servers = []*http.Server{newSrv(svc.overrideAddr, nil, mux, c.Timeout)}
	} else {
		for _, a := range c.Addresses {
			svc.servers = append(svc.servers, newSrv(a, nil, mux, c.Timeout))
		}

		for _, a := range c.SecureAddresses {
			svc.servers = append(svc.servers, newSrv(a, c.TLS, mux, c.Timeout))
		}
	}

	svc.setupPprof(c.Pprof)

	return svc, nil
}

// setupPprof sets the pprof properties of svc.
func (svc *Service) setupPprof(c *PprofConfig) {
	if !c.Enabled {
		// Set to zero explicitly in case pprof used to be enabled before a
		// reconfiguration took place.
		runtime.SetBlockProfileRate(0)
		runtime.SetMutexProfileFraction(0)

		return
	}

	runtime.SetBlockProfileRate(1)
	runtime.SetMutexProfileFraction(1)

	pprofMux := http.NewServeMux()
	pprofutil.RoutePprof(pprofMux)

	svc.pprofPort = c.Port
	addr := netip.AddrPortFrom(netip.AddrFrom4([4]byte{127, 0, 0, 1}), c.Port)

	// TODO(a.garipov): Consider making pprof timeout configurable.
	svc.pprof = newSrv(addr, nil, pprofMux, 10*time.Minute)
}

// newSrv returns a new *http.Server with the given parameters.
func newSrv(
	addr netip.AddrPort,
	tlsConf *tls.Config,
	h http.Handler,
	timeout time.Duration,
) (srv *http.Server) {
	addrStr := addr.String()
	srv = &http.Server{
		Addr:              addrStr,
		Handler:           h,
		TLSConfig:         tlsConf,
		ReadTimeout:       timeout,
		WriteTimeout:      timeout,
		IdleTimeout:       timeout,
		ReadHeaderTimeout: timeout,
	}

	if tlsConf == nil {
		srv.ErrorLog = log.StdLog("websvc: plain http: "+addrStr, log.ERROR)
	} else {
		srv.ErrorLog = log.StdLog("websvc: https: "+addrStr, log.ERROR)
	}

	return srv
}

// newMux returns a new HTTP request multiplexer for the AdGuard Home web
// service.
func newMux(svc *Service) (mux *httptreemux.ContextMux) {
	mux = httptreemux.NewContextMux()

	routes := []struct {
		handler http.HandlerFunc
		method  string
		pattern string
		isJSON  bool
	}{{
		handler: svc.handleGetHealthCheck,
		method:  http.MethodGet,
		pattern: PathHealthCheck,
		isJSON:  false,
	}, {
		handler: http.FileServer(http.FS(svc.frontend)).ServeHTTP,
		method:  http.MethodGet,
		pattern: PathFrontend,
		isJSON:  false,
	}, {
		handler: http.FileServer(http.FS(svc.frontend)).ServeHTTP,
		method:  http.MethodGet,
		pattern: PathRoot,
		isJSON:  false,
	}, {
		handler: svc.handleGetSettingsAll,
		method:  http.MethodGet,
		pattern: PathV1SettingsAll,
		isJSON:  true,
	}, {
		handler: svc.handlePatchSettingsDNS,
		method:  http.MethodPatch,
		pattern: PathV1SettingsDNS,
		isJSON:  true,
	}, {
		handler: svc.handlePatchSettingsHTTP,
		method:  http.MethodPatch,
		pattern: PathV1SettingsHTTP,
		isJSON:  true,
	}, {
		handler: svc.handleGetV1SystemInfo,
		method:  http.MethodGet,
		pattern: PathV1SystemInfo,
		isJSON:  true,
	}}

	for _, r := range routes {
		var hdlr http.Handler
		if r.isJSON {
			hdlr = jsonMw(r.handler)
		} else {
			hdlr = r.handler
		}

		mux.Handle(r.method, r.pattern, logMw(hdlr))
	}

	return mux
}

// addrs returns all addresses on which this server serves the HTTP API.  addrs
// must not be called simultaneously with Start.  If svc was initialized with
// ":0" addresses, addrs will not return the actual bound ports until Start is
// finished.
func (svc *Service) addrs() (addrs, secureAddrs []netip.AddrPort) {
	if svc.overrideAddr != (netip.AddrPort{}) {
		return []netip.AddrPort{svc.overrideAddr}, nil
	}

	for _, srv := range svc.servers {
		// Use MustParseAddrPort, since no errors should technically happen
		// here, because all servers must have a valid address.
		addrPort := netip.MustParseAddrPort(srv.Addr)

		// [srv.Serve] will set TLSConfig to an almost empty value, so, instead
		// of relying only on the nilness of TLSConfig, check the length of the
		// certificates field as well.
		if srv.TLSConfig == nil || len(srv.TLSConfig.Certificates) == 0 {
			addrs = append(addrs, addrPort)
		} else {
			secureAddrs = append(secureAddrs, addrPort)
		}
	}

	return addrs, secureAddrs
}

// handleGetHealthCheck is the handler for the GET /health-check HTTP API.
func (svc *Service) handleGetHealthCheck(w http.ResponseWriter, _ *http.Request) {
	_, _ = io.WriteString(w, "OK")
}

// type check
var _ agh.Service = (*Service)(nil)

// Start implements the [agh.Service] interface for *Service.  svc may be nil.
// After Start exits, all HTTP servers have tried to start, possibly failing and
// writing error messages to the log.
func (svc *Service) Start() (err error) {
	if svc == nil {
		return nil
	}

	pprofEnabled := svc.pprof != nil
	srvNum := len(svc.servers) + mathutil.BoolToNumber[int](pprofEnabled)

	wg := &sync.WaitGroup{}
	wg.Add(srvNum)
	for _, srv := range svc.servers {
		go serve(srv, wg)
	}

	if pprofEnabled {
		go serve(svc.pprof, wg)
	}

	wg.Wait()

	return nil
}

// serve starts and runs srv and writes all errors into its log.
func serve(srv *http.Server, wg *sync.WaitGroup) {
	addr := srv.Addr
	defer log.OnPanic(addr)

	var proto string
	var l net.Listener
	var err error
	if srv.TLSConfig == nil {
		proto = "http"
		l, err = net.Listen("tcp", addr)
	} else {
		proto = "https"
		l, err = tls.Listen("tcp", addr, srv.TLSConfig)
	}
	if err != nil {
		srv.ErrorLog.Printf("starting srv %s: binding: %s", addr, err)
	}

	// Update the server's address in case the address had the port zero, which
	// would mean that a random available port was automatically chosen.
	srv.Addr = l.Addr().String()

	log.Info("websvc: starting srv %s://%s", proto, srv.Addr)

	l = &waitListener{
		Listener:      l,
		firstAcceptWG: wg,
	}

	err = srv.Serve(l)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		srv.ErrorLog.Printf("starting srv %s: %s", addr, err)
	}
}

// Shutdown implements the [agh.Service] interface for *Service.  svc may be
// nil.
func (svc *Service) Shutdown(ctx context.Context) (err error) {
	if svc == nil {
		return nil
	}

	defer func() { err = errors.Annotate(err, "shutting down: %w") }()

	var errs []error
	for _, srv := range svc.servers {
		shutdownErr := srv.Shutdown(ctx)
		if shutdownErr != nil {
			errs = append(errs, fmt.Errorf("srv %s: %w", srv.Addr, shutdownErr))
		}
	}

	if svc.pprof != nil {
		shutdownErr := svc.pprof.Shutdown(ctx)
		if shutdownErr != nil {
			errs = append(errs, fmt.Errorf("pprof srv %s: %w", svc.pprof.Addr, shutdownErr))
		}
	}

	return errors.Join(errs...)
}
