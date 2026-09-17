package repository

import (
	"context"
	"database/sql"
	"errors"

	"goshort/model"
)

// UserRepository defines persistent user operations.
type UserRepository interface {
	Create(ctx context.Context, user model.User) (model.User, error)
	FindByEmail(ctx context.Context, email string) (model.User, error)
	FindByID(ctx context.Context, id int64) (model.User, error)
}

// PostgresUserRepository implements UserRepository using PostgreSQL.
type PostgresUserRepository struct {
	db *sql.DB
}

// NewPostgresUserRepository creates a PostgreSQL-backed user repository.
func NewPostgresUserRepository(db *sql.DB) *PostgresUserRepository {
	return &PostgresUserRepository{
		db: db,
	}
}

// Create stores a new user and returns its generated ID and timestamp.
func (r *PostgresUserRepository) Create(
	ctx context.Context,
	user model.User,
) (model.User, error) {
	const query = `
		INSERT INTO users (
			email,
			password_hash,
			role
		)
		VALUES ($1, $2, $3)
		RETURNING id, created_at
	`

	err := r.db.QueryRowContext(
		ctx,
		query,
		user.Email,
		user.PasswordHash,
		user.Role,
	).Scan(
		&user.ID,
		&user.CreatedAt,
	)

	if err != nil {
		return model.User{}, err
	}

	return user, nil
}

// FindByEmail retrieves a user by email.
func (r *PostgresUserRepository) FindByEmail(
	ctx context.Context,
	email string,
) (model.User, error) {
	const query = `
		SELECT
			id,
			email,
			password_hash,
			role,
			created_at
		FROM users
		WHERE email = $1
	`

	var user model.User

	err := r.db.QueryRowContext(
		ctx,
		query,
		email,
	).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
	)

	if err != nil {
		return model.User{}, err
	}

	return user, nil
}

// FindByID retrieves a user by database ID.
func (r *PostgresUserRepository) FindByID(
	ctx context.Context,
	id int64,
) (model.User, error) {
	const query = `
		SELECT
			id,
			email,
			password_hash,
			role,
			created_at
		FROM users
		WHERE id = $1
	`

	var user model.User

	err := r.db.QueryRowContext(
		ctx,
		query,
		id,
	).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
	)

	if err != nil {
		return model.User{}, err
	}

	return user, nil
}

// IsUserNotFound reports whether a user lookup returned no record.
func IsUserNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}
