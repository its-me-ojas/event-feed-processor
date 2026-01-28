package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Processor metrics
	EventsProcessed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "feed_events_processed_total",
		Help: "The total number of processed",
	}, []string{"status", "type"})

	EventDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "feed_event_processing_duration_seconds",
		Help:    "Time taken to process an event",
		Buckets: prometheus.DefBuckets,
	}, []string{"type"})

	// API Metrics
	HttpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of Http requets",
	}, []string{"method", "path", "status"})

	HttpRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "Duration of HTTP requests",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path"})
)
