package home

import (
	"context"
	"crypto/tls"
	"io/fs"
	"net/http"
	"net/netip"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghhttp"
	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/golibs/errors"
	"github.com/AdguardTeam/golibs/log"
	"github.com/AdguardTeam/golibs/netutil"
	"github.com/NYTimes/gziphandler"
	"github.com/lucas-clemente/quic-go"
	"github.com/lucas-clemente/quic-go/http3"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
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

	BindHost     netip.Addr
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

// Web is the web UI and API server.
type Web struct {
	conf *webConfig

	// TODO(a.garipov): Refactor all these servers.
	httpServer *http.Server

	// httpServerBeta is a server for new client.
	httpServerBeta *http.Server

	// handlerBeta is the handler for new client.
	handlerBeta http.Handler

	// installerBeta is the pre-install handler for new client.
	installerBeta http.Handler

	// httpsServer is the server that handles HTTPS traffic.  If it is not nil,
	// [Web.http3Server] must also not be nil.
	httpsServer httpsServer

	forceHTTPS bool
}

// newWeb creates a new instance of the web UI and API server.
func newWeb(conf *webConfig) (w *Web) {
	log.Info("web: initializing")

	w = &Web{
		conf: conf,
	}

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

	return w
}

// webCheckPortAvailable checks if port, which is considered an HTTPS port, is
// available, unless the HTTPS server isn't active.
//
// TODO(a.garipov): Adapt for HTTP/3.
func webCheckPortAvailable(port int) (ok bool) {
	if Context.web.httpsServer.server != nil {
		return true
	}

	return aghnet.CheckPort("tcp", netip.AddrPortFrom(config.BindHost, uint16(port))) == nil
}

// TLSConfigChanged updates the TLS configuration and restarts the HTTPS server
// if necessary.
func (web *Web) TLSConfigChanged(ctx context.Context, tlsConf tlsConfigSettings) {
	log.Debug("web: applying new tls configuration")
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
		shutdownSrv3(web.httpsServer.server3)

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
	for !web.httpsServer.inShutdown {
		printHTTPAddresses(aghhttp.SchemeHTTP)
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
	web.httpsServer.inShutdown = true
	web.httpsServer.cond.L.Unlock()

	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, shutdownTimeout)
	defer cancel()

	shutdownSrv(ctx, web.httpsServer.server)
	shutdownSrv3(web.httpsServer.server3)
	shutdownSrv(ctx, web.httpServer)
	shutdownSrv(ctx, web.httpServerBeta)

	log.Info("stopped http server")
}

func (web *Web) tlsServerLoop() {
	for {
		web.httpsServer.cond.L.Lock()
		if web.httpsServer.inShutdown {
			web.httpsServer.cond.L.Unlock()
			break
		}

		// this mechanism doesn't let us through until all conditions are met
		for !web.httpsServer.enabled { // sleep until necessary data is supplied
			web.httpsServer.cond.Wait()
			if web.httpsServer.inShutdown {
				web.httpsServer.cond.L.Unlock()
				return
			}
		}

		web.httpsServer.cond.L.Unlock()

		addr := netutil.JoinHostPort(web.conf.BindHost.String(), web.conf.PortHTTPS)
		web.httpsServer.server = &http.Server{
			ErrorLog: log.StdLog("web: https", log.DEBUG),
			Addr:     addr,
			TLSConfig: &tls.Config{
				Certificates: []tls.Certificate{web.httpsServer.cert},
				RootCAs:      Context.tlsRoots,
				CipherSuites: Context.tlsCipherIDs,
				MinVersion:   tls.VersionTLS12,
			},
			Handler:           withMiddlewares(Context.mux, limitRequestBody),
			ReadTimeout:       web.conf.ReadTimeout,
			ReadHeaderTimeout: web.conf.ReadHeaderTimeout,
			WriteTimeout:      web.conf.WriteTimeout,
		}

		printHTTPAddresses(aghhttp.SchemeHTTPS)

		if web.conf.serveHTTP3 {
			go web.mustStartHTTP3(addr)
		}

		log.Debug("web: starting https server")
		err := web.httpsServer.server.ListenAndServeTLS("", "")
		if !errors.Is(err, http.ErrServerClosed) {
			cleanupAlways()
			log.Fatalf("web: https: %s", err)
		}
	}
}

func (web *Web) mustStartHTTP3(address string) {
	defer log.OnPanic("web: http3")

	web.httpsServer.server3 = &http3.Server{
		// TODO(a.garipov): See if there is a way to use the error log as
		// well as timeouts here.
		Addr: address,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{web.httpsServer.cert},
			RootCAs:      Context.tlsRoots,
			CipherSuites: Context.tlsCipherIDs,
			MinVersion:   tls.VersionTLS12,
		},
		Handler: withMiddlewares(Context.mux, limitRequestBody),
	}

	log.Debug("web: starting http/3 server")
	err := web.httpsServer.server3.ListenAndServe()
	if !errors.Is(err, quic.ErrServerClosed) {
		cleanupAlways()
		log.Fatalf("web: http3: %s", err)
	}
}
