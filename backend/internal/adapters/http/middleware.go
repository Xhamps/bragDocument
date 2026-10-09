package http

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/xhamps/bragdocument/backend/internal/telemetry"
)

// HeaderRequestID is read from the request and always set on the response.
const HeaderRequestID = "X-Request-ID"

// RequestID reads or generates a request id and stores it in the request
// context so logs and error bodies can carry it.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(HeaderRequestID)
		if id == "" {
			id = uuid.NewString()
		}
		c.Header(HeaderRequestID, id)
		c.Request = c.Request.WithContext(telemetry.WithRequestID(c.Request.Context(), id))
		c.Next()
	}
}

// MaxBody caps request bodies at n bytes; a larger body fails at read time
// with *http.MaxBytesError, which bindJSON reports as 413.
func MaxBody(n int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, n)
		c.Next()
	}
}

// Logger writes one structured line per request.
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		slog.LogAttrs(c.Request.Context(), slog.LevelInfo, "http request",
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status", c.Writer.Status()),
			slog.Duration("latency", time.Since(start)),
		)
	}
}

// Recovery turns a panic into a 500 with the request id, and logs the stack.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				slog.LogAttrs(c.Request.Context(), slog.LevelError, "panic recovered",
					slog.Any("panic", r),
					slog.String("stack", string(debug.Stack())),
				)
				c.AbortWithStatusJSON(http.StatusInternalServerError,
					errorBody(c, "internal", "internal server error", nil))
			}
		}()
		c.Next()
	}
}

// CORS lets the web app at origin (APP_URL) call the api cross-origin and
// answers preflights with 204. An empty origin disables it.
func CORS(origin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if origin == "" || c.GetHeader("Origin") != origin {
			c.Next()
			return
		}
		h := c.Writer.Header()
		h.Set("Access-Control-Allow-Origin", origin)
		h.Add("Vary", "Origin")
		h.Set("Access-Control-Expose-Headers", HeaderRequestID+", Content-Disposition")
		if c.Request.Method == http.MethodOptions {
			h.Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE")
			h.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, "+HeaderRequestID)
			h.Set("Access-Control-Max-Age", "600")
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}
