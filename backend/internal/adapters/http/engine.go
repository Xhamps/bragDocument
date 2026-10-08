package http

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// maxBodyBytes caps every request body; no route accepts more than a few KB.
const maxBodyBytes = 1 << 20

// NewEngine builds the Gin engine with the standard middleware chain and the
// /metrics endpoint. Feature handlers register their routes on it afterwards.
//
// Middleware order: request id → max body → logger → metrics → recovery. Recovery runs
// innermost: the three middleware outside it cannot realistically panic, and
// placing recovery inside them means a recovered panic is still logged and
// counted as a 500.
func NewEngine(reg *prometheus.Registry) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	e := gin.New()
	m := newMetrics(reg)
	e.Use(RequestID(), MaxBody(maxBodyBytes), Logger(), m.Middleware(), Recovery())
	e.GET("/metrics", gin.WrapH(promhttp.HandlerFor(reg, promhttp.HandlerOpts{})))
	return e
}
