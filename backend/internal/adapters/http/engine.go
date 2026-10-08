package http

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NewEngine builds the Gin engine with the standard middleware chain and the
// /metrics endpoint. Feature handlers register their routes on it afterwards.
//
// Middleware order: recovery → request id → logger → metrics. Recovery runs
// outermost so it catches panics from everything below; the request id is
// already in the context by the time a handler can panic.
func NewEngine(reg *prometheus.Registry) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	e := gin.New()
	m := newMetrics(reg)
	e.Use(Recovery(), RequestID(), Logger(), m.Middleware())
	e.GET("/metrics", gin.WrapH(promhttp.HandlerFor(reg, promhttp.HandlerOpts{})))
	return e
}
