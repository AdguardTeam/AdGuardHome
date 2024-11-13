package websvc

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"sync"
	"time"

	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/netutil/urlutil"
)

// server contains an *http.Server as well as entities and data associated with
// it.
//
// TODO(a.garipov):  Join with similar structs in other projects and move to
// golibs/netutil/httputil.
//
// TODO(a.garipov):  Once the above standardization is complete, consider
// merging debugsvc and websvc into a single httpsvc.
type server struct {
	// mu protects http, logger, tcpListener, and url.
	mu          *sync.Mutex
	http        *http.Server
	logger      *slog.Logger
	tcpListener *net.TCPListener
	url         *url.URL

	tlsConf     *tls.Config
	initialAddr netip.AddrPort
}

// loggerKeyServer is the key used by [server] to identify itself.
const loggerKeyServer = "server"

// newServer returns a *server that is ready to serve HTTP queries.  The TCP
// listener is not started.  handler must not be nil.
func newServer(
	baseLogger *slog.Logger,
	initialAddr netip.AddrPort,
	tlsConf *tls.Config,
	handler http.Handler,
	timeout time.Duration,
) (s *server) {
	u := &url.URL{
		Scheme: urlutil.SchemeHTTP,
		Host:   initialAddr.String(),
	}

	if tlsConf != nil {
		u.Scheme = urlutil.SchemeHTTPS
	}

	logger := baseLogger.With(loggerKeyServer, u)

	return &server{
		mu: &sync.Mutex{},
		http: &http.Server{
			Handler:           handler,
			ReadTimeout:       timeout,
			ReadHeaderTimeout: timeout,
			WriteTimeout:      timeout,
			IdleTimeout:       timeout,
			ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
		},
		logger: logger,
		url:    u,

		tlsConf:     tlsConf,
		initialAddr: initialAddr,
	}
}

// localAddr returns the local address of the server if the server has started
// listening; otherwise, it returns nil.
func (s *server) localAddr() (addr net.Addr) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if l := s.tcpListener; l != nil {
		return l.Addr()
	}

	return nil
}

// serve starts s.  baseLogger is used as a base logger for s.  If s fails to
// serve with anything other than [http.ErrServerClosed], it causes an unhandled
// panic.  It is intended to be used as a goroutine.
//
// TODO(a.garipov):  Improve error handling.
func (s *server) serve(ctx context.Context, baseLogger *slog.Logger) {
	l, err := net.ListenTCP("tcp", net.TCPAddrFromAddrPort(s.initialAddr))
	if err != nil {
		s.logger.ErrorContext(ctx, "listening tcp", slogutil.KeyError, err)

		panic(fmt.Errorf("websvc: listening tcp: %w", err))
	}

	func() {
		s.mu.Lock()
		defer s.mu.Unlock()

		s.tcpListener = l

		// Reassign the address in case the port was zero.
		s.url.Host = l.Addr().String()
		s.logger = baseLogger.With(loggerKeyServer, s.url)
		s.http.ErrorLog = slog.NewLogLogger(s.logger.Handler(), slog.LevelError)
	}()

	s.logger.InfoContext(ctx, "starting")
	defer s.logger.InfoContext(ctx, "started")

	err = s.http.Serve(l)
	if err == nil || errors.Is(err, http.ErrServerClosed) {
		return
	}

	s.logger.ErrorContext(ctx, "serving", slogutil.KeyError, err)

	panic(fmt.Errorf("websvc: serving: %w", err))
}

// shutdown shuts s down.
func (s *server) shutdown(ctx context.Context) (err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var errs []error
	err = s.http.Shutdown(ctx)
	if err != nil {
		errs = append(errs, fmt.Errorf("shutting down server %s: %w", s.url, err))
	}

	// Close the listener separately, as it might not have been closed if the
	// context has been canceled.
	//
	// NOTE:  The listener could remain uninitialized if [net.ListenTCP] failed
	// in [s.serve].
	if l := s.tcpListener; l != nil {
		err = l.Close()
		if err != nil && !errors.Is(err, net.ErrClosed) {
			errs = append(errs, fmt.Errorf("closing listener for server %s: %w", s.url, err))
		}
	}

	return errors.Join(errs...)
}
