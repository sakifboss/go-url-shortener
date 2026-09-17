package database

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// NewPostgres creates a PostgreSQL connection pool using the
// database URL supplied through the GOSHORT_DATABASE_URL environment variable.
func NewPostgres() (*sql.DB, error) {
	dsn := os.Getenv("GOSHORT_DATABASE_URL")

	if dsn == "" {
		return nil, fmt.Errorf("GOSHORT_DATABASE_URL is not set")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// These settings configure the database/sql connection pool.
	// Exact values can be tuned later based on application load.
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	db.SetConnMaxIdleTime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return db, nil
}
