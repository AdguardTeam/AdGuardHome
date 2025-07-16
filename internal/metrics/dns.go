package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// DNS query result types matching internal/stats package
const (
	ResultNotFiltered  = "not_filtered"
	ResultFiltered     = "filtered"
	ResultSafeBrowsing = "safe_browsing"
	ResultSafeSearch   = "safe_search"
	ResultParental     = "parental"
)

// DNSQueriesByResult tracks DNS queries by their processing result
var DNSQueriesByResult = prometheus.NewCounterVec(prometheus.CounterOpts{
	Name: "adguard_dns_queries_by_result_total",
	Help: "Total number of DNS queries by processing result",
}, []string{"result"})

// RegisterDNSMetrics registers all DNS-related metrics with the provided registry
func RegisterDNSMetrics(registry *prometheus.Registry) {
	registry.MustRegister(DNSQueriesByResult)
}

// IncrementDNSQueryByResult increments counters for a specific query result type
func IncrementDNSQueryByResult(result string) {
	DNSQueriesByResult.WithLabelValues(result).Inc()
}
