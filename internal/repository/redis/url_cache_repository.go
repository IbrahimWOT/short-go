package redis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"short-go/internal/domain"
)

const urlKeyPrefix = "url:"

type urlCacheRepository struct {
	client *redis.Client
}

// NewURLCacheRepository constructs a concrete Redis repository for URL caching.
func NewURLCacheRepository(client *redis.Client) domain.URLCacheRepository {
	return &urlCacheRepository{client: client}
}

func cacheKey(shortCode string) string {
	return urlKeyPrefix + shortCode
}

// Get retrieves a URL entity from the Redis cache.
func (r *urlCacheRepository) Get(ctx context.Context, shortCode string) (*domain.URL, error) {
	key := cacheKey(shortCode)
	val, err := r.client.Get(ctx, key).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, domain.ErrURLNotFound
		}
		return nil, fmt.Errorf("redis error getting key %s: %w", key, err)
	}

	var u domain.URL
	if err := json.Unmarshal(val, &u); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cached url: %w", err)
	}

	return &u, nil
}

// Set saves a URL entity into Redis with the given expiration duration.
func (r *urlCacheRepository) Set(ctx context.Context, u *domain.URL, ttl time.Duration) error {
	key := cacheKey(u.ShortCode)
	data, err := json.Marshal(u)
	if err != nil {
		return fmt.Errorf("failed to marshal url for caching: %w", err)
	}

	if err := r.client.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("redis error setting key %s: %w", key, err)
	}

	return nil
}

// Delete evicts a URL entry from Redis cache.
func (r *urlCacheRepository) Delete(ctx context.Context, shortCode string) error {
	key := cacheKey(shortCode)
	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("redis error deleting key %s: %w", key, err)
	}
	return nil
}
