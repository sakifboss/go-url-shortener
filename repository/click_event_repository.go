package repository

import (
	"context"
	"database/sql"
	"fmt"

	"goshort/model"
)

type ClickEventRepository interface {
	Create(ctx context.Context, event model.ClickEvent) error
}

type PostgresClickEventRepository struct {
	db *sql.DB
}

func NewPostgresClickEventRepository(
	db *sql.DB,
) *PostgresClickEventRepository {
	return &PostgresClickEventRepository{
		db: db,
	}
}

func (r *PostgresClickEventRepository) Create(
	ctx context.Context,
	event model.ClickEvent,
) error {
	const query = `
		INSERT INTO click_events (
			event_key,
			url_id,
			clicked_at,
			user_agent,
			referrer,
			device_category
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (event_key) DO NOTHING
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		event.EventKey,
		event.URLID,
		event.ClickedAt,
		event.UserAgent,
		event.Referrer,
		event.DeviceCategory,
	)

	if err != nil {
		return fmt.Errorf(
			"create click event: %w",
			err,
		)
	}

	return nil
}
