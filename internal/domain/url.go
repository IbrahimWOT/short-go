package domain

import (
	"time"
)

// URL represents the core URL entity mapping a shortened code to its target destination.
type URL struct {
	ID          int64      `json:"id"`
	ShortCode   string     `json:"short_code"`
	OriginalURL string     `json:"original_url"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	ClickCount  int64      `json:"click_count"`
	IsActive    bool       `json:"is_active"`
}

// IsExpired checks whether the URL has surpassed its expiration timestamp.
func (u *URL) IsExpired() bool {
	if u.ExpiresAt == nil {
		return false
	}
	return time.Now().UTC().After(*u.ExpiresAt)
}

// CanRedirect verifies that the URL is active and not expired.
func (u *URL) CanRedirect() error {
	if !u.IsActive {
		return ErrURLInactive
	}
	if u.IsExpired() {
		return ErrURLExpired
	}
	return nil
}
