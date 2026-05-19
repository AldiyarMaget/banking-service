package domain
import "context"
type AnalyticsUseCase interface {
	SetDailyLimit(ctx context.Context, accountID string, limit int64) error
}
