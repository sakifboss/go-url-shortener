package model

import "time"

// RefreshToken represents a server-side refresh-token record.
// Only the token hash is persisted.
type RefreshToken struct {
	ID        int64
	UserID    int64
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}
