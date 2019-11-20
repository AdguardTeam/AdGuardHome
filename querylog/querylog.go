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
}

// QueryLog - main interface
type QueryLog interface {
	// Close query log object
	Close()

	// Add a log entry
	Add(question *dns.Msg, answer *dns.Msg, result *dnsfilter.Result, elapsed time.Duration, ip net.IP, upstream string)

	// WriteDiskConfig - write configuration
	WriteDiskConfig(dc *DiskConfig)
}

// Config - configuration object
type Config struct {
	Enabled  bool
	BaseDir  string // directory where log file is stored
	Interval uint32 // interval to rotate logs (in days)

	// Called when the configuration is changed by HTTP request
	ConfigModified func()

	// Register an HTTP handler
	HTTPRegister func(string, string, func(http.ResponseWriter, *http.Request))
}

// New - create a new instance of the query log
func New(conf Config) QueryLog {
	return newQueryLog(conf)
}
