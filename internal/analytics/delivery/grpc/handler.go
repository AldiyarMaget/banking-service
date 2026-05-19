package grpc
import (
	"context"
	"banking-service/internal/analytics/domain"
	analyticsv1 "github.com/AldiyarMaget/banking-generated/gen/go/proto/analytics/v1"
)
type AnalyticsHandler struct {
	analyticsv1.UnimplementedAnalyticsServiceServer
	usecase domain.AnalyticsUseCase
}
func NewAnalyticsHandler(u domain.AnalyticsUseCase) *AnalyticsHandler { return &AnalyticsHandler{usecase: u} }
func (h *AnalyticsHandler) SetDailyLimit(ctx context.Context, req *analyticsv1.SetDailyLimitRequest) (*analyticsv1.SetDailyLimitResponse, error) {
	h.usecase.SetDailyLimit(ctx, req.AccountId, req.LimitAmount)
	return &analyticsv1.SetDailyLimitResponse{Status: "SUCCESS"}, nil
}
