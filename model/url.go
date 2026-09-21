package model

import "time"

// URL represents a shortened URL stored in PostgreSQL.
type URL struct {
	ID          int64
	ShortCode   string
	CustomAlias *string
	OriginalURL string
	UserID      *int64
	CreatedAt   time.Time
	ExpiresAt   *time.Time
	IsActive    bool
}
