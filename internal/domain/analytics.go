package domain

import "time"

// ReferrerMetric aggregates click counts grouped by HTTP Referrer.
type ReferrerMetric struct {
	Referrer string `json:"referrer"`
	Clicks   int64  `json:"clicks"`
}

// UserAgentMetric aggregates click counts grouped by User Agent / Browser family.
type UserAgentMetric struct {
	UserAgent string `json:"user_agent"`
	Clicks    int64  `json:"clicks"`
}

// AnalyticsSummary provides aggregated reporting data for a specific short code.
type AnalyticsSummary struct {
	ShortCode      string            `json:"short_code"`
	OriginalURL    string            `json:"original_url"`
	TotalClicks    int64             `json:"total_clicks"`
	CreatedAt      time.Time         `json:"created_at"`
	LastAccessedAt *time.Time        `json:"last_accessed_at,omitempty"`
	TopReferrers   []ReferrerMetric  `json:"top_referrers"`
	RecentClicks   []ClickLog        `json:"recent_clicks"`
}
