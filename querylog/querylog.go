package querylog

import (
	"net"
	"net/http"
	"time"

	"github.com/AdguardTeam/AdGuardHome/dnsfilter"
	"github.com/miekg/dns"
)

// DiskConfig - configuration settings that are stored on disk
type DiskConfig struct {
	Enabled  bool
	Interval uint32
	MemSize  uint32
}

// QueryLog - main interface
type QueryLog interface {
	// Close query log object
	Close()

	// Add a log entry
	Add(params AddParams)

	// WriteDiskConfig - write configuration
	WriteDiskConfig(dc *DiskConfig)
}

// Config - configuration object
type Config struct {
	Enabled  bool
	BaseDir  string // directory where log file is stored
	Interval uint32 // interval to rotate logs (in days)
	MemSize  uint32 // number of entries kept in memory before they are flushed to disk

	// Called when the configuration is changed by HTTP request
	ConfigModified func()

	// Register an HTTP handler
	HTTPRegister func(string, string, func(http.ResponseWriter, *http.Request))
}

// AddParams - parameters for Add()
type AddParams struct {
	Question   *dns.Msg
	Answer     *dns.Msg          // The response we sent to the client (optional)
	OrigAnswer *dns.Msg          // The response from an upstream server (optional)
	Result     *dnsfilter.Result // Filtering result (optional)
	Elapsed    time.Duration     // Time spent for processing the request
	ClientIP   net.IP
	Upstream   string
}

// New - create a new instance of the query log
func New(conf Config) QueryLog {
	return newQueryLog(conf)
}
