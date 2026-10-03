package domain

import "time"

// ClickLog represents a recorded access event for an individual URL redirect.
type ClickLog struct {
	ID        int64     `json:"id"`
	URLID     int64     `json:"url_id"`
	ShortCode string    `json:"short_code"`
	ClickedAt time.Time `json:"clicked_at"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	Referrer  string    `json:"referrer"`
}

// ClickEvent represents an immutable payload pushed into the asynchronous buffer.
type ClickEvent struct {
	URLID     int64     `json:"url_id"`
	ShortCode string    `json:"short_code"`
	ClickedAt time.Time `json:"clicked_at"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
	Referrer  string    `json:"referrer"`
}
