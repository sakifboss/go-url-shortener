package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"

	"goshort/model"
)

type UserRepository interface {
	Create(ctx context.Context, user model.User) (model.User, error)
	FindByEmail(ctx context.Context, email string) (model.User, error)
	FindByID(ctx context.Context, id int64) (model.User, error)
}

type PostgresUserRepository struct {
	db *sql.DB
}

func NewPostgresUserRepository(db *sql.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

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

// IsUserNotFound reports whether the repository error means
// that the requested user does not exist.
func IsUserNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}

// IsUniqueViolation reports whether PostgreSQL rejected an
// operation because of a UNIQUE constraint.
func IsUniqueViolation(err error) bool {
	var pgError *pgconn.PgError

	if !errors.As(err, &pgError) {
		return false
	}

	// PostgreSQL SQLSTATE 23505 = unique_violation.
	return pgError.Code == "23505"
}
