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
