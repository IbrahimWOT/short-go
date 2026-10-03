package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"short-go/internal/domain"
)

// URLHandler handles HTTP requests for shortening, redirection, and analytics.
type URLHandler struct {
	service domain.URLService
}

// NewURLHandler creates a new instance of URLHandler.
func NewURLHandler(service domain.URLService) *URLHandler {
	return &URLHandler{service: service}
}

// Shorten godoc
// @Summary Shorten a long URL
// @Description Accepts a long URL, generates a Base62 short key or custom alias, stores it and caches in Redis.
// @Accept json
// @Produce json
// @Param request body domain.ShortenRequest true "URL Shortening Request"
// @Success 201 {object} domain.ShortenResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 409 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/shorten [post]
func (h *URLHandler) Shorten(c *gin.Context) {
	var req domain.ShortenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_request",
			"message": "original_url is required and must be a valid URL",
		})
		return
	}

	resp, err := h.service.Shorten(c.Request.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrInvalidURL):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_url", "message": err.Error()})
		case errors.Is(err, domain.ErrInvalidShortCode):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_alias", "message": err.Error()})
		case errors.Is(err, domain.ErrShortCodeConflict):
			c.JSON(http.StatusConflict, gin.H{"error": "conflict", "message": "custom alias is already taken"})
		case errors.Is(err, domain.ErrURLExpired):
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_expiration", "message": "expiration time must be in the future"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "failed to shorten url"})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status": "success",
		"data":   resp,
	})
}

// Redirect godoc
// @Summary Redirect short URL
// @Description High-throughput 302 Found redirection with asynchronous click audit tracking.
// @Param shortCode path string true "Short code"
// @Success 302
// @Failure 404 {object} map[string]interface{}
// @Failure 410 {object} map[string]interface{}
// @Router /{shortCode} [get]
func (h *URLHandler) Redirect(c *gin.Context) {
	shortCode := c.Param("shortCode")
	if shortCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "short code is required"})
		return
	}

	clickEvent := domain.ClickEvent{
		ShortCode: shortCode,
		ClickedAt: time.Now().UTC(),
		IPAddress: c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		Referrer:  c.Request.Referer(),
	}

	originalURL, err := h.service.Resolve(c.Request.Context(), shortCode, clickEvent)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrURLNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "not_found", "message": "short URL not found"})
		case errors.Is(err, domain.ErrURLExpired):
			c.JSON(http.StatusGone, gin.H{"error": "expired", "message": "short URL has expired"})
		case errors.Is(err, domain.ErrURLInactive):
			c.JSON(http.StatusForbidden, gin.H{"error": "inactive", "message": "short URL is inactive"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "failed to resolve url"})
		}
		return
	}

	c.Redirect(http.StatusFound, originalURL)
}

// GetAnalytics godoc
// @Summary Fetch URL analytics
// @Description Returns click statistics, top referrers, and recent clicks for a short URL.
// @Param shortCode path string true "Short code"
// @Produce json
// @Success 200 {object} domain.AnalyticsSummary
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/analytics/{shortCode} [get]
func (h *URLHandler) GetAnalytics(c *gin.Context) {
	shortCode := c.Param("shortCode")
	if shortCode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "short code is required"})
		return
	}

	summary, err := h.service.GetAnalytics(c.Request.Context(), shortCode)
	if err != nil {
		if errors.Is(err, domain.ErrURLNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not_found", "message": "short URL not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error", "message": "failed to fetch analytics"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   summary,
	})
}
