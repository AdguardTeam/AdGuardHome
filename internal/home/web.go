package home

import (
	"context"
	"crypto/tls"
	"io/fs"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/AdGuardHome/internal/aghtls"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/NYTimes/gziphandler"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// HTTP scheme constants.
const (
	schemeHTTP  = "http"
	schemeHTTPS = "https"
)

const (
	// readTimeout is the maximum duration for reading the entire request,
	// including the body.
	readTimeout = 60 * time.Second
	// readHdrTimeout is the amount of time allowed to read request headers.
	readHdrTimeout = 60 * time.Second
	// writeTimeout is the maximum duration before timing out writes of the
	// response.
	writeTimeout = 60 * time.Second
)

type webConfig struct {
	clientFS     fs.FS
	clientBetaFS fs.FS

	BindHost     net.IP
	BindPort     int
	BetaBindPort int
	PortHTTPS    int

	// ReadTimeout is an option to pass to http.Server for setting an
	// appropriate field.
	ReadTimeout time.Duration

	// ReadHeaderTimeout is an option to pass to http.Server for setting an
	// appropriate field.
	ReadHeaderTimeout time.Duration

	// WriteTimeout is an option to pass to http.Server for setting an
	// appropriate field.
	WriteTimeout time.Duration

	firstRun bool
}

// HTTPSServer - HTTPS Server
type HTTPSServer struct {
	server   *http.Server
	cond     *sync.Cond
	condLock sync.Mutex
	shutdown bool // if TRUE, don't restart the server
	enabled  bool
	cert     tls.Certificate
}

// Web - module object
type Web struct {
	conf        *webConfig
	forceHTTPS  bool
	httpServer  *http.Server // HTTP module
	httpsServer HTTPSServer  // HTTPS module

	// handlerBeta is the handler for new client.
	handlerBeta http.Handler
	// installerBeta is the pre-install handler for new client.
	installerBeta http.Handler

	// httpServerBeta is a server for new client.
	httpServerBeta *http.Server
}

// CreateWeb - create module
func CreateWeb(conf *webConfig) *Web {
	log.Info("Initialize web module")

	w := Web{}
	w.conf = conf

	clientFS := http.FileServer(http.FS(conf.clientFS))
	betaClientFS := http.FileServer(http.FS(conf.clientBetaFS))

	// if not configured, redirect / to /install.html, otherwise redirect /install.html to /
	Context.mux.Handle("/", withMiddlewares(clientFS, gziphandler.GzipHandler, optionalAuthHandler, postInstallHandler))
	w.handlerBeta = withMiddlewares(betaClientFS, gziphandler.GzipHandler, optionalAuthHandler, postInstallHandler)

	// add handlers for /install paths, we only need them when we're not configured yet
	if conf.firstRun {
		log.Info("This is the first launch of AdGuard Home, redirecting everything to /install.html ")
		Context.mux.Handle("/install.html", preInstallHandler(clientFS))
		w.installerBeta = preInstallHandler(betaClientFS)
		w.registerInstallHandlers()
		// This must be removed in API v1.
		w.registerBetaInstallHandlers()
	} else {
		registerControlHandlers()
	}

	w.httpsServer.cond = sync.NewCond(&w.httpsServer.condLock)
	return &w
}

// WebCheckPortAvailable - check if port is available
// BUT: if we are already using this port, no need
func WebCheckPortAvailable(port int) bool {
	return Context.web.httpsServer.server != nil ||
		aghnet.CheckPort("tcp", config.BindHost, port) == nil
}

// TLSConfigChanged updates the TLS configuration and restarts the HTTPS server
// if necessary.
func (web *Web) TLSConfigChanged(ctx context.Context, tlsConf tlsConfigSettings) {
	log.Debug("Web: applying new TLS configuration")
	web.conf.PortHTTPS = tlsConf.PortHTTPS
	web.forceHTTPS = (tlsConf.ForceHTTPS && tlsConf.Enabled && tlsConf.PortHTTPS != 0)

	enabled := tlsConf.Enabled &&
		tlsConf.PortHTTPS != 0 &&
		len(tlsConf.PrivateKeyData) != 0 &&
		len(tlsConf.CertificateChainData) != 0
	var cert tls.Certificate
	var err error
	if enabled {
		cert, err = tls.X509KeyPair(tlsConf.CertificateChainData, tlsConf.PrivateKeyData)
		if err != nil {
			log.Fatal(err)
		}
	}

	web.httpsServer.cond.L.Lock()
	if web.httpsServer.server != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, shutdownTimeout)
		shutdownSrv(ctx, web.httpsServer.server)
		cancel()
	}

	web.httpsServer.enabled = enabled
	web.httpsServer.cert = cert
	web.httpsServer.cond.Broadcast()
	web.httpsServer.cond.L.Unlock()
}

// Start - start serving HTTP requests
func (web *Web) Start() {
	log.Println("AdGuard Home is available at the following addresses:")

	// for https, we have a separate goroutine loop
	go web.tlsServerLoop()

	// this loop is used as an ability to change listening host and/or port
	for !web.httpsServer.shutdown {
		printHTTPAddresses(schemeHTTP)
		errs := make(chan error, 2)

		// Use an h2c handler to support unencrypted HTTP/2, e.g. for proxies.
		hdlr := h2c.NewHandler(withMiddlewares(Context.mux, limitRequestBody), &http2.Server{})

		// Create a new instance, because the Web is not usable after Shutdown.
		hostStr := web.conf.BindHost.String()
		web.httpServer = &http.Server{
			ErrorLog:          log.StdLog("web: plain", log.DEBUG),
			Addr:              netutil.JoinHostPort(hostStr, web.conf.BindPort),
			Handler:           hdlr,
			ReadTimeout:       web.conf.ReadTimeout,
			ReadHeaderTimeout: web.conf.ReadHeaderTimeout,
			WriteTimeout:      web.conf.WriteTimeout,
		}
		go func() {
			defer log.OnPanic("web: plain")

			errs <- web.httpServer.ListenAndServe()
		}()

		web.startBetaServer(hostStr)

		err := <-errs
		if !errors.Is(err, http.ErrServerClosed) {
			cleanupAlways()
			log.Fatal(err)
		}

		// We use ErrServerClosed as a sign that we need to rebind on a new
		// address, so go back to the start of the loop.
	}
}

// startBetaServer starts the beta HTTP server if necessary.
func (web *Web) startBetaServer(hostStr string) {
	if web.conf.BetaBindPort == 0 {
		return
	}

	// Use an h2c handler to support unencrypted HTTP/2, e.g. for proxies.
	hdlr := h2c.NewHandler(
		withMiddlewares(Context.mux, limitRequestBody, web.wrapIndexBeta),
		&http2.Server{},
	)

	web.httpServerBeta = &http.Server{
		ErrorLog:          log.StdLog("web: plain: beta", log.DEBUG),
		Addr:              netutil.JoinHostPort(hostStr, web.conf.BetaBindPort),
		Handler:           hdlr,
		ReadTimeout:       web.conf.ReadTimeout,
		ReadHeaderTimeout: web.conf.ReadHeaderTimeout,
		WriteTimeout:      web.conf.WriteTimeout,
	}
	go func() {
		defer log.OnPanic("web: plain: beta")

		betaErr := web.httpServerBeta.ListenAndServe()
		if betaErr != nil && !errors.Is(betaErr, http.ErrServerClosed) {
			log.Error("starting beta http server: %s", betaErr)
		}
	}()
}

// Close gracefully shuts down the HTTP servers.
func (web *Web) Close(ctx context.Context) {
	log.Info("stopping http server...")

	web.httpsServer.cond.L.Lock()
	web.httpsServer.shutdown = true
	web.httpsServer.cond.L.Unlock()

	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, shutdownTimeout)
	defer cancel()

	shutdownSrv(ctx, web.httpsServer.server)
	shutdownSrv(ctx, web.httpServer)
	shutdownSrv(ctx, web.httpServerBeta)

	log.Info("stopped http server")
}

func (web *Web) tlsServerLoop() {
	for {
		web.httpsServer.cond.L.Lock()
		if web.httpsServer.shutdown {
			web.httpsServer.cond.L.Unlock()
			break
		}

		// this mechanism doesn't let us through until all conditions are met
		for !web.httpsServer.enabled { // sleep until necessary data is supplied
			web.httpsServer.cond.Wait()
			if web.httpsServer.shutdown {
				web.httpsServer.cond.L.Unlock()
				return
			}
		}

		web.httpsServer.cond.L.Unlock()

		// prepare HTTPS server
		address := netutil.JoinHostPort(web.conf.BindHost.String(), web.conf.PortHTTPS)
		web.httpsServer.server = &http.Server{
			ErrorLog: log.StdLog("web: https", log.DEBUG),
			Addr:     address,
			TLSConfig: &tls.Config{
				Certificates: []tls.Certificate{web.httpsServer.cert},
				RootCAs:      Context.tlsRoots,
				CipherSuites: aghtls.SaferCipherSuites(),
				MinVersion:   tls.VersionTLS12,
			},
			Handler:           withMiddlewares(Context.mux, limitRequestBody),
			ReadTimeout:       web.conf.ReadTimeout,
			ReadHeaderTimeout: web.conf.ReadHeaderTimeout,
			WriteTimeout:      web.conf.WriteTimeout,
		}

		printHTTPAddresses(schemeHTTPS)
		err := web.httpsServer.server.ListenAndServeTLS("", "")
		if err != http.ErrServerClosed {
			cleanupAlways()
			log.Fatal(err)
		}
	}
}
