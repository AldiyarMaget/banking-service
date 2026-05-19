package usecase
import (
	"context"
	"banking-service/internal/analytics/domain"
)
type analyticsUseCase struct{}
func NewAnalyticsUseCase() domain.AnalyticsUseCase { return &analyticsUseCase{} }
func (u *analyticsUseCase) SetDailyLimit(ctx context.Context, accountID string, limit int64) error { return nil }
