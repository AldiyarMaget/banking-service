package grpc

import (
	"context"
	"errors"

	"banking-service/internal/account/domain"
	accountv1 "github.com/AldiyarMaget/banking-generated/gen/go/proto/account/v1"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AccountHandler struct {
	accountv1.UnimplementedAccountServiceServer
	usecase domain.AccountUseCase
}

func NewAccountHandler(u domain.AccountUseCase) *AccountHandler {
	return &AccountHandler{
		usecase: u,
	}
}

func (h *AccountHandler) CreateAccount(ctx context.Context, req *accountv1.CreateAccountRequest) (*accountv1.CreateAccountResponse, error) {
	// Validate input
	if req.IdempotencyKey == "" {
		return nil, status.Error(codes.InvalidArgument, "idempotency_key is required")
	}
	if req.CustomerId == "" {
		return nil, status.Error(codes.InvalidArgument, "customer_id is required")
	}
	if req.Currency == "" {
		return nil, status.Error(codes.InvalidArgument, "currency is required")
	}

	accountID := uuid.New().String()

	// Call Usecase, passing the context and idempotency_key
	_, err := h.usecase.CreateAccount(ctx, req.IdempotencyKey, accountID, req.CustomerId, req.Currency)
	if err != nil {
		// Handle idempotency pattern gracefully (200 OK equivalent)
		if errors.Is(err, domain.ErrAlreadyProcessed) {
			return &accountv1.CreateAccountResponse{
				Status: "ALREADY_PROCESSED",
			}, nil
		}
		return nil, status.Errorf(codes.Internal, "failed to create account: %v", err)
	}

	return &accountv1.CreateAccountResponse{
		AccountId: accountID,
		Status:    "CREATED",
	}, nil
}
