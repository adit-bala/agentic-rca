package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
)

const listenAddr = ":9090"

// Common metrics that can be used across services
var (
	// APIRequestLatency tracks the latency of API requests
	APIRequestLatency = Histogram(
		"api_request_latency_seconds",
		"Latency of API requests in seconds",
		prometheus.DefBuckets,
		"service", "endpoint", "method",
	)

	// APIRequestTotal tracks the total number of API requests
	APIRequestTotal = Counter(
		"api_requests_total",
		"Total number of API requests",
		"service", "endpoint", "method", "status",
	)

	// ActiveConnections tracks the number of active connections
	ActiveConnections = Gauge(
		"active_connections",
		"Number of active connections",
		"service",
	)

	// ErrorTotal tracks the total number of errors
	ErrorTotal = Counter(
		"errors_total",
		"Total number of errors",
		"service", "type",
	)
)

func Counter(name, help string, labelKeys ...string) *prometheus.CounterVec {
	return promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: name,
			Help: help,
		},
		labelKeys,
	)
}

func Inc(c *prometheus.CounterVec, labels prometheus.Labels, v float64) {
	c.With(labels).Add(v)
}

func Gauge(name, help string, labelKeys ...string) *prometheus.GaugeVec {
	return promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: name,
			Help: help,
		},
		labelKeys,
	)
}

func Set(g *prometheus.GaugeVec, labels prometheus.Labels, v float64) {
	g.With(labels).Set(v)
}

func Histogram(name, help string, buckets []float64, labelKeys ...string) *prometheus.HistogramVec {
	return promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    name,
			Help:    help,
			Buckets: buckets, // e.g. prometheus.DefBuckets
		},
		labelKeys,
	)
}

func Observe(h *prometheus.HistogramVec, labels prometheus.Labels, v float64) {
	h.With(labels).Observe(v)
}

func Start(logger zerolog.Logger) {
	go func() {
		http.Handle("/metrics", promhttp.Handler())

		if err := http.ListenAndServe(listenAddr, nil); err != nil && err != http.ErrServerClosed {
			logger.Fatal().Err(err).Msg("metrics server error")
		}
	}()
}
