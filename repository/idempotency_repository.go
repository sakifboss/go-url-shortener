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
	State          string
}

type IdempotencyRepository interface {
	Get(
		ctx context.Context,
		userID int64,
		key string,
	) (*IdempotencyRecord, error)

	Reserve(
		ctx context.Context,
		userID int64,
		key string,
	) (bool, error)

	Complete(
		ctx context.Context,
		record IdempotencyRecord,
	) error

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
		SELECT user_id, idempotency_key, response_body, status_code, state
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
		&record.State,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, fmt.Errorf("get idempotency record: %w", err)
	}

	return &record, nil
}

func (r *PostgresIdempotencyRepository) Reserve(
	ctx context.Context,
	userID int64,
	key string,
) (bool, error) {
	const query = `
		INSERT INTO idempotency_keys (
			user_id,
			idempotency_key,
			response_body,
			status_code,
			state
		)
		VALUES ($1, $2, '{}'::jsonb, 0, 'pending')
		ON CONFLICT (user_id, idempotency_key)
		DO NOTHING
		RETURNING id
	`

	var id int64
	err := r.db.QueryRowContext(
		ctx,
		query,
		userID,
		key,
	).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}

		return false, fmt.Errorf("reserve idempotency record: %w", err)
	}

	return true, nil
}

func (r *PostgresIdempotencyRepository) Complete(
	ctx context.Context,
	record IdempotencyRecord,
) error {
	if !json.Valid(record.ResponseBody) {
		return fmt.Errorf("invalid response JSON")
	}

	const query = `
		UPDATE idempotency_keys
		SET response_body = $1,
			status_code = $2,
			state = 'completed'
		WHERE user_id = $3
		  AND idempotency_key = $4
		  AND state = 'pending'
	`

	result, err := r.db.ExecContext(
		ctx,
		query,
		record.ResponseBody,
		record.StatusCode,
		record.UserID,
		record.IdempotencyKey,
	)
	if err != nil {
		return fmt.Errorf("complete idempotency record: %w", err)
	}

	updated, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check idempotency completion: %w", err)
	}
	if updated != 1 {
		return fmt.Errorf("idempotency record was not pending")
	}

	return nil
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
			status_code,
			state
		)
		VALUES ($1, $2, $3, $4, 'completed')
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
