package repository

import (
	"context"
	"banking-service/internal/analytics/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type analyticsRepository struct {
	pool *pgxpool.Pool
}

func NewAnalyticsRepository(pool *pgxpool.Pool) domain.AnalyticsRepository {
	return &analyticsRepository{pool: pool}
}

func (r *analyticsRepository) SetDailyLimit(ctx context.Context, limit *domain.DailyLimit) error {
	query := `
		INSERT INTO daily_limits (user_id, daily_limit, currency)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id)
		DO UPDATE SET daily_limit = EXCLUDED.daily_limit, currency = EXCLUDED.currency`
	_, err := r.pool.Exec(ctx, query, limit.UserID, limit.DailyLimit, limit.Currency)
	return err
}
