package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/its-me-ojas/event-driven-feed/internal/metrics"
)

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func NewResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		w, http.StatusOK,
	}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := NewResponseWriter(w)
		next.ServeHTTP(rw, r)
		duration := time.Since(start).Seconds()
		route := mux.CurrentRoute(r)
		pathTemplate, _ := route.GetPathTemplate()
		if pathTemplate == "" {
			pathTemplate = "unknown"
		}
		status := strconv.Itoa(rw.statusCode)
		metrics.HttpRequestDuration.WithLabelValues(r.Method, pathTemplate, status).Observe(duration)
		metrics.HttpRequestDuration.WithLabelValues(r.Method, pathTemplate).Observe(duration)
	})
}
