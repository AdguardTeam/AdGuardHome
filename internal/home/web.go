package home

import (
	"context"
	"crypto/tls"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"net/netip"
	"runtime"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/agh"
	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/updater"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/netutil/httputil"
	"github.com/AdguardTeam/golibs/netutil/urlutil"
	"github.com/AdguardTeam/golibs/osutil"
	"github.com/AdguardTeam/golibs/osutil/executil"
	"github.com/NYTimes/gziphandler"
	"github.com/quic-go/quic-go/http3"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// TODO(a.garipov): Make configurable.
const (
	// readTimeout is the maximum duration for reading the entire request,
	// including the body.
	readTimeout = 60 * time.Second
	// readHdrTimeout is the amount of time allowed to read request headers.
	readHdrTimeout = 60 * time.Second
	// writeTimeout is the maximum duration before timing out writes of the
	// response.
	writeTimeout = 5 * time.Minute
)

// webAPIConfig is a configuration structure for webAPI.
type webAPIConfig struct {
	// CommandConstructor is used to run external commands.  It must not be nil.
	CommandConstructor executil.CommandConstructor

	// updater is used for updating AdGuard home.  If disableUpdate is set to
	// false, it must not be nil.
	updater *updater.Updater

	// logger is a slog logger used in webAPI. It must not be nil.
	logger *slog.Logger

	// baseLogger is used to create loggers for other entities.  It must not be
	// nil.
	baseLogger *slog.Logger

	// confModifier is used to update the global configuration.
	confModifier agh.ConfigModifier

	// httpReg registers HTTP handlers.  It must not be nil.
	httpReg aghhttp.Registrar

	// tlsManager contains the current configuration and state of TLS
	// encryption.  It must not be nil.
	tlsManager *tlsManager

	// auth stores web user information and handles authentication.  It must not
	// be nil.
	auth *auth

	// mux is the default *http.ServeMux, the same as [globalContext.mux].  It
	// must not be nil.
	mux *http.ServeMux

	// clientFS is used to initialize file server.  It must not be nil.
	clientFS fs.FS

	// BindAddr is the binding address with port for plain HTTP web interface.
	BindAddr netip.AddrPort

	// workDir is the base working directory.
	workDir string

	// confPath is the configuration file path.
	confPath string

	// ReadTimeout is an option to pass to http.Server for setting an
	// appropriate field.
	ReadTimeout time.Duration

	// ReadHeaderTimeout is an option to pass to http.Server for setting an
	// appropriate field.
	ReadHeaderTimeout time.Duration

	// WriteTimeout is an option to pass to http.Server for setting an
	// appropriate field.
	WriteTimeout time.Duration

	// defaultWebPort is the suggested default HTTP port for the install wizard.
	defaultWebPort uint16

	// firstRun, if true, tells AdGuard Home to register install handlers.
	firstRun bool

	// disableUpdate, if true, tells AdGuard Home to not check for updates.
	disableUpdate bool

	// runningAsService flag is set to true when options are passed from the
	// service runner.
	runningAsService bool

	// serveHTTP3, if true, tells AdGuard Home to start HTTP3 server.
	serveHTTP3 bool
}

// httpsServer contains the data for the HTTPS server.
type httpsServer struct {
	// server is the pre-HTTP/3 HTTPS server.
	server *http.Server
	// server3 is the HTTP/3 HTTPS server.  If it is not nil,
	// [httpsServer.server] must also be non-nil.
	server3 *http3.Server

	// TODO(a.garipov): Why is there a *sync.Cond here?  Remove.
	cond       *sync.Cond
	condLock   sync.Mutex
	cert       tls.Certificate
	inShutdown bool
	enabled    bool
}

// webAPI is the web UI and API server.
type webAPI struct {
	conf *webAPIConfig

	// confModifier is used to update the global configuration.
	confModifier agh.ConfigModifier

	// cmdCons is used to run external commands.
	cmdCons executil.CommandConstructor

	// httpReg registers HTTP handlers.
	httpReg aghhttp.Registrar

	// TODO(a.garipov): Refactor all these servers.
	httpServer *http.Server

	// logger is a slog logger used in webAPI. It must not be nil.
	logger *slog.Logger

	// baseLogger is used to create loggers for other entities.  It must not be
	// nil.
	baseLogger *slog.Logger

	// tlsManager contains the current configuration and state of TLS
	// encryption.
	tlsManager *tlsManager

	// auth stores web user information and handles authentication.
	auth *auth

	// httpsServer is the server that handles HTTPS traffic.  If it is not nil,
	// [Web.http3Server] must also not be nil.
	httpsServer httpsServer

	// startTime is the start time of the web API server in Unix milliseconds.
	startTime time.Time
}

// newWebAPI creates a new instance of the web UI and API server.  conf must be
// valid.
//
// TODO(a.garipov):  Return a proper error.
func newWebAPI(ctx context.Context, conf *webAPIConfig) (w *webAPI) {
	conf.logger.InfoContext(ctx, "initializing")

	w = &webAPI{
		conf:         conf,
		confModifier: conf.confModifier,
		httpReg:      conf.httpReg,
		cmdCons:      conf.CommandConstructor,
		logger:       conf.logger,
		baseLogger:   conf.baseLogger,
		tlsManager:   conf.tlsManager,
		auth:         conf.auth,
		startTime:    time.Now(),
	}

	clientFS := http.FileServer(http.FS(conf.clientFS))

	mux := conf.mux
	// if not configured, redirect / to /install.html, otherwise redirect /install.html to /
	mux.Handle("/", withMiddlewares(clientFS, gziphandler.GzipHandler, w.postInstallHandler))

	// add handlers for /install paths, we only need them when we're not configured yet
	if conf.firstRun {
		conf.logger.InfoContext(
			ctx,
			"This is the first launch of AdGuard Home, redirecting everything to /install.html",
		)

		mux.Handle("/install.html", w.preInstallHandler(clientFS))
		w.registerInstallHandlers()
	} else {
		w.registerControlHandlers()
	}

	w.httpsServer.cond = sync.NewCond(&w.httpsServer.condLock)

	return w
}

// tlsConfigChanged updates the TLS configuration and restarts the HTTPS server
// if necessary.  tlsConf must not be nil.
func (web *webAPI) tlsConfigChanged(ctx context.Context, tlsConf *tlsConfigSettings) {
	defer slogutil.RecoverAndExit(ctx, web.logger, osutil.ExitCodeFailure)

	web.logger.DebugContext(ctx, "applying new tls configuration")

	enabled := tlsConf.Enabled &&
		tlsConf.PortHTTPS != 0 &&
		len(tlsConf.PrivateKeyData) != 0 &&
		len(tlsConf.CertificateChainData) != 0
	var cert tls.Certificate
	var err error
	if enabled {
		cert, err = tls.X509KeyPair(tlsConf.CertificateChainData, tlsConf.PrivateKeyData)
		if err != nil {
			panic(err)
		}
	}

	web.httpsServer.cond.L.Lock()
	if web.httpsServer.server != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, shutdownTimeout)
		shutdownSrv(ctx, web.logger, web.httpsServer.server)
		shutdownSrv3(ctx, web.logger, web.httpsServer.server3)

		cancel()
	}

	web.httpsServer.enabled = enabled
	web.httpsServer.cert = cert
	web.httpsServer.cond.Broadcast()
	web.httpsServer.cond.L.Unlock()
}

// loggerKeyServer is the key used by [webAPI] to identify servers.
const loggerKeyServer = "server"

// start - start serving HTTP requests
func (web *webAPI) start(ctx context.Context) {
	defer slogutil.RecoverAndExit(ctx, web.logger, osutil.ExitCodeFailure)

	web.logger.InfoContext(ctx, "AdGuard Home is available at the following addresses:")

	// for https, we have a separate goroutine loop
	go web.tlsServerLoop(ctx)

	// this loop is used as an ability to change listening host and/or port
	for !web.httpsServer.inShutdown {
		printHTTPAddresses(urlutil.SchemeHTTP, web.tlsManager)
		errs := make(chan error, 2)

		// Use an h2c handler to support unencrypted HTTP/2, e.g. for proxies.
		hdlr := h2c.NewHandler(
			withMiddlewares(web.conf.mux, limitRequestBody),
			&http2.Server{},
		)

		logger := web.baseLogger.With(loggerKeyServer, "plain")

		// TODO(a.garipov):  Remove other logs like this in other code.
		logMw := httputil.NewLogMiddleware(logger, slog.LevelDebug)
		hdlr = logMw.Wrap(hdlr)

		// Create a new instance, because the Web is not usable after Shutdown.
		web.httpServer = &http.Server{
			Addr:              web.conf.BindAddr.String(),
			Handler:           web.auth.middleware().Wrap(hdlr),
			ReadTimeout:       web.conf.ReadTimeout,
			ReadHeaderTimeout: web.conf.ReadHeaderTimeout,
			WriteTimeout:      web.conf.WriteTimeout,
			ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
		}
		go func() {
			defer slogutil.RecoverAndLog(ctx, logger)

			logger.InfoContext(ctx, "starting plain server", "addr", web.httpServer.Addr)

			errs <- web.httpServer.ListenAndServe()
		}()

		err := <-errs
		if !errors.Is(err, http.ErrServerClosed) {
			cleanupAlways()
			panic(err)
		}

		// We use ErrServerClosed as a sign that we need to rebind on a new
		// address, so go back to the start of the loop.
	}
}

// close gracefully shuts down the HTTP servers.
func (web *webAPI) close(ctx context.Context) {
	web.logger.InfoContext(ctx, "stopping http server")

	web.httpsServer.cond.L.Lock()
	web.httpsServer.inShutdown = true
	web.httpsServer.cond.L.Unlock()

	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, shutdownTimeout)
	defer cancel()

	shutdownSrv(ctx, web.logger, web.httpsServer.server)
	shutdownSrv3(ctx, web.logger, web.httpsServer.server3)
	shutdownSrv(ctx, web.logger, web.httpServer)

	if web.auth != nil {
		web.auth.close(ctx)
	}

	web.logger.InfoContext(ctx, "stopped http server")
}

// tlsServerLoop implements retry logic for http server start.
func (web *webAPI) tlsServerLoop(ctx context.Context) {
	defer slogutil.RecoverAndExit(ctx, web.logger, osutil.ExitCodeFailure)

	for {
		shouldContinue := web.serveTLS(ctx)
		if !shouldContinue {
			return
		}
	}
}

// serveTLS initializes and starts the HTTPS server.  Returns true when next
// retry is necessary.
func (web *webAPI) serveTLS(ctx context.Context) (next bool) {
	if !web.waitForTLSReady() {
		return false
	}

	var portHTTPS uint16
	func() {
		config.RLock()
		defer config.RUnlock()

		portHTTPS = config.TLS.PortHTTPS
	}()

	addr := netip.AddrPortFrom(web.conf.BindAddr.Addr(), portHTTPS).String()
	logger := web.baseLogger.With(loggerKeyServer, "https")

	// TODO(a.garipov):  Remove other logs like this in other code.
	logMw := httputil.NewLogMiddleware(logger, slog.LevelDebug)
	hdlr := logMw.Wrap(withMiddlewares(web.conf.mux, limitRequestBody))

	web.httpsServer.server = &http.Server{
		Addr:    addr,
		Handler: web.auth.middleware().Wrap(hdlr),
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{web.httpsServer.cert},
			RootCAs:      web.tlsManager.rootCerts,
			CipherSuites: web.tlsManager.customCipherIDs,
			MinVersion:   tls.VersionTLS12,
		},
		ReadTimeout:       web.conf.ReadTimeout,
		ReadHeaderTimeout: web.conf.ReadHeaderTimeout,
		WriteTimeout:      web.conf.WriteTimeout,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	printHTTPAddresses(urlutil.SchemeHTTPS, web.tlsManager)

	if web.conf.serveHTTP3 {
		go web.mustStartHTTP3(ctx, addr)
	}

	logger.InfoContext(ctx, "starting https server")
	err := web.httpsServer.server.ListenAndServeTLS("", "")
	if !errors.Is(err, http.ErrServerClosed) {
		cleanupAlways()
		panic(fmt.Errorf("https: %w", err))
	}

	return true
}

// waitForTLSReady blocks until the HTTPS server is enabled or a shutdown signal
// is received.  Returns true when server is ready.
func (web *webAPI) waitForTLSReady() (ok bool) {
	web.httpsServer.cond.L.Lock()
	defer web.httpsServer.cond.L.Unlock()

	if web.httpsServer.inShutdown {
		return false
	}

	// this mechanism doesn't let us through until all conditions are met
	for !web.httpsServer.enabled { // sleep until necessary data is supplied
		web.httpsServer.cond.Wait()
		if web.httpsServer.inShutdown {
			return false
		}
	}

	return true
}

// mustStartHTTP3 initializes and starts HTTP3 server.
func (web *webAPI) mustStartHTTP3(ctx context.Context, address string) {
	defer slogutil.RecoverAndExit(ctx, web.logger, osutil.ExitCodeFailure)

	web.httpsServer.server3 = &http3.Server{
		// TODO(a.garipov): See if there is a way to use the error log as
		// well as timeouts here.
		Addr: address,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{web.httpsServer.cert},
			RootCAs:      web.tlsManager.rootCerts,
			CipherSuites: web.tlsManager.customCipherIDs,
			MinVersion:   tls.VersionTLS12,
		},
		Handler: web.auth.middleware().Wrap(withMiddlewares(web.conf.mux, limitRequestBody)),
	}

	web.logger.DebugContext(ctx, "starting http/3 server")
	err := web.httpsServer.server3.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		cleanupAlways()
		panic(fmt.Errorf("http3: %w", err))
	}
}

// startPprof launches the debug and profiling server on the provided port.
func startPprof(baseLogger *slog.Logger, port uint16) {
	addr := netip.AddrPortFrom(netutil.IPv4Localhost(), port)

	runtime.SetBlockProfileRate(1)
	runtime.SetMutexProfileFraction(1)

	mux := http.NewServeMux()
	httputil.RoutePprof(mux)

	ctx := context.Background()
	logger := baseLogger.With(slogutil.KeyPrefix, "pprof")

	go func() {
		defer slogutil.RecoverAndLog(ctx, logger)

		logger.InfoContext(ctx, "listening", "addr", addr)
		err := http.ListenAndServe(addr.String(), mux)
		if !errors.Is(err, http.ErrServerClosed) {
			logger.ErrorContext(ctx, "shutting down", slogutil.KeyError, err)
		}
	}()
}
