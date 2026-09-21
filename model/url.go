package model

import "time"

// URL represents a shortened URL stored in PostgreSQL.
type URL struct {
	ID          int64      `json:"id"`
	ShortCode   string     `json:"short_code"`
	CustomAlias *string    `json:"custom_alias,omitempty"`
	OriginalURL string     `json:"original_url"`
	UserID      *int64     `json:"user_id,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	IsActive    bool       `json:"is_active"`
}
