package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// RequestLogger returns a Gin middleware that logs each request.
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)
		status := c.Writer.Status()

		attrs := []any{
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", status,
			"duration_ms", duration.Milliseconds(),
			"ip", c.ClientIP(),
			"content_length", c.Request.ContentLength,
		}

		switch {
		case status == 401 || status == 403:
			slog.Warn("request", attrs...)
		case status >= 500:
			slog.Error("request", attrs...)
		default:
			slog.Info("request", attrs...)
		}
	}
}
