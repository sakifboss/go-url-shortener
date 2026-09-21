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

type ClickAnalyticsRepository interface {
	GetAnalytics(ctx context.Context, urlID int64) (model.ClickAnalytics, error)
}

func (r *PostgresClickEventRepository) GetAnalytics(
	ctx context.Context,
	urlID int64,
) (model.ClickAnalytics, error) {
	var analytics model.ClickAnalytics

	if err := r.db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM click_events WHERE url_id = $1`,
		urlID,
	).Scan(&analytics.TotalClicks); err != nil {
		return model.ClickAnalytics{}, fmt.Errorf("count click events: %w", err)
	}

	deviceRows, err := r.db.QueryContext(ctx, `
		SELECT COALESCE(NULLIF(device_category, ''), 'unknown'), COUNT(*)
		FROM click_events
		WHERE url_id = $1
		GROUP BY 1
		ORDER BY COUNT(*) DESC, 1
	`, urlID)
	if err != nil {
		return model.ClickAnalytics{}, fmt.Errorf("query device analytics: %w", err)
	}
	defer deviceRows.Close()

	for deviceRows.Next() {
		var item model.AnalyticsCount
		if err := deviceRows.Scan(&item.Name, &item.Count); err != nil {
			return model.ClickAnalytics{}, fmt.Errorf("scan device analytics: %w", err)
		}
		analytics.ByDevice = append(analytics.ByDevice, item)
	}
	if err := deviceRows.Err(); err != nil {
		return model.ClickAnalytics{}, fmt.Errorf("read device analytics: %w", err)
	}

	referrerRows, err := r.db.QueryContext(ctx, `
		SELECT COALESCE(NULLIF(referrer, ''), 'direct'), COUNT(*)
		FROM click_events
		WHERE url_id = $1
		GROUP BY 1
		ORDER BY COUNT(*) DESC, 1
	`, urlID)
	if err != nil {
		return model.ClickAnalytics{}, fmt.Errorf("query referrer analytics: %w", err)
	}
	defer referrerRows.Close()

	for referrerRows.Next() {
		var item model.AnalyticsCount
		if err := referrerRows.Scan(&item.Name, &item.Count); err != nil {
			return model.ClickAnalytics{}, fmt.Errorf("scan referrer analytics: %w", err)
		}
		analytics.ByReferrer = append(analytics.ByReferrer, item)
	}
	if err := referrerRows.Err(); err != nil {
		return model.ClickAnalytics{}, fmt.Errorf("read referrer analytics: %w", err)
	}

	dailyRows, err := r.db.QueryContext(ctx, `
		SELECT TO_CHAR(clicked_at AT TIME ZONE 'UTC', 'YYYY-MM-DD'), COUNT(*)
		FROM click_events
		WHERE url_id = $1
		GROUP BY 1
		ORDER BY 1
	`, urlID)
	if err != nil {
		return model.ClickAnalytics{}, fmt.Errorf("query daily analytics: %w", err)
	}
	defer dailyRows.Close()

	for dailyRows.Next() {
		var item model.DailyClickCount
		if err := dailyRows.Scan(&item.Date, &item.Count); err != nil {
			return model.ClickAnalytics{}, fmt.Errorf("scan daily analytics: %w", err)
		}
		analytics.DailyClicks = append(analytics.DailyClicks, item)
	}
	if err := dailyRows.Err(); err != nil {
		return model.ClickAnalytics{}, fmt.Errorf("read daily analytics: %w", err)
	}

	return analytics, nil
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
