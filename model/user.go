package model

import "time"

// User represents an authenticated GoShort user.
type User struct {
	ID           int64
	Email        string
	PasswordHash string
	Role         string
	CreatedAt    time.Time
}
