package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestDNSMetrics(t *testing.T) {
	// Create a new counter for isolated testing
	testCounter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "test_dns_queries_by_result_total",
		Help: "Test counter for DNS queries by processing result",
	}, []string{"result"})

	// Test incrementing queries by result
	testCounter.WithLabelValues(ResultFiltered).Inc()
	testCounter.WithLabelValues(ResultNotFiltered).Inc()
	testCounter.WithLabelValues(ResultSafeBrowsing).Inc()
	testCounter.WithLabelValues(ResultNotFiltered).Inc() // Add another not filtered

	// Verify result counters
	filteredValue := testutil.ToFloat64(testCounter.WithLabelValues(ResultFiltered))
	if filteredValue != 1 {
		t.Errorf("Expected filtered queries to be 1, got %f", filteredValue)
	}

	notFilteredValue := testutil.ToFloat64(testCounter.WithLabelValues(ResultNotFiltered))
	if notFilteredValue != 2 {
		t.Errorf("Expected not filtered queries to be 2, got %f", notFilteredValue)
	}

	safeBrowsingValue := testutil.ToFloat64(testCounter.WithLabelValues(ResultSafeBrowsing))
	if safeBrowsingValue != 1 {
		t.Errorf("Expected safe browsing queries to be 1, got %f", safeBrowsingValue)
	}
}

func TestRegisterDNSMetrics(t *testing.T) {
	registry := prometheus.NewRegistry()

	// This should not panic
	RegisterDNSMetrics(registry)

	// Registering again should panic due to duplicate registration
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when registering metrics twice")
		}
	}()
	RegisterDNSMetrics(registry)
}
