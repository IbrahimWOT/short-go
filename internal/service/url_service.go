package service

import (
	"context"
	"fmt"
	neturl "net/url"
	"strings"
	"time"

	"short-go/internal/domain"
	"short-go/pkg/base62"
)

type urlService struct {
	urlRepo       domain.URLRepository
	cacheRepo     domain.URLCacheRepository
	analyticsRepo domain.AnalyticsRepository
	collector     domain.AnalyticsCollector
	baseURL       string
	defaultTTL    time.Duration
}

// NewURLService constructs the business logic service implementation for URLs and Redirection.
func NewURLService(
	urlRepo domain.URLRepository,
	cacheRepo domain.URLCacheRepository,
	analyticsRepo domain.AnalyticsRepository,
	collector domain.AnalyticsCollector,
	baseURL string,
	defaultTTL time.Duration,
) domain.URLService {
	// Clean baseURL to prevent trailing slashes
	baseURL = strings.TrimRight(baseURL, "/")

	return &urlService{
		urlRepo:       urlRepo,
		cacheRepo:     cacheRepo,
		analyticsRepo: analyticsRepo,
		collector:     collector,
		baseURL:       baseURL,
		defaultTTL:    defaultTTL,
	}
}

// Shorten creates a new shortened URL mapping.
func (s *urlService) Shorten(ctx context.Context, req domain.ShortenRequest) (*domain.ShortenResponse, error) {
	// 1. Strict URL validation
	parsedURL, err := neturl.ParseRequestURI(req.OriginalURL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Host == "" {
		return nil, domain.ErrInvalidURL
	}

	// 2. Validate expiration if provided
	if req.ExpiresAt != nil && req.ExpiresAt.Before(time.Now().UTC()) {
		return nil, domain.ErrURLExpired
	}

	u := &domain.URL{
		OriginalURL: req.OriginalURL,
		ExpiresAt:   req.ExpiresAt,
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
		IsActive:    true,
	}

	// 3. Short code generation or custom alias
	if req.CustomAlias != "" {
		alias := strings.TrimSpace(req.CustomAlias)
		if len(alias) < 3 || len(alias) > 16 || !base62.IsValid(alias) {
			return nil, domain.ErrInvalidShortCode
		}
		u.ShortCode = alias

		if err := s.urlRepo.Create(ctx, u); err != nil {
			return nil, err
		}
	} else {
		// Atomic ID reservation from PostgreSQL sequence
		nextID, err := s.urlRepo.ReserveNextID(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed allocating short code id: %w", err)
		}

		code, err := base62.Encode(nextID)
		if err != nil {
			return nil, fmt.Errorf("failed encoding base62 short code: %w", err)
		}

		u.ID = nextID
		u.ShortCode = code

		if err := s.urlRepo.Create(ctx, u); err != nil {
			return nil, err
		}
	}

	// 4. Eagerly warm Redis cache
	ttl := s.defaultTTL
	if u.ExpiresAt != nil {
		remaining := time.Until(*u.ExpiresAt)
		if remaining > 0 && remaining < ttl {
			ttl = remaining
		}
	}

	if ttl > 0 {
		_ = s.cacheRepo.Set(ctx, u, ttl)
	}

	return &domain.ShortenResponse{
		ID:          u.ID,
		ShortCode:   u.ShortCode,
		ShortURL:    fmt.Sprintf("%s/%s", s.baseURL, u.ShortCode),
		OriginalURL: u.OriginalURL,
		CreatedAt:   u.CreatedAt,
		ExpiresAt:   u.ExpiresAt,
	}, nil
}

// Resolve looks up the destination URL (Redis -> Postgres fallback) and emits an asynchronous click event.
func (s *urlService) Resolve(ctx context.Context, shortCode string, click domain.ClickEvent) (string, error) {
	var targetURL *domain.URL

	// 1. Check Redis cache first
	cached, err := s.cacheRepo.Get(ctx, shortCode)
	if err == nil && cached != nil {
		targetURL = cached
	} else {
		// 2. Cache miss: Fallback to PostgreSQL
		dbURL, err := s.urlRepo.GetByShortCode(ctx, shortCode)
		if err != nil {
			return "", err
		}
		targetURL = dbURL

		// Asynchronously or eagerly populate cache
		ttl := s.defaultTTL
		if targetURL.ExpiresAt != nil {
			remaining := time.Until(*targetURL.ExpiresAt)
			if remaining > 0 && remaining < ttl {
				ttl = remaining
			}
		}
		if ttl > 0 {
			_ = s.cacheRepo.Set(ctx, targetURL, ttl)
		}
	}

	// 3. Domain validation (active, expiration)
	if err := targetURL.CanRedirect(); err != nil {
		return "", err
	}

	// 4. Dispatch non-blocking asynchronous click event to bounded worker pool
	click.URLID = targetURL.ID
	click.ShortCode = targetURL.ShortCode
	if click.ClickedAt.IsZero() {
		click.ClickedAt = time.Now().UTC()
	}
	s.collector.Enqueue(click)

	return targetURL.OriginalURL, nil
}

// GetAnalytics retrieves reporting metrics for a short URL.
func (s *urlService) GetAnalytics(ctx context.Context, shortCode string) (*domain.AnalyticsSummary, error) {
	return s.analyticsRepo.GetSummary(ctx, shortCode, 20)
}
