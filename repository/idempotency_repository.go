package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

type IdempotencyRecord struct {
	UserID         int64
	IdempotencyKey string
	ResponseBody   []byte
	StatusCode     int
}

type IdempotencyRepository interface {
	Get(
		ctx context.Context,
		userID int64,
		key string,
	) (*IdempotencyRecord, error)

	Create(
		ctx context.Context,
		record IdempotencyRecord,
	) (bool, error)
}

type PostgresIdempotencyRepository struct {
	db *sql.DB
}

func NewPostgresIdempotencyRepository(db *sql.DB) *PostgresIdempotencyRepository {
	return &PostgresIdempotencyRepository{db: db}
}

func (r *PostgresIdempotencyRepository) Get(
	ctx context.Context,
	userID int64,
	key string,
) (*IdempotencyRecord, error) {
	const query = `
		SELECT user_id, idempotency_key, response_body, status_code
		FROM idempotency_keys
		WHERE user_id = $1 AND idempotency_key = $2
	`

	var record IdempotencyRecord

	err := r.db.QueryRowContext(
		ctx,
		query,
		userID,
		key,
	).Scan(
		&record.UserID,
		&record.IdempotencyKey,
		&record.ResponseBody,
		&record.StatusCode,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, fmt.Errorf("get idempotency record: %w", err)
	}

	return &record, nil
}

func (r *PostgresIdempotencyRepository) Create(
	ctx context.Context,
	record IdempotencyRecord,
) (bool, error) {
	if !json.Valid(record.ResponseBody) {
		return false, fmt.Errorf("invalid response JSON")
	}

	const query = `
		INSERT INTO idempotency_keys (
			user_id,
			idempotency_key,
			response_body,
			status_code
		)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, idempotency_key)
		DO NOTHING
		RETURNING id
	`

	var id int64

	err := r.db.QueryRowContext(
		ctx,
		query,
		record.UserID,
		record.IdempotencyKey,
		record.ResponseBody,
		record.StatusCode,
	).Scan(&id)

	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}

		return false, fmt.Errorf(
			"create idempotency record: %w",
			err,
		)
	}

	return true, nil
}
