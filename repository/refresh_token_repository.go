package repository

import (
	"context"
	"database/sql"

	"goshort/model"
)

type RefreshTokenRepository interface {
	Create(ctx context.Context, token model.RefreshToken) (model.RefreshToken, error)
	FindActiveByHash(ctx context.Context, tokenHash string) (model.RefreshToken, error)
	Revoke(ctx context.Context, id int64) error
	Rotate(
		ctx context.Context,
		oldTokenID int64,
		newToken model.RefreshToken,
	) (model.RefreshToken, error)
}

type PostgresRefreshTokenRepository struct {
	db *sql.DB
}

func NewPostgresRefreshTokenRepository(
	db *sql.DB,
) *PostgresRefreshTokenRepository {
	return &PostgresRefreshTokenRepository{
		db: db,
	}
}

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

// Rotate atomically revokes the old refresh token and creates
// a new refresh token in the same database transaction.
func (r *PostgresRefreshTokenRepository) Rotate(
	ctx context.Context,
	oldTokenID int64,
	newToken model.RefreshToken,
) (model.RefreshToken, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return model.RefreshToken{}, err
	}

	defer func() {
		_ = tx.Rollback()
	}()

	const revokeQuery = `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE id = $1
		  AND revoked_at IS NULL
	`

	result, err := tx.ExecContext(
		ctx,
		revokeQuery,
		oldTokenID,
	)
	if err != nil {
		return model.RefreshToken{}, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return model.RefreshToken{}, err
	}

	if rowsAffected == 0 {
		return model.RefreshToken{}, sql.ErrNoRows
	}

	const createQuery = `
		INSERT INTO refresh_tokens (
			user_id,
			token_hash,
			expires_at
		)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`

	err = tx.QueryRowContext(
		ctx,
		createQuery,
		newToken.UserID,
		newToken.TokenHash,
		newToken.ExpiresAt,
	).Scan(
		&newToken.ID,
		&newToken.CreatedAt,
	)

	if err != nil {
		return model.RefreshToken{}, err
	}

	if err := tx.Commit(); err != nil {
		return model.RefreshToken{}, err
	}

	return newToken, nil
}
