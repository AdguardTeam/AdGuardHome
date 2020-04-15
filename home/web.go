package home

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"strconv"
	"sync"

	"github.com/AdguardTeam/AdGuardHome/util"
	"github.com/AdguardTeam/golibs/log"
	"github.com/NYTimes/gziphandler"
	"github.com/gobuffalo/packr"
)

type WebConfig struct {
	firstRun  bool
	BindHost  string
	BindPort  int
	PortHTTPS int
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
	conf        *WebConfig
	forceHTTPS  bool
	portHTTPS   int
	httpServer  *http.Server // HTTP module
	httpsServer HTTPSServer  // HTTPS module
}

// CreateWeb - create module
func CreateWeb(conf *WebConfig) *Web {
	log.Info("Initialize web module")

	w := Web{}
	w.conf = conf

	// Initialize and run the admin Web interface
	box := packr.NewBox("../build/static")

	// if not configured, redirect / to /install.html, otherwise redirect /install.html to /
	http.Handle("/", postInstallHandler(optionalAuthHandler(gziphandler.GzipHandler(http.FileServer(box)))))

	// add handlers for /install paths, we only need them when we're not configured yet
	if conf.firstRun {
		log.Info("This is the first launch of AdGuard Home, redirecting everything to /install.html ")
		http.Handle("/install.html", preInstallHandler(http.FileServer(box)))
		w.registerInstallHandlers()
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
		err := util.CheckPortAvailable(config.BindHost, port)
		if err != nil {
			return false
		}
	}
	return true
}

// TLSConfigChanged - called when TLS configuration has changed
func (web *Web) TLSConfigChanged(tlsConf tlsConfigSettings) {
	log.Debug("Web: applying new TLS configuration")
	web.conf.PortHTTPS = tlsConf.PortHTTPS
	web.forceHTTPS = (tlsConf.ForceHTTPS && tlsConf.Enabled && tlsConf.PortHTTPS != 0)
	web.portHTTPS = tlsConf.PortHTTPS

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
		_ = web.httpsServer.server.Shutdown(context.TODO())
	}
	web.httpsServer.enabled = enabled
	web.httpsServer.cert = cert
	web.httpsServer.cond.Broadcast()
	web.httpsServer.cond.L.Unlock()
}

// Start - start serving HTTP requests
func (web *Web) Start() {
	// for https, we have a separate goroutine loop
	go web.httpServerLoop()

	// this loop is used as an ability to change listening host and/or port
	for !web.httpsServer.shutdown {
		printHTTPAddresses("http")

		// we need to have new instance, because after Shutdown() the Server is not usable
		address := net.JoinHostPort(web.conf.BindHost, strconv.Itoa(web.conf.BindPort))
		web.httpServer = &http.Server{
			Addr: address,
		}
		err := web.httpServer.ListenAndServe()
		if err != http.ErrServerClosed {
			cleanupAlways()
			log.Fatal(err)
		}
		// We use ErrServerClosed as a sign that we need to rebind on new address, so go back to the start of the loop
	}
}

// Close - stop HTTP server, possibly waiting for all active connections to be closed
func (web *Web) Close() {
	log.Info("Stopping HTTP server...")
	web.httpsServer.cond.L.Lock()
	web.httpsServer.shutdown = true
	web.httpsServer.cond.L.Unlock()
	if web.httpsServer.server != nil {
		_ = web.httpsServer.server.Shutdown(context.TODO())
	}
	if web.httpServer != nil {
		_ = web.httpServer.Shutdown(context.TODO())
	}

	log.Info("Stopped HTTP server")
}

func (web *Web) httpServerLoop() {
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
		address := net.JoinHostPort(web.conf.BindHost, strconv.Itoa(web.conf.PortHTTPS))
		web.httpsServer.server = &http.Server{
			Addr: address,
			TLSConfig: &tls.Config{
				Certificates: []tls.Certificate{web.httpsServer.cert},
				MinVersion:   tls.VersionTLS12,
				RootCAs:      Context.tlsRoots,
				CipherSuites: Context.tlsCiphers,
			},
		}

		printHTTPAddresses("https")
		err := web.httpsServer.server.ListenAndServeTLS("", "")
		if err != http.ErrServerClosed {
			cleanupAlways()
			log.Fatal(err)
		}
	}
}
