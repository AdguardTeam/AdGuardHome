package home

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"net/netip"
	"runtime"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/agh"
	"github.com/AdguardTeam/AdGuardHome/internal/aghalg"
	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/aghtls"
	"github.com/AdguardTeam/AdGuardHome/internal/updater"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/logutil/slogutil"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/AdguardTeam/golibs/netutil/httputil"
	"github.com/AdguardTeam/golibs/osutil"
	"github.com/AdguardTeam/golibs/osutil/executil"
	"github.com/AdguardTeam/golibs/service"
	"github.com/NYTimes/gziphandler"
	"github.com/quic-go/quic-go/http3"
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

// unit is a convenience alias for an empty struct.
type unit = struct{}

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

	// hostsContainer is used for DNS initialization on updates.
	hostsContainer *aghnet.HostsContainer

	// clientFS is used to initialize file server.  It must not be nil.
	clientFS fs.FS

	// BindAddr is the binding address with port for plain HTTP web interface.
	BindAddr netip.AddrPort

	// workDir is the base working directory.
	workDir string

	// confPath is the configuration file path.
	confPath string

	// pidFilePath is a path to a PID file.
	pidFilePath string

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
	// logger is used for logging the operation of the server.  It must not be
	// nil.
	logger *slog.Logger

	// server is the pre-HTTP/3 HTTPS server.
	server *http.Server

	// server3 is the HTTP/3 HTTPS server.  If it is not nil,
	// [httpsServer.server] must also be non-nil.
	server3 *http3.Server

	// mu protects enabled, and shutdown.  It must not be nil.
	mu *sync.Mutex

	// reconfigured wakes the TLS server loop waiting in [waitForTLSReady]
	// whenever enabled or shutdown changes.
	reconfigured chan unit

	// shutdown is true when this httpsServer is shutting down.
	shutdown bool

	// enabled is true when this httpsServer is ready to use.
	enabled bool
}

// notifyReconfigured notifies the loop waiting in [waitForTLSReady].
func (srv *httpsServer) notifyReconfigured(ctx context.Context) {
	select {
	case srv.reconfigured <- unit{}:
	default:
		srv.logger.WarnContext(ctx, "reconfigured channel is full")
	}
}

// inShutdown reports whether the server is in shutdown process.
func (srv *httpsServer) inShutdown() (ok bool) {
	srv.mu.Lock()
	defer srv.mu.Unlock()

	return srv.shutdown
}

// waitForTLSReady blocks until the server is enabled or a shutdown signal is
// received.  Returns true when server is ready.  Must be run with one goroutine
// only.
func (srv *httpsServer) waitForTLSReady() (ok bool) {
	for {
		// Wait until necessary data is supplied or a shutdown is requested.
		<-srv.reconfigured

		srv.mu.Lock()

		switch {
		case srv.shutdown:
			srv.mu.Unlock()

			return false
		case srv.enabled:
			srv.mu.Unlock()

			return true
		default:
			srv.mu.Unlock()
		}
	}
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

	// tlsManager manages the TLS state.  It must not be nil.
	//
	// TODO(m.kazantsev):  Replace with [aghtls.Manager] once
	// [aghtls.TLSConfigProvider] is merged with it.
	tlsManager service.Interface

	// tlsConfProvider is used to provide the TLS configuration.
	//
	// TODO(m.kazantsev):  Merge with [aghtls.Manager].
	tlsConfProvider aghtls.TLSConfigProvider

	// auth stores web user information and handles authentication.
	auth *auth

	// hostsContainer is used for DNS initialization on updates.
	hostsContainer *aghnet.HostsContainer

	// httpsServer is the server that handles HTTPS traffic.  If it is not nil,
	// [Web.http3Server] must also not be nil.
	//
	// TODO(d.kolyshev):  Make it a pointer.
	httpsServer httpsServer

	// pidFilePath is used for cleanup.
	pidFilePath string

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
		conf:            conf,
		confModifier:    conf.confModifier,
		httpReg:         conf.httpReg,
		cmdCons:         conf.CommandConstructor,
		logger:          conf.logger,
		baseLogger:      conf.baseLogger,
		tlsManager:      conf.tlsManager,
		tlsConfProvider: conf.tlsManager,
		auth:            conf.auth,
		pidFilePath:     conf.pidFilePath,
		startTime:       time.Now(),
		hostsContainer:  conf.hostsContainer,
	}

	clientFS := http.FileServer(http.FS(conf.clientFS))

	mux := conf.mux
	// if not configured, redirect / to /install.html, otherwise redirect /install.html to /
	mux.Handle("/", httputil.Wrap(
		clientFS,
		httputil.MiddlewareFunc(w.postInstallHandler),
		httputil.MiddlewareFunc(gziphandler.GzipHandler),
	))

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

	w.httpsServer.logger = conf.baseLogger.With(slogutil.KeyPrefix, "https_server")
	w.httpsServer.mu = &sync.Mutex{}
	w.httpsServer.reconfigured = make(chan unit, 1)

	return w
}

// tlsConfigChanged updates the TLS configuration and restarts the HTTPS server
// if necessary.  tlsConf must not be nil.
func (web *webAPI) tlsConfigChanged(ctx context.Context, tlsConf *aghtls.ExtendedTLSConfig) {
	defer slogutil.RecoverAndExit(ctx, web.logger, osutil.ExitCodeFailure)

	web.logger.DebugContext(ctx, "applying new tls configuration")

	enabled := tlsConf.Enabled &&
		tlsConf.PortHTTPS != 0 &&
		len(tlsConf.PrivateKeyData) != 0 &&
		len(tlsConf.CertificateChainData) != 0

	// TODO(d.kolyshev):  Consider protecting server with mu.
	if web.httpsServer.server != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, shutdownTimeout)
		shutdownSrv(ctx, web.logger, web.httpsServer.server)
		shutdownSrv3(ctx, web.logger, web.httpsServer.server3)

		cancel()
	}

	func() {
		web.httpsServer.mu.Lock()
		defer web.httpsServer.mu.Unlock()

		web.httpsServer.enabled = enabled
	}()

	web.httpsServer.notifyReconfigured(ctx)
}

// loggerKeyServer is the key used by [webAPI] to identify servers.
const loggerKeyServer = "server"

// getBindAddr returns the network and address strings to use when creating a
// listener on addr and port.  network must be either "tcp" or "udp".  The
// address family of addr is preserved:  for the unspecified IPv4 address the
// IPv4-only network is returned, since Go's wildcard listeners otherwise
// accept connections of both address families on platforms that support
// IPv4-mapped IPv6 addresses.  For the unspecified IPv6 address the returned
// address is in the ":port" form, which enables dual-stack listening.
func getBindAddr(network string, addr netip.Addr, port uint16) (listenNetwork, addrStr string) {
	switch {
	case !addr.IsUnspecified():
		return network, netip.AddrPortFrom(addr, port).String()
	case addr.Is4():
		return network + "4", netip.AddrPortFrom(addr, port).String()
	default:
		return network, netutil.JoinHostPort("", port)
	}
}

// start starts serving HTTP requests.
func (web *webAPI) start(ctx context.Context) {
	defer slogutil.RecoverAndExit(ctx, web.logger, osutil.ExitCodeFailure)

	web.logger.InfoContext(ctx, "AdGuard Home is available at the following addresses:")

	go func() {
		// Apply the initial TLS configuration.  The background context is used
		// because tlsConfigChanged wraps context with timeout on its own and shuts
		// down the server, which handles current request.
		web.tlsConfigChanged(context.Background(), web.tlsConfProvider.ExtendedTLSConfig())

		// For https, we have a separate goroutine loop.
		web.tlsServerLoop(ctx)
	}()

	// This loop is used as an ability to change listening host and/or port.
	for !web.httpsServer.inShutdown() {
		printHTTPAddresses(ctx, web.logger)
		errs := make(chan error, 2)

		logger := web.baseLogger.With(loggerKeyServer, "plain")
		hdlr := web.wrapMux(logger)

		// Enable unencrypted HTTP/2, e.g. for proxies.
		protocols := &http.Protocols{}
		protocols.SetUnencryptedHTTP2(true)
		protocols.SetHTTP1(true)

		network, addrStr := getBindAddr("tcp", web.conf.BindAddr.Addr(), web.conf.BindAddr.Port())

		// Create a new instance, because the Web is not usable after Shutdown.
		web.httpServer = &http.Server{
			Addr:              addrStr,
			Handler:           hdlr,
			ReadTimeout:       web.conf.ReadTimeout,
			ReadHeaderTimeout: web.conf.ReadHeaderTimeout,
			WriteTimeout:      web.conf.WriteTimeout,
			ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
			Protocols:         protocols,
		}
		go func() {
			defer slogutil.RecoverAndLog(ctx, logger)

			logger.InfoContext(ctx, "starting plain server", "addr", addrStr)

			ln, lErr := net.Listen(network, addrStr)
			if lErr != nil {
				errs <- lErr

				return
			}

			errs <- web.httpServer.Serve(ln)
		}()

		err := <-errs
		if !errors.Is(err, http.ErrServerClosed) {
			cleanupAlways(ctx, logger, web.pidFilePath)

			panic(err)
		}

		// We use ErrServerClosed as a sign that we need to rebind on a new
		// address, so go back to the start of the loop.
	}
}

// wrapMux wraps mux with common middlewares.  l must not be nil.
func (web *webAPI) wrapMux(l *slog.Logger) (h http.Handler) {
	h = httputil.Wrap(web.conf.mux, httputil.MiddlewareFunc(limitRequestBody))

	// TODO(a.garipov):  Remove other logs like this in other code.
	logMw := httputil.NewLogMiddleware(l, slog.LevelDebug)
	h = logMw.Wrap(h)

	return web.auth.middleware().Wrap(h)
}

// close gracefully shuts down the HTTP servers.
func (web *webAPI) close(ctx context.Context) {
	web.logger.InfoContext(ctx, "stopping http server")

	func() {
		web.httpsServer.mu.Lock()
		defer web.httpsServer.mu.Unlock()

		web.httpsServer.shutdown = true
	}()

	web.httpsServer.notifyReconfigured(ctx)

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
	if !web.httpsServer.waitForTLSReady() {
		return false
	}

	var portHTTPS uint16
	func() {
		config.RLock()
		defer config.RUnlock()

		portHTTPS = config.TLS.PortHTTPS
	}()

	network, addrStr := getBindAddr("tcp", web.conf.BindAddr.Addr(), portHTTPS)
	logger := web.baseLogger.With(loggerKeyServer, "https")

	hdlr := web.wrapMux(logger)

	web.httpsServer.server = &http.Server{
		Addr:              addrStr,
		Handler:           hdlr,
		TLSConfig:         web.tlsConfProvider.TLSConfig(),
		ReadTimeout:       web.conf.ReadTimeout,
		ReadHeaderTimeout: web.conf.ReadHeaderTimeout,
		WriteTimeout:      web.conf.WriteTimeout,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}

	extTLSConf := web.tlsConfProvider.ExtendedTLSConfig()
	printHTTPSAddresses(ctx, web.logger, extTLSConf)

	if web.conf.serveHTTP3 {
		go web.mustStartHTTP3(ctx, portHTTPS)
	}

	logger.InfoContext(ctx, "starting https server")
	ln, err := net.Listen(network, addrStr)
	if err == nil {
		err = web.httpsServer.server.ServeTLS(ln, "", "")
	}

	if !errors.Is(err, http.ErrServerClosed) {
		cleanupAlways(ctx, logger, web.pidFilePath)

		panic(fmt.Errorf("https: %w", err))
	}

	return true
}

// mustStartHTTP3 initializes and starts HTTP3 server on the configured bind
// address with the given port.
func (web *webAPI) mustStartHTTP3(ctx context.Context, port uint16) {
	defer slogutil.RecoverAndExit(ctx, web.logger, osutil.ExitCodeFailure)

	logger := web.baseLogger.With(loggerKeyServer, "http3")
	hdlr := web.wrapMux(logger)

	network, addrStr := getBindAddr("udp", web.conf.BindAddr.Addr(), port)

	web.httpsServer.server3 = &http3.Server{
		// TODO(a.garipov): See if there is a way to use the error log as
		// well as timeouts here.
		Addr:      addrStr,
		TLSConfig: web.tlsConfProvider.TLSConfig(),
		Handler:   hdlr,
	}

	web.logger.DebugContext(ctx, "starting http/3 server")
	err := serveHTTP3(ctx, logger, web.httpsServer.server3, network, addrStr)
	if !errors.Is(err, http.ErrServerClosed) {
		cleanupAlways(ctx, logger, web.pidFilePath)

		panic(fmt.Errorf("http3: %w", err))
	}
}

// serveHTTP3 listens for UDP packets on the given network and address, and
// serves HTTP/3 requests on srv until it is closed.  The created packet
// connection is closed before returning, since [http3.Server.Serve] does not
// close connections provided by the caller.  logger and srv must not be nil.
func serveHTTP3(
	ctx context.Context,
	logger *slog.Logger,
	srv *http3.Server,
	network string,
	addrStr string,
) (err error) {
	conn, err := net.ListenPacket(network, addrStr)
	if err != nil {
		// Don't wrap the error because it's informative enough as is.
		return err
	}
	defer slogutil.CloseAndLog(ctx, logger, conn, slog.LevelDebug)

	return srv.Serve(conn)
}

// startPprof launches the debug and profiling server on the provided port on
// both IPv4 and IPv6 loopback addresses.
func startPprof(baseLogger *slog.Logger, port uint16) {
	runtime.SetBlockProfileRate(1)
	runtime.SetMutexProfileFraction(1)

	mux := http.NewServeMux()
	httputil.RoutePprof(mux)

	ctx := context.Background()
	logger := baseLogger.With(slogutil.KeyPrefix, "pprof")

	go servePprof(ctx, logger, mux, netutil.IPv4Localhost(), port)
	go servePprof(ctx, logger, mux, netutil.IPv6Localhost(), port)
}

// servePprof serves the pprof HTTP endpoints on the given host and port.
func servePprof(
	ctx context.Context,
	logger *slog.Logger,
	mux *http.ServeMux,
	host netip.Addr,
	port uint16,
) {
	defer slogutil.RecoverAndLog(ctx, logger)

	addrStr := netip.AddrPortFrom(host, port).String()
	logger.InfoContext(ctx, "listening", "addr", addrStr)
	err := http.ListenAndServe(addrStr, mux)
	if !errors.Is(err, http.ErrServerClosed) {
		logger.ErrorContext(ctx, "shutting down", slogutil.KeyError, err)
	}
}

// handleTLSStatus is the handler for the GET /control/tls/status HTTP API.
func (web *webAPI) handleTLSStatus(w http.ResponseWriter, r *http.Request) {
	tlsConf := web.tlsConfProvider.ExtendedTLSConfig()

	data := &tlsConfig{
		tlsConfigSettingsExt: tlsConfigSettingsExt{
			tlsConfigSettings: confToTLSSettings(tlsConf),
			ServePlainDNS:     aghalg.BoolToNullBool(tlsConf.ServePlainDNS),
		},
		tlsConfigStatus: tlsConfigStatusFromConf(&tlsConf.Status),
	}

	web.marshalTLS(r.Context(), w, r, data)
}

// handleTLSValidate is the handler for the POST /control/tls/validate HTTP API.
func (web *webAPI) handleTLSValidate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	setts, err := unmarshalTLS(r)
	if err != nil {
		// errFmt does not follow error message guidelines because it is sent
		// directly to the frontend.
		const errFmt = "Failed to unmarshal TLS config: %s"

		aghhttp.ErrorAndLog(ctx, web.logger, r, w, http.StatusBadRequest, errFmt, err)

		return
	}

	extTLSConf := web.tlsConfProvider.ExtendedTLSConfig()

	if setts.PrivateKeySaved {
		setts.PrivateKey = extTLSConf.PrivateKey
	}

	if err = web.validateTLSSettings(setts); err != nil {
		web.logger.InfoContext(ctx, "validating tls settings", slogutil.KeyError, err)

		aghhttp.ErrorAndLog(ctx, web.logger, r, w, http.StatusBadRequest, "%s", err)

		return
	}

	// Skip the error check, since we are only interested in the value of
	// status.WarningValidation.
	status := &aghtls.TLSConfigStatus{}
	_ = loadTLSConfig(
		ctx,
		web.logger,
		web.tlsConfProvider,
		confFromTLSSettings(&setts.tlsConfigSettings),
		status,
	)
	resp := &tlsConfig{
		tlsConfigSettingsExt: setts,
		tlsConfigStatus:      tlsConfigStatusFromConf(status),
	}

	web.marshalTLS(ctx, w, r, resp)
}

// validateTLSSettings returns error if the setts are not valid.
func (web *webAPI) validateTLSSettings(setts tlsConfigSettingsExt) (err error) {
	if !setts.Enabled {
		if setts.ServePlainDNS == aghalg.NBFalse {
			// TODO(a.garipov): Support full disabling of all DNS.
			return errors.Error("plain DNS is required in case encryption protocols are disabled")
		}

		return nil
	}

	var (
		tlsConf      tlsConfigSettings
		webAPIAddr   netip.Addr
		webAPIPort   uint16
		plainDNSPort uint16
	)

	func() {
		config.Lock()
		defer config.Unlock()

		tlsConf = config.TLS
		webAPIAddr = config.HTTPConfig.Address.Addr()
		webAPIPort = config.HTTPConfig.Address.Port()
		plainDNSPort = config.DNS.Port
	}()

	err = validatePorts(
		tcpPort(webAPIPort),
		tcpPort(setts.PortHTTPS),
		tcpPort(setts.PortDNSOverTLS),
		tcpPort(setts.PortDNSCrypt),
		udpPort(plainDNSPort),
		udpPort(setts.PortDNSOverQUIC),
	)
	if err != nil {
		// Don't wrap the error because it's informative enough as is.
		return err
	}

	// Don't wrap the error because it's informative enough as is.
	return checkPortAvailability(tlsConf, setts.tlsConfigSettings, webAPIAddr)
}

// checkPortAvailability checks [tlsConfigSettings.PortHTTPS],
// [tlsConfigSettings.PortDNSOverTLS], and
// [tlsConfigSettings.PortDNSOverQUIC] are available for use.  It checks
// the current configuration and, if needed, attempts to bind to the port.  The
// function returns human-readable error messages for the frontend.  This is
// best-effort check to prevent an "address already in use" error.
//
// TODO(a.garipov): Adapt for HTTP/3.
func checkPortAvailability(
	currConf tlsConfigSettings,
	newConf tlsConfigSettings,
	addr netip.Addr,
) (err error) {
	const (
		networkTCP = "tcp"
		networkUDP = "udp"

		protoHTTPS = "HTTPS"
		protoDoT   = "DNS-over-TLS"
		protoDoQ   = "DNS-over-QUIC"
	)

	needBindingCheck := []struct {
		network  string
		proto    string
		currPort uint16
		newPort  uint16
	}{{
		network:  networkTCP,
		proto:    protoHTTPS,
		currPort: currConf.PortHTTPS,
		newPort:  newConf.PortHTTPS,
	}, {
		network:  networkTCP,
		proto:    protoDoT,
		currPort: currConf.PortDNSOverTLS,
		newPort:  newConf.PortDNSOverTLS,
	}, {
		network:  networkUDP,
		proto:    protoDoQ,
		currPort: currConf.PortDNSOverQUIC,
		newPort:  newConf.PortDNSOverQUIC,
	}}

	var errs []error
	for _, v := range needBindingCheck {
		port := v.newPort
		if v.currPort == port {
			continue
		}

		addrPort := netip.AddrPortFrom(addr, port)
		err = aghnet.CheckPort(v.network, addrPort)
		if err != nil {
			errs = append(errs, fmt.Errorf("port %d for %s is not available", port, v.proto))
		}
	}

	return errors.Join(errs...)
}

// validatePorts validates the uniqueness of TCP and UDP ports for AdGuard Home
// DNS protocols.
func validatePorts(
	bindPort, dohPort, dotPort, dnscryptTCPPort tcpPort,
	dnsPort, doqPort udpPort,
) (err error) {
	tcpPorts := aghalg.UniqChecker[tcpPort]{}
	addPorts(
		tcpPorts,
		bindPort,
		dohPort,
		dotPort,
		dnscryptTCPPort,
		tcpPort(dnsPort),
	)

	err = tcpPorts.Validate()
	if err != nil {
		return fmt.Errorf("validating tcp ports: %w", err)
	}

	udpPorts := aghalg.UniqChecker[udpPort]{}
	addPorts(udpPorts, dnsPort, doqPort)

	err = udpPorts.Validate()
	if err != nil {
		return fmt.Errorf("validating udp ports: %w", err)
	}

	return nil
}

// handleTLSConfigure is the handler for the POST /control/tls/configure HTTP
// API.
//
// TODO(m.kazantsev):  Improve maintainability.
func (web *webAPI) handleTLSConfigure(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	req, err := unmarshalTLS(r)
	if err != nil {
		aghhttp.ErrorAndLog(
			ctx,
			web.logger,
			r,
			w,
			http.StatusBadRequest,
			"Failed to unmarshal TLS config: %s",
			err,
		)

		return
	}

	var restartHTTPS bool
	defer func() {
		if restartHTTPS {
			web.confModifier.Apply(ctx)
		}
	}()

	extTLSConf := web.tlsConfProvider.ExtendedTLSConfig()

	if req.PrivateKeySaved {
		req.PrivateKey = extTLSConf.PrivateKey
	}

	req.StrictSNICheck = extTLSConf.StrictSNICheck

	if err = web.validateTLSSettings(req); err != nil {
		aghhttp.ErrorAndLog(ctx, web.logger, r, w, http.StatusBadRequest, "%s", err)

		return
	}

	status := &aghtls.TLSConfigStatus{}
	conf := confFromTLSSettings(&req.tlsConfigSettings)
	err = loadTLSConfig(ctx, web.logger, web.tlsConfProvider, conf, status)
	if err != nil {
		resp := &tlsConfig{
			tlsConfigSettingsExt: req,
			tlsConfigStatus:      tlsConfigStatusFromConf(status),
		}

		web.marshalTLS(ctx, w, r, resp)

		return
	}

	newTLSConf := conf
	newTLSConf.Status = *status

	restartHTTPS, err = web.tlsConfProvider.SetExtendedTLSConfig(ctx, req.ServePlainDNS, newTLSConf)
	if err != nil {
		aghhttp.ErrorAndLog(ctx, web.logger, r, w, http.StatusInternalServerError, "%s", err)

		return
	}

	err = web.reconfigureDNSServer(ctx)
	if err != nil {
		web.logger.ErrorContext(ctx, "reconfiguring dns server", slogutil.KeyError, err)

		aghhttp.ErrorAndLog(ctx, web.logger, r, w, http.StatusInternalServerError, "%s", err)

		return
	}

	resp := &tlsConfig{
		tlsConfigSettingsExt: req,
		tlsConfigStatus:      tlsConfigStatusFromConf(status),
	}

	web.writeTLSConfigureResponse(ctx, w, r, resp)

	defer func() {
		// The background context is used because the TLSConfigChanged wraps context
		// with timeout on its own and shuts down the server, which handles current
		// request.  It is also should be done in a separate goroutine due to the
		// same reason.
		if restartHTTPS {
			go web.tlsConfigChanged(context.Background(), newTLSConf)
		}
	}()
}

// writeTLSConfigureResponse writes the response for the POST
// /control/tls/configure HTTP API.  All arguments must not be nil.
func (web *webAPI) writeTLSConfigureResponse(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	resp *tlsConfig,
) {
	web.marshalTLS(ctx, w, r, resp)

	rc := http.NewResponseController(w)
	err := rc.Flush()
	if err != nil {
		web.logger.ErrorContext(ctx, "flushing response", slogutil.KeyError, err)
	}
}

// reconfigureDNSServer reconfigures the DNS server.
func (web *webAPI) reconfigureDNSServer(ctx context.Context) (err error) {
	newConf, err := newServerConfig(
		&config.DNS,
		config.Clients.Sources,
		config.HTTPConfig.DoH,
		web.tlsConfProvider,
		web.httpReg,
		globalContext.clients.storage,
		web.confModifier,
	)
	if err != nil {
		return fmt.Errorf("generating forwarding dns server config: %w", err)
	}

	err = globalContext.dnsServer.Reconfigure(ctx, newConf)
	if err != nil {
		return fmt.Errorf("starting forwarding dns server: %w", err)
	}

	return nil
}

// marshalTLS encodes sensitive fields and writes data as JSON.  All arguments
// must not be nil.  data is modified in place.
func (web *webAPI) marshalTLS(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	data *tlsConfig,
) {
	if data.CertificateChain != "" {
		encoded := base64.StdEncoding.EncodeToString([]byte(data.CertificateChain))
		data.CertificateChain = encoded
	}

	if data.PrivateKey != "" {
		data.PrivateKeySaved = true
		data.PrivateKey = ""
	}

	aghhttp.WriteJSONResponseOK(ctx, web.logger, w, r, *data)
}

// type check
var _ service.Shutdowner = (*webAPI)(nil)

// Shutdown implements the [service.Shutdowner] interface.  It gracefully shuts
// down the web API server.  It always returns nil error.
func (web *webAPI) Shutdown(ctx context.Context) (err error) {
	web.close(ctx)

	return nil
}
