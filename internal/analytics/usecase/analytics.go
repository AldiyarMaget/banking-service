package usecase

import (
	"context"
	"fmt"
	"banking-service/internal/analytics/domain"
	accountv1 "github.com/AldiyarMaget/banking-generated/gen/go/proto/account/v1"
)

type analyticsUseCase struct {
	repo          domain.AnalyticsRepository
	accountClient accountv1.AccountServiceClient
}

func NewAnalyticsUseCase(repo domain.AnalyticsRepository, ac accountv1.AccountServiceClient) domain.AnalyticsUseCase {
	return &analyticsUseCase{
		repo:          repo,
		accountClient: ac,
	}
}

func (u *analyticsUseCase) SetDailyLimit(ctx context.Context, accountID string, limit int64) error {
	resp, err := u.accountClient.GetAccount(ctx, &accountv1.GetAccountRequest{
		AccountId: accountID,
	})
	if err != nil {
		return fmt.Errorf("failed to get account info: %w", err)
	}

	dl := &domain.DailyLimit{
		UserID:     resp.CustomerId,
		DailyLimit: limit,
		Currency:   resp.Currency,
	}
	if err := u.repo.SetDailyLimit(ctx, dl); err != nil {
		return fmt.Errorf("failed to set daily limit: %w", err)
	}

	return nil
}
