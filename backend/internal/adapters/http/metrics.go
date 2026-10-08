package http

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
)

type metrics struct {
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
}

func newMetrics(reg prometheus.Registerer) *metrics {
	m := &metrics{
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "HTTP requests by method, route template, and status.",
		}, []string{"method", "route", "status"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency by method, route template, and status.",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "route", "status"}),
	}
	reg.MustRegister(m.requests, m.duration)
	return m
}

// Middleware records one observation per request, labelled by the route
// template (not the raw path) so cardinality stays bounded.
func (m *metrics) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		status := strconv.Itoa(c.Writer.Status())
		m.requests.WithLabelValues(c.Request.Method, route, status).Inc()
		m.duration.WithLabelValues(c.Request.Method, route, status).Observe(time.Since(start).Seconds())
	}
}
