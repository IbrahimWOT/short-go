package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"short-go/internal/config"
)

// SetupRouter configures middleware, health checks, and API endpoints.
func SetupRouter(cfg *config.Config, urlHandler *URLHandler) *gin.Engine {
	if cfg.Server.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Global Middleware
	router.Use(gin.Recovery())
	router.Use(Logger())
	router.Use(Recovery())
	router.Use(SecurityHeaders())

	// Liveness and readiness probe
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"service": "short-go",
			"env":     cfg.Server.Env,
		})
	})

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		v1.POST("/shorten", urlHandler.Shorten)
		v1.GET("/analytics/:shortCode", urlHandler.GetAnalytics)
	}

	// Redirection endpoint at root level: GET /:shortCode
	// Note: skips /health, /api, etc.
	router.GET("/:shortCode", func(c *gin.Context) {
		code := c.Param("shortCode")
		if code == "health" || code == "favicon.ico" {
			c.Status(http.StatusNotFound)
			return
		}
		urlHandler.Redirect(c)
	})

	return router
}
