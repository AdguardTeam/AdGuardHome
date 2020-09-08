package home

import (
	"crypto/x509"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/AdguardTeam/AdGuardHome/dhcpd"
	"github.com/AdguardTeam/AdGuardHome/dnsfilter"
	"github.com/AdguardTeam/AdGuardHome/dnsforward"
	"github.com/AdguardTeam/AdGuardHome/querylog"
	"github.com/AdguardTeam/AdGuardHome/stats"
	"github.com/AdguardTeam/AdGuardHome/update"
	"github.com/AdguardTeam/AdGuardHome/util"

	"github.com/AdguardTeam/golibs/log"
)

// Global context
type homeContext struct {
	// Modules
	// --

	clients    clientsContainer     // per-client-settings module
	stats      stats.Stats          // statistics module
	queryLog   querylog.QueryLog    // query log module
	dnsServer  *dnsforward.Server   // DNS module
	rdns       *RDNS                // rDNS module
	whois      *Whois               // WHOIS module
	dnsFilter  *dnsfilter.Dnsfilter // DNS filtering module
	dhcpServer *dhcpd.Server        // DHCP module
	auth       *Auth                // HTTP authentication module
	filters    Filtering            // DNS filtering module
	web        *Web                 // Web (HTTP, HTTPS) module
	tls        *TLSMod              // TLS module
	autoHosts  util.AutoHosts       // IP-hostname pairs taken from system configuration (e.g. /etc/hosts) files
	updater    *update.Updater

	// Runtime properties
	// --

	controlLock      sync.Mutex
	configFilename   string         // Config filename (can be overridden via the command line arguments)
	workDir          string         // Location of our directory, used to protect against CWD being somewhere else
	firstRun         bool           // if set to true, don't run any services except HTTP web inteface, and serve only first-run html
	pidFileName      string         // PID file name.  Empty if no PID file was created.
	disableUpdate    bool           // If set, don't check for updates
	tlsRoots         *x509.CertPool // list of root CAs for TLSv1.2
	tlsCiphers       []uint16       // list of TLS ciphers to use
	transport        *http.Transport
	client           *http.Client
	appSignalChannel chan os.Signal // Channel for receiving OS signals by the console app
	// runningAsService flag is set to true when options are passed from the service runner
	runningAsService bool
}

// getDataDir returns path to the directory where we store databases and filters
func (c *homeContext) getDataDir() string {
	return filepath.Join(c.workDir, dataDir)
}

// Context - a global context object
var Context homeContext

func (c *homeContext) cleanup() {
	log.Info("Stopping AdGuard Home")

	if c.web != nil {
		c.web.Close()
		c.web = nil
	}
	if c.auth != nil {
		c.auth.Close()
		c.auth = nil
	}

	err := c.stopDNSServer()
	if err != nil {
		log.Error("Couldn't stop DNS server: %s", err)
	}

	if c.dhcpServer != nil {
		c.dhcpServer.Stop()
	}

	c.autoHosts.Close()

	if c.tls != nil {
		c.tls.Close()
		c.tls = nil
	}
}

// This function is called before application exits
func (c *homeContext) cleanupAlways() {
	if len(c.pidFileName) != 0 {
		_ = os.Remove(c.pidFileName)
	}
	log.Info("Stopped")
}
