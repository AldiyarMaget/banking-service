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

func (h *AccountHandler) GetAccount(ctx context.Context, req *accountv1.GetAccountRequest) (*accountv1.GetAccountResponse, error) {
	return &accountv1.GetAccountResponse{}, nil
}

func (h *AccountHandler) UpdateBalance(ctx context.Context, req *accountv1.UpdateBalanceRequest) (*accountv1.UpdateBalanceResponse, error) {
	return &accountv1.UpdateBalanceResponse{}, nil
}

func (h *AccountHandler) GetAccountHistory(ctx context.Context, req *accountv1.GetAccountHistoryRequest) (*accountv1.GetAccountHistoryResponse, error) {
	return &accountv1.GetAccountHistoryResponse{}, nil
}

func (h *AccountHandler) FreezeAccount(ctx context.Context, req *accountv1.FreezeAccountRequest) (*accountv1.FreezeAccountResponse, error) {
	return &accountv1.FreezeAccountResponse{}, nil
}

func (h *AccountHandler) CloseAccount(ctx context.Context, req *accountv1.CloseAccountRequest) (*accountv1.CloseAccountResponse, error) {
	return &accountv1.CloseAccountResponse{}, nil
}

func (h *AccountHandler) UpdateAccountStatus(ctx context.Context, req *accountv1.UpdateAccountStatusRequest) (*accountv1.UpdateAccountStatusResponse, error) {
	return &accountv1.UpdateAccountStatusResponse{}, nil
}
