package grpc

import (
	"context"
	"banking-service/internal/analytics/domain"
	analyticsv1 "github.com/AldiyarMaget/banking-generated/gen/go/proto/analytics/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AnalyticsHandler struct {
	analyticsv1.UnimplementedAnalyticsServiceServer
	usecase domain.AnalyticsUseCase
}

func NewAnalyticsHandler(u domain.AnalyticsUseCase) *AnalyticsHandler { return &AnalyticsHandler{usecase: u} }

func (h *AnalyticsHandler) SetDailyLimit(ctx context.Context, req *analyticsv1.SetDailyLimitRequest) (*analyticsv1.SetDailyLimitResponse, error) {
	err := h.usecase.SetDailyLimit(ctx, req.AccountId, req.LimitAmount)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to set daily limit: %v", err)
	}
	return &analyticsv1.SetDailyLimitResponse{Status: "SUCCESS"}, nil
}
