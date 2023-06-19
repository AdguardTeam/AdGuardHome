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
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/next/agh"
	"github.com/AdguardTeam/AdGuardHome/internal/next/dnssvc"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	httptreemux "github.com/dimfeld/httptreemux/v5"
)

// ConfigManager is the configuration manager interface.
type ConfigManager interface {
	DNS() (svc agh.ServiceWithConfig[*dnssvc.Config])
	Web() (svc agh.ServiceWithConfig[*Config])

	UpdateDNS(ctx context.Context, c *dnssvc.Config) (err error)
	UpdateWeb(ctx context.Context, c *Config) (err error)
}

// Config is the AdGuard Home web service configuration structure.
type Config struct {
	// ConfigManager is used to show information about services as well as
	// dynamically reconfigure them.
	ConfigManager ConfigManager

	// Frontend is the filesystem with the frontend and other statically
	// compiled files.
	Frontend fs.FS

	// TLS is the optional TLS configuration.  If TLS is not nil,
	// SecureAddresses must not be empty.
	TLS *tls.Config

	// Start is the time of start of AdGuard Home.
	Start time.Time

	// Addresses are the addresses on which to serve the plain HTTP API.
	Addresses []netip.AddrPort

	// SecureAddresses are the addresses on which to serve the HTTPS API.  If
	// SecureAddresses is not empty, TLS must not be nil.
	SecureAddresses []netip.AddrPort

	// Timeout is the timeout for all server operations.
	Timeout time.Duration

	// ForceHTTPS tells if all requests to Addresses should be redirected to a
	// secure address instead.
	//
	// TODO(a.garipov): Use; define rules, which address to redirect to.
	ForceHTTPS bool
}

// Service is the AdGuard Home web service.  A nil *Service is a valid
// [agh.Service] that does nothing.
type Service struct {
	confMgr    ConfigManager
	frontend   fs.FS
	tls        *tls.Config
	start      time.Time
	servers    []*http.Server
	timeout    time.Duration
	forceHTTPS bool
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

	frontend, err := fs.Sub(c.Frontend, "build/static")
	if err != nil {
		return nil, fmt.Errorf("frontend fs: %w", err)
	}

	svc = &Service{
		confMgr:    c.ConfigManager,
		frontend:   frontend,
		tls:        c.TLS,
		start:      c.Start,
		timeout:    c.Timeout,
		forceHTTPS: c.ForceHTTPS,
	}

	mux := newMux(svc)

	for _, a := range c.Addresses {
		addr := a.String()
		errLog := log.StdLog("websvc: plain http: "+addr, log.ERROR)
		svc.servers = append(svc.servers, &http.Server{
			Addr:              addr,
			Handler:           mux,
			ErrorLog:          errLog,
			ReadTimeout:       c.Timeout,
			WriteTimeout:      c.Timeout,
			IdleTimeout:       c.Timeout,
			ReadHeaderTimeout: c.Timeout,
		})
	}

	for _, a := range c.SecureAddresses {
		addr := a.String()
		errLog := log.StdLog("websvc: https: "+addr, log.ERROR)
		svc.servers = append(svc.servers, &http.Server{
			Addr:              addr,
			Handler:           mux,
			TLSConfig:         c.TLS,
			ErrorLog:          errLog,
			ReadTimeout:       c.Timeout,
			WriteTimeout:      c.Timeout,
			IdleTimeout:       c.Timeout,
			ReadHeaderTimeout: c.Timeout,
		})
	}

	return svc, nil
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
	for _, srv := range svc.servers {
		addrPort, err := netip.ParseAddrPort(srv.Addr)
		if err != nil {
			// Technically shouldn't happen, since all servers must have a valid
			// address.
			panic(fmt.Errorf("websvc: server %q: bad address: %w", srv.Addr, err))
		}

		// srv.Serve will set TLSConfig to an almost empty value, so, instead of
		// relying only on the nilness of TLSConfig, check the length of the
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

	wg := &sync.WaitGroup{}
	wg.Add(len(svc.servers))
	for _, srv := range svc.servers {
		go serve(srv, wg)
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

	var errs []error
	for _, srv := range svc.servers {
		serr := srv.Shutdown(ctx)
		if serr != nil {
			errs = append(errs, fmt.Errorf("shutting down srv %s: %w", srv.Addr, serr))
		}
	}

	if len(errs) > 0 {
		return errors.List("shutting down", errs...)
	}

	return nil
}

// Config returns the current configuration of the web service.  Config must not
// be called simultaneously with Start.  If svc was initialized with ":0"
// addresses, addrs will not return the actual bound ports until Start is
// finished.
func (svc *Service) Config() (c *Config) {
	c = &Config{
		ConfigManager: svc.confMgr,
		TLS:           svc.tls,
		// Leave Addresses and SecureAddresses empty and get the actual
		// addresses that include the :0 ones later.
		Start:      svc.start,
		Timeout:    svc.timeout,
		ForceHTTPS: svc.forceHTTPS,
	}

	c.Addresses, c.SecureAddresses = svc.addrs()

	return c
}
