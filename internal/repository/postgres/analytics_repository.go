package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"short-go/internal/domain"
)

type analyticsRepository struct {
	db *sql.DB
}

// NewAnalyticsRepository constructs a concrete PostgreSQL repository for analytics and click logs.
func NewAnalyticsRepository(db *sql.DB) domain.AnalyticsRepository {
	return &analyticsRepository{db: db}
}

// RecordClick persists a single click entry into PostgreSQL.
func (r *analyticsRepository) RecordClick(ctx context.Context, click *domain.ClickLog) error {
	query := `
		INSERT INTO click_logs (url_id, short_code, clicked_at, ip_address, user_agent, referrer)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`

	if click.ClickedAt.IsZero() {
		click.ClickedAt = time.Now().UTC()
	}

	err := r.db.QueryRowContext(
		ctx,
		query,
		click.URLID,
		click.ShortCode,
		click.ClickedAt,
		click.IPAddress,
		click.UserAgent,
		click.Referrer,
	).Scan(&click.ID)

	if err != nil {
		return fmt.Errorf("failed to record click: %w", err)
	}

	return nil
}

// RecordClicksBatch writes a slice of clicks in a single high-throughput multi-row INSERT query.
func (r *analyticsRepository) RecordClicksBatch(ctx context.Context, clicks []*domain.ClickLog) error {
	if len(clicks) == 0 {
		return nil
	}

	// Prepare multi-value statement: ($1, $2, $3, $4, $5, $6), ($7, $8, ...), ...
	valueStrings := make([]string, 0, len(clicks))
	valueArgs := make([]interface{}, 0, len(clicks)*6)

	for i, c := range clicks {
		if c.ClickedAt.IsZero() {
			c.ClickedAt = time.Now().UTC()
		}
		offset := i * 6
		valueStrings = append(
			valueStrings,
			fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d)",
				offset+1, offset+2, offset+3, offset+4, offset+5, offset+6),
		)
		valueArgs = append(valueArgs, c.URLID, c.ShortCode, c.ClickedAt, c.IPAddress, c.UserAgent, c.Referrer)
	}

	stmt := fmt.Sprintf(
		"INSERT INTO click_logs (url_id, short_code, clicked_at, ip_address, user_agent, referrer) VALUES %s",
		strings.Join(valueStrings, ","),
	)

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction for batch click insert: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	if _, err := tx.ExecContext(ctx, stmt, valueArgs...); err != nil {
		return fmt.Errorf("failed to execute batch insert: %w", err)
	}

	// Also aggregate click counts on URLs table in the same transaction
	urlClickCount := make(map[int64]int64)
	for _, c := range clicks {
		urlClickCount[c.URLID]++
	}

	for urlID, count := range urlClickCount {
		updateStmt := `UPDATE urls SET click_count = click_count + $1, updated_at = CURRENT_TIMESTAMP WHERE id = $2`
		if _, err := tx.ExecContext(ctx, updateStmt, count, urlID); err != nil {
			return fmt.Errorf("failed to increment click counts for url %d: %w", urlID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit batch click transaction: %w", err)
	}

	return nil
}

// GetSummary retrieves comprehensive metrics, top referrers, and recent clicks for a shortCode.
func (r *analyticsRepository) GetSummary(ctx context.Context, shortCode string, recentLimit int) (*domain.AnalyticsSummary, error) {
	if recentLimit <= 0 {
		recentLimit = 10
	}

	// 1. Fetch URL metadata
	urlQuery := `
		SELECT short_code, original_url, click_count, created_at
		FROM urls
		WHERE short_code = $1`

	summary := &domain.AnalyticsSummary{
		ShortCode:    shortCode,
		TopReferrers: make([]domain.ReferrerMetric, 0),
		RecentClicks: make([]domain.ClickLog, 0),
	}

	err := r.db.QueryRowContext(ctx, urlQuery, shortCode).Scan(
		&summary.ShortCode,
		&summary.OriginalURL,
		&summary.TotalClicks,
		&summary.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrURLNotFound
		}
		return nil, fmt.Errorf("failed to fetch url for analytics: %w", err)
	}

	// 2. Fetch last clicked timestamp
	lastClickedQuery := `SELECT MAX(clicked_at) FROM click_logs WHERE short_code = $1`
	var lastClicked sql.NullTime
	_ = r.db.QueryRowContext(ctx, lastClickedQuery, shortCode).Scan(&lastClicked)
	if lastClicked.Valid {
		summary.LastAccessedAt = &lastClicked.Time
	}

	// 3. Fetch Top Referrers
	referrerQuery := `
		SELECT COALESCE(NULLIF(referrer, ''), 'Direct / Unknown') as ref, COUNT(*) as count
		FROM click_logs
		WHERE short_code = $1
		GROUP BY ref
		ORDER BY count DESC
		LIMIT 5`

	refRows, err := r.db.QueryContext(ctx, referrerQuery, shortCode)
	if err == nil {
		defer refRows.Close()
		for refRows.Next() {
			var ref domain.ReferrerMetric
			if err := refRows.Scan(&ref.Referrer, &ref.Clicks); err == nil {
				summary.TopReferrers = append(summary.TopReferrers, ref)
			}
		}
	}

	// 4. Fetch Recent Clicks
	recentQuery := `
		SELECT id, url_id, short_code, clicked_at, ip_address, user_agent, referrer
		FROM click_logs
		WHERE short_code = $1
		ORDER BY clicked_at DESC
		LIMIT $2`

	recentRows, err := r.db.QueryContext(ctx, recentQuery, shortCode, recentLimit)
	if err == nil {
		defer recentRows.Close()
		for recentRows.Next() {
			var cl domain.ClickLog
			if err := recentRows.Scan(
				&cl.ID,
				&cl.URLID,
				&cl.ShortCode,
				&cl.ClickedAt,
				&cl.IPAddress,
				&cl.UserAgent,
				&cl.Referrer,
			); err == nil {
				summary.RecentClicks = append(summary.RecentClicks, cl)
			}
		}
	}

	return summary, nil
}
