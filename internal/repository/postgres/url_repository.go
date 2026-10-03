package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"short-go/internal/domain"
)

type urlRepository struct {
	db *sql.DB
}

// NewURLRepository constructs a concrete PostgreSQL repository for URL operations.
func NewURLRepository(db *sql.DB) domain.URLRepository {
	return &urlRepository{db: db}
}

// ReserveNextID allocates the next ID from the underlying PostgreSQL sequence.
func (r *urlRepository) ReserveNextID(ctx context.Context) (int64, error) {
	query := `SELECT nextval(pg_get_serial_sequence('urls', 'id'))`
	var id int64
	if err := r.db.QueryRowContext(ctx, query).Scan(&id); err != nil {
		return 0, fmt.Errorf("failed to reserve next url id: %w", err)
	}
	return id, nil
}

// Create persists a URL entity into PostgreSQL.
// If url.ID is already set, it uses OVERRIDING SYSTEM VALUE for the identity column.
func (r *urlRepository) Create(ctx context.Context, u *domain.URL) error {
	var query string
	var err error

	if u.ID > 0 {
		query = `
			INSERT INTO urls (id, short_code, original_url, created_at, updated_at, expires_at, click_count, is_active)
			OVERRIDING SYSTEM VALUE
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING created_at, updated_at`

		err = r.db.QueryRowContext(
			ctx,
			query,
			u.ID,
			u.ShortCode,
			u.OriginalURL,
			u.CreatedAt,
			u.UpdatedAt,
			u.ExpiresAt,
			u.ClickCount,
			u.IsActive,
		).Scan(&u.CreatedAt, &u.UpdatedAt)
	} else {
		query = `
			INSERT INTO urls (short_code, original_url, created_at, updated_at, expires_at, click_count, is_active)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id, created_at, updated_at`

		err = r.db.QueryRowContext(
			ctx,
			query,
			u.ShortCode,
			u.OriginalURL,
			u.CreatedAt,
			u.UpdatedAt,
			u.ExpiresAt,
			u.ClickCount,
			u.IsActive,
		).Scan(&u.ID, &u.CreatedAt, &u.UpdatedAt)
	}

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key value") || strings.Contains(err.Error(), "23505") {
			return domain.ErrShortCodeConflict
		}
		return fmt.Errorf("failed to insert url: %w", err)
	}

	return nil
}

// GetByID retrieves a single URL entity by primary key ID.
func (r *urlRepository) GetByID(ctx context.Context, id int64) (*domain.URL, error) {
	query := `
		SELECT id, short_code, original_url, created_at, updated_at, expires_at, click_count, is_active
		FROM urls
		WHERE id = $1`

	u := &domain.URL{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&u.ID,
		&u.ShortCode,
		&u.OriginalURL,
		&u.CreatedAt,
		&u.UpdatedAt,
		&u.ExpiresAt,
		&u.ClickCount,
		&u.IsActive,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrURLNotFound
		}
		return nil, fmt.Errorf("failed to fetch url by id %d: %w", id, err)
	}

	return u, nil
}

// GetByShortCode retrieves a single URL entity by short_code.
func (r *urlRepository) GetByShortCode(ctx context.Context, shortCode string) (*domain.URL, error) {
	query := `
		SELECT id, short_code, original_url, created_at, updated_at, expires_at, click_count, is_active
		FROM urls
		WHERE short_code = $1`

	u := &domain.URL{}
	err := r.db.QueryRowContext(ctx, query, shortCode).Scan(
		&u.ID,
		&u.ShortCode,
		&u.OriginalURL,
		&u.CreatedAt,
		&u.UpdatedAt,
		&u.ExpiresAt,
		&u.ClickCount,
		&u.IsActive,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrURLNotFound
		}
		return nil, fmt.Errorf("failed to fetch url by short code %s: %w", shortCode, err)
	}

	return u, nil
}

// IncrementClickCount updates the recorded click count for a URL by delta.
func (r *urlRepository) IncrementClickCount(ctx context.Context, urlID int64, delta int64) error {
	query := `
		UPDATE urls
		SET click_count = click_count + $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2`

	_, err := r.db.ExecContext(ctx, query, delta, urlID)
	if err != nil {
		return fmt.Errorf("failed to increment click count for url %d: %w", urlID, err)
	}

	return nil
}
