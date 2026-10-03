package handler

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger returns a gin.HandlerFunc that logs HTTP request details.
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()

		if raw != "" {
			path = path + "?" + raw
		}

		log.Printf("[HTTP] %d | %13v | %15s | %-7s %s",
			statusCode,
			latency,
			clientIP,
			method,
			path,
		)
	}
}

// Recovery returns a middleware that recovers from any panic and returns 500 JSON.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[PANIC RECOVERED] %v", r)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error":   "internal server error",
					"message": "an unexpected error occurred",
				})
			}
		}()
		c.Next()
	}
}

// SecurityHeaders adds essential defensive HTTP headers.
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Next()
	}
}
