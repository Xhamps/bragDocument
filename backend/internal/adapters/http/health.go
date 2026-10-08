package http

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Check is one dependency probed by /readyz. Required dependencies make
// readiness fail with 503; optional ones are reported as "degraded" while the
// service keeps answering 200.
type Check struct {
	Name     string
	Required bool
	Ping     func(context.Context) error
}

// RegisterHealth adds /healthz (liveness) and /readyz (readiness).
func RegisterHealth(e *gin.Engine, checks []Check) {
	e.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	e.GET("/readyz", func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		status := http.StatusOK
		body := make(map[string]string, len(checks))
		for _, ch := range checks {
			switch err := ch.Ping(ctx); {
			case err == nil:
				body[ch.Name] = "ok"
			case ch.Required:
				body[ch.Name] = "down"
				status = http.StatusServiceUnavailable
			default:
				body[ch.Name] = "degraded"
			}
		}
		c.JSON(status, body)
	})
}
