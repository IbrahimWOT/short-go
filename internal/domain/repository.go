package domain

import (
	"context"
	"time"
)

// URLRepository defines the persistent storage operations for URL mappings.
type URLRepository interface {
	// ReserveNextID allocates the next identity sequence value from PostgreSQL.
	// This enables deterministic, collision-free Base62 encoding prior to final insertion.
	ReserveNextID(ctx context.Context) (int64, error)

	// Create persists a new URL entity into storage.
	Create(ctx context.Context, url *URL) error

	// GetByID retrieves a URL by its primary key ID.
	GetByID(ctx context.Context, id int64) (*URL, error)

	// GetByShortCode retrieves a URL entity matching the given short code.
	GetByShortCode(ctx context.Context, shortCode string) (*URL, error)

	// IncrementClickCount increases the recorded access count for a URL by delta.
	IncrementClickCount(ctx context.Context, urlID int64, delta int64) error
}

// URLCacheRepository defines the low-latency caching operations for URL lookups.
type URLCacheRepository interface {
	// Get retrieves a cached URL entity by short code.
	// Returns ErrURLNotFound on cache miss.
	Get(ctx context.Context, shortCode string) (*URL, error)

	// Set stores a URL entity in cache with the designated time-to-live.
	Set(ctx context.Context, url *URL, ttl time.Duration) error

	// Delete evicts a URL entity from the cache.
	Delete(ctx context.Context, shortCode string) error
}

// AnalyticsRepository defines persistence operations for click logs and aggregation.
type AnalyticsRepository interface {
	// RecordClick persists an individual click event into the audit log.
	RecordClick(ctx context.Context, click *ClickLog) error

	// RecordClicksBatch performs a high-throughput multi-row batch insert.
	RecordClicksBatch(ctx context.Context, clicks []*ClickLog) error

	// GetSummary aggregates metrics and fetches recent activity for a given short code.
	GetSummary(ctx context.Context, shortCode string, recentLimit int) (*AnalyticsSummary, error)
}
