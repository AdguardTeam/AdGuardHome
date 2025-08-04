package metrics

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// DNS query result types matching internal/stats package
const (
	ResultNotFiltered  = "not_filtered"
	ResultFiltered     = "filtered"
	ResultSafeBrowsing = "safe_browsing"
	ResultSafeSearch   = "safe_search"
	ResultParental     = "parental"
	ResultUnknown      = "unknown"
)

// DNSQueries tracks DNS queries by their processing result
var DNSQueries = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "adguard_dns_queries_total",
	Help: "Total number of DNS queries by processing result",
}, []string{"result"})

// DNSResponseTime tracks DNS query response times using native exponential histogram
var DNSResponseTime = prometheus.NewHistogramVec(prometheus.HistogramOpts{
	Name:                        "adguard_dns_response_time_seconds",
	Help:                        "DNS query response time in seconds",
	NativeHistogramBucketFactor: 1.1,
}, []string{"result"})

// RegisterDNSMetrics registers all DNS-related metrics with the provided registry
func RegisterDNSMetrics(registry *prometheus.Registry) {
	registry.MustRegister(DNSQueries)
	registry.MustRegister(DNSResponseTime)
}

// IncrementDNSQueryByResult increments counters for a specific query result type
func IncrementDNSQueryByResult(result string) {
	DNSQueries.WithLabelValues(result).Inc()
}

// ObserveDNSResponseTime records a DNS query response time
func ObserveDNSResponseTime(result string, duration time.Duration) {
	DNSResponseTime.WithLabelValues(result).Observe(duration.Seconds())
}
