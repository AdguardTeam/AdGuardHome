package querylog

import (
	"net"
	"net/http"
	"time"

	"github.com/AdguardTeam/AdGuardHome/dnsfilter"
	"github.com/miekg/dns"
)

// QueryLog - main interface
type QueryLog interface {
	Start()

	// Close query log object
	Close()

	// Add a log entry
	Add(params AddParams)

	// WriteDiskConfig - write configuration
	WriteDiskConfig(c *Config)
}

// Config - configuration object
type Config struct {
	Enabled           bool   // enable the module
	FileEnabled       bool   // write logs to file
	BaseDir           string // directory where log file is stored
	Interval          uint32 // interval to rotate logs (in days)
	MemSize           uint32 // number of entries kept in memory before they are flushed to disk
	AnonymizeClientIP bool   // anonymize clients' IP addresses

	// Called when the configuration is changed by HTTP request
	ConfigModified func()

	// Register an HTTP handler
	HTTPRegister func(string, string, func(http.ResponseWriter, *http.Request))
}

// AddParams - parameters for Add()
type AddParams struct {
	Question    *dns.Msg
	Answer      *dns.Msg          // The response we sent to the client (optional)
	OrigAnswer  *dns.Msg          // The response from an upstream server (optional)
	Result      *dnsfilter.Result // Filtering result (optional)
	Elapsed     time.Duration     // Time spent for processing the request
	ClientIP    net.IP
	Upstream    string // Upstream server URL
	ClientProto string // Protocol for the client connection: "" (plain), "doh", "dot"
}

// New - create a new instance of the query log
func New(conf Config) QueryLog {
	return newQueryLog(conf)
}
