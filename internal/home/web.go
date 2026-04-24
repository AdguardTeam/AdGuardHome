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

	// dohMux is the dedicated multiplexer used when a separate HTTPS admin
	// listen address is configured (see [tlsConfigSettings.AdminListenAddr]).
	// It contains only DoH routes and is used by the HTTPS server on
	// [tlsConfigSettings.PortHTTPS] in that mode.  It must not be nil.
	dohMux *http.ServeMux

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
	//
	// When [tlsConfigSettings.AdminListenAddr] is unset, this server handles
	// both admin UI/API and DoH requests.  Otherwise, it handles DoH requests
	// only, and admin UI/API requests are handled by [webAPI.adminHTTPSServer].
	httpsServer httpsServer

	// adminHTTPSServer is the HTTPS server that serves only the admin UI and
	// API on [tlsConfigSettings.AdminListenAddr].  It is only started when
	// [tlsConfigSettings.AdminListenAddr] is valid and has a non-zero port.
	adminHTTPSServer httpsServer

	// startTime is the start time of the web API server in Unix milliseconds.
	startTime time.Time
}

// adminListenAddr returns the currently configured dedicated HTTPS admin
// listen address, if any.  It returns the zero value if the feature is
// disabled.
func adminListenAddr(tlsConf *tlsConfigSettings) (addr netip.AddrPort) {
	if tlsConf == nil {
		return netip.AddrPort{}
	}

	a := tlsConf.AdminListenAddr
	if !a.IsValid() || a.Port() == 0 {
		return netip.AddrPort{}
	}

	return a
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
	w.adminHTTPSServer.cond = sync.NewCond(&w.adminHTTPSServer.condLock)

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
		shutCtx, cancel := context.WithTimeout(ctx, shutdownTimeout)
		shutdownSrv(shutCtx, web.logger, web.httpsServer.server)
		shutdownSrv3(shutCtx, web.logger, web.httpsServer.server3)

		cancel()
	}

	web.httpsServer.enabled = enabled
	web.httpsServer.cert = cert
	web.httpsServer.cond.Broadcast()
	web.httpsServer.cond.L.Unlock()

	// The admin HTTPS server shares its cert with the DoH HTTPS server and is
	// only enabled when a dedicated admin listen address is configured.
	adminEnabled := enabled && adminListenAddr(tlsConf) != (netip.AddrPort{})

	web.adminHTTPSServer.cond.L.Lock()
	if web.adminHTTPSServer.server != nil {
		shutCtx, cancel := context.WithTimeout(ctx, shutdownTimeout)
		shutdownSrv(shutCtx, web.logger, web.adminHTTPSServer.server)

		cancel()
	}

	web.adminHTTPSServer.enabled = adminEnabled
	web.adminHTTPSServer.cert = cert
	web.adminHTTPSServer.cond.Broadcast()
	web.adminHTTPSServer.cond.L.Unlock()
}

// loggerKeyServer is the key used by [webAPI] to identify servers.
const loggerKeyServer = "server"

// start - start serving HTTP requests
func (web *webAPI) start(ctx context.Context) {
	defer slogutil.RecoverAndExit(ctx, web.logger, osutil.ExitCodeFailure)

	web.logger.InfoContext(ctx, "AdGuard Home is available at the following addresses:")

	// for https, we have a separate goroutine loop
	go web.tlsServerLoop(ctx)
	go web.adminTLSServerLoop(ctx)

	// this loop is used as an ability to change listening host and/or port
	for !web.httpsServer.inShutdown {
		printHTTPAddresses(ctx, web.logger, urlutil.SchemeHTTP, web.tlsManager)
		errs := make(chan error, 2)

		hdlr := withMiddlewares(web.conf.mux, limitRequestBody)

		logger := web.baseLogger.With(loggerKeyServer, "plain")

		// TODO(a.garipov):  Remove other logs like this in other code.
		logMw := httputil.NewLogMiddleware(logger, slog.LevelDebug)
		hdlr = logMw.Wrap(hdlr)

		hdlr = web.auth.middleware().Wrap(hdlr)

		// Use an h2c handler to support unencrypted HTTP/2, e.g. for proxies.
		//
		// NOTE:  The auth middleware must be inside the h2c handler to ensure
		// it applies to upgraded HTTP/2 connections as well.  See AG-51779.
		hdlr = h2c.NewHandler(hdlr, &http2.Server{})

		// Create a new instance, because the Web is not usable after Shutdown.
		web.httpServer = &http.Server{
			Addr:              web.conf.BindAddr.String(),
			Handler:           hdlr,
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
	web.httpsServer.cond.Broadcast()
	web.httpsServer.cond.L.Unlock()

	web.adminHTTPSServer.cond.L.Lock()
	web.adminHTTPSServer.inShutdown = true
	web.adminHTTPSServer.cond.Broadcast()
	web.adminHTTPSServer.cond.L.Unlock()

	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, shutdownTimeout)
	defer cancel()

	shutdownSrv(ctx, web.logger, web.httpsServer.server)
	shutdownSrv3(ctx, web.logger, web.httpsServer.server3)
	shutdownSrv(ctx, web.logger, web.adminHTTPSServer.server)
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

	var (
		portHTTPS   uint16
		adminListen netip.AddrPort
	)
	func() {
		config.RLock()
		defer config.RUnlock()

		portHTTPS = config.TLS.PortHTTPS
		adminListen = adminListenAddr(&config.TLS)
	}()

	addr := netip.AddrPortFrom(web.conf.BindAddr.Addr(), portHTTPS).String()
	logger := web.baseLogger.With(loggerKeyServer, "https")

	// TODO(a.garipov):  Remove other logs like this in other code.
	logMw := httputil.NewLogMiddleware(logger, slog.LevelDebug)

	// When a dedicated admin HTTPS listen address is configured, this server
	// serves DoH only and must not require web-UI authentication.  Otherwise,
	// it continues to serve both admin UI and DoH on the same port.
	var hdlr http.Handler
	if adminListen != (netip.AddrPort{}) {
		hdlr = logMw.Wrap(withMiddlewares(web.conf.dohMux, limitRequestBody))
	} else {
		hdlr = logMw.Wrap(withMiddlewares(web.conf.mux, limitRequestBody))
		hdlr = web.auth.middleware().Wrap(hdlr)
	}

	web.httpsServer.server = &http.Server{
		Addr:    addr,
		Handler: hdlr,
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

	printHTTPAddresses(ctx, web.logger, urlutil.SchemeHTTPS, web.tlsManager)

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

// adminTLSServerLoop implements retry logic for the dedicated admin HTTPS
// server.  The loop is a no-op while [tlsConfigSettings.AdminListenAddr] is
// unset.
func (web *webAPI) adminTLSServerLoop(ctx context.Context) {
	defer slogutil.RecoverAndExit(ctx, web.logger, osutil.ExitCodeFailure)

	for {
		shouldContinue := web.serveAdminTLS(ctx)
		if !shouldContinue {
			return
		}
	}
}

// serveAdminTLS initializes and starts the dedicated admin HTTPS server on
// [tlsConfigSettings.AdminListenAddr], if configured.  It returns true when
// another retry is required.
func (web *webAPI) serveAdminTLS(ctx context.Context) (next bool) {
	if !web.waitForAdminTLSReady() {
		return false
	}

	var adminListen netip.AddrPort
	func() {
		config.RLock()
		defer config.RUnlock()

		adminListen = adminListenAddr(&config.TLS)
	}()

	if adminListen == (netip.AddrPort{}) {
		// The feature was disabled between ready-broadcast and here.  Loop
		// back and wait again.
		return true
	}

	logger := web.baseLogger.With(loggerKeyServer, "admin-https")

	// TODO(a.garipov):  Remove other logs like this in other code.
	logMw := httputil.NewLogMiddleware(logger, slog.LevelDebug)
	hdlr := logMw.Wrap(withMiddlewares(web.conf.mux, limitRequestBody))

	web.adminHTTPSServer.server = &http.Server{
		Addr:    adminListen.String(),
		Handler: web.auth.middleware().Wrap(hdlr),
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{web.adminHTTPSServer.cert},
			RootCAs:      web.tlsManager.rootCerts,
			CipherSuites: web.tlsManager.customCipherIDs,
			MinVersion:   tls.VersionTLS12,
		},
		ReadTimeout:       web.conf.ReadTimeout,
		ReadHeaderTimeout: web.conf.ReadHeaderTimeout,
		WriteTimeout:      web.conf.WriteTimeout,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	logger.InfoContext(ctx, "starting admin https server", "addr", adminListen.String())
	err := web.adminHTTPSServer.server.ListenAndServeTLS("", "")
	if !errors.Is(err, http.ErrServerClosed) {
		cleanupAlways()
		panic(fmt.Errorf("admin https: %w", err))
	}

	return true
}

// waitForAdminTLSReady blocks until the admin HTTPS server is enabled or a
// shutdown signal is received.  Returns true when the server is ready.
func (web *webAPI) waitForAdminTLSReady() (ok bool) {
	web.adminHTTPSServer.cond.L.Lock()
	defer web.adminHTTPSServer.cond.L.Unlock()

	if web.adminHTTPSServer.inShutdown {
		return false
	}

	for !web.adminHTTPSServer.enabled {
		web.adminHTTPSServer.cond.Wait()
		if web.adminHTTPSServer.inShutdown {
			return false
		}
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

	var adminListen netip.AddrPort
	func() {
		config.RLock()
		defer config.RUnlock()

		adminListen = adminListenAddr(&config.TLS)
	}()

	// Mirror [webAPI.serveTLS]: when a dedicated admin HTTPS listen address
	// is configured, HTTP/3 on [tlsConfigSettings.PortHTTPS] serves DoH only
	// and must not require web-UI authentication.
	var hdlr http.Handler
	if adminListen != (netip.AddrPort{}) {
		hdlr = withMiddlewares(web.conf.dohMux, limitRequestBody)
	} else {
		hdlr = web.auth.middleware().Wrap(withMiddlewares(web.conf.mux, limitRequestBody))
	}

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
		Handler: hdlr,
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
