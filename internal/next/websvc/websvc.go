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
	"io/fs"
	"log/slog"
	"net/http"
	"net/netip"
	"runtime"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/next/agh"
	"github.com/AdguardTeam/AdGuardHome/internal/next/dnssvc"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/netutil/httputil"
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
	logger       *slog.Logger
	confMgr      ConfigManager
	frontend     fs.FS
	tls          *tls.Config
	pprof        *server
	start        time.Time
	overrideAddr netip.AddrPort
	servers      []*server
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
		logger:       c.Logger,
		confMgr:      c.ConfigManager,
		frontend:     c.Frontend,
		tls:          c.TLS,
		start:        c.Start,
		overrideAddr: c.OverrideAddress,
		timeout:      c.Timeout,
		forceHTTPS:   c.ForceHTTPS,
	}

	mux := http.NewServeMux()
	svc.route(mux)

	if svc.overrideAddr != (netip.AddrPort{}) {
		svc.servers = []*server{newServer(svc.logger, svc.overrideAddr, nil, mux, c.Timeout)}
	} else {
		for _, a := range c.Addresses {
			svc.servers = append(svc.servers, newServer(svc.logger, a, nil, mux, c.Timeout))
		}

		for _, a := range c.SecureAddresses {
			svc.servers = append(svc.servers, newServer(svc.logger, a, c.TLS, mux, c.Timeout))
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
	httputil.RoutePprof(pprofMux)

	svc.pprofPort = c.Port
	addr := netip.AddrPortFrom(netip.AddrFrom4([4]byte{127, 0, 0, 1}), c.Port)

	svc.pprof = newServer(svc.logger, addr, nil, pprofMux, 10*time.Minute)
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
		addrPort := netutil.NetAddrToAddrPort(srv.localAddr())
		if addrPort == (netip.AddrPort{}) {
			continue
		}

		if srv.tlsConf == nil {
			addrs = append(addrs, addrPort)
		} else {
			secureAddrs = append(secureAddrs, addrPort)
		}
	}

	return addrs, secureAddrs
}

// type check
var _ agh.ServiceWithConfig[*Config] = (*Service)(nil)

// Start implements the [agh.Service] interface for *Service.  svc may be nil.
// After Start exits, all HTTP servers have tried to start, possibly failing and
// writing error messages to the log.
//
// TODO(a.garipov):  Use the context for cancelation as well.
func (svc *Service) Start(ctx context.Context) (err error) {
	if svc == nil {
		return nil
	}

	svc.logger.InfoContext(ctx, "starting")
	defer svc.logger.InfoContext(ctx, "started")

	for _, srv := range svc.servers {
		go srv.serve(ctx, svc.logger)
	}

	if svc.pprof != nil {
		go svc.pprof.serve(ctx, svc.logger)
	}

	return svc.wait(ctx)
}

// wait waits until either the context is canceled or all servers have started.
func (svc *Service) wait(ctx context.Context) (err error) {
	for !svc.serversHaveStarted() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Wait and let the other goroutines do their job.
			runtime.Gosched()
		}
	}

	return nil
}

// serversHaveStarted returns true if all servers have started serving.
func (svc *Service) serversHaveStarted() (started bool) {
	started = len(svc.servers) != 0
	for _, srv := range svc.servers {
		started = started && srv.localAddr() != nil
	}

	if svc.pprof != nil {
		started = started && svc.pprof.localAddr() != nil
	}

	return started
}

// Shutdown implements the [agh.Service] interface for *Service.  svc may be
// nil.
func (svc *Service) Shutdown(ctx context.Context) (err error) {
	if svc == nil {
		return nil
	}

	svc.logger.InfoContext(ctx, "shutting down")
	defer svc.logger.InfoContext(ctx, "shut down")

	defer func() { err = errors.Annotate(err, "shutting down: %w") }()

	var errs []error
	for _, srv := range svc.servers {
		shutdownErr := srv.shutdown(ctx)
		if shutdownErr != nil {
			// Don't wrap the error, because it's informative enough as is.
			errs = append(errs, err)
		}
	}

	if svc.pprof != nil {
		shutdownErr := svc.pprof.shutdown(ctx)
		if shutdownErr != nil {
			errs = append(errs, fmt.Errorf("pprof: %w", shutdownErr))
		}
	}

	return errors.Join(errs...)
}
