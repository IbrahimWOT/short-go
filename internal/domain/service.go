package domain

import (
	"context"
	"time"
)

// ShortenRequest contains the input payload for creating a shortened URL.
type ShortenRequest struct {
	OriginalURL string     `json:"original_url" binding:"required,url"`
	CustomAlias string     `json:"custom_alias,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// ShortenResponse represents the returned metadata after a URL is shortened.
type ShortenResponse struct {
	ID          int64      `json:"id"`
	ShortCode   string     `json:"short_code"`
	ShortURL    string     `json:"short_url"`
	OriginalURL string     `json:"original_url"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// URLService defines the high-level business use cases for the URL Shortener.
type URLService interface {
	// Shorten validates the input URL, assigns or generates a Base62 key,
	// stores the record in PostgreSQL, and eagerly warms the Redis cache.
	Shorten(ctx context.Context, req ShortenRequest) (*ShortenResponse, error)

	// Resolve looks up the original URL (cache-first, falling back to database),
	// validates freshness/activation, and dispatches an asynchronous click event.
	Resolve(ctx context.Context, shortCode string, clickInfo ClickEvent) (string, error)

	// GetAnalytics returns aggregated metrics and recent click audits.
	GetAnalytics(ctx context.Context, shortCode string) (*AnalyticsSummary, error)
}

// AnalyticsCollector manages the asynchronous ingestion and worker lifecycle
// for recording redirection clicks without increasing HTTP latency.
type AnalyticsCollector interface {
	// Enqueue accepts a click event into a buffered channel.
	// Returns false if the buffer is saturated (non-blocking fallback).
	Enqueue(event ClickEvent) bool

	// Start launches background worker routines to drain the event channel.
	Start(ctx context.Context)

	// Stop signals workers to finish processing queued events and flushes pending writes.
	Stop(ctx context.Context) error
}
