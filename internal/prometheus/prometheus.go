package prometheus

import (
	"net"
	"net/http"
	"strconv"

	"github.com/AdguardTeam/AdGuardHome/internal/stats"
	"github.com/AdguardTeam/golibs/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Config - Prometheus configuration
type Config struct {
	Enabled   bool   `yaml:"enabled"`
	BindHost  string `yaml:"bind_host"`
	BindPort  int    `yaml:"bind_port"`
	Namespace string `yaml:"namespace"`
}

type Server struct {
	conf Config
	mux  *http.ServeMux

	dnsRequests *prometheus.CounterVec
	dnsDuration prometheus.Histogram
}

func Create(config Config) *Server {
	s := Server{}
	s.conf.Enabled = config.Enabled
	s.conf.BindHost = config.BindHost
	s.conf.BindPort = config.BindPort
	s.conf.Namespace = config.Namespace

	if !s.conf.Enabled {
		return &s
	}

	s.mux = http.NewServeMux()
	s.mux.Handle("/metrics", promhttp.Handler())

	s.dnsRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name:      "requests_total",
			Subsystem: "dns",
			Namespace: s.conf.Namespace,
			Help:      "Counter of DNS requests made per result type.",
		},
		[]string{
			"result",
		},
	)
	s.dnsDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:      "request_duration_seconds",
		Subsystem: "dns",
		Namespace: s.conf.Namespace,
		Buckets:   prometheus.ExponentialBuckets(0.00025, 2, 16), // from 0.25ms to 8 seconds
	})

	return &s
}

// Update counters
func (s *Server) Update(e stats.Entry) {
	if !s.conf.Enabled {
		return
	}

	labels := prometheus.Labels{
		"result": e.Result.String(),
	}

	s.dnsRequests.With(labels).Inc()
	s.dnsDuration.Observe(float64(e.Time) / 1000000)
}

// Start server
func (s *Server) Start() {
	if !s.conf.Enabled {
		return
	}

	port := strconv.Itoa(s.conf.BindPort)
	addr := net.JoinHostPort(s.conf.BindHost, port)
	go func() {
		if err := http.ListenAndServe(addr, s.mux); err != nil {
			log.Error("Failed to run Prometheus server: %s", err)
		}
	}()
}
