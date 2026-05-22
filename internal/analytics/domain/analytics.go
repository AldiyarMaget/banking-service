package domain

import "context"

type DailyLimit struct {
	UserID     string
	DailyLimit int64
	Currency   string
}

type AnalyticsRepository interface {
	SetDailyLimit(ctx context.Context, limit *DailyLimit) error
}

type AnalyticsUseCase interface {
	SetDailyLimit(ctx context.Context, accountID string, limit int64) error
}
