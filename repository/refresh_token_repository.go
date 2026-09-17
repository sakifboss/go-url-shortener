package repository

import (
	"context"
	"database/sql"

	"goshort/model"
)

// RefreshTokenRepository defines persistent refresh-token operations.
type RefreshTokenRepository interface {
	Create(ctx context.Context, token model.RefreshToken) (model.RefreshToken, error)
	FindActiveByHash(ctx context.Context, tokenHash string) (model.RefreshToken, error)
	Revoke(ctx context.Context, id int64) error
}

// PostgresRefreshTokenRepository implements refresh-token storage
// using PostgreSQL.
type PostgresRefreshTokenRepository struct {
	db *sql.DB
}

// NewPostgresRefreshTokenRepository creates a PostgreSQL-backed
// refresh-token repository.
func NewPostgresRefreshTokenRepository(
	db *sql.DB,
) *PostgresRefreshTokenRepository {
	return &PostgresRefreshTokenRepository{
		db: db,
	}
}

// Create stores only the hashed refresh token.
func (r *PostgresRefreshTokenRepository) Create(
	ctx context.Context,
	token model.RefreshToken,
) (model.RefreshToken, error) {
	const query = `
		INSERT INTO refresh_tokens (
			user_id,
			token_hash,
			expires_at
		)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		token.UserID,
		token.TokenHash,
		token.ExpiresAt,
	).Scan(
		&token.ID,
		&token.CreatedAt,
	)

	if err != nil {
		return model.RefreshToken{}, err
	}

	return token, nil
}

// FindActiveByHash retrieves a refresh token that has not been
// revoked and has not expired.
func (r *PostgresRefreshTokenRepository) FindActiveByHash(
	ctx context.Context,
	tokenHash string,
) (model.RefreshToken, error) {
	const query = `
		SELECT
			id,
			user_id,
			token_hash,
			expires_at,
			revoked_at,
			created_at
		FROM refresh_tokens
		WHERE token_hash = $1
		  AND revoked_at IS NULL
		  AND expires_at > NOW()
	`

	var token model.RefreshToken

	err := r.db.QueryRowContext(
		ctx,
		query,
		tokenHash,
	).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.RevokedAt,
		&token.CreatedAt,
	)

	if err != nil {
		return model.RefreshToken{}, err
	}

	return token, nil
}

// Revoke invalidates a refresh token.
func (r *PostgresRefreshTokenRepository) Revoke(
	ctx context.Context,
	id int64,
) error {
	const query = `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE id = $1
		  AND revoked_at IS NULL
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		id,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}
