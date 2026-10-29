package middleware

import (
	"time"

	"pulsa-backend/util"

	"github.com/gin-gonic/gin"
)

// RequestLogger — log setiap request lengkap dengan latency & status
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		latency := time.Since(start)

		entry := util.Log.WithFields(map[string]interface{}{
			"method":   c.Request.Method,
			"path":     c.Request.URL.Path,
			"status":   c.Writer.Status(),
			"latency":  latency.String(),
			"ip":       c.ClientIP(),
			"agent_id": c.GetInt64("agent_id"),
			"admin_id": c.GetInt64("admin_id"),
		})

		switch {
		case c.Writer.Status() >= 500:
			entry.Error("request failed")
		case c.Writer.Status() >= 400:
			entry.Warn("client error")
		default:
			entry.Info("request ok")
		}
	}
}

// Recovery — tangkap panic agar tidak crash & mudah di-debug
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				util.Log.WithField("panic", err).Error("[PANIC] recovered")
				util.InternalError(c, "internal server error")
				c.Abort()
			}
		}()
		c.Next()
	}
}
