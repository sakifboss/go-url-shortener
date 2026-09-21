package model

import "time"

type ClickEvent struct {
	ID             int64
	EventKey       string
	URLID          int64
	ClickedAt      time.Time
	UserAgent      string
	Referrer       string
	DeviceCategory string
}

type ClickAnalytics struct {
	TotalClicks int64             `json:"total_clicks"`
	ByDevice    []AnalyticsCount  `json:"by_device"`
	ByReferrer  []AnalyticsCount  `json:"by_referrer"`
	DailyClicks []DailyClickCount `json:"daily_clicks"`
}

type AnalyticsCount struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

type DailyClickCount struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}
