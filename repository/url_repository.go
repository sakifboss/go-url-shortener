package repository

import (
	"context"
	"database/sql"
	"errors"

	"goshort/model"
)

// URLRepository defines persistent storage operations required
// by the URL service.
type URLRepository interface {
	Create(ctx context.Context, url model.URL) (model.URL, error)
	FindByID(ctx context.Context, id int64) (model.URL, error)
	FindByShortCode(ctx context.Context, shortCode string) (model.URL, error)
	Update(ctx context.Context, url model.URL) (model.URL, error)
	Delete(ctx context.Context, id int64) error
}

// PostgresURLRepository implements URLRepository using PostgreSQL.
type PostgresURLRepository struct {
	db *sql.DB
}

// NewPostgresURLRepository creates a PostgreSQL-backed repository.
func NewPostgresURLRepository(db *sql.DB) *PostgresURLRepository {
	return &PostgresURLRepository{
		db: db,
	}
}

// Create stores a shortened URL and retrieves the generated ID
// and creation timestamp from PostgreSQL.
func (r *PostgresURLRepository) Create(
	ctx context.Context,
	url model.URL,
) (model.URL, error) {
	const query = `
		INSERT INTO urls (
			short_code,
			custom_alias,
			original_url,
			user_id,
			expires_at,
			is_active
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		url.ShortCode,
		url.CustomAlias,
		url.OriginalURL,
		url.UserID,
		url.ExpiresAt,
		url.IsActive,
	).Scan(
		&url.ID,
		&url.CreatedAt,
	)

	if err != nil {
		return model.URL{}, err
	}

	return url, nil
}

// FindByID retrieves a URL by its database ID.
func (r *PostgresURLRepository) FindByID(
	ctx context.Context,
	id int64,
) (model.URL, error) {
	const query = `
		SELECT
			id,
			short_code,
			custom_alias,
			original_url,
			user_id,
			created_at,
			expires_at,
			is_active
		FROM urls
		WHERE id = $1
	`

	var url model.URL

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&url.ID,
		&url.ShortCode,
		&url.CustomAlias,
		&url.OriginalURL,
		&url.UserID,
		&url.CreatedAt,
		&url.ExpiresAt,
		&url.IsActive,
	)

	if err != nil {
		return model.URL{}, err
	}

	return url, nil
}

// FindByShortCode retrieves an active URL for redirect.
func (r *PostgresURLRepository) FindByShortCode(
	ctx context.Context,
	shortCode string,
) (model.URL, error) {
	const query = `
		SELECT
			id,
			short_code,
			custom_alias,
			original_url,
			user_id,
			created_at,
			expires_at,
			is_active
		FROM urls
		WHERE short_code = $1
		  AND is_active = TRUE
		  AND (expires_at IS NULL OR expires_at > NOW())
	`

	var url model.URL

	err := r.db.QueryRowContext(ctx, query, shortCode).Scan(
		&url.ID,
		&url.ShortCode,
		&url.CustomAlias,
		&url.OriginalURL,
		&url.UserID,
		&url.CreatedAt,
		&url.ExpiresAt,
		&url.IsActive,
	)

	if err != nil {
		return model.URL{}, err
	}

	return url, nil
}

// Update modifies the editable fields of an existing URL.
func (r *PostgresURLRepository) Update(
	ctx context.Context,
	url model.URL,
) (model.URL, error) {
	const query = `
		UPDATE urls
		SET
			original_url = $1,
			expires_at = $2,
			is_active = $3
		WHERE id = $4
		RETURNING
			id,
			short_code,
			custom_alias,
			original_url,
			user_id,
			created_at,
			expires_at,
			is_active
	`

	var updated model.URL

	err := r.db.QueryRowContext(
		ctx,
		query,
		url.OriginalURL,
		url.ExpiresAt,
		url.IsActive,
		url.ID,
	).Scan(
		&updated.ID,
		&updated.ShortCode,
		&updated.CustomAlias,
		&updated.OriginalURL,
		&updated.UserID,
		&updated.CreatedAt,
		&updated.ExpiresAt,
		&updated.IsActive,
	)

	if err != nil {
		return model.URL{}, err
	}

	return updated, nil
}

// Delete permanently removes a URL by ID.
func (r *PostgresURLRepository) Delete(
	ctx context.Context,
	id int64,
) error {
	const query = `
		DELETE FROM urls
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, id)
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

// IsNotFound reports whether a repository error means
// that the requested record does not exist.
func IsNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}
