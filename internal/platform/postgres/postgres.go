package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"short-go/internal/config"
)

// NewClient establishes a pooled connection to PostgreSQL using pgx driver.
func NewClient(cfg *config.DBConfig) (*sql.DB, error) {
	db, err := sql.Open("pgx", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("database ping failed: %w", err)
	}

	return db, nil
}

// RunMigrations applies raw SQL up migrations if the tables do not exist yet.
func RunMigrations(db *sql.DB, migrationFilePath string) error {
	content, err := os.ReadFile(migrationFilePath)
	if err != nil {
		return fmt.Errorf("failed reading migration file %s: %w", migrationFilePath, err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := db.ExecContext(ctx, string(content)); err != nil {
		return fmt.Errorf("failed executing migration %s: %w", migrationFilePath, err)
	}

	return nil
}
