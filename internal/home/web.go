package home

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/AdguardTeam/AdGuardHome/internal/aghnet"
	"github.com/AdguardTeam/golibs/log"
	"github.com/NYTimes/gziphandler"
	"github.com/gobuffalo/packr"
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
	firstRun     bool
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

	// Initialize and run the admin Web interface
	box := packr.NewBox("../../build/static")
	boxBeta := packr.NewBox("../../build2/static")

	// if not configured, redirect / to /install.html, otherwise redirect /install.html to /
	Context.mux.Handle("/", withMiddlewares(http.FileServer(box), gziphandler.GzipHandler, optionalAuthHandler, postInstallHandler))
	w.handlerBeta = withMiddlewares(http.FileServer(boxBeta), gziphandler.GzipHandler, optionalAuthHandler, postInstallHandler)

	// add handlers for /install paths, we only need them when we're not configured yet
	if conf.firstRun {
		log.Info("This is the first launch of AdGuard Home, redirecting everything to /install.html ")
		Context.mux.Handle("/install.html", preInstallHandler(http.FileServer(box)))
		w.installerBeta = preInstallHandler(http.FileServer(boxBeta))
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
	alreadyRunning := false
	if Context.web.httpsServer.server != nil {
		alreadyRunning = true
	}
	if !alreadyRunning {
		err := aghnet.CheckPortAvailable(config.BindHost, port)
		if err != nil {
			return false
		}
	}
	return true
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
		shutdownSrv(ctx, cancel, web.httpsServer.server)
	}

	web.httpsServer.enabled = enabled
	web.httpsServer.cert = cert
	web.httpsServer.cond.Broadcast()
	web.httpsServer.cond.L.Unlock()
}

// Start - start serving HTTP requests
func (web *Web) Start() {
	// for https, we have a separate goroutine loop
	go web.tlsServerLoop()

	// this loop is used as an ability to change listening host and/or port
	for !web.httpsServer.shutdown {
		printHTTPAddresses(schemeHTTP)
		errs := make(chan error, 2)

		hostStr := web.conf.BindHost.String()
		// we need to have new instance, because after Shutdown() the Server is not usable
		web.httpServer = &http.Server{
			ErrorLog:          log.StdLog("web: http", log.DEBUG),
			Addr:              net.JoinHostPort(hostStr, strconv.Itoa(web.conf.BindPort)),
			Handler:           withMiddlewares(Context.mux, limitRequestBody),
			ReadTimeout:       web.conf.ReadTimeout,
			ReadHeaderTimeout: web.conf.ReadHeaderTimeout,
			WriteTimeout:      web.conf.WriteTimeout,
		}
		go func() {
			errs <- web.httpServer.ListenAndServe()
		}()

		if web.conf.BetaBindPort != 0 {
			web.httpServerBeta = &http.Server{
				ErrorLog:          log.StdLog("web: http", log.DEBUG),
				Addr:              net.JoinHostPort(hostStr, strconv.Itoa(web.conf.BetaBindPort)),
				Handler:           withMiddlewares(Context.mux, limitRequestBody, web.wrapIndexBeta),
				ReadTimeout:       web.conf.ReadTimeout,
				ReadHeaderTimeout: web.conf.ReadHeaderTimeout,
				WriteTimeout:      web.conf.WriteTimeout,
			}
			go func() {
				betaErr := web.httpServerBeta.ListenAndServe()
				if betaErr != nil {
					log.Error("starting beta http server: %s", betaErr)
				}
			}()
		}

		err := <-errs
		if err != http.ErrServerClosed {
			cleanupAlways()
			log.Fatal(err)
		}
		// We use ErrServerClosed as a sign that we need to rebind on new address, so go back to the start of the loop
	}
}

// Close gracefully shuts down the HTTP servers.
func (web *Web) Close(ctx context.Context) {
	log.Info("stopping http server...")

	web.httpsServer.cond.L.Lock()
	web.httpsServer.shutdown = true
	web.httpsServer.cond.L.Unlock()

	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, shutdownTimeout)

	shutdownSrv(ctx, cancel, web.httpsServer.server)
	shutdownSrv(ctx, cancel, web.httpServer)
	shutdownSrv(ctx, cancel, web.httpServerBeta)

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
		address := net.JoinHostPort(web.conf.BindHost.String(), strconv.Itoa(web.conf.PortHTTPS))
		web.httpsServer.server = &http.Server{
			ErrorLog: log.StdLog("web: https", log.DEBUG),
			Addr:     address,
			TLSConfig: &tls.Config{
				Certificates: []tls.Certificate{web.httpsServer.cert},
				MinVersion:   tls.VersionTLS12,
				RootCAs:      Context.tlsRoots,
				CipherSuites: Context.tlsCiphers,
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
